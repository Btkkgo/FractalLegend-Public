package miningreward

import (
	"fractallegend/game-server/internal/miningpower"
	"reflect"
	"testing"
)

func TestG21ClosureSybilAndLowPowerLimitations(t *testing.T) {
	before, e := AllocateTEST(2, []miningpower.SealedWeight{{CharacterID: "A", Power: 2}, {CharacterID: "B", Power: 1}})
	if e != nil || before.Grants[0].Quantity != 1 || before.Grants[1].Quantity != 1 {
		t.Fatal(before, e)
	}
	after, e := AllocateTEST(2, []miningpower.SealedWeight{{CharacterID: "A1", Power: 1}, {CharacterID: "A2", Power: 1}, {CharacterID: "B", Power: 1}})
	if e != nil {
		t.Fatal(e)
	}
	var quantities []int64
	for _, g := range after.Grants {
		quantities = append(quantities, g.Quantity)
	}
	if !reflect.DeepEqual(quantities, []int64{1, 1, 0}) {
		t.Fatal(quantities)
	}
	for block := 0; block < 100; block++ {
		r, e := AllocateTEST(1, []miningpower.SealedWeight{{CharacterID: "A", Power: 1}, {CharacterID: "B", Power: 100}})
		if e != nil || r.Grants[0].Quantity != 0 || r.Grants[1].Quantity != 1 {
			t.Fatal(block, r, e)
		}
	}
}
