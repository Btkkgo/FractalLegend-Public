package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fractallegend/game-server/internal/ledger"
)

var postgresLedgerNow = time.Date(2026, 9, 20, 15, 0, 0, 0, time.UTC)

const postgresFixtureLiabilityAccountID = "pg-fb-fixture-liability"

type postgresLedgerFixture struct {
	store   *Store
	service *ledger.Service
}

func newPostgresLedgerFixture(t *testing.T) postgresLedgerFixture {
	t.Helper()
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var sequence atomic.Int64
	options := ledger.Options{Now: func() time.Time { return postgresLedgerNow }, NewID: func(prefix string) string {
		return fmt.Sprintf("%s-postgres-%d", prefix, sequence.Add(1))
	}}
	service := ledger.NewService(store, options)
	for _, account := range []struct{ id, owner string }{{"pg-fb-a", "pg-player-a"}, {"pg-fb-b", "pg-player-b"}} {
		if _, err := service.CreatePlayerAccount(ctx, account.id, account.owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.CreateSystemAccount(ctx, "pg-fb-system", "pg-system-spend"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSystemAccount(ctx, postgresFixtureLiabilityAccountID, "pg-test-fixture-liability"); err != nil {
		t.Fatal(err)
	}
	return postgresLedgerFixture{store: store, service: service}
}

func postgresCredit(t *testing.T, f postgresLedgerFixture, account string, amount int64, id string) ledger.LedgerTransaction {
	t.Helper()
	ctx := context.Background()
	tx, err := f.store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var before, liabilityBefore int64
	rows, err := tx.Query(ctx, "SELECT account_id,balance FROM fb_ledger_accounts WHERE account_id=ANY($1) ORDER BY account_id FOR UPDATE", []string{account, postgresFixtureLiabilityAccountID})
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var idValue string
		var balanceValue int64
		if err = rows.Scan(&idValue, &balanceValue); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		if idValue == account {
			before = balanceValue
		} else if idValue == postgresFixtureLiabilityAccountID {
			liabilityBefore = balanceValue
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		t.Fatal(err)
	}
	rows.Close()
	after, err := ledger.AddAmount(before, amount)
	if err != nil {
		t.Fatal(err)
	}
	liabilityAfter, err := ledger.AddAmount(liabilityBefore, -amount)
	if err != nil {
		t.Fatal(err)
	}
	created := postgresLedgerNow
	transactionID, creditEntryID, debitEntryID := "fixture-transaction-"+id, "fixture-entry-credit-"+id, "fixture-entry-debit-"+id
	if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_transactions(transaction_id,transaction_type,status,reference_type,reference_id,spend_classification,reason,created_at) VALUES($1,'PLAYER_TRANSFER','POSTED','TEST_FIXTURE',$2,'','test-only genesis fixture',$3)`, transactionID, id, created); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_entries(entry_id,transaction_id,entry_order,account_id,entry_type,amount,direction,balance_before,balance_after,created_at) VALUES($1,$2,0,$3,'PLAYER_TRANSFER',$4,'CREDIT',$5,$6,$7),($8,$2,1,$9,'PLAYER_TRANSFER',$10,'DEBIT',$11,$12,$7)`, creditEntryID, transactionID, account, amount, before, after, created, debitEntryID, postgresFixtureLiabilityAccountID, -amount, liabilityBefore, liabilityAfter); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "UPDATE fb_ledger_accounts SET balance=$1,revision=revision+1,updated_at=$2 WHERE account_id=$3", after, created, account); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "UPDATE fb_ledger_accounts SET balance=$1,revision=revision+1,updated_at=$2 WHERE account_id=$3", liabilityAfter, created, postgresFixtureLiabilityAccountID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return ledger.LedgerTransaction{ID: transactionID, Type: ledger.TransactionType("TEST_FIXTURE"), Status: ledger.StatusPosted, Reference: ledger.LedgerReference{Type: "TEST_FIXTURE", ID: id}, Entries: []ledger.LedgerEntry{{ID: creditEntryID, TransactionID: transactionID, AccountID: account, EntryType: ledger.TransactionType("TEST_FIXTURE"), Amount: amount, Direction: ledger.DirectionCredit, BalanceBefore: before, BalanceAfter: after, CreatedAt: created}, {ID: debitEntryID, TransactionID: transactionID, AccountID: postgresFixtureLiabilityAccountID, EntryType: ledger.TransactionType("TEST_FIXTURE"), Amount: -amount, Direction: ledger.DirectionDebit, BalanceBefore: liabilityBefore, BalanceAfter: liabilityAfter, CreatedAt: created}}, CreatedAt: created}
}

func postgresBalance(t *testing.T, service *ledger.Service, account string) int64 {
	t.Helper()
	value, err := service.Balance(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestPostgresLedgerPersistsAcrossRestartAndRetryIsIdempotent(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 1000, "restart-credit")
	reference := ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "restart-transfer"}
	first, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 300, reference)
	if err != nil {
		t.Fatal(err)
	}
	restartedStore, err := Open(context.Background(), os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer restartedStore.Close()
	restarted := ledger.NewService(restartedStore, ledger.Options{Now: func() time.Time { return postgresLedgerNow.Add(time.Minute) }})
	second, err := restarted.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 300, reference)
	if err != nil || second.ID != first.ID || postgresBalance(t, restarted, "pg-fb-a") != 700 || postgresBalance(t, restarted, "pg-fb-b") != 300 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	loaded, err := restarted.Transaction(context.Background(), first.ID)
	if err != nil || len(loaded.Entries) != 2 || loaded.Reference != reference {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestPostgresLedgerConcurrentDoubleSpendUsesRowLocks(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 100, "concurrent-credit")
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 2, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: fmt.Sprintf("pg-concurrent-%03d", index)})
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ledger.ErrInsufficientBalance) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 50 || postgresBalance(t, f.service, "pg-fb-a") != 0 || postgresBalance(t, f.service, "pg-fb-b") != 100 {
		t.Fatalf("successes=%d a=%d b=%d", successes.Load(), postgresBalance(t, f.service, "pg-fb-a"), postgresBalance(t, f.service, "pg-fb-b"))
	}
	report, err := f.service.Reconcile(context.Background())
	if err != nil || !report.Balanced {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestPostgresLedgerOppositeTransfersUseDeterministicLockOrder(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 100, "opposite-a")
	postgresCredit(t, f, "pg-fb-b", 100, "opposite-b")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			from, to := "pg-fb-a", "pg-fb-b"
			if index%2 == 1 {
				from, to = to, from
			}
			_, transferErr := f.service.TransferFB(context.Background(), from, to, 1, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: fmt.Sprintf("pg-opposite-%03d", index)})
			if transferErr != nil {
				t.Errorf("opposite transfer %d: %v", index, transferErr)
			}
		}(i)
	}
	wg.Wait()
	if postgresBalance(t, f.service, "pg-fb-a") != 100 || postgresBalance(t, f.service, "pg-fb-b") != 100 {
		t.Fatalf("a=%d b=%d", postgresBalance(t, f.service, "pg-fb-a"), postgresBalance(t, f.service, "pg-fb-b"))
	}
}

