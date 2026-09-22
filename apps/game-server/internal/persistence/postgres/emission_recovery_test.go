package postgres

import (
	"context"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"

	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/systemspend"
)

func reserveBlocks(t *testing.T, service *miningblock.Service, count int) []miningblock.Receipt {
	t.Helper()
	var receipts []miningblock.Receipt
	for i := 0; i < count; i++ {
		r, err := service.Create(context.Background(), "g18-recovery-block-"+string(rune('a'+i)))
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, r)
	}
	return receipts
}

func assertRecoveryPool(t *testing.T, store *Store, net, reserved, remaining, debt int64) emission.Snapshot {
	t.Helper()
	ctx := context.Background()
	snap, err := emission.NewService(store, emission.DevelopmentRuleVersion).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p := snap.Pool
	if p.TotalEmissionCapacity != net || p.TotalReserved != reserved || p.RemainingCapacity != remaining || p.RecoveryDebt != debt || p.TotalDistributed != 0 || net != reserved+remaining-debt {
		t.Fatalf("pool net/reserved/remaining/debt=%d/%d/%d/%d, expected=%d/%d/%d/%d", p.TotalEmissionCapacity, p.TotalReserved, p.RemainingCapacity, p.RecoveryDebt, net, reserved, remaining, debt)
	}
	er, err := store.ReconcileBlackIronEmission(ctx)
	if err != nil || !er.Balanced {
		t.Fatalf("emission recovery reconciliation=%+v err=%v", er, err)
	}
	br, err := store.ReconcileMiningBlocks(ctx)
	if err != nil || !br.Balanced {
		t.Fatalf("block recovery reconciliation=%+v err=%v", br, err)
	}
	return snap
}

func TestG18RecoveryRefundRemainingBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		fund, refund, net, remaining, debt int64
	}{
		{"less", 30, 10, 20, 10, 0},
		{"equal", 20, 10, 10, 0, 0},
		{"more", 20, 15, 5, 0, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, blocks, contributions, spend := g18Fund(t, tc.fund)
			reserveBlocks(t, blocks, 1)
			if _, err := contributions.RefundSystemSpend(context.Background(), g15Refund(spend, "g18-recovery-refund-"+tc.name, tc.refund)); err != nil {
				t.Fatal(err)
			}
			snap := assertRecoveryPool(t, store, tc.net, 10, tc.remaining, tc.debt)
			last := snap.RecoveryEntries[len(snap.RecoveryEntries)-1]
			if last.SourceType != emissionRefundSource || last.RemainingBefore != tc.fund-10 || last.RemainingAfter != tc.remaining || last.DebtAfter != tc.debt || last.NetEmissionDelta != -tc.refund {
				t.Fatalf("refund audit=%+v", last)
			}
		})
	}
}

func TestG18RecoveryFutureEmissionOffsetsDebtFirst(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		added, net, remaining, debt int64
	}{
		{"less", 5, 5, 0, 5},
		{"equal", 10, 10, 0, 0},
		{"more", 20, 20, 10, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, blocks, contributions, spend := g18Fund(t, 10)
			reserveBlocks(t, blocks, 1)
			if _, err := contributions.RefundSystemSpend(context.Background(), g15Refund(spend, "g18-future-refund", 10)); err != nil {
				t.Fatal(err)
			}
			if _, err := blocks.Create(context.Background(), "g18-must-block-on-debt"); !errors.Is(err, miningblock.ErrRecoveryDebt) {
				t.Fatalf("debt gate=%v", err)
			}
			future, err := systemspend.NewService(store, g15Registry(t)).Post(context.Background(), g15Intent("g18-future-"+tc.name, tc.added))
			if err != nil {
				t.Fatal(err)
			}
			first, err := emission.NewService(store, emission.DevelopmentRuleVersion).Apply(context.Background(), future.ID)
			if err != nil {
				t.Fatal(err)
			}
			snap := assertRecoveryPool(t, store, tc.net, 10, tc.remaining, tc.debt)
			last := snap.RecoveryEntries[len(snap.RecoveryEntries)-1]
			if last.SourceType != emissionSpendSource || last.DebtBefore != 10 || last.DebtAfter != tc.debt || last.RemainingAfter != tc.remaining {
				t.Fatalf("future audit=%+v", last)
			}
			replayed, err := emission.NewService(store, emission.DevelopmentRuleVersion).Apply(context.Background(), future.ID)
			if err != nil || !reflect.DeepEqual(first, replayed) {
				t.Fatalf("emission replay=%+v err=%v", replayed, err)
			}
			if tc.debt > 0 {
				if _, err = blocks.Create(context.Background(), "g18-debt-still-blocks"); !errors.Is(err, miningblock.ErrRecoveryDebt) {
					t.Fatalf("remaining debt gate=%v", err)
				}
			}
		})
	}
}

