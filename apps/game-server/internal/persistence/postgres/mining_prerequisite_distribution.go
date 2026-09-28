package postgres

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"fractallegend/game-server/internal/emission"
	"github.com/jackc/pgx/v5"
)

// transferReservedToDistributedPrerequisiteTEST proves the accounting and
// independent recovery source in an isolated synthetic database. It creates
// no mining reward, inventory item, player balance, or production entrypoint.
func (s *Store) transferReservedToDistributedPrerequisiteTEST(ctx context.Context, sourceID string) error {
	if s == nil || s.pool == nil || !validMiningBlockID(sourceID) {
		return emission.ErrInvariant
	}
	var dbName string
	if err := s.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return err
	}
	if !strings.HasPrefix(dbName, "fractal_g9_test_") {
		return emission.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	pool, err := loadEmissionPoolTx(ctx, tx, true)
	if err != nil {
		return err
	}
	if !emissionRecoveryConserved(pool) {
		return emission.ErrInvariant
	}
	var prior string
	err = tx.QueryRow(ctx, `SELECT distribution_id FROM mining_prerequisite_distributions WHERE reservation_source_id=$1`, sourceID).Scan(&prior)
	if err == nil {
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	var blockID, instanceID, status, reservationStatus string
	var amount, blockReward, reservationAmount int64
	err = tx.QueryRow(ctx, `SELECT b.display_block_id,b.block_instance_id,m.status,r.status,b.reservation_amount,m.reward_reserved,r.amount
		FROM mining_reservation_instance_bindings b JOIN mining_blocks m ON m.block_id=b.display_block_id
		JOIN mining_block_reservations r ON r.block_id=m.block_id
		WHERE b.reservation_source_id=$1 FOR UPDATE OF m,r`, sourceID).
		Scan(&blockID, &instanceID, &status, &reservationStatus, &amount, &blockReward, &reservationAmount)
	if err != nil {
		return err
	}
	if blockID == "" || instanceID == "" || status != "FINALIZED" || reservationStatus != "ACTIVE" ||
		amount != blockReward || amount != reservationAmount || amount <= 0 || pool.RecoveryDebt != 0 ||
		pool.TotalReserved < amount || pool.TotalDistributed > math.MaxInt64-amount || pool.Revision == math.MaxInt64 {
		return emission.ErrInvariant
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	id := newContributionID("p0-test-distribution")
	_, err = tx.Exec(ctx, `UPDATE black_iron_emission_pools SET total_reserved=$1,total_distributed=$2,
		revision=$3,updated_at=$4 WHERE pool_id='GLOBAL'`, pool.TotalReserved-amount,
		pool.TotalDistributed+amount, pool.Revision+1, now)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO mining_prerequisite_distributions
		(distribution_id,reservation_source_id,amount,pool_revision,created_at) VALUES($1,$2,$3,$4,$5)`,
		id, sourceID, amount, pool.Revision+1, now)
	if err != nil {
		return err
	}
	err = appendRecoveryEntryTx(ctx, tx, emission.RecoveryEntry{
		ID: newContributionID("emission-recovery"), SourceType: "G21_P0_TEST_DISTRIBUTION",
		SourceID: sourceID, TestDistributionID: &id, ReservedDelta: -amount,
		DistributedDelta: amount, DistributedBefore: pool.TotalDistributed,
		DistributedAfter: pool.TotalDistributed + amount, RemainingBefore: pool.RemainingCapacity,
		RemainingAfter: pool.RemainingCapacity, DebtBefore: 0, DebtAfter: 0,
		PoolRevision: pool.Revision + 1, CreatedAt: now,
	})
	if err != nil {
		return err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("distribution_before_commit"); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("distribution_after_commit_before_response"); err != nil {
			return err
		}
	}
	return nil
}

// A read-only boundary for a future settlement service. A positive recovery
// debt suspends new work; historical completed replay is a separate path.
func (s *Store) MiningSettlementPrerequisiteReady(ctx context.Context) (bool, error) {
	if s == nil || s.pool == nil {
		return false, emission.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	pool, err := loadEmissionPoolTx(ctx, tx, false)
	if err != nil {
		return false, err
	}
	if !emissionRecoveryConserved(pool) {
		return false, emission.ErrInvariant
	}
	return pool.RecoveryDebt == 0, tx.Commit(ctx)
}
