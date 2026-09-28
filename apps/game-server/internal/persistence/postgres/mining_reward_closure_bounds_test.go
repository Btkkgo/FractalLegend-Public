package postgres

import (
	"context"
	"errors"
	"fractallegend/game-server/internal/emission"
	"math"
	"testing"
)

func TestG21ClosureInventoryQuantityBounds(t *testing.T) {
	for _, tc := range []struct {
		quantity int64
		valid    bool
	}{{0, true}, {10, true}, {math.MaxInt32, true}, {math.MaxInt32 + 1, false}, {-1, false}} {
		got, e := checkedMiningRewardQuantity(tc.quantity)
		if tc.valid {
			if e != nil || int64(got) != tc.quantity {
				t.Fatalf("quantity=%d got=%d err=%v", tc.quantity, got, e)
			}
		} else if !errors.Is(e, ErrMiningRewardInvariant) {
			t.Fatalf("invalid quantity=%d got=%d err=%v", tc.quantity, got, e)
		}
	}
}

func TestG21ClosurePoolOverflowAndDebtGate(t *testing.T) {
	if _, e := emission.AddCapacity(math.MaxInt64, 1); !errors.Is(e, emission.ErrOverflow) {
		t.Fatal("overflow not rejected", e)
	}
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx := context.Background()
	// Explicit corrupted-state stress only: authority reservation stays R10.
	// Reachable-state conservation is covered by real refund/capacity histories.
	before := g21ClosureSnapshot(t, s)
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	_, e = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_emission_capacity=9223372036854775807,total_distributed=9223372036854775807,total_reserved=10,remaining_capacity=0,recovery_debt=10`)
	if e == nil {
		t.Fatal("overflowing pool constraint accepted")
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if g21ClosureSnapshot(t, s) != before {
		t.Fatal("overflow rejection mutated rows")
	}
	r, e := s.SettleMiningRewardTEST(ctx, cmd)
	if e != nil || r.TotalOre != 10 {
		t.Fatal("valid settlement after rejection", e)
	}
}
