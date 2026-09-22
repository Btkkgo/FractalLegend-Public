package postgres

import (
	"context"
	"errors"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/systemspend"
	"github.com/jackc/pgx/v5"
)

const (
	emissionSpendSource  = "G15_ELIGIBLE_SYSTEM_SPEND"
	emissionRefundSource = "G14_REFUND_COMPENSATION"
)

// ApplyBlackIronEmission consumes only a persisted, validated G15 spend. The
// caller supplies no eligibility or emission amount. The development rule is
// explicitly selected by the internal service; there is no gameplay route.
func (s *Store) ApplyBlackIronEmission(ctx context.Context, spendID, ruleVersion string) (emission.Receipt, error) {
	if s == nil || s.pool == nil {
		return emission.Receipt{}, emission.ErrUnavailable
	}
	if _, err := emission.Calculate(ruleVersion, 1); err != nil {
		return emission.Receipt{}, err
	}
	for attempt := 0; attempt < 32; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if err != nil {
			return emission.Receipt{}, err
		}
		receipt, err := s.applyBlackIronEmissionTx(ctx, tx, spendID, ruleVersion)
		if err == nil {
			err = s.checkEmissionFailure("before_commit")
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if err == nil {
			return receipt, nil
		}
		if !retryContributionTransaction(err) {
			return emission.Receipt{}, err
		}
		select {
		case <-ctx.Done():
			return emission.Receipt{}, ctx.Err()
		case <-time.After(time.Duration((attempt%8)+1) * time.Millisecond):
		}
	}
	return emission.Receipt{}, contribution.ErrRetryExhausted
}

func (s *Store) applyBlackIronEmissionTx(ctx context.Context, tx pgx.Tx, spendID, ruleVersion string) (emission.Receipt, error) {
	var operationID, fbTransactionID string
	err := tx.QueryRow(ctx, `SELECT operation_id,fb_transaction_id FROM system_spends WHERE spend_id=$1`, spendID).Scan(&operationID, &fbTransactionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return emission.Receipt{}, emission.ErrNotFound
	}
	if err != nil {
		return emission.Receipt{}, err
	}
	// G14 locks the same original FB transaction before inserting a refund.
	// This serializes first application with every compensation, including a
	// refund that arrives before the emission consequence is applied.
	var lockedID string
	if err = tx.QueryRow(ctx, `SELECT transaction_id FROM fb_ledger_transactions WHERE transaction_id=$1 FOR UPDATE`, fbTransactionID).Scan(&lockedID); err != nil {
		return emission.Receipt{}, err
	}
	spend, err := loadSystemSpendTx(ctx, tx, operationID)
	if err != nil || spend.ID != spendID || spend.FBTransactionID != fbTransactionID {
		return emission.Receipt{}, emission.ErrInvariant
	}
	if err = validateSystemSpendTx(ctx, tx, spend); err != nil {
		return emission.Receipt{}, emission.ErrInvariant
	}
	prior, err := loadEmissionReceiptBySourceTx(ctx, tx, emissionSpendSource, spend.ID)
	if err == nil {
		if !spend.Eligible || prior.RuleVersion != ruleVersion || prior.EligibleSpend != spend.FBAmount {
			return emission.Receipt{}, emission.ErrConflict
		}
		return prior, nil
	}
	if !errors.Is(err, emission.ErrNotFound) {
		return emission.Receipt{}, err
	}
	if !spend.Eligible {
		return emission.Receipt{}, nil
	}
	amount, err := emission.Calculate(ruleVersion, spend.FBAmount)
	if err != nil {
		return emission.Receipt{}, err
	}
	receipt, err := s.appendEmissionEntryTx(ctx, tx, emission.Entry{
		ID: newContributionID("emission-entry"), SourceType: emissionSpendSource,
		SourceID: spend.ID, SpendID: spend.ID, EligibleSpendAmount: spend.FBAmount,
		EmissionAmount: amount, RuleVersion: ruleVersion,
	})
	if err != nil {
		return emission.Receipt{}, err
	}
	rows, err := tx.Query(ctx, `SELECT compensation_id,amount FROM contribution_compensations WHERE original_fb_transaction_id=$1 ORDER BY created_at,compensation_id`, spend.FBTransactionID)
	if err != nil {
		return emission.Receipt{}, err
	}
	type compensation struct {
		id     string
		amount int64
	}
	var compensations []compensation
	for rows.Next() {
		var value compensation
		if err = rows.Scan(&value.id, &value.amount); err != nil {
			rows.Close()
			return emission.Receipt{}, err
		}
		compensations = append(compensations, value)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return emission.Receipt{}, err
	}
	rows.Close()
	for _, value := range compensations {
		if err = s.applyEmissionRefundTx(ctx, tx, spend, value.id, value.amount, ruleVersion); err != nil {
			return emission.Receipt{}, err
		}
	}
	return receipt, nil
}

