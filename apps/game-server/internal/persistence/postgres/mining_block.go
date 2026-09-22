package postgres

import (
	"context"
	"errors"
	"math"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningblock"
	"github.com/jackc/pgx/v5"
)

func (s *Store) checkMiningBlockFailure(point string) error {
	if s.miningBlockFailureInjector != nil {
		return s.miningBlockFailureInjector(point)
	}
	return nil
}

func (s *Store) miningBlockTransaction(ctx context.Context, run func(pgx.Tx) (miningblock.Receipt, error)) (miningblock.Receipt, error) {
	if s == nil || s.pool == nil {
		return miningblock.Receipt{}, miningblock.ErrUnavailable
	}
	for attempt := 0; attempt < 32; attempt++ {
		tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
		if err != nil {
			return miningblock.Receipt{}, err
		}
		receipt, err := run(tx)
		if err == nil {
			err = s.checkMiningBlockFailure("before_commit")
		}
		if err == nil {
			err = tx.Commit(ctx)
		} else {
			_ = tx.Rollback(ctx)
		}
		if err == nil {
			if err = s.checkMiningBlockFailure("after_commit_before_response"); err != nil {
				return miningblock.Receipt{}, err
			}
			return receipt, nil
		}
		if !retryContributionTransaction(err) {
			return miningblock.Receipt{}, err
		}
		select {
		case <-ctx.Done():
			return miningblock.Receipt{}, ctx.Err()
		case <-time.After(time.Duration((attempt%8)+1) * time.Millisecond):
		}
	}
	return miningblock.Receipt{}, contribution.ErrRetryExhausted
}

func validMiningBlockID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, c := range id {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' || c == ':') {
			return false
		}
	}
	return true
}

func validateMiningPool(pool emission.Pool) error {
	if !emissionRecoveryConserved(pool) || pool.Revision < 0 {
		return miningblock.ErrInvariant
	}
	return nil
}

