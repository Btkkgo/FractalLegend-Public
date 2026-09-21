package items

import (
	"errors"
	"sort"
)

type CanonicalItem struct {
	ID, Name, ItemType, SourceFile, SourceRecord, SemanticStatus string
	LegacyID                                                     int
}

type ItemDefinition struct {
	ID             string `json:"id"`
	LegacyID       int    `json:"legacyId"`
	Name           string `json:"name"`
	ItemType       string `json:"itemType"`
	SourceFile     string `json:"sourceFile"`
	SourceRecord   string `json:"sourceRecord"`
	SemanticStatus string `json:"semanticStatus"`
}

type ItemInstance struct {
	InstanceID       string `json:"instanceId"`
	DefinitionID     string `json:"definitionId"`
	LegacyID         int    `json:"legacyId"`
	Name             string `json:"name"`
	ItemType         string `json:"itemType"`
	Quantity         int    `json:"quantity"`
	SlotIndex        int    `json:"slotIndex"`
	OwnerCharacterID string `json:"ownerCharacterId,omitempty"`
	Location         string `json:"location"`
	EquipmentSlot    string `json:"equipmentSlot,omitempty"`
}

type Registry struct{ definitions map[string]ItemDefinition }

func NewRegistry(canonical []CanonicalItem) (*Registry, error) {
	r := &Registry{definitions: map[string]ItemDefinition{}}
	for _, item := range canonical {
		if item.ID == "" || item.LegacyID < 0 || item.Name == "" || item.ItemType == "" || item.SourceFile == "" || item.SourceRecord == "" {
			return nil, errors.New("invalid canonical item")
		}
		if _, exists := r.definitions[item.ID]; exists {
			return nil, errors.New("duplicate canonical item")
		}
		r.definitions[item.ID] = ItemDefinition{ID: item.ID, LegacyID: item.LegacyID, Name: item.Name, ItemType: item.ItemType, SourceFile: item.SourceFile, SourceRecord: item.SourceRecord, SemanticStatus: item.SemanticStatus}
	}
	if len(r.definitions) == 0 {
		return nil, errors.New("item registry is empty")
	}
	return r, nil
}

func (r *Registry) Get(id string) (ItemDefinition, bool) {
	value, ok := r.definitions[id]
	return value, ok
}
func (r *Registry) List() []ItemDefinition {
	out := make([]ItemDefinition, 0, len(r.definitions))
	for _, v := range r.definitions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func NewInstance(id string, definition ItemDefinition, quantity int) ItemInstance {
	return ItemInstance{InstanceID: id, DefinitionID: definition.ID, LegacyID: definition.LegacyID, Name: definition.Name, ItemType: definition.ItemType, Quantity: quantity, SlotIndex: -1, Location: "GROUND"}
}
