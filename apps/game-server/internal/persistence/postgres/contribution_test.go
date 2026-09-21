package postgres

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5/pgconn"
)

func g13Fixture(t *testing.T) (*Store, *contribution.Service, postgresLedgerFixture) {
	t.Helper()
	fixture := newPostgresLedgerFixture(t)
	postgresCredit(t, fixture, "pg-fb-a", 10_000, "g13-start-a")
	postgresCredit(t, fixture, "pg-fb-b", 10_000, "g13-start-b")
	service := contribution.NewService(fixture.store)
	for _, player := range []string{"pg-player-a", "pg-player-b"} {
		if _, err := service.CreateAccount(context.Background(), player); err != nil {
			t.Fatal(err)
		}
	}
	return fixture.store, service, fixture
}

func g13Request(player, account, id string, amount int64) contribution.SpendRequest {
	return contribution.SpendRequest{
		PlayerID: player, PlayerFBAccountID: account, SystemFBAccountID: "pg-fb-system",
		Source: contribution.SourceSystemService, SourceID: id, EligibleSpend: amount,
		RuleVersion: contribution.RuleVersionV1,
	}
}

func TestG13PostgresAtomicSpendIdempotencyRestartAndAudit(t *testing.T) {
	store, service, fb := g13Fixture(t)
	ctx := context.Background()
	request := g13Request("pg-player-a", "pg-fb-a", "system-service-1", 100)
	first, err := service.PostSystemSpend(ctx, request)
	if err != nil || first.Entry.Amount != 100 || first.Entry.BalanceBefore != 0 || first.Entry.BalanceAfter != 100 || first.Entry.RuleVersion != contribution.RuleVersionV1 || first.FBTransaction.Type != "SYSTEM_SPEND" {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	if got := postgresBalance(t, fb.service, "pg-fb-a"); got != 9_900 {
		t.Fatalf("FB after spend=%d", got)
	}
	second, err := service.PostSystemSpend(ctx, request)
	if err != nil || second.Entry.ID != first.Entry.ID || second.FBTransaction.ID != first.FBTransaction.ID {
		t.Fatalf("replay=%+v err=%v", second, err)
	}
	if got := postgresBalance(t, fb.service, "pg-fb-a"); got != 9_900 {
		t.Fatalf("FB changed on replay=%d", got)
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	restarted := contribution.NewService(reopened)
	third, err := restarted.PostSystemSpend(ctx, request)
	if err != nil || third.Entry.ID != first.Entry.ID {
		t.Fatalf("restart replay=%+v err=%v", third, err)
	}
	account, err := restarted.Account(ctx, "pg-player-a")
	if err != nil || account.Balance != 100 || account.Revision != 2 {
		t.Fatalf("account=%+v err=%v", account, err)
	}
	entries, err := restarted.Entries(ctx, "pg-player-a")
	if err != nil || len(entries) != 1 || entries[0].FBTransactionID != first.FBTransaction.ID {
		t.Fatalf("entries=%+v err=%v", entries, err)
	}
	audit, err := restarted.AuditEvents(ctx, "pg-player-a")
	if err != nil || len(audit) != 1 || audit[0].EntryID != first.Entry.ID {
		t.Fatalf("audit=%+v err=%v", audit, err)
	}
	report, err := restarted.Reconcile(ctx)
	if err != nil || !report.Balanced || len(report.Mismatches) != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if _, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "system-service-1", 101)); !errors.Is(err, contribution.ErrSourceConflict) {
		t.Fatalf("changed intent error=%v", err)
	}
	_ = store
}

func TestG13PostgresAllFailureGatesRollbackFBAndContribution(t *testing.T) {
	for _, point := range []string{
		contribution.FailureBeforeFBDebit,
		contribution.FailureAfterFBDebit,
		contribution.FailureBeforeEntry,
		contribution.FailureAfterEntry,
		contribution.FailureAfterAccountUpdate,
		contribution.FailureBeforeFinalState,
		contribution.FailureBeforeCommit,
	} {
		t.Run(point, func(t *testing.T) {
			store, service, fb := g13Fixture(t)
			store.contributionFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected G13 failure")
				}
				return nil
			}
			request := g13Request("pg-player-a", "pg-fb-a", "failed-"+point, 10)
			if _, err := service.PostSystemSpend(context.Background(), request); err == nil {
				t.Fatal("injected failure committed")
			}
			if got := postgresBalance(t, fb.service, "pg-fb-a"); got != 10_000 {
				t.Fatalf("FB changed after rollback=%d", got)
			}
			account, _ := service.Account(context.Background(), "pg-player-a")
			entries, _ := service.Entries(context.Background(), "pg-player-a")
			if account.Balance != 0 || len(entries) != 0 {
				t.Fatalf("partial Contribution account=%+v entries=%+v", account, entries)
			}
			store.contributionFailureInjector = nil
			if _, err := service.PostSystemSpend(context.Background(), request); err != nil {
				t.Fatalf("retry after rollback: %v", err)
			}
		})
	}
}

