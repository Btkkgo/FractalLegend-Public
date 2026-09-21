package ledger

import (
	"context"
	"math"
	"sort"
	"sync"
)

const FailureAfterFirstEntry = "after_first_entry"

type MemoryRepository struct {
	mu           sync.Mutex
	accounts     map[string]LedgerAccount
	owners       map[string]string
	transactions map[string]LedgerTransaction
	references   map[string]string
	compensated  map[string]string
	audit        map[string][]AuditEvent
	failures     map[string]error
	sequence     int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		accounts:     map[string]LedgerAccount{},
		owners:       map[string]string{},
		transactions: map[string]LedgerTransaction{},
		references:   map[string]string{},
		compensated:  map[string]string{},
		audit:        map[string][]AuditEvent{},
		failures:     map[string]error{},
	}
}

func (r *MemoryRepository) CreateLedgerAccount(ctx context.Context, value LedgerAccount) (LedgerAccount, error) {
	if err := ctx.Err(); err != nil {
		return LedgerAccount{}, err
	}
	if value.ID == "" || value.OwnerID == "" || (value.OwnerType != OwnerPlayer && value.OwnerType != OwnerSystem) || value.Currency != CurrencyFB || value.Balance != 0 {
		return LedgerAccount{}, ErrInvalidAccount
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.accounts[value.ID]; exists {
		return LedgerAccount{}, ErrConflict
	}
	ownerKey := string(value.OwnerType) + "\x00" + value.OwnerID
	if _, exists := r.owners[ownerKey]; exists {
		return LedgerAccount{}, ErrConflict
	}
	value.Revision = 1
	r.accounts[value.ID] = value
	r.owners[ownerKey] = value.ID
	return value, nil
}

func (r *MemoryRepository) LoadLedgerAccount(ctx context.Context, id string) (LedgerAccount, error) {
	if err := ctx.Err(); err != nil {
		return LedgerAccount{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.accounts[id]
	if !ok {
		return LedgerAccount{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) PostLedgerTransaction(ctx context.Context, draft PostDraft) (LedgerTransaction, error) {
	if err := ctx.Err(); err != nil {
		return LedgerTransaction{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existingID, ok := r.references[referenceKey(draft.Reference)]; ok {
		existing := r.transactions[existingID]
		if !sameIntent(existing, draft) {
			return LedgerTransaction{}, ErrReferenceConflict
		}
		return cloneTransaction(existing), nil
	}
	if err := ValidatePostDraft(draft); err != nil {
		return LedgerTransaction{}, err
	}
	if draft.OriginalTransactionID != "" {
		if _, ok := r.transactions[draft.OriginalTransactionID]; !ok {
			return LedgerTransaction{}, ErrNotFound
		}
		key := draft.OriginalTransactionID
		if _, exists := r.compensated[key]; exists {
			return LedgerTransaction{}, ErrAlreadyCompensated
		}
	}
	balances := make(map[string]int64, len(draft.Entries))
	for _, entry := range draft.Entries {
		account, ok := r.accounts[entry.AccountID]
		if !ok {
			return LedgerTransaction{}, ErrNotFound
		}
		next, err := AddAmount(account.Balance, entry.Amount)
		if err != nil {
			return LedgerTransaction{}, err
		}
		if next < 0 {
			return LedgerTransaction{}, ErrInsufficientBalance
		}
		balances[entry.AccountID] = next
	}
	if failure := r.failures[FailureAfterFirstEntry]; failure != nil {
		delete(r.failures, FailureAfterFirstEntry)
		return LedgerTransaction{}, failure
	}
	tx := LedgerTransaction{
		ID: draft.ID, Type: draft.Type, Status: StatusPosted, Reference: draft.Reference,
		OriginalTransactionID: draft.OriginalTransactionID,
		SpendClassification:   draft.SpendClassification, Reason: draft.Reason, CreatedAt: draft.CreatedAt,
	}
	for _, item := range draft.Entries {
		account := r.accounts[item.AccountID]
		entry := LedgerEntry{ID: item.ID, TransactionID: draft.ID, AccountID: item.AccountID, EntryType: draft.Type, Amount: item.Amount, Direction: direction(item.Amount), BalanceBefore: account.Balance, BalanceAfter: balances[item.AccountID], CreatedAt: draft.CreatedAt}
		account.Balance = entry.BalanceAfter
		account.Revision++
		account.UpdatedAt = draft.CreatedAt
		r.accounts[item.AccountID] = account
		tx.Entries = append(tx.Entries, entry)
	}
	r.transactions[tx.ID] = cloneTransaction(tx)
	r.references[referenceKey(tx.Reference)] = tx.ID
	if tx.OriginalTransactionID != "" {
		r.compensated[tx.OriginalTransactionID] = tx.ID
	}
	r.sequence++
	r.audit[tx.ID] = append(r.audit[tx.ID], AuditEvent{Sequence: r.sequence, TransactionID: tx.ID, Reference: tx.Reference, AccountIDs: accountIDs(tx.Entries), Amount: absoluteMagnitude(tx.Entries), Type: eventType(tx.Type), Result: "POSTED", OccurredAt: tx.CreatedAt})
	return cloneTransaction(tx), nil
}

func (r *MemoryRepository) LoadLedgerTransaction(ctx context.Context, id string) (LedgerTransaction, error) {
	if err := ctx.Err(); err != nil {
		return LedgerTransaction{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.transactions[id]
	if !ok {
		return LedgerTransaction{}, ErrNotFound
	}
	return cloneTransaction(value), nil
}

func (r *MemoryRepository) ReconcileLedger(ctx context.Context) (ReconciliationReport, error) {
	if err := ctx.Err(); err != nil {
		return ReconciliationReport{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	totals := map[string]int64{}
	for _, tx := range r.transactions {
		for _, entry := range tx.Entries {
			totals[entry.AccountID] += entry.Amount
		}
	}
	report := ReconciliationReport{Balanced: true}
	ids := make([]string, 0, len(r.accounts))
	for id := range r.accounts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if r.accounts[id].Balance != totals[id] {
			report.Balanced = false
			report.Mismatches = append(report.Mismatches, ReconciliationMismatch{AccountID: id, Balance: r.accounts[id].Balance, EntriesTotal: totals[id]})
		}
	}
	return report, nil
}

func (r *MemoryRepository) LedgerTotalSupply(ctx context.Context) (TotalSupplyReport, error) {
	if err := ctx.Err(); err != nil {
		return TotalSupplyReport{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var report TotalSupplyReport
	for _, account := range r.accounts {
		var err error
		report.AccountTotal, err = AddAmount(report.AccountTotal, account.Balance)
		if err != nil {
			return TotalSupplyReport{}, err
		}
	}
	for _, tx := range r.transactions {
		var delta int64
		for _, entry := range tx.Entries {
			var err error
			delta, err = AddAmount(delta, entry.Amount)
			if err != nil {
				return TotalSupplyReport{}, err
			}
		}
		var err error
		report.EntryTotal, err = AddAmount(report.EntryTotal, delta)
		if err != nil {
			return TotalSupplyReport{}, err
		}
	}
	return report, nil
}

func (r *MemoryRepository) LedgerAuditEvents(ctx context.Context, transactionID string) ([]AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.transactions[transactionID]; !ok {
		return nil, ErrNotFound
	}
	result := append([]AuditEvent(nil), r.audit[transactionID]...)
	for i := range result {
		result[i].AccountIDs = append([]string(nil), result[i].AccountIDs...)
	}
	return result, nil
}

func (r *MemoryRepository) InjectFailure(point string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures[point] = err
}

func ValidatePostDraft(draft PostDraft) error {
	if draft.ID == "" || draft.Reference.Type == "" || draft.Reference.ID == "" {
		return ErrInvalidReference
	}
	if len(draft.Entries) == 0 {
		return ErrInvalidTransaction
	}
	seen := map[string]struct{}{}
	var sum int64
	for _, entry := range draft.Entries {
		if entry.ID == "" || entry.AccountID == "" || entry.Amount == 0 {
			return ErrInvalidTransaction
		}
		if _, exists := seen[entry.AccountID]; exists {
			return ErrInvalidTransaction
		}
		seen[entry.AccountID] = struct{}{}
		var err error
		sum, err = AddAmount(sum, entry.Amount)
		if err != nil {
			return err
		}
	}
	switch draft.Type {
	case TransactionPlayerTransfer, TransactionSystemSpend, TransactionRefund:
		if len(draft.Entries) != 2 || sum != 0 {
			return ErrUnbalanced
		}
	case TransactionReversal:
		if len(draft.Entries) < 2 || sum != 0 || draft.OriginalTransactionID == "" || draft.Reason == "" {
			return ErrInvalidTransaction
		}
	default:
		return ErrInvalidTransaction
	}
	return nil
}

func AddAmount(left, right int64) (int64, error) {
	if (right > 0 && left > math.MaxInt64-right) || (right < 0 && left < math.MinInt64-right) {
		return 0, ErrAmountOverflow
	}
	return left + right, nil
}

func sameIntent(existing LedgerTransaction, draft PostDraft) bool {
	if existing.Type != draft.Type || existing.Reference != draft.Reference || existing.OriginalTransactionID != draft.OriginalTransactionID || existing.SpendClassification != draft.SpendClassification || existing.Reason != draft.Reason || len(existing.Entries) != len(draft.Entries) {
		return false
	}
	for i := range draft.Entries {
		if existing.Entries[i].AccountID != draft.Entries[i].AccountID || existing.Entries[i].Amount != draft.Entries[i].Amount {
			return false
		}
	}
	return true
}

func referenceKey(value LedgerReference) string { return value.Type + "\x00" + value.ID }

func direction(amount int64) Direction {
	if amount < 0 {
		return DirectionDebit
	}
	return DirectionCredit
}

func cloneTransaction(value LedgerTransaction) LedgerTransaction {
	value.Entries = append([]LedgerEntry(nil), value.Entries...)
	return value
}

func accountIDs(entries []LedgerEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.AccountID)
	}
	return result
}

func absoluteMagnitude(entries []LedgerEntry) int64 {
	var amount int64
	for _, entry := range entries {
		if entry.Amount > amount {
			amount = entry.Amount
		}
	}
	if amount != 0 {
		return amount
	}
	for _, entry := range entries {
		if entry.Amount == math.MinInt64 {
			return math.MaxInt64
		}
		if -entry.Amount > amount {
			amount = -entry.Amount
		}
	}
	return amount
}

func eventType(value TransactionType) string {
	switch value {
	case TransactionPlayerTransfer:
		return "FB_TRANSFER_COMPLETED"
	case TransactionRefund:
		return "FB_REFUND"
	case TransactionReversal:
		return "FB_REVERSAL"
	default:
		return "FB_LEDGER_POSTED"
	}
}
