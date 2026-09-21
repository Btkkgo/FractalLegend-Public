package postgres

import (
	"context"
	"errors"
	"math"

	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) CreateLedgerAccount(ctx context.Context, value ledger.LedgerAccount) (ledger.LedgerAccount, error) {
	if value.ID == "" || value.OwnerID == "" || (value.OwnerType != ledger.OwnerPlayer && value.OwnerType != ledger.OwnerSystem) || value.Currency != ledger.CurrencyFB || value.Balance != 0 {
		return ledger.LedgerAccount{}, ledger.ErrInvalidAccount
	}
	value.Balance = 0
	value.Revision = 1
	_, err := s.pool.Exec(ctx, `INSERT INTO fb_ledger_accounts(account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at) VALUES($1,$2,$3,$4,0,1,$5,$6)`, value.ID, value.OwnerID, value.OwnerType, value.Currency, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return ledger.LedgerAccount{}, classifyLedgerError(err)
	}
	return value, nil
}

func (s *Store) LoadLedgerAccount(ctx context.Context, id string) (ledger.LedgerAccount, error) {
	return loadLedgerAccountRow(s.pool.QueryRow(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=$1`, id))
}

func (s *Store) PostLedgerTransaction(ctx context.Context, draft ledger.PostDraft) (ledger.LedgerTransaction, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	defer tx.Rollback(ctx)
	result, err := s.postLedgerTransactionTx(ctx, tx, draft)
	if err != nil {
		return ledger.LedgerTransaction{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	return result, nil
}

func (s *Store) LoadLedgerTransaction(ctx context.Context, id string) (ledger.LedgerTransaction, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	defer tx.Rollback(ctx)
	value, err := loadLedgerTransactionTx(ctx, tx, `transaction_id=$1`, id)
	if err != nil {
		return ledger.LedgerTransaction{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	return value, nil
}

func (s *Store) ReconcileLedger(ctx context.Context) (ledger.ReconciliationReport, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.account_id,a.balance,coalesce(sum(e.amount),0)::bigint FROM fb_ledger_accounts a LEFT JOIN fb_ledger_entries e ON e.account_id=a.account_id GROUP BY a.account_id,a.balance HAVING a.balance <> coalesce(sum(e.amount),0)::bigint ORDER BY a.account_id`)
	if err != nil {
		return ledger.ReconciliationReport{}, classifyLedgerError(err)
	}
	defer rows.Close()
	report := ledger.ReconciliationReport{Balanced: true}
	for rows.Next() {
		var mismatch ledger.ReconciliationMismatch
		if err = rows.Scan(&mismatch.AccountID, &mismatch.Balance, &mismatch.EntriesTotal); err != nil {
			return ledger.ReconciliationReport{}, classifyLedgerError(err)
		}
		report.Balanced = false
		report.Mismatches = append(report.Mismatches, mismatch)
	}
	return report, classifyLedgerError(rows.Err())
}

func (s *Store) LedgerTotalSupply(ctx context.Context) (ledger.TotalSupplyReport, error) {
	var report ledger.TotalSupplyReport
	if err := s.pool.QueryRow(ctx, "SELECT coalesce(sum(balance),0)::bigint FROM fb_ledger_accounts").Scan(&report.AccountTotal); err != nil {
		return ledger.TotalSupplyReport{}, classifyLedgerError(err)
	}
	rows, err := s.pool.Query(ctx, `SELECT t.transaction_type,coalesce(sum(e.amount),0)::bigint FROM fb_ledger_transactions t JOIN fb_ledger_entries e ON e.transaction_id=t.transaction_id GROUP BY t.transaction_type`)
	if err != nil {
		return ledger.TotalSupplyReport{}, classifyLedgerError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var kind ledger.TransactionType
		var delta int64
		if err = rows.Scan(&kind, &delta); err != nil {
			return ledger.TotalSupplyReport{}, classifyLedgerError(err)
		}
		report.EntryTotal, err = ledger.AddAmount(report.EntryTotal, delta)
		if err != nil {
			return ledger.TotalSupplyReport{}, err
		}
		_ = kind
	}
	return report, classifyLedgerError(rows.Err())
}

func (s *Store) LedgerAuditEvents(ctx context.Context, transactionID string) ([]ledger.AuditEvent, error) {
	rows, err := s.pool.Query(ctx, `SELECT sequence,transaction_id,reference_type,reference_id,account_ids,amount,event_type,result,occurred_at FROM fb_ledger_audit_events WHERE transaction_id=$1 ORDER BY sequence`, transactionID)
	if err != nil {
		return nil, classifyLedgerError(err)
	}
	defer rows.Close()
	var result []ledger.AuditEvent
	for rows.Next() {
		var event ledger.AuditEvent
		if err = rows.Scan(&event.Sequence, &event.TransactionID, &event.Reference.Type, &event.Reference.ID, &event.AccountIDs, &event.Amount, &event.Type, &event.Result, &event.OccurredAt); err != nil {
			return nil, classifyLedgerError(err)
		}
		result = append(result, event)
	}
	if err = rows.Err(); err != nil {
		return nil, classifyLedgerError(err)
	}
	if len(result) == 0 {
		var exists bool
		if err = s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM fb_ledger_transactions WHERE transaction_id=$1)", transactionID).Scan(&exists); err != nil {
			return nil, classifyLedgerError(err)
		}
		if !exists {
			return nil, ledger.ErrNotFound
		}
	}
	return result, nil
}

