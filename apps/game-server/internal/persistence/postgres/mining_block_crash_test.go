package postgres

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningblock"
)

// The child is killed by the operating system while the database transaction
// holds its connection. A driver error injected in the parent is not a crash.
func TestG18CrashHelperProcess(t *testing.T) {
	if verify := os.Getenv("G18_VERIFY_CASE"); verify != "" {
		// This is a newly started process, distinct from both the crashed
		// worker and the parent test. It reads the same PostgreSQL database.
		ctx := context.Background()
		store, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		blocks, err := store.SnapshotMiningBlocks(ctx)
		if err != nil {
			t.Fatal(err)
		}
		pool, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
		if err != nil {
			t.Fatal(err)
		}
		committed := os.Getenv("G18_VERIFY_COMMITTED") == "true"
		switch verify {
		case "create":
			want := 0
			if committed {
				want = 1
			}
			if len(blocks.Blocks) != want || len(blocks.Reservations) != want || len(blocks.Entries) != want || len(blocks.Receipts) != want ||
				pool.Pool.TotalReserved != int64(want*10) || pool.Pool.RemainingCapacity != int64((1-want)*10) {
				t.Fatalf("independent restart create state: blocks=%+v pool=%+v", blocks, pool.Pool)
			}
		case "finalize", "cancel":
			want := 1
			if committed {
				want = 2
			}
			if len(blocks.Blocks) != 1 || len(blocks.Reservations) != 1 || len(blocks.Entries) != want || len(blocks.Receipts) != want {
				t.Fatalf("independent restart transition state: %+v", blocks)
			}
			status := miningblock.StatusOpen
			if committed && verify == "finalize" {
				status = miningblock.StatusFinalized
			} else if committed {
				status = miningblock.StatusCancelled
			}
			if blocks.Blocks[0].Status != status || pool.Pool.TotalDistributed != 0 {
				t.Fatalf("independent restart status/pool: %+v %+v", blocks.Blocks[0], pool.Pool)
			}
			if verify == "cancel" {
				reserved, debt := int64(10), int64(10)
				if committed {
					reserved, debt = 0, 0
				}
				if pool.Pool.TotalReserved != reserved || pool.Pool.RecoveryDebt != debt || pool.Pool.RemainingCapacity != 0 {
					t.Fatalf("independent restart cancellation debt: %+v", pool.Pool)
				}
			}
		default:
			t.Fatal("unknown verification action")
		}
		return
	}
	spec := os.Getenv("G18_CRASH_CASE")
	if spec == "" {
		return
	}
	parts := strings.Split(spec, "/")
	if len(parts) != 2 {
		t.Fatal("invalid child case")
	}
	ctx := context.Background()
	store, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	store.miningBlockFailureInjector = func(at string) error {
		if at == parts[1] {
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Kill()
			os.Exit(91)
		}
		return nil
	}
	svc := miningblock.NewService(store, miningblock.DevelopmentRuleVersion, false)
	switch parts[0] {
	case "create":
		_, err = svc.Create(ctx, "g18-process-create")
	case "finalize":
		_, err = svc.Finalize(ctx, os.Getenv("G18_CRASH_BLOCK_ID"))
	case "cancel":
		_, err = svc.Cancel(ctx, os.Getenv("G18_CRASH_BLOCK_ID"))
	default:
		t.Fatal("invalid child action")
	}
	t.Fatalf("child survived expected kill: %v", err)
}

func runG18CrashedChild(t *testing.T, action, point, blockID string) *Store {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestG18CrashHelperProcess$", "-test.v")
	cmd.Env = append(os.Environ(), "G18_CRASH_CASE="+action+"/"+point, "G18_CRASH_BLOCK_ID="+blockID)
	output, err := cmd.CombinedOutput()
	if err == nil || strings.Contains(string(output), "child survived expected kill") {
		t.Fatalf("child was not killed at %s/%s: err=%v output=%s", action, point, err, output)
	}
	if !strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "exit status 91") {
		t.Fatalf("unexpected child failure: %v output=%s", err, output)
	}
	verification := exec.Command(os.Args[0], "-test.run=^TestG18CrashHelperProcess$", "-test.v")
	verification.Env = append(os.Environ(), "G18_VERIFY_CASE="+action,
		"G18_VERIFY_COMMITTED="+fmt.Sprint(point == "after_commit_before_response"))
	if output, err = verification.CombinedOutput(); err != nil {
		t.Fatalf("independent restart verification failed: %v output=%s", err, output)
	}
	reopened, err := Open(context.Background(), os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	return reopened
}

