package miningreward

import (
	"math"
	"math/rand"
	"reflect"
	"testing"

	"fractallegend/game-server/internal/miningpower"
)

func TestAllocateTESTLargestRemainderUsesCharacterBytes(t *testing.T) {
	tests := []struct {
		name string
		r    int64
		in   []miningpower.SealedWeight
		want []Grant
	}{
		{"unequal", 10, []miningpower.SealedWeight{
			{CharacterID: "C", Power: 3}, {CharacterID: "A", Power: 1}, {CharacterID: "B", Power: 2},
		}, []Grant{
			{CharacterID: "A", Power: 1, Quotient: 1, Remainder: 4, Bonus: 1, Quantity: 2},
			{CharacterID: "B", Power: 2, Quotient: 3, Remainder: 2, Quantity: 3},
			{CharacterID: "C", Power: 3, Quotient: 5, Quantity: 5},
		}},
		{"equal", 2, []miningpower.SealedWeight{
			{CharacterID: "C", Power: 1}, {CharacterID: "B", Power: 1}, {CharacterID: "A", Power: 1},
		}, []Grant{
			{CharacterID: "A", Power: 1, Remainder: 2, Bonus: 1, Quantity: 1},
			{CharacterID: "B", Power: 1, Remainder: 2, Bonus: 1, Quantity: 1},
			{CharacterID: "C", Power: 1, Remainder: 2},
		}},
		{"numeric bytes", 1, []miningpower.SealedWeight{
			{CharacterID: "2", Power: 1}, {CharacterID: "10", Power: 1},
		}, []Grant{
			{CharacterID: "10", Power: 1, Remainder: 1, Bonus: 1, Quantity: 1},
			{CharacterID: "2", Power: 1, Remainder: 1},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AllocateTEST(tc.r, tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Grants, tc.want) || got.Total != tc.r {
				t.Fatalf("allocation=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func TestAllocateTESTExactRatiosZeroWeightsAndByteOrder(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reward  int64
		weights []miningpower.SealedWeight
		want    []int64
	}{
		{"exact-ratio", 12, []miningpower.SealedWeight{{CharacterID: "C", Power: 3}, {CharacterID: "B", Power: 2}, {CharacterID: "A", Power: 1}}, []int64{2, 4, 6}},
		{"zero-weight", 3, []miningpower.SealedWeight{{CharacterID: "A", Power: 0}, {CharacterID: "B", Power: 2}}, []int64{0, 3}},
		{"all-zero", 10, []miningpower.SealedWeight{{CharacterID: "A", Power: 0}, {CharacterID: "B", Power: 0}, {CharacterID: "C", Power: 0}}, []int64{0, 0, 0}},
		{"case-sensitive", 1, []miningpower.SealedWeight{{CharacterID: "a", Power: 1}, {CharacterID: "A", Power: 1}}, []int64{1, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := AllocateTEST(tc.reward, tc.weights)
			if err != nil || len(got.Grants) != len(tc.want) {
				t.Fatalf("allocation=%+v %v", got, err)
			}
			for i, want := range tc.want {
				if got.Grants[i].Quantity != want {
					t.Fatalf("grant %d=%d want=%d", i, got.Grants[i].Quantity, want)
				}
			}
		})
	}
	for _, reward := range []int64{0, -1} {
		if _, err := AllocateTEST(reward, []miningpower.SealedWeight{{CharacterID: "A", Power: 1}}); err == nil {
			t.Fatalf("invalid reservation %d accepted", reward)
		}
	}
}

func TestAllocateTESTAllSixInputPermutations(t *testing.T) {
	base := []miningpower.SealedWeight{{CharacterID: "A", Power: 1}, {CharacterID: "B", Power: 2}, {CharacterID: "C", Power: 3}}
	want, err := AllocateTEST(10, base)
	if err != nil {
		t.Fatal(err)
	}
	for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		got, err := AllocateTEST(10, []miningpower.SealedWeight{base[order[0]], base[order[1]], base[order[2]]})
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("permutation=%v changed allocation=%+v err=%v", order, got, err)
		}
	}
}

func TestAllocateTESTWideProductAndConservation(t *testing.T) {
	wide32, err := AllocateTEST(math.MaxInt32, []miningpower.SealedWeight{
		{CharacterID: "B", Power: math.MaxInt64 / 2}, {CharacterID: "A", Power: math.MaxInt64 / 2},
	})
	if err != nil || wide32.Grants[0].Quantity != 1073741824 || wide32.Grants[1].Quantity != 1073741823 {
		t.Fatalf("wide 32-bit reward product=%+v %v", wide32, err)
	}
	boundary, err := AllocateTEST(1, []miningpower.SealedWeight{
		{CharacterID: "B", Power: 1}, {CharacterID: "A", Power: math.MaxInt64 - 1},
	})
	if err != nil || boundary.Power != math.MaxInt64 || boundary.Grants[0].Quantity != 1 || boundary.Grants[1].Quantity != 0 {
		t.Fatalf("total power boundary=%+v %v", boundary, err)
	}
	wide, err := AllocateTEST(math.MaxInt64, []miningpower.SealedWeight{
		{CharacterID: "B", Power: 1}, {CharacterID: "A", Power: math.MaxInt64 - 1},
	})
	if err != nil || wide.Power != math.MaxInt64 || wide.Total != math.MaxInt64 ||
		wide.Grants[0].Quantity != math.MaxInt64-1 || wide.Grants[1].Quantity != 1 {
		t.Fatalf("wide integer product=%+v %v", wide, err)
	}
	rng := rand.New(rand.NewSource(21))
	for n := 1; n <= 500; n += 7 {
		weights := make([]miningpower.SealedWeight, n)
		for i := range weights {
			weights[i] = miningpower.SealedWeight{CharacterID: string(rune(0x1000 + i)), Power: int64(rng.Intn(1000))}
		}
		for _, reward := range []int64{1, int64(n), 500, math.MaxInt64} {
			got, err := AllocateTEST(reward, weights)
			if err != nil {
				t.Fatalf("n=%d reward=%d err=%v", n, reward, err)
			}
			var sum int64
			for i, g := range got.Grants {
				if g.Quantity < 0 || g.Quantity > reward || (i > 0 && got.Grants[i-1].CharacterID >= g.CharacterID) {
					t.Fatalf("n=%d reward=%d grant=%+v", n, reward, g)
				}
				if sum > reward-g.Quantity {
					t.Fatalf("n=%d reward=%d allocation overflow", n, reward)
				}
				sum += g.Quantity
			}
			if sum != reward || got.Total != reward {
				t.Fatalf("n=%d reward=%d sum=%d", n, reward, sum)
			}
			permuted := append([]miningpower.SealedWeight(nil), weights...)
			rng.Shuffle(len(permuted), func(i, j int) { permuted[i], permuted[j] = permuted[j], permuted[i] })
			again, err := AllocateTEST(reward, permuted)
			if err != nil || !reflect.DeepEqual(got, again) {
				t.Fatalf("n=%d reward=%d input permutation changed result", n, reward)
			}
		}
	}
	for sample := 0; sample < 1000; sample++ {
		n := 1 + rng.Intn(500)
		reward := int64(1 + rng.Intn(math.MaxInt32))
		weights := make([]miningpower.SealedWeight, n)
		for i := range weights {
			weights[i] = miningpower.SealedWeight{CharacterID: string(rune(0x2000 + i)), Power: int64(1 + rng.Intn(1000000))}
		}
		got, err := AllocateTEST(reward, weights)
		if err != nil {
			t.Fatalf("sample=%d allocation error=%v", sample, err)
		}
		var sum, quotients, bonuses int64
		for _, grant := range got.Grants {
			if grant.Bonus < 0 || grant.Bonus > 1 || grant.Quantity != grant.Quotient+grant.Bonus {
				t.Fatalf("sample=%d invalid grant=%+v", sample, grant)
			}
			sum += grant.Quantity
			quotients += grant.Quotient
			bonuses += grant.Bonus
		}
		if sum != reward || bonuses != reward-quotients {
			t.Fatalf("sample=%d reward=%d sum=%d quotient=%d bonuses=%d", sample, reward, sum, quotients, bonuses)
		}
	}
}

func TestAllocateTESTRejectsInvalidAndZeroPower(t *testing.T) {
	if got, err := AllocateTEST(10, nil); err != nil || got.Total != 0 || len(got.Grants) != 0 {
		t.Fatalf("zero power=%+v %v", got, err)
	}
	for _, in := range [][]miningpower.SealedWeight{
		{{CharacterID: "A", Power: -1}},
		{{CharacterID: "A", Power: math.MaxInt64}, {CharacterID: "B", Power: 1}},
		{{CharacterID: "A", Power: 1}, {CharacterID: "A", Power: 1}},
	} {
		if _, err := AllocateTEST(1, in); err == nil {
			t.Fatalf("accepted invalid weights %+v", in)
		}
	}
	weights := make([]miningpower.SealedWeight, 501)
	for i := range weights {
		weights[i] = miningpower.SealedWeight{CharacterID: string(rune(0x1000 + i)), Power: 1}
	}
	if _, err := AllocateTEST(501, weights); err == nil {
		t.Fatal("accepted 501 beneficiaries")
	}
}
