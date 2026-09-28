package postgres

import (
	"context"
	"errors"
	"fmt"
	"fractallegend/game-server/internal/contribution"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func closureSQL(t *testing.T, s *Store, sql string) {
	t.Helper()
	if _, err := s.pool.Exec(context.Background(), sql); err != nil {
		t.Fatal(err)
	}
}
func closureTimeoutFixture(t *testing.T) (*Store, MiningRewardSettlementCommand) {
	t.Helper()
	s, p, i := g20Fixture(t)
	g19Catalog(t, s)
	return s, closureSeal(t, s, p, i, "closure-timeout")
}
func closureErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != code {
		t.Fatalf("want SQLSTATE %s got %v", code, err)
	}
}
func closureZero(t *testing.T, s *Store, before string) {
	t.Helper()
	if after := closureSnapshot(t, s); after != before {
		t.Fatal("failed settlement changed persistent rows (pool, history, commands, states, inventory, grants, lots, receipt, revisions)")
	}
	child := exec.Command(os.Args[0], "-test.run=^TestG21ClosureIndependentZeroVerifier$", "-test.v")
	child.Env = append(os.Environ(), "G21_CLOSURE_EXPECTED_ZERO="+rewardHash([]byte(before)))
	if out, err := child.CombinedOutput(); err != nil {
		t.Fatalf("independent verifier failed: %v %s", err, out)
	}
	t.Logf("ZERO controller_pid=%d independent_verifier_pid=%d full_public_rows_sha256=%s", os.Getpid(), child.ProcessState.Pid(), rewardHash([]byte(before)))
}
func closureFreshPay(t *testing.T, s *Store, cmd MiningRewardSettlementCommand) {
	t.Helper()
	r, err := s.SettleMiningRewardTEST(context.Background(), cmd)
	if err != nil || r.TotalOre != 10 {
		t.Fatalf("fresh recovery=%+v %v", r, err)
	}
	closureReconcile(t, s)
}

func TestG21ClosureSQLFailureEveryWriteStage(t *testing.T) {
	for _, stage := range []struct {
		name, table, event string
		deferred           bool
	}{{"lot", "mining_reward_issuance_lots", "INSERT", false}, {"stack", "character_inventory_items", "INSERT", false}, {"owner", "characters", "UPDATE", false}, {"pool", "black_iron_emission_pools", "UPDATE", false}, {"journal", "black_iron_emission_recovery_entries", "INSERT", false}, {"grant", "mining_reward_grants", "INSERT", false}, {"receipt", "mining_reward_settlement_receipts", "INSERT", false}, {"state", "mining_reward_settlement_states", "UPDATE", false}, {"deferred_check", "mining_reward_settlement_states", "UPDATE", true}} {
		t.Run(stage.name, func(t *testing.T) {
			s, cmd := closureTimeoutFixture(t)
			closureSQL(t, s, `CREATE OR REPLACE FUNCTION closure_fail_write() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'closure forced write failure' USING ERRCODE='P0001'; END $$`)
			ddl := fmt.Sprintf("CREATE TRIGGER closure_write_failure BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION closure_fail_write()", stage.event, stage.table)
			if stage.deferred {
				ddl = fmt.Sprintf("CREATE CONSTRAINT TRIGGER closure_write_failure AFTER %s ON %s DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION closure_fail_write()", stage.event, stage.table)
			}
			closureSQL(t, s, ddl)
			defer closureSQL(t, s, "DROP TRIGGER IF EXISTS closure_write_failure ON "+stage.table)
			before := closureSnapshot(t, s)
			_, err := s.SettleMiningRewardTEST(context.Background(), cmd)
			closureErrorCode(t, err, "P0001")
			closureZero(t, s, before)
			closureSQL(t, s, "DROP TRIGGER closure_write_failure ON "+stage.table)
			closureFreshPay(t, s, cmd)
		})
	}
}

