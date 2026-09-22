package postgres

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/systemspend"
)

func TestG17EligibleSpendExactReplayRestartSnapshotAndNoneligible(t *testing.T) {
	store, _, fb := g13Fixture(t)
	ctx := context.Background()
	spends := systemspend.NewService(store, g15Registry(t))
	spend, err := spends.Post(ctx, g15Intent("g17-first", 100))
	if err != nil {
		t.Fatal(err)
	}
	service := emission.NewService(store, emission.DevelopmentRuleVersion)
	first, err := service.Apply(ctx, spend.ID)
	if err != nil || first.EntryID == "" || first.SourceID != spend.ID || first.EligibleSpend != 100 || first.EmissionAdded != 100 || first.PoolBefore != 0 || first.PoolAfter != 100 || first.RemainingCapacity != 100 || first.RuleVersion != emission.DevelopmentRuleVersion || first.CreatedAt.Location().String() != "UTC" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if got := postgresBalance(t, fb.service, "pg-fb-a"); got != 9_900 {
		t.Fatalf("G17 changed FB balance: %d", got)
	}
	replay, err := service.Apply(ctx, spend.ID)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("same-process replay=%+v err=%v", replay, err)
	}
	if _, err := emission.NewService(store, "UNKNOWN").Apply(ctx, spend.ID); !errors.Is(err, emission.ErrUnknownRule) {
		t.Fatalf("unknown rule replay=%v", err)
	}
	if _, err := emission.NewService(store, "").Apply(ctx, spend.ID); !errors.Is(err, emission.ErrUnknownRule) {
		t.Fatalf("missing active rule replay=%v", err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := emission.NewService(reopened, emission.DevelopmentRuleVersion)
	third, err := restarted.Apply(ctx, spend.ID)
	if err != nil || !reflect.DeepEqual(first, third) {
		t.Fatalf("restart replay=%+v err=%v", third, err)
	}
	noneligible := g15Intent("g17-noneligible", 5)
	noneligible.ProducerType = systemspend.ProducerInternalTestNonEligible
	other, err := spends.Post(ctx, noneligible)
	if err != nil {
		t.Fatal(err)
	}
	zero, err := service.Apply(ctx, other.ID)
	if err != nil || zero.EmissionAdded != 0 || zero.EntryID != "" {
		t.Fatalf("noneligible=%+v err=%v", zero, err)
	}
	for _, fake := range []string{"deposit", "player-trade", "transfer", "tip", "gift", "guild-salary", "withdrawal", "recycle", "monster-drop", "boss-reward", "dungeon-reward", "siege-reward"} {
		if _, err := service.Apply(ctx, fake); !errors.Is(err, emission.ErrNotFound) {
			t.Fatalf("fake source %q: %v", fake, err)
		}
	}
	snapshot, err := restarted.Snapshot(ctx)
	if err != nil || snapshot.Pool.TotalEligibleSpendObserved != 100 || snapshot.Pool.TotalEmissionCapacity != 100 || snapshot.Pool.RemainingCapacity != 100 || snapshot.Pool.TotalReserved != 0 || snapshot.Pool.TotalDistributed != 0 || len(snapshot.Entries) != 1 || len(snapshot.Receipts) != 1 || !reflect.DeepEqual(snapshot.Receipts[0], first) {
		t.Fatalf("snapshot=%+v err=%v", snapshot, err)
	}
	again, err := service.Snapshot(ctx)
	if err != nil || !reflect.DeepEqual(snapshot, again) {
		t.Fatalf("snapshot reload mismatch=%+v err=%v", again, err)
	}
	report, err := restarted.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 1 {
		t.Fatalf("reconciliation=%+v err=%v", report, err)
	}
}

func TestG17RefundAndReversalCreateImmutableNegativeCompensations(t *testing.T) {
	store, contributions, _ := g13Fixture(t)
	ctx := context.Background()
	spends := systemspend.NewService(store, g15Registry(t))
	service := emission.NewService(store, emission.DevelopmentRuleVersion)
	spend, err := spends.Post(ctx, g15Intent("g17-refund", 100))
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.Apply(ctx, spend.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contributions.RefundSystemSpend(ctx, g15Refund(spend, "g17-refund-40", 40)); err != nil {
		t.Fatal(err)
	}
	partial, err := service.Snapshot(ctx)
	if err != nil || partial.Pool.TotalRefundedSpend != 40 || partial.Pool.TotalEmissionCapacity != 60 || partial.Pool.RemainingCapacity != 60 || len(partial.Entries) != 2 || partial.Entries[1].EligibleSpendAmount != -40 || partial.Entries[1].EmissionAmount != -40 {
		t.Fatalf("partial=%+v err=%v", partial, err)
	}
	reversal := g15Refund(spend, "g17-reversal-60", 60)
	reversal.Reversal = true
	if _, err = contributions.RefundSystemSpend(ctx, reversal); err != nil {
		t.Fatal(err)
	}
	full, err := service.Snapshot(ctx)
	if err != nil || full.Pool.TotalRefundedSpend != 100 || full.Pool.TotalEmissionCapacity != 0 || full.Pool.RemainingCapacity != 0 || len(full.Entries) != 3 || full.Entries[2].EmissionAmount != -60 {
		t.Fatalf("full=%+v err=%v", full, err)
	}
	replayed, err := service.Apply(ctx, spend.ID)
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("receipt changed by refund=%+v err=%v", replayed, err)
	}
	late, err := spends.Post(ctx, g15Intent("g17-refund-before-apply", 80))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contributions.RefundSystemSpend(ctx, g15Refund(late, "g17-late-30", 30)); err != nil {
		t.Fatal(err)
	}
	lateReceipt, err := service.Apply(ctx, late.ID)
	if err != nil || lateReceipt.EmissionAdded != 80 || lateReceipt.PoolBefore != 0 || lateReceipt.PoolAfter != 80 {
		t.Fatalf("late receipt=%+v err=%v", lateReceipt, err)
	}
	settled, err := service.Snapshot(ctx)
	if err != nil || settled.Pool.TotalEmissionCapacity != 50 || settled.Pool.TotalRefundedSpend != 130 || len(settled.Entries) != 5 {
		t.Fatalf("settled=%+v err=%v", settled, err)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 5 {
		t.Fatalf("reconciliation=%+v err=%v", report, err)
	}
}

func TestG17ConcurrentDuplicateAndDistinctSourcesConserveCapacity(t *testing.T) {
	store, _, _ := g13Fixture(t)
	ctx := context.Background()
	spends := systemspend.NewService(store, g15Registry(t))
	service := emission.NewService(store, emission.DevelopmentRuleVersion)
	first, err := spends.Post(ctx, g15Intent("g17-ten-duplicates", 20))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	receipts := make(chan emission.Receipt, 10)
	errs := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			receipt, err := service.Apply(ctx, first.ID)
			receipts <- receipt
			errs <- err
		}()
	}
	wg.Wait()
	close(receipts)
	close(errs)
	var canonical emission.Receipt
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for receipt := range receipts {
		if canonical.EntryID == "" {
			canonical = receipt
		} else if !reflect.DeepEqual(canonical, receipt) {
			t.Fatalf("concurrent replay differs: %+v vs %+v", canonical, receipt)
		}
	}
	ids := make([]string, 10)
	for i := range ids {
		intent := g15Intent("g17-distinct-"+string(rune('a'+i)), 3)
		spend, err := spends.Post(ctx, intent)
		if err != nil {
			t.Fatal(err)
		}
		ids[i] = spend.ID
	}
	errs = make(chan error, len(ids))
	for _, id := range ids {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Apply(ctx, id); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil || snapshot.Pool.TotalEligibleSpendObserved != 50 || snapshot.Pool.TotalEmissionCapacity != 50 || snapshot.Pool.RemainingCapacity != 50 || len(snapshot.Entries) != 11 {
		t.Fatalf("concurrent snapshot=%+v err=%v", snapshot, err)
	}
}

func TestG17RollbackAndAggregateMismatchAreVisible(t *testing.T) {
	for _, point := range []string{"after_entry", "after_pool", "after_receipt", "before_commit"} {
		t.Run(point, func(t *testing.T) {
			store, _, _ := g13Fixture(t)
			ctx := context.Background()
			spends := systemspend.NewService(store, g15Registry(t))
			spend, err := spends.Post(ctx, g15Intent("g17-failure-"+point, 12))
			if err != nil {
				t.Fatal(err)
			}
			service := emission.NewService(store, emission.DevelopmentRuleVersion)
			store.emissionFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected emission rollback")
				}
				return nil
			}
			if _, err := service.Apply(ctx, spend.ID); err == nil {
				t.Fatal("injected failure committed")
			}
			store.emissionFailureInjector = nil
			snapshot, err := service.Snapshot(ctx)
			if err != nil || snapshot.Pool.TotalEmissionCapacity != 0 || len(snapshot.Entries) != 0 || len(snapshot.Receipts) != 0 {
				t.Fatalf("half state=%+v err=%v", snapshot, err)
			}
			if _, err := service.Apply(ctx, spend.ID); err != nil {
				t.Fatalf("retry after rollback: %v", err)
			}
			if _, err := store.pool.Exec(ctx, `UPDATE black_iron_emission_pools SET total_emission_capacity=total_emission_capacity+1,remaining_capacity=remaining_capacity+1 WHERE pool_id='GLOBAL'`); err != nil {
				t.Fatal(err)
			}
			report, err := service.Reconcile(ctx)
			if err != nil || report.Balanced {
				t.Fatalf("aggregate corruption unnoticed: %+v err=%v", report, err)
			}
		})
	}
}

