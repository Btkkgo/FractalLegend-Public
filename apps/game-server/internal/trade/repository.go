package trade

import (
	"context"
	"errors"
	"sort"
	"sync"

	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/persistence"
)

const (
	FailureAfterFirstCharacterWrite = "after_first_character_write"
	FailureAfterItemOwnershipUpdate = "after_item_ownership_update"
	FailureAfterLedgerWrite         = "after_ledger_write"
	FailureDuringBalanceUpdate      = "during_balance_update"
	FailureBeforeCompletedState     = "before_completed_state"
	FailureBeforeCommit             = "before_commit"
)

type Transaction interface {
	Session() *Session
	LoadCharacter(string) (persistence.CharacterAggregate, error)
	SaveCharacter(persistence.CharacterAggregate) error
	LockItem(ItemLock) error
	ValidateItemLock(ItemLock) error
	UnlockItem(string) error
	UnlockAll() error
	AppendAudit(AuditEvent) error
	PrepareSettlement([]string, []string, []string) error
	LoadLedgerAccountByOwner(string) (ledger.LedgerAccount, error)
	PostLedgerTransaction(ledger.PostDraft) (ledger.LedgerTransaction, error)
	CheckFailure(string) error
}

type Repository interface {
	CreateTrade(context.Context, Session, AuditEvent) error
	LoadTrade(context.Context, string) (Session, error)
	Transact(context.Context, string, func(Transaction) error) (Session, error)
	IsItemLocked(context.Context, string) (bool, error)
	AuditEvents(context.Context, string) ([]AuditEvent, error)
	RecordSettlementFailure(context.Context, string, AuditEvent) error
}

type memoryTransaction struct {
	session            *Session
	characters         map[string]persistence.CharacterAggregate
	locks              map[string]ItemLock
	audit              []AuditEvent
	failure            error
	writes             int
	ledgerAccounts     map[string]ledger.LedgerAccount
	ledgerOwners       map[string]string
	ledgerTransactions map[string]ledger.LedgerTransaction
	ledgerReferences   map[string]string
	failures           map[string]error
}

