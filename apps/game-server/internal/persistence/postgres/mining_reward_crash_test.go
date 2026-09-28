package postgres

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestG21SettlementCrashChild(t *testing.T) {
	point := os.Getenv("G21_SETTLEMENT_CRASH_POINT")
	if point == "" {
		return
	}
	ctx := context.Background()
	s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	cmd := MiningRewardSettlementCommand{
		CommandID:          "g21-crash-command",
		BlockInstanceID:    os.Getenv("G21_SETTLEMENT_CRASH_INSTANCE"),
		ExpectedSealDigest: os.Getenv("G21_SETTLEMENT_CRASH_SEAL_DIGEST"),
	}
	if os.Getenv("G21_SETTLEMENT_VERIFY") == "" {
		s.settlementFailureInjector = func(at string) error {
			if at != point {
				return nil
			}
			if err := os.WriteFile(os.Getenv("G21_SETTLEMENT_MARKER"), []byte(at), 0600); err != nil {
				return err
			}
			process, _ := os.FindProcess(os.Getpid())
			_ = process.Kill()
			os.Exit(91)
			return nil
		}
		_, err = s.SettleMiningRewardTEST(ctx, cmd)
		t.Fatalf("worker survived point=%s err=%v", point, err)
	}
	var before int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_settlement_receipts`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	committed := point == "after_commit_ack_lost"
	if (before == 1) != committed {
		t.Fatalf("post-kill receipts=%d committed=%v", before, committed)
	}
	var prior MiningRewardSettlementReceipt
	if committed {
		tx, e := s.pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		prior, e = loadMiningRewardReceiptTx(ctx, tx, cmd.CommandID)
		_ = tx.Rollback(ctx)
		if e != nil {
			t.Fatal(e)
		}
	}
	replayed, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || replayed.Status != "COMPLETED" || replayed.TotalOre != 10 {
		t.Fatalf("restart replay=%+v %v", replayed, err)
	}
	if committed && !reflect.DeepEqual(prior, replayed) {
		t.Fatal("lost ACK changed receipt")
	}
	var reserved, distributed, remaining, grants, lots, stacks, consumed, receipts int
	if err = s.pool.QueryRow(ctx, `SELECT
		(SELECT total_reserved FROM black_iron_emission_pools),
		(SELECT total_distributed FROM black_iron_emission_pools),
		(SELECT remaining_capacity FROM black_iron_emission_pools),
		(SELECT count(*) FROM mining_reward_grants),
		(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
		(SELECT count(*) FROM mining_reward_inventory_projections),
		(SELECT count(*) FROM mining_reward_reservation_consumptions),
		(SELECT count(*) FROM mining_reward_settlement_receipts)`).
		Scan(&reserved, &distributed, &remaining, &grants, &lots, &stacks, &consumed, &receipts); err != nil {
		t.Fatal(err)
	}
	if reserved != 0 || distributed != 10 || remaining != 10 || grants != 1 || lots != 1 || stacks != 1 || consumed != 1 || receipts != 1 {
		t.Fatalf("post-restart state S/D/M/grant/lot/stack/consume/receipt=%d/%d/%d/%d/%d/%d/%d/%d",
			reserved, distributed, remaining, grants, lots, stacks, consumed, receipts)
	}
	emission, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !emission.Balanced {
		t.Fatalf("emission recovery=%+v %v", emission, err)
	}
	reward, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || !reward.Balanced {
		t.Fatalf("reward recovery=%+v %v", reward, err)
	}
}

func TestG21SettlementNineRealProcessCrashBarriers(t *testing.T) {
	for _, point := range []string{
		"before_issuance", "after_issuance", "after_inventory_projection",
		"after_reserved_to_distributed", "after_recovery_history", "after_grant",
		"after_receipt", "before_commit", "after_commit_ack_lost",
	} {
		t.Run(point, func(t *testing.T) {
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, principal, intent)
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(t.TempDir(), "barrier")
			env := append(os.Environ(), "G21_SETTLEMENT_CRASH_POINT="+point,
				"G21_SETTLEMENT_CRASH_INSTANCE="+intent.BlockInstanceID,
				"G21_SETTLEMENT_CRASH_SEAL_DIGEST="+seal.CanonicalDigest,
				"G21_SETTLEMENT_MARKER="+marker)
			childCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			defer cancel()
			worker := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestG21SettlementCrashChild$", "-test.v")
			worker.Env = env
			out, err := worker.CombinedOutput()
			if _, markerErr := os.Stat(marker); markerErr != nil || err == nil ||
				strings.Contains(string(out), "worker survived") {
				t.Fatalf("worker did not die at %s: marker=%v exit=%v output=%s", point, markerErr, err, out)
			}
			verifier := exec.CommandContext(childCtx, os.Args[0], "-test.run=^TestG21SettlementCrashChild$", "-test.v")
			verifier.Env = append(env, "G21_SETTLEMENT_VERIFY=1")
			out, err = verifier.CombinedOutput()
			if err != nil {
				t.Fatalf("independent restart verifier %s: %v %s", point, err, out)
			}
			t.Logf("independent worker kill and verifier PASS: %s", point)
		})
	}
}
