package miningpower

import (
	"reflect"
	"testing"
)

func TestG20ParticipantDisplayAudit(t *testing.T) {
	s := g20Snapshot()
	s.Participants[0].BlockID = "wrong-display"
	r, e := ReconcileSnapshot(s)
	if e != nil || r.Status == "PASS" {
		t.Fatalf("display mismatch concealed: %+v %v", r, e)
	}
	for _, f := range r.Findings {
		if f.Field == "blockID" && f.Expected != nil && *f.Expected == s.Activities[0].BlockID && f.Actual != nil && *f.Actual == "wrong-display" {
			return
		}
	}
	t.Fatalf("display mismatch missing exact evidence: %+v", r)
}

func TestG20AmbiguousHistory(t *testing.T) {
	for _, kind := range []string{"rule", "tool", "map", "session", "source", "block", "activity"} {
		t.Run(kind, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				s := g20Snapshot()
				switch kind {
				case "rule":
					v := s.Rules[0]
					v.Scale++
					s.Rules = append(s.Rules, v)
					if reverse {
						s.Rules[0], s.Rules[1] = s.Rules[1], s.Rules[0]
					}
				case "tool":
					v := s.Tools[0]
					v.BasePowerUnits++
					s.Tools = append(s.Tools, v)
					if reverse {
						s.Tools[0], s.Tools[1] = s.Tools[1], s.Tools[0]
					}
				case "map":
					v := s.Maps[0]
					v.ModifierScaled++
					s.Maps = append(s.Maps, v)
					if reverse {
						s.Maps[0], s.Maps[1] = s.Maps[1], s.Maps[0]
					}
				case "session":
					v := s.Sessions[0]
					v.ExpiresAt = v.ExpiresAt.Add(-1)
					s.Sessions = append(s.Sessions, v)
					if reverse {
						s.Sessions[0], s.Sessions[1] = s.Sessions[1], s.Sessions[0]
					}
				case "source":
					v := s.Sources[0]
					v.ExpiresAt = v.ExpiresAt.Add(-1)
					s.Sources = append(s.Sources, v)
					if reverse {
						s.Sources[0], s.Sources[1] = s.Sources[1], s.Sources[0]
					}
				case "block":
					v := s.Blocks[0]
					v.ScheduledEndAt = v.ScheduledEndAt.Add(-1)
					s.Blocks = append(s.Blocks, v)
					if reverse {
						s.Blocks[0], s.Blocks[1] = s.Blocks[1], s.Blocks[0]
					}
				case "activity":
					v := s.Activities[0]
					v.ValidatedPower++
					s.Activities = append(s.Activities, v)
					if reverse {
						s.Activities[0], s.Activities[1] = s.Activities[1], s.Activities[0]
					}
				}
				r, e := ReconcileSnapshot(s)
				if e != nil || r.Status != "UNVERIFIABLE" {
					t.Fatalf("%s reverse=%v report=%+v %v", kind, reverse, r, e)
				}
				found := false
				for _, f := range r.Findings {
					if f.Identity == aggregateKey(s.Participants[0]).label() && f.Field == "participantPower" && f.Status == "UNVERIFIABLE" && f.Expected == nil && f.Delta == nil {
						found = true
					}
				}
				if !found {
					t.Fatalf("ambiguous group leaked numeric expectation: %+v", r)
				}
				rebuilt, e := RebuildParticipants(s)
				if e != nil || len(rebuilt.Participants) != 0 {
					t.Fatalf("ambiguous facts projected: %+v %v", rebuilt, e)
				}
			}
		})
	}
	t.Run("unaffected-neighbor", func(t *testing.T) {
		s := g20Snapshot()
		other := s.Sessions[0]
		other.ID = "unaffected-session"
		source := s.Sources[0]
		source.ID = "unaffected-event"
		source.ActivityID = "mpa:" + source.ID
		source.ActivitySessionID = other.ID
		a := s.Activities[0]
		a.ActivityID = source.ActivityID
		a.SourceEventID = source.ID
		a.ActivitySessionID = other.ID
		p := s.Participants[0]
		p.ActivitySessionID = other.ID
		s.Sessions = append(s.Sessions, other)
		s.Sources = append(s.Sources, source)
		s.Activities = append(s.Activities, a)
		s.Participants = append(s.Participants, p)
		bad := s.Sources[0]
		bad.ExpiresAt = bad.ExpiresAt.Add(-1)
		s.Sources = append(s.Sources, bad)
		first, e := RebuildParticipants(s)
		if e != nil || len(first.Participants) != 1 || first.Participants[0].ActivitySessionID != other.ID {
			t.Fatalf("unaffected neighbor lost: %+v %v", first, e)
		}
		s.Sources[0], s.Sources[2] = s.Sources[2], s.Sources[0]
		last, e := RebuildParticipants(s)
		if e != nil || !reflect.DeepEqual(first, last) {
			t.Fatalf("reversal changed expectation: first=%+v last=%+v %v", first, last, e)
		}
	})
}

func TestG20AmbiguousActualParticipant(t *testing.T) {
	var baseline ReconciliationReport
	for _, reverse := range []bool{false, true} {
		s := g20Snapshot()
		p := s.Participants[0]
		p.ValidatedPower = 125
		s.Participants = append(s.Participants, p)
		if reverse {
			s.Participants[0], s.Participants[1] = s.Participants[1], s.Participants[0]
		}
		r, e := ReconcileSnapshot(s)
		if e != nil || r.Status != "UNVERIFIABLE" {
			t.Fatalf("duplicate actual=%+v %v", r, e)
		}
		found := false
		for _, f := range r.Findings {
			if f.Field == "participantPower" && f.Status == "UNVERIFIABLE" && f.Expected != nil && *f.Expected == "100" && f.Actual == nil && f.Delta == nil {
				found = true
			}
		}
		if !found {
			t.Fatalf("ambiguous actual selected arbitrary numeric value: %+v", r)
		}
		if reverse && !reflect.DeepEqual(baseline, r) {
			t.Fatalf("actual ordering changed report: first=%+v last=%+v", baseline, r)
		}
		baseline = r
	}
}
