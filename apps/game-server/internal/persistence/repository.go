package persistence

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrNotFound         = errors.New("persistence record not found")
	ErrConflict         = errors.New("persistence record already exists")
	ErrAccountNotFound  = errors.New("persistence account not found")
	ErrStaleRevision    = errors.New("stale character revision")
	ErrInvalidAggregate = errors.New("invalid character aggregate")
	ErrUnavailable      = errors.New("persistence unavailable")
)

const (
	InventoryLocation = "INVENTORY"
	EquipmentLocation = "EQUIPMENT"
)

type Account struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Character struct {
	ID, AccountID, Name, ClassID, ClassName string
	Level                                   int
	EXP                                     int64
	HP, MP                                  float64
	Revision                                int64
}

type WorldState struct {
	MapID string
	X, Y  int
}

type ItemInstance struct {
	InstanceID, DefinitionID, Name, ItemType string
	LegacyID                                 int
	Quantity, SlotIndex                      int
	Location, EquipmentSlot                  string
}

type EquippedItem struct {
	Slot, ItemInstanceID string
}

type SkillState struct {
	SkillID string
	Learned bool
	Level   int
}

type CharacterAggregate struct {
	Character Character
	World     WorldState
	Items     []ItemInstance
	Equipment []EquippedItem
	Skills    []SkillState
}

type CharacterRepository interface {
	CreateAccount(context.Context, Account) error
	LoadAccount(context.Context, string) (Account, error)
	CreateCharacter(context.Context, CharacterAggregate) (int64, error)
	LoadCharacter(context.Context, string) (CharacterAggregate, error)
	SaveCharacter(context.Context, CharacterAggregate, int64) (int64, error)
}

func ValidateAggregate(value CharacterAggregate) error {
	c := value.Character
	if c.ID == "" || c.AccountID == "" || c.Name == "" || c.ClassID == "" || c.ClassName == "" || c.Level < 1 || c.EXP < 0 || c.HP < 0 || c.MP < 0 || value.World.MapID == "" {
		return ErrInvalidAggregate
	}
	itemsByID := make(map[string]ItemInstance, len(value.Items))
	for _, item := range value.Items {
		if item.InstanceID == "" || item.DefinitionID == "" || item.Name == "" || item.ItemType == "" || item.Quantity < 1 || item.SlotIndex < 0 || (item.Location != InventoryLocation && item.Location != EquipmentLocation) {
			return ErrInvalidAggregate
		}
		if _, exists := itemsByID[item.InstanceID]; exists {
			return ErrInvalidAggregate
		}
		itemsByID[item.InstanceID] = item
	}
	equippedByInstance := make(map[string]string, len(value.Equipment))
	slots := make(map[string]bool, len(value.Equipment))
	for _, equipped := range value.Equipment {
		item, exists := itemsByID[equipped.ItemInstanceID]
		if equipped.Slot == "" || !exists || slots[equipped.Slot] || item.ItemType != "EQUIPMENT" || item.Location != EquipmentLocation || item.EquipmentSlot != equipped.Slot {
			return ErrInvalidAggregate
		}
		if _, exists = equippedByInstance[equipped.ItemInstanceID]; exists {
			return ErrInvalidAggregate
		}
		slots[equipped.Slot] = true
		equippedByInstance[equipped.ItemInstanceID] = equipped.Slot
	}
	for _, item := range value.Items {
		_, equipped := equippedByInstance[item.InstanceID]
		if (item.Location == EquipmentLocation) != equipped || (item.Location == InventoryLocation && item.EquipmentSlot != "") {
			return ErrInvalidAggregate
		}
	}
	skills := make(map[string]bool, len(value.Skills))
	for _, skill := range value.Skills {
		if skill.SkillID == "" || skill.Level < 1 || !skill.Learned || skills[skill.SkillID] {
			return ErrInvalidAggregate
		}
		skills[skill.SkillID] = true
	}
	return nil
}

type MemoryRepository struct {
	mu         sync.Mutex
	accounts   map[string]Account
	characters map[string]CharacterAggregate
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{accounts: map[string]Account{}, characters: map[string]CharacterAggregate{}}
}

func (r *MemoryRepository) CreateAccount(ctx context.Context, value Account) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if value.ID == "" {
		return ErrInvalidAggregate
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.accounts[value.ID]; exists {
		return ErrConflict
	}
	now := time.Now().UTC()
	if value.CreatedAt.IsZero() {
		value.CreatedAt = now
	}
	if value.UpdatedAt.IsZero() {
		value.UpdatedAt = value.CreatedAt
	}
	r.accounts[value.ID] = value
	return nil
}

func (r *MemoryRepository) LoadAccount(ctx context.Context, id string) (Account, error) {
	if err := ctx.Err(); err != nil {
		return Account{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, exists := r.accounts[id]
	if !exists {
		return Account{}, ErrNotFound
	}
	return value, nil
}

func (r *MemoryRepository) CreateCharacter(ctx context.Context, value CharacterAggregate) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := ValidateAggregate(value); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.accounts[value.Character.AccountID]; !exists {
		return 0, ErrAccountNotFound
	}
	if _, exists := r.characters[value.Character.ID]; exists {
		return 0, ErrConflict
	}
	value.Character.Revision = 1
	r.characters[value.Character.ID] = cloneAggregate(value)
	return 1, nil
}

func (r *MemoryRepository) LoadCharacter(ctx context.Context, id string) (CharacterAggregate, error) {
	if err := ctx.Err(); err != nil {
		return CharacterAggregate{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	value, exists := r.characters[id]
	if !exists {
		return CharacterAggregate{}, ErrNotFound
	}
	return cloneAggregate(value), nil
}

func (r *MemoryRepository) SaveCharacter(ctx context.Context, value CharacterAggregate, expectedRevision int64) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if err := ValidateAggregate(value); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	current, exists := r.characters[value.Character.ID]
	if !exists {
		return 0, ErrNotFound
	}
	if current.Character.AccountID != value.Character.AccountID {
		return 0, fmt.Errorf("%w: account ownership changed", ErrInvalidAggregate)
	}
	if current.Character.Revision != expectedRevision {
		return 0, ErrStaleRevision
	}
	value.Character.Revision = expectedRevision + 1
	r.characters[value.Character.ID] = cloneAggregate(value)
	return value.Character.Revision, nil
}

func cloneAggregate(value CharacterAggregate) CharacterAggregate {
	value.Items = append([]ItemInstance(nil), value.Items...)
	value.Equipment = append([]EquippedItem(nil), value.Equipment...)
	value.Skills = append([]SkillState(nil), value.Skills...)
	return value
}