// G14 calls this inside its existing FB + Contribution refund transaction.
// Historical G13 refunds and G15 spends with no applied emission remain
// untouched; a later Apply catches up every persisted compensation atomically.
func (s *Store) applyEmissionRefundForCompensationTx(ctx context.Context, tx pgx.Tx, originalFBTransactionID, compensationID string, amount int64) error {
	var operationID string
	err := tx.QueryRow(ctx, `SELECT operation_id FROM system_spends WHERE fb_transaction_id=$1`, originalFBTransactionID).Scan(&operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	spend, err := loadSystemSpendTx(ctx, tx, operationID)
	if err != nil || !spend.Eligible {
		return emission.ErrInvariant
	}
	first, err := loadEmissionReceiptBySourceTx(ctx, tx, emissionSpendSource, spend.ID)
	if errors.Is(err, emission.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.applyEmissionRefundTx(ctx, tx, spend, compensationID, amount, first.RuleVersion)
}

func (s *Store) applyEmissionRefundTx(ctx context.Context, tx pgx.Tx, spend systemspend.SystemSpend, compensationID string, amount int64, ruleVersion string) error {
	if amount <= 0 || amount > spend.FBAmount {
		return emission.ErrInvariant
	}
	if _, err := loadEmissionReceiptBySourceTx(ctx, tx, emissionRefundSource, compensationID); err == nil {
		return nil
	} else if !errors.Is(err, emission.ErrNotFound) {
		return err
	}
	var previousRefunded int64
	err := tx.QueryRow(ctx, `SELECT COALESCE(-sum(eligible_spend_amount),0)::bigint FROM black_iron_emission_entries WHERE spend_id=$1 AND source_type=$2`, spend.ID, emissionRefundSource).Scan(&previousRefunded)
	if err != nil || previousRefunded < 0 || previousRefunded > spend.FBAmount || amount > spend.FBAmount-previousRefunded {
		return emission.ErrInvariant
	}
	before, err := emission.Calculate(ruleVersion, spend.FBAmount-previousRefunded)
	if err != nil {
		return err
	}
	remaining := spend.FBAmount - previousRefunded - amount
	var after int64
	if remaining > 0 {
		after, err = emission.Calculate(ruleVersion, remaining)
		if err != nil {
			return err
		}
	}
	if after >= before {
		return emission.ErrInvariant
	}
	_, err = s.appendEmissionEntryTx(ctx, tx, emission.Entry{
		ID: newContributionID("emission-entry"), SourceType: emissionRefundSource,
		SourceID: compensationID, SpendID: spend.ID, EligibleSpendAmount: -amount,
		EmissionAmount: after - before, RuleVersion: ruleVersion,
	})
	return err
}

func (s *Store) appendEmissionEntryTx(ctx context.Context, tx pgx.Tx, entry emission.Entry) (emission.Receipt, error) {
	pool, err := loadEmissionPoolTx(ctx, tx, true)
	if err != nil {
		return emission.Receipt{}, err
	}
	if pool.TotalReserved != 0 || pool.TotalDistributed != 0 || pool.TotalEmissionCapacity != pool.RemainingCapacity || pool.TotalRefundedSpend > pool.TotalEligibleSpendObserved {
		return emission.Receipt{}, emission.ErrInvariant
	}
	before := pool.TotalEmissionCapacity
	var after, observed, refunded int64
	if entry.EmissionAmount > 0 && entry.EligibleSpendAmount > 0 {
		after, err = emission.AddCapacity(before, entry.EmissionAmount)
		if err == nil {
			observed, err = emission.AddCapacity(pool.TotalEligibleSpendObserved, entry.EligibleSpendAmount)
		}
		refunded = pool.TotalRefundedSpend
	} else if entry.EmissionAmount < 0 && entry.EligibleSpendAmount < 0 {
		if before < -entry.EmissionAmount || entry.EligibleSpendAmount == -1<<63 {
			return emission.Receipt{}, emission.ErrInvariant
		}
		after = before + entry.EmissionAmount
		observed = pool.TotalEligibleSpendObserved
		refunded, err = emission.AddCapacity(pool.TotalRefundedSpend, -entry.EligibleSpendAmount)
		if err == nil && refunded > observed {
			err = emission.ErrInvariant
		}
	} else {
		return emission.Receipt{}, emission.ErrInvariant
	}
	if err != nil {
		return emission.Receipt{}, err
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	entry.CreatedAt = now
	entry.PoolRevision = pool.Revision + 1
	if entry.PoolRevision <= 0 {
		return emission.Receipt{}, emission.ErrOverflow
	}
	var compensationID any
	if entry.SourceType == emissionRefundSource {
		compensationID = entry.SourceID
	}
	_, err = tx.Exec(ctx, `INSERT INTO black_iron_emission_entries(entry_id,source_type,source_id,spend_id,compensation_id,eligible_spend_amount,emission_amount,rule_version,created_at,pool_revision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, entry.ID, entry.SourceType, entry.SourceID, entry.SpendID, compensationID, entry.EligibleSpendAmount, entry.EmissionAmount, entry.RuleVersion, entry.CreatedAt, entry.PoolRevision)
	if err != nil {
		return emission.Receipt{}, err
	}
	if err = s.checkEmissionFailure("after_entry"); err != nil {
		return emission.Receipt{}, err
	}
	_, err = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_eligible_spend_observed=$1,total_refunded_spend=$2,total_emission_capacity=$3,remaining_capacity=$3,rule_version=$4,revision=$5,updated_at=$6 WHERE pool_id='GLOBAL'`, observed, refunded, after, entry.RuleVersion, entry.PoolRevision, now)
	if err != nil {
		return emission.Receipt{}, err
	}
	if err = s.checkEmissionFailure("after_pool"); err != nil {
		return emission.Receipt{}, err
	}
	receipt := emission.Receipt{ID: newContributionID("emission-receipt"), EntryID: entry.ID, SourceID: entry.SourceID,
		EligibleSpend: entry.EligibleSpendAmount, EmissionAdded: entry.EmissionAmount,
		PoolBefore: before, PoolAfter: after, RemainingCapacity: after,
		RuleVersion: entry.RuleVersion, CreatedAt: now}
	_, err = tx.Exec(ctx, `INSERT INTO black_iron_emission_receipts(receipt_id,entry_id,source_id,eligible_spend,emission_added,pool_before,pool_after,remaining_capacity,rule_version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, receipt.ID, receipt.EntryID, receipt.SourceID, receipt.EligibleSpend, receipt.EmissionAdded, receipt.PoolBefore, receipt.PoolAfter, receipt.RemainingCapacity, receipt.RuleVersion, receipt.CreatedAt)
	if err != nil {
		return emission.Receipt{}, err
	}
	if err = s.checkEmissionFailure("after_receipt"); err != nil {
		return emission.Receipt{}, err
	}
	return loadEmissionReceiptBySourceTx(ctx, tx, entry.SourceType, entry.SourceID)
}

func (s *Store) checkEmissionFailure(point string) error {
	if s.emissionFailureInjector != nil {
		return s.emissionFailureInjector(point)
	}
	return nil
}
