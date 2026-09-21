package ledger

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 9, 20, 14, 0, 0, 0, time.UTC)

const fixtureLiabilityAccountID = "fb-test-fixture-liability"

type fixture struct {
	repo    *MemoryRepository
	service *Service
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	repo := NewMemoryRepository()
	var sequence atomic.Int64
	options := Options{
		Now: fixedClock,
		NewID: func(prefix string) string {
			return prefix + "-test-" + string(rune('a'+sequence.Add(1)))
		},
	}
	service := NewService(repo, options)
	for _, account := range []struct{ id, owner string }{{"fb-a", "player-a"}, {"fb-b", "player-b"}} {
		if _, err := service.CreatePlayerAccount(context.Background(), account.id, account.owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.CreateSystemAccount(context.Background(), "fb-system-spend", "system-spend"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateSystemAccount(context.Background(), fixtureLiabilityAccountID, "test-fixture-liability"); err != nil {
		t.Fatal(err)
	}
	return fixture{repo: repo, service: service}
}

func fixedClock() time.Time { return fixedNow }

func ref(kind, id string) LedgerReference { return LedgerReference{Type: kind, ID: id} }

func credit(t *testing.T, f fixture, account string, amount int64, id string) LedgerTransaction {
	t.Helper()
	tx, err := seedFixtureBalance(f.repo, account, amount, id)
	if err != nil {
		t.Fatal(err)
	}
	return tx
}

// seedFixtureBalance is test-only genesis data. It bypasses the production
// Repository and Service surfaces so no callable funding path ships in G11.
func seedFixtureBalance(repo *MemoryRepository, accountID string, amount int64, id string) (LedgerTransaction, error) {
	if amount <= 0 {
		return LedgerTransaction{}, ErrInvalidAmount
	}
	repo.mu.Lock()
	defer repo.mu.Unlock()
	account, ok := repo.accounts[accountID]
	if !ok {
		return LedgerTransaction{}, ErrNotFound
	}
	next, err := AddAmount(account.Balance, amount)
	if err != nil {
		return LedgerTransaction{}, err
	}
	liability, ok := repo.accounts[fixtureLiabilityAccountID]
	if !ok || liability.OwnerType != OwnerSystem {
		return LedgerTransaction{}, ErrNotFound
	}
	liabilityNext, err := AddAmount(liability.Balance, -amount)
	if err != nil {
		return LedgerTransaction{}, err
	}
	now := fixedClock()
	txID := "fixture-transaction-" + id
	entry := LedgerEntry{ID: "fixture-entry-credit-" + id, TransactionID: txID, AccountID: accountID, EntryType: TransactionType("TEST_FIXTURE"), Amount: amount, Direction: DirectionCredit, BalanceBefore: account.Balance, BalanceAfter: next, CreatedAt: now}
	offset := LedgerEntry{ID: "fixture-entry-debit-" + id, TransactionID: txID, AccountID: fixtureLiabilityAccountID, EntryType: TransactionType("TEST_FIXTURE"), Amount: -amount, Direction: DirectionDebit, BalanceBefore: liability.Balance, BalanceAfter: liabilityNext, CreatedAt: now}
	tx := LedgerTransaction{ID: txID, Type: TransactionType("TEST_FIXTURE"), Status: StatusPosted, Reference: ref("TEST_FIXTURE", id), Entries: []LedgerEntry{entry, offset}, CreatedAt: now}
	account.Balance = next
	account.Revision++
	account.UpdatedAt = now
	repo.accounts[accountID] = account
	liability.Balance = liabilityNext
	liability.Revision++
	liability.UpdatedAt = now
	repo.accounts[fixtureLiabilityAccountID] = liability
	repo.transactions[txID] = cloneTransaction(tx)
	repo.references[referenceKey(tx.Reference)] = txID
	return tx, nil
}

func balance(t *testing.T, f fixture, account string) int64 {
	t.Helper()
	value, err := f.service.Balance(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestCreateAccountStartsAtZero(t *testing.T) {
	f := newFixture(t)
	account, err := f.service.CreatePlayerAccount(context.Background(), "fb-c", "player-c")
	if err != nil || account.Balance != 0 || account.Revision != 1 || account.OwnerType != OwnerPlayer {
		t.Fatalf("account=%+v err=%v", account, err)
	}
	currency := reflect.ValueOf(account).FieldByName("Currency")
	if !currency.IsValid() || currency.String() != "FB" {
		t.Fatalf("account currency is not fixed to FB: %+v", account)
	}
}

func TestTestOnlyFixtureSeedsIntegerBaseUnits(t *testing.T) {
	f := newFixture(t)
	tx := credit(t, f, "fb-a", 1000, "credit-1000")
	if balance(t, f, "fb-a") != 1000 || tx.Type != TransactionType("TEST_FIXTURE") || len(tx.Entries) != 2 || signedSum(tx.Entries) != 0 || tx.Entries[0].Amount != 1000 {
		t.Fatalf("balance=%d tx=%+v", balance(t, f, "fb-a"), tx)
	}
}

func TestInternalTransferConservesSupply(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 1000, "transfer-seed")
	tx, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 300, ref("PLAYER_TRANSFER", "transfer-300"))
	if err != nil {
		t.Fatal(err)
	}
	if balance(t, f, "fb-a") != 700 || balance(t, f, "fb-b") != 300 || signedSum(tx.Entries) != 0 {
		t.Fatalf("a=%d b=%d entries=%+v", balance(t, f, "fb-a"), balance(t, f, "fb-b"), tx.Entries)
	}
	report, err := f.service.TotalSupply(context.Background())
	if err != nil || report.AccountTotal != 0 || report.EntryTotal != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestInsufficientBalanceLeavesBothAccountsUnchanged(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "insufficient-seed")
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 101, ref("PLAYER_TRANSFER", "too-much"))
	if !errors.Is(err, ErrInsufficientBalance) || balance(t, f, "fb-a") != 100 || balance(t, f, "fb-b") != 0 {
		t.Fatalf("err=%v a=%d b=%d", err, balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestZeroAmountIsRejected(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 0, ref("PLAYER_TRANSFER", "zero"))
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("err=%v", err)
	}
}

func TestNegativeAmountIsRejected(t *testing.T) {
	f := newFixture(t)
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", -1, ref("PLAYER_TRANSFER", "negative"))
	if !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("err=%v", err)
	}
}

func TestSelfTransferIsRejectedWithoutEntries(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "self-seed")
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-a", 20, ref("PLAYER_TRANSFER", "self"))
	if !errors.Is(err, ErrInvalidTransfer) || balance(t, f, "fb-a") != 100 {
		t.Fatalf("err=%v balance=%d", err, balance(t, f, "fb-a"))
	}
}

func TestDuplicateReferenceIsIdempotent(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "duplicate-seed")
	want := ref("PLAYER_TRANSFER", "duplicate")
	first, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 40, want)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 40, want)
	if err != nil || first.ID != second.ID || balance(t, f, "fb-a") != 60 || balance(t, f, "fb-b") != 40 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestReferenceCollisionWithDifferentIntentFailsClosed(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "collision-seed")
	want := ref("PLAYER_TRANSFER", "collision")
	if _, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 40, want); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 41, want); !errors.Is(err, ErrReferenceConflict) {
		t.Fatalf("err=%v", err)
	}
}

