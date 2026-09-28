package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/systemspend"
	"github.com/jackc/pgx/v5"
)

// These fixtures only create R10 reservations through the real G18 writer.
// Larger distributed/reserved totals are accumulated, never forged in SQL.
func closureRecoveryFixture(t *testing.T, distributed, reserved int) (*Store, []MiningRewardSettlementCommand, []MiningRewardSettlementReceipt, []string) {
	t.Helper()
	s, p, first := g20Fixture(t)
	g19Catalog(t, s)
	closureCapacity(t, s, "closure-extra-80", 80)
	ctx := context.Background()
	var commands []MiningRewardSettlementCommand
	var receipts []MiningRewardSettlementReceipt
	var open []string
	for n := 0; n < distributed+reserved; n++ {
		intent := first
		if n > 0 {
			born, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Create(ctx, fmt.Sprintf("closure-block-%d", n))
			if err != nil {
				t.Fatal(err)
			}
			b, err := s.LoadMiningBlock(ctx, born.BlockID)
			if err != nil {
				t.Fatal(err)
			}
			session := g20Session(t, s, first)
			session.ID = fmt.Sprintf("closure-session-%d", n)
			session.BlockID = b.ID
			session.BlockInstanceID = b.BlockInstanceID
			session.OpenedAt = b.StartedAt
			session.ExpiresAt = b.ScheduledEndAt
			g20Insert(t, s, "mining_power_sessions", session)
			source := g20Source(t, s, first)
			source.ID = fmt.Sprintf("closure-event-%d", n)
			source.ActivityID = "mpa:" + source.ID
			source.BlockID = b.ID
			source.BlockInstanceID = b.BlockInstanceID
			source.ActivitySessionID = session.ID
			source.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
			source.ExpiresAt = b.ScheduledEndAt
			g20Insert(t, s, "mining_power_source_events", source)
			intent = miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID, ActivitySessionID: session.ID, BlockID: b.ID, BlockInstanceID: b.BlockInstanceID}
		}
		if n >= distributed {
			open = append(open, intent.BlockID)
			continue
		}
		cmd := closureSeal(t, s, p, intent, fmt.Sprintf("closure-pay-%d", n))
		commands = append(commands, cmd)
		r, err := s.SettleMiningRewardTEST(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, r)
	}
	return s, commands, receipts, open
}
func closureSeal(t *testing.T, s *Store, p miningpower.Principal, i miningpower.ActionIntent, id string) MiningRewardSettlementCommand {
	t.Helper()
	g20Accept(t, s, p, i)
	ctx := context.Background()
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	return MiningRewardSettlementCommand{CommandID: id, BlockInstanceID: i.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
}
func closureCapacity(t *testing.T, s *Store, id string, amount int64) {
	t.Helper()
	ctx := context.Background()
	spend, err := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent(id, amount))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID); err != nil {
		t.Fatal(err)
	}
}
func closureRefund(t *testing.T, s *Store, id string, amount int64) {
	t.Helper()
	ctx := context.Background()
	spend, err := s.LoadSystemSpend(ctx, "closure-extra-80")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, id, amount)); err != nil {
		t.Fatal(err)
	}
}
func closurePool(t *testing.T, s *Store, want [5]int64) {
	t.Helper()
	var got [5]int64
	if err := s.pool.QueryRow(context.Background(), `SELECT total_emission_capacity,total_reserved,total_distributed,remaining_capacity,recovery_debt FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&got[0], &got[1], &got[2], &got[3], &got[4]); err != nil {
		t.Fatal(err)
	}
	if got != want || got[0] != got[1]+got[2]+got[3]-got[4] {
		t.Fatalf("C/S/D/M/H=%v want=%v", got, want)
	}
}

// Snapshot every persistent public row, including inventory, FB, contribution,
// all revisions, seals, commands, journals and receipts, with deterministic order.
func closureSnapshot(t *testing.T, s *Store) string {
	t.Helper()
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var n string
		if err = rows.Scan(&n); err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	var result strings.Builder
	for _, n := range names {
		var data string
		err = s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(v ORDER BY v::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(r) v FROM `+pgx.Identifier{n}.Sanitize()+` r) q`).Scan(&data)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(&result, "%s=%s\n", n, data)
	}
	return result.String()
}
func closureInventory(t *testing.T, s *Store) string {
	t.Helper()
	var data string
	if err := s.pool.QueryRow(context.Background(), `SELECT jsonb_build_object('items',(SELECT jsonb_agg(to_jsonb(i) ORDER BY i.instance_id) FROM character_inventory_items i),'owners',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.id) FROM characters c),'lots',(SELECT jsonb_agg(to_jsonb(l) ORDER BY l.issuance_id) FROM mining_reward_issuance_lots l),'receipts',(SELECT jsonb_agg(to_jsonb(r) ORDER BY r.settlement_id) FROM mining_reward_settlement_receipts r))::text`).Scan(&data); err != nil {
		t.Fatal(err)
	}
	return data
}
func closureReconcile(t *testing.T, s *Store) {
	t.Helper()
	before := closureSnapshot(t, s)
	r, err := s.ReconcileBlackIronEmission(context.Background())
	if err != nil || !r.Balanced {
		t.Fatalf("recovery=%+v err=%v", r, err)
	}
	if closureSnapshot(t, s) != before {
		t.Fatal("read-only recovery wrote data")
	}
}

