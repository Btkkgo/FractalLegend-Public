package miningpower

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func g20Snapshot() Snapshot {
	start := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	session := ActivitySession{ID: "session-1", PlayerID: "player-1", AccountID: "account-1", BlockID: "block-1", BlockInstanceID: "instance-1", ToolReference: "TEST_BASIC", MapReference: "TEST_MAP", RuleVersion: DevelopmentRuleVersion, State: "ACTIVE", OpenedAt: start, ExpiresAt: start.Add(time.Minute), CreatedAt: start, UpdatedAt: start}
	source := SourceEvent{ID: "event-1", ActivityID: "mpa:event-1", PlayerID: "player-1", BlockID: "block-1", BlockInstanceID: "instance-1", ActivitySessionID: "session-1", ToolReference: "TEST_BASIC", MapReference: "TEST_MAP", RuleVersion: DevelopmentRuleVersion, ObservedAt: start.Add(time.Second), ExpiresAt: start.Add(55 * time.Second), ActivityWeightScaled: 1000000, ServerEligibility: StatusValid, EvidenceKind: SyntheticEvidence}
	a := ValidatedMiningActivity{ActivityID: "mpa:event-1", SourceEventID: "event-1", PlayerID: "player-1", BlockID: "block-1", BlockInstanceID: "instance-1", ActivitySessionID: "session-1", ToolReference: "TEST_BASIC", MapReference: "TEST_MAP", RuleVersion: DevelopmentRuleVersion, Inputs: ValidatedInputs{100, 1000000, 1000000, 1000000}, ValidatedPower: 100, AcceptedAt: start.Add(5 * time.Second), ObservedAt: source.ObservedAt, ExpiresAt: source.ExpiresAt, Decision: DecisionAccepted, Status: StatusValid}
	a.ValidationSnapshot = MiningBlockValidationSnapshot{BlockInstanceID: "instance-1", BlockID: "block-1", BlockHeight: 1, CreateCommandID: "create-1", Status: "OPEN", G18RuleVersion: "DEV_G18_BLOCK_V1", G20RuleVersion: DevelopmentRuleVersion, SourceEvidenceVersion: "G18_SCHEMA_0012", StartedAt: start, ScheduledEndAt: start.Add(time.Minute), ValidatedAt: a.AcceptedAt}
	p := MiningParticipant{PlayerID: "player-1", BlockID: "block-1", BlockInstanceID: "instance-1", ActivitySessionID: "session-1", RuleVersion: DevelopmentRuleVersion, ToolReference: "TEST_BASIC", MapReference: "TEST_MAP", ValidatedPower: 100, ActivityCount: 1, CreatedAt: a.AcceptedAt, UpdatedAt: a.AcceptedAt}
	return Snapshot{Rules: []RuleManifest{DevelopmentManifest()}, Tools: []ToolProfile{{"TEST_BASIC", DevelopmentRuleVersion, SyntheticKind, 100, 1000000}}, Maps: []MapProfile{{"TEST_MAP", DevelopmentRuleVersion, SyntheticKind, 1000000}}, Sessions: []ActivitySession{session}, Sources: []SourceEvent{source}, Blocks: []BlockContext{{ID: "block-1", Status: "OPEN", StartedAt: start, ScheduledEndAt: start.Add(time.Minute), BlockInstanceID: "instance-1"}}, Activities: []ValidatedMiningActivity{a}, Participants: []MiningParticipant{p}}
}

