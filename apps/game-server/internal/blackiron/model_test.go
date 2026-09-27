package blackiron

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

func syntheticAlias() Alias {
	return Alias{DefinitionID: "synthetic-bun-material", LegacyID: 19001, LegacyName: "Bun", Evidence: "G19_SYNTHETIC_FIXTURE"}
}
func syntheticAsset() Asset {
	return Asset{InstanceID: "ore-1", DefinitionID: "synthetic-bun-material", LegacyID: 19001, Name: "Bun", ItemType: "MATERIAL", Quantity: 7, SlotIndex: 2, Location: "INVENTORY"}
}
func TestG19ExplicitIdentityPreservesAssetAndCanonicalReplay(t *testing.T) {
	a := syntheticAsset()
	got, err := Normalize(a, syntheticAlias())
	if err != nil {
		t.Fatal(err)
	}
	want := a
	want.Name = "黑铁矿石"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
	replay, err := Normalize(got, syntheticAlias())
	if err != nil || replay != got {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
}
func TestG19IdentityAndMaterialValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Asset, *Alias)
	}{
		{"unrelated-name", func(a *Asset, m *Alias) { a.DefinitionID = "title-bun-fan" }},
		{"legacy-id", func(a *Asset, m *Alias) { a.LegacyID = 49 }},
		{"currency", func(a *Asset, m *Alias) { a.ItemType = "CURRENCY" }},
		{"unknown-name", func(a *Asset, m *Alias) { a.Name = "馒头点" }},
		{"equipment", func(a *Asset, m *Alias) { a.Location = "EQUIPMENT" }},
		{"negative", func(a *Asset, m *Alias) { a.Quantity = -1 }},
		{"zero", func(a *Asset, m *Alias) { a.Quantity = 0 }},
		{"quantity-overflow", func(a *Asset, m *Alias) { a.Quantity = int64(math.MaxInt32) + 1 }},
		{"no-evidence", func(a *Asset, m *Alias) { m.Evidence = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, m := syntheticAsset(), syntheticAlias()
			tc.mutate(&a, &m)
			if _, err := Normalize(a, m); !errors.Is(err, ErrInvalid) {
				t.Fatalf("accepted: %v", err)
			}
		})
	}
}
func TestG19CheckedQuantityConservationBoundaries(t *testing.T) {
	for _, tc := range []struct {
		a, b, want int64
		invalid    bool
	}{{0, 0, 0, false}, {0, 1, 1, false}, {5, 7, 12, false}, {math.MaxInt64 - 1, 1, math.MaxInt64, false}, {math.MaxInt64, 1, 0, true}, {0, -1, 0, true}, {-1, 1, 0, true}} {
		got, err := AddQuantity(tc.a, tc.b)
		if (err != nil) != tc.invalid || (!tc.invalid && got != tc.want) {
			t.Fatalf("%+v got=%d err=%v", tc, got, err)
		}
	}
}