func TestG13PostgresHighConcurrencyAndDuplicateSource(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	var success atomic.Int64
	errorsSeen := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("concurrent-%d", i), 1))
			if err != nil {
				errorsSeen <- err
				return
			}
			success.Add(1)
		}(i)
	}
	wg.Wait()
	close(errorsSeen)
	for err := range errorsSeen {
		t.Errorf("concurrent posting: %v", err)
	}
	account, err := service.Account(ctx, "pg-player-a")
	if err != nil || success.Load() != 100 || account.Balance != 100 || account.Revision != 101 || postgresBalance(t, fb.service, "pg-fb-a") != 9_900 {
		t.Fatalf("success=%d account=%+v err=%v", success.Load(), account, err)
	}
	entries, err := service.Entries(ctx, "pg-player-a")
	if err != nil || len(entries) != 100 {
		t.Fatalf("concurrent entries=%d err=%v", len(entries), err)
	}
	for i, entry := range entries {
		if entry.BalanceBefore != int64(i) || entry.BalanceAfter != int64(i+1) {
			t.Fatalf("concurrent entry order at %d: %+v", i, entry)
		}
	}
	request := g13Request("pg-player-b", "pg-fb-b", "same-source", 5)
	var duplicated atomic.Int64
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.PostSystemSpend(ctx, request); err != nil {
				t.Errorf("duplicate: %v", err)
			} else {
				duplicated.Add(1)
			}
		}()
	}
	wg.Wait()
	other, _ := service.Account(ctx, "pg-player-b")
	if duplicated.Load() != 100 || other.Balance != 5 || other.Revision != 2 || postgresBalance(t, fb.service, "pg-fb-b") != 9_995 {
		t.Fatalf("duplicates=%d other=%+v", duplicated.Load(), other)
	}
}

func TestG13PostgresNonEligibleSourcesAndG12TradeCreditZero(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	service := contribution.NewService(fixture.store)
	ctx := context.Background()
	for _, player := range []string{"trade-player-a", "trade-player-b"} {
		if _, err := service.CreateAccount(ctx, player); err != nil {
			t.Fatal(err)
		}
	}
	request := contribution.SpendRequest{PlayerID: "trade-player-a", PlayerFBAccountID: "g12-fb-a", SystemFBAccountID: "g12-fb-c", Source: contribution.SourceDeposit, SourceID: "deposit-test", EligibleSpend: 100, RuleVersion: contribution.RuleVersionV1}
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrIneligibleSource) {
		t.Fatalf("deposit error=%v", err)
	}
	request.Source = contribution.Source("CLIENT_FORGED")
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrUnknownSource) {
		t.Fatalf("unknown error=%v", err)
	}
	preparePostgresG12FBOnly(t, fixture, "g13-trade-regression", "trade-player-a", "trade-player-b", 100, 0)
	if _, err := fixture.trade.FinalizeTrade(ctx, "g13-trade-regression"); err != nil {
		t.Fatal(err)
	}
	if got := postgresBalance(t, fixture.ledger, "g12-fb-a"); got != 900 {
		t.Fatalf("buyer FB=%d", got)
	}
	if got := postgresBalance(t, fixture.ledger, "g12-fb-b"); got != 600 {
		t.Fatalf("seller FB=%d", got)
	}
	for _, player := range []string{"trade-player-a", "trade-player-b"} {
		account, err := service.Account(ctx, player)
		if err != nil || account.Balance != 0 {
			t.Fatalf("player=%s account=%+v err=%v", player, account, err)
		}
	}
}

