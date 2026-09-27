package postgres

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"fractallegend/game-server/internal/miningpower"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type g20ProcessRequest struct {
	Principal miningpower.Principal
	Intent    miningpower.ActionIntent
	Stage     string
	Committed bool
}

func g20ProcessInput(t *testing.T) (*Store, g20ProcessRequest) {
	t.Helper()
	var input g20ProcessRequest
	if e := json.Unmarshal([]byte(os.Getenv("FRACTAL_G20_PROCESS_REQUEST")), &input); e != nil {
		t.Fatal(e)
	}
	s, e := Open(context.Background(), os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	return s, input
}
func TestG20CrashWorker(t *testing.T) {
	if os.Getenv("FRACTAL_G20_PROCESS_ROLE") != "worker" {
		return
	}
	s, input := g20ProcessInput(t)
	s.miningPowerFailureInjector = func(stage string) error {
		if stage == input.Stage {
			fmt.Println("G20_READY_TO_KILL")
			select {}
		}
		return nil
	}
	r, e := s.ValidateMiningActivity(context.Background(), input.Principal, input.Intent, miningpower.DevelopmentRuleVersion)
	t.Fatalf("worker escaped crash stage: %+v %v", r, e)
}
func TestG20CrashVerifier(t *testing.T) {
	if os.Getenv("FRACTAL_G20_PROCESS_ROLE") != "verifier" {
		return
	}
	s, input := g20ProcessInput(t)
	beforeA, beforeP := g20Counts(t, s)
	want := 0
	if input.Committed {
		want = 1
	}
	if beforeA != want || beforeP != want {
		t.Fatalf("process crash partial state=%d/%d want=%d", beforeA, beforeP, want)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	r, e := s.ValidateMiningActivity(ctx, input.Principal, input.Intent, miningpower.DevelopmentRuleVersion)
	if e != nil {
		t.Fatal(e)
	}
	if input.Committed {
		if r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0 {
			t.Fatalf("committed replay=%+v", r)
		}
	} else if r.Status != miningpower.StatusValid || r.AppliedPower != 100 {
		t.Fatalf("uncommitted recovery=%+v", r)
	}
	snapshot, e := s.SnapshotMiningPower(ctx, input.Intent.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || len(snapshot.Activities) != 1 || len(snapshot.Participants) != 1 || snapshot.Participants[0].ValidatedPower != 100 || snapshot.Participants[0].ActivityCount != 1 {
		t.Fatalf("process replay state=%+v %v", snapshot, e)
	}
	report, e := s.ReconcileMiningPower(ctx, input.Intent.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || report.Status != "PASS" {
		t.Fatalf("process report=%+v %v", report, e)
	}
	fmt.Println("G20_PROCESS_VERIFIED_ONE_FACT_ONE_INCREMENT")
}
func TestG20IndependentProcessCrashMatrix(t *testing.T) {
	binary, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	for _, stage := range []string{"before_activity", "after_activity", "after_participant", "before_commit", "after_commit_before_response"} {
		t.Run(stage, func(t *testing.T) {
			_, p, i := g20Fixture(t)
			input := g20ProcessRequest{Principal: p, Intent: i, Stage: stage, Committed: stage == "after_commit_before_response"}
			raw, e := json.Marshal(input)
			if e != nil {
				t.Fatal(e)
			}
			env := append(os.Environ(), "FRACTAL_G20_PROCESS_REQUEST="+string(raw))
			worker := exec.Command(binary, "-test.run=^TestG20CrashWorker$", "-test.timeout=30s")
			worker.Env = append(env, "FRACTAL_G20_PROCESS_ROLE=worker")
			pipe, e := worker.StdoutPipe()
			if e != nil {
				t.Fatal(e)
			}
			if e = worker.Start(); e != nil {
				t.Fatal(e)
			}
			ready := make(chan bool, 1)
			go func() {
				scan := bufio.NewScanner(pipe)
				for scan.Scan() {
					if scan.Text() == "G20_READY_TO_KILL" {
						ready <- true
						return
					}
				}
				ready <- false
			}()
			select {
			case ok := <-ready:
				if !ok {
					_ = worker.Wait()
					t.Fatal("worker did not reach requested crash point")
				}
			case <-time.After(15 * time.Second):
				_ = worker.Process.Kill()
				_ = worker.Wait()
				t.Fatal("crash worker readiness timeout")
			}
			if e = worker.Process.Kill(); e != nil {
				t.Fatal(e)
			}
			if e = worker.Wait(); e == nil {
				t.Fatal("killed worker exited successfully")
			}
			verifier := exec.Command(binary, "-test.run=^TestG20CrashVerifier$", "-test.timeout=30s")
			verifier.Env = append(env, "FRACTAL_G20_PROCESS_ROLE=verifier")
			output, e := verifier.CombinedOutput()
			if e != nil || !strings.Contains(string(output), "G20_PROCESS_VERIFIED_ONE_FACT_ONE_INCREMENT") {
				t.Fatalf("independent verifier failed: %s %v", output, e)
			}
			t.Logf("OS-killed worker pid=%d; separate verifier process confirmed stage=%s", worker.Process.Pid, stage)
		})
	}
}
