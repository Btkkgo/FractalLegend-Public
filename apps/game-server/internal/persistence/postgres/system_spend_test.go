package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"testing"
	"testing/fstest"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/systemspend"
)

func g15Registry(t *testing.T) systemspend.Registry {
	t.Helper()
	registry, err := systemspend.NewRegistry([]systemspend.Producer{
		{Type: systemspend.ProducerInternalTestEligible, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Refundable: true, PartialRefundAllowed: true, Active: true, Description: "G15 internal eligible fixture"},
		{Type: systemspend.ProducerInternalTestNonEligible, Active: true, Description: "G15 internal non-eligible fixture"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func g15Intent(id string, amount int64) systemspend.Intent {
	return systemspend.Intent{OperationID: id, PlayerID: "pg-player-a", PlayerFBAccountID: "pg-fb-a", SystemFBAccountID: "pg-fb-system", ProducerType: systemspend.ProducerInternalTestEligible, ProducerReference: "producer-" + id, FBAmount: amount}
}

func g15Refund(spend systemspend.SystemSpend, id string, amount int64) contribution.RefundRequest {
	return contribution.RefundRequest{OriginalFBTransactionID: spend.FBTransactionID, PlayerID: spend.PlayerID, PlayerFBAccountID: spend.PlayerFBAccountID, ReferenceID: id, Amount: amount}
}

func TestG15PostgresAtomicSpendReplayConflictRestartAndRefund(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	intent := g15Intent("atomic-1", 100)
	first, err := service.Post(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || first.FBTransactionID == "" || first.ContributionEntryID == "" || first.RuleVersion != contribution.RuleVersionV1 || first.ContributionAmount != 100 || first.RefundStatus != systemspend.RefundNone || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("first=%+v", first)
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 100 {
		t.Fatalf("contribution=%+v", account)
	}
	replayed, err := service.Post(ctx, intent)
	if err != nil || replayed.ID != first.ID || replayed.FBTransactionID != first.FBTransactionID || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("replay=%+v err=%v", replayed, err)
	}
	for _, changed := range []func(*systemspend.Intent){
		func(x *systemspend.Intent) { x.FBAmount++ },
		func(x *systemspend.Intent) { x.PlayerID = "pg-player-b" },
		func(x *systemspend.Intent) { x.ProducerReference = "changed-ref" },
		func(x *systemspend.Intent) { x.ProducerType = systemspend.ProducerInternalTestNonEligible },
	} {
		other := intent
		changed(&other)
		if _, err := service.Post(ctx, other); !errors.Is(err, systemspend.ErrConflict) {
			t.Fatalf("conflicting replay=%+v err=%v", other, err)
		}
	}
	other := g15Intent("another-operation", 1)
	other.ProducerReference = intent.ProducerReference
	if _, err := service.Post(ctx, other); !errors.Is(err, systemspend.ErrReferenceConflict) {
		t.Fatalf("duplicate producer reference=%v", err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := systemspend.NewService(reopened, g15Registry(t))
	afterRestart, err := restarted.Post(ctx, intent)
	if err != nil || afterRestart.ID != first.ID {
		t.Fatalf("restart replay=%+v err=%v", afterRestart, err)
	}
	if _, err := contributionService.RefundSystemSpend(ctx, g15Refund(first, "g15-partial-40", 40)); err != nil {
		t.Fatal(err)
	}
	partial, err := restarted.Load(ctx, first.OperationID)
	if err != nil || partial.RefundedAmount != 40 || partial.RefundStatus != systemspend.RefundPartial || postgresBalance(t, fb.service, "pg-fb-a") != 9_940 {
		t.Fatalf("partial=%+v err=%v", partial, err)
	}
	if _, err := contributionService.RefundSystemSpend(ctx, g15Refund(first, "g15-final-60", 60)); err != nil {
		t.Fatal(err)
	}
	full, err := restarted.Load(ctx, first.OperationID)
	if err != nil || full.RefundedAmount != 100 || full.RefundStatus != systemspend.RefundFull || postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
		t.Fatalf("full=%+v err=%v", full, err)
	}
	account, _ = contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 0 {
		t.Fatalf("account after full refund=%+v", account)
	}
	report, err := restarted.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 1 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG15NonEligibleSpendNeverMintsContribution(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	service := systemspend.NewService(store, g15Registry(t))
	intent := g15Intent("noneligible-1", 75)
	intent.ProducerType = systemspend.ProducerInternalTestNonEligible
	spend, err := service.Post(context.Background(), intent)
	if err != nil || spend.Eligible || spend.ContributionAmount != 0 || spend.ContributionEntryID != "" || spend.RuleVersion != "" {
		t.Fatalf("noneligible=%+v err=%v", spend, err)
	}
	account, _ := contributionService.Account(context.Background(), "pg-player-a")
	if account.Balance != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 9_925 {
		t.Fatalf("account=%+v", account)
	}
	report, err := service.Reconcile(context.Background())
	if err != nil || !report.Balanced || report.Checked != 1 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG15RecoveryDebtAndFutureEligibleCredit(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	original, err := service.Post(ctx, g15Intent("recovery-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO contribution_consumptions(consumption_id,player_id,amount,created_at) VALUES('g15-prior-use','pg-player-a',80,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE contribution_accounts SET balance=20,revision=revision+1 WHERE player_id='pg-player-a'`); err != nil {
		t.Fatal(err)
	}
	refund, err := contributionService.RefundSystemSpend(ctx, g15Refund(original, "g15-recovery-refund", 100))
	if err != nil || refund.Compensation.DebtCreated != 80 {
		t.Fatalf("refund=%+v err=%v", refund, err)
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 80 || postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
		t.Fatalf("after refund=%+v", account)
	}
	next, err := service.Post(ctx, g15Intent("recovery-next", 50))
	if err != nil || next.ContributionAmount != 50 {
		t.Fatalf("future credit=%+v err=%v", next, err)
	}
	account, _ = contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 30 {
		t.Fatalf("after debt recovery=%+v", account)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 2 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG15RollbackAtEveryEconomicBoundary(t *testing.T) {
	for _, point := range []string{"before_fb_debit", "after_fb_debit", "before_contribution_entry", "after_contribution_entry", "after_economic_effect", "after_spend_record", "before_commit"} {
		t.Run(point, func(t *testing.T) {
			store, contributionService, fb := g13Fixture(t)
			ctx := context.Background()
			service := systemspend.NewService(store, g15Registry(t))
			store.contributionFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected G15 rollback")
				}
				return nil
			}
			store.systemSpendFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected G15 rollback")
				}
				return nil
			}
			intent := g15Intent("rollback-"+point, 10)
			if _, err := service.Post(ctx, intent); err == nil {
				t.Fatal("injected failure committed")
			}
			var count int
			if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM system_spends`).Scan(&count); err != nil || count != 0 {
				t.Fatalf("spend leaked count=%d err=%v", count, err)
			}
			account, _ := contributionService.Account(ctx, "pg-player-a")
			if account.Balance != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
				t.Fatalf("economic effect leaked account=%+v", account)
			}
			store.contributionFailureInjector = nil
			store.systemSpendFailureInjector = nil
			if _, err := service.Post(ctx, intent); err != nil {
				t.Fatalf("retry after rollback=%v", err)
			}
		})
	}
}

func TestG15HundredDuplicateConcurrentRequests(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	intent := g15Intent("hundred-same", 50)
	var wg sync.WaitGroup
	results := make(chan systemspend.SystemSpend, 100)
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value, err := service.Post(ctx, intent)
			results <- value
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	var id string
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent duplicate: %v", err)
		}
	}
	for value := range results {
		if id == "" {
			id = value.ID
		} else if value.ID != id {
			t.Fatalf("different spend IDs: %q %q", id, value.ID)
		}
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 50 || postgresBalance(t, fb.service, "pg-fb-a") != 9_950 {
		t.Fatalf("duplicate economic effect account=%+v", account)
	}
}

func TestG15HundredUniqueConcurrentSpends(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.Post(ctx, g15Intent(fmt.Sprintf("unique-%03d", i), 1))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("unique spend=%v", err)
		}
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 100 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("economic totals account=%+v", account)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 100 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG15HundredMixedSpendAndRefundConcurrentOperations(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	original, err := service.Post(ctx, g15Intent("mixed-original", 1000))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, err := service.Post(ctx, g15Intent(fmt.Sprintf("mixed-spend-%03d", i), 1))
				errs <- err
			} else {
				_, err := contributionService.RefundSystemSpend(ctx, g15Refund(original, fmt.Sprintf("mixed-refund-%03d", i), 1))
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("mixed operation: %v", err)
		}
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if account.Balance != 1000 || postgresBalance(t, fb.service, "pg-fb-a") != 9_000 {
		t.Fatalf("mixed economic totals account=%+v", account)
	}
	loaded, err := service.Load(ctx, original.OperationID)
	if err != nil || loaded.RefundedAmount != 50 {
		t.Fatalf("original after mixed=%+v err=%v", loaded, err)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 51 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG15PostgresProperty400SpendRefundSequences(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	var expectedSpent int64
	var state uint64 = 15015
	for i := 0; i < 400; i++ {
		state = state*6364136223846793005 + 1442695040888963407
		amount := int64(1 + state%10)
		intent := g15Intent(fmt.Sprintf("property-db-%03d", i), amount)
		posted, err := service.Post(ctx, intent)
		if err != nil || posted.ContributionAmount != amount || posted.RuleVersion != contribution.RuleVersionV1 {
			t.Fatalf("iteration=%d post=%+v err=%v", i, posted, err)
		}
		replay, err := service.Post(ctx, intent)
		if err != nil || replay.ID != posted.ID || replay.FBTransactionID != posted.FBTransactionID {
			t.Fatalf("iteration=%d duplicate=%+v err=%v", i, replay, err)
		}
		state = state*6364136223846793005 + 1442695040888963407
		refunded := int64(state % uint64(amount+1))
		if refunded > 0 {
			request := g15Refund(posted, fmt.Sprintf("property-refund-%03d", i), refunded)
			first, err := contributionService.RefundSystemSpend(ctx, request)
			if err != nil || first.TotalRefunded != refunded {
				t.Fatalf("iteration=%d refund=%+v err=%v", i, first, err)
			}
			replayed, err := contributionService.RefundSystemSpend(ctx, request)
			if err != nil || replayed.Compensation.ID != first.Compensation.ID {
				t.Fatalf("iteration=%d refund replay=%+v err=%v", i, replayed, err)
			}
		}
		loaded, err := service.Load(ctx, intent.OperationID)
		if err != nil || loaded.RefundedAmount != refunded || loaded.RuleVersion != contribution.RuleVersionV1 {
			t.Fatalf("iteration=%d loaded=%+v err=%v", i, loaded, err)
		}
		expectedSpent += amount - refunded
		account, err := contributionService.Account(ctx, "pg-player-a")
		if err != nil || account.Balance != expectedSpent || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 10_000-expectedSpent {
			t.Fatalf("iteration=%d expected=%d account=%+v err=%v", i, expectedSpent, account, err)
		}
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced || report.Checked != 400 {
		t.Fatalf("property reconcile=%+v err=%v", report, err)
	}
	t.Log("property_iterations=400 passed=400")
}

func TestG15PostgresFailClosedSecurity(t *testing.T) {
	store, _, fb := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	for name, mutate := range map[string]func(*systemspend.Intent){
		"insufficient_fb": func(x *systemspend.Intent) { x.FBAmount = 10_001 },
		"cross_player":    func(x *systemspend.Intent) { x.PlayerID = "pg-player-b" },
		"wrong_account":   func(x *systemspend.Intent) { x.PlayerFBAccountID = "pg-fb-b" },
	} {
		intent := g15Intent("security-"+name, 10)
		mutate(&intent)
		if _, err := service.Post(ctx, intent); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	if postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
		t.Fatal("fail-closed requests changed FB")
	}
	var count int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM system_spends`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed requests created spend count=%d err=%v", count, err)
	}
}

func TestG15ReconciliationDetectsRuleMismatch(t *testing.T) {
	store, _, _ := g13Fixture(t)
	ctx := context.Background()
	service := systemspend.NewService(store, g15Registry(t))
	intent := g15Intent("corrupted-rule", 10)
	if _, err := service.Post(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `ALTER TABLE system_spends DISABLE TRIGGER system_spends_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE system_spends SET rule_version='CORRUPTED_RULE' WHERE operation_id=$1`, intent.OperationID); err != nil {
		t.Fatal(err)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || report.Balanced || len(report.Mismatches) == 0 {
		t.Fatalf("corrupted rule undetected report=%+v err=%v", report, err)
	}
	if _, err := service.Load(ctx, intent.OperationID); !errors.Is(err, systemspend.ErrReconciliation) {
		t.Fatalf("corrupt spend loaded: %v", err)
	}
}

func TestG15MigrationUpgradesG14SchemaWithoutLosingAccounts(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if len(entry.Name()) < 5 || entry.Name()[:4] > "0006" {
			continue
		}
		name := "migrations/" + entry.Name()
		contents, readErr := fs.ReadFile(migrations, name)
		if readErr != nil {
			t.Fatal(readErr)
		}
		old[name] = &fstest.MapFile{Data: contents}
	}
	if len(old) != 6 {
		t.Fatalf("historical migration count=%d", len(old))
	}
	if err = store.MigrateFS(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO contribution_accounts(player_id,balance,recovery_debt,revision,created_at,updated_at) VALUES('g15-upgrade-player',0,0,1,now(),now())`); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var versions, accounts, spends int
	if err = store.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if err = store.pool.QueryRow(ctx, `SELECT count(*) FROM contribution_accounts WHERE player_id='g15-upgrade-player'`).Scan(&accounts); err != nil {
		t.Fatal(err)
	}
	if err = store.pool.QueryRow(ctx, `SELECT count(*) FROM system_spends`).Scan(&spends); err != nil {
		t.Fatal(err)
	}
	if versions != 7 || accounts != 1 || spends != 0 {
		t.Fatalf("upgrade versions=%d accounts=%d spends=%d", versions, accounts, spends)
	}
}

type lostCommitResponseRepository struct {
	store *Store
	lost  bool
}

func (r *lostCommitResponseRepository) PostSystemSpend(ctx context.Context, request systemspend.ResolvedIntent) (systemspend.SystemSpend, error) {
	value, err := r.store.PostSystemSpend(ctx, request)
	if err == nil && !r.lost {
		r.lost = true
		return systemspend.SystemSpend{}, errors.New("simulated lost commit response")
	}
	return value, err
}
func (r *lostCommitResponseRepository) LoadSystemSpend(ctx context.Context, operation string) (systemspend.SystemSpend, error) {
	return r.store.LoadSystemSpend(ctx, operation)
}
func (r *lostCommitResponseRepository) ReconcileSystemSpends(ctx context.Context) (systemspend.ReconciliationReport, error) {
	return r.store.ReconcileSystemSpends(ctx)
}

func TestG15LostCommitResponseRestartRetryExactlyOnce(t *testing.T) {
	store, contributionService, fb := g13Fixture(t)
	ctx := context.Background()
	intent := g15Intent("lost-response", 100)
	service := systemspend.NewService(&lostCommitResponseRepository{store: store}, g15Registry(t))
	if _, err := service.Post(ctx, intent); err == nil {
		t.Fatal("lost response was delivered")
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	retried := systemspend.NewService(reopened, g15Registry(t))
	spend, err := retried.Post(ctx, intent)
	if err != nil || spend.FBAmount != 100 {
		t.Fatalf("retry=%+v err=%v", spend, err)
	}
	var spends, credits int
	if err = reopened.pool.QueryRow(ctx, `SELECT count(*) FROM system_spends WHERE operation_id=$1`, intent.OperationID).Scan(&spends); err != nil {
		t.Fatal(err)
	}
	if err = reopened.pool.QueryRow(ctx, `SELECT count(*) FROM contribution_entries WHERE source_id=$1`, "g15:"+intent.OperationID).Scan(&credits); err != nil {
		t.Fatal(err)
	}
	account, _ := contributionService.Account(ctx, "pg-player-a")
	if spends != 1 || credits != 1 || account.Balance != 100 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("effects spends=%d credits=%d account=%+v", spends, credits, account)
	}
}

func TestG15RefundPolicySnapshotIsEnforcedByG14(t *testing.T) {
	store, contributionService, _ := g13Fixture(t)
	ctx := context.Background()
	partialForbidden, err := systemspend.NewRegistry([]systemspend.Producer{{Type: systemspend.ProducerInternalTestEligible, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Refundable: true, PartialRefundAllowed: false, Active: true, Description: "full refund only"}})
	if err != nil {
		t.Fatal(err)
	}
	fullOnly := systemspend.NewService(store, partialForbidden)
	spend, err := fullOnly.Post(ctx, g15Intent("full-only", 100))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contributionService.RefundSystemSpend(ctx, g15Refund(spend, "forbidden-partial", 40)); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("partial refund allowed: %v", err)
	}
	if _, err = contributionService.RefundSystemSpend(ctx, g15Refund(spend, "permitted-full", 100)); err != nil {
		t.Fatalf("full refund failed: %v", err)
	}
	notRefundable, err := systemspend.NewRegistry([]systemspend.Producer{{Type: systemspend.ProducerInternalTestEligible, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Active: true, Description: "no refunds"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := systemspend.NewService(store, notRefundable).Post(ctx, g15Intent("not-refundable", 100))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = contributionService.RefundSystemSpend(ctx, g15Refund(second, "forbidden-full", 100)); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("nonrefundable spend refunded: %v", err)
	}
}