func TestG13PostgresPropertySequenceTwoHundredIterations(t *testing.T) {
	store, service, fb := g13Fixture(t)
	ctx := context.Background()
	rng := rand.New(rand.NewSource(13))
	var expected int64
	var credits int
	for i := 0; i < 200; i++ {
		amount := int64(rng.Intn(20) + 1)
		request := g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("property-%03d", i), amount)
		if i%5 == 0 {
			request.Source = contribution.SourceRefund
			if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrIneligibleSource) {
				t.Fatalf("iteration %d refund error=%v", i, err)
			}
			continue
		}
		if i%17 == 0 {
			store.contributionFailureInjector = func(point string) error {
				if point == contribution.FailureAfterEntry {
					return errors.New("property rollback")
				}
				return nil
			}
			if _, err := service.PostSystemSpend(ctx, request); err == nil {
				t.Fatalf("iteration %d failure committed", i)
			}
			store.contributionFailureInjector = nil
			continue
		}
		first, err := service.PostSystemSpend(ctx, request)
		if err != nil || first.Entry.Amount != amount || first.Entry.EligibleSpend != amount {
			t.Fatalf("iteration %d first=%+v err=%v", i, first, err)
		}
		if i%7 == 0 {
			replay, err := service.PostSystemSpend(ctx, request)
			if err != nil || replay.Entry.ID != first.Entry.ID {
				t.Fatalf("iteration %d replay=%+v err=%v", i, replay, err)
			}
		}
		expected += amount
		credits++
	}
	account, _ := service.Account(ctx, "pg-player-a")
	entries, err := service.Entries(ctx, "pg-player-a")
	if err != nil || account.Balance != expected || len(entries) != credits || postgresBalance(t, fb.service, "pg-fb-a") != 10_000-expected {
		t.Fatalf("account=%+v credits=%d entries=%d err=%v", account, credits, len(entries), err)
	}
	var running int64
	for _, entry := range entries {
		if entry.BalanceBefore != running || entry.BalanceAfter != running+entry.Amount || entry.RuleVersion != contribution.RuleVersionV1 {
			t.Fatalf("entry continuity: %+v running=%d", entry, running)
		}
		running = entry.BalanceAfter
	}
	report, err := service.Reconcile(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
}

