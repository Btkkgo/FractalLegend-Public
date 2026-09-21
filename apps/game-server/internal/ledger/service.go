package ledger

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"time"
)

type Options struct {
	Now   func() time.Time
	NewID func(string) string
	Emit  func(AuditEvent)
}

type Service struct {
	repo  Repository
	now   func() time.Time
	newID func(string) string
	emit  func(AuditEvent)
}

func NewService(repo Repository, options Options) *Service {
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	if options.NewID == nil {
		options.NewID = secureID
	}
	return &Service{repo: repo, now: options.Now, newID: options.NewID, emit: options.Emit}
}

func (s *Service) CreatePlayerAccount(ctx context.Context, accountID, ownerID string) (LedgerAccount, error) {
	return s.createAccount(ctx, accountID, ownerID, OwnerPlayer)
}

func (s *Service) CreateSystemAccount(ctx context.Context, accountID, ownerID string) (LedgerAccount, error) {
	return s.createAccount(ctx, accountID, ownerID, OwnerSystem)
}

func (s *Service) createAccount(ctx context.Context, accountID, ownerID string, ownerType OwnerType) (LedgerAccount, error) {
	if s == nil || s.repo == nil || accountID == "" || ownerID == "" {
		return LedgerAccount{}, ErrInvalidAccount
	}
	now := s.now()
	return s.repo.CreateLedgerAccount(ctx, LedgerAccount{ID: accountID, OwnerID: ownerID, OwnerType: ownerType, Currency: CurrencyFB, CreatedAt: now, UpdatedAt: now})
}

func (s *Service) Balance(ctx context.Context, accountID string) (int64, error) {
	account, err := s.repo.LoadLedgerAccount(ctx, accountID)
	return account.Balance, err
}

func (s *Service) Transaction(ctx context.Context, transactionID string) (LedgerTransaction, error) {
	return s.repo.LoadLedgerTransaction(ctx, transactionID)
}

func (s *Service) Reconcile(ctx context.Context) (ReconciliationReport, error) {
	report, err := s.repo.ReconcileLedger(ctx)
	if err == nil && !report.Balanced && s.emit != nil {
		accountIDs := make([]string, 0, len(report.Mismatches))
		for _, mismatch := range report.Mismatches {
			accountIDs = append(accountIDs, mismatch.AccountID)
		}
		s.emit(AuditEvent{AccountIDs: accountIDs, Type: "FB_RECONCILIATION_MISMATCH", Result: "MISMATCH", OccurredAt: s.now()})
	}
	return report, err
}

func (s *Service) TotalSupply(ctx context.Context) (TotalSupplyReport, error) {
	return s.repo.LedgerTotalSupply(ctx)
}

func (s *Service) AuditEvents(ctx context.Context, transactionID string) ([]AuditEvent, error) {
	return s.repo.LedgerAuditEvents(ctx, transactionID)
}

