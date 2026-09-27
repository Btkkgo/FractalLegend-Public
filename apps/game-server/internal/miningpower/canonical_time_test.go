package miningpower

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestG20CanonicalTime(t *testing.T) {
	instant := time.Date(2026, 9, 27, 12, 34, 56, 123456789, time.UTC)
	monotonic := time.Now()
	if monotonic == monotonic.Round(0) {
		t.Fatal("fixture must carry a monotonic clock")
	}
	for _, tc := range []struct {
		name  string
		input time.Time
	}{
		{"local", instant.In(time.Local)},
		{"utc", instant},
		{"fixed-zone", instant.In(time.FixedZone("offset", 9*3600))},
		{"monotonic", monotonic},
		{"zero", time.Time{}},
		{"zero-local", time.Time{}.In(time.Local)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := CanonicalTime(tc.input)
			if got.Location() != time.UTC || got != got.Round(0) || !tc.input.Equal(got) || tc.input.Nanosecond() != got.Nanosecond() {
				t.Fatalf("representation/instant/precision changed: before=%#v after=%#v", tc.input, got)
			}
			if tc.input.IsZero() && got != (time.Time{}) {
				t.Fatal("zero must remain canonical zero")
			}
			if CanonicalTime(got) != got {
				t.Fatal("canonicalization must be idempotent")
			}
			raw, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			var replay time.Time
			if err = json.Unmarshal(raw, &replay); err != nil {
				t.Fatal(err)
			}
			if got != replay {
				t.Fatalf("strict JSON equality failed: %s", raw)
			}
		})
	}
}

// Every authoritative time field is checked, including nested historical
// evidence. Omitting any field from a domain boundary breaks this test.
func TestG20CanonicalDomainTimes(t *testing.T) {
	stamp := time.Date(2026, 9, 27, 12, 34, 56, 123456789, time.FixedZone("offset", -7*3600))
	for _, tc := range []struct {
		name  string
		value any
	}{
		{"session", ActivitySession{OpenedAt: stamp, ExpiresAt: stamp, CreatedAt: stamp, UpdatedAt: stamp}.Canonical()},
		{"source", SourceEvent{ObservedAt: stamp, ExpiresAt: stamp}.Canonical()},
		{"block", BlockContext{StartedAt: stamp, ScheduledEndAt: stamp, ValidatedAt: stamp}.Canonical()},
		{"validation", MiningBlockValidationSnapshot{StartedAt: stamp, ScheduledEndAt: stamp, ValidatedAt: stamp}.Canonical()},
		{"activity", ValidatedMiningActivity{AcceptedAt: stamp, ObservedAt: stamp, ExpiresAt: stamp, ValidationSnapshot: MiningBlockValidationSnapshot{StartedAt: stamp, ScheduledEndAt: stamp, ValidatedAt: stamp}}.Canonical()},
		{"participant", MiningParticipant{CreatedAt: stamp, UpdatedAt: stamp}.Canonical()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var visit func(reflect.Value, string)
			visit = func(v reflect.Value, path string) {
				if v.Type() == reflect.TypeOf(time.Time{}) {
					got := v.Interface().(time.Time)
					if got.Location() != time.UTC || !got.Equal(stamp) || got.Nanosecond() != 123456789 || got != got.Round(0) {
						t.Fatalf("%s not canonical or instant changed: %#v", path, got)
					}
					return
				}
				if v.Kind() == reflect.Struct {
					for i := 0; i < v.NumField(); i++ {
						visit(v.Field(i), path+"."+v.Type().Field(i).Name)
					}
				}
			}
			visit(reflect.ValueOf(tc.value), tc.name)
		})
	}
}

func TestG20CanonicalReceiptTimezoneIndependence(t *testing.T) {
	// Identical nanosecond-precision input in separate TZ processes. No expected
	// value is normalized in the equality assertion.
	stamp := time.Date(2026, 9, 27, 12, 34, 56, 123456789, time.UTC).In(time.Local)
	a := ValidatedMiningActivity{ActivityID: "time-vector", ValidatedPower: 100, AcceptedAt: stamp, ObservedAt: stamp, ExpiresAt: stamp.Add(time.Minute), ValidationSnapshot: MiningBlockValidationSnapshot{StartedAt: stamp, ScheduledEndAt: stamp.Add(time.Minute), ValidatedAt: stamp}}.Canonical()
	raw, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var replay ValidatedMiningActivity
	if err = json.Unmarshal(raw, &replay); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, replay) {
		t.Fatalf("strict receipt equality failed: %s", raw)
	}
	if a.AcceptedAt.Format(time.RFC3339Nano) != "2026-09-27T12:34:56.123456789Z" {
		t.Fatal("instant or nanoseconds changed")
	}
	t.Logf("CANONICAL_RECEIPT_SHA256=%s", fmt.Sprintf("%x", sha256.Sum256(raw)))
}

func TestG20CanonicalRebuildKeepsInput(t *testing.T) {
	snapshot := g20Snapshot()
	before := snapshot.Activities[0].AcceptedAt.In(time.FixedZone("offset", 9*3600))
	snapshot.Activities[0].AcceptedAt = before
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := RebuildParticipants(snapshot)
	if err != nil || rebuilt.Status != "PASS" || len(rebuilt.Participants) != 1 {
		t.Fatalf("rebuild: %v %v", rebuilt, err)
	}
	p := rebuilt.Participants[0]
	if p.CreatedAt.Location() != time.UTC || p.UpdatedAt.Location() != time.UTC || !p.CreatedAt.Equal(before) || !p.UpdatedAt.Equal(before) {
		t.Fatal("rebuild must project canonical timestamps preserving their instant")
	}
	after, err := json.Marshal(snapshot)
	if err != nil || string(raw) != string(after) || snapshot.Activities[0].AcceptedAt != before {
		t.Fatal("rebuild mutated immutable input")
	}
}
