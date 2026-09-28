package postgres

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"fractallegend/game-server/internal/emission"
	"github.com/jackc/pgx/v5"
)

func emissionRecoveryConserved(pool emission.Pool) bool {
	if pool.TotalEmissionCapacity < 0 || pool.TotalReserved < 0 || pool.TotalDistributed < 0 ||
		pool.RemainingCapacity < 0 || pool.RecoveryDebt < 0 {
		return false
	}
	capacity := big.NewInt(pool.TotalReserved)
	capacity.Add(capacity, big.NewInt(pool.TotalDistributed))
	if big.NewInt(pool.RecoveryDebt).Cmp(capacity) > 0 {
		return false
	}
	capacity.Add(capacity, big.NewInt(pool.RemainingCapacity))
	capacity.Sub(capacity, big.NewInt(pool.RecoveryDebt))
	return capacity.Cmp(big.NewInt(pool.TotalEmissionCapacity)) == 0
}

func appendRecoveryEntryTx(ctx context.Context, tx pgx.Tx, value emission.RecoveryEntry) error {
	if value.SourceType != "G21_P0_TEST_DISTRIBUTION" && value.SourceType != "G21_TEST_MINING_SETTLEMENT" {
		// All other pool mutations leave D unchanged, including late G14 refunds.
		if err := tx.QueryRow(ctx, `SELECT total_distributed FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).
			Scan(&value.DistributedBefore); err != nil {
			return err
		}
		value.DistributedAfter = value.DistributedBefore
	}
	_, err := tx.Exec(ctx, `INSERT INTO black_iron_emission_recovery_entries
		 (recovery_entry_id,source_type,source_id,emission_entry_id,block_entry_id,test_distribution_id,settlement_consumption_id,
		 net_emission_delta,reserved_delta,distributed_delta,distributed_before,distributed_after,
		 remaining_before,remaining_after,debt_before,debt_after,pool_revision,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		value.ID, value.SourceType, value.SourceID, value.EmissionEntryID, value.BlockEntryID, value.TestDistributionID, value.SettlementConsumptionID,
		value.NetEmissionDelta, value.ReservedDelta, value.DistributedDelta, value.DistributedBefore, value.DistributedAfter,
		value.RemainingBefore, value.RemainingAfter, value.DebtBefore, value.DebtAfter, value.PoolRevision, value.CreatedAt)
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
	var net, reserved, distributed, remaining, debt int64
	for i, record := range records {
		if record.PoolRevision != int64(i+1) {
			bad(fmt.Sprintf("recovery %s revision gap", record.ID))
		}
		if record.RemainingBefore != remaining || record.DebtBefore != debt || record.DistributedBefore != distributed {
			bad(fmt.Sprintf("recovery %s before state", record.ID))
		}
		var expectedRemaining, expectedDebt, expectedReserved, expectedDistributed, expectedNet int64
		expectedRemaining, expectedDebt, expectedReserved, expectedDistributed, expectedNet = remaining, debt, reserved, distributed, net
		if record.SourceType != "G21_P0_TEST_DISTRIBUTION" && record.SourceType != "G21_TEST_MINING_SETTLEMENT" &&
			(record.DistributedDelta != 0 || record.TestDistributionID != nil || record.SettlementConsumptionID != nil) {
			bad(fmt.Sprintf("recovery %s unexpected distribution", record.ID))
		}
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
		case "G21_P0_TEST_DISTRIBUTION":
			var id, sourceID, instanceID, blockID string
			var amount, revision int64
			var createdAt time.Time
			err := tx.QueryRow(ctx, `SELECT d.distribution_id,d.reservation_source_id,b.block_instance_id,b.display_block_id,d.amount,d.pool_revision,d.created_at
				FROM mining_prerequisite_distributions d JOIN mining_reservation_instance_bindings b
				ON b.reservation_source_id=d.reservation_source_id WHERE d.distribution_id=$1`, record.TestDistributionID).
				Scan(&id, &sourceID, &instanceID, &blockID, &amount, &revision, &createdAt)
			if err != nil || record.TestDistributionID == nil || id != *record.TestDistributionID ||
				sourceID != record.SourceID || instanceID == "" || blockID == "" || revision != record.PoolRevision || !record.CreatedAt.Equal(createdAt) ||
				record.EmissionEntryID != nil || record.BlockEntryID != nil || record.NetEmissionDelta != 0 ||
				record.ReservedDelta != -amount || record.DistributedDelta != amount || amount <= 0 ||
				reserved < amount || debt != 0 {
				bad(fmt.Sprintf("recovery %s invalid distribution", record.ID))
				continue
			}
			expectedReserved -= amount
			expectedDistributed += amount
		case "G21_TEST_MINING_SETTLEMENT":
			if record.SettlementConsumptionID == nil {
				bad(fmt.Sprintf("recovery %s missing settlement consumption", record.ID))
				continue
			}
			var consumptionID, sourceID, instanceID, settlementID string
			var amount, revision, grantSum, lotSum int64
			var createdAt time.Time
			err := tx.QueryRow(ctx, `SELECT c.consumption_id,c.reservation_source_id,c.block_instance_id,c.settlement_id,
				c.amount,c.pool_revision,c.created_at,
				(SELECT COALESCE(sum(g.quantity),0) FROM mining_reward_grants g WHERE g.settlement_id=c.settlement_id),
				(SELECT COALESCE(sum(l.quantity),0) FROM mining_reward_issuance_lots l WHERE l.settlement_id=c.settlement_id AND l.status='G21_SETTLED')
				FROM mining_reward_reservation_consumptions c WHERE c.consumption_id=$1`, *record.SettlementConsumptionID).
				Scan(&consumptionID, &sourceID, &instanceID, &settlementID, &amount, &revision, &createdAt, &grantSum, &lotSum)
			var receiptCount int
			if err == nil {
				err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_settlement_receipts
					WHERE settlement_id=$1 AND block_instance_id=$2 AND reservation_source_id=$3 AND total_ore=$4`,
					settlementID, instanceID, sourceID, amount).Scan(&receiptCount)
			}
			if err != nil || consumptionID != *record.SettlementConsumptionID || sourceID != record.SourceID ||
				instanceID == "" || receiptCount != 1 || grantSum != amount || lotSum != amount ||
				revision != record.PoolRevision || !record.CreatedAt.Equal(createdAt) ||
				record.EmissionEntryID != nil || record.BlockEntryID != nil || record.TestDistributionID != nil ||
				record.NetEmissionDelta != 0 || record.ReservedDelta != -amount || record.DistributedDelta != amount ||
				amount <= 0 || reserved < amount || debt != 0 {
				bad(fmt.Sprintf("recovery %s invalid settlement", record.ID))
				continue
			}
			// Counting a receipt does not verify its immutable canonical bytes or
			// its complete source/artifact chain. Recovery must fail closed just
			// like historical settlement replay when any part is corrupted.
			var commandID string
			if err := tx.QueryRow(ctx, `SELECT command_id FROM mining_reward_settlement_receipts
				WHERE settlement_id=$1`, settlementID).Scan(&commandID); err != nil {
				bad(fmt.Sprintf("recovery %s missing settlement receipt", record.ID))
				continue
			}
			if _, err := loadMiningRewardReceiptTx(ctx, tx, commandID); err != nil {
				bad(fmt.Sprintf("recovery %s invalid settlement receipt/artifacts", record.ID))
				continue
			}
			expectedReserved -= amount
			expectedDistributed += amount
		default:
			bad(fmt.Sprintf("recovery %s unknown source", record.ID))
		}
		if record.RemainingAfter != expectedRemaining || record.DebtAfter != expectedDebt || record.DistributedAfter != expectedDistributed {
			bad(fmt.Sprintf("recovery %s after state", record.ID))
		}
		net, reserved, distributed, remaining, debt = expectedNet, expectedReserved, expectedDistributed, expectedRemaining, expectedDebt
	}
	if net != pool.TotalEmissionCapacity || reserved != pool.TotalReserved || distributed != pool.TotalDistributed || remaining != pool.RemainingCapacity || debt != pool.RecoveryDebt {
		bad("replayed recovery state differs from pool")
	}
	return mismatches
}