func TestG20UnverifiableAggregate(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{"missing-source", func(s *Snapshot) { s.Sources = nil }},
		{"unknown-rule", func(s *Snapshot) { s.Rules = nil }},
		{"mixed-valid-unverifiable", func(s *Snapshot) {
			a := s.Activities[0]
			a.ActivityID = "mpa:event-2"
			a.SourceEventID = "event-2"
			a.AcceptedAt = a.AcceptedAt.Add(time.Second)
			s.Activities = append(s.Activities, a)
			s.Participants[0].ValidatedPower = 200
			s.Participants[0].ActivityCount = 2
			s.Participants[0].UpdatedAt = a.AcceptedAt
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := g20Snapshot()
			tc.mutate(&s)
			r, err := RebuildParticipants(s)
			if err != nil || len(r.Participants) != 0 {
				t.Fatalf("unverifiable subtotal exposed: %+v %v", r, err)
			}
			report, err := ReconcileSnapshot(s)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, f := range report.Findings {
				if f.Identity == "player-1/instance-1/session-1/"+DevelopmentRuleVersion && f.Field == "participantPower" {
					found = true
					if f.Expected != nil || f.Delta != nil || f.Actual == nil || f.Status != "UNVERIFIABLE" {
						t.Fatalf("false numeric expectation: %+v", f)
					}
				}
			}
			if !found {
				t.Fatalf("missing null-expected finding: %+v", report)
			}
		})
	}
	t.Run("overflow-order-and-valid-neighbor", func(t *testing.T) {
		s := g20Snapshot()
		s.Tools[0].BasePowerUnits = math.MaxInt64
		s.Activities[0].Inputs.BasePowerUnits = math.MaxInt64
		s.Activities[0].ValidatedPower = math.MaxInt64
		s.Participants[0].ValidatedPower = math.MaxInt64
		s.Tools = append(s.Tools, ToolProfile{"TEST_ONE", DevelopmentRuleVersion, SyntheticKind, 1, 1000000})
		oneSession := s.Sessions[0]
		oneSession.ID = "neighbor-session"
		oneSession.ToolReference = "TEST_ONE"
		s.Sessions = append(s.Sessions, oneSession)
		neighborSource := s.Sources[0]
		neighborSource.ID = "neighbor-event"
		neighborSource.ActivityID = "mpa:neighbor-event"
		neighborSource.ActivitySessionID = oneSession.ID
		neighborSource.ToolReference = "TEST_ONE"
		s.Sources = append(s.Sources, neighborSource)
		neighbor := s.Activities[0]
		neighbor.ActivityID = neighborSource.ActivityID
		neighbor.SourceEventID = neighborSource.ID
		neighbor.ActivitySessionID = oneSession.ID
		neighbor.ToolReference = "TEST_ONE"
		neighbor.Inputs.BasePowerUnits = 1
		neighbor.ValidatedPower = 1
		s.Activities = append(s.Activities, neighbor)
		neighborParticipant := s.Participants[0]
		neighborParticipant.ActivitySessionID = oneSession.ID
		neighborParticipant.ToolReference = "TEST_ONE"
		neighborParticipant.ValidatedPower = 1
		s.Participants = append(s.Participants, neighborParticipant)
		// Same session's second fact uses a small activity weight and rounds to
		// a positive amount, making its true group sum exceed MaxInt64.
		second := s.Activities[0]
		second.ActivityID = "mpa:overflow-event"
		second.SourceEventID = "overflow-event"
		second.Inputs.ActivityWeightScaled = 1
		second.ValidatedPower = 9223372036854
		secondSource := s.Sources[0]
		secondSource.ID = second.SourceEventID
		secondSource.ActivityID = second.ActivityID
		secondSource.ActivityWeightScaled = 1
		s.Sources = append(s.Sources, secondSource)
		s.Activities = append(s.Activities, second)
		s.Participants[0].ActivityCount = 2
		before, err := ReconcileSnapshot(s)
		if err != nil {
			t.Fatal(err)
		}
		s.Activities[0], s.Activities[2] = s.Activities[2], s.Activities[0]
		after, err := ReconcileSnapshot(s)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatalf("overflow depends on iteration order: before=%+v after=%+v %v", before, after, err)
		}
		built, err := RebuildParticipants(s)
		if err != nil || len(built.Participants) != 1 || built.Participants[0].ActivitySessionID != oneSession.ID || built.Participants[0].ValidatedPower != 1 {
			t.Fatalf("unaffected participant lost: %+v %v", built, err)
		}
	})
}

func TestG20HistoricalReferenceSyntax(t *testing.T) {
	for _, ref := range []string{"", strings.Repeat("a", 129), "map/forged", "地图"} {
		for _, kind := range []string{"tool", "map"} {
			t.Run(kind+"/"+ref, func(t *testing.T) {
				s := g20Snapshot()
				if kind == "tool" {
					s.Tools[0].Reference = ref
					s.Sessions[0].ToolReference = ref
					s.Sources[0].ToolReference = ref
					s.Activities[0].ToolReference = ref
					s.Participants[0].ToolReference = ref
				} else {
					s.Maps[0].Reference = ref
					s.Sessions[0].MapReference = ref
					s.Sources[0].MapReference = ref
					s.Activities[0].MapReference = ref
					s.Participants[0].MapReference = ref
				}
				r, err := ReconcileSnapshot(s)
				if err != nil || r.Status != "UNVERIFIABLE" {
					t.Fatalf("invalid historical reference accepted: %+v %v", r, err)
				}
			})
		}
	}
}