func TestConcurrentDoubleSpendAllowsOnlyOneDebit(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "double-spend-seed")
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 80, ref("PLAYER_TRANSFER", "double-spend-"+string(rune('a'+index))))
			if err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrInsufficientBalance) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 1 || balance(t, f, "fb-a") != 20 || balance(t, f, "fb-b") != 80 {
		t.Fatalf("success=%d a=%d b=%d", successes.Load(), balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestAtomicFailureRollsBackPreparedDebit(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "rollback-seed")
	f.repo.InjectFailure(FailureAfterFirstEntry, errors.New("synthetic credit-side failure"))
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 60, ref("PLAYER_TRANSFER", "rollback"))
	if err == nil || balance(t, f, "fb-a") != 100 || balance(t, f, "fb-b") != 0 {
		t.Fatalf("err=%v a=%d b=%d", err, balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestRestartRetryReturnsPersistedTransaction(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "restart-seed")
	want := ref("PLAYER_TRANSFER", "restart")
	first, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 30, want)
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewService(f.repo, Options{Now: fixedClock})
	second, err := restarted.TransferFB(context.Background(), "fb-a", "fb-b", 30, want)
	if err != nil || first.ID != second.ID || balance(t, f, "fb-a") != 70 || balance(t, f, "fb-b") != 30 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestPostedHistoryIsImmutable(t *testing.T) {
	f := newFixture(t)
	tx := credit(t, f, "fb-a", 100, "immutable")
	tx.Entries[0].Amount = 999999
	loaded, err := f.service.Transaction(context.Background(), tx.ID)
	if err != nil || loaded.Entries[0].Amount != 100 || balance(t, f, "fb-a") != 100 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestSystemSpendRefundCreatesCompensatingEntries(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "refund-seed")
	spent, err := f.service.SystemSpend(context.Background(), "fb-a", "fb-system-spend", 25, SpendEligible, ref("SYSTEM_SPEND", "spend-1"))
	if err != nil {
		t.Fatal(err)
	}
	refunded, err := f.service.Refund(context.Background(), spent.ID, ref("REFUND", "refund-1"))
	if err != nil || refunded.OriginalTransactionID != spent.ID || balance(t, f, "fb-a") != 100 || len(refunded.Entries) != 2 {
		t.Fatalf("refund=%+v err=%v", refunded, err)
	}
	loaded, err := f.service.Transaction(context.Background(), spent.ID)
	if err != nil || loaded.ID != spent.ID || loaded.Type != TransactionSystemSpend {
		t.Fatalf("original=%+v err=%v", loaded, err)
	}
}

func TestDuplicateRefundReturnsExistingResult(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "duplicate-refund-seed")
	spent, _ := f.service.SystemSpend(context.Background(), "fb-a", "fb-system-spend", 25, SpendNonEligible, ref("SYSTEM_SPEND", "spend-duplicate-refund"))
	want := ref("REFUND", "duplicate-refund")
	first, err := f.service.Refund(context.Background(), spent.ID, want)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.service.Refund(context.Background(), spent.ID, want)
	if err != nil || first.ID != second.ID || balance(t, f, "fb-a") != 100 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestSecondRefundWithDifferentReferenceIsRejected(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "second-refund-seed")
	spent, _ := f.service.SystemSpend(context.Background(), "fb-a", "fb-system-spend", 25, SpendEligible, ref("SYSTEM_SPEND", "spend-second-refund"))
	if _, err := f.service.Refund(context.Background(), spent.ID, ref("REFUND", "refund-first")); err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Refund(context.Background(), spent.ID, ref("REFUND", "refund-second")); !errors.Is(err, ErrAlreadyCompensated) {
		t.Fatalf("err=%v", err)
	}
}

func TestRefundedTransactionCannotAlsoBeReversed(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "cross-compensation-seed")
	spent, err := f.service.SystemSpend(context.Background(), "fb-a", "fb-system-spend", 25, SpendEligible, ref("SYSTEM_SPEND", "cross-compensation-spend"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.Refund(context.Background(), spent.ID, ref("REFUND", "cross-compensation-refund")); err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.ReverseTransaction(context.Background(), spent.ID, "must not compensate twice", ref("REVERSAL", "cross-compensation-reversal")); !errors.Is(err, ErrAlreadyCompensated) {
		t.Fatalf("cross compensation error=%v", err)
	}
}

func TestReversalCreatesOppositeEntries(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "reversal-seed")
	transfer, _ := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 30, ref("PLAYER_TRANSFER", "before-reversal"))
	reversal, err := f.service.ReverseTransaction(context.Background(), transfer.ID, "operator correction", ref("REVERSAL", "reverse-transfer"))
	if err != nil || reversal.OriginalTransactionID != transfer.ID || balance(t, f, "fb-a") != 100 || balance(t, f, "fb-b") != 0 || signedSum(reversal.Entries) != 0 {
		t.Fatalf("reversal=%+v err=%v", reversal, err)
	}
}

