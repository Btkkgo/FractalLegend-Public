package emission

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnavailable = errors.New("black iron emission repository unavailable")
	ErrNotFound    = errors.New("authoritative G15 spend not found")
	ErrConflict    = errors.New("black iron emission source conflicts with persisted rule")
	ErrInvariant   = errors.New("black iron emission invariant failed")
)

const GlobalPoolID = "GLOBAL"

type Pool struct {
	ID                         string
	TotalEligibleSpendObserved int64
	TotalRefundedSpend         int64
	TotalEmissionCapacity      int64
	TotalReserved              int64
	TotalDistributed           int64
	RemainingCapacity          int64
	RecoveryDebt               int64
	RuleVersion                string
	Revision                   int64
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

type Entry struct {
	ID                  string
	SourceType          string
	SourceID            string
	SpendID             string
	EligibleSpendAmount int64
	EmissionAmount      int64
	RuleVersion         string
	CreatedAt           time.Time
	PoolRevision        int64
}

type Receipt struct {
	ID                string
	EntryID           string
	SourceID          string
	EligibleSpend     int64
	EmissionAdded     int64
	PoolBefore        int64
	PoolAfter         int64
	RemainingCapacity int64
	RuleVersion       string
	CreatedAt         time.Time
}

type Snapshot struct {
	Pool            Pool
	Entries         []Entry
	Receipts        []Receipt
	RecoveryEntries []RecoveryEntry
}

// RecoveryEntry is the immutable consequence of one pool mutation. Positive
// emission and cancelled reservations repay debt before exposing capacity.
type RecoveryEntry struct {
	ID               string
	SourceType       string
	SourceID         string
	EmissionEntryID  *string
	BlockEntryID     *string
	NetEmissionDelta int64
	ReservedDelta    int64
	RemainingBefore  int64
	RemainingAfter   int64
	DebtBefore       int64
	DebtAfter        int64
	PoolRevision     int64
	CreatedAt        time.Time
}

type ReconciliationReport struct {
	Balanced   bool
	Checked    int
	Mismatches []string
}

type Repository interface {
	ApplyBlackIronEmission(context.Context, string, string) (Receipt, error)
	LoadBlackIronEmission(context.Context, string) (Receipt, error)
	SnapshotBlackIronEmission(context.Context) (Snapshot, error)
	ReconcileBlackIronEmission(context.Context) (ReconciliationReport, error)
}

// Service is an internal server-side boundary. No network route is connected.
// The active rule comes from server configuration; G17 provides only a
// development rule and therefore no production emission activation.
type Service struct {
	repo              Repository
	activeRuleVersion string
}

func NewService(repo Repository, activeRuleVersion string) *Service {
	return &Service{repo: repo, activeRuleVersion: activeRuleVersion}
}

func (s *Service) Apply(ctx context.Context, spendID string) (Receipt, error) {
	if s == nil || s.repo == nil {
		return Receipt{}, ErrUnavailable
	}
	if _, err := Calculate(s.activeRuleVersion, 1); err != nil {
		return Receipt{}, err
	}
	if !validSourceID(spendID) {
		return Receipt{}, ErrNotFound
	}
	return s.repo.ApplyBlackIronEmission(ctx, spendID, s.activeRuleVersion)
}

func (s *Service) Load(ctx context.Context, spendID string) (Receipt, error) {
	if s == nil || s.repo == nil {
		return Receipt{}, ErrUnavailable
	}
	if !validSourceID(spendID) {
		return Receipt{}, ErrNotFound
	}
	return s.repo.LoadBlackIronEmission(ctx, spendID)
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	if s == nil || s.repo == nil {
		return Snapshot{}, ErrUnavailable
	}
	return s.repo.SnapshotBlackIronEmission(ctx)
}

func (s *Service) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	if s == nil || s.repo == nil {
		return ReconciliationReport{}, ErrUnavailable
	}
	return s.repo.ReconcileBlackIronEmission(ctx)
}

func validSourceID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}