func (s *Service) TransferFB(ctx context.Context, fromAccountID, toAccountID string, amount int64, reference LedgerReference) (LedgerTransaction, error) {
	if amount <= 0 {
		return LedgerTransaction{}, ErrInvalidAmount
	}
	if fromAccountID == "" || toAccountID == "" || fromAccountID == toAccountID {
		return LedgerTransaction{}, ErrInvalidTransfer
	}
	if err := validateReference(reference); err != nil {
		return LedgerTransaction{}, err
	}
	from, err := s.repo.LoadLedgerAccount(ctx, fromAccountID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	to, err := s.repo.LoadLedgerAccount(ctx, toAccountID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	if from.OwnerType != OwnerPlayer || to.OwnerType != OwnerPlayer {
		return LedgerTransaction{}, ErrInvalidTransfer
	}
	return s.post(ctx, TransactionPlayerTransfer, reference, "", "", "", []EntryDraft{{AccountID: fromAccountID, Amount: -amount}, {AccountID: toAccountID, Amount: amount}})
}

func (s *Service) SystemSpend(ctx context.Context, playerAccountID, systemAccountID string, amount int64, classification SpendClassification, reference LedgerReference) (LedgerTransaction, error) {
	if amount <= 0 {
		return LedgerTransaction{}, ErrInvalidAmount
	}
	if classification != SpendEligible && classification != SpendNonEligible {
		return LedgerTransaction{}, ErrInvalidTransaction
	}
	if err := validateReference(reference); err != nil {
		return LedgerTransaction{}, err
	}
	player, err := s.repo.LoadLedgerAccount(ctx, playerAccountID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	system, err := s.repo.LoadLedgerAccount(ctx, systemAccountID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	if player.OwnerType != OwnerPlayer || system.OwnerType != OwnerSystem {
		return LedgerTransaction{}, ErrInvalidTransaction
	}
	return s.post(ctx, TransactionSystemSpend, reference, "", classification, "", []EntryDraft{{AccountID: playerAccountID, Amount: -amount}, {AccountID: systemAccountID, Amount: amount}})
}

func (s *Service) Refund(ctx context.Context, originalTransactionID string, reference LedgerReference) (LedgerTransaction, error) {
	if err := validateReference(reference); err != nil {
		return LedgerTransaction{}, err
	}
	original, err := s.repo.LoadLedgerTransaction(ctx, originalTransactionID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	if original.Type != TransactionSystemSpend || len(original.Entries) != 2 {
		return LedgerTransaction{}, ErrInvalidTransaction
	}
	return s.post(ctx, TransactionRefund, reference, original.ID, original.SpendClassification, "", reverseEntries(original.Entries))
}

func (s *Service) ReverseTransaction(ctx context.Context, originalTransactionID, reason string, reference LedgerReference) (LedgerTransaction, error) {
	if err := validateReference(reference); err != nil {
		return LedgerTransaction{}, err
	}
	if reason == "" {
		return LedgerTransaction{}, ErrInvalidTransaction
	}
	original, err := s.repo.LoadLedgerTransaction(ctx, originalTransactionID)
	if err != nil {
		return LedgerTransaction{}, err
	}
	if original.Type == TransactionRefund || original.Type == TransactionReversal || original.Status != StatusPosted {
		return LedgerTransaction{}, ErrInvalidTransaction
	}
	return s.post(ctx, TransactionReversal, reference, original.ID, original.SpendClassification, reason, reverseEntries(original.Entries))
}

func (s *Service) post(ctx context.Context, kind TransactionType, reference LedgerReference, originalID string, classification SpendClassification, reason string, entries []EntryDraft) (LedgerTransaction, error) {
	now := s.now()
	for i := range entries {
		entries[i].ID = s.newID("fb-entry")
	}
	draft := PostDraft{ID: s.newID("fb-transaction"), Type: kind, Reference: reference, OriginalTransactionID: originalID, SpendClassification: classification, Reason: reason, Entries: entries, CreatedAt: now}
	s.emitDraft(draft, "FB_TRANSACTION_CREATED", "CREATED")
	result, err := s.repo.PostLedgerTransaction(ctx, draft)
	if err != nil {
		s.emitDraft(draft, "FB_TRANSACTION_FAILED", "FAILED")
		return LedgerTransaction{}, err
	}
	eventType := "FB_TRANSACTION_COMPLETED"
	if kind == TransactionReversal {
		eventType = "FB_TRANSACTION_REVERSED"
	}
	s.emitDraft(draft, eventType, "POSTED")
	return result, nil
}

func (s *Service) emitDraft(draft PostDraft, kind, result string) {
	if s.emit == nil {
		return
	}
	accountIDs := make([]string, 0, len(draft.Entries))
	var amount int64
	for _, entry := range draft.Entries {
		accountIDs = append(accountIDs, entry.AccountID)
		candidate := entry.Amount
		if candidate < 0 {
			if candidate == -1<<63 {
				candidate = 1<<63 - 1
			} else {
				candidate = -candidate
			}
		}
		if candidate > amount {
			amount = candidate
		}
	}
	s.emit(AuditEvent{TransactionID: draft.ID, Reference: draft.Reference, AccountIDs: accountIDs, Amount: amount, Type: kind, Result: result, OccurredAt: draft.CreatedAt})
}

func reverseEntries(entries []LedgerEntry) []EntryDraft {
	result := make([]EntryDraft, 0, len(entries))
	for _, entry := range entries {
		result = append(result, EntryDraft{AccountID: entry.AccountID, Amount: -entry.Amount})
	}
	return result
}

func validateReference(reference LedgerReference) error {
	if reference.Type == "" || reference.ID == "" {
		return ErrInvalidReference
	}
	return nil
}

func secureID(prefix string) string {
	var raw [16]byte
	if _, err := cryptorand.Read(raw[:]); err != nil {
		panic("secure random source unavailable")
	}
	return prefix + "-" + hex.EncodeToString(raw[:])
}
