package postgres

import (
	"context"
	"fmt"
	"fractallegend/game-server/internal/miningpower"
	"sync"
	"testing"
	"time"
)

func TestG20ParallelDifferentPlayers(t *testing.T) {
	s, _, i := g20Fixture(t)
	session := g20Session(t, s, i)
	source := g20Source(t, s, i)
	n := 10
	principals := make([]miningpower.Principal, n)
	intents := make([]miningpower.ActionIntent, n)
	for k := 0; k < n; k++ {
		id := fmt.Sprintf("parallel-%d", k)
		g19Seed(t, s, id)
		ss := session
		ss.ID = id + "-session"
		ss.PlayerID = id
		ss.AccountID = id + "-account"
		g20Insert(t, s, "mining_power_sessions", ss)
		event := source
		event.ID = id + "-event"
		event.ActivityID = "mpa:" + event.ID
		event.PlayerID = id
		event.ActivitySessionID = ss.ID
		intents[k] = g20RegisteredSource(t, s, event)
		principals[k] = miningpower.Principal{PlayerID: id, AccountID: ss.AccountID}
	}
	start := make(chan struct{})
	errs := make(chan error, n)
	results := make(chan miningpower.ValidationResult, n)
	var wg sync.WaitGroup
	for k := range intents {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			<-start
			r, e := s.ValidateMiningActivity(context.Background(), principals[k], intents[k], miningpower.DevelopmentRuleVersion)
			results <- r
			errs <- e
		}(k)
	}
	close(start)
	wg.Wait()
	close(errs)
	close(results)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for r := range results {
		if r.Status != miningpower.StatusValid || r.AppliedPower != 100 {
			t.Fatalf("cross-player concurrent acceptance=%+v", r)
		}
	}
	snap, e := s.SnapshotMiningPower(context.Background(), i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || len(snap.Activities) != n || len(snap.Participants) != n {
		t.Fatalf("isolated player aggregates=%+v %v", snap, e)
	}
}

func TestG20ExpiredSession(t *testing.T) {
	s, p, i := g20Fixture(t)
	ss := g20Session(t, s, i)
	ss.ID = "expired-session"
	ss.ExpiresAt = ss.OpenedAt.Add(time.Microsecond)
	g20Insert(t, s, "mining_power_sessions", ss)
	source := g20Source(t, s, i)
	source.ID = "expired-session-event"
	source.ActivityID = "mpa:" + source.ID
	source.ActivitySessionID = ss.ID
	source.ObservedAt = ss.OpenedAt
	intent := g20RegisteredSource(t, s, source)
	r, e := s.ValidateMiningActivity(context.Background(), p, intent, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusExpired || r.AppliedPower != 0 {
		t.Fatalf("session expiry=%+v %v", r, e)
	}
	a, b := g20Counts(t, s)
	if a != 0 || b != 0 {
		t.Fatal("expired session wrote facts")
	}
}

func TestG20FutureObservationAtValidation(t *testing.T) {
	s, p, i := g20Fixture(t)
	source := g20Source(t, s, i)
	source.ID = "future-at-validation"
	source.ActivityID = "mpa:" + source.ID
	source.ObservedAt = time.Now().UTC().Add(300 * time.Millisecond).Truncate(time.Microsecond)
	intent := g20RegisteredSource(t, s, source)
	s.miningPowerFailureInjector = func(stage string) error {
		if stage == "after_validation" {
			time.Sleep(time.Until(source.ObservedAt) + time.Millisecond)
		}
		return nil
	}
	r, e := s.ValidateMiningActivity(context.Background(), p, intent, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusInvalid || r.AppliedPower != 0 {
		t.Fatalf("future observation at adapter snapshot must reject cleanly: %+v %v", r, e)
	}
}
