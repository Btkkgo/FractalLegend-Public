package contribution

import (
	"context"
	"errors"
	"time"

	"fractallegend/game-server/internal/ledger"
)

var (
	ErrInvalidSource    = errors.New("invalid contribution source reference")
	ErrIneligibleSource = errors.New("source is not eligible for contribution")
	ErrSourceConflict   = errors.New("contribution source conflicts with existing posting")
	ErrNotFound         = errors.New("contribution record not found")
	ErrInvalidAccount   = errors.New("invalid contribution account")
	ErrOverflow         = errors.New("contribution balance overflow")
	ErrUnavailable      = errors.New("contribution repository unavailable")
	ErrRetryExhausted   = errors.New("contribution transaction retry exhausted")
	ErrInvalidRefund    = errors.New("invalid contribution refund")
	ErrOverRefund       = errors.New("contribution refund exceeds original spend")
	ErrRefundConflict   = errors.New("contribution refund reference conflicts with existing intent")
	ErrRecoveryHold     = errors.New("contribution recovery hold")
	ErrInsufficient     = errors.New("insufficient available contribution")
	ErrManualReview     = errors.New("contribution reconciliation requires manual review")
)

const (
	FailureBeforeFBDebit      = "before_fb_debit"
	FailureAfterFBDebit       = "after_fb_debit"
	FailureBeforeEntry        = "before_contribution_entry"
	FailureAfterEntry         = "after_contribution_entry"
	FailureAfterAccountUpdate = "after_contribution_account_update"
	FailureBeforeFinalState   = "before_final_state"
	FailureBeforeCommit       = "before_commit"
	FailureAfterFBRefund      = "after_fb_refund"
	FailureAfterCompensation  = "after_compensation"
)

type Account struct {
	PlayerID       string
	Balance        int64
	RecoveryDebt   int64
	ReviewRequired bool
	Revision       int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Entry struct {
	ID                string
	PlayerID          string
	PlayerFBAccountID string
	SystemFBAccountID string
	Source            Source
	SourceID          string
	EligibleSpend     int64
	Amount            int64
	DebtSettled       int64
	BalanceBefore     int64
	BalanceAfter      int64
	RuleVersion       string
	FBTransactionID   string
	CreatedAt         time.Time
}

type RefundRequest struct {
	OriginalFBTransactionID string
	PlayerID                string
	PlayerFBAccountID       string
	ReferenceID             string
	Amount                  int64
	Reversal                bool
}

type Compensation struct {
	ID                      string
	OriginalEntryID         string
	OriginalFBTransactionID string
	FBTransactionID         string
	PlayerID                string
	ReferenceID             string
	Amount                  int64
	AvailableReversed       int64
	DebtCreated             int64
	BalanceBefore           int64
	BalanceAfter            int64
	DebtBefore              int64
	DebtAfter               int64
	RuleVersion             string
	CreatedAt               time.Time
}

type RefundResult struct {
	Compensation        Compensation
	FBTransaction       ledger.LedgerTransaction
	TotalRefunded       int64
	RemainingRefundable int64
}

type SpendRequest struct {
	PlayerID          string
	PlayerFBAccountID string
	SystemFBAccountID string
	Source            Source
	SourceID          string
	EligibleSpend     int64
	RuleVersion       string
}

type PostingResult struct {
	Entry         Entry
	FBTransaction ledger.LedgerTransaction
}

type AuditEvent struct {
	Sequence        int64
	EntryID         string
	PlayerID        string
	Source          Source
	SourceID        string
	RuleVersion     string
	Amount          int64
	FBTransactionID string
	CreatedAt       time.Time
}

type ReconciliationMismatch struct {
	PlayerID     string
	Balance      int64
	EntriesTotal int64
}

type ReconciliationReport struct {
	Balanced   bool
	Mismatches []ReconciliationMismatch
}

type Repository interface {
	CreateContributionAccount(context.Context, string) (Account, error)
	LoadContributionAccount(context.Context, string) (Account, error)
	PostContributionSystemSpend(context.Context, SpendRequest) (PostingResult, error)
	RefundContributionSystemSpend(context.Context, RefundRequest) (RefundResult, error)
	ValidateContributionSpend(context.Context, string, int64) error
	ContributionEntries(context.Context, string) ([]Entry, error)
	ContributionCompensations(context.Context, string) ([]Compensation, error)
	ContributionAuditEvents(context.Context, string) ([]AuditEvent, error)
	ReconcileContribution(context.Context) (ReconciliationReport, error)
}
