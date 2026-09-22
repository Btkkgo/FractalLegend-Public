package systemspend

import (
	"context"
	"errors"
	"strings"
	"time"

	"fractallegend/game-server/internal/contribution"
)

var (
	ErrUnavailable       = errors.New("system spend repository unavailable")
	ErrInvalidIntent     = errors.New("invalid server-side system spend intent")
	ErrClientIntent      = errors.New("client-originated system spend intent is forbidden")
	ErrUnknownProducer   = errors.New("unknown system spend producer")
	ErrDisabledProducer  = errors.New("system spend producer is disabled")
	ErrUnknownRule       = errors.New("unknown contribution rule")
	ErrConflict          = errors.New("system spend operation conflicts with existing intent")
	ErrReferenceConflict = errors.New("system spend producer reference already used")
	ErrNotFound          = errors.New("system spend not found")
	ErrReconciliation    = errors.New("system spend reconciliation failed")
)

type ProducerType string

const (
	ProducerInternalTestEligible    ProducerType = "INTERNAL_TEST_ELIGIBLE"
	ProducerInternalTestNonEligible ProducerType = "INTERNAL_TEST_NON_ELIGIBLE"
	ProducerNPCService              ProducerType = "NPC_SERVICE"
	ProducerEquipmentUpgrade        ProducerType = "EQUIPMENT_UPGRADE"
	ProducerManufacturing           ProducerType = "MANUFACTURING"
	ProducerMiningToolCraft         ProducerType = "MINING_TOOL_CRAFT"
	ProducerOrdinalsActivation      ProducerType = "ORDINALS_ACTIVATION"
	ProducerShop                    ProducerType = "SHOP"
)

type Producer struct {
	Type                 ProducerType
	Eligible             bool
	AllowedRuleVersion   string
	Refundable           bool
	PartialRefundAllowed bool
	Active               bool
	Description          string
}

type Registry struct{ producers map[ProducerType]Producer }

// NewRegistry is an internal server configuration boundary. G15 can activate
// only its test producers; gameplay producers remain disabled even if misconfigured.
func NewRegistry(values []Producer) (Registry, error) {
	registry := Registry{producers: make(map[ProducerType]Producer, len(values))}
	for _, p := range values {
		if !knownProducer(p.Type) || strings.TrimSpace(p.Description) == "" {
			return Registry{}, ErrUnknownProducer
		}
		if _, exists := registry.producers[p.Type]; exists {
			return Registry{}, ErrConflict
		}
		if p.Active && p.Type != ProducerInternalTestEligible && p.Type != ProducerInternalTestNonEligible {
			return Registry{}, ErrDisabledProducer
		}
		if (p.Type == ProducerInternalTestEligible && !p.Eligible) ||
			(p.Type == ProducerInternalTestNonEligible && p.Eligible) {
			return Registry{}, ErrInvalidIntent
		}
		if p.PartialRefundAllowed && !p.Refundable {
			return Registry{}, ErrInvalidIntent
		}
		if p.Eligible && p.AllowedRuleVersion != contribution.RuleVersionV1 {
			return Registry{}, ErrUnknownRule
		}
		if !p.Eligible && (p.AllowedRuleVersion != "" || p.Refundable) {
			return Registry{}, ErrInvalidIntent
		}
		registry.producers[p.Type] = p
	}
	return registry, nil
}

func ProductionRegistry() Registry {
	values := []Producer{
		{Type: ProducerNPCService, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Description: "NPC services; disabled in G15"},
		{Type: ProducerEquipmentUpgrade, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Description: "Equipment upgrades; disabled in G15"},
		{Type: ProducerManufacturing, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Description: "Manufacturing; disabled in G15"},
		{Type: ProducerMiningToolCraft, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Description: "Mining tool crafting; disabled in G15"},
		{Type: ProducerOrdinalsActivation, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Description: "Ordinals activation; disabled in G15"},
		{Type: ProducerShop, Eligible: false, Description: "Shop; disabled in G15"},
	}
	registry, _ := NewRegistry(values)
	return registry
}

func knownProducer(value ProducerType) bool {
	switch value {
	case ProducerInternalTestEligible, ProducerInternalTestNonEligible, ProducerNPCService,
		ProducerEquipmentUpgrade, ProducerManufacturing, ProducerMiningToolCraft,
		ProducerOrdinalsActivation, ProducerShop:
		return true
	default:
		return false
	}
}

func (r Registry) Resolve(value ProducerType) (Producer, error) {
	producer, ok := r.producers[value]
	if !ok {
		return Producer{}, ErrUnknownProducer
	}
	if !producer.Active {
		return Producer{}, ErrDisabledProducer
	}
	return producer, nil
}

// Intent has no eligibility, Contribution amount, or rule-version inputs.
// A network JSON decoder must not turn client data into this server intent.
type Intent struct {
	OperationID       string
	PlayerID          string
	PlayerFBAccountID string
	SystemFBAccountID string
	ProducerType      ProducerType
	ProducerReference string
	FBAmount          int64
	Metadata          map[string]string
}

func (*Intent) UnmarshalJSON([]byte) error { return ErrClientIntent }

type ResolvedIntent struct {
	Intent
	Eligible             bool
	RuleVersion          string
	ContributionAmount   int64
	Refundable           bool
	PartialRefundAllowed bool
}

type RefundStatus string

const (
	RefundNone    RefundStatus = "NONE"
	RefundPartial RefundStatus = "PARTIAL"
	RefundFull    RefundStatus = "FULL"
)

type SystemSpend struct {
	ID                   string
	OperationID          string
	PlayerID             string
	PlayerFBAccountID    string
	SystemFBAccountID    string
	ProducerType         ProducerType
	ProducerReference    string
	FBAmount             int64
	Eligible             bool
	RuleVersion          string
	ContributionAmount   int64
	FBTransactionID      string
	ContributionEntryID  string
	RefundedAmount       int64
	RefundStatus         RefundStatus
	Refundable           bool
	PartialRefundAllowed bool
	Status               string
	Metadata             map[string]string
	CreatedAt            time.Time
	CompletedAt          time.Time
}

type ReconciliationMismatch struct {
	OperationID string
	Reason      string
}

type ReconciliationReport struct {
	Balanced   bool
	Checked    int
	Mismatches []ReconciliationMismatch
}

type Repository interface {
	PostSystemSpend(context.Context, ResolvedIntent) (SystemSpend, error)
	LoadSystemSpend(context.Context, string) (SystemSpend, error)
	ReconcileSystemSpends(context.Context) (ReconciliationReport, error)
}
