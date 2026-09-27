package postgres

import (
	"context"
	"fmt"
	"fractallegend/game-server/internal/miningpower"
	"reflect"
	"testing"
	"time"
)

func TestG20OrderTimezoneAndAudit(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	session := g20Session(t, s, i)
	base := g20Source(t, s, i)
	intents := []miningpower.ActionIntent{}
	for k, offset := range []int{-7, 0, 8} {
		source := base
		source.ID = fmt.Sprintf("ordered-%d", k)
		source.ActivityID = "mpa:" + source.ID
		source.ObservedAt = session.OpenedAt.Add(time.Duration(k+1) * time.Microsecond).In(time.FixedZone("test-zone", offset*3600))
		source.ActivityWeightScaled = []int64{500000, 1000000, 1250000}[k]
		intents = append(intents, g20RegisteredSource(t, s, source))
	}
	for k := len(intents) - 1; k >= 0; k-- {
		a := g20Accept(t, s, p, intents[k])
		if a.ValidatedPower != []int64{50, 100, 125}[k] || a.BlockInstanceID != i.BlockInstanceID || a.ValidationSnapshot.BlockHeight < 1 || a.ValidationSnapshot.CreateCommandID == "" || a.ValidationSnapshot.G18RuleVersion == "" || a.ValidationSnapshot.G20RuleVersion != miningpower.DevelopmentRuleVersion || a.AcceptedAt.Nanosecond()%1000 != 0 || a.ValidationSnapshot.ValidatedAt.After(a.AcceptedAt) {
			t.Fatalf("audit/timezone context=%+v", a)
		}
	}
	snap, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil {
		t.Fatal(e)
	}
	original, e := miningpower.RebuildParticipants(snap)
	if e != nil || len(original.Participants) != 1 || original.Participants[0].ValidatedPower != 275 || original.Participants[0].ActivityCount != 3 {
		t.Fatalf("ordered aggregate=%+v %v", original, e)
	}
	for l, r := 0, len(snap.Activities)-1; l < r; l, r = l+1, r-1 {
		snap.Activities[l], snap.Activities[r] = snap.Activities[r], snap.Activities[l]
	}
	reordered, e := miningpower.RebuildParticipants(snap)
	if e != nil || !reflect.DeepEqual(original, reordered) {
		t.Fatal("min/max or rounded aggregate depends on activity iteration order")
	}
}

func TestG20AcceptedFractionalZeroTotal(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	rule := miningpower.DevelopmentRuleVersion
	g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: "TEST_FRACTION", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: 1, EfficiencyScaled: 500000})
	session := g20Session(t, s, i)
	session.ID = "fraction-session"
	session.ToolReference = "TEST_FRACTION"
	g20Insert(t, s, "mining_power_sessions", session)
	source := g20Source(t, s, i)
	source.ID = "fraction-event"
	source.ActivityID = "mpa:fraction-event"
	source.ActivitySessionID = session.ID
	source.ToolReference = session.ToolReference
	source.ActivityWeightScaled = 500000
	intent := g20RegisteredSource(t, s, source)
	a := g20Accept(t, s, p, intent)
	if a.ValidatedPower != 0 {
		t.Fatal("fractional power acquired minimum-one/carry")
	}
	total, e := s.TotalValidatedPower(ctx, i.BlockInstanceID, rule)
	if e != nil || total != 0 {
		t.Fatalf("accepted zero total=%d %v", total, e)
	}
	r, e := s.ValidateMiningActivity(ctx, p, intent, rule)
	if e != nil || r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0 {
		t.Fatalf("zero replay=%+v %v", r, e)
	}
	snap, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, rule)
	if e != nil || len(snap.Activities) != 1 || len(snap.Participants) != 1 || snap.Participants[0].ActivityCount != 1 || snap.Participants[0].ValidatedPower != 0 {
		t.Fatalf("zero audit=%+v %v", snap, e)
	}
}