func (t *memoryTransaction) Session() *Session { return t.session }
func (t *memoryTransaction) LoadCharacter(id string) (persistence.CharacterAggregate, error) {
	value, ok := t.characters[id]
	if !ok {
		return persistence.CharacterAggregate{}, persistence.ErrNotFound
	}
	return cloneAggregate(value), nil
}
func (t *memoryTransaction) SaveCharacter(value persistence.CharacterAggregate) error {
	if err := persistence.ValidateAggregate(value); err != nil {
		return err
	}
	current, ok := t.characters[value.Character.ID]
	if !ok {
		return persistence.ErrNotFound
	}
	value.Character.Revision = current.Character.Revision + 1
	t.characters[value.Character.ID] = cloneAggregate(value)
	t.writes++
	if t.writes == 1 && t.failure != nil {
		return t.failure
	}
	return nil
}
func (t *memoryTransaction) LockItem(lock ItemLock) error {
	if current, ok := t.locks[lock.ItemInstanceID]; ok && current.TradeID != lock.TradeID {
		return ErrItemLocked
	}
	t.locks[lock.ItemInstanceID] = lock
	return nil
}
func (t *memoryTransaction) ValidateItemLock(want ItemLock) error {
	current, ok := t.locks[want.ItemInstanceID]
	if !ok || current.TradeID != want.TradeID || current.OwnerCharacterID != want.OwnerCharacterID || current.Quantity != want.Quantity {
		return ErrItemLocked
	}
	return nil
}
func (t *memoryTransaction) UnlockItem(id string) error {
	if current, ok := t.locks[id]; ok && current.TradeID == t.session.TradeID {
		delete(t.locks, id)
	}
	return nil
}
func (t *memoryTransaction) UnlockAll() error {
	for id, lock := range t.locks {
		if lock.TradeID == t.session.TradeID {
			delete(t.locks, id)
		}
	}
	return nil
}
func (t *memoryTransaction) AppendAudit(event AuditEvent) error {
	t.audit = append(t.audit, event)
	return nil
}
func (t *memoryTransaction) PrepareSettlement(_, _, _ []string) error { return nil }
func (t *memoryTransaction) LoadLedgerAccountByOwner(owner string) (ledger.LedgerAccount, error) {
	id, ok := t.ledgerOwners[owner]
	if !ok {
		return ledger.LedgerAccount{}, ledger.ErrNotFound
	}
	return t.ledgerAccounts[id], nil
}
func (t *memoryTransaction) PostLedgerTransaction(draft ledger.PostDraft) (ledger.LedgerTransaction, error) {
	key := draft.Reference.Type + "\x00" + draft.Reference.ID
	if id, ok := t.ledgerReferences[key]; ok {
		existing := t.ledgerTransactions[id]
		if !sameLedgerIntent(existing, draft) {
			return ledger.LedgerTransaction{}, ledger.ErrReferenceConflict
		}
		return cloneLedgerTransaction(existing), nil
	}
	if err := ledger.ValidatePostDraft(draft); err != nil {
		return ledger.LedgerTransaction{}, err
	}
	balances := make(map[string]int64, len(draft.Entries))
	for _, entry := range draft.Entries {
		account, ok := t.ledgerAccounts[entry.AccountID]
		if !ok {
			return ledger.LedgerTransaction{}, ledger.ErrNotFound
		}
		next, err := ledger.AddAmount(account.Balance, entry.Amount)
		if err != nil {
			return ledger.LedgerTransaction{}, err
		}
		if next < 0 {
			return ledger.LedgerTransaction{}, ledger.ErrInsufficientBalance
		}
		balances[entry.AccountID] = next
	}
	result := ledger.LedgerTransaction{ID: draft.ID, Type: draft.Type, Status: ledger.StatusPosted, Reference: draft.Reference, OriginalTransactionID: draft.OriginalTransactionID, SpendClassification: draft.SpendClassification, Reason: draft.Reason, CreatedAt: draft.CreatedAt}
	for _, item := range draft.Entries {
		account := t.ledgerAccounts[item.AccountID]
		direction := ledger.DirectionCredit
		if item.Amount < 0 {
			direction = ledger.DirectionDebit
		}
		entry := ledger.LedgerEntry{ID: item.ID, TransactionID: draft.ID, AccountID: item.AccountID, EntryType: draft.Type, Amount: item.Amount, Direction: direction, BalanceBefore: account.Balance, BalanceAfter: balances[item.AccountID], CreatedAt: draft.CreatedAt}
		account.Balance = entry.BalanceAfter
		account.Revision++
		account.UpdatedAt = draft.CreatedAt
		t.ledgerAccounts[account.ID] = account
		result.Entries = append(result.Entries, entry)
		if len(result.Entries) == 1 {
			if err := t.CheckFailure(FailureDuringBalanceUpdate); err != nil {
				return ledger.LedgerTransaction{}, err
			}
		}
	}
	t.ledgerTransactions[result.ID] = cloneLedgerTransaction(result)
	t.ledgerReferences[key] = result.ID
	return cloneLedgerTransaction(result), nil
}
func (t *memoryTransaction) CheckFailure(point string) error {
	if err := t.failures[point]; err != nil {
		delete(t.failures, point)
		return err
	}
	return nil
}

type MemoryRepository struct {
	mu                 sync.Mutex
	sessions           map[string]Session
	characters         map[string]persistence.CharacterAggregate
	locks              map[string]ItemLock
	audit              map[string][]AuditEvent
	failures           map[string]error
	sequence           int64
	ledgerAccounts     map[string]ledger.LedgerAccount
	ledgerOwners       map[string]string
	ledgerTransactions map[string]ledger.LedgerTransaction
	ledgerReferences   map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{sessions: map[string]Session{}, characters: map[string]persistence.CharacterAggregate{}, locks: map[string]ItemLock{}, audit: map[string][]AuditEvent{}, failures: map[string]error{}, ledgerAccounts: map[string]ledger.LedgerAccount{}, ledgerOwners: map[string]string{}, ledgerTransactions: map[string]ledger.LedgerTransaction{}, ledgerReferences: map[string]string{}}
}

