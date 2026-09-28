package postgres

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
)

func g21P0CrashIntent() (miningpower.Principal, miningpower.ActionIntent) {
	return miningpower.Principal{AccountID: "g20-player-account", PlayerID: "g20-player"},
		miningpower.ActionIntent{ActivityID: "mpa:g20-event", SourceEventID: "g20-event", ActivitySessionID: "g20-session",
			BlockID: os.Getenv("G21P0_CRASH_BLOCK"), BlockInstanceID: os.Getenv("G21P0_CRASH_INSTANCE")}
}

func TestG21P0CrashChild(t *testing.T) {
	mode := os.Getenv("G21P0_CRASH_MODE")
	if mode == "" {
		return
	}
	ctx := context.Background()
	s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	verify := os.Getenv("G21P0_VERIFY") != ""
	point := os.Getenv("G21P0_CRASH_POINT")
	committed := strings.Contains(point, "after_commit")
	if !verify {
		kill := func(at string) error {
			if at == point {
				if mode == "acceptance_inflight" {
					if e := os.WriteFile(os.Getenv("G21P0_MARKER"), []byte("activity inserted in open transaction"), 0600); e != nil {
						return e
					}
					select {}
				}
				p, _ := os.FindProcess(os.Getpid())
				_ = p.Kill()
				os.Exit(91)
			}
			return nil
		}
		s.prerequisiteFailureInjector = kill
		s.miningBlockFailureInjector = kill
		s.miningPowerFailureInjector = kill
	}
	principal, intent := g21P0CrashIntent()
	switch mode {
	case "acceptance", "acceptance_inflight":
		if verify {
			var n int
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_activities`).Scan(&n); err != nil || n != 0 {
				t.Fatalf("uncommitted acceptance visible=%d %v", n, err)
			}
			if mode == "acceptance_inflight" {
				seal, e := s.LoadMiningPowerSeal(ctx, intent.BlockInstanceID)
				if e != nil || seal.ActivityCount != 0 {
					t.Fatalf("inflight seal=%+v %v", seal, e)
				}
				got, e := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
				if e != nil || got.ReasonCode != "BLOCK_SEALED" {
					t.Fatalf("killed acceptance replay=%+v %v", got, e)
				}
				return
			}
			got, e := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
			if e != nil || got.Status != miningpower.StatusValid {
				t.Fatalf("accept retry=%+v %v", got, e)
			}
			if _, e = s.FinalizeMiningBlock(ctx, intent.BlockID); e != nil {
				t.Fatal(e)
			}
			seal, e := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if e != nil || seal.ActivityCount != 1 {
				t.Fatalf("seal after retry=%+v %v", seal, e)
			}
			return
		}
		_, err = s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
	case "seal":
		if verify {
			var state string
			if err = s.pool.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1`, intent.BlockInstanceID).Scan(&state); err != nil {
				t.Fatal(err)
			}
			want := "OPEN"
			if committed {
				want = "SEALED"
			}
			if state != want {
				t.Fatalf("seal state=%s want=%s", state, want)
			}
			seal, e := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if e != nil || seal.ActivityCount != 1 {
				t.Fatalf("seal retry=%+v %v", seal, e)
			}
			replay, e := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
			if e != nil || replay.Status != miningpower.StatusDuplicate || replay.AppliedPower != 0 {
				t.Fatalf("accepted replay=%+v %v", replay, e)
			}
			return
		}
		_, err = s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	case "binding":
		service := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false)
		if verify {
			var n int
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reservation_instance_bindings`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			want := 0
			if committed {
				want = 1
			}
			if n != want {
				t.Fatalf("binding count=%d want=%d", n, want)
			}
			r, e := service.Create(ctx, "g21-p0-crash-binding")
			if e != nil || r.BlockID == "" {
				t.Fatalf("binding retry=%+v %v", r, e)
			}
			var bound int
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reservation_instance_bindings`).Scan(&bound); err != nil || bound != 1 {
				t.Fatalf("binding replay=%d %v", bound, err)
			}
			return
		}
		_, err = service.Create(ctx, "g21-p0-crash-binding")
	case "distribution":
		source := os.Getenv("G21P0_CRASH_SOURCE")
		if verify {
			var n int
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_prerequisite_distributions`).Scan(&n); err != nil {
				t.Fatal(err)
			}
			want := 0
			if committed {
				want = 1
			}
			if n != want {
				t.Fatalf("distribution count=%d want=%d", n, want)
			}
			if err = s.transferReservedToDistributedPrerequisiteTEST(ctx, source); err != nil {
				t.Fatal(err)
			}
			if err = s.transferReservedToDistributedPrerequisiteTEST(ctx, source); err != nil {
				t.Fatal(err)
			}
			p, e := s.SnapshotBlackIronEmission(ctx)
			if e != nil || p.Pool.TotalReserved != 0 || p.Pool.TotalDistributed != 10 || p.Pool.RemainingCapacity != 10 {
				t.Fatalf("distribution retry=%+v %v", p.Pool, e)
			}
			r, e := s.ReconcileBlackIronEmission(ctx)
			if e != nil || !r.Balanced {
				t.Fatalf("rebuild=%+v %v", r, e)
			}
			return
		}
		err = s.transferReservedToDistributedPrerequisiteTEST(ctx, source)
	case "issuance":
		if verify {
			var lots, projections int
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_issuance_lots`).Scan(&lots); err != nil {
				t.Fatal(err)
			}
			if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_inventory_projections`).Scan(&projections); err != nil {
				t.Fatal(err)
			}
			want := 0
			if committed {
				want = 1
			}
			if lots != want || projections != want {
				t.Fatalf("issuance state lots/projections=%d/%d want=%d", lots, projections, want)
			}
			first, e := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-crash-reward", intent.BlockInstanceID, "g20-player", 3)
			if e != nil {
				t.Fatal(e)
			}
			second, e := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-crash-reward", intent.BlockInstanceID, "g20-player", 3)
			if e != nil || first != second {
				t.Fatalf("issuance replay=%+v %+v %v", first, second, e)
			}
			r, e := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
			if e != nil || !r.Balanced || r.Checked != 1 {
				t.Fatalf("issuance audit=%+v %v", r, e)
			}
			return
		}
		_, err = s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-crash-reward", intent.BlockInstanceID, "g20-player", 3)
	default:
		t.Fatalf("unknown crash mode %s", mode)
	}
	t.Fatalf("child survived expected kill mode=%s point=%s err=%v", mode, point, err)
}

func TestG21P0RealProcessCrashAndIndependentRestart(t *testing.T) {
	for _, tc := range []struct{ mode, point string }{
		{"acceptance", "before_commit"},
		{"acceptance_inflight", "after_activity"},
		{"seal", "seal_before_commit"}, {"seal", "seal_after_commit_before_response"},
		{"binding", "before_commit"}, {"binding", "after_commit_before_response"},
		{"distribution", "distribution_before_commit"}, {"distribution", "distribution_after_commit_before_response"},
		{"issuance", "issuance_before_commit"}, {"issuance", "issuance_after_commit_before_response"},
	} {
		t.Run(tc.mode+"/"+tc.point, func(t *testing.T) {
			var s *Store
			var blockID, instanceID, sourceID string
			switch tc.mode {
			case "acceptance", "acceptance_inflight", "seal", "issuance":
				var p miningpower.Principal
				var i miningpower.ActionIntent
				s, p, i = g20Fixture(t)
				blockID, instanceID = i.BlockID, i.BlockInstanceID
				if tc.mode == "seal" || tc.mode == "issuance" {
					g20Accept(t, s, p, i)
					if _, err := s.FinalizeMiningBlock(context.Background(), blockID); err != nil {
						t.Fatal(err)
					}
				}
				if tc.mode == "issuance" {
					g19Catalog(t, s)
					if _, err := s.SealMiningPowerTEST(context.Background(), instanceID); err != nil {
						t.Fatal(err)
					}
				}
			case "binding":
				s, _ = g20Fund(t, 20)
			case "distribution":
				var service *miningblock.Service
				s, service = g20Fund(t, 20)
				r, err := service.Create(context.Background(), "g21-p0-crash-distribution")
				if err != nil {
					t.Fatal(err)
				}
				if _, err = service.Finalize(context.Background(), r.BlockID); err != nil {
					t.Fatal(err)
				}
				if err = s.pool.QueryRow(context.Background(), `SELECT reservation_source_id FROM mining_reservation_instance_bindings WHERE display_block_id=$1`, r.BlockID).Scan(&sourceID); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			marker := t.TempDir() + "/activity-inflight"
			base := append(os.Environ(), "G21P0_CRASH_MODE="+tc.mode, "G21P0_CRASH_POINT="+tc.point,
				"G21P0_CRASH_BLOCK="+blockID, "G21P0_CRASH_INSTANCE="+instanceID,
				"G21P0_CRASH_SOURCE="+sourceID, "G21P0_MARKER="+marker)
			child := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG21P0CrashChild$", "-test.v")
			child.Env = base
			if tc.mode == "acceptance_inflight" {
				if err := child.Start(); err != nil {
					t.Fatal(err)
				}
				for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
					if _, err := os.Stat(marker); err == nil {
						break
					}
				}
				if _, err := os.Stat(marker); err != nil {
					_ = child.Process.Kill()
					t.Fatalf("child did not reach activity boundary: %v", err)
				}
				if _, err := s.FinalizeMiningBlock(ctx, blockID); err != nil {
					_ = child.Process.Kill()
					t.Fatal(err)
				}
				sealDone := make(chan error, 1)
				go func() { _, err := s.SealMiningPowerTEST(ctx, instanceID); sealDone <- err }()
				select {
				case err := <-sealDone:
					_ = child.Process.Kill()
					t.Fatalf("seal passed in-flight transaction: %v", err)
				case <-time.After(50 * time.Millisecond):
				}
				_ = child.Process.Kill()
				_, _ = child.Process.Wait()
				if err := <-sealDone; err != nil {
					t.Fatal(err)
				}
			} else {
				output, err := child.CombinedOutput()
				if err == nil || strings.Contains(string(output), "child survived expected kill") || (!strings.Contains(err.Error(), "killed") && !strings.Contains(err.Error(), "exit status 91")) {
					t.Fatalf("unexpected child exit: %v %s", err, output)
				}
			}
			verifier := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestG21P0CrashChild$", "-test.v")
			verifier.Env = append(base, "G21P0_VERIFY=1")
			out, err := verifier.CombinedOutput()
			if err != nil {
				t.Fatalf("independent verifier: %v %s", err, out)
			}
			t.Logf("independent verifier passed mode=%s point=%s", tc.mode, tc.point)
		})
	}
}