func TestPostgresLedgerFailureInjectionRollsBackBothSides(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 100, "rollback-credit")
	f.store.ledgerFailureInjector = func(point string) error {
		if point == ledger.FailureAfterFirstEntry {
			return errors.New("synthetic ledger credit-side failure")
		}
		return nil
	}
	_, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 60, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "rollback-transfer"})
	if err == nil || postgresBalance(t, f.service, "pg-fb-a") != 100 || postgresBalance(t, f.service, "pg-fb-b") != 0 {
		t.Fatalf("err=%v a=%d b=%d", err, postgresBalance(t, f.service, "pg-fb-a"), postgresBalance(t, f.service, "pg-fb-b"))
	}
	var transactionCount, entryCount int
	if err = f.store.pool.QueryRow(context.Background(), "SELECT count(*) FROM fb_ledger_transactions WHERE reference_type='PLAYER_TRANSFER' AND reference_id='rollback-transfer'").Scan(&transactionCount); err != nil {
		t.Fatal(err)
	}
	if err = f.store.pool.QueryRow(context.Background(), "SELECT count(*) FROM fb_ledger_entries WHERE transaction_id IN (SELECT transaction_id FROM fb_ledger_transactions WHERE reference_id='rollback-transfer')").Scan(&entryCount); err != nil {
		t.Fatal(err)
	}
	if transactionCount != 0 || entryCount != 0 {
		t.Fatalf("partial transaction escaped rollback: transactions=%d entries=%d", transactionCount, entryCount)
	}
}