func TestG21ClosurePostgresRetryClassificationAndExhaustion(t *testing.T) {
	for _, tc := range []struct {
		name, code         string
		failures, attempts int
		success            bool
	}{{"serialization_then_valid", "40001", 2, 3, true}, {"owner_serialization_then_valid", "40001", 2, 3, true}, {"owner_deadlock_then_valid", "40P01", 2, 3, true}, {"deadlock_then_valid", "40P01", 2, 3, true}, {"serialization_exhaustion", "40001", 5, 5, false}, {"deadlock_exhaustion", "40P01", 5, 5, false}, {"numeric_overflow_no_retry", "22003", 5, 1, false}, {"constraint_no_retry", "23514", 5, 1, false}} {
		t.Run(tc.name, func(t *testing.T) {
			s, cmd := closureTimeoutFixture(t)
			// Sequences are deliberately outside rollback. Every attempt must observe a
			// strictly larger PostgreSQL transaction ID, proving a fresh real transaction.
			closureSQL(t, s, `DROP SEQUENCE IF EXISTS closure_attempts,closure_last_xid; CREATE SEQUENCE closure_attempts; CREATE SEQUENCE closure_last_xid`)
			closureSQL(t, s, fmt.Sprintf(`CREATE OR REPLACE FUNCTION closure_retry_failure() RETURNS trigger LANGUAGE plpgsql AS $$ DECLARE n bigint; prior bigint; current_id bigint; BEGIN n:=nextval('closure_attempts'); prior:=(SELECT last_value FROM closure_last_xid); current_id:=txid_current(); IF current_id<=prior THEN RAISE EXCEPTION 'transaction reused' USING ERRCODE='P0001'; END IF; PERFORM setval('closure_last_xid',current_id); IF n<=%d THEN RAISE EXCEPTION 'forced PostgreSQL retry classification' USING ERRCODE='%s'; END IF; RETURN NEW; END $$`, tc.failures, tc.code))
			target, event := "mining_reward_issuance_lots", "INSERT"
			if strings.HasPrefix(tc.name, "owner_") {
				target, event = "characters", "UPDATE"
			}
			closureSQL(t, s, fmt.Sprintf("CREATE TRIGGER closure_retry_failure BEFORE %s ON %s FOR EACH ROW EXECUTE FUNCTION closure_retry_failure()", event, target))
			defer closureSQL(t, s, "DROP TRIGGER IF EXISTS closure_retry_failure ON "+target)
			before := closureSnapshot(t, s)
			r, err := s.SettleMiningRewardTEST(context.Background(), cmd)
			var attempts int
			if e := s.pool.QueryRow(context.Background(), `SELECT last_value FROM closure_attempts`).Scan(&attempts); e != nil {
				t.Fatal(e)
			}
			if attempts != tc.attempts {
				t.Fatalf("attempts=%d want=%d", attempts, tc.attempts)
			}
			if tc.success {
				if err != nil || r.TotalOre != 10 {
					t.Fatalf("retry result=%+v err=%v", r, err)
				}
				closureReconcile(t, s)
			} else {
				closureErrorCode(t, err, tc.code)
				if tc.attempts == 5 && !strings.Contains(err.Error(), "retry exhausted") {
					t.Fatalf("missing exhaustion error: %v", err)
				}
				closureZero(t, s, before)
			}
			closureSQL(t, s, "DROP TRIGGER closure_retry_failure ON "+target)
			closureFreshPay(t, s, cmd)
			t.Logf("SQLSTATE=%s PostgreSQL attempts=%d fresh transaction IDs verified", tc.code, tc.attempts)
		})
	}
}