func TestG18RecoveryCancelRepaysDebtFirst(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		fund, blocks, refund           int64
		net, reserved, remaining, debt int64
	}{
		{"less", 30, 3, 30, 0, 20, 0, 20},
		{"equal", 10, 1, 10, 0, 0, 0, 0},
		{"more", 20, 1, 15, 5, 0, 5, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, service, contributions, spend := g18Fund(t, tc.fund)
			opened := reserveBlocks(t, service, int(tc.blocks))
			if _, err := contributions.RefundSystemSpend(context.Background(), g15Refund(spend, "g18-cancel-refund-"+tc.name, tc.refund)); err != nil {
				t.Fatal(err)
			}
			first, err := service.Cancel(context.Background(), opened[0].BlockID)
			if err != nil {
				t.Fatal(err)
			}
			snap := assertRecoveryPool(t, store, tc.net, tc.reserved, tc.remaining, tc.debt)
			last := snap.RecoveryEntries[len(snap.RecoveryEntries)-1]
			if last.SourceType != "G18_BLOCK_CANCEL" || last.ReservedDelta != -10 || last.RemainingAfter != tc.remaining || last.DebtAfter != tc.debt || first.PoolAfter != tc.remaining {
				t.Fatalf("cancel audit=%+v receipt=%+v", last, first)
			}
			replayed, err := service.Cancel(context.Background(), opened[0].BlockID)
			if err != nil || !reflect.DeepEqual(first, replayed) {
				t.Fatalf("cancel replay=%+v err=%v", replayed, err)
			}
		})
	}
}

func TestG18RecoveryRestartExactReplayAndImmutableAudit(t *testing.T) {
	store, blocks, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	opened := reserveBlocks(t, blocks, 1)[0]
	first, err := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-restart-debt", 10))
	if err != nil {
		t.Fatal(err)
	}
	before := assertRecoveryPool(t, store, 0, 10, 0, 10)
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	after := assertRecoveryPool(t, reopened, 0, 10, 0, 10)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("debt snapshot changed after restart")
	}
	replayed, err := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-restart-debt", 10))
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("refund replay=%+v err=%v", replayed, err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE black_iron_emission_recovery_entries SET debt_after=0 WHERE source_id=$1`, opened.BlockID); err == nil {
		t.Fatal("immutable recovery audit changed")
	}
	if _, err = store.pool.Exec(ctx, `UPDATE black_iron_emission_pools SET total_reserved=9,recovery_debt=9 WHERE pool_id='GLOBAL'`); err != nil {
		t.Fatal(err)
	}
	report, err := reopened.ReconcileBlackIronEmission(ctx)
	if err != nil || report.Balanced {
		t.Fatalf("forged aggregate passed=%+v err=%v", report, err)
	}
}

func TestG18RecoveryDebtAndRemainingNeverNegative(t *testing.T) {
	store, blocks, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	reserveBlocks(t, blocks, 1)
	if _, err := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-negative-guard", 10)); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`UPDATE black_iron_emission_pools SET recovery_debt=-1 WHERE pool_id='GLOBAL'`,
		`UPDATE black_iron_emission_pools SET remaining_capacity=-1 WHERE pool_id='GLOBAL'`,
	} {
		if _, err := store.pool.Exec(ctx, query); err == nil {
			t.Fatalf("negative aggregate accepted: %s", query)
		}
	}
	assertRecoveryPool(t, store, 0, 10, 0, 10)
}

func TestG18RecoveryConcurrentRefundAndFutureEmission(t *testing.T) {
	store, blocks, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	reserveBlocks(t, blocks, 1)
	future, err := systemspend.NewService(store, g15Registry(t)).Post(ctx, g15Intent("g18-concurrent-future", 7))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, e := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-concurrent-refund", 10))
		errs <- e
	}()
	go func() {
		defer wg.Done()
		<-start
		_, e := emission.NewService(store, emission.DevelopmentRuleVersion).Apply(ctx, future.ID)
		errs <- e
	}()
	close(start)
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatalf("concurrent refund/emission=%v", e)
		}
	}
	assertRecoveryPool(t, store, 7, 10, 0, 3)
}

func TestG18RecoveryConcurrentRefundCancelAndDuplicateRefund(t *testing.T) {
	store, blocks, contributions, spend := g18Fund(t, 20)
	ctx := context.Background()
	opened := reserveBlocks(t, blocks, 1)[0]
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, e := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-concurrent-cancel-refund", 15))
		errs <- e
	}()
	go func() { defer wg.Done(); <-start; _, e := blocks.Cancel(ctx, opened.BlockID); errs <- e }()
	close(start)
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatalf("concurrent refund/cancel=%v", e)
		}
	}
	before := assertRecoveryPool(t, store, 5, 0, 5, 0)
	const duplicates = 8
	results := make(chan error, duplicates)
	for i := 0; i < duplicates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := contributions.RefundSystemSpend(ctx, g15Refund(spend, "g18-concurrent-cancel-refund", 15))
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	for e := range results {
		if e != nil {
			t.Fatalf("duplicate refund=%v", e)
		}
	}
	after := assertRecoveryPool(t, store, 5, 0, 5, 0)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("duplicate refund changed recovery journal")
	}
}