func (s *Store) CreateMiningBlock(ctx context.Context, commandID, ruleVersion string) (miningblock.Receipt, error) {
	if !validMiningBlockID(commandID) {
		return miningblock.Receipt{}, miningblock.ErrInvalidCommand
	}
	if ruleVersion != miningblock.DevelopmentRuleVersion {
		return miningblock.Receipt{}, miningblock.ErrUnknownRule
	}
	return s.miningBlockTransaction(ctx, func(tx pgx.Tx) (miningblock.Receipt, error) {
		pool, err := loadEmissionPoolTx(ctx, tx, true)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if err = validateMiningPool(pool); err != nil {
			return miningblock.Receipt{}, err
		}
		var priorBlockID string
		err = tx.QueryRow(ctx, `SELECT block_id FROM mining_blocks WHERE create_command_id=$1`, commandID).Scan(&priorBlockID)
		if err == nil {
			return loadMiningReceiptTx(ctx, tx, priorBlockID, miningblock.ActionOpen)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return miningblock.Receipt{}, err
		}
		if pool.RecoveryDebt > 0 {
			return miningblock.Receipt{}, miningblock.ErrRecoveryDebt
		}
		if pool.RemainingCapacity < miningblock.DevelopmentReward {
			return miningblock.Receipt{}, miningblock.ErrInsufficientCapacity
		}
		if pool.Revision == math.MaxInt64 {
			return miningblock.Receipt{}, miningblock.ErrOverflow
		}
		var height int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(block_height),0) FROM mining_blocks`).Scan(&height); err != nil {
			return miningblock.Receipt{}, err
		}
		if height == math.MaxInt64 {
			return miningblock.Receipt{}, miningblock.ErrOverflow
		}
		height++
		blockID := newContributionID("mining-block")
		now := time.Now().UTC().Truncate(time.Microsecond)
		poolAfter := pool.RemainingCapacity - miningblock.DevelopmentReward
		if err = s.checkMiningBlockFailure("before_block"); err != nil {
			return miningblock.Receipt{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO mining_blocks(block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,reward_reserved,pool_revision_at_reservation,created_at,updated_at) VALUES($1,$2,$3,'OPEN',$4,$5,$6,$7,$8,$5,$5)`, blockID, height, commandID, ruleVersion, now, now.Add(miningblock.DevelopmentDuration), miningblock.DevelopmentReward, pool.Revision+1)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if err = s.checkMiningBlockFailure("after_block"); err != nil {
			return miningblock.Receipt{}, err
		}
		_, err = tx.Exec(ctx, `INSERT INTO mining_block_reservations(block_id,amount,status,created_at,updated_at) VALUES($1,$2,'ACTIVE',$3,$3)`, blockID, miningblock.DevelopmentReward, now)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		_, err = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_reserved=total_reserved+$1,remaining_capacity=$2,revision=$3,updated_at=$4 WHERE pool_id='GLOBAL'`, miningblock.DevelopmentReward, poolAfter, pool.Revision+1, now)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if err = s.checkMiningBlockFailure("after_reserve"); err != nil {
			return miningblock.Receipt{}, err
		}
		entry := miningblock.Entry{ID: newContributionID("mining-block-entry"), BlockID: blockID, BlockHeight: height, Action: miningblock.ActionOpen, CapacityDelta: miningblock.DevelopmentReward, PoolBefore: pool.RemainingCapacity, PoolAfter: poolAfter, BlockStatusBefore: "", BlockStatusAfter: miningblock.StatusOpen, RuleVersion: ruleVersion, CreatedAt: now}
		receipt, err := s.appendMiningBlockEntryTx(ctx, tx, entry)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		err = appendRecoveryEntryTx(ctx, tx, emission.RecoveryEntry{
			ID: newContributionID("emission-recovery"), SourceType: "G18_BLOCK_OPEN", SourceID: blockID, BlockEntryID: &entry.ID,
			ReservedDelta: miningblock.DevelopmentReward, RemainingBefore: pool.RemainingCapacity, RemainingAfter: poolAfter,
			DebtBefore: pool.RecoveryDebt, DebtAfter: pool.RecoveryDebt, PoolRevision: pool.Revision + 1, CreatedAt: now,
		})
		return receipt, err
	})
}

func (s *Store) FinalizeMiningBlock(ctx context.Context, blockID string) (miningblock.Receipt, error) {
	return s.changeMiningBlock(ctx, blockID, miningblock.ActionFinalize)
}

func (s *Store) CancelMiningBlock(ctx context.Context, blockID string) (miningblock.Receipt, error) {
	return s.changeMiningBlock(ctx, blockID, miningblock.ActionCancel)
}

func (s *Store) changeMiningBlock(ctx context.Context, blockID, action string) (miningblock.Receipt, error) {
	if !validMiningBlockID(blockID) {
		return miningblock.Receipt{}, miningblock.ErrInvalidCommand
	}
	if action != miningblock.ActionFinalize && action != miningblock.ActionCancel {
		return miningblock.Receipt{}, miningblock.ErrInvalidCommand
	}
	return s.miningBlockTransaction(ctx, func(tx pgx.Tx) (miningblock.Receipt, error) {
		pool, err := loadEmissionPoolTx(ctx, tx, true)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if err = validateMiningPool(pool); err != nil {
			return miningblock.Receipt{}, err
		}
		block, err := loadMiningBlockTx(ctx, tx, blockID, true)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if block.RuleVersion != miningblock.DevelopmentRuleVersion {
			return miningblock.Receipt{}, miningblock.ErrUnknownRule
		}
		if (action == miningblock.ActionFinalize && block.Status == miningblock.StatusFinalized) || (action == miningblock.ActionCancel && block.Status == miningblock.StatusCancelled) {
			return loadMiningReceiptTx(ctx, tx, blockID, action)
		}
		if block.Status != miningblock.StatusOpen {
			return miningblock.Receipt{}, miningblock.ErrConflict
		}
		var reservation miningblock.Reservation
		err = tx.QueryRow(ctx, `SELECT block_id,amount,released,status,created_at,updated_at FROM mining_block_reservations WHERE block_id=$1 FOR UPDATE`, blockID).Scan(&reservation.BlockID, &reservation.Amount, &reservation.Released, &reservation.Status, &reservation.CreatedAt, &reservation.UpdatedAt)
		if err != nil || reservation.Amount != block.RewardReserved || reservation.Status != "ACTIVE" || reservation.Released != 0 {
			return miningblock.Receipt{}, miningblock.ErrInvariant
		}
		now := time.Now().UTC().Truncate(time.Microsecond)
		entry := miningblock.Entry{ID: newContributionID("mining-block-entry"), BlockID: blockID, BlockHeight: block.Height, Action: action, PoolBefore: pool.RemainingCapacity, PoolAfter: pool.RemainingCapacity, BlockStatusBefore: miningblock.StatusOpen, RuleVersion: block.RuleVersion, CreatedAt: now}
		if action == miningblock.ActionFinalize {
			entry.BlockStatusAfter = miningblock.StatusFinalized
			_, err = tx.Exec(ctx, `UPDATE mining_blocks SET status='FINALIZED',finalized_at=$2,updated_at=$2 WHERE block_id=$1`, blockID, now)
		} else {
			if pool.TotalReserved < block.RewardReserved || pool.Revision == math.MaxInt64 {
				return miningblock.Receipt{}, miningblock.ErrInvariant
			}
			repaid := min(block.RewardReserved, pool.RecoveryDebt)
			available := block.RewardReserved - repaid
			if pool.RemainingCapacity > math.MaxInt64-available {
				return miningblock.Receipt{}, miningblock.ErrOverflow
			}
			entry.BlockStatusAfter = miningblock.StatusCancelled
			entry.CapacityDelta = -available
			entry.PoolAfter = pool.RemainingCapacity + available
			_, err = tx.Exec(ctx, `UPDATE mining_blocks SET status='CANCELLED',cancelled_at=$2,reward_returned=$3,updated_at=$2 WHERE block_id=$1`, blockID, now, block.RewardReserved)
			if err == nil {
				_, err = tx.Exec(ctx, `UPDATE mining_block_reservations SET status='RELEASED',released=$2,updated_at=$3 WHERE block_id=$1`, blockID, block.RewardReserved, now)
			}
			if err == nil {
				_, err = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_reserved=total_reserved-$1,remaining_capacity=$2,recovery_debt=$3,revision=$4,updated_at=$5 WHERE pool_id='GLOBAL'`, block.RewardReserved, entry.PoolAfter, pool.RecoveryDebt-repaid, pool.Revision+1, now)
			}
		}
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if err = s.checkMiningBlockFailure("after_state_change"); err != nil {
			return miningblock.Receipt{}, err
		}
		receipt, err := s.appendMiningBlockEntryTx(ctx, tx, entry)
		if err != nil {
			return miningblock.Receipt{}, err
		}
		if action == miningblock.ActionCancel {
			err = appendRecoveryEntryTx(ctx, tx, emission.RecoveryEntry{
				ID: newContributionID("emission-recovery"), SourceType: "G18_BLOCK_CANCEL", SourceID: blockID, BlockEntryID: &entry.ID,
				ReservedDelta: -block.RewardReserved, RemainingBefore: pool.RemainingCapacity, RemainingAfter: entry.PoolAfter,
				DebtBefore: pool.RecoveryDebt, DebtAfter: pool.RecoveryDebt - min(block.RewardReserved, pool.RecoveryDebt), PoolRevision: pool.Revision + 1, CreatedAt: now,
			})
		}
		return receipt, err
	})
}

