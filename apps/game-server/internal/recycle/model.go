package recycle

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrInvalidIntent  = errors.New("invalid recycle intent")
	ErrClientIntent   = errors.New("client-originated recycle intent is forbidden")
	ErrUnknownRule    = errors.New("unknown recycle rule")
	ErrDisabledRule   = errors.New("disabled recycle rule")
	ErrInvalidRule    = errors.New("invalid recycle rule")
	ErrNotFound       = errors.New("recycle item not found")
	ErrOwnership      = errors.New("recycle item ownership mismatch")
	ErrRevision       = errors.New("recycle item revision mismatch")
	ErrLocked         = errors.New("recycle item is locked or equipped")
	ErrConsumed       = errors.New("recycle item already consumed")
	ErrConflict       = errors.New("conflicting recycle operation")
	ErrReconciliation = errors.New("recycle reconciliation mismatch")
)

// Intent is created by an internal server producer. A network JSON payload cannot
// directly authorize this operation or specify an award.
type Intent struct {
	OperationID          string
	PlayerID             string
	ItemInstanceID       string
	ExpectedItemRevision int64
	RuleID               string
	RequestedAt          time.Time
}

func (*Intent) UnmarshalJSON([]byte) error { return ErrClientIntent }

func (i Intent) Valid() bool {
	return validID(i.OperationID, 120) && validID(i.PlayerID, 128) &&
		validID(i.ItemInstanceID, 128) && validID(i.RuleID, 128) &&
		i.ExpectedItemRevision > 0 && !i.RequestedAt.IsZero()
}

func validID(s string, max int) bool {
	if s == "" || len(s) > max {
		return false
	}
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':' || c == '/') {
			return false
		}
	}
	return true
}

type Material struct {
	ID       string
	Quantity int64
}

const MaterialRecycleScrap = "MATERIAL_RECYCLE_SCRAP"

// AntiFarmPolicy is a future boundary. G16 deliberately does not choose caps.
type AntiFarmPolicy struct {
	PeriodCapPolicy         string
	DiminishingReturnPolicy string
}

type Rule struct {
	ID                   string
	Version              string
	TemplateID           string
	Enabled              bool
	Recyclable           bool
	VerifiedInput        []Material
	MaterialOutputs      []Material
	ReputationReward     int64
	MaxReturnBasisPoints int64
	InputProvenance      string
	AntiFarm             AntiFarmPolicy
	CreatedAt            time.Time
}

type Registry struct{ rules map[string]Rule }

// ProductionRegistry has no enabled rule until a later, separately reviewed
// gameplay integration gate provides verified bills of materials.
func ProductionRegistry() Registry { return Registry{rules: map[string]Rule{}} }

func NewRegistry(values []Rule) (Registry, error) {
	r := Registry{rules: make(map[string]Rule, len(values))}
	for _, value := range values {
		if !validID(value.ID, 128) || !validID(value.Version, 64) ||
			!validID(value.TemplateID, 512) || value.CreatedAt.IsZero() ||
			strings.TrimSpace(value.InputProvenance) == "" ||
			value.MaxReturnBasisPoints < 0 || value.MaxReturnBasisPoints > 5000 ||
			value.ReputationReward < 0 || value.ReputationReward > 1_000_000_000 || value.AntiFarm.PeriodCapPolicy == "" ||
			value.AntiFarm.DiminishingReturnPolicy == "" {
			return Registry{}, ErrInvalidRule
		}
		if _, exists := r.rules[value.ID]; exists {
			return Registry{}, ErrInvalidRule
		}
		input, ok := materialMap(value.VerifiedInput)
		if !ok || len(input) == 0 {
			return Registry{}, ErrInvalidRule
		}
		output, ok := materialMap(value.MaterialOutputs)
		if !ok {
			return Registry{}, ErrInvalidRule
		}
		for id, amount := range output {
			if id == "BLACK_IRON_ORE" || input[id] == 0 || amount > input[id]/2 ||
				amount*10000 > input[id]*value.MaxReturnBasisPoints {
				return Registry{}, ErrInvalidRule
			}
		}
		value.VerifiedInput = cloneMaterials(value.VerifiedInput)
		value.MaterialOutputs = cloneMaterials(value.MaterialOutputs)
		r.rules[value.ID] = value
	}
	return r, nil
}

func (r Registry) Resolve(id string) (Rule, error) {
	v, ok := r.rules[id]
	if !ok {
		return Rule{}, ErrUnknownRule
	}
	if !v.Enabled || !v.Recyclable {
		return Rule{}, ErrDisabledRule
	}
	v.VerifiedInput = cloneMaterials(v.VerifiedInput)
	v.MaterialOutputs = cloneMaterials(v.MaterialOutputs)
	return v, nil
}

func materialMap(values []Material) (map[string]int64, bool) {
	out := make(map[string]int64, len(values))
	for _, v := range values {
		if v.ID != MaterialRecycleScrap || v.Quantity < 0 || v.Quantity > 1_000_000_000 {
			return nil, false
		}
		if _, exists := out[v.ID]; exists {
			return nil, false
		}
		out[v.ID] = v.Quantity
	}
	return out, true
}

func cloneMaterials(values []Material) []Material {
	out := append([]Material(nil), values...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

type Receipt struct {
	OperationID         string
	PlayerID            string
	ItemInstanceID      string
	ItemTemplateID      string
	ItemRevision        int64
	RuleID              string
	RuleVersion         string
	RequestedAt         time.Time
	MaterialsAwarded    []Material
	ReputationAwarded   int64
	FBAwarded           int64
	ContributionAwarded int64
	BlackIronAwarded    int64
	CreatedAt           time.Time
}

func (r Receipt) MaterialJSON() ([]byte, error) {
	return json.Marshal(cloneMaterials(r.MaterialsAwarded))
}

type ReconciliationReport struct {
	Checked  int
	Balanced bool
	Problems []string
}