func (r *MemoryRepository) seedLedgerAccount(value ledger.LedgerAccount) error {
	if value.ID == "" || value.OwnerID == "" || value.OwnerType != ledger.OwnerPlayer || value.Currency != ledger.CurrencyFB || value.Balance < 0 {
		return ledger.ErrInvalidAccount
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.ledgerAccounts[value.ID]; exists {
		return ledger.ErrConflict
	}
	if _, exists := r.ledgerOwners[value.OwnerID]; exists {
		return ledger.ErrConflict
	}
	r.ledgerAccounts[value.ID] = value
	r.ledgerOwners[value.OwnerID] = value.ID
	return nil
}

func (r *MemoryRepository) replaceLedgerAccount(value ledger.LedgerAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.ledgerAccounts[value.ID]; !exists || value.Balance < 0 {
		return ledger.ErrNotFound
	}
	r.ledgerAccounts[value.ID] = value
	return nil
}

func (r *MemoryRepository) ledgerAccountByOwner(ctx context.Context, owner string) (ledger.LedgerAccount, error) {
	if err := ctx.Err(); err != nil {
		return ledger.LedgerAccount{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id, ok := r.ledgerOwners[owner]
	if !ok {
		return ledger.LedgerAccount{}, ledger.ErrNotFound
	}
	return r.ledgerAccounts[id], nil
}

func (r *MemoryRepository) ledgerTransaction(ctx context.Context, id string) (ledger.LedgerTransaction, error) {
	if err := ctx.Err(); err != nil {
		return ledger.LedgerTransaction{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.ledgerTransactions[id]
	if !ok {
		return ledger.LedgerTransaction{}, ledger.ErrNotFound
	}
	return cloneLedgerTransaction(value), nil
}

func (r *MemoryRepository) SeedCharacter(value persistence.CharacterAggregate) error {
	if err := persistence.ValidateAggregate(value); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.characters[value.Character.ID]; ok {
		return persistence.ErrConflict
	}
	r.characters[value.Character.ID] = cloneAggregate(value)
	return nil
}

func (r *MemoryRepository) ReplaceCharacter(value persistence.CharacterAggregate) error {
	if err := persistence.ValidateAggregate(value); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.characters[value.Character.ID]; !ok {
		return persistence.ErrNotFound
	}
	r.characters[value.Character.ID] = cloneAggregate(value)
	return nil
}

func (r *MemoryRepository) Character(ctx context.Context, id string) (persistence.CharacterAggregate, error) {
	if err := ctx.Err(); err != nil {
		return persistence.CharacterAggregate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.characters[id]
	if !ok {
		return persistence.CharacterAggregate{}, persistence.ErrNotFound
	}
	return cloneAggregate(v), nil
}

func (r *MemoryRepository) CreateTrade(ctx context.Context, value Session, event AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[value.TradeID]; ok {
		return persistence.ErrConflict
	}
	r.sessions[value.TradeID] = cloneSession(value)
	r.appendAudit(event)
	return nil
}

func (r *MemoryRepository) LoadTrade(ctx context.Context, id string) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return cloneSession(v), nil
}

func (r *MemoryRepository) Transact(ctx context.Context, id string, fn func(Transaction) error) (Session, error) {
	if err := ctx.Err(); err != nil {
		return Session{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, ok := r.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	s := cloneSession(value)
	characters := make(map[string]persistence.CharacterAggregate, len(r.characters))
	for key, character := range r.characters {
		characters[key] = cloneAggregate(character)
	}
	locks := make(map[string]ItemLock, len(r.locks))
	for key, lock := range r.locks {
		locks[key] = lock
	}
	accounts := make(map[string]ledger.LedgerAccount, len(r.ledgerAccounts))
	for key, account := range r.ledgerAccounts {
		accounts[key] = account
	}
	owners := make(map[string]string, len(r.ledgerOwners))
	for key, id := range r.ledgerOwners {
		owners[key] = id
	}
	transactions := make(map[string]ledger.LedgerTransaction, len(r.ledgerTransactions))
	for key, value := range r.ledgerTransactions {
		transactions[key] = cloneLedgerTransaction(value)
	}
	references := make(map[string]string, len(r.ledgerReferences))
	for key, id := range r.ledgerReferences {
		references[key] = id
	}
	failures := make(map[string]error, len(r.failures))
	for key, value := range r.failures {
		failures[key] = value
	}
	tx := &memoryTransaction{session: &s, characters: characters, locks: locks, failure: failures[FailureAfterFirstCharacterWrite], ledgerAccounts: accounts, ledgerOwners: owners, ledgerTransactions: transactions, ledgerReferences: references, failures: failures}
	delete(r.failures, FailureAfterFirstCharacterWrite)
	for _, point := range []string{FailureAfterItemOwnershipUpdate, FailureDuringBalanceUpdate, FailureAfterLedgerWrite, FailureBeforeCompletedState, FailureBeforeCommit} {
		delete(r.failures, point)
	}
	if err := fn(tx); err != nil {
		return Session{}, err
	}
	if err := tx.CheckFailure(FailureBeforeCommit); err != nil {
		return Session{}, err
	}
	r.sessions[id], r.characters, r.locks = cloneSession(s), characters, locks
	r.ledgerAccounts, r.ledgerOwners, r.ledgerTransactions, r.ledgerReferences = accounts, owners, transactions, references
	for _, event := range tx.audit {
		r.appendAudit(event)
	}
	return cloneSession(s), nil
}

func (r *MemoryRepository) IsItemLocked(ctx context.Context, id string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.locks[id]
	return ok, nil
}

func (r *MemoryRepository) AuditEvents(ctx context.Context, id string) ([]AuditEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[id]; !ok {
		return nil, ErrNotFound
	}
	result := append([]AuditEvent(nil), r.audit[id]...)
	for i := range result {
		result[i].ParticipantIDs = append([]string(nil), result[i].ParticipantIDs...)
		result[i].LedgerTransactionIDs = append([]string(nil), result[i].LedgerTransactionIDs...)
	}
	return result, nil
}

func (r *MemoryRepository) RecordSettlementFailure(ctx context.Context, id string, event AuditEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.sessions[id]; !ok {
		return ErrNotFound
	}
	r.appendAudit(event)
	return nil
}

func (r *MemoryRepository) InjectFailure(point string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failures[point] = err
}

func (r *MemoryRepository) appendAudit(event AuditEvent) {
	r.sequence++
	event.Sequence = r.sequence
	event.ParticipantIDs = append([]string(nil), event.ParticipantIDs...)
	event.LedgerTransactionIDs = append([]string(nil), event.LedgerTransactionIDs...)
	r.audit[event.TradeID] = append(r.audit[event.TradeID], event)
}

func cloneLedgerTransaction(value ledger.LedgerTransaction) ledger.LedgerTransaction {
	value.Entries = append([]ledger.LedgerEntry(nil), value.Entries...)
	return value
}

func sameLedgerIntent(existing ledger.LedgerTransaction, draft ledger.PostDraft) bool {
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

func sortedUnique(values []string) []string {
	seen := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func cloneAggregate(value persistence.CharacterAggregate) persistence.CharacterAggregate {
	value.Items = append([]persistence.ItemInstance(nil), value.Items...)
	value.Equipment = append([]persistence.EquippedItem(nil), value.Equipment...)
	value.Skills = append([]persistence.SkillState(nil), value.Skills...)
	return value
}

func settlementError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrInventoryCapacity) || errors.Is(err, ErrInvalidOwnership) || errors.Is(err, ErrInvalidQuantity) || errors.Is(err, ErrItemLocked) || errors.Is(err, ErrNotReady) || errors.Is(err, ErrTerminalState) {
		return err
	}
	return errors.Join(ErrSettlementFailed, err)
}