func TestG13PostgresReconciliationDetectsDriftAndHistoryIsImmutable(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	posted, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "drift-1", 10))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE contribution_entries SET amount=11 WHERE entry_id=$1`, posted.Entry.ID); err == nil {
		t.Fatal("posted contribution entry was mutable")
	}
	if _, err := store.pool.Exec(ctx, `UPDATE contribution_accounts SET balance=balance+1 WHERE player_id='pg-player-a'`); err != nil {
		t.Fatal(err)
	}
	report, err := service.Reconcile(ctx)
	if err != nil || report.Balanced || len(report.Mismatches) != 1 || report.Mismatches[0].Balance != 11 || report.Mismatches[0].EntriesTotal != 10 {
		t.Fatalf("drift report=%+v err=%v", report, err)
	}
}

func TestG13PostgresRuleTamperingAndCrossOwnerSpendFailClosed(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	request := g13Request("pg-player-a", "pg-fb-a", "reject-version", 10)
	request.RuleVersion = "CONTRIBUTION_RULE_V2"
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrRuleVersion) {
		t.Fatalf("version error=%v", err)
	}
	request = g13Request("pg-player-a", "pg-fb-b", "cross-owner", 10)
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrInvalidSource) {
		t.Fatalf("cross-owner error=%v", err)
	}
	request = g13Request("pg-player-a", "pg-fb-a", "zero", 0)
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrInvalidAmount) {
		t.Fatalf("zero error=%v", err)
	}
	request = g13Request("pg-player-a", "pg-fb-a", "global-source", 10)
	if _, err := service.PostSystemSpend(ctx, request); err != nil {
		t.Fatal(err)
	}
	request = g13Request("pg-player-b", "pg-fb-b", "global-source", 10)
	if _, err := service.PostSystemSpend(ctx, request); !errors.Is(err, contribution.ErrSourceConflict) {
		t.Fatalf("source conflict=%v", err)
	}
	if got := postgresBalance(t, fb.service, "pg-fb-b"); got != 10_000 {
		t.Fatalf("FB changed=%d", got)
	}
}

func TestG13PostgresMultiplePlayersPostConcurrentlyWithoutLostUpdates(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			player, account := "pg-player-a", "pg-fb-a"
			if i%2 == 1 {
				player, account = "pg-player-b", "pg-fb-b"
			}
			_, err := service.PostSystemSpend(ctx, g13Request(player, account, fmt.Sprintf("two-player-%d", i), 2))
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Errorf("posting: %v", err)
		}
	}
	for _, row := range []struct{ player, account string }{{"pg-player-a", "pg-fb-a"}, {"pg-player-b", "pg-fb-b"}} {
		account, err := service.Account(ctx, row.player)
		if err != nil || account.Balance != 100 || postgresBalance(t, fb.service, row.account) != 9_900 {
			t.Fatalf("player=%s account=%+v err=%v", row.player, account, err)
		}
	}
}

func TestG13PostgresRetriesSerializationWithoutDoubleCredit(t *testing.T) {
	store, service, fb := g13Fixture(t)
	var injected atomic.Int64
	store.contributionFailureInjector = func(point string) error {
		if point == contribution.FailureBeforeCommit && injected.Add(1) == 1 {
			return &pgconn.PgError{Code: "40001", Message: "synthetic serialization conflict"}
		}
		return nil
	}
	result, err := service.PostSystemSpend(context.Background(), g13Request("pg-player-a", "pg-fb-a", "retry-once", 9))
	if err != nil || result.Entry.Amount != 9 || injected.Load() < 2 {
		t.Fatalf("result=%+v attempts=%d err=%v", result, injected.Load(), err)
	}
	entries, _ := service.Entries(context.Background(), "pg-player-a")
	if len(entries) != 1 || postgresBalance(t, fb.service, "pg-fb-a") != 9_991 {
		t.Fatalf("retry entries=%+v", entries)
	}
}

func TestG13PostgresCompetingFBSpendsDoNotCreateExtraContribution(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan error, 40)
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			_, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", fmt.Sprintf("eligible-%d", i), 1))
			results <- err
		}(i)
		go func(i int) {
			defer wg.Done()
			_, err := fb.service.SystemSpend(ctx, "pg-fb-a", "pg-fb-system", 1, ledger.SpendNonEligible, ledger.LedgerReference{Type: "SYSTEM_SPEND", ID: fmt.Sprintf("noneligible-%d", i)})
			results <- err
		}(i)
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Errorf("competing spend: %v", err)
		}
	}
	account, err := service.Account(ctx, "pg-player-a")
	if err != nil || account.Balance != 20 || postgresBalance(t, fb.service, "pg-fb-a") != 9_960 {
		t.Fatalf("account=%+v err=%v", account, err)
	}
}

func TestG13PostgresRejectsForgedEntryWithoutEligibleFBSystemSpend(t *testing.T) {
	store, service, _ := g13Fixture(t)
	ctx := context.Background()
	if _, err := service.Account(ctx, "pg-player-a"); err != nil {
		t.Fatal(err)
	}
	_, err := store.pool.Exec(ctx, `INSERT INTO contribution_entries(entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,balance_before,balance_after,rule_version,fb_transaction_id,created_at) VALUES($1,$2,$3,$4,$5,$6,5,5,0,5,$7,$8,$9)`,
		"forged-entry", "pg-player-a", "pg-fb-a", "pg-fb-system", contribution.SourceSystemService, "forged", contribution.RuleVersionV1, "fixture-transaction-g13-start-a", postgresLedgerNow)
	if err == nil {
		t.Fatal("unrelated FB transaction was accepted as an eligible system spend")
	}
}

func TestG13PostgresRejectsRefundAndReversalWithoutContributionCompensation(t *testing.T) {
	_, service, fb := g13Fixture(t)
	ctx := context.Background()
	posted, err := service.PostSystemSpend(ctx, g13Request("pg-player-a", "pg-fb-a", "spend-no-reversal", 25))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fb.service.Refund(ctx, posted.FBTransaction.ID, ledger.LedgerReference{Type: "REFUND", ID: "g13-refund"}); !errors.Is(err, ledger.ErrInvalidTransaction) {
		t.Fatalf("refund error=%v", err)
	}
	if _, err := fb.service.ReverseTransaction(ctx, posted.FBTransaction.ID, "test reversal", ledger.LedgerReference{Type: "REVERSAL", ID: "g13-reversal"}); !errors.Is(err, ledger.ErrInvalidTransaction) {
		t.Fatalf("reversal error=%v", err)
	}
	account, err := service.Account(ctx, "pg-player-a")
	if err != nil || account.Balance != 25 || postgresBalance(t, fb.service, "pg-fb-a") != 9_975 {
		t.Fatalf("account=%+v err=%v", account, err)
	}
}
