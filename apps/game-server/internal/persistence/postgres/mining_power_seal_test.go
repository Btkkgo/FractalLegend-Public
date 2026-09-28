package postgres

import (
	"context"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningpower"
)

// This catches a Seal implementation that rejects historical replay, accepts
// a new source after Seal, or snapshots only a current aggregate.
func TestG21P0SealRetainsAcceptedReplayAndRejectsNewSource(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	accepted, err := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
	if err != nil || accepted.Status != miningpower.StatusValid || accepted.AppliedPower != 100 {
		t.Fatalf("accept=%+v %v", accepted, err)
	}
	if _, err = s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil || seal.ActivityCount != 1 || seal.ParticipantCount != 1 || seal.TotalValidMiningPower != 100 || seal.CanonicalDigest == "" {
		t.Fatalf("seal=%+v %v", seal, err)
	}
	replay, err := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
	if err != nil || replay.Status != miningpower.StatusDuplicate || replay.AppliedPower != 0 || replay.Original == nil || *replay.Original != *accepted.Original {
		t.Fatalf("replay=%+v %v", replay, err)
	}
	source, err := miningPowerRead[miningpower.SourceEvent](ctx, s.pool, `SELECT data FROM mining_power_source_events WHERE source_event_id=$1`, intent.SourceEventID)
	if err != nil {
		t.Fatal(err)
	}
	source.ID, source.ActivityID = "g21-p0-after-seal", "mpa:g21-p0-after-seal"
	g20Insert(t, s, "mining_power_source_events", source)
	newIntent := intent
	newIntent.SourceEventID, newIntent.ActivityID = source.ID, source.ActivityID
	got, err := s.ValidateMiningActivity(ctx, principal, newIntent, miningpower.DevelopmentRuleVersion)
	if err != nil || got.ReasonCode != "BLOCK_SEALED" || got.AppliedPower != 0 {
		t.Fatalf("new source=%+v %v", got, err)
	}
}

// Three concurrent actors prove that the database lock, rather than a time
// cutoff or process mutex, orders acceptance, seal, and exact replay.
func TestG21P0AcceptanceCommitWhileSealWaits(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	inserted := make(chan struct{})
	release := make(chan struct{})
	s.miningPowerFailureInjector = func(stage string) error {
		if stage == "after_activity" {
			close(inserted)
			<-release
		}
		return nil
	}
	type result struct {
		value miningpower.ValidationResult
		err   error
	}
	acceptDone := make(chan result, 1)
	go func() {
		v, e := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
		acceptDone <- result{v, e}
	}()
	select {
	case <-inserted:
	case <-time.After(5 * time.Second):
		t.Fatal("acceptance did not reach in-flight boundary")
	}
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		close(release)
		t.Fatal(err)
	}
	replayDone := make(chan result, 1)
	go func() {
		v, e := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion)
		replayDone <- result{v, e}
	}()
	sealDone := make(chan struct {
		seal miningpower.SettlementInputSeal
		err  error
	}, 1)
	go func() {
		v, e := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
		sealDone <- struct {
			seal miningpower.SettlementInputSeal
			err  error
		}{v, e}
	}()
	select {
	case got := <-sealDone:
		close(release)
		t.Fatalf("seal passed uncommitted acceptance: %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	accepted := <-acceptDone
	if accepted.err != nil || accepted.value.Status != miningpower.StatusValid {
		t.Fatalf("accept=%+v", accepted)
	}
	sealed := <-sealDone
	if sealed.err != nil || sealed.seal.ActivityCount != 1 || sealed.seal.TotalValidMiningPower != 100 {
		t.Fatalf("seal=%+v", sealed)
	}
	replayed := <-replayDone
	if replayed.err != nil || replayed.value.Status != miningpower.StatusDuplicate || replayed.value.AppliedPower != 0 {
		t.Fatalf("replay=%+v", replayed)
	}
}

func TestG21P0BeneficiaryMergesSessionsAndSeparatesCharacters(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, principal, intent)
	session := g20Session(t, s, intent)
	source := g20Source(t, s, intent)
	secondSession := session
	secondSession.ID = "g21-p0-second-session"
	g20Insert(t, s, "mining_power_sessions", secondSession)
	secondSource := source
	secondSource.ID = "g21-p0-second-event"
	secondSource.ActivityID = "mpa:" + secondSource.ID
	secondSource.ActivitySessionID = secondSession.ID
	secondIntent := g20RegisteredSource(t, s, secondSource)
	g20Accept(t, s, principal, secondIntent)
	g19Seed(t, s, "g21-p0-other-character")
	if _, err := s.pool.Exec(ctx, `UPDATE characters SET account_id=$2 WHERE id=$1`, "g21-p0-other-character", principal.AccountID); err != nil {
		t.Fatal(err)
	}
	otherSession := session
	otherSession.ID = "g21-p0-other-session"
	otherSession.PlayerID = "g21-p0-other-character"
	otherSession.AccountID = principal.AccountID
	g20Insert(t, s, "mining_power_sessions", otherSession)
	otherSource := source
	otherSource.ID = "g21-p0-other-event"
	otherSource.ActivityID = "mpa:" + otherSource.ID
	otherSource.PlayerID = otherSession.PlayerID
	otherSource.ActivitySessionID = otherSession.ID
	otherIntent := g20RegisteredSource(t, s, otherSource)
	g20Accept(t, s, miningpower.Principal{AccountID: otherSession.AccountID, PlayerID: otherSession.PlayerID}, otherIntent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil || seal.ActivityCount != 3 || seal.ParticipantCount != 2 || seal.TotalValidMiningPower != 300 {
		t.Fatalf("seal=%+v %v", seal, err)
	}
	weights := map[string]miningpower.SealedWeight{}
	for _, w := range seal.ParticipantWeights {
		weights[w.CharacterID] = w
	}
	if weights["g20-player"].Power != 200 || weights["g20-player"].ActivityCount != 2 || weights["g21-p0-other-character"].Power != 100 {
		t.Fatalf("beneficiary weights=%+v", weights)
	}
}