func TestPostgresLedgerEntriesAndTransactionsAreImmutable(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	tx := postgresCredit(t, f, "pg-fb-a", 100, "immutable-credit")
	ctx := context.Background()
	if _, err := f.store.pool.Exec(ctx, "UPDATE fb_ledger_entries SET amount=999 WHERE entry_id=$1", tx.Entries[0].ID); err == nil {
		t.Fatal("immutable entry update succeeded")
	}
	if _, err := f.store.pool.Exec(ctx, "DELETE FROM fb_ledger_entries WHERE entry_id=$1", tx.Entries[0].ID); err == nil {
		t.Fatal("immutable entry delete succeeded")
	}
	if _, err := f.store.pool.Exec(ctx, "DELETE FROM fb_ledger_transactions WHERE transaction_id=$1", tx.ID); err == nil {
		t.Fatal("immutable transaction delete succeeded")
	}
	if postgresBalance(t, f.service, "pg-fb-a") != 100 {
		t.Fatal("immutable mutation changed balance")
	}
}

func TestPostgresLedgerDatabaseRejectsUnbalancedPostingsAtCommit(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	ctx := context.Background()
	tx, err := f.store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_transactions(transaction_id,transaction_type,status,reference_type,reference_id,spend_classification,reason,created_at) VALUES('pg-unbalanced-transaction','PLAYER_TRANSFER','POSTED','PLAYER_TRANSFER','pg-unbalanced','','constraint test',$1)`, postgresLedgerNow); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_entries(entry_id,transaction_id,entry_order,account_id,entry_type,amount,direction,balance_before,balance_after,created_at) VALUES('pg-unbalanced-entry','pg-unbalanced-transaction',0,'pg-fb-a','PLAYER_TRANSFER',1,'CREDIT',0,1,$1)`, postgresLedgerNow); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err == nil {
		t.Fatal("database committed an unbalanced ledger transaction")
	}
	var count int
	if err = f.store.pool.QueryRow(ctx, "SELECT count(*) FROM fb_ledger_transactions WHERE transaction_id='pg-unbalanced-transaction'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("unbalanced transaction survived failed commit: %d", count)
	}
}

