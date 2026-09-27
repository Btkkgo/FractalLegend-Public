package postgres

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestG19CrashHelperProcess(t *testing.T) {
	point := os.Getenv("G19_CRASH_POINT")
	verify := os.Getenv("G19_VERIFY_COMMIT")
	if point == "" && verify == "" {
		return
	}
	ctx := context.Background()
	s, e := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if verify != "" {
		var name string
		var quantity, markers, revision int
		if e = s.pool.QueryRow(ctx, `SELECT name,quantity FROM character_inventory_items WHERE character_id='crash'`).Scan(&name, &quantity); e != nil {
			t.Fatal(e)
		}
		if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM black_iron_migration_receipts`).Scan(&markers); e != nil {
			t.Fatal(e)
		}
		if e = s.pool.QueryRow(ctx, `SELECT revision FROM characters WHERE id='crash'`).Scan(&revision); e != nil {
			t.Fatal(e)
		}
		wantName, wantMarkers, wantRevision := "Bun", 0, 1
		if verify == "true" {
			wantName, wantMarkers, wantRevision = "黑铁矿石", 1, 2
		}
		if name != wantName || quantity != 9 || markers != wantMarkers || revision != wantRevision {
			t.Fatalf("independent restart: name=%s quantity=%d marker=%d revision=%d", name, quantity, markers, revision)
		}
		first, e := s.MigrateBlackIronInventory(ctx, "crash")
		if e != nil || first.PreTotal != 9 || first.PostTotal != 9 {
			t.Fatalf("retry=%+v err=%v", first, e)
		}
		second, e := s.MigrateBlackIronInventory(ctx, "crash")
		if e != nil || first.CreatedAt != second.CreatedAt || second.PostTotal != 9 {
			t.Fatalf("restart replay=%+v e=%v", second, e)
		}
		g19RequireBalanced(t, s, 1)
		return
	}
	s.blackIronFailureInjector = func(at string) error {
		if at == point {
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Kill()
			os.Exit(91)
		}
		return nil
	}
	_, e = s.MigrateBlackIronInventory(ctx, "crash")
	t.Fatalf("child survived expected kill: %v", e)
}
func TestG19RealProcessCrashBeforeDuringAfterCommitIndependentRestart(t *testing.T) {
	for _, point := range []string{"before_items", "after_items", "after_revision", "after_receipt", "before_commit", "after_commit"} {
		t.Run(point, func(t *testing.T) {
			s := g19Fixture(t)
			g19Seed(t, s, "crash", 9)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG19CrashHelperProcess$", "-test.v")
			child.Env = append(os.Environ(), "G19_CRASH_POINT="+point)
			output, e := child.CombinedOutput()
			if e == nil || strings.Contains(string(output), "child survived expected kill") || (!strings.Contains(e.Error(), "killed") && !strings.Contains(e.Error(), "exit status 91")) {
				t.Fatalf("unexpected crash result: %v %s", e, output)
			}
			// A separate new process observes durable state and retries the operation.
			verifier := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG19CrashHelperProcess$", "-test.v")
			committed := "false"
			if point == "after_commit" {
				committed = "true"
			}
			verifier.Env = append(os.Environ(), "G19_VERIFY_COMMIT="+committed)
			output, e = verifier.CombinedOutput()
			if e != nil {
				t.Fatalf("restart verify=%v %s", e, output)
			}
			g19RequireBalanced(t, s, 1)
		})
	}
}
