package skills

import (
	"errors"
	"math/rand"
	"sort"
	"time"
)

type RuntimeType string
type SourceMode string

const (
	ActiveDamage    RuntimeType = "ACTIVE_DAMAGE"
	Legacy          SourceMode  = "LEGACY"
	FractalOverride SourceMode  = "FRACTAL_OVERRIDE"
	Mixed           SourceMode  = "MIXED"
)

type CanonicalSkill struct {
	ID             string
	LegacyID       int
	Name           string
	ClassID        string
	LegacyJob      int
	SourceFile     string
	SourceRecord   string
	SemanticStatus string
}

type Override struct {
	ID               string
	CanonicalSkillID string
	OverrideVersion  string
	RuntimeType      RuntimeType
	ManaCost         int
	Cooldown         time.Duration
	CastTime         time.Duration
	Range            int
	DamageMin        int
	DamageMax        int
	DamageType       string
	TargetType       string
	RuleVersion      int
	SourceMode       SourceMode
}

type RuntimeSkillDefinition struct {
	ID                   string        `json:"id"`
	CanonicalSkillID     string        `json:"canonicalSkillId"`
	LegacyID             int           `json:"legacyId"`
	Name                 string        `json:"name"`
	ClassID              string        `json:"classId"`
	RuntimeType          RuntimeType   `json:"runtimeType"`
	ManaCost             int           `json:"manaCost"`
	Cooldown             time.Duration `json:"-"`
	CooldownMS           int           `json:"cooldownMs"`
	CastTime             time.Duration `json:"-"`
	CastTimeMS           int           `json:"castTimeMs"`
	Range                int           `json:"range"`
	DamageMin            int           `json:"damageMin"`
	DamageMax            int           `json:"damageMax"`
	DamageType           string        `json:"damageType"`
	TargetType           string        `json:"targetType"`
	RuleVersion          int           `json:"ruleVersion"`
	SourceMode           SourceMode    `json:"sourceMode"`
	LegacySourceFile     string        `json:"legacySourceFile"`
	LegacySourceRecord   string        `json:"legacySourceRecord"`
	LegacySemanticStatus string        `json:"legacySemanticStatus"`
	OverrideVersion      string        `json:"overrideVersion"`
}

type Registry struct {
	definitions map[string]RuntimeSkillDefinition
	quarantined []string
}

func NewRegistry(canonical []CanonicalSkill, overrides []Override) (*Registry, error) {
	bases := make(map[string]CanonicalSkill, len(canonical))
	for _, base := range canonical {
		if base.ID != "" && base.LegacyID >= 0 && base.Name != "" && base.ClassID != "" && base.SourceFile != "" && base.SourceRecord != "" {
			bases[base.ID] = base
		}
	}
	r := &Registry{definitions: map[string]RuntimeSkillDefinition{}}
	for _, override := range overrides {
		base, ok := bases[override.CanonicalSkillID]
		if !ok || !validOverride(override) {
			r.quarantined = append(r.quarantined, override.ID)
			continue
		}
		if _, duplicate := r.definitions[override.ID]; duplicate {
			r.quarantined = append(r.quarantined, override.ID)
			continue
		}
		r.definitions[override.ID] = RuntimeSkillDefinition{
			ID: override.ID, CanonicalSkillID: base.ID, LegacyID: base.LegacyID, Name: base.Name, ClassID: base.ClassID,
			RuntimeType: override.RuntimeType, ManaCost: override.ManaCost, Cooldown: override.Cooldown, CooldownMS: int(override.Cooldown.Milliseconds()),
			CastTime: override.CastTime, CastTimeMS: int(override.CastTime.Milliseconds()), Range: override.Range,
			DamageMin: override.DamageMin, DamageMax: override.DamageMax, DamageType: override.DamageType, TargetType: override.TargetType,
			RuleVersion: override.RuleVersion, SourceMode: override.SourceMode, LegacySourceFile: base.SourceFile,
			LegacySourceRecord: base.SourceRecord, LegacySemanticStatus: base.SemanticStatus, OverrideVersion: override.OverrideVersion,
		}
	}
	sort.Strings(r.quarantined)
	if len(r.definitions) == 0 {
		return nil, errors.New("no valid runtime skills")
	}
	return r, nil
}

func validOverride(v Override) bool {
	return v.ID != "" && v.CanonicalSkillID != "" && v.OverrideVersion != "" && v.RuntimeType == ActiveDamage &&
		v.ManaCost >= 0 && v.Cooldown > 0 && v.CastTime >= 0 && v.Range >= 0 && v.DamageMin >= 0 && v.DamageMax >= v.DamageMin &&
		v.DamageType == "PHYSICAL" && v.TargetType == "MONSTER" && v.RuleVersion == 1 &&
		(v.SourceMode == Legacy || v.SourceMode == FractalOverride || v.SourceMode == Mixed)
}

func (r *Registry) Get(id string) (RuntimeSkillDefinition, bool) {
	v, ok := r.definitions[id]
	return v, ok
}

func (r *Registry) Quarantined() []string { return append([]string(nil), r.quarantined...) }

func DamageV1(def RuntimeSkillDefinition, attackMin, attackMax, defense, minimum int, rng *rand.Rand) int {
	base := def.DamageMin
	if def.DamageMax > def.DamageMin {
		base += rng.Intn(def.DamageMax - def.DamageMin + 1)
	}
	result := base + (attackMin+attackMax)/2 - defense
	if result < minimum {
		return minimum
	}
	return result
}