type rowScanner interface {
	Scan(...any) error
}

func loadLedgerAccountRow(row rowScanner) (ledger.LedgerAccount, error) {
	var value ledger.LedgerAccount
	err := row.Scan(&value.ID, &value.OwnerID, &value.OwnerType, &value.Currency, &value.Balance, &value.Revision, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.LedgerAccount{}, ledger.ErrNotFound
	}
	return value, classifyLedgerError(err)
}

func loadLedgerByReferenceTx(ctx context.Context, tx pgx.Tx, reference ledger.LedgerReference) (ledger.LedgerTransaction, error) {
	return loadLedgerTransactionTx(ctx, tx, `reference_type=$1 AND reference_id=$2`, reference.Type, reference.ID)
}

func loadLedgerTransactionTx(ctx context.Context, tx pgx.Tx, predicate string, args ...any) (ledger.LedgerTransaction, error) {
	var value ledger.LedgerTransaction
	query := `SELECT transaction_id,transaction_type,status,reference_type,reference_id,coalesce(original_transaction_id,''),spend_classification,reason,created_at FROM fb_ledger_transactions WHERE ` + predicate
	err := tx.QueryRow(ctx, query, args...).Scan(&value.ID, &value.Type, &value.Status, &value.Reference.Type, &value.Reference.ID, &value.OriginalTransactionID, &value.SpendClassification, &value.Reason, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.LedgerTransaction{}, ledger.ErrNotFound
	}
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	rows, err := tx.Query(ctx, `SELECT entry_id,transaction_id,account_id,entry_type,amount,direction,balance_before,balance_after,created_at FROM fb_ledger_entries WHERE transaction_id=$1 ORDER BY entry_order`, value.ID)
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var entry ledger.LedgerEntry
		if err = rows.Scan(&entry.ID, &entry.TransactionID, &entry.AccountID, &entry.EntryType, &entry.Amount, &entry.Direction, &entry.BalanceBefore, &entry.BalanceAfter, &entry.CreatedAt); err != nil {
			return ledger.LedgerTransaction{}, classifyLedgerError(err)
		}
		value.Entries = append(value.Entries, entry)
	}
	return value, classifyLedgerError(rows.Err())
}

func sameLedgerIntent(existing ledger.LedgerTransaction, draft ledger.PostDraft) bool {
	if existing.Type != draft.Type || existing.Reference != draft.Reference || existing.OriginalTransactionID != draft.OriginalTransactionID || existing.SpendClassification != draft.SpendClassification || existing.Reason != draft.Reason || len(existing.Entries) != len(draft.Entries) {
		return false
	}
	for index := range draft.Entries {
		if existing.Entries[index].AccountID != draft.Entries[index].AccountID || existing.Entries[index].Amount != draft.Entries[index].Amount {
			return false
		}
	}
	return true
}

func accountIDsInEntryOrder(entries []ledger.LedgerEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.AccountID)
	}
	return result
}

func auditMagnitude(entries []ledger.LedgerEntry) int64 {
	var result int64
	for _, entry := range entries {
		candidate := entry.Amount
		if candidate < 0 {
			if candidate == math.MinInt64 {
				return math.MaxInt64
			}
			candidate = -candidate
		}
		if candidate > result {
			result = candidate
		}
	}
	return result
}

func ledgerEventType(value ledger.TransactionType) string {
	switch value {
	case ledger.TransactionPlayerTransfer:
		return "FB_TRANSFER_COMPLETED"
	case ledger.TransactionRefund:
		return "FB_REFUND"
	case ledger.TransactionReversal:
		return "FB_REVERSAL"
	default:
		return "FB_LEDGER_POSTED"
	}
}

func classifyLedgerError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "fb_ledger_transactions_reference_type_reference_id_key":
				return ledger.ErrReferenceConflict
			case "fb_ledger_single_compensation_idx":
				return ledger.ErrAlreadyCompensated
			default:
				return ledger.ErrConflict
			}
		}
	}
	return err
}

var _ ledger.Repository = (*Store)(nil)
