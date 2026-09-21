package equipment

import (
	"errors"
	"sort"
)

type Slot string
type SourceMode string

const (
	Weapon        Slot = "WEAPON"
	Armor         Slot = "ARMOR"
	Helmet        Slot = "HELMET"
	Necklace      Slot = "NECKLACE"
	BraceletLeft  Slot = "BRACELET_LEFT"
	BraceletRight Slot = "BRACELET_RIGHT"
	RingLeft      Slot = "RING_LEFT"
	RingRight     Slot = "RING_RIGHT"
	Belt          Slot = "BELT"
	Boots         Slot = "BOOTS"
	Other         Slot = "OTHER"

	Legacy          SourceMode = "LEGACY"
	FractalOverride SourceMode = "FRACTAL_OVERRIDE"
	Mixed           SourceMode = "MIXED"
)

var validSlots = map[Slot]bool{
	Weapon: true, Armor: true, Helmet: true, Necklace: true,
	BraceletLeft: true, BraceletRight: true, RingLeft: true, RingRight: true,
	Belt: true, Boots: true, Other: true,
}

func ValidSlot(slot Slot) bool { return validSlots[slot] }

type Modifiers struct {
	HPBonus           float64 `json:"hpBonus"`
	MPBonus           float64 `json:"mpBonus"`
	AttackMinBonus    float64 `json:"attackMinBonus"`
	AttackMaxBonus    float64 `json:"attackMaxBonus"`
	DefenseBonus      float64 `json:"defenseBonus"`
	MagicDefenseBonus float64 `json:"magicDefenseBonus"`
}

type CanonicalEquipment struct {
	ID, Name, SourceFile, SourceRecord, SemanticStatus string
	LegacyID                                           int
}

type Override struct {
	DefinitionID    string
	Slot            Slot
	AllowedClasses  []string
	RequiredLevel   int
	Modifiers       Modifiers
	RuleVersion     int
	SourceMode      SourceMode
	OverrideVersion string
}

type RuntimeEquipmentDefinition struct {
	DefinitionID         string     `json:"definitionId"`
	LegacyID             int        `json:"legacyId"`
	Name                 string     `json:"name"`
	Slot                 Slot       `json:"slot"`
	AllowedClasses       []string   `json:"allowedClasses"`
	RequiredLevel        int        `json:"requiredLevel"`
	Modifiers            Modifiers  `json:"modifiers"`
	RuleVersion          int        `json:"ruleVersion"`
	SourceMode           SourceMode `json:"sourceMode"`
	LegacySourceFile     string     `json:"legacySourceFile"`
	LegacySourceRecord   string     `json:"legacySourceRecord"`
	LegacySemanticStatus string     `json:"legacySemanticStatus"`
	OverrideVersion      string     `json:"overrideVersion"`
}

type Registry struct {
	definitions map[string]RuntimeEquipmentDefinition
}

func NewRegistry(canonical []CanonicalEquipment, overrides []Override) (*Registry, error) {
	bases := make(map[string]CanonicalEquipment, len(canonical))
	for _, item := range canonical {
		if item.ID == "" || item.LegacyID < 0 || item.Name == "" || item.SourceFile == "" || item.SourceRecord == "" {
			continue
		}
		bases[item.ID] = item
	}
	r := &Registry{definitions: map[string]RuntimeEquipmentDefinition{}}
	for _, override := range overrides {
		base, ok := bases[override.DefinitionID]
		if !ok || !validOverride(override) {
			continue
		}
		if _, duplicate := r.definitions[override.DefinitionID]; duplicate {
			return nil, errors.New("duplicate runtime equipment definition")
		}
		classes := append([]string(nil), override.AllowedClasses...)
		sort.Strings(classes)
		r.definitions[override.DefinitionID] = RuntimeEquipmentDefinition{
			DefinitionID: base.ID, LegacyID: base.LegacyID, Name: base.Name,
			Slot: override.Slot, AllowedClasses: classes, RequiredLevel: override.RequiredLevel,
			Modifiers: override.Modifiers, RuleVersion: override.RuleVersion, SourceMode: override.SourceMode,
			LegacySourceFile: base.SourceFile, LegacySourceRecord: base.SourceRecord,
			LegacySemanticStatus: base.SemanticStatus, OverrideVersion: override.OverrideVersion,
		}
	}
	if len(r.definitions) == 0 {
		return nil, errors.New("no valid runtime equipment")
	}
	return r, nil
}

func validOverride(value Override) bool {
	if value.DefinitionID == "" || !ValidSlot(value.Slot) || len(value.AllowedClasses) == 0 || value.RequiredLevel < 0 || value.RuleVersion != 1 || value.OverrideVersion == "" {
		return false
	}
	if value.SourceMode != Legacy && value.SourceMode != FractalOverride && value.SourceMode != Mixed {
		return false
	}
	for _, classID := range value.AllowedClasses {
		if classID == "" {
			return false
		}
	}
	m := value.Modifiers
	return m.HPBonus >= 0 && m.MPBonus >= 0 && m.AttackMinBonus >= 0 && m.AttackMaxBonus >= 0 && m.DefenseBonus >= 0 && m.MagicDefenseBonus >= 0
}

func (r *Registry) Get(id string) (RuntimeEquipmentDefinition, bool) {
	value, ok := r.definitions[id]
	return value, ok
}
func (r *Registry) List() []RuntimeEquipmentDefinition {
	out := make([]RuntimeEquipmentDefinition, 0, len(r.definitions))
	for _, value := range r.definitions {
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DefinitionID < out[j].DefinitionID })
	return out
}