func TestDuplicateReversalCannotCompensateTwice(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "duplicate-reversal-seed")
	transfer, _ := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 30, ref("PLAYER_TRANSFER", "duplicate-reversal-transfer"))
	want := ref("REVERSAL", "duplicate-reversal")
	first, err := f.service.ReverseTransaction(context.Background(), transfer.ID, "operator correction", want)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.service.ReverseTransaction(context.Background(), transfer.ID, "operator correction", want)
	if err != nil || first.ID != second.ID {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	if _, err = f.service.ReverseTransaction(context.Background(), transfer.ID, "another correction", ref("REVERSAL", "different-reference")); !errors.Is(err, ErrAlreadyCompensated) {
		t.Fatalf("err=%v", err)
	}
}

func TestReversalFailsWhenDestinationCannotReturnFunds(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "reversal-insufficient-seed")
	transfer, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 80, ref("PLAYER_TRANSFER", "reversal-insufficient-transfer"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.service.TransferFB(context.Background(), "fb-b", "fb-a", 50, ref("PLAYER_TRANSFER", "spend-before-reversal")); err != nil {
		t.Fatal(err)
	}
	_, err = f.service.ReverseTransaction(context.Background(), transfer.ID, "insufficient destination balance", ref("REVERSAL", "reversal-insufficient"))
	if !errors.Is(err, ErrInsufficientBalance) || balance(t, f, "fb-a") != 70 || balance(t, f, "fb-b") != 30 {
		t.Fatalf("err=%v a=%d b=%d", err, balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestManyInternalTransfersPreserveSupply(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 1000, "conservation-seed")
	for i := 0; i < 500; i++ {
		from, to := "fb-a", "fb-b"
		if i%2 == 1 {
			from, to = to, from
		}
		if _, err := f.service.TransferFB(context.Background(), from, to, 1, ref("PLAYER_TRANSFER", "conservation-"+time.Unix(int64(i), 0).UTC().Format("150405"))); err != nil {
			t.Fatal(err)
		}
	}
	report, err := f.service.TotalSupply(context.Background())
	if err != nil || report.AccountTotal != 0 || report.EntryTotal != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestReconciliationMatchesBalancesAndEntries(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "reconcile-seed")
	_, _ = f.service.TransferFB(context.Background(), "fb-a", "fb-b", 30, ref("PLAYER_TRANSFER", "reconcile-transfer"))
	report, err := f.service.Reconcile(context.Background())
	if err != nil || !report.Balanced || len(report.Mismatches) != 0 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
}

func TestReconciliationDetectsMismatchWithoutRepair(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "reconcile-mismatch-seed")
	f.repo.mu.Lock()
	account := f.repo.accounts["fb-a"]
	account.Balance = 99
	f.repo.accounts["fb-a"] = account
	f.repo.mu.Unlock()
	report, err := f.service.Reconcile(context.Background())
	if err != nil || report.Balanced || len(report.Mismatches) != 1 || report.Mismatches[0].Balance != 99 || report.Mismatches[0].EntriesTotal != 100 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	if balance(t, f, "fb-a") != 99 {
		t.Fatal("reconciliation silently repaired the mismatch")
	}
}

func TestRepositoryReloadPreservesAccountsTransactionsEntriesAndReferences(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "reload-seed")
	want := ref("PLAYER_TRANSFER", "reload-transfer")
	first, _ := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 30, want)
	reloaded := NewService(f.repo, Options{Now: fixedClock})
	loaded, err := reloaded.Transaction(context.Background(), first.ID)
	retry, retryErr := reloaded.TransferFB(context.Background(), "fb-a", "fb-b", 30, want)
	if err != nil || retryErr != nil || loaded.ID != first.ID || retry.ID != first.ID || len(loaded.Entries) != 2 {
		t.Fatalf("loaded=%+v retry=%+v err=%v retryErr=%v", loaded, retry, err, retryErr)
	}
}

