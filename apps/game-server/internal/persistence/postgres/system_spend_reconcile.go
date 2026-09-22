package postgres

import (
	"context"
	"fmt"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/systemspend"
	"github.com/jackc/pgx/v5"
)

func validateSystemSpendTx(ctx context.Context, tx pgx.Tx, spend systemspend.SystemSpend) error {
	var kind ledger.TransactionType
	var status ledger.TransactionStatus
	var class ledger.SpendClassification
	var refType, refID string
	err := tx.QueryRow(ctx, `SELECT transaction_type,status,spend_classification,reference_type,reference_id FROM fb_ledger_transactions WHERE transaction_id=$1`, spend.FBTransactionID).Scan(&kind, &status, &class, &refType, &refID)
	if err != nil {
		return systemspend.ErrReconciliation
	}
	expectedClass := ledger.SpendNonEligible
	expectedType := "SYSTEM_SPEND_NON_ELIGIBLE"
	expectedID := g15SourcePrefix + spend.OperationID
	if spend.Eligible {
		expectedClass = ledger.SpendEligible
		expectedType = "CONTRIBUTION_SYSTEM_SPEND"
		expectedID = string(contribution.SourceSystemService) + ":" + g15SourcePrefix + spend.OperationID + ":" + spend.RuleVersion
	}
	if kind != ledger.TransactionSystemSpend || status != ledger.StatusPosted || class != expectedClass || refType != expectedType || refID != expectedID {
		return systemspend.ErrReconciliation
	}
	rows, err := tx.Query(ctx, `SELECT account_id,amount FROM fb_ledger_entries WHERE transaction_id=$1 ORDER BY entry_order`, spend.FBTransactionID)
	if err != nil {
		return err
	}
	type entry struct {
		account string
		amount  int64
	}
	var entries []entry
	for rows.Next() {
		var item entry
		if err = rows.Scan(&item.account, &item.amount); err != nil {
			rows.Close()
			return err
		}
		entries = append(entries, item)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(entries) != 2 || entries[0].account != spend.PlayerFBAccountID || entries[0].amount != -spend.FBAmount || entries[1].account != spend.SystemFBAccountID || entries[1].amount != spend.FBAmount {
		return systemspend.ErrReconciliation
	}
	var owner string
	var playerType, systemType ledger.OwnerType
	if err = tx.QueryRow(ctx, `SELECT owner_id,owner_type FROM fb_ledger_accounts WHERE account_id=$1`, spend.PlayerFBAccountID).Scan(&owner, &playerType); err != nil {
		return systemspend.ErrReconciliation
	}
	if err = tx.QueryRow(ctx, `SELECT owner_type FROM fb_ledger_accounts WHERE account_id=$1`, spend.SystemFBAccountID).Scan(&systemType); err != nil {
		return systemspend.ErrReconciliation
	}
	if owner != spend.PlayerID || playerType != ledger.OwnerPlayer || systemType != ledger.OwnerSystem || spend.Status != "COMPLETED" {
		return systemspend.ErrReconciliation
	}
	if spend.Eligible {
		var id, player, playerAccount, systemAccount, sourceID, ruleVersion, fbID string
		var source contribution.Source
		var eligibleSpend, amount int64
		err = tx.QueryRow(ctx, `SELECT entry_id,player_id,player_fb_account_id,system_fb_account_id,source,source_id,eligible_spend,amount,rule_version,fb_transaction_id FROM contribution_entries WHERE entry_id=$1`, spend.ContributionEntryID).Scan(&id, &player, &playerAccount, &systemAccount, &source, &sourceID, &eligibleSpend, &amount, &ruleVersion, &fbID)
		if err != nil || id != spend.ContributionEntryID || player != spend.PlayerID || playerAccount != spend.PlayerFBAccountID || systemAccount != spend.SystemFBAccountID || source != contribution.SourceSystemService || sourceID != g15SourcePrefix+spend.OperationID || eligibleSpend != spend.FBAmount || amount != spend.ContributionAmount || ruleVersion != spend.RuleVersion || fbID != spend.FBTransactionID {
			return systemspend.ErrReconciliation
		}
		expected, ruleErr := (contribution.EligibilityPolicy{}).Evaluate(source, spend.FBAmount, spend.RuleVersion)
		if ruleErr != nil || expected != spend.ContributionAmount {
			return systemspend.ErrReconciliation
		}
	} else {
		var count int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM contribution_entries WHERE fb_transaction_id=$1`, spend.FBTransactionID).Scan(&count); err != nil {
			return err
		}
		if count != 0 || spend.ContributionEntryID != "" || spend.ContributionAmount != 0 || spend.RuleVersion != "" {
			return systemspend.ErrReconciliation
		}
	}
	var refunded, compensated int64
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(e.amount),0)::bigint FROM fb_ledger_transactions t JOIN fb_ledger_entries e ON e.transaction_id=t.transaction_id AND e.account_id=$2 WHERE t.original_transaction_id=$1 AND t.transaction_type IN ('REFUND','REVERSAL')`, spend.FBTransactionID, spend.PlayerFBAccountID).Scan(&refunded); err != nil {
		return err
	}
	if err = tx.QueryRow(ctx, `SELECT COALESCE(sum(amount),0)::bigint FROM contribution_compensations WHERE original_fb_transaction_id=$1`, spend.FBTransactionID).Scan(&compensated); err != nil {
		return err
	}
	if refunded < 0 || refunded > spend.FBAmount || (spend.Eligible && refunded != compensated) || (!spend.Eligible && compensated != 0) || refunded != spend.RefundedAmount || (refunded > 0 && !spend.Refundable) || (refunded > 0 && refunded < spend.FBAmount && !spend.PartialRefundAllowed) {
		return systemspend.ErrReconciliation
	}
	return nil
}

// ReconcileSystemSpends checks each immutable spend against FB and Contribution
// history, then reuses the existing account-level reconciliation services.
func (s *Store) ReconcileSystemSpends(ctx context.Context) (systemspend.ReconciliationReport, error) {
	if s == nil || s.pool == nil {
		return systemspend.ReconciliationReport{}, systemspend.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT operation_id FROM system_spends ORDER BY operation_id`)
	if err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	var operations []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return systemspend.ReconciliationReport{}, err
		}
		operations = append(operations, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return systemspend.ReconciliationReport{}, err
	}
	rows.Close()
	report := systemspend.ReconciliationReport{Balanced: true, Checked: len(operations)}
	for _, id := range operations {
		value, loadErr := loadSystemSpendTx(ctx, tx, id)
		if loadErr == nil {
			loadErr = validateSystemSpendTx(ctx, tx, value)
		}
		if loadErr != nil {
			report.Balanced = false
			report.Mismatches = append(report.Mismatches, systemspend.ReconciliationMismatch{OperationID: id, Reason: "ledger, rule, refund, or ownership mismatch"})
		}
	}
	var orphanCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM contribution_entries c WHERE c.source='SYSTEM_SERVICE' AND c.source_id LIKE 'g15:%' AND NOT EXISTS (SELECT 1 FROM system_spends s WHERE s.contribution_entry_id=c.entry_id)`).Scan(&orphanCount); err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	if orphanCount != 0 {
		report.Balanced = false
		report.Mismatches = append(report.Mismatches, systemspend.ReconciliationMismatch{Reason: fmt.Sprintf("%d orphan G15 Contribution entries", orphanCount)})
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM fb_ledger_transactions t WHERE t.reference_type='SYSTEM_SPEND_NON_ELIGIBLE' AND t.reference_id LIKE 'g15:%' AND NOT EXISTS (SELECT 1 FROM system_spends s WHERE s.fb_transaction_id=t.transaction_id)`).Scan(&orphanCount); err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	if orphanCount != 0 {
		report.Balanced = false
		report.Mismatches = append(report.Mismatches, systemspend.ReconciliationMismatch{Reason: fmt.Sprintf("%d orphan G15 non-eligible FB debits", orphanCount)})
	}
	if err = tx.Commit(ctx); err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	fb, err := s.ReconcileLedger(ctx)
	if err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	contributions, err := s.ReconcileContribution(ctx)
	if err != nil {
		return systemspend.ReconciliationReport{}, err
	}
	if !fb.Balanced {
		report.Balanced = false
		report.Mismatches = append(report.Mismatches, systemspend.ReconciliationMismatch{Reason: "FB account reconciliation mismatch"})
	}
	if !contributions.Balanced {
		report.Balanced = false
		report.Mismatches = append(report.Mismatches, systemspend.ReconciliationMismatch{Reason: "Contribution account or recovery debt mismatch"})
	}
	return report, nil
}
