package trade

import (
	"context"
	"errors"
	"sync"

	"fractallegend/game-server/internal/persistence"
)

const FailureAfterFirstCharacterWrite = "after_first_character_write"

type Transaction interface {
	Session() *Session
	LoadCharacter(string) (persistence.CharacterAggregate, error)
	SaveCharacter(persistence.CharacterAggregate) error
	LockItem(ItemLock) error
	ValidateItemLock(ItemLock) error
	UnlockItem(string) error
	UnlockAll() error
	AppendAudit(AuditEvent) error
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
	session    *Session
	characters map[string]persistence.CharacterAggregate
	locks      map[string]ItemLock
	audit      []AuditEvent
	failure    error
	writes     int
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

type MemoryRepository struct {
	mu         sync.Mutex
	sessions   map[string]Session
	characters map[string]persistence.CharacterAggregate
	locks      map[string]ItemLock
	audit      map[string][]AuditEvent
	failures   map[string]error
	sequence   int64
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{sessions: map[string]Session{}, characters: map[string]persistence.CharacterAggregate{}, locks: map[string]ItemLock{}, audit: map[string][]AuditEvent{}, failures: map[string]error{}}
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
	tx := &memoryTransaction{session: &s, characters: characters, locks: locks, failure: r.failures[FailureAfterFirstCharacterWrite]}
	delete(r.failures, FailureAfterFirstCharacterWrite)
	if err := fn(tx); err != nil {
		return Session{}, err
	}
	r.sessions[id], r.characters, r.locks = cloneSession(s), characters, locks
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
	r.audit[event.TradeID] = append(r.audit[event.TradeID], event)
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