// A forged journal and aggregate that agree with each other must still fail
// reconciliation because the emission amount is not implied by the G15 source.
func TestG17ReconciliationRejectsWrongRuleAmount(t *testing.T) {
	store, _, _ := g13Fixture(t)
	ctx := context.Background()
	spend, err := systemspend.NewService(store, g15Registry(t)).Post(ctx, g15Intent("g17-corrupt-rule-amount", 12))
	if err != nil {
		t.Fatal(err)
	}
	service := emission.NewService(store, emission.DevelopmentRuleVersion)
	if _, err := service.Apply(ctx, spend.ID); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{
		`ALTER TABLE black_iron_emission_entries DISABLE TRIGGER black_iron_emission_entries_immutable`,
		`ALTER TABLE black_iron_emission_receipts DISABLE TRIGGER black_iron_emission_receipts_immutable`,
		`UPDATE black_iron_emission_entries SET emission_amount=13 WHERE source_id=$1`,
		`UPDATE black_iron_emission_receipts SET emission_added=13,pool_after=13,remaining_capacity=13 WHERE source_id=$1`,
		`UPDATE black_iron_emission_pools SET total_emission_capacity=13,remaining_capacity=13 WHERE pool_id='GLOBAL'`,
	} {
		var args []any
		if strings.Contains(query, "$1") {
			args = append(args, spend.ID)
		}
		if _, err := store.pool.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	report, err := service.Reconcile(ctx)
	if err != nil || report.Balanced {
		t.Fatalf("forged emission amount was accepted: %+v err=%v", report, err)
	}
}

func TestG17RefundEmissionFailureRollsBackFBAndContribution(t *testing.T) {
	store, contributions, fb := g13Fixture(t)
	ctx := context.Background()
	spend, err := systemspend.NewService(store, g15Registry(t)).Post(ctx, g15Intent("g17-refund-atomic", 100))
	if err != nil {
		t.Fatal(err)
	}
	service := emission.NewService(store, emission.DevelopmentRuleVersion)
	if _, err := service.Apply(ctx, spend.ID); err != nil {
		t.Fatal(err)
	}
	store.emissionFailureInjector = func(at string) error {
		if at == "after_entry" {
			return errors.New("injected refund emission failure")
		}
		return nil
	}
	request := g15Refund(spend, "g17-refund-atomic-40", 40)
	if _, err := contributions.RefundSystemSpend(ctx, request); err == nil {
		t.Fatal("refund committed despite emission failure")
	}
	store.emissionFailureInjector = nil
	account, err := contributions.Account(ctx, spend.PlayerID)
	if err != nil || account.Balance != 100 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("economic rollback account=%+v err=%v", account, err)
	}
	snapshot, err := service.Snapshot(ctx)
	if err != nil || snapshot.Pool.TotalEmissionCapacity != 100 || len(snapshot.Entries) != 1 {
		t.Fatalf("emission rollback=%+v err=%v", snapshot, err)
	}
	if _, err := contributions.RefundSystemSpend(ctx, request); err != nil {
		t.Fatalf("retry after rollback: %v", err)
	}
	snapshot, err = service.Snapshot(ctx)
	if err != nil || snapshot.Pool.TotalEmissionCapacity != 60 || len(snapshot.Entries) != 2 {
		t.Fatalf("committed refund=%+v err=%v", snapshot, err)
	}
}
