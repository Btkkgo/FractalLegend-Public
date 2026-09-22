package postgres

import (
	"context"
	"errors"
	"fmt"

	"fractallegend/game-server/internal/emission"
	"github.com/jackc/pgx/v5"
)

func loadEmissionPoolTx(ctx context.Context, tx pgx.Tx, lock bool) (emission.Pool, error) {
	query := `SELECT pool_id,total_eligible_spend_observed,total_refunded_spend,total_emission_capacity,total_reserved,total_distributed,remaining_capacity,rule_version,revision,created_at,updated_at FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`
	if lock {
		query += ` FOR UPDATE`
	}
	var value emission.Pool
	err := tx.QueryRow(ctx, query).Scan(&value.ID, &value.TotalEligibleSpendObserved, &value.TotalRefundedSpend, &value.TotalEmissionCapacity, &value.TotalReserved, &value.TotalDistributed, &value.RemainingCapacity, &value.RuleVersion, &value.Revision, &value.CreatedAt, &value.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return emission.Pool{}, emission.ErrInvariant
	}
	if err != nil {
		return emission.Pool{}, err
	}
	value.CreatedAt = value.CreatedAt.UTC()
	value.UpdatedAt = value.UpdatedAt.UTC()
	return value, nil
}

func loadEmissionReceiptBySourceTx(ctx context.Context, tx pgx.Tx, sourceType, sourceID string) (emission.Receipt, error) {
	var value emission.Receipt
	err := tx.QueryRow(ctx, `SELECT r.receipt_id,r.entry_id,r.source_id,r.eligible_spend,r.emission_added,r.pool_before,r.pool_after,r.remaining_capacity,r.rule_version,r.created_at FROM black_iron_emission_entries e JOIN black_iron_emission_receipts r ON r.entry_id=e.entry_id WHERE e.source_type=$1 AND e.source_id=$2`, sourceType, sourceID).Scan(&value.ID, &value.EntryID, &value.SourceID, &value.EligibleSpend, &value.EmissionAdded, &value.PoolBefore, &value.PoolAfter, &value.RemainingCapacity, &value.RuleVersion, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		var orphan bool
		if scanErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM black_iron_emission_entries WHERE source_type=$1 AND source_id=$2)`, sourceType, sourceID).Scan(&orphan); scanErr != nil {
			return emission.Receipt{}, scanErr
		}
		if orphan {
			return emission.Receipt{}, emission.ErrInvariant
		}
		return emission.Receipt{}, emission.ErrNotFound
	}
	if err != nil {
		return emission.Receipt{}, err
	}
	value.CreatedAt = value.CreatedAt.UTC()
	return value, nil
}

func (s *Store) LoadBlackIronEmission(ctx context.Context, spendID string) (emission.Receipt, error) {
	if s == nil || s.pool == nil {
		return emission.Receipt{}, emission.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return emission.Receipt{}, err
	}
	defer tx.Rollback(ctx)
	value, err := loadEmissionReceiptBySourceTx(ctx, tx, emissionSpendSource, spendID)
	if err != nil {
		return emission.Receipt{}, err
	}
	return value, tx.Commit(ctx)
}

func snapshotEmissionTx(ctx context.Context, tx pgx.Tx) (emission.Snapshot, error) {
	pool, err := loadEmissionPoolTx(ctx, tx, false)
	if err != nil {
		return emission.Snapshot{}, err
	}
	result := emission.Snapshot{Pool: pool, Entries: []emission.Entry{}, Receipts: []emission.Receipt{}}
	rows, err := tx.Query(ctx, `SELECT entry_id,source_type,source_id,spend_id,eligible_spend_amount,emission_amount,rule_version,created_at,pool_revision FROM black_iron_emission_entries ORDER BY pool_revision`)
	if err != nil {
		return emission.Snapshot{}, err
	}
	for rows.Next() {
		var e emission.Entry
		err = rows.Scan(&e.ID, &e.SourceType, &e.SourceID, &e.SpendID, &e.EligibleSpendAmount, &e.EmissionAmount, &e.RuleVersion, &e.CreatedAt, &e.PoolRevision)
		if err != nil {
			rows.Close()
			return emission.Snapshot{}, err
		}
		e.CreatedAt = e.CreatedAt.UTC()
		result.Entries = append(result.Entries, e)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return emission.Snapshot{}, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT r.receipt_id,r.entry_id,r.source_id,r.eligible_spend,r.emission_added,r.pool_before,r.pool_after,r.remaining_capacity,r.rule_version,r.created_at FROM black_iron_emission_receipts r JOIN black_iron_emission_entries e ON e.entry_id=r.entry_id ORDER BY e.pool_revision`)
	if err != nil {
		return emission.Snapshot{}, err
	}
	for rows.Next() {
		var r emission.Receipt
		if err = rows.Scan(&r.ID, &r.EntryID, &r.SourceID, &r.EligibleSpend, &r.EmissionAdded, &r.PoolBefore, &r.PoolAfter, &r.RemainingCapacity, &r.RuleVersion, &r.CreatedAt); err != nil {
			rows.Close()
			return emission.Snapshot{}, err
		}
		r.CreatedAt = r.CreatedAt.UTC()
		result.Receipts = append(result.Receipts, r)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return emission.Snapshot{}, err
	}
	rows.Close()
	return result, nil
}

