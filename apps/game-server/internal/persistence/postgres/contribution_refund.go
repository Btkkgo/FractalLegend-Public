package postgres

import (
	"context"
	"errors"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"github.com/jackc/pgx/v5"
)

// RefundContributionSystemSpend is the only G14 path allowed through the
// otherwise closed G13 FB refund gate. The FB and Contribution effects share
// one PostgreSQL commit. No client-controlled balance or debt is accepted.
func (s *Store) RefundContributionSystemSpend(ctx context.Context, request contribution.RefundRequest) (contribution.RefundResult, error) {
	if s == nil || s.pool == nil {
		return contribution.RefundResult{}, contribution.ErrUnavailable
	}
	for attempt := 0; attempt < 24; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if err != nil {
			return contribution.RefundResult{}, classifyContributionError(err)
		}
		result, err := s.refundContributionSystemSpendTx(ctx, tx, request)
		if err == nil {
			err = s.checkContributionFailure(contribution.FailureBeforeCommit)
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if err == nil {
			return result, nil
		}
		if !retryContributionTransaction(err) {
			if errors.Is(err, contribution.ErrManualReview) || errors.Is(err, contribution.ErrRuleVersion) {
				s.markContributionReview(ctx, request.PlayerID)
			}
			return contribution.RefundResult{}, classifyContributionError(err)
		}
		select {
		case <-ctx.Done():
			return contribution.RefundResult{}, ctx.Err()
		case <-time.After(time.Duration((attempt%8)+1) * time.Millisecond):
		}
	}
	return contribution.RefundResult{}, contribution.ErrRetryExhausted
}

