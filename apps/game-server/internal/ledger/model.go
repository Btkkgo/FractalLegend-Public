package ledger

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound            = errors.New("ledger record not found")
	ErrConflict            = errors.New("ledger record conflicts with existing state")
	ErrInvalidAccount      = errors.New("invalid ledger account")
	ErrInvalidAmount       = errors.New("invalid ledger amount")
	ErrAmountOverflow      = errors.New("ledger amount overflow")
	ErrInvalidReference    = errors.New("invalid ledger reference")
	ErrReferenceConflict   = errors.New("ledger reference conflicts with another intent")
	ErrInvalidTransfer     = errors.New("INVALID_TRANSFER")
	ErrInsufficientBalance = errors.New("insufficient FB balance")
	ErrUnbalanced          = errors.New("ledger transaction is not balanced")
	ErrInvalidTransaction  = errors.New("invalid ledger transaction")
	ErrAlreadyCompensated  = errors.New("ledger transaction is already compensated")
)

type OwnerType string

type Currency string

const (
	OwnerPlayer OwnerType = "PLAYER"
	OwnerSystem OwnerType = "SYSTEM"
	CurrencyFB  Currency  = "FB"
)

type TransactionType string

const (
	TransactionPlayerTransfer TransactionType = "PLAYER_TRANSFER"
	TransactionSystemSpend    TransactionType = "SYSTEM_SPEND"
	TransactionRefund         TransactionType = "REFUND"
	TransactionReversal       TransactionType = "REVERSAL"
)

type Direction string

const (
	DirectionDebit  Direction = "DEBIT"
	DirectionCredit Direction = "CREDIT"
)

type TransactionStatus string

const StatusPosted TransactionStatus = "POSTED"

type SpendClassification string

const (
	SpendEligible    SpendClassification = "ELIGIBLE"
	SpendNonEligible SpendClassification = "NON_ELIGIBLE"
)

type LedgerReference struct {
	Type string
	ID   string
}

type LedgerAccount struct {
	ID        string
	OwnerID   string
	OwnerType OwnerType
	Currency  Currency
	Balance   int64
	Revision  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type LedgerEntry struct {
	ID            string
	TransactionID string
	AccountID     string
	EntryType     TransactionType
	Amount        int64
	Direction     Direction
	BalanceBefore int64
	BalanceAfter  int64
	CreatedAt     time.Time
}

type LedgerTransaction struct {
	ID                    string
	Type                  TransactionType
	Status                TransactionStatus
	Reference             LedgerReference
	OriginalTransactionID string
	SpendClassification   SpendClassification
	Reason                string
	Entries               []LedgerEntry
	CreatedAt             time.Time
}

type EntryDraft struct {
	ID        string
	AccountID string
	Amount    int64
}

type PostDraft struct {
	ID                    string
	Type                  TransactionType
	Reference             LedgerReference
	OriginalTransactionID string
	SpendClassification   SpendClassification
	Reason                string
	Entries               []EntryDraft
	CreatedAt             time.Time
}

type AuditEvent struct {
	Sequence      int64
	TransactionID string
	Reference     LedgerReference
	AccountIDs    []string
	Amount        int64
	Type          string
	Result        string
	OccurredAt    time.Time
}

type ReconciliationMismatch struct {
	AccountID    string
	Balance      int64
	EntriesTotal int64
}

type ReconciliationReport struct {
	Balanced   bool
	Mismatches []ReconciliationMismatch
}

type TotalSupplyReport struct {
	AccountTotal int64
	EntryTotal   int64
}

type Repository interface {
	CreateLedgerAccount(context.Context, LedgerAccount) (LedgerAccount, error)
	LoadLedgerAccount(context.Context, string) (LedgerAccount, error)
	PostLedgerTransaction(context.Context, PostDraft) (LedgerTransaction, error)
	LoadLedgerTransaction(context.Context, string) (LedgerTransaction, error)
	ReconcileLedger(context.Context) (ReconciliationReport, error)
	LedgerTotalSupply(context.Context) (TotalSupplyReport, error)
	LedgerAuditEvents(context.Context, string) ([]AuditEvent, error)
}
