package postgres

import (
	"context"
	"errors"
	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/systemspend"
	"testing"
	"time"
)

func g21WaitTwoBlocked(t *testing.T, s *Store) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var n int
		if e := s.pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		if n >= 2 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("two concurrent DB waiters absent: %d", n)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestG21ClosureDifferentCommandsConcurrentSameInstance(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	blocker, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(context.Background())
	if _, e = blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); e != nil {
		t.Fatal(e)
	}
	type result struct {
		r MiningRewardSettlementReceipt
		e error
	}
	ch := make(chan result, 2)
	for _, id := range []string{"closure-concurrent-A", "closure-concurrent-B"} {
		c := cmd
		c.CommandID = id
		go func() { r, e := s.SettleMiningRewardTEST(ctx, c); ch <- result{r, e} }()
	}
	g21WaitTwoBlocked(t, s)
	if e = blocker.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	wins, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		got := <-ch
		if got.e == nil {
			if got.r.TotalOre != 10 {
				t.Fatal(got.r)
			}
			wins++
		} else if errors.Is(got.e, ErrMiningRewardConflict) {
			conflicts++
		} else {
			t.Fatal(got.e)
		}
	}
	if wins != 1 || conflicts != 1 {
		t.Fatalf("wins/conflicts %d/%d", wins, conflicts)
	}
	var commands, receipts, consumed, grants, lots, stacks int
	if e = s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_commands),(SELECT count(*) FROM mining_reward_settlement_receipts),(SELECT count(*) FROM mining_reward_reservation_consumptions),(SELECT count(*) FROM mining_reward_grants),(SELECT count(*) FROM mining_reward_issuance_lots),(SELECT count(*) FROM mining_reward_inventory_projections)`).Scan(&commands, &receipts, &consumed, &grants, &lots, &stacks); e != nil {
		t.Fatal(e)
	}
	if commands != 1 || receipts != 1 || consumed != 1 || grants != 1 || lots != 1 || stacks != 1 {
		t.Fatal("double economic artifact")
	}
	closureReconcile(t, s)
}

func TestG21ClosureCapacityRecoveryConcurrentSettlement(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// C20/S10/M10 -> refund15 -> C5/S10/M0/H5. No issued ore exists yet.
	fund, e := s.LoadSystemSpend(ctx, "g18-fund")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(fund, "closure-concurrent-capacity-refund", 15)); e != nil {
		t.Fatal(e)
	}
	closurePool(t, s, [5]int64{5, 10, 0, 0, 5})
	spend, e := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("closure-concurrent-capacity", 5))
	if e != nil {
		t.Fatal(e)
	}
	blocker, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer blocker.Rollback(context.Background())
	if _, e = blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); e != nil {
		t.Fatal(e)
	}
	capacity := make(chan error, 1)
	settled := make(chan error, 1)
	go func() {
		_, e := emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID)
		capacity <- e
	}()
	go func() { _, e := s.SettleMiningRewardTEST(ctx, cmd); settled <- e }()
	g21WaitTwoBlocked(t, s)
	if e = blocker.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if e = <-capacity; e != nil {
		t.Fatal(e)
	}
	e = <-settled
	if e != nil && !errors.Is(e, ErrMiningRewardDebt) {
		t.Fatal(e)
	}
	if errors.Is(e, ErrMiningRewardDebt) {
		closurePool(t, s, [5]int64{10, 10, 0, 0, 0})
	} else {
		closurePool(t, s, [5]int64{10, 0, 10, 0, 0})
	}
	r, e := s.SettleMiningRewardTEST(ctx, cmd)
	if e != nil || r.TotalOre != 10 {
		t.Fatal("post-race exact recovery", e)
	}
	closurePool(t, s, [5]int64{10, 0, 10, 0, 0})
	closureReconcile(t, s)
	before := g21ClosureSnapshot(t, s)
	if _, e = emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.SettleMiningRewardTEST(ctx, cmd); e != nil {
		t.Fatal(e)
	}
	if before != g21ClosureSnapshot(t, s) {
		t.Fatal("capacity/settlement replay delta")
	}
}