func TestG20RebuildAndReconcile(t *testing.T) {
	base := g20Snapshot()
	before, _ := json.Marshal(base)
	rebuilt, err := RebuildParticipants(base)
	if err != nil || rebuilt.Status != "PASS" || !reflect.DeepEqual(rebuilt.Participants, base.Participants) {
		t.Fatalf("T81 rebuilt=%+v err=%v", rebuilt, err)
	}
	report, err := ReconcileSnapshot(base)
	if err != nil || report.Status != "PASS" || len(report.Findings) != 0 {
		t.Fatalf("T69 exact report=%+v %v", report, err)
	}
	after, _ := json.Marshal(base)
	if string(before) != string(after) {
		t.Fatal("T80 pure reconciliation mutated inputs")
	}
	for _, tc := range []struct {
		name, field string
		mutate      func(*Snapshot)
	}{
		{"T70/missing-source", "source", func(s *Snapshot) { s.Sources = nil }},
		{"T71/duplicate-activity", "activityIdentity", func(s *Snapshot) { s.Activities = append(s.Activities, s.Activities[0]) }},
		{"T71/duplicate-source", "sourceIdentity", func(s *Snapshot) { s.Sources = append(s.Sources, s.Sources[0]) }},
		{"T72/unknown-rule", "rule", func(s *Snapshot) { s.Rules[0].Version = "unknown"; s.Activities[0].RuleVersion = "unknown" }},
		{"T73/invalid-tool", "tool", func(s *Snapshot) { s.Tools = nil }},
		{"T73/invalid-map", "map", func(s *Snapshot) { s.Maps = nil }},
		{"T74/power-mismatch", "activityPower", func(s *Snapshot) { s.Activities[0].ValidatedPower = 125 }},
		{"T75/missing-participant", "participant", func(s *Snapshot) { s.Participants = nil }},
		{"T76/extra-participant", "participant", func(s *Snapshot) {
			p := s.Participants[0]
			p.ActivitySessionID = "extra"
			s.Participants = append(s.Participants, p)
		}},
		{"T77/wrong-power", "participantPower", func(s *Snapshot) { s.Participants[0].ValidatedPower = 125 }},
		{"T77/wrong-count", "activityCount", func(s *Snapshot) { s.Participants[0].ActivityCount = 2 }},
		{"T78/wrong-time", "updatedAt", func(s *Snapshot) { s.Participants[0].UpdatedAt = s.Participants[0].UpdatedAt.Add(time.Second) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := g20Snapshot()
			tc.mutate(&s)
			before, _ := json.Marshal(s)
			r, err := ReconcileSnapshot(s)
			if err != nil || r.Status == "PASS" {
				t.Fatalf("corruption not reported: %+v %v", r, err)
			}
			found := false
			for _, f := range r.Findings {
				if f.Field == tc.field {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing finding %s: %+v", tc.field, r)
			}
			after, _ := json.Marshal(s)
			if string(before) != string(after) {
				t.Fatal("corrupt history silently repaired")
			}
		})
	}
	t.Run("T79/large-delta", func(t *testing.T) {
		s := g20Snapshot()
		s.Tools[0].BasePowerUnits = math.MaxInt64
		s.Activities[0].Inputs.BasePowerUnits = math.MaxInt64
		s.Activities[0].ValidatedPower = math.MaxInt64
		s.Participants[0].ValidatedPower = math.MinInt64
		r, err := ReconcileSnapshot(s)
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range r.Findings {
			if f.Field == "participantPower" && f.Delta != nil && *f.Delta == "-18446744073709551615" {
				return
			}
		}
		t.Fatalf("delta overflow or missing: %+v", r)
	})
	t.Run("T70/unconsumed-source", func(t *testing.T) {
		s := g20Snapshot()
		s.Activities = nil
		s.Participants = nil
		r, err := ReconcileSnapshot(s)
		if err != nil || r.Status != "PASS" {
			t.Fatalf("unconsumed source reported missing: %+v %v", r, err)
		}
	})
	t.Run("T63/historical-close", func(t *testing.T) {
		s := g20Snapshot()
		s.Sessions[0].State = "CLOSED"
		s.Blocks[0].Status = "FINALIZED"
		r, err := ReconcileSnapshot(s)
		if err != nil || r.Status != "PASS" {
			t.Fatalf("historical activity retroactively invalidated: %+v %v", r, err)
		}
	})
	t.Run("T82/empty", func(t *testing.T) {
		r, err := RebuildParticipants(Snapshot{})
		if err != nil || r.Status != "PASS" || len(r.Participants) != 0 {
			t.Fatalf("empty rebuild %+v %v", r, err)
		}
	})
}
