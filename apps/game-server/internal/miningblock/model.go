package miningblock

import (
	"context"
	"errors"
	"time"
)

const (
	DevelopmentRuleVersion       = "DEV_G18_FIXED_BLOCK_REWARD"
	DevelopmentReward      int64 = 10
	DevelopmentDuration          = time.Minute
	StatusOpen                   = "OPEN"
	StatusFinalized              = "FINALIZED"
	StatusCancelled              = "CANCELLED"
	ActionOpen                   = "OPEN"
	ActionFinalize               = "FINALIZE"
	ActionCancel                 = "CANCEL"
)

var (
	ErrUnavailable          = errors.New("mining block repository unavailable")
	ErrUnknownRule          = errors.New("mining block rule is not approved")
	ErrInvalidCommand       = errors.New("invalid mining block command")
	ErrInsufficientCapacity = errors.New("insufficient emission capacity")
	ErrConflict             = errors.New("mining block command conflicts with persisted state")
	ErrNotFound             = errors.New("mining block not found")
	ErrInvariant            = errors.New("mining block invariant failed")
	ErrOverflow             = errors.New("mining block overflow")
	ErrRecoveryDebt         = errors.New("emission recovery debt blocks new reservation")
)

type Block struct {
	ID                        string
	Height                    int64
	CreateCommandID           string
	Status                    string
	RuleVersion               string
	StartedAt                 time.Time
	ScheduledEndAt            time.Time
	FinalizedAt               *time.Time
	CancelledAt               *time.Time
	RewardReserved            int64
	RewardReleased            int64
	RewardReturned            int64
	PoolRevisionAtReservation int64
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type Reservation struct {
	BlockID   string
	Amount    int64
	Released  int64
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Entry struct {
	ID                string
	BlockID           string
	BlockHeight       int64
	Action            string
	CapacityDelta     int64
	PoolBefore        int64
	PoolAfter         int64
	BlockStatusBefore string
	BlockStatusAfter  string
	RuleVersion       string
	CreatedAt         time.Time
}

type Receipt struct {
	ID             string
	BlockID        string
	BlockHeight    int64
	Action         string
	RewardReserved int64
	PoolBefore     int64
	PoolAfter      int64
	BlockStatus    string
	RuleVersion    string
	CreatedAt      time.Time
}

type Snapshot struct {
	Blocks       []Block
	Reservations []Reservation
	Entries      []Entry
	Receipts     []Receipt
}

type ReconciliationReport struct {
	Balanced   bool
	Checked    int
	Mismatches []string
}

type Repository interface {
	CreateMiningBlock(context.Context, string, string) (Receipt, error)
	FinalizeMiningBlock(context.Context, string) (Receipt, error)
	CancelMiningBlock(context.Context, string) (Receipt, error)
	LoadMiningBlock(context.Context, string) (Block, error)
	SnapshotMiningBlocks(context.Context) (Snapshot, error)
	ReconcileMiningBlocks(context.Context) (ReconciliationReport, error)
}

// Service is internal and has no gameplay or client route. Production has no
// approved reward or duration rule, so every production command fails closed.
type Service struct {
	repo        Repository
	ruleVersion string
	production  bool
}

func NewService(repo Repository, ruleVersion string, production bool) *Service {
	return &Service{repo: repo, ruleVersion: ruleVersion, production: production}
}

func (s *Service) checkRule() error {
	if s == nil || s.repo == nil {
		return ErrUnavailable
	}
	if s.production || s.ruleVersion != DevelopmentRuleVersion {
		return ErrUnknownRule
	}
	return nil
}

func (s *Service) Create(ctx context.Context, commandID string) (Receipt, error) {
	if err := s.checkRule(); err != nil {
		return Receipt{}, err
	}
	if !validID(commandID) {
		return Receipt{}, ErrInvalidCommand
	}
	return s.repo.CreateMiningBlock(ctx, commandID, s.ruleVersion)
}

func (s *Service) Finalize(ctx context.Context, blockID string) (Receipt, error) {
	if err := s.checkRule(); err != nil {
		return Receipt{}, err
	}
	if !validID(blockID) {
		return Receipt{}, ErrInvalidCommand
	}
	return s.repo.FinalizeMiningBlock(ctx, blockID)
}

func (s *Service) Cancel(ctx context.Context, blockID string) (Receipt, error) {
	if err := s.checkRule(); err != nil {
		return Receipt{}, err
	}
	if !validID(blockID) {
		return Receipt{}, ErrInvalidCommand
	}
	return s.repo.CancelMiningBlock(ctx, blockID)
}

func (s *Service) Recover(ctx context.Context, blockID string) (Block, error) {
	if err := s.checkRule(); err != nil {
		return Block{}, err
	}
	if !validID(blockID) {
		return Block{}, ErrInvalidCommand
	}
	report, err := s.repo.ReconcileMiningBlocks(ctx)
	if err != nil {
		return Block{}, err
	}
	if !report.Balanced {
		return Block{}, ErrInvariant
	}
	return s.repo.LoadMiningBlock(ctx, blockID)
}

func validID(value string) bool {
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
