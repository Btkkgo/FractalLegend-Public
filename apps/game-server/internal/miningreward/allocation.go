package miningreward

import (
	"errors"
	"math"
	"math/big"
	"sort"

	"fractallegend/game-server/internal/miningpower"
)

var ErrInvalidAllocation = errors.New("invalid mining reward allocation")

type Grant struct {
	PlayerID, AccountID, CharacterID string
	Power                            int64
	Quotient, Remainder, Bonus       int64
	Quantity                         int64
}

type Allocation struct {
	Total  int64
	Power  int64
	Grants []Grant
}

// AllocateTEST is pure. The caller must obtain R and weights from the locked
// reservation and persisted Seal; this function never supplies authority.
func AllocateTEST(reward int64, weights []miningpower.SealedWeight) (Allocation, error) {
	out := Allocation{Grants: make([]Grant, 0, len(weights))}
	if reward <= 0 || len(weights) > miningpower.TestParticipantCap {
		return Allocation{}, ErrInvalidAllocation
	}
	ordered := append([]miningpower.SealedWeight(nil), weights...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].CharacterID < ordered[j].CharacterID })
	var power int64
	for i, w := range ordered {
		if w.CharacterID == "" || w.Power < 0 ||
			(i > 0 && ordered[i-1].CharacterID == w.CharacterID) ||
			w.Power > math.MaxInt64-power {
			return Allocation{}, ErrInvalidAllocation
		}
		power += w.Power
		out.Grants = append(out.Grants, Grant{
			PlayerID: w.PlayerID, AccountID: w.AccountID, CharacterID: w.CharacterID, Power: w.Power,
		})
	}
	out.Power = power
	if power == 0 {
		return out, nil
	}
	bigReward := big.NewInt(reward)
	bigPower := big.NewInt(power)
	var used int64
	for i, w := range ordered {
		product := new(big.Int).Mul(bigReward, big.NewInt(w.Power))
		quotient, remainder := new(big.Int).QuoRem(product, bigPower, new(big.Int))
		q, r := quotient.Int64(), remainder.Int64()
		out.Grants[i].Quotient = q
		out.Grants[i].Remainder = r
		used += q // Sum of quotients cannot exceed positive R.
	}
	left := reward - used
	rank := make([]int, 0, len(ordered))
	for i, w := range ordered {
		if w.Power > 0 {
			rank = append(rank, i)
		}
	}
	sort.Slice(rank, func(i, j int) bool {
		a, b := out.Grants[rank[i]], out.Grants[rank[j]]
		if a.Remainder == b.Remainder {
			return a.CharacterID < b.CharacterID
		}
		return a.Remainder > b.Remainder
	})
	if left < 0 || left > int64(len(rank)) {
		return Allocation{}, ErrInvalidAllocation
	}
	for i := int64(0); i < left; i++ {
		out.Grants[rank[i]].Bonus = 1
	}
	for i := range out.Grants {
		g := &out.Grants[i]
		g.Quantity = g.Quotient + g.Bonus
	}
	out.Total = reward
	return out, nil
}
