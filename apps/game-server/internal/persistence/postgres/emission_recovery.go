package postgres

import (
	"context"
	"fmt"
	"math"

	"fractallegend/game-server/internal/emission"
	"github.com/jackc/pgx/v5"
)

func emissionRecoveryConserved(pool emission.Pool) bool {
	if pool.TotalEmissionCapacity < 0 || pool.TotalReserved < 0 || pool.TotalDistributed != 0 ||
		pool.RemainingCapacity < 0 || pool.RecoveryDebt < 0 || pool.RecoveryDebt > pool.TotalReserved ||
		pool.TotalReserved > math.MaxInt64-pool.RemainingCapacity {
		return false
	}
	return pool.TotalEmissionCapacity == pool.TotalReserved+pool.RemainingCapacity-pool.RecoveryDebt
}

func appendRecoveryEntryTx(ctx context.Context, tx pgx.Tx, value emission.RecoveryEntry) error {
	_, err := tx.Exec(ctx, `INSERT INTO black_iron_emission_recovery_entries
		(recovery_entry_id,source_type,source_id,emission_entry_id,block_entry_id,net_emission_delta,reserved_delta,
		 remaining_before,remaining_after,debt_before,debt_after,pool_revision,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		value.ID, value.SourceType, value.SourceID, value.EmissionEntryID, value.BlockEntryID,
		value.NetEmissionDelta, value.ReservedDelta, value.RemainingBefore, value.RemainingAfter,
		value.DebtBefore, value.DebtAfter, value.PoolRevision, value.CreatedAt)
	return err
}

// Every pool revision has one immutable, source-bound recovery entry. Replay
// the journal independently of the mutable pool and fail without repairing it.
func reconcileRecoveryTx(ctx context.Context, tx pgx.Tx, pool emission.Pool, records []emission.RecoveryEntry) []string {
	var mismatches []string
	bad := func(reason string) { mismatches = append(mismatches, reason) }
	if !emissionRecoveryConserved(pool) {
		bad("extended pool conservation")
	}
	if pool.Revision != int64(len(records)) {
		bad("recovery journal revision/cardinality")
	}
	var net, reserved, remaining, debt int64
	for i, record := range records {
		if record.PoolRevision != int64(i+1) {
			bad(fmt.Sprintf("recovery %s revision gap", record.ID))
		}
		if record.RemainingBefore != remaining || record.DebtBefore != debt {
			bad(fmt.Sprintf("recovery %s before state", record.ID))
		}
		var expectedRemaining, expectedDebt, expectedReserved, expectedNet int64
		expectedRemaining, expectedDebt, expectedReserved, expectedNet = remaining, debt, reserved, net
		switch record.SourceType {
		case emissionSpendSource, emissionRefundSource:
			var id, sourceType, sourceID string
			var amount, revision, receiptRemaining int64
			var createdAtTime = record.CreatedAt
			err := tx.QueryRow(ctx, `SELECT e.entry_id,e.source_type,e.source_id,e.emission_amount,e.pool_revision,r.remaining_capacity,e.created_at
				FROM black_iron_emission_entries e JOIN black_iron_emission_receipts r ON r.entry_id=e.entry_id
				WHERE e.entry_id=$1`, record.EmissionEntryID).Scan(&id, &sourceType, &sourceID, &amount, &revision, &receiptRemaining, &createdAtTime)
			if err != nil || record.EmissionEntryID == nil || record.BlockEntryID != nil || id != *record.EmissionEntryID ||
				sourceType != record.SourceType || sourceID != record.SourceID || amount != record.NetEmissionDelta ||
				revision != record.PoolRevision || receiptRemaining != record.RemainingAfter || !record.CreatedAt.Equal(createdAtTime) {
				bad(fmt.Sprintf("recovery %s emission source/receipt", record.ID))
			}
			if record.ReservedDelta != 0 {
				bad(fmt.Sprintf("recovery %s emission reserved delta", record.ID))
			}
			if record.NetEmissionDelta > 0 {
				repay := min(record.NetEmissionDelta, debt)
				if remaining > math.MaxInt64-(record.NetEmissionDelta-repay) || net > math.MaxInt64-record.NetEmissionDelta {
					bad("recovery positive overflow")
					continue
				}
				expectedDebt -= repay
				expectedRemaining += record.NetEmissionDelta - repay
				expectedNet += record.NetEmissionDelta
			} else if record.NetEmissionDelta < 0 && record.NetEmissionDelta != math.MinInt64 {
				compensation := -record.NetEmissionDelta
				if net < compensation || debt > math.MaxInt64-(compensation-min(remaining, compensation)) {
					bad("recovery negative overflow")
					continue
				}
				available := min(remaining, compensation)
				expectedRemaining -= available
				expectedDebt += compensation - available
				expectedNet -= compensation
			} else {
				bad(fmt.Sprintf("recovery %s invalid emission amount", record.ID))
			}
		case "G18_BLOCK_OPEN", "G18_BLOCK_CANCEL":
			var id, blockID, action string
			var capacityDelta, poolBefore, poolAfter, reward int64
			var createdAtTime = record.CreatedAt
			err := tx.QueryRow(ctx, `SELECT e.entry_id,e.block_id,e.action,e.capacity_delta,e.pool_before,e.pool_after,b.reward_reserved,e.created_at
				FROM mining_block_entries e JOIN mining_blocks b ON b.block_id=e.block_id WHERE e.entry_id=$1`, record.BlockEntryID).
				Scan(&id, &blockID, &action, &capacityDelta, &poolBefore, &poolAfter, &reward, &createdAtTime)
			if err != nil || record.BlockEntryID == nil || record.EmissionEntryID != nil || id != *record.BlockEntryID ||
				blockID != record.SourceID || poolBefore != remaining || poolAfter != record.RemainingAfter ||
				!record.CreatedAt.Equal(createdAtTime) || reward <= 0 {
				bad(fmt.Sprintf("recovery %s block source/receipt", record.ID))
			}
			if record.SourceType == "G18_BLOCK_OPEN" {
				if action != "OPEN" || record.NetEmissionDelta != 0 || record.ReservedDelta != reward || capacityDelta != reward || remaining < reward || debt != 0 || reserved > math.MaxInt64-reward {
					bad(fmt.Sprintf("recovery %s invalid reservation", record.ID))
					continue
				}
				expectedReserved += reward
				expectedRemaining -= reward
			} else {
				repay := min(reward, debt)
				available := reward - repay
				if action != "CANCEL" || record.NetEmissionDelta != 0 || record.ReservedDelta != -reward || capacityDelta != -available || reserved < reward || remaining > math.MaxInt64-available {
					bad(fmt.Sprintf("recovery %s invalid cancellation", record.ID))
					continue
				}
				expectedReserved -= reward
				expectedDebt -= repay
				expectedRemaining += available
			}
		default:
			bad(fmt.Sprintf("recovery %s unknown source", record.ID))
		}
		if record.RemainingAfter != expectedRemaining || record.DebtAfter != expectedDebt {
			bad(fmt.Sprintf("recovery %s after state", record.ID))
		}
		net, reserved, remaining, debt = expectedNet, expectedReserved, expectedRemaining, expectedDebt
	}
	if net != pool.TotalEmissionCapacity || reserved != pool.TotalReserved || remaining != pool.RemainingCapacity || debt != pool.RecoveryDebt {
		bad("replayed recovery state differs from pool")
	}
	return mismatches
}
