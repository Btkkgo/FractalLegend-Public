package postgres

import (
	"context"
	"fmt"
	"math"

	"fractallegend/game-server/internal/miningblock"
	"github.com/jackc/pgx/v5"
)

func snapshotMiningBlocksTx(ctx context.Context, tx pgx.Tx) (miningblock.Snapshot, error) {
	result := miningblock.Snapshot{Blocks: []miningblock.Block{}, Reservations: []miningblock.Reservation{}, Entries: []miningblock.Entry{}, Receipts: []miningblock.Receipt{}}
	rows, err := tx.Query(ctx, `SELECT block_id FROM mining_blocks ORDER BY block_height`)
	if err != nil {
		return result, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return result, err
		}
		ids = append(ids, id)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	for _, id := range ids {
		b, e := loadMiningBlockTx(ctx, tx, id, false)
		if e != nil {
			return result, e
		}
		result.Blocks = append(result.Blocks, b)
	}
	rows, err = tx.Query(ctx, `SELECT block_id,amount,released,status,created_at,updated_at FROM mining_block_reservations ORDER BY block_id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v miningblock.Reservation
		if err = rows.Scan(&v.BlockID, &v.Amount, &v.Released, &v.Status, &v.CreatedAt, &v.UpdatedAt); err != nil {
			rows.Close()
			return result, err
		}
		v.CreatedAt = v.CreatedAt.UTC()
		v.UpdatedAt = v.UpdatedAt.UTC()
		result.Reservations = append(result.Reservations, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT entry_id,block_id,block_height,action,capacity_delta,pool_before,pool_after,block_status_before,block_status_after,rule_version,created_at FROM mining_block_entries ORDER BY block_height,created_at,entry_id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v miningblock.Entry
		if err = rows.Scan(&v.ID, &v.BlockID, &v.BlockHeight, &v.Action, &v.CapacityDelta, &v.PoolBefore, &v.PoolAfter, &v.BlockStatusBefore, &v.BlockStatusAfter, &v.RuleVersion, &v.CreatedAt); err != nil {
			rows.Close()
			return result, err
		}
		v.CreatedAt = v.CreatedAt.UTC()
		result.Entries = append(result.Entries, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	rows, err = tx.Query(ctx, `SELECT receipt_id,block_id,block_height,action,reward_reserved,pool_before,pool_after,block_status,rule_version,created_at FROM mining_block_receipts ORDER BY block_height,created_at,receipt_id`)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var v miningblock.Receipt
		if err = rows.Scan(&v.ID, &v.BlockID, &v.BlockHeight, &v.Action, &v.RewardReserved, &v.PoolBefore, &v.PoolAfter, &v.BlockStatus, &v.RuleVersion, &v.CreatedAt); err != nil {
			rows.Close()
			return result, err
		}
		v.CreatedAt = v.CreatedAt.UTC()
		result.Receipts = append(result.Receipts, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()
	return result, nil
}

func (s *Store) SnapshotMiningBlocks(ctx context.Context) (miningblock.Snapshot, error) {
	if s == nil || s.pool == nil {
		return miningblock.Snapshot{}, miningblock.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return miningblock.Snapshot{}, err
	}
	defer tx.Rollback(ctx)
	value, err := snapshotMiningBlocksTx(ctx, tx)
	if err != nil {
		return miningblock.Snapshot{}, err
	}
	return value, tx.Commit(ctx)
}

func (s *Store) ReconcileMiningBlocks(ctx context.Context) (miningblock.ReconciliationReport, error) {
	if s == nil || s.pool == nil {
		return miningblock.ReconciliationReport{}, miningblock.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return miningblock.ReconciliationReport{}, err
	}
	defer tx.Rollback(ctx)
	pool, err := loadEmissionPoolTx(ctx, tx, false)
	if err != nil {
		return miningblock.ReconciliationReport{}, err
	}
	snapshot, err := snapshotMiningBlocksTx(ctx, tx)
	if err != nil {
		return miningblock.ReconciliationReport{}, err
	}
	report := miningblock.ReconciliationReport{Balanced: true, Checked: len(snapshot.Blocks)}
	bad := func(msg string) { report.Balanced = false; report.Mismatches = append(report.Mismatches, msg) }
	if validateMiningPool(pool) != nil {
		bad("pool conservation")
	}
	blocks := make(map[string]miningblock.Block, len(snapshot.Blocks))
	reservations := make(map[string]miningblock.Reservation, len(snapshot.Reservations))
	entries := make(map[string]map[string]miningblock.Entry)
	receipts := make(map[string]map[string]miningblock.Receipt)
	var active int64
	for i, b := range snapshot.Blocks {
		if b.Height != int64(i+1) {
			bad(fmt.Sprintf("block %s height gap or duplicate", b.ID))
		}
		if b.RewardReserved <= 0 || b.RewardReleased != 0 || b.RuleVersion != miningblock.DevelopmentRuleVersion {
			bad(fmt.Sprintf("block %s invalid reward or rule", b.ID))
		}
		blocks[b.ID] = b
	}
	for _, r := range snapshot.Reservations {
		if _, ok := blocks[r.BlockID]; !ok {
			bad(fmt.Sprintf("reservation %s has no block", r.BlockID))
		}
		reservations[r.BlockID] = r
	}
	for _, e := range snapshot.Entries {
		b, ok := blocks[e.BlockID]
		if !ok {
			bad(fmt.Sprintf("entry %s has no block", e.ID))
			continue
		}
		if _, ok := entries[e.BlockID]; !ok {
			entries[e.BlockID] = make(map[string]miningblock.Entry)
		}
		entries[e.BlockID][e.Action] = e
		if e.BlockHeight != b.Height || e.RuleVersion != b.RuleVersion || e.PoolAfter != e.PoolBefore-e.CapacityDelta {
			bad(fmt.Sprintf("entry %s differs from block", e.ID))
		}
	}
	for _, r := range snapshot.Receipts {
		if _, ok := receipts[r.BlockID]; !ok {
			receipts[r.BlockID] = make(map[string]miningblock.Receipt)
		}
		receipts[r.BlockID][r.Action] = r
		e, ok := entries[r.BlockID][r.Action]
		if !ok || r.BlockHeight != e.BlockHeight || r.RuleVersion != e.RuleVersion || r.RewardReserved != blocks[r.BlockID].RewardReserved || r.PoolBefore != e.PoolBefore || r.PoolAfter != e.PoolAfter || r.BlockStatus != e.BlockStatusAfter || !r.CreatedAt.Equal(e.CreatedAt) {
			bad(fmt.Sprintf("receipt %s differs from entry", r.ID))
		}
	}
	if len(snapshot.Entries) != len(snapshot.Receipts) {
		bad("entry/receipt cardinality mismatch")
	}
	for _, b := range snapshot.Blocks {
		r, ok := reservations[b.ID]
		if !ok || r.Amount != b.RewardReserved {
			bad(fmt.Sprintf("block %s lacks matching reservation", b.ID))
			continue
		}
		open, ok := entries[b.ID][miningblock.ActionOpen]
		if !ok || open.CapacityDelta != b.RewardReserved || open.BlockStatusAfter != miningblock.StatusOpen || open.BlockStatusBefore != "" {
			bad(fmt.Sprintf("block %s lacks open entry", b.ID))
		}
		if len(entries[b.ID]) != len(receipts[b.ID]) {
			bad(fmt.Sprintf("block %s lacks receipt", b.ID))
		}
		switch b.Status {
		case miningblock.StatusOpen:
			if r.Status != "ACTIVE" || r.Released != 0 || b.RewardReturned != 0 || len(entries[b.ID]) != 1 {
				bad(fmt.Sprintf("open block %s has invalid reservation", b.ID))
			}
		case miningblock.StatusFinalized:
			e, ok := entries[b.ID][miningblock.ActionFinalize]
			if r.Status != "ACTIVE" || r.Released != 0 || b.RewardReturned != 0 || !ok || e.CapacityDelta != 0 || len(entries[b.ID]) != 2 {
				bad(fmt.Sprintf("finalized block %s has invalid reservation", b.ID))
			}
		case miningblock.StatusCancelled:
			e, ok := entries[b.ID][miningblock.ActionCancel]
			if r.Status != "RELEASED" || r.Released != r.Amount || b.RewardReturned != r.Amount || !ok || e.CapacityDelta > 0 || e.CapacityDelta < -r.Amount || len(entries[b.ID]) != 2 {
				bad(fmt.Sprintf("cancelled block %s lacks release", b.ID))
			}
		default:
			bad(fmt.Sprintf("block %s has unknown status", b.ID))
		}
		if r.Status == "ACTIVE" {
			if active > math.MaxInt64-r.Amount {
				bad("active reservation sum overflow")
			} else {
				active += r.Amount
			}
		}
	}
	if active != pool.TotalReserved {
		bad("active block reservations differ from pool reserved")
	}
	emissionSnapshot, err := snapshotEmissionTx(ctx, tx)
	if err != nil {
		return miningblock.ReconciliationReport{}, err
	}
	for _, mismatch := range reconcileRecoveryTx(ctx, tx, pool, emissionSnapshot.RecoveryEntries) {
		bad(mismatch)
	}
	if err = tx.Commit(ctx); err != nil {
		return miningblock.ReconciliationReport{}, err
	}
	return report, nil
}
