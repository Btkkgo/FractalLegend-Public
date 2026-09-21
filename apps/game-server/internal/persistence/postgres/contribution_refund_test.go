package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"sync"
	"testing"
	"testing/fstest"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5"
)

func g14Refund(original contribution.PostingResult, id string, amount int64) contribution.RefundRequest {
	return contribution.RefundRequest{
		OriginalFBTransactionID: original.FBTransaction.ID,
		PlayerID:                original.Entry.PlayerID,
		PlayerFBAccountID:       original.Entry.PlayerFBAccountID,
		ReferenceID:             id, Amount: amount,
	}
}

func TestG14ReversalAndFailClosedReferences(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-reversal-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	reversal := g14Refund(original, "g14-reversal", 25)
	reversal.Reversal = true
	result, err := service.RefundSystemSpend(ctx, reversal)
	if err != nil || result.FBTransaction.Type != ledger.TransactionReversal || result.Compensation.Amount != 25 {
		t.Fatalf("reversal=%+v err=%v", result, err)
	}
	changed := reversal
	changed.Amount = 26
	if _, err := service.RefundSystemSpend(ctx, changed); !errors.Is(err, contribution.ErrRefundConflict) {
		t.Fatalf("conflicting id=%v", err)
	}
	wrongPlayer := g14Refund(original, "wrong-player", 1)
	wrongPlayer.PlayerID = "pg-player-b"
	if _, err := service.RefundSystemSpend(ctx, wrongPlayer); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("wrong player=%v", err)
	}
	wrongAccount := g14Refund(original, "wrong-account", 1)
	wrongAccount.PlayerFBAccountID = "pg-fb-b"
	if _, err := service.RefundSystemSpend(ctx, wrongAccount); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("wrong account=%v", err)
	}
	missing := g14Refund(original, "missing-original", 1)
	missing.OriginalFBTransactionID = "missing"
	if _, err := service.RefundSystemSpend(ctx, missing); !errors.Is(err, contribution.ErrNotFound) {
		t.Fatalf("missing=%v", err)
	}
	noneligible, err := fb.service.SystemSpend(ctx, "pg-fb-a", "pg-fb-system", 10, ledger.SpendNonEligible, ledger.LedgerReference{Type: "G14_NONELIGIBLE", ID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	request := g14Refund(original, "noneligible", 1)
	request.OriginalFBTransactionID = noneligible.ID
	if _, err := service.RefundSystemSpend(ctx, request); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("noneligible=%v", err)
	}
	ordinary, err := fb.service.Refund(ctx, noneligible.ID, ledger.LedgerReference{Type: "G14_NONELIGIBLE_REFUND", ID: "one"})
	if err != nil || ordinary.Type != ledger.TransactionRefund {
		t.Fatalf("ordinary refund=%+v err=%v", ordinary, err)
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 75 || account.RecoveryDebt != 0 {
		t.Fatalf("contribution changed=%+v", account)
	}
	transfer, err := fb.service.TransferFB(ctx, "pg-fb-a", "pg-fb-b", 1, ledger.LedgerReference{Type: "G14_TRANSFER", ID: "one"})
	if err != nil {
		t.Fatal(err)
	}
	request.OriginalFBTransactionID = transfer.ID
	request.ReferenceID = "transfer"
	if _, err := service.RefundSystemSpend(ctx, request); !errors.Is(err, contribution.ErrInvalidRefund) {
		t.Fatalf("transfer=%v", err)
	}
}

func TestG14UnknownOriginalRuleAndManualReview(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-unknown-rule", 100))
	if err != nil {
		t.Fatal(err)
	}
	// A damaged historical row cannot be silently recomputed with today's rule.
	if _, err = store.pool.Exec(ctx, `ALTER TABLE contribution_entries DISABLE TRIGGER contribution_entries_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `ALTER TABLE contribution_entries DROP CONSTRAINT contribution_entries_check1`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE contribution_entries SET rule_version='UNKNOWN_RULE' WHERE entry_id=$1`, original.Entry.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RefundSystemSpend(ctx, g14Refund(original, "unknown-rule", 10)); !errors.Is(err, contribution.ErrRuleVersion) {
		t.Fatalf("unknown rule=%v", err)
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 100 || !account.ReviewRequired {
		t.Fatalf("account=%+v", account)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE contribution_accounts SET balance=99 WHERE player_id='pg-player-a'`); err != nil {
		t.Fatal(err)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || report.Balanced || len(report.Mismatches) == 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	account, _ = service.Account(ctx, "pg-player-a")
	if !account.ReviewRequired {
		t.Fatalf("manual review not persisted: %+v", account)
	}
	if err := service.ValidateSpend(ctx, "pg-player-a", 1); !errors.Is(err, contribution.ErrManualReview) {
		t.Fatalf("guard=%v", err)
	}
}

func TestG14ReconciliationDetectsMissingOrphanAndExcessCompensation(t *testing.T) {
	for _, scenario := range []string{"missing", "orphan", "excess"} {
		t.Run(scenario, func(t *testing.T) {
			store, service, _ := g13Fixture(t)
			ctx := context.Background()
			original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-reconcile", 100))
			if err != nil {
				t.Fatal(err)
			}
			result, err := service.RefundSystemSpend(ctx, g14Refund(original, "g14-reconcile-refund", 40))
			if err != nil {
				t.Fatal(err)
			}
			other, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-other", 10))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.pool.Exec(ctx, `ALTER TABLE contribution_compensations DISABLE TRIGGER contribution_compensations_immutable`); err != nil {
				t.Fatal(err)
			}
			if _, err = store.pool.Exec(ctx, `ALTER TABLE contribution_compensations DISABLE TRIGGER contribution_compensation_valid`); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "missing":
				_, err = store.pool.Exec(ctx, `DELETE FROM contribution_compensations WHERE compensation_id=$1`, result.Compensation.ID)
			case "orphan":
				_, err = store.pool.Exec(ctx, `UPDATE contribution_compensations SET original_fb_transaction_id=$2 WHERE compensation_id=$1`, result.Compensation.ID, other.FBTransaction.ID)
			case "excess":
				_, err = store.pool.Exec(ctx, `UPDATE contribution_compensations SET amount=41,available_reversed=41,balance_after=59 WHERE compensation_id=$1`, result.Compensation.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			report, err := service.Reconcile(ctx)
			if err != nil || report.Balanced {
				t.Fatalf("scenario=%s report=%+v err=%v", scenario, report, err)
			}
		})
	}
}

func TestG14MigrationUpgradesExistingG13Account(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	for version := 1; version <= 5; version++ {
		path := fmt.Sprintf("migrations/%04d_", version)
		entries, err := fs.ReadDir(migrations, "migrations")
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if len(entry.Name()) >= 5 && "migrations/"+entry.Name()[:5] == path {
				data, err := fs.ReadFile(migrations, "migrations/"+entry.Name())
				if err != nil {
					t.Fatal(err)
				}
				old["migrations/"+entry.Name()] = &fstest.MapFile{Data: data}
			}
		}
	}
	if len(old) != 5 {
		t.Fatalf("old migration count=%d", len(old))
	}
	if err := store.MigrateFS(ctx, old); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO contribution_accounts(player_id,balance,revision,created_at,updated_at) VALUES('existing-g13',0,1,now(),now())`); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	account, err := store.LoadContributionAccount(ctx, "existing-g13")
	if err != nil || account.RecoveryDebt != 0 || account.ReviewRequired || account.Balance != 0 {
		t.Fatalf("upgraded=%+v err=%v", account, err)
	}
}

func TestG14DatabaseRejectsOrphanFBCompensation(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-db-orphan", 100))
	if err != nil {
		t.Fatal(err)
	}
	for _, referenceType := range []string{"UNAUTHORIZED_REFUND", "CONTRIBUTION_REFUND"} {
		tx, err := store.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO fb_ledger_transactions(transaction_id,transaction_type,status,reference_type,reference_id,original_transaction_id,spend_classification,created_at) VALUES($1,'REFUND','POSTED',$2,$3,$4,'ELIGIBLE',now())`, "orphan-"+referenceType, referenceType, "orphan-"+referenceType, original.FBTransaction.ID)
		if err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(ctx); err == nil {
			t.Fatalf("orphan %s committed", referenceType)
		}
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 100 {
		t.Fatalf("account=%+v", account)
	}
}

func TestG14FullPartialAndOverRefund(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	for i, amount := range []int64{20, 30, 50} {
		result, err := service.RefundSystemSpend(ctx, g14Refund(original, fmt.Sprintf("partial-%d", i), amount))
		if err != nil || result.Compensation.Amount != amount || result.TotalRefunded != []int64{20, 50, 100}[i] {
			t.Fatalf("refund %d: %+v %v", i, result, err)
		}
	}
	compensations, err := service.Compensations(ctx, "pg-player-a")
	if err != nil || len(compensations) != 3 || compensations[0].OriginalEntryID != original.Entry.ID || compensations[2].Amount != 50 {
		t.Fatalf("audit=%+v err=%v", compensations, err)
	}
	if _, err := service.RefundSystemSpend(ctx, g14Refund(original, "excess", 1)); !errors.Is(err, contribution.ErrOverRefund) {
		t.Fatalf("over refund: %v", err)
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
		t.Fatalf("account=%+v FB=%d", account, postgresBalance(t, fb.service, "pg-fb-a"))
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG14RecoveryDebtAndCreditOrdering(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-debt-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic prior use: G14 does not expose a production Contribution spend producer.
	_, err = store.pool.Exec(ctx, `INSERT INTO contribution_consumptions(consumption_id,player_id,amount,created_at) VALUES('prior-use','pg-player-a',80,now())`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `UPDATE contribution_accounts SET balance=20,revision=revision+1 WHERE player_id='pg-player-a'`)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.RefundSystemSpend(ctx, g14Refund(original, "g14-debt-refund", 100))
	if err != nil || result.Compensation.DebtCreated != 80 {
		t.Fatalf("refund=%+v err=%v", result, err)
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 80 {
		t.Fatalf("debt=%+v", account)
	}
	if err := service.ValidateSpend(ctx, "pg-player-a", 1); !errors.Is(err, contribution.ErrRecoveryHold) {
		t.Fatalf("spend guard=%v", err)
	}
	for i, wantDebt := range []int64{30, 0} {
		posted, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("g14-repay-%d", i), 50))
		if err != nil || posted.Entry.DebtSettled != []int64{50, 30}[i] {
			t.Fatalf("repay=%+v err=%v", posted, err)
		}
		account, _ = service.Account(ctx, "pg-player-a")
		if account.RecoveryDebt != wantDebt || account.Balance != []int64{0, 20}[i] {
			t.Fatalf("account=%+v", account)
		}
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG14ConcurrentDuplicateAndRestartReplay(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-concurrent-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	request := g14Refund(original, "same-id", 40)
	var wg sync.WaitGroup
	results := make(chan contribution.RefundResult, 100)
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := service.RefundSystemSpend(ctx, request)
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("duplicate: %v", err)
		}
	}
	for result := range results {
		if result.TotalRefunded != 40 {
			t.Errorf("result=%+v", result)
		}
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	result, err := contribution.NewService(reopened).RefundSystemSpend(ctx, request)
	if err != nil || result.TotalRefunded != 40 || postgresBalance(t, fb.service, "pg-fb-a") != 9_940 {
		t.Fatalf("replay=%+v err=%v", result, err)
	}
}

func TestG14ConcurrentDifferentRefundsCannotExceedOriginal(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-race-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.RefundSystemSpend(ctx, g14Refund(original, fmt.Sprintf("race-%d", i), 70))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	var passed, rejected int
	for err := range results {
		if err == nil {
			passed++
		} else if errors.Is(err, contribution.ErrOverRefund) {
			rejected++
		} else {
			t.Fatal(err)
		}
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if passed != 1 || rejected != 1 || account.Balance != 30 || postgresBalance(t, fb.service, "pg-fb-a") != 9_970 {
		t.Fatalf("passed=%d rejected=%d account=%+v", passed, rejected, account)
	}
}

func TestG14HundredDifferentPartialRefundsAndConcurrentCredits(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-hundred-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.RefundSystemSpend(ctx, g14Refund(original, fmt.Sprintf("hundred-%03d", i), 1))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("concurrent partial: %v", err)
		}
	}
	account, _ := service.Account(ctx, "pg-player-a")
	if account.Balance != 0 || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 10_000 {
		t.Fatalf("account=%+v", account)
	}
	// Credit and refund share the same FB and Contribution account lock order.
	original2, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-credit-race-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs2 := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			if i%2 == 0 {
				_, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("g14-credit-race-%03d", i), 1))
				errs2 <- err
			} else {
				_, err := service.RefundSystemSpend(ctx, g14Refund(original2, fmt.Sprintf("g14-refund-race-%03d", i), 1))
				errs2 <- err
			}
		}(i)
	}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			report, err := service.Reconcile(ctx)
			if err != nil {
				errs2 <- err
			} else if !report.Balanced {
				errs2 <- errors.New("concurrent reconciliation falsely reported mismatch")
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs2)
	for err := range errs2 {
		if err != nil {
			t.Errorf("credit/refund race: %v", err)
		}
	}
	account, _ = service.Account(ctx, "pg-player-a")
	if account.Balance != 100 || account.RecoveryDebt != 0 {
		t.Fatalf("final account=%+v", account)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestG14RollbackAfterFBAndAfterCompensation(t *testing.T) {
	for _, point := range []string{contribution.FailureAfterFBRefund, contribution.FailureAfterCompensation, contribution.FailureBeforeCommit} {
		t.Run(point, func(t *testing.T) {
			store, service, fb := g13Fixture(t)
			ctx := context.Background()
			original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-failure-original", 100))
			if err != nil {
				t.Fatal(err)
			}
			store.contributionFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected G14 failure")
				}
				return nil
			}
			if _, err := service.RefundSystemSpend(ctx, g14Refund(original, "g14-failure", 40)); err == nil {
				t.Fatal("partial commit")
			}
			store.contributionFailureInjector = nil
			account, _ := service.Account(ctx, "pg-player-a")
			if account.Balance != 100 || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
				t.Fatalf("partial state=%+v", account)
			}
			report, err := service.Reconcile(ctx)
			if err != nil || !report.Balanced {
				t.Fatalf("report=%+v err=%v", report, err)
			}
			if _, err := service.RefundSystemSpend(ctx, g14Refund(original, "g14-failure", 40)); err != nil {
				t.Fatalf("retry after rollback: %v", err)
			}
		})
	}
}

func TestG14DeferredFBLedgerCommitFailureRollsBackCompensation(t *testing.T) {
	store, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-deferred-failure-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `CREATE FUNCTION g14_fail_fb_commit() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'simulated FB ledger commit failure'; END; $$ LANGUAGE plpgsql;
       CREATE CONSTRAINT TRIGGER g14_fb_commit_failure AFTER INSERT ON fb_ledger_transactions DEFERRABLE INITIALLY DEFERRED
       FOR EACH ROW WHEN (NEW.reference_type='CONTRIBUTION_REFUND') EXECUTE FUNCTION g14_fail_fb_commit()`)
	if err != nil {
		t.Fatal(err)
	}
	request := g14Refund(original, "g14-deferred-failure", 40)
	if _, err = service.RefundSystemSpend(ctx, request); err == nil {
		t.Fatal("deferred FB failure committed")
	}
	account, _ := service.Account(ctx, "pg-player-a")
	compensations, err := service.Compensations(ctx, "pg-player-a")
	if err != nil || len(compensations) != 0 || account.Balance != 100 || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("partial commit account=%+v compensations=%+v err=%v", account, compensations, err)
	}
	if _, err = store.pool.Exec(ctx, `DROP TRIGGER g14_fb_commit_failure ON fb_ledger_transactions; DROP FUNCTION g14_fail_fb_commit()`); err != nil {
		t.Fatal(err)
	}
	if _, err = service.RefundSystemSpend(ctx, request); err != nil {
		t.Fatalf("retry after failed commit: %v", err)
	}
}

func TestG14ConnectionLossBeforeCommitRestartsWithoutHalfRefund(t *testing.T) {
	store, service, fb := g13Fixture(t)
	ctx := context.Background()
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-crash-original", 100))
	if err != nil {
		t.Fatal(err)
	}
	request := g14Refund(original, "g14-crash-refund", 40)
	conn, err := pgx.Connect(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.refundContributionSystemSpendTx(ctx, tx, request); err != nil {
		t.Fatal(err)
	}
	// Closing the connection without COMMIT models a process/connection loss.
	if err = conn.Close(ctx); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := contribution.NewService(reopened)
	account, err := restarted.Account(ctx, "pg-player-a")
	if err != nil || account.Balance != 100 || account.RecoveryDebt != 0 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("uncommitted state leaked: %+v err=%v", account, err)
	}
	first, err := restarted.RefundSystemSpend(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := restarted.RefundSystemSpend(ctx, request)
	if err != nil || replay.Compensation.ID != first.Compensation.ID {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	account, _ = restarted.Account(ctx, "pg-player-a")
	if account.Balance != 60 || postgresBalance(t, fb.service, "pg-fb-a") != 9_940 {
		t.Fatalf("restart account=%+v", account)
	}
}

func TestG14PropertyTwoHundredRefundSequences(t *testing.T) {
	_, service, _ := g13Fixture(t)
	ctx := context.Background()
	rng := rand.New(rand.NewSource(14))
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-property-original", 5000))
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for i := 0; i < 200; i++ {
		amount := int64(rng.Intn(40) + 1)
		if amount > 5000-total {
			amount = 5000 - total
		}
		if amount == 0 {
			break
		}
		request := g14Refund(original, fmt.Sprintf("g14-property-%03d", i), amount)
		first, err := service.RefundSystemSpend(ctx, request)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		second, err := service.RefundSystemSpend(ctx, request)
		if err != nil || second.Compensation.ID != first.Compensation.ID {
			t.Fatalf("replay %d: %+v %v", i, second, err)
		}
		total += amount
		account, _ := service.Account(ctx, "pg-player-a")
		if total > 5000 || first.TotalRefunded != total || account.Balance != 5000-total || account.RecoveryDebt < 0 {
			t.Fatalf("iteration %d: total=%d account=%+v", i, total, account)
		}
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestG14PropertyDebtRecoveryTwoHundredIterations(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	rng := rand.New(rand.NewSource(1402))
	original, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "g14-debt-property-original", 5000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `INSERT INTO contribution_consumptions(consumption_id,player_id,amount,created_at) VALUES('property-prior-use','pg-player-a',4500,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err = store.pool.Exec(ctx, `UPDATE contribution_accounts SET balance=500,revision=revision+1 WHERE player_id='pg-player-a'`); err != nil {
		t.Fatal(err)
	}
	var refunded, additionalEarned int64
	for i := 0; i < 200; i++ {
		amount := int64(rng.Intn(20) + 1)
		request := g14Refund(original, fmt.Sprintf("g14-debt-property-refund-%03d", i), amount)
		first, err := service.RefundSystemSpend(ctx, request)
		if err != nil {
			t.Fatalf("refund iteration %d: %v", i, err)
		}
		replay, err := service.RefundSystemSpend(ctx, request)
		if err != nil || replay.Compensation.ID != first.Compensation.ID {
			t.Fatalf("replay iteration %d: %+v %v", i, replay, err)
		}
		refunded += amount
		credit := int64(rng.Intn(10) + 1)
		before, _ := service.Account(ctx, "pg-player-a")
		posted, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("g14-debt-property-credit-%03d", i), credit))
		if err != nil {
			t.Fatalf("credit iteration %d: %v", i, err)
		}
		wantSettled := credit
		if before.RecoveryDebt < wantSettled {
			wantSettled = before.RecoveryDebt
		}
		if posted.Entry.DebtSettled != wantSettled {
			t.Fatalf("iteration %d settled=%d want=%d", i, posted.Entry.DebtSettled, wantSettled)
		}
		additionalEarned += credit
		account, _ := service.Account(ctx, "pg-player-a")
		net := 5000 + additionalEarned - 4500 - refunded
		wantAvailable, wantDebt := net, int64(0)
		if net < 0 {
			wantAvailable = 0
			wantDebt = -net
		}
		if refunded > 5000 || account.Balance != wantAvailable || account.RecoveryDebt != wantDebt || (account.RecoveryDebt > 0 && account.Balance > 0) {
			t.Fatalf("iteration %d refunded=%d account=%+v want available=%d debt=%d", i, refunded, account, wantAvailable, wantDebt)
		}
		if wantDebt > 0 && !errors.Is(service.ValidateSpend(ctx, "pg-player-a", 1), contribution.ErrRecoveryHold) {
			t.Fatalf("iteration %d debt bypass", i)
		}
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}
