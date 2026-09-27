// Package miningpower owns TEST / NON-PRODUCTION mining power facts. Power is
// neither an asset nor a reward. No client route or economy writer is exposed.
package miningpower

import (
	"context"
	"errors"
	"math"
	"time"
)

const (
	DevelopmentRuleVersion       = "DEV_G20_MINING_POWER_V1"
	SyntheticKind                = "TEST_NON_PRODUCTION"
	SyntheticEvidence            = "SYNTHETIC_TEST"
	AlgorithmID                  = "INTEGER_PRODUCT_FLOOR_ONCE_V1"
	Scale                  int64 = 1000000
	MaxMultiplier          int64 = 1000000000
	MaxPower               int64 = math.MaxInt64
	StatusValid                  = "VALID"
	StatusInvalid                = "INVALID"
	StatusDuplicate              = "DUPLICATE"
	StatusExpired                = "EXPIRED"
	StatusReplayed               = "REPLAYED"
	StatusNotEligible            = "NOT_ELIGIBLE"
	DecisionAccepted             = "ACCEPTED"
)

var (
	ErrInvalidIntent        = errors.New("invalid mining action intent")
	ErrClientPowerForbidden = errors.New("client mining power forbidden")
	ErrInvalidInput         = errors.New("invalid authoritative mining input")
	ErrUnknownRule          = errors.New("mining power rule not approved")
	ErrOverflow             = errors.New("mining power overflow")
	ErrUnavailable          = errors.New("mining power repository unavailable")
	ErrInvariant            = errors.New("mining power invariant failed")
	ErrRetryExhausted       = errors.New("mining power retries exhausted")
)

type Principal struct{ AccountID, PlayerID string }
type ActionIntent struct{ ActivityID, SourceEventID, ActivitySessionID, BlockID, BlockInstanceID string }
type ValidatedInputs struct{ BasePowerUnits, EfficiencyScaled, ActivityWeightScaled, MapModifierScaled int64 }
type RuleManifest struct {
	Version, Kind, AlgorithmID                   string
	Scale, MaxBasePower, MaxMultiplier, MaxPower int64
}

func DevelopmentManifest() RuleManifest {
	return RuleManifest{DevelopmentRuleVersion, SyntheticKind, AlgorithmID, Scale, MaxPower, MaxMultiplier, MaxPower}
}

type ToolProfile struct {
	Reference, RuleVersion, Kind     string
	BasePowerUnits, EfficiencyScaled int64
}
type MapProfile struct {
	Reference, RuleVersion, Kind string
	ModifierScaled               int64
}
type ActivitySession struct {
	ID, PlayerID, AccountID, BlockID, ToolReference, MapReference, RuleVersion, State string
	BlockInstanceID                                                                   string
	OpenedAt, ExpiresAt, CreatedAt, UpdatedAt                                         time.Time
}
type SourceEvent struct {
	ID, ActivityID, PlayerID, BlockID, ActivitySessionID, ToolReference, MapReference, RuleVersion string
	BlockInstanceID                                                                                string
	ObservedAt, ExpiresAt                                                                          time.Time
	ActivityWeightScaled                                                                           int64
	ServerEligibility, EvidenceKind                                                                string
}
type BlockContext struct {
	ID, Status                                                           string
	StartedAt, ScheduledEndAt                                            time.Time
	BlockInstanceID, CreateCommandID, RuleVersion, SourceEvidenceVersion string
	Height                                                               int64
	ValidatedAt                                                          time.Time
}

// MiningBlockValidationSnapshot is G20-owned historical evidence, captured by
// ordinary SELECT before the acceptance transaction. It is never refreshed.
type MiningBlockValidationSnapshot struct {
	BlockInstanceID, BlockID, CreateCommandID, Status, G18RuleVersion, G20RuleVersion, SourceEvidenceVersion string
	BlockHeight                                                                                              int64
	StartedAt, ScheduledEndAt, ValidatedAt                                                                   time.Time
}
type ValidatedMiningActivity struct {
	ActivityID, SourceEventID, PlayerID, BlockID, ActivitySessionID, ToolReference, MapReference, RuleVersion string
	BlockInstanceID                                                                                           string
	ValidationSnapshot                                                                                        MiningBlockValidationSnapshot
	Inputs                                                                                                    ValidatedInputs
	ValidatedPower                                                                                            int64
	AcceptedAt, ObservedAt, ExpiresAt                                                                         time.Time
	Decision, Status                                                                                          string
}
type MiningParticipant struct {
	PlayerID, BlockID, ActivitySessionID, RuleVersion, ToolReference, MapReference string
	BlockInstanceID                                                                string
	ValidatedPower, ActivityCount                                                  int64
	CreatedAt, UpdatedAt                                                           time.Time
}
type ValidationResult struct {
	Status, ReasonCode string
	AppliedPower       int64
	Original           *ValidatedMiningActivity
}
type Snapshot struct {
	Rules        []RuleManifest
	Tools        []ToolProfile
	Maps         []MapProfile
	Sessions     []ActivitySession
	Sources      []SourceEvent
	Blocks       []BlockContext
	Activities   []ValidatedMiningActivity
	Participants []MiningParticipant
}
type ReconciliationFinding struct {
	Identity, Field, Status string
	Expected, Actual, Delta *string
}
type ReconciliationReport struct {
	Status      string
	Checked     int
	Findings    []ReconciliationFinding
	Comparisons []ReconciliationFinding
}
type RebuildResult struct {
	Status       string
	Participants []MiningParticipant
	Findings     []ReconciliationFinding
}
type Repository interface {
	ValidateMiningActivity(context.Context, Principal, ActionIntent, string) (ValidationResult, error)
	SnapshotMiningPower(context.Context, string, string) (Snapshot, error)
	ReconcileMiningPower(context.Context, string, string) (ReconciliationReport, error)
	TotalValidatedPower(context.Context, string, string) (int64, error)
}
type MiningToolEligibility interface {
	ResolveTool(context.Context, string, string) (ToolProfile, error)
}
type MiningMapEligibility interface {
	ResolveMap(context.Context, string, string) (MapProfile, error)
}

// MiningBlockContext exposes only the upstream identity, window and status.
type MiningBlockContext interface {
	ReadBlock(context.Context, string) (BlockContext, error)
}