func (s *Store) appendMiningBlockEntryTx(ctx context.Context, tx pgx.Tx, entry miningblock.Entry) (miningblock.Receipt, error) {
	_, err := tx.Exec(ctx, `INSERT INTO mining_block_entries(entry_id,block_id,block_height,action,capacity_delta,pool_before,pool_after,block_status_before,block_status_after,rule_version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, entry.ID, entry.BlockID, entry.BlockHeight, entry.Action, entry.CapacityDelta, entry.PoolBefore, entry.PoolAfter, entry.BlockStatusBefore, entry.BlockStatusAfter, entry.RuleVersion, entry.CreatedAt)
	if err != nil {
		return miningblock.Receipt{}, err
	}
	if err = s.checkMiningBlockFailure("after_entry"); err != nil {
		return miningblock.Receipt{}, err
	}
	receipt := miningblock.Receipt{ID: newContributionID("mining-block-receipt"), BlockID: entry.BlockID, BlockHeight: entry.BlockHeight, Action: entry.Action, RewardReserved: miningblock.DevelopmentReward, PoolBefore: entry.PoolBefore, PoolAfter: entry.PoolAfter, BlockStatus: entry.BlockStatusAfter, RuleVersion: entry.RuleVersion, CreatedAt: entry.CreatedAt}
	_, err = tx.Exec(ctx, `INSERT INTO mining_block_receipts(receipt_id,entry_id,block_id,block_height,action,reward_reserved,pool_before,pool_after,block_status,rule_version,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, receipt.ID, entry.ID, receipt.BlockID, receipt.BlockHeight, receipt.Action, receipt.RewardReserved, receipt.PoolBefore, receipt.PoolAfter, receipt.BlockStatus, receipt.RuleVersion, receipt.CreatedAt)
	if err != nil {
		return miningblock.Receipt{}, err
	}
	if err = s.checkMiningBlockFailure("after_receipt"); err != nil {
		return miningblock.Receipt{}, err
	}
	return loadMiningReceiptTx(ctx, tx, entry.BlockID, entry.Action)
}

func loadMiningReceiptTx(ctx context.Context, tx pgx.Tx, blockID, action string) (miningblock.Receipt, error) {
	var value miningblock.Receipt
	err := tx.QueryRow(ctx, `SELECT receipt_id,block_id,block_height,action,reward_reserved,pool_before,pool_after,block_status,rule_version,created_at FROM mining_block_receipts WHERE block_id=$1 AND action=$2`, blockID, action).Scan(&value.ID, &value.BlockID, &value.BlockHeight, &value.Action, &value.RewardReserved, &value.PoolBefore, &value.PoolAfter, &value.BlockStatus, &value.RuleVersion, &value.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningblock.Receipt{}, miningblock.ErrInvariant
	}
	value.CreatedAt = value.CreatedAt.UTC()
	return value, err
}

func loadMiningBlockTx(ctx context.Context, tx pgx.Tx, blockID string, lock bool) (miningblock.Block, error) {
	query := `SELECT block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,finalized_at,cancelled_at,reward_reserved,reward_released,reward_returned,pool_revision_at_reservation,created_at,updated_at FROM mining_blocks WHERE block_id=$1`
	if lock {
		query += ` FOR UPDATE`
	}
	var b miningblock.Block
	err := tx.QueryRow(ctx, query, blockID).Scan(&b.ID, &b.Height, &b.CreateCommandID, &b.Status, &b.RuleVersion, &b.StartedAt, &b.ScheduledEndAt, &b.FinalizedAt, &b.CancelledAt, &b.RewardReserved, &b.RewardReleased, &b.RewardReturned, &b.PoolRevisionAtReservation, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningblock.Block{}, miningblock.ErrNotFound
	}
	if err != nil {
		return miningblock.Block{}, err
	}
	b.StartedAt = b.StartedAt.UTC()
	b.ScheduledEndAt = b.ScheduledEndAt.UTC()
	b.CreatedAt = b.CreatedAt.UTC()
	b.UpdatedAt = b.UpdatedAt.UTC()
	if b.FinalizedAt != nil {
		t := b.FinalizedAt.UTC()
		b.FinalizedAt = &t
	}
	if b.CancelledAt != nil {
		t := b.CancelledAt.UTC()
		b.CancelledAt = &t
	}
	return b, nil
}

func (s *Store) LoadMiningBlock(ctx context.Context, blockID string) (miningblock.Block, error) {
	if s == nil || s.pool == nil {
		return miningblock.Block{}, miningblock.ErrUnavailable
	}
	if !validMiningBlockID(blockID) {
		return miningblock.Block{}, miningblock.ErrInvalidCommand
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return miningblock.Block{}, err
	}
	defer tx.Rollback(ctx)
	b, err := loadMiningBlockTx(ctx, tx, blockID, false)
	if err != nil {
		return miningblock.Block{}, err
	}
	return b, tx.Commit(ctx)
}

var _ miningblock.Repository = (*Store)(nil)