func (s *Store) SnapshotBlackIronEmission(ctx context.Context) (emission.Snapshot, error) {
	if s == nil || s.pool == nil {
		return emission.Snapshot{}, emission.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return emission.Snapshot{}, err
	}
	defer tx.Rollback(ctx)
	value, err := snapshotEmissionTx(ctx, tx)
	if err != nil {
		return emission.Snapshot{}, err
	}
	return value, tx.Commit(ctx)
}

func (s *Store) ReconcileBlackIronEmission(ctx context.Context) (emission.ReconciliationReport, error) {
	if s == nil || s.pool == nil {
		return emission.ReconciliationReport{}, emission.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return emission.ReconciliationReport{}, err
	}
	defer tx.Rollback(ctx)
	snapshot, err := snapshotEmissionTx(ctx, tx)
	if err != nil {
		return emission.ReconciliationReport{}, err
	}
	report := emission.ReconciliationReport{Balanced: true, Checked: len(snapshot.Entries)}
	bad := func(reason string) { report.Balanced = false; report.Mismatches = append(report.Mismatches, reason) }
	pool := snapshot.Pool
	var gross, refunded, net int64
	spendGross := make(map[string]int64)
	spendRefunded := make(map[string]int64)
	spendRule := make(map[string]string)
	var previousPoolAfter int64
	if pool.TotalReserved != 0 || pool.TotalDistributed != 0 || pool.RemainingCapacity != pool.TotalEmissionCapacity {
		bad("pool conservation")
	}
	if pool.Revision != int64(len(snapshot.Entries)) {
		bad("pool revision differs from journal")
	}
	if len(snapshot.Receipts) != len(snapshot.Entries) {
		bad("entry/receipt cardinality mismatch")
	}
	for i, entry := range snapshot.Entries {
		if entry.PoolRevision != int64(i+1) {
			bad(fmt.Sprintf("entry %s has discontinuous revision", entry.ID))
		}
		if entry.SourceType == emissionSpendSource {
			var eligible bool
			var amount int64
			if err = tx.QueryRow(ctx, `SELECT eligible,fb_amount FROM system_spends WHERE spend_id=$1`, entry.SpendID).Scan(&eligible, &amount); err != nil || !eligible || entry.SourceID != entry.SpendID || entry.EligibleSpendAmount != amount || entry.EmissionAmount <= 0 {
				bad(fmt.Sprintf("entry %s lacks valid G15 source", entry.ID))
			}
			expected, ruleErr := emission.Calculate(entry.RuleVersion, entry.EligibleSpendAmount)
			if ruleErr != nil || expected != entry.EmissionAmount {
				bad(fmt.Sprintf("entry %s has wrong rule amount", entry.ID))
			}
			if _, exists := spendGross[entry.SpendID]; exists {
				bad(fmt.Sprintf("entry %s duplicates a G15 source", entry.ID))
			}
			spendGross[entry.SpendID], spendRule[entry.SpendID] = entry.EligibleSpendAmount, entry.RuleVersion
			gross, err = emission.AddCapacity(gross, entry.EligibleSpendAmount)
		} else if entry.SourceType == emissionRefundSource {
			var originalFB string
			var amount int64
			var spendFB string
			lookupErr := tx.QueryRow(ctx, `SELECT c.original_fb_transaction_id,c.amount,s.fb_transaction_id FROM contribution_compensations c JOIN system_spends s ON s.spend_id=$2 WHERE c.compensation_id=$1`, entry.SourceID, entry.SpendID).Scan(&originalFB, &amount, &spendFB)
			if lookupErr != nil || originalFB != spendFB || entry.EligibleSpendAmount != -amount || entry.EmissionAmount >= 0 {
				bad(fmt.Sprintf("entry %s lacks valid G14 compensation", entry.ID))
			}
			original := spendGross[entry.SpendID]
			prior := spendRefunded[entry.SpendID]
			refundAmount := -entry.EligibleSpendAmount
			if original <= 0 || entry.RuleVersion != spendRule[entry.SpendID] || refundAmount <= 0 || prior > original || refundAmount > original-prior {
				bad(fmt.Sprintf("entry %s has invalid refund sequence", entry.ID))
			} else {
				before, beforeErr := emission.Calculate(entry.RuleVersion, original-prior)
				var after int64
				var afterErr error
				if original-prior-refundAmount > 0 {
					after, afterErr = emission.Calculate(entry.RuleVersion, original-prior-refundAmount)
				}
				if beforeErr != nil || afterErr != nil || after-before != entry.EmissionAmount {
					bad(fmt.Sprintf("entry %s has wrong refund rule amount", entry.ID))
				}
				spendRefunded[entry.SpendID] = prior + refundAmount
			}
			refunded, err = emission.AddCapacity(refunded, -entry.EligibleSpendAmount)
		} else {
			bad(fmt.Sprintf("entry %s has unknown source type", entry.ID))
		}
		if err != nil {
			bad("journal amount overflow")
		}
		if entry.EmissionAmount > 0 {
			net, err = emission.AddCapacity(net, entry.EmissionAmount)
		} else if net < -entry.EmissionAmount {
			bad("journal capacity went negative")
		} else {
			net += entry.EmissionAmount
		}
		if err != nil {
			bad("journal capacity overflow")
		}
		if i < len(snapshot.Receipts) {
			r := snapshot.Receipts[i]
			if r.EntryID != entry.ID || r.SourceID != entry.SourceID || r.EligibleSpend != entry.EligibleSpendAmount || r.EmissionAdded != entry.EmissionAmount || r.RuleVersion != entry.RuleVersion || r.PoolBefore != previousPoolAfter || r.PoolAfter != r.RemainingCapacity || r.PoolAfter-r.PoolBefore != r.EmissionAdded {
				bad(fmt.Sprintf("entry %s receipt mismatch", entry.ID))
			}
			previousPoolAfter = r.PoolAfter
		}
	}
	if gross != pool.TotalEligibleSpendObserved || refunded != pool.TotalRefundedSpend || net != pool.TotalEmissionCapacity || previousPoolAfter != pool.TotalEmissionCapacity || refunded > gross {
		bad("journal sums differ from pool")
	}
	if len(snapshot.Entries) > 0 && pool.RuleVersion != snapshot.Entries[len(snapshot.Entries)-1].RuleVersion {
		bad("pool rule version differs from latest entry")
	}
	var missing int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM system_spends s WHERE s.eligible AND NOT EXISTS(SELECT 1 FROM black_iron_emission_entries e WHERE e.source_type=$1 AND e.source_id=s.spend_id)`, emissionSpendSource).Scan(&missing); err != nil {
		return emission.ReconciliationReport{}, err
	}
	if missing != 0 {
		bad(fmt.Sprintf("%d eligible G15 spends lack emission consequence", missing))
	}
	if err = tx.Commit(ctx); err != nil {
		return emission.ReconciliationReport{}, err
	}
	return report, nil
}

var _ emission.Repository = (*Store)(nil)
