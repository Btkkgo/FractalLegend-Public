package drops

import (
	"errors"
	"math/rand"
	"sort"
)

type RuntimeDropDefinition struct {
	ID                  string `json:"id"`
	MonsterDefinitionID string `json:"monsterDefinitionId"`
	DropTableID         string `json:"dropTableId"`
	ItemDefinitionID    string `json:"itemDefinitionId"`
	RawRule             string `json:"rawRule"`
	Numerator           int    `json:"numerator"`
	Denominator         int    `json:"denominator"`
	Quantity            int    `json:"quantity"`
	RuleVersion         int    `json:"ruleVersion"`
	SemanticSource      string `json:"semanticSource"`
}

func (d RuntimeDropDefinition) Chance() float64 { return float64(d.Numerator) / float64(d.Denominator) }

type Registry struct {
	byMonster   map[string]RuntimeDropDefinition
	definitions map[string]RuntimeDropDefinition
}

func NewRegistry(definitions []RuntimeDropDefinition) (*Registry, error) {
	r := &Registry{byMonster: map[string]RuntimeDropDefinition{}, definitions: map[string]RuntimeDropDefinition{}}
	for _, d := range definitions {
		if d.ID == "" || d.MonsterDefinitionID == "" || d.DropTableID == "" || d.ItemDefinitionID == "" || d.RawRule == "" || d.Numerator < 0 || d.Denominator <= 0 || d.Numerator > d.Denominator || d.Quantity <= 0 || d.RuleVersion != 1 || d.SemanticSource == "" {
			return nil, errors.New("invalid runtime drop definition")
		}
		if _, ok := r.definitions[d.ID]; ok {
			return nil, errors.New("duplicate drop definition")
		}
		if _, ok := r.byMonster[d.MonsterDefinitionID]; ok {
			return nil, errors.New("multiple G7 rules for monster")
		}
		r.definitions[d.ID] = d
		r.byMonster[d.MonsterDefinitionID] = d
	}
	if len(r.definitions) == 0 {
		return nil, errors.New("drop registry is empty")
	}
	return r, nil
}
func (r *Registry) ForMonster(id string) (RuntimeDropDefinition, bool) {
	v, ok := r.byMonster[id]
	return v, ok
}
func (r *Registry) List() []RuntimeDropDefinition {
	out := make([]RuntimeDropDefinition, 0, len(r.definitions))
	for _, v := range r.definitions {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
func ShouldDrop(definition RuntimeDropDefinition, rng *rand.Rand) bool {
	if definition.Numerator >= definition.Denominator {
		return true
	}
	return rng.Intn(definition.Denominator) < definition.Numerator
}