func TestOverflowProtectionRejectsCredit(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", math.MaxInt64, "max-credit")
	_, err := seedFixtureBalance(f.repo, "fb-a", 1, "overflow")
	if !errors.Is(err, ErrAmountOverflow) || balance(t, f, "fb-a") != math.MaxInt64 {
		t.Fatalf("err=%v balance=%d", err, balance(t, f, "fb-a"))
	}
}

func TestTransferOverflowRollsBackBothAccounts(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 1, "overflow-source")
	credit(t, f, "fb-b", math.MaxInt64, "overflow-destination")
	_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 1, ref("PLAYER_TRANSFER", "overflow-transfer"))
	if !errors.Is(err, ErrAmountOverflow) || balance(t, f, "fb-a") != 1 || balance(t, f, "fb-b") != math.MaxInt64 {
		t.Fatalf("err=%v a=%d b=%d", err, balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestDeterministicLedgerInvariantSequence(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 5000, "property-seed")
	rng := rand.New(rand.NewSource(110011))
	for i := 0; i < 1000; i++ {
		from, to := "fb-a", "fb-b"
		if rng.Intn(2) == 1 {
			from, to = to, from
		}
		amount := int64(rng.Intn(5) + 1)
		if balance(t, f, from) < amount {
			from, to = to, from
		}
		_, err := f.service.TransferFB(context.Background(), from, to, amount, ref("PLAYER_TRANSFER", "property-"+time.Unix(int64(i), 0).UTC().Format("150405")))
		if err != nil {
			t.Fatal(err)
		}
	}
	reconciliation, err := f.service.Reconcile(context.Background())
	if err != nil || !reconciliation.Balanced {
		t.Fatalf("reconciliation=%+v err=%v", reconciliation, err)
	}
	supply, err := f.service.TotalSupply(context.Background())
	if err != nil || supply.AccountTotal != 0 || supply.EntryTotal != 0 || balance(t, f, "fb-a") < 0 || balance(t, f, "fb-b") < 0 {
		t.Fatalf("supply=%+v err=%v", supply, err)
	}
}

func TestOneHundredConcurrentTransfersCannotOverspend(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "high-concurrency-seed")
	var success atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			_, err := f.service.TransferFB(context.Background(), "fb-a", "fb-b", 2, ref("PLAYER_TRANSFER", "concurrent-"+time.Unix(int64(index), 0).UTC().Format("150405")))
			if err == nil {
				success.Add(1)
			} else if !errors.Is(err, ErrInsufficientBalance) {
				t.Errorf("unexpected error: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if success.Load() != 50 || balance(t, f, "fb-a") != 0 || balance(t, f, "fb-b") != 100 {
		t.Fatalf("success=%d a=%d b=%d", success.Load(), balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
	reconciliation, err := f.service.Reconcile(context.Background())
	if err != nil || !reconciliation.Balanced {
		t.Fatalf("reconciliation=%+v err=%v", reconciliation, err)
	}
}

func TestOppositeDirectionConcurrentTransfersCompleteWithoutDeadlock(t *testing.T) {
	f := newFixture(t)
	credit(t, f, "fb-a", 100, "opposite-a")
	credit(t, f, "fb-b", 100, "opposite-b")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			from, to := "fb-a", "fb-b"
			if index%2 == 1 {
				from, to = to, from
			}
			if _, err := f.service.TransferFB(context.Background(), from, to, 1, ref("PLAYER_TRANSFER", "opposite-"+time.Unix(int64(index), 0).UTC().Format("150405"))); err != nil {
				t.Errorf("opposite transfer %d: %v", index, err)
			}
		}(i)
	}
	wg.Wait()
	if balance(t, f, "fb-a") != 100 || balance(t, f, "fb-b") != 100 {
		t.Fatalf("a=%d b=%d", balance(t, f, "fb-a"), balance(t, f, "fb-b"))
	}
}

func TestLedgerLifecycleEventsExposeCreatedCompletedFailedReversedAndMismatch(t *testing.T) {
	repo := NewMemoryRepository()
	var events []AuditEvent
	var sequence atomic.Int64
	service := NewService(repo, Options{
		Now: fixedClock,
		NewID: func(prefix string) string {
			return fmt.Sprintf("%s-events-%d", prefix, sequence.Add(1))
		},
		Emit: func(event AuditEvent) { events = append(events, event) },
	})
	for _, value := range []struct{ account, owner string }{{"events-a", "events-player-a"}, {"events-b", "events-player-b"}} {
		if _, err := service.CreatePlayerAccount(context.Background(), value.account, value.owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.CreateSystemAccount(context.Background(), fixtureLiabilityAccountID, "events-fixture-liability"); err != nil {
		t.Fatal(err)
	}
	if _, err := seedFixtureBalance(repo, "events-a", 10, "events-seed"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.TransferFB(context.Background(), "events-a", "events-b", 11, ref("PLAYER_TRANSFER", "events-failed")); !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("failure error=%v", err)
	}
	transfer, err := service.TransferFB(context.Background(), "events-a", "events-b", 5, ref("PLAYER_TRANSFER", "events-completed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.ReverseTransaction(context.Background(), transfer.ID, "event coverage", ref("REVERSAL", "events-reversed")); err != nil {
		t.Fatal(err)
	}
	repo.mu.Lock()
	account := repo.accounts["events-a"]
	account.Balance--
	repo.accounts["events-a"] = account
	repo.mu.Unlock()
	if _, err = service.Reconcile(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"FB_TRANSACTION_CREATED": false, "FB_TRANSACTION_COMPLETED": false, "FB_TRANSACTION_FAILED": false, "FB_TRANSACTION_REVERSED": false, "FB_RECONCILIATION_MISMATCH": false}
	for _, event := range events {
		if _, ok := want[event.Type]; ok {
			want[event.Type] = true
		}
	}
	for kind, found := range want {
		if !found {
			t.Errorf("missing lifecycle event %s: %+v", kind, events)
		}
	}
}

func signedSum(entries []LedgerEntry) int64 {
	var total int64
	for _, entry := range entries {
		total += entry.Amount
	}
	return total
}