func TestG18IndependentProcessCrashMatrix(t *testing.T) {
	for _, point := range []string{"before_block", "after_reserve", "after_receipt", "after_commit_before_response"} {
		t.Run("create/"+point, func(t *testing.T) {
			_, _, _, _ = g18Fund(t, 10)
			store := runG18CrashedChild(t, "create", point, "")
			service := miningblock.NewService(store, miningblock.DevelopmentRuleVersion, false)
			snapshot, err := store.SnapshotMiningBlocks(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			committed := point == "after_commit_before_response"
			want := 0
			if committed {
				want = 1
			}
			if len(snapshot.Blocks) != want || len(snapshot.Reservations) != want || len(snapshot.Receipts) != want {
				t.Fatalf("partial/ghost create after crash: %+v", snapshot)
			}
			first, err := service.Create(context.Background(), "g18-process-create")
			if err != nil || first.BlockHeight != 1 || first.PoolBefore != 10 || first.PoolAfter != 0 {
				t.Fatalf("retry create=%+v err=%v", first, err)
			}
			if committed && !reflect.DeepEqual(first, snapshot.Receipts[0]) {
				t.Fatal("commit-before-response receipt changed")
			}
			assertRecoveryPool(t, store, 10, 10, 0, 0)
		})
	}
	for _, action := range []string{"finalize", "cancel"} {
		for _, point := range []string{"after_state_change", "after_receipt", "after_commit_before_response"} {
			t.Run(action+"/"+point, func(t *testing.T) {
				_, service, contributions, spend := g18Fund(t, 10)
				open, err := service.Create(context.Background(), "g18-process-open")
				if err != nil {
					t.Fatal(err)
				}
				if action == "cancel" {
					if _, err = contributions.RefundSystemSpend(context.Background(), g15Refund(spend, "g18-process-refund", 10)); err != nil {
						t.Fatal(err)
					}
				}
				reopened := runG18CrashedChild(t, action, point, open.BlockID)
				before, err := reopened.SnapshotMiningBlocks(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				committed := point == "after_commit_before_response"
				wantEntries := 1
				if committed {
					wantEntries = 2
				}
				if len(before.Entries) != wantEntries || len(before.Receipts) != wantEntries || len(before.Blocks) != 1 || len(before.Reservations) != 1 {
					t.Fatalf("partial transition: %+v", before)
				}
				status := miningblock.StatusOpen
				if committed && action == "cancel" {
					status = miningblock.StatusCancelled
				} else if committed {
					status = miningblock.StatusFinalized
				}
				if before.Blocks[0].Status != status {
					t.Fatalf("wrong state %s expected %s", before.Blocks[0].Status, status)
				}
				service = miningblock.NewService(reopened, miningblock.DevelopmentRuleVersion, false)
				var receipt miningblock.Receipt
				if action == "cancel" {
					receipt, err = service.Cancel(context.Background(), open.BlockID)
				} else {
					receipt, err = service.Finalize(context.Background(), open.BlockID)
				}
				if err != nil || receipt.BlockID != open.BlockID {
					t.Fatalf("recovered transition=%+v err=%v", receipt, err)
				}
				if committed && !reflect.DeepEqual(receipt, before.Receipts[1]) {
					t.Fatal("commit-before-response transition receipt changed")
				}
				after, err := reopened.SnapshotMiningBlocks(context.Background())
				if err != nil || len(after.Entries) != 2 || len(after.Receipts) != 2 {
					t.Fatalf("duplicated transition: %+v err=%v", after, err)
				}
				if action == "cancel" {
					assertRecoveryPool(t, reopened, 0, 0, 0, 0)
				} else {
					assertRecoveryPool(t, reopened, 10, 10, 0, 0)
				}
			})
		}
	}
}