func TestG21ClosureStatementTimeoutAndInflightCancellation(t *testing.T) {
	t.Run("production_default_60s", func(t *testing.T) {
		s, cmd := closureTimeoutFixture(t)
		closureSQL(t, s, `CREATE OR REPLACE FUNCTION closure_timeout_default() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF current_setting('statement_timeout')<>'1min' THEN RAISE EXCEPTION 'production timeout is not 60s'; END IF; RETURN NEW; END $$; CREATE TRIGGER closure_timeout_default BEFORE INSERT ON mining_reward_issuance_lots FOR EACH ROW EXECUTE FUNCTION closure_timeout_default()`)
		defer closureSQL(t, s, `DROP TRIGGER IF EXISTS closure_timeout_default ON mining_reward_issuance_lots`)
		closureFreshPay(t, s, cmd)
	})
	for _, kind := range []string{"statement_timeout", "inflight_context_cancel"} {
		t.Run(kind, func(t *testing.T) {
			s, cmd := closureTimeoutFixture(t)
			closureSQL(t, s, `DROP SEQUENCE IF EXISTS closure_sleep_entered; CREATE SEQUENCE closure_sleep_entered; CREATE OR REPLACE FUNCTION closure_sleep() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM nextval('closure_sleep_entered'); PERFORM pg_sleep(2); RETURN NEW; END $$; CREATE TRIGGER closure_sleep BEFORE INSERT ON mining_reward_issuance_lots FOR EACH ROW EXECUTE FUNCTION closure_sleep()`)
			defer closureSQL(t, s, `DROP TRIGGER IF EXISTS closure_sleep ON mining_reward_issuance_lots`)
			ctx := context.Background()
			cancel := func() {}
			if kind == "statement_timeout" {
				s.settlementStatementTimeout = 50 * time.Millisecond
			} else {
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
			}
			defer cancel()
			before := closureSnapshot(t, s)
			started := time.Now()
			_, err := s.SettleMiningRewardTEST(ctx, cmd)
			if kind == "statement_timeout" {
				closureErrorCode(t, err, "57014")
			} else if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("inflight cancellation=%v", err)
			}
			if elapsed := time.Since(started); elapsed > time.Second {
				t.Fatalf("timeout rollback took %s", elapsed)
			}
			var entered bool
			if err := s.pool.QueryRow(context.Background(), `SELECT is_called FROM closure_sleep_entered`).Scan(&entered); err != nil || !entered {
				t.Fatalf("cancellation did not interrupt an executing PostgreSQL statement: entered=%v err=%v", entered, err)
			}
			closureZero(t, s, before)
			s.settlementStatementTimeout = 0
			closureSQL(t, s, `DROP TRIGGER closure_sleep ON mining_reward_issuance_lots`)
			closureFreshPay(t, s, cmd)
		})
	}
}

func TestG21ClosureIndependentZeroVerifier(t *testing.T) {
	expected := os.Getenv("G21_CLOSURE_EXPECTED_ZERO")
	if expected == "" {
		return
	}
	s, err := Open(context.Background(), os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if got := rewardHash([]byte(closureSnapshot(t, s))); got != expected {
		t.Fatalf("independent process snapshot hash=%s expected=%s", got, expected)
	}
}

func TestG21ClosureActualLockTimeoutZeroAndFreshRecovery(t *testing.T) {
	s, cmd := closureTimeoutFixture(t)
	ctx := context.Background()
	blocker, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	before := closureSnapshot(t, s)
	started := time.Now()
	_, err = s.SettleMiningRewardTEST(ctx, cmd)
	closureErrorCode(t, err, "55P03")
	if elapsed := time.Since(started); elapsed < 4*time.Second || elapsed > 8*time.Second {
		t.Fatalf("lock timeout elapsed=%s", elapsed)
	}
	if err = blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	closureZero(t, s, before)
	closureFreshPay(t, s, cmd)
}

type closureTransactionCounter struct{ begins atomic.Int64 }

func (c *closureTransactionCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if strings.HasPrefix(strings.ToLower(data.SQL), "begin") {
		c.begins.Add(1)
	}
	return ctx
}
func (c *closureTransactionCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {
}
func TestG21ClosureDomainErrorsAreNotRetried(t *testing.T) {
	for _, kind := range []string{"debt", "fingerprint"} {
		t.Run(kind, func(t *testing.T) {
			s, cmd := closureTimeoutFixture(t)
			ctx := context.Background()
			if kind == "debt" {
				spend, err := s.LoadSystemSpend(ctx, "g18-fund")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "closure-domain-debt", 15)); err != nil {
					t.Fatal(err)
				}
			} else {
				cmd.ExpectedSealDigest = strings.Repeat("0", 64)
			}
			config := s.pool.Config()
			counter := &closureTransactionCounter{}
			config.ConnConfig.Tracer = counter
			pool, err := pgxpool.NewWithConfig(ctx, config)
			if err != nil {
				t.Fatal(err)
			}
			defer pool.Close()
			traced := &Store{pool: pool}
			before := closureSnapshot(t, s)
			_, err = traced.SettleMiningRewardTEST(ctx, cmd)
			want := ErrMiningRewardConflict
			if kind == "debt" {
				want = ErrMiningRewardDebt
			}
			if !errors.Is(err, want) {
				t.Fatalf("domain rejection=%v want=%v", err, want)
			}
			if n := counter.begins.Load(); n != 1 {
				t.Fatalf("domain error began %d transactions want1", n)
			}
			closureZero(t, s, before)
		})
	}
}
