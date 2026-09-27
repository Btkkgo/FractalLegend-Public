package postgres

import (
	"context"
	"encoding/json"
	"fractallegend/game-server/internal/miningpower"
	"reflect"
	"testing"
	"time"
)

func g20CanonicalFields(t *testing.T, value any) {
	t.Helper()
	var visit func(reflect.Value, string)
	visit = func(v reflect.Value, field string) {
		if v.Type() == reflect.TypeOf(time.Time{}) {
			stamp := v.Interface().(time.Time)
			if stamp.Location() != time.UTC || stamp != stamp.Round(0) {
				t.Fatalf("%s not canonical: %#v", field, stamp)
			}
			return
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i), field+"."+v.Type().Field(i).Name)
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i), field)
			}
		}
	}
	visit(reflect.ValueOf(value), "domain")
}

func g20CanonicalDecode[T any](t *testing.T, s *Store, input T) {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	got, err := miningPowerRead[T](ctx, s.pool, `SELECT $1::jsonb`, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	g20CanonicalFields(t, got)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	rows, err := miningPowerReadRows[T](ctx, tx, `SELECT $1::jsonb`, string(raw))
	if err != nil || len(rows) != 1 {
		t.Fatalf("bulk decode: %v %v", rows, err)
	}
	g20CanonicalFields(t, rows[0])
	if !reflect.DeepEqual(got, rows[0]) {
		t.Fatal("single/bulk boundary representation differs")
	}
	// Canonicalizing representation must preserve the original persisted instant
	// and precision in both decoder paths, including historic offset JSON.
	var compare func(reflect.Value, reflect.Value)
	compare = func(before, after reflect.Value) {
		if before.Type() == reflect.TypeOf(time.Time{}) {
			b, a := before.Interface().(time.Time), after.Interface().(time.Time)
			if !b.Equal(a) || b.Nanosecond() != a.Nanosecond() {
				t.Fatal("decode changed time instant or precision")
			}
			return
		}
		if before.Kind() == reflect.Struct {
			for i := 0; i < before.NumField(); i++ {
				compare(before.Field(i), after.Field(i))
			}
		} else if !reflect.DeepEqual(before.Interface(), after.Interface()) {
			t.Fatal("decode changed non-time field")
		}
	}
	compare(reflect.ValueOf(input), reflect.ValueOf(got))
}

func TestG20CanonicalPersistenceBoundaries(t *testing.T) {
	s := g20FreshStore(t)
	stamp := time.Date(2026, 9, 27, 12, 34, 56, 123456789, time.FixedZone("historic-offset", 9*3600))
	t.Run("session", func(t *testing.T) {
		g20CanonicalDecode(t, s, miningpower.ActivitySession{ID: "session", OpenedAt: stamp, ExpiresAt: stamp, CreatedAt: stamp, UpdatedAt: stamp})
	})
	t.Run("source", func(t *testing.T) {
		g20CanonicalDecode(t, s, miningpower.SourceEvent{ID: "source", ObservedAt: stamp, ExpiresAt: stamp})
	})
	t.Run("block", func(t *testing.T) {
		g20CanonicalDecode(t, s, miningpower.BlockContext{ID: "block", StartedAt: stamp, ScheduledEndAt: stamp, ValidatedAt: stamp})
	})
	snapshot := miningpower.MiningBlockValidationSnapshot{BlockID: "block", StartedAt: stamp, ScheduledEndAt: stamp, ValidatedAt: stamp}
	t.Run("validation", func(t *testing.T) { g20CanonicalDecode(t, s, snapshot) })
	t.Run("activity", func(t *testing.T) {
		g20CanonicalDecode(t, s, miningpower.ValidatedMiningActivity{ActivityID: "activity", AcceptedAt: stamp, ObservedAt: stamp, ExpiresAt: stamp, ValidationSnapshot: snapshot})
	})
	t.Run("participant", func(t *testing.T) {
		g20CanonicalDecode(t, s, miningpower.MiningParticipant{PlayerID: "player", CreatedAt: stamp, UpdatedAt: stamp})
	})
}

func TestG20CanonicalReceiptAndAggregate(t *testing.T) {
	s, p, i := g20Fixture(t)
	a := g20Accept(t, s, p, i)
	g20CanonicalFields(t, a)
	r, err := s.ValidateMiningActivity(context.Background(), p, i, miningpower.DevelopmentRuleVersion)
	if err != nil || r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0 || r.Original == nil || !reflect.DeepEqual(a, *r.Original) {
		t.Fatalf("strict persisted replay differs: %v %v", r, err)
	}
	snap, err := s.SnapshotMiningPower(context.Background(), i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if err != nil {
		t.Fatal(err)
	}
	g20CanonicalFields(t, snap)
	if len(snap.Activities) != 1 || !reflect.DeepEqual(a, snap.Activities[0]) {
		t.Fatal("snapshot fact differs from original receipt")
	}
	rebuilt, err := miningpower.RebuildParticipants(snap)
	if err != nil || rebuilt.Status != "PASS" {
		t.Fatalf("rebuild %v %v", rebuilt, err)
	}
	g20CanonicalFields(t, rebuilt)
}