func (s *Store) refundContributionSystemSpendTx(ctx context.Context, tx pgx.Tx, request contribution.RefundRequest) (contribution.RefundResult, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "contribution-refund:"+request.ReferenceID); err != nil {
		return contribution.RefundResult{}, err
	}
	// Reference lock precedes original spend lock, FB account locks, and finally
	// the Contribution account lock. Every refund follows this same order.
	prior, err := loadContributionRefundByReferenceTx(ctx, tx, request.ReferenceID)
	if err == nil {
		if prior.Compensation.OriginalFBTransactionID != request.OriginalFBTransactionID || prior.Compensation.PlayerID != request.PlayerID || prior.Compensation.Amount != request.Amount || prior.FBTransaction.Type != refundKind(request) || len(prior.FBTransaction.Entries) != 2 || prior.FBTransaction.Entries[0].AccountID != request.PlayerFBAccountID {
			return contribution.RefundResult{}, contribution.ErrRefundConflict
		}
		return prior, nil
	}
	if !errors.Is(err, contribution.ErrNotFound) {
		return contribution.RefundResult{}, err
	}
	var originalType ledger.TransactionType
	var originalClass ledger.SpendClassification
	var originalReference string
	if err = tx.QueryRow(ctx, `SELECT transaction_type,spend_classification,reference_type FROM fb_ledger_transactions WHERE transaction_id=$1 FOR UPDATE`, request.OriginalFBTransactionID).Scan(&originalType, &originalClass, &originalReference); err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	if originalType != ledger.TransactionSystemSpend || originalClass != ledger.SpendEligible || originalReference != "CONTRIBUTION_SYSTEM_SPEND" {
		return contribution.RefundResult{}, contribution.ErrInvalidRefund
	}
	var original contribution.Entry
	err = tx.QueryRow(ctx, `SELECT entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,debt_settled,balance_before,balance_after,rule_version,fb_transaction_id,created_at FROM contribution_entries WHERE fb_transaction_id=$1`, request.OriginalFBTransactionID).Scan(&original.ID, &original.PlayerID, &original.PlayerFBAccountID, &original.SystemFBAccountID, &original.Source, &original.SourceID, &original.EligibleSpend, &original.Amount, &original.DebtSettled, &original.BalanceBefore, &original.BalanceAfter, &original.RuleVersion, &original.FBTransactionID, &original.CreatedAt)
	if err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	if original.PlayerID != request.PlayerID || original.PlayerFBAccountID != request.PlayerFBAccountID {
		return contribution.RefundResult{}, contribution.ErrInvalidRefund
	}
	originalFB, err := loadLedgerTransactionTx(ctx, tx, "transaction_id=$1", original.FBTransactionID)
	if err != nil {
		return contribution.RefundResult{}, err
	}
	if len(originalFB.Entries) != 2 || originalFB.Entries[0].AccountID != original.PlayerFBAccountID || originalFB.Entries[0].Amount != -original.EligibleSpend || originalFB.Entries[1].AccountID != original.SystemFBAccountID || originalFB.Entries[1].Amount != original.EligibleSpend {
		return contribution.RefundResult{}, contribution.ErrManualReview
	}
	generated, err := (contribution.EligibilityPolicy{}).Evaluate(original.Source, original.EligibleSpend, original.RuleVersion)
	if err != nil {
		return contribution.RefundResult{}, err
	}
	if generated != original.Amount || original.EligibleSpend != original.Amount {
		return contribution.RefundResult{}, contribution.ErrManualReview
	}
	var refunded, compensated int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(p.amount),0)::bigint FROM fb_ledger_transactions t JOIN fb_ledger_entries p ON p.transaction_id=t.transaction_id AND p.account_id=$2 WHERE t.original_transaction_id=$1 AND t.transaction_type IN ('REFUND','REVERSAL')`, original.FBTransactionID, original.PlayerFBAccountID).Scan(&refunded); err != nil {
		return contribution.RefundResult{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint FROM contribution_compensations WHERE original_entry_id=$1`, original.ID).Scan(&compensated); err != nil {
		return contribution.RefundResult{}, err
	}
	if refunded != compensated || refunded < 0 || refunded > original.EligibleSpend {
		return contribution.RefundResult{}, contribution.ErrManualReview
	}
	// G15 snapshots refund permissions on the immutable spend. Historical G13
	// postings have no SystemSpend row and retain their G14 behavior.
	var refundable, partialAllowed bool
	policyErr := tx.QueryRow(ctx, `SELECT refundable,partial_refund_allowed FROM system_spends WHERE fb_transaction_id=$1`, original.FBTransactionID).Scan(&refundable, &partialAllowed)
	if policyErr != nil && !errors.Is(policyErr, pgx.ErrNoRows) {
		return contribution.RefundResult{}, policyErr
	}
	if policyErr == nil && (!refundable || (!partialAllowed && request.Amount != original.EligibleSpend-refunded)) {
		return contribution.RefundResult{}, contribution.ErrInvalidRefund
	}
	if request.Amount > original.EligibleSpend-refunded {
		return contribution.RefundResult{}, contribution.ErrOverRefund
	}
	// Check ownership before money moves; the Ledger helper locks both FB rows.
	playerFB, err := loadLedgerAccountRow(tx.QueryRow(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=$1`, original.PlayerFBAccountID))
	if err != nil {
		return contribution.RefundResult{}, err
	}
	systemFB, err := loadLedgerAccountRow(tx.QueryRow(ctx, `SELECT account_id,owner_id,owner_type,currency,balance,revision,created_at,updated_at FROM fb_ledger_accounts WHERE account_id=$1`, original.SystemFBAccountID))
	if err != nil {
		return contribution.RefundResult{}, err
	}
	if playerFB.OwnerID != request.PlayerID || playerFB.OwnerType != ledger.OwnerPlayer || systemFB.OwnerType != ledger.OwnerSystem {
		return contribution.RefundResult{}, contribution.ErrInvalidRefund
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	draft := ledger.PostDraft{ID: newContributionID("fb-transaction"), Type: refundKind(request), Reference: ledger.LedgerReference{Type: "CONTRIBUTION_REFUND", ID: request.ReferenceID}, OriginalTransactionID: original.FBTransactionID, SpendClassification: ledger.SpendEligible, CreatedAt: now, Entries: []ledger.EntryDraft{{ID: newContributionID("fb-entry"), AccountID: original.PlayerFBAccountID, Amount: request.Amount}, {ID: newContributionID("fb-entry"), AccountID: original.SystemFBAccountID, Amount: -request.Amount}}}
	if request.Reversal {
		draft.Reason = "G14 contribution spend reversal"
	}
	posted, err := s.postLedgerTransactionTxWithContributionRefund(ctx, tx, draft, true)
	if err != nil {
		return contribution.RefundResult{}, err
	}
	if err = s.checkContributionFailure(contribution.FailureAfterFBRefund); err != nil {
		return contribution.RefundResult{}, err
	}
	var account contribution.Account
	err = tx.QueryRow(ctx, `SELECT player_id,balance,recovery_debt,review_required,revision,created_at,updated_at FROM contribution_accounts WHERE player_id=$1 FOR UPDATE`, request.PlayerID).Scan(&account.PlayerID, &account.Balance, &account.RecoveryDebt, &account.ReviewRequired, &account.Revision, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	if account.ReviewRequired {
		return contribution.RefundResult{}, contribution.ErrManualReview
	}
	if err = verifyContributionPlayerTx(ctx, tx, account, pendingContributionRefund{OriginalID: original.FBTransactionID, Amount: request.Amount}); err != nil {
		return contribution.RefundResult{}, err
	}
	reversed := request.Amount
	if account.Balance < reversed {
		reversed = account.Balance
	}
	debtCreated := request.Amount - reversed
	debtAfter, err := ledger.AddAmount(account.RecoveryDebt, debtCreated)
	if err != nil {
		return contribution.RefundResult{}, contribution.ErrOverflow
	}
	comp := contribution.Compensation{ID: newContributionID("contribution-compensation"), OriginalEntryID: original.ID, OriginalFBTransactionID: original.FBTransactionID, FBTransactionID: posted.ID, PlayerID: request.PlayerID, ReferenceID: request.ReferenceID, Amount: request.Amount, AvailableReversed: reversed, DebtCreated: debtCreated, BalanceBefore: account.Balance, BalanceAfter: account.Balance - reversed, DebtBefore: account.RecoveryDebt, DebtAfter: debtAfter, RuleVersion: original.RuleVersion, CreatedAt: now}
	_, err = tx.Exec(ctx, `INSERT INTO contribution_compensations(compensation_id,original_entry_id,original_fb_transaction_id,fb_transaction_id,player_id,reference_id,amount,available_reversed,debt_created,balance_before,balance_after,debt_before,debt_after,rule_version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, comp.ID, comp.OriginalEntryID, comp.OriginalFBTransactionID, comp.FBTransactionID, comp.PlayerID, comp.ReferenceID, comp.Amount, comp.AvailableReversed, comp.DebtCreated, comp.BalanceBefore, comp.BalanceAfter, comp.DebtBefore, comp.DebtAfter, comp.RuleVersion, comp.CreatedAt)
	if err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	if err = s.checkContributionFailure(contribution.FailureAfterCompensation); err != nil {
		return contribution.RefundResult{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE contribution_accounts SET balance=$1,recovery_debt=$2,revision=revision+1,updated_at=$3 WHERE player_id=$4`, comp.BalanceAfter, comp.DebtAfter, now, account.PlayerID)
	if err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	// Return the persisted record so the first response and reference replay agree.
	return loadContributionRefundByReferenceTx(ctx, tx, request.ReferenceID)
}

func refundKind(request contribution.RefundRequest) ledger.TransactionType {
	if request.Reversal {
		return ledger.TransactionReversal
	}
	return ledger.TransactionRefund
}

func (s *Store) ContributionCompensations(ctx context.Context, playerID string) ([]contribution.Compensation, error) {
	if s == nil || s.pool == nil {
		return nil, contribution.ErrUnavailable
	}
	rows, err := s.pool.Query(ctx, `SELECT compensation_id,original_entry_id,original_fb_transaction_id,fb_transaction_id,player_id,reference_id,amount,available_reversed,debt_created,balance_before,balance_after,debt_before,debt_after,rule_version,created_at FROM contribution_compensations WHERE player_id=$1 ORDER BY created_at,compensation_id`, playerID)
	if err != nil {
		return nil, classifyContributionError(err)
	}
	defer rows.Close()
	var result []contribution.Compensation
	for rows.Next() {
		var c contribution.Compensation
		if err = rows.Scan(&c.ID, &c.OriginalEntryID, &c.OriginalFBTransactionID, &c.FBTransactionID, &c.PlayerID, &c.ReferenceID, &c.Amount, &c.AvailableReversed, &c.DebtCreated, &c.BalanceBefore, &c.BalanceAfter, &c.DebtBefore, &c.DebtAfter, &c.RuleVersion, &c.CreatedAt); err != nil {
			return nil, classifyContributionError(err)
		}
		c.CreatedAt = c.CreatedAt.UTC()
		result = append(result, c)
	}
	return result, classifyContributionError(rows.Err())
}

func loadContributionRefundByReferenceTx(ctx context.Context, tx pgx.Tx, reference string) (contribution.RefundResult, error) {
	var c contribution.Compensation
	err := tx.QueryRow(ctx, `SELECT compensation_id,original_entry_id,original_fb_transaction_id,fb_transaction_id,player_id,reference_id,amount,available_reversed,debt_created,balance_before,balance_after,debt_before,debt_after,rule_version,created_at FROM contribution_compensations WHERE reference_id=$1`, reference).Scan(&c.ID, &c.OriginalEntryID, &c.OriginalFBTransactionID, &c.FBTransactionID, &c.PlayerID, &c.ReferenceID, &c.Amount, &c.AvailableReversed, &c.DebtCreated, &c.BalanceBefore, &c.BalanceAfter, &c.DebtBefore, &c.DebtAfter, &c.RuleVersion, &c.CreatedAt)
	if err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	c.CreatedAt = c.CreatedAt.UTC()
	posted, err := loadLedgerTransactionTx(ctx, tx, "transaction_id=$1", c.FBTransactionID)
	if err != nil {
		return contribution.RefundResult{}, err
	}
	var total, originalAmount int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint FROM contribution_compensations WHERE original_entry_id=$1`, c.OriginalEntryID).Scan(&total); err != nil {
		return contribution.RefundResult{}, err
	}
	if err = tx.QueryRow(ctx, `SELECT eligible_spend FROM contribution_entries WHERE entry_id=$1`, c.OriginalEntryID).Scan(&originalAmount); err != nil {
		return contribution.RefundResult{}, classifyContributionError(err)
	}
	return contribution.RefundResult{Compensation: c, FBTransaction: posted, TotalRefunded: total, RemainingRefundable: originalAmount - total}, nil
}

// verifyContributionPlayerTx detects account drift and orphaned FB refund
// records before any sensitive operation. Callers hold the account row lock.
type pendingContributionRefund struct {
	OriginalID string
	Amount     int64
}

func verifyContributionPlayerTx(ctx context.Context, tx pgx.Tx, account contribution.Account, pendingRefund ...pendingContributionRefund) error {
	var earned, settled, consumed, reversed, debtCreated, refunded int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint,COALESCE(sum(debt_settled),0)::bigint FROM contribution_entries WHERE player_id=$1`, account.PlayerID).Scan(&earned, &settled); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint FROM contribution_consumptions WHERE player_id=$1`, account.PlayerID).Scan(&consumed); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(available_reversed),0)::bigint,COALESCE(sum(debt_created),0)::bigint FROM contribution_compensations WHERE player_id=$1`, account.PlayerID).Scan(&reversed, &debtCreated); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(p.amount),0)::bigint FROM fb_ledger_transactions t JOIN contribution_entries e ON e.fb_transaction_id=t.original_transaction_id JOIN fb_ledger_entries p ON p.transaction_id=t.transaction_id AND p.account_id=e.player_fb_account_id WHERE e.player_id=$1 AND t.transaction_type IN ('REFUND','REVERSAL')`, account.PlayerID).Scan(&refunded); err != nil {
		return err
	}
	var compensated int64
	if err := tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint FROM contribution_compensations WHERE player_id=$1`, account.PlayerID).Scan(&compensated); err != nil {
		return err
	}
	var pending int64
	var pendingOriginal string
	if len(pendingRefund) > 0 {
		pending = pendingRefund[0].Amount
		pendingOriginal = pendingRefund[0].OriginalID
	}
	if refunded != compensated+pending || account.Balance != earned-settled-consumed-reversed || account.RecoveryDebt != debtCreated-settled || (account.RecoveryDebt > 0 && account.Balance > 0) {
		return contribution.ErrManualReview
	}
	var broken bool
	err := tx.QueryRow(ctx, `SELECT EXISTS(
       SELECT 1 FROM contribution_entries e
       LEFT JOIN LATERAL (SELECT COALESCE(sum(amount),0)::bigint total FROM contribution_compensations WHERE original_entry_id=e.entry_id) c ON true
       LEFT JOIN LATERAL (SELECT COALESCE(sum(p.amount),0)::bigint total FROM fb_ledger_transactions t JOIN fb_ledger_entries p ON p.transaction_id=t.transaction_id AND p.account_id=e.player_fb_account_id WHERE t.original_transaction_id=e.fb_transaction_id AND t.transaction_type IN ('REFUND','REVERSAL')) f ON true
	       WHERE e.player_id=$1 AND (c.total>e.amount OR f.total>e.eligible_spend OR c.total+CASE WHEN e.fb_transaction_id=$2 THEN $3::bigint ELSE 0 END<>f.total OR e.rule_version<>'CONTRIBUTION_RULE_V1')
       UNION ALL
       SELECT 1 FROM contribution_compensations c JOIN contribution_entries e ON e.entry_id=c.original_entry_id JOIN fb_ledger_transactions t ON t.transaction_id=c.fb_transaction_id
       WHERE c.player_id=$1 AND (c.original_fb_transaction_id<>e.fb_transaction_id OR t.original_transaction_id<>e.fb_transaction_id OR t.reference_type<>'CONTRIBUTION_REFUND' OR t.reference_id<>c.reference_id OR c.rule_version<>e.rule_version)
	       )`, account.PlayerID, pendingOriginal, pending).Scan(&broken)
	if err != nil {
		return err
	}
	if broken {
		return contribution.ErrManualReview
	}
	return nil
}

func (s *Store) ValidateContributionSpend(ctx context.Context, playerID string, amount int64) error {
	if s == nil || s.pool == nil {
		return contribution.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var account contribution.Account
	err = tx.QueryRow(ctx, `SELECT player_id,balance,recovery_debt,review_required,revision,created_at,updated_at FROM contribution_accounts WHERE player_id=$1 FOR UPDATE`, playerID).Scan(&account.PlayerID, &account.Balance, &account.RecoveryDebt, &account.ReviewRequired, &account.Revision, &account.CreatedAt, &account.UpdatedAt)
	if err != nil {
		return classifyContributionError(err)
	}
	if account.ReviewRequired {
		return contribution.ErrManualReview
	}
	if err = verifyContributionPlayerTx(ctx, tx, account); err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, contribution.ErrManualReview) {
			s.markContributionReview(ctx, playerID)
		}
		return err
	}
	if err = contribution.ValidateAvailableSpend(account, amount); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
