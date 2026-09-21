package persistence

import (
	"context"
	"errors"
	"testing"
	"time"
)

func aggregateFixture() CharacterAggregate {
	return CharacterAggregate{
		Character: Character{
			ID: "character-1", AccountID: "account-1", Name: "Persistence Warrior",
			ClassID: "class-warrior", ClassName: "战士", Level: 1, EXP: 25,
			HP: 120, MP: 32,
		},
		World: WorldState{MapID: "map-1", X: 7, Y: 9},
		Items: []ItemInstance{
			{InstanceID: "item-material", DefinitionID: "item-awakening", LegacyID: 20, Name: "觉醒石", ItemType: "MATERIAL", Quantity: 1, SlotIndex: 0, Location: InventoryLocation},
			{InstanceID: "item-weapon", DefinitionID: "item-mafa-tulong", LegacyID: 411, Name: "玛法屠龙", ItemType: "EQUIPMENT", Quantity: 1, SlotIndex: 1, Location: EquipmentLocation, EquipmentSlot: "WEAPON"},
		},
		Equipment: []EquippedItem{{Slot: "WEAPON", ItemInstanceID: "item-weapon"}},
		Skills:    []SkillState{{SkillID: "skill-liehuo", Learned: true, Level: 1}},
	}
}

func TestMemoryRepositoryRoundTripsCompleteCharacterAggregate(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	now := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	account := Account{ID: "account-1", CreatedAt: now, UpdatedAt: now}
	if err := repository.CreateAccount(ctx, account); err != nil {
		t.Fatal(err)
	}
	aggregate := aggregateFixture()
	revision, err := repository.CreateCharacter(ctx, aggregate)
	if err != nil {
		t.Fatal(err)
	}
	if revision != 1 {
		t.Fatalf("initial revision=%d", revision)
	}
	loaded, err := repository.LoadCharacter(ctx, aggregate.Character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Character.Revision != 1 || loaded.World != aggregate.World || len(loaded.Items) != 2 || loaded.Items[1].InstanceID != "item-weapon" || len(loaded.Equipment) != 1 || loaded.Equipment[0].ItemInstanceID != "item-weapon" || len(loaded.Skills) != 1 || loaded.Skills[0].SkillID != "skill-liehuo" {
		t.Fatalf("round trip mismatch: %#v", loaded)
	}
}

func TestAggregateRejectsDuplicatedOrInconsistentItemLocation(t *testing.T) {
	cases := map[string]func(*CharacterAggregate){
		"duplicate instance":                        func(value *CharacterAggregate) { value.Items = append(value.Items, value.Items[0]) },
		"equipped item marked inventory":            func(value *CharacterAggregate) { value.Items[1].Location = InventoryLocation },
		"missing equipment relation":                func(value *CharacterAggregate) { value.Equipment = nil },
		"equipment relation points to missing item": func(value *CharacterAggregate) { value.Equipment[0].ItemInstanceID = "missing" },
		"material equipped":                         func(value *CharacterAggregate) { value.Equipment[0].ItemInstanceID = "item-material" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			value := aggregateFixture()
			mutate(&value)
			if err := ValidateAggregate(value); !errors.Is(err, ErrInvalidAggregate) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestMemoryRepositoryRejectsStaleAndConcurrentSave(t *testing.T) {
	ctx := context.Background()
	repository := NewMemoryRepository()
	if err := repository.CreateAccount(ctx, Account{ID: "account-1"}); err != nil {
		t.Fatal(err)
	}
	value := aggregateFixture()
	revision, err := repository.CreateCharacter(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := repository.LoadCharacter(ctx, value.Character.ID)
	second, _ := repository.LoadCharacter(ctx, value.Character.ID)
	first.Character.EXP = 40
	next, err := repository.SaveCharacter(ctx, first, revision)
	if err != nil || next != 2 {
		t.Fatalf("first save revision=%d error=%v", next, err)
	}
	second.Character.EXP = 80
	if _, err = repository.SaveCharacter(ctx, second, revision); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("stale save error=%v", err)
	}
	loaded, _ := repository.LoadCharacter(ctx, value.Character.ID)
	if loaded.Character.EXP != 40 || loaded.Character.Revision != 2 {
		t.Fatalf("stale writer changed aggregate: %#v", loaded.Character)
	}
}

func TestRepositoryRejectsCharacterForUnknownAccount(t *testing.T) {
	repository := NewMemoryRepository()
	if _, err := repository.CreateCharacter(context.Background(), aggregateFixture()); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("error=%v", err)
	}
}
