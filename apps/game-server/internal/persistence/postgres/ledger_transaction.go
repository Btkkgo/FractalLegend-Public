package postgres

import (
	"context"
	"errors"
	"sort"
	"strconv"

	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5"
)

// postLedgerTransactionTx posts an immutable ledger transaction inside an
// existing database transaction. G12 uses this entry point so item ownership,
// FB movements, the receipt, and the completed trade state share one commit.
func (s *Store) postLedgerTransactionTx(ctx context.Context, tx pgx.Tx, draft ledger.PostDraft) (ledger.LedgerTransaction, error) {
	return s.postLedgerTransactionTxWithContributionRefund(ctx, tx, draft, false)
}

// The override is private to the G14 atomic orchestrator. Public Ledger writes
// retain G13's fail-closed gate for eligible Contribution spends.
func (s *Store) postLedgerTransactionTxWithContributionRefund(ctx context.Context, tx pgx.Tx, draft ledger.PostDraft, contributionRefund bool) (ledger.LedgerTransaction, error) {
	if err := ledger.ValidatePostDraft(draft); err != nil {
		return ledger.LedgerTransaction{}, err
	}
	lockKey := strconv.Itoa(len(draft.Reference.Type)) + ":" + draft.Reference.Type + draft.Reference.ID
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", lockKey); err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	existing, err := loadLedgerByReferenceTx(ctx, tx, draft.Reference)
	if err == nil {
		if !sameLedgerIntent(existing, draft) {
			return ledger.LedgerTransaction{}, ledger.ErrReferenceConflict
		}
		return existing, nil
	}
	if !errors.Is(err, ledger.ErrNotFound) {
		return ledger.LedgerTransaction{}, err
	}
	if draft.OriginalTransactionID != "" {
		var original, originalReferenceType string
		if err = tx.QueryRow(ctx, "SELECT transaction_id,reference_type FROM fb_ledger_transactions WHERE transaction_id=$1 FOR UPDATE", draft.OriginalTransactionID).Scan(&original, &originalReferenceType); errors.Is(err, pgx.ErrNoRows) {
			return ledger.LedgerTransaction{}, ledger.ErrNotFound
		} else if err != nil {
			return ledger.LedgerTransaction{}, classifyLedgerError(err)
		}
		// G14 refunds require the private atomic Contribution coordinator.
		// All other shared Ledger callers fail closed for G13-linked spends.
		if originalReferenceType == "CONTRIBUTION_SYSTEM_SPEND" && (!contributionRefund || draft.Reference.Type != "CONTRIBUTION_REFUND" || (draft.Type != ledger.TransactionRefund && draft.Type != ledger.TransactionReversal)) {
			return ledger.LedgerTransaction{}, ledger.ErrInvalidTransaction
		}
		if contributionRefund && originalReferenceType != "CONTRIBUTION_SYSTEM_SPEND" { return ledger.LedgerTransaction{}, ledger.ErrInvalidTransaction }
		if !contributionRefund {
			var compensated bool
			if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM fb_ledger_transactions WHERE transaction_type IN ('REFUND','REVERSAL') AND original_transaction_id=$1)", draft.OriginalTransactionID).Scan(&compensated); err != nil {
				return ledger.LedgerTransaction{}, classifyLedgerError(err)
			}
			if compensated { return ledger.LedgerTransaction{}, ledger.ErrAlreadyCompensated }
		}
	}

	accountIDs := make([]string, 0, len(draft.Entries))
	for _, entry := range draft.Entries {
		accountIDs = append(accountIDs, entry.AccountID)
	}
	sort.Strings(accountIDs)
	rows, err := tx.Query(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=ANY($1) ORDER BY account_id FOR UPDATE`, accountIDs)
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	accounts := map[string]ledger.LedgerAccount{}
	for rows.Next() {
		var value ledger.LedgerAccount
		if err = rows.Scan(&value.ID, &value.OwnerID, &value.OwnerType, &value.Currency, &value.Balance, &value.Revision, &value.CreatedAt, &value.UpdatedAt); err != nil {
			rows.Close()
			return ledger.LedgerTransaction{}, classifyLedgerError(err)
		}
		accounts[value.ID] = value
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	rows.Close()
	if len(accounts) != len(draft.Entries) {
		return ledger.LedgerTransaction{}, ledger.ErrNotFound
	}

	balances := map[string]int64{}
	for _, entry := range draft.Entries {
		next, addErr := ledger.AddAmount(accounts[entry.AccountID].Balance, entry.Amount)
		if addErr != nil {
			return ledger.LedgerTransaction{}, addErr
		}
		if next < 0 {
			return ledger.LedgerTransaction{}, ledger.ErrInsufficientBalance
		}
		balances[entry.AccountID] = next
	}
	var original any
	if draft.OriginalTransactionID != "" {
		original = draft.OriginalTransactionID
	}
	_, err = tx.Exec(ctx, `INSERT INTO fb_ledger_transactions(transaction_id,transaction_type,status,reference_type,reference_id,original_transaction_id,spend_classification,reason,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, draft.ID, draft.Type, ledger.StatusPosted, draft.Reference.Type, draft.Reference.ID, original, draft.SpendClassification, draft.Reason, draft.CreatedAt)
	if err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	result := ledger.LedgerTransaction{ID: draft.ID, Type: draft.Type, Status: ledger.StatusPosted, Reference: draft.Reference, OriginalTransactionID: draft.OriginalTransactionID, SpendClassification: draft.SpendClassification, Reason: draft.Reason, CreatedAt: draft.CreatedAt}
	for index, item := range draft.Entries {
		account := accounts[item.AccountID]
		direction := ledger.DirectionCredit
		if item.Amount < 0 {
			direction = ledger.DirectionDebit
		}
		entry := ledger.LedgerEntry{ID: item.ID, TransactionID: draft.ID, AccountID: item.AccountID, EntryType: draft.Type, Amount: item.Amount, Direction: direction, BalanceBefore: account.Balance, BalanceAfter: balances[item.AccountID], CreatedAt: draft.CreatedAt}
		if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_entries(entry_id,transaction_id,entry_order,account_id,entry_type,amount,direction,balance_before,balance_after,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, entry.ID, entry.TransactionID, index, entry.AccountID, entry.EntryType, entry.Amount, entry.Direction, entry.BalanceBefore, entry.BalanceAfter, entry.CreatedAt); err != nil {
			return ledger.LedgerTransaction{}, classifyLedgerError(err)
		}
		if _, err = tx.Exec(ctx, `UPDATE fb_ledger_accounts SET balance=$1,revision=revision+1,updated_at=$2 WHERE account_id=$3`, entry.BalanceAfter, draft.CreatedAt, entry.AccountID); err != nil {
			return ledger.LedgerTransaction{}, classifyLedgerError(err)
		}
		result.Entries = append(result.Entries, entry)
		if index == 0 && s.ledgerFailureInjector != nil {
			if err = s.ledgerFailureInjector(ledger.FailureAfterFirstEntry); err != nil {
				return ledger.LedgerTransaction{}, err
			}
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO fb_ledger_audit_events(transaction_id,reference_type,reference_id,account_ids,amount,event_type,result,occurred_at) VALUES($1,$2,$3,$4,$5,$6,'POSTED',$7)`, result.ID, result.Reference.Type, result.Reference.ID, accountIDsInEntryOrder(result.Entries), auditMagnitude(result.Entries), ledgerEventType(result.Type), result.CreatedAt); err != nil {
		return ledger.LedgerTransaction{}, classifyLedgerError(err)
	}
	return result, nil
}
