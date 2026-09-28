package postgres

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Retiring a current rule/session is fixture-only: historical seals, commands,
// receipts, grants, lots, projections and consumption records stay untouched.
func g21T044ChangedRule(t *testing.T, s *Store) {
	t.Helper()
	g21AuthorityCorrupt(t, s, `UPDATE mining_power_sessions SET data=jsonb_set(data,'{RuleVersion}','"FUTURE_RULE"')`)
}
func g21T044Exact(t *testing.T, s *Store, cmd MiningRewardSettlementCommand, want MiningRewardSettlementReceipt) {
	t.Helper()
	before := g21ClosureSnapshot(t, s)
	got, err := s.SettleMiningRewardTEST(context.Background(), cmd)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("historical replay changed receipt: got=%+v error=%v", got, err)
	}
	raw, err := canonicalMiningRewardReceipt(got)
	if err != nil {
		t.Fatal(err)
	}
	var stored []byte
	if err = s.pool.QueryRow(context.Background(), `SELECT canonical_bytes FROM mining_reward_settlement_receipts WHERE command_id=$1`, cmd.CommandID).Scan(&stored); err != nil || !bytes.Equal(raw, stored) {
		t.Fatalf("receipt bytes differ: %v", err)
	}
	g21AuthorityUnchanged(t, s, before)
}
func TestG21T044CompletedReplayPrecedence(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"A_rule_binding", `UPDATE mining_power_sessions SET data=jsonb_set(data,'{RuleVersion}','"FUTURE_RULE"')`},
		{"current_rule_retired", `DELETE FROM mining_power_rules`},
		{"current_beneficiary_binding", `DELETE FROM mining_power_beneficiary_bindings`},
		{"current_reservation_binding", `DELETE FROM mining_reservation_instance_bindings`},
		{"current_catalog", `DELETE FROM black_iron_identity_aliases`},
		{"current_inventory", `UPDATE character_inventory_items SET quantity=quantity+1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, i, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			first, err := s.SettleMiningRewardTEST(ctx, cmd)
			if err != nil || first.Status != "COMPLETED" {
				t.Fatalf("first: %v", err)
			}
			g21AuthorityCorrupt(t, s, tc.sql)
			future, err := miningpower.NewService(s, "FUTURE_RULE", false).Validate(ctx, p, i)
			if err != nil || future.ReasonCode != "BINDING_CHANGED" || future.Original != nil {
				t.Fatalf("G20 protection: %+v %v", future, err)
			}
			restarted, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer restarted.Close()
			g21T044Exact(t, restarted, cmd, first)
		})
	}
}
func TestG21T044ChangedFingerprintNeverDisclosesReceipt(t *testing.T) {
	s, _, _, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	g21T044ChangedRule(t, s)
	for _, bad := range []MiningRewardSettlementCommand{
		{cmd.CommandID, cmd.BlockInstanceID, strings.Repeat("0", 64)},
		{cmd.CommandID, "mining-block-instance-" + strings.Repeat("a", 32), cmd.ExpectedSealDigest},
	} {
		before := g21ClosureSnapshot(t, s)
		got, err := s.SettleMiningRewardTEST(ctx, bad)
		if !errors.Is(err, ErrMiningRewardConflict) || !reflect.DeepEqual(got, MiningRewardSettlementReceipt{}) {
			t.Fatalf("mismatched command disclosed result: %+v %v", got, err)
		}
		g21AuthorityUnchanged(t, s, before)
	}
}
func TestG21T044UnfinishedAndNewBindingChanged(t *testing.T) {
	for _, unfinished := range []bool{true, false} {
		name := "D_new"
		if unfinished {
			name = "C_unfinished"
		}
		t.Run(name, func(t *testing.T) {
			s, p, i, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			if unfinished {
				stop := errors.New("T044 precommit interruption")
				s.settlementFailureInjector = func(at string) error {
					if at == "before_commit" {
						return stop
					}
					return nil
				}
				if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, stop) {
					t.Fatal(err)
				}
				s.settlementFailureInjector = nil
			}
			g21T044ChangedRule(t, s)
			before := g21ClosureSnapshot(t, s)
			g20, err := miningpower.NewService(s, "FUTURE_RULE", false).Validate(ctx, p, i)
			if err != nil || g20.ReasonCode != "BINDING_CHANGED" || g20.Original != nil {
				t.Fatalf("G20: %+v %v", g20, err)
			}
			got, err := s.SettleMiningRewardTEST(ctx, cmd)
			if err == nil || !strings.Contains(err.Error(), "BINDING_CHANGED") || !errors.Is(err, ErrMiningRewardInvariant) || !reflect.DeepEqual(got, MiningRewardSettlementReceipt{}) {
				t.Fatalf("unfinished/new binding protection: %+v %v", got, err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
}
func TestG21T044CompletedDebtAndConsumedReservation(t *testing.T) {
	s, _, _, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	var consumed int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_reservation_consumptions WHERE settlement_id=$1`, first.SettlementID).Scan(&consumed); err != nil {
		t.Fatal(err)
	}
	if consumed != 1 {
		t.Fatalf("consumed=%d", consumed)
	}
	t.Run("F_consumed", func(t *testing.T) { g21T044Exact(t, s, cmd, first) })
	spend, err := s.LoadSystemSpend(ctx, "g18-fund")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-t044-refund", 20)); err != nil {
		t.Fatal(err)
	}
	var debt int64
	if err = s.pool.QueryRow(ctx, `SELECT recovery_debt FROM black_iron_emission_pools`).Scan(&debt); err != nil || debt != 10 {
		t.Fatalf("debt=%d %v", debt, err)
	}
	g21T044ChangedRule(t, s)
	t.Run("E_debt", func(t *testing.T) { g21T044Exact(t, s, cmd, first) })
}
func TestG21T044ReplayVerifierChild(t *testing.T) {
	if os.Getenv("G21_T044_VERIFY") == "" {
		return
	}
	s, err := Open(context.Background(), os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cmd := MiningRewardSettlementCommand{"g21-crash-command", os.Getenv("G21_SETTLEMENT_CRASH_INSTANCE"), os.Getenv("G21_SETTLEMENT_CRASH_SEAL_DIGEST")}
	var raw []byte
	if err = s.pool.QueryRow(context.Background(), `SELECT canonical_bytes FROM mining_reward_settlement_receipts WHERE command_id=$1`, cmd.CommandID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	want, err := decodeCanonicalMiningRewardReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	want.CanonicalDigest = rewardHash(raw)
	g21T044Exact(t, s, cmd, want)
}
func TestG21T044ACKLossThenChangedBinding(t *testing.T) {
	s, _, _, cmd := g21AuthoritySealed(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	marker := filepath.Join(t.TempDir(), "committed")
	env := append(os.Environ(), "G21_SETTLEMENT_CRASH_POINT=after_commit_ack_lost", "G21_SETTLEMENT_CRASH_INSTANCE="+cmd.BlockInstanceID, "G21_SETTLEMENT_CRASH_SEAL_DIGEST="+cmd.ExpectedSealDigest, "G21_SETTLEMENT_MARKER="+marker)
	child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG21SettlementCrashChild$", "-test.v")
	child.Env = env
	out, err := child.CombinedOutput()
	if _, e := os.Stat(marker); e != nil || err == nil || strings.Contains(string(out), "worker survived") {
		t.Fatalf("worker not killed after commit: %v %s", err, out)
	}
	g21T044ChangedRule(t, s)
	before := g21ClosureSnapshot(t, s)
	verifier := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG21T044ReplayVerifierChild$", "-test.v")
	verifier.Env = append(env, "G21_T044_VERIFY=1")
	if out, err = verifier.CombinedOutput(); err != nil {
		t.Fatalf("fresh replay verifier: %v %s", err, out)
	}
	g21AuthorityUnchanged(t, s, before)
}

// Actual query tracing ensures a future refactor cannot accidentally authorize
// historical success using mutable current prerequisites again.
func TestG21T044ReplayReadsHistoricalAuthorityOnly(t *testing.T) {
	s, _, _, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	cfg := s.pool.Config().Copy()
	trace := &g21GateTrace{}
	cfg.ConnConfig.Tracer = trace
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	replayStore := &Store{pool: pool}
	before := g21ClosureSnapshot(t, s)
	got, err := replayStore.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || !reflect.DeepEqual(got, first) {
		t.Fatalf("replay: %+v %v", got, err)
	}
	for _, sql := range trace.queries {
		for _, table := range []string{"mining_power_sessions", "mining_power_rules", "mining_power_beneficiary_bindings", "mining_reservation_instance_bindings", "mining_block_reservations", "black_iron_emission_pools", "black_iron_identity_aliases", "black_iron_migration_catalog", "character_inventory_items", "JOIN characters"} {
			if strings.Contains(sql, table) {
				t.Fatalf("completed replay queried current authority %s: %s", table, sql)
			}
		}
	}
	g21AuthorityUnchanged(t, s, before)
}
func TestG21T044ImmutableIdentityCorruptionRejected(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"command_settlement", `UPDATE mining_reward_commands SET settlement_id='different-settlement'`},
		{"historical_projection", `UPDATE mining_reward_inventory_projections SET quantity=quantity+1`},
		{"not_completed_state", `UPDATE mining_reward_settlement_states SET state='SEALED',revision=1,command_id=NULL,settlement_id=NULL,completed_at=NULL`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			g21AuthorityCorrupt(t, s, tc.sql)
			before := g21ClosureSnapshot(t, s)
			got, err := s.SettleMiningRewardTEST(ctx, cmd)
			if err == nil || !reflect.DeepEqual(got, MiningRewardSettlementReceipt{}) {
				t.Fatalf("invalid immutable history disclosed: %+v %v", got, err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
}