func TestG21ClosureRecoveryExactDebtVectors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		d, r   int
		refund int64
		want   [5]int64
	}{{"T085", 1, 0, 20, [5]int64{80, 0, 10, 70, 0}}, {"T086_T089_T095", 10, 0, 30, [5]int64{70, 0, 100, 0, 30}}, {"T087", 6, 2, 30, [5]int64{70, 20, 60, 0, 10}}} {
		t.Run(tc.name, func(t *testing.T) {
			s, cmds, receipts, _ := closureRecoveryFixture(t, tc.d, tc.r)
			closurePool(t, s, [5]int64{100, int64(tc.r * 10), int64(tc.d * 10), int64(100 - (tc.d+tc.r)*10), 0})
			inventory := closureInventory(t, s)
			closureRefund(t, s, "closure-refund", tc.refund)
			closurePool(t, s, tc.want)
			if closureInventory(t, s) != inventory {
				t.Fatal("refund changed inventory/revisions/FINAL")
			}
			baseline := closureSnapshot(t, s)
			for n := 0; n < 100; n++ {
				closureRefund(t, s, "closure-refund", tc.refund)
			}
			if closureSnapshot(t, s) != baseline {
				t.Fatal("100 refund retries changed state")
			}
			for n, cmd := range cmds {
				r, err := s.SettleMiningRewardTEST(context.Background(), cmd)
				if err != nil || !reflect.DeepEqual(r, receipts[n]) {
					t.Fatalf("debt historical replay mismatch: %v", err)
				}
			}
			if closureSnapshot(t, s) != baseline {
				t.Fatal("historical replay changed state")
			}
			closureReconcile(t, s)
		})
	}
}
func TestG21ClosureRecoveryCapacityRepayment(t *testing.T) {
	for _, tc := range []struct {
		name string
		a    int64
		want [5]int64
	}{{"T090", 10, [5]int64{80, 0, 100, 0, 20}}, {"T091", 30, [5]int64{100, 0, 100, 0, 0}}, {"T092", 40, [5]int64{110, 0, 100, 10, 0}}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _, _ := closureRecoveryFixture(t, 10, 0)
			closureRefund(t, s, "closure-refund", 30)
			closurePool(t, s, [5]int64{70, 0, 100, 0, 30})
			inventory := closureInventory(t, s)
			closureCapacity(t, s, "closure-repay", tc.a)
			closurePool(t, s, tc.want)
			if closureInventory(t, s) != inventory {
				t.Fatal("capacity changed inventory/FINAL")
			}
			before := closureSnapshot(t, s)
			closureCapacity(t, s, "closure-repay", tc.a)
			if closureSnapshot(t, s) != before {
				t.Fatal("capacity replay changed state")
			}
			closureReconcile(t, s)
		})
	}
}
func TestG21ClosureRecoveryDebtPauseClearAndCancel(t *testing.T) {
	t.Run("T088_T093", func(t *testing.T) {
		s, _, _, open := closureRecoveryFixture(t, 9, 1)
		ctx := context.Background()
		var session miningpower.ActivitySession
		var source miningpower.SourceEvent
		if err := s.pool.QueryRow(ctx, `SELECT data FROM mining_power_sessions WHERE block_instance_id=(SELECT block_instance_id FROM mining_blocks WHERE block_id=$1)`, open[0]).Scan(&session); err != nil {
			t.Fatal(err)
		}
		if err := s.pool.QueryRow(ctx, `SELECT data FROM mining_power_source_events WHERE block_instance_id=$1`, session.BlockInstanceID).Scan(&source); err != nil {
			t.Fatal(err)
		}
		cmd := closureSeal(t, s, miningpower.Principal{AccountID: session.AccountID, PlayerID: session.PlayerID}, miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID, ActivitySessionID: session.ID, BlockID: session.BlockID, BlockInstanceID: session.BlockInstanceID}, "closure-resume")
		closureRefund(t, s, "closure-pause-refund", 10)
		closurePool(t, s, [5]int64{90, 10, 90, 0, 10})
		before := closureSnapshot(t, s)
		if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardDebt) {
			t.Fatalf("want debt pause: %v", err)
		}
		if closureSnapshot(t, s) != before {
			t.Fatal("debt pause changed persistent state")
		}
		closureCapacity(t, s, "closure-clear", 10)
		closurePool(t, s, [5]int64{100, 10, 90, 0, 0})
		r, err := s.SettleMiningRewardTEST(ctx, cmd)
		if err != nil || r.TotalOre != 10 {
			t.Fatalf("resume=%+v %v", r, err)
		}
		closurePool(t, s, [5]int64{100, 0, 100, 0, 0})
		closureReconcile(t, s)
	})
	t.Run("T099_T100", func(t *testing.T) {
		s, _, receipts, open := closureRecoveryFixture(t, 6, 2)
		closureRefund(t, s, "closure-cancel-refund", 30)
		closurePool(t, s, [5]int64{70, 20, 60, 0, 10})
		inventory := closureInventory(t, s)
		for _, block := range open {
			if _, err := s.CancelMiningBlock(context.Background(), block); err != nil {
				t.Fatal(err)
			}
		}
		closurePool(t, s, [5]int64{70, 0, 60, 10, 0})
		if closureInventory(t, s) != inventory {
			t.Fatal("cancel changed Ore/revisions/FINAL")
		}
		before := closureSnapshot(t, s)
		if _, err := s.CancelMiningBlock(context.Background(), receipts[0].DisplayBlockID); !errors.Is(err, miningblock.ErrConflict) {
			t.Fatalf("consumed cancellation=%v", err)
		}
		if closureSnapshot(t, s) != before {
			t.Fatal("consumed cancel changed data")
		}
		closureReconcile(t, s)
	})
}
func TestG21ClosureRecoveryRebuildAndCorruption(t *testing.T) {
	t.Run("T096", func(t *testing.T) {
		s, _, _, _ := closureRecoveryFixture(t, 6, 2)
		closurePool(t, s, [5]int64{100, 20, 60, 20, 0})
		closureReconcile(t, s)
		closureRefund(t, s, "closure-mixed-refund", 30)
		closurePool(t, s, [5]int64{70, 20, 60, 0, 10})
		closureReconcile(t, s)
	})
	for _, tc := range []struct{ name, sql string }{{"T097_missing_receipt", `DELETE FROM mining_reward_settlement_receipts`}, {"T097_corrupt_receipt", `UPDATE mining_reward_settlement_receipts SET canonical_digest=repeat('0',64)`}, {"T097_missing_consume", `DELETE FROM mining_reward_reservation_consumptions`}, {"T097_corrupt_consume", `UPDATE mining_reward_reservation_consumptions SET amount=amount-1`}, {"T098_gap", `DELETE FROM black_iron_emission_recovery_entries WHERE pool_revision=2`}, {"T098_before_state", `UPDATE black_iron_emission_recovery_entries SET remaining_before=remaining_before+1 WHERE pool_revision=2`}} {
		t.Run(tc.name, func(t *testing.T) {
			s, _, _, _ := closureRecoveryFixture(t, 1, 0)
			ctx := context.Background()
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			before := closureSnapshot(t, s)
			report, err := s.ReconcileBlackIronEmission(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if report.Balanced || len(report.Mismatches) == 0 {
				t.Fatalf("corrupt recovery accepted: %+v", report)
			}
			if closureSnapshot(t, s) != before {
				t.Fatal("corruption audit repaired data")
			}
		})
	}
}

func TestG21ClosureRecoveryRefundRaceProcess(t *testing.T) {
	kind := os.Getenv("G21_CLOSURE_RACE_KIND")
	if kind == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var cmd MiningRewardSettlementCommand
	if err = json.Unmarshal([]byte(os.Getenv("G21_CLOSURE_RACE_COMMAND")), &cmd); err != nil {
		t.Fatal(err)
	}
	marker := os.Getenv("G21_CLOSURE_RACE_MARKER")
	if err = os.WriteFile(marker+"."+kind, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		if _, e := os.Stat(marker); e == nil {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
	result := "OK"
	if kind == "refund" {
		closureRefund(t, s, "closure-race-refund", 10)
	} else {
		_, err = s.SettleMiningRewardTEST(ctx, cmd)
		if err != nil {
			if !errors.Is(err, ErrMiningRewardDebt) {
				t.Fatal(err)
			}
			result = "RECOVERY_DEBT_PAUSED"
		}
	}
	if err = os.WriteFile(marker+"."+kind+".result", []byte(result), 0600); err != nil {
		t.Fatal(err)
	}
}
func TestG21ClosureRecoveryRefundRaceIndependentProcesses(t *testing.T) {
	s, _, _, open := closureRecoveryFixture(t, 9, 1)
	ctx := context.Background()
	var session miningpower.ActivitySession
	var source miningpower.SourceEvent
	if err := s.pool.QueryRow(ctx, `SELECT data FROM mining_power_sessions WHERE block_instance_id=(SELECT block_instance_id FROM mining_blocks WHERE block_id=$1)`, open[0]).Scan(&session); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT data FROM mining_power_source_events WHERE block_instance_id=$1`, session.BlockInstanceID).Scan(&source); err != nil {
		t.Fatal(err)
	}
	command := closureSeal(t, s, miningpower.Principal{AccountID: session.AccountID, PlayerID: session.PlayerID}, miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID, ActivitySessionID: session.ID, BlockID: session.BlockID, BlockInstanceID: session.BlockInstanceID}, "closure-race-pay")
	closurePool(t, s, [5]int64{100, 10, 90, 0, 0})
	blocker, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "race-start")
	raw, err := json.Marshal(command)
	if err != nil {
		t.Fatal(err)
	}
	var workers []*exec.Cmd
	outputs := make([]bytes.Buffer, 2)
	for i, kind := range []string{"settle", "refund"} {
		worker := exec.Command(os.Args[0], "-test.run=^TestG21ClosureRecoveryRefundRaceProcess$", "-test.v")
		worker.Env = append(os.Environ(), "G21_CLOSURE_RACE_KIND="+kind, "G21_CLOSURE_RACE_MARKER="+marker, "G21_CLOSURE_RACE_COMMAND="+string(raw))
		worker.Stdout = &outputs[i]
		worker.Stderr = &outputs[i]
		if err = worker.Start(); err != nil {
			t.Fatal(err)
		}
		defer worker.Process.Kill()
		t.Logf("race controller_pid=%d %s_worker_pid=%d", os.Getpid(), kind, worker.Process.Pid)
		workers = append(workers, worker)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, a := os.Stat(marker + ".settle")
		_, b := os.Stat(marker + ".refund")
		if a == nil && b == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("race workers not ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err = os.WriteFile(marker, []byte("start"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		var waiting int
		err = s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("both workers did not contend on database locks")
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err = blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	for i, worker := range workers {
		if err = worker.Wait(); err != nil {
			t.Fatalf("race worker %d: %v %s", i, err, outputs[i].String())
		}
	}
	status, err := os.ReadFile(marker + ".settle.result")
	if err != nil {
		t.Fatal(err)
	}
	if string(status) == "OK" {
		closurePool(t, s, [5]int64{90, 0, 100, 0, 10})
	} else if string(status) == "RECOVERY_DEBT_PAUSED" {
		closurePool(t, s, [5]int64{90, 10, 90, 0, 10})
	} else {
		t.Fatalf("worker result=%s", status)
	}
	closureReconcile(t, s)
	closureZero(t, s, closureSnapshot(t, s))
	t.Logf("two independent worker processes serialized: %s", status)
}
func TestG21ClosureRecoveryDuplicateRevisionRejected(t *testing.T) {
	s, _, _, _ := closureRecoveryFixture(t, 1, 0)
	before := closureSnapshot(t, s)
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(ctx, `UPDATE black_iron_emission_recovery_entries SET pool_revision=1 WHERE pool_revision=2`)
	closureErrorCode(t, err, "23505")
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	closureZero(t, s, before)
}