func TestPostgresLedgerRefundReversalReconciliationAndSupply(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 1000, "accounting-credit")
	spent, err := f.service.SystemSpend(context.Background(), "pg-fb-a", "pg-fb-system", 100, ledger.SpendEligible, ledger.LedgerReference{Type: "SYSTEM_SPEND", ID: "accounting-spend"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.Refund(context.Background(), spent.ID, ledger.LedgerReference{Type: "REFUND", ID: "accounting-refund"}); err != nil {
		t.Fatal(err)
	}
	transfer, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 300, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "accounting-transfer"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.ReverseTransaction(context.Background(), transfer.ID, "accepted test correction", ledger.LedgerReference{Type: "REVERSAL", ID: "accounting-reversal"}); err != nil {
		t.Fatal(err)
	}
	reconciliation, err := f.service.Reconcile(context.Background())
	if err != nil || !reconciliation.Balanced {
		t.Fatalf("reconciliation=%+v err=%v", reconciliation, err)
	}
	supply, err := f.service.TotalSupply(context.Background())
	if err != nil || supply.AccountTotal != 0 || supply.EntryTotal != 0 {
		t.Fatalf("supply=%+v err=%v", supply, err)
	}
	events, err := f.service.AuditEvents(context.Background(), transfer.ID)
	if err != nil || len(events) != 1 || events[0].Type != "FB_TRANSFER_COMPLETED" || events[0].Result != "POSTED" {
		t.Fatalf("events=%+v err=%v", events, err)
	}
}

func TestPostgresLedgerReversalFailsWhenDestinationFundsAreGone(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 100, "reversal-insufficient")
	original, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 80, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "pg-reversal-original"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.TransferFB(context.Background(), "pg-fb-b", "pg-fb-a", 50, ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "pg-reversal-spend"}); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.ReverseTransaction(context.Background(), original.ID, "destination cannot repay", ledger.LedgerReference{Type: "REVERSAL", ID: "pg-reversal-insufficient"})
	if !errors.Is(err, ledger.ErrInsufficientBalance) || postgresBalance(t, f.service, "pg-fb-a") != 70 || postgresBalance(t, f.service, "pg-fb-b") != 30 {
		t.Fatalf("err=%v a=%d b=%d", err, postgresBalance(t, f.service, "pg-fb-a"), postgresBalance(t, f.service, "pg-fb-b"))
	}
}

func TestPostgresLedgerDatabaseConstraintsProtectOwnerAndReferenceUniqueness(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	if _, err := f.service.CreatePlayerAccount(context.Background(), "pg-fb-a-duplicate", "pg-player-a"); !errors.Is(err, ledger.ErrConflict) {
		t.Fatalf("duplicate owner error=%v", err)
	}
	postgresCredit(t, f, "pg-fb-a", 100, "reference-collision-seed")
	reference := ledger.LedgerReference{Type: "PLAYER_TRANSFER", ID: "pg-reference-collision"}
	if _, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 10, reference); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.TransferFB(context.Background(), "pg-fb-a", "pg-fb-b", 11, reference); !errors.Is(err, ledger.ErrReferenceConflict) {
		t.Fatalf("reference collision error=%v", err)
	}
	spent, err := f.service.SystemSpend(context.Background(), "pg-fb-a", "pg-fb-system", 10, ledger.SpendEligible, ledger.LedgerReference{Type: "SYSTEM_SPEND", ID: "pg-single-compensation-spend"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.Refund(context.Background(), spent.ID, ledger.LedgerReference{Type: "REFUND", ID: "pg-single-compensation-refund"}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.ReverseTransaction(context.Background(), spent.ID, "must not compensate twice", ledger.LedgerReference{Type: "REVERSAL", ID: "pg-single-compensation-reversal"}); !errors.Is(err, ledger.ErrAlreadyCompensated) {
		t.Fatalf("cross compensation error=%v", err)
	}
}

func TestPostgresLedgerReconciliationDetectsMismatchWithoutRepair(t *testing.T) {
	f := newPostgresLedgerFixture(t)
	postgresCredit(t, f, "pg-fb-a", 100, "reconciliation-mismatch")
	if _, err := f.store.pool.Exec(context.Background(), "UPDATE fb_ledger_accounts SET balance=99 WHERE account_id='pg-fb-a'"); err != nil {
		t.Fatal(err)
	}
	report, err := f.service.Reconcile(context.Background())
	if err != nil || report.Balanced || len(report.Mismatches) != 1 || report.Mismatches[0].Balance != 99 || report.Mismatches[0].EntriesTotal != 100 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if postgresBalance(t, f.service, "pg-fb-a") != 99 {
		t.Fatal("reconciliation silently repaired PostgreSQL state")
	}
}
