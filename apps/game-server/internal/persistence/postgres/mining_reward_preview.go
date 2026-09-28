package postgres

import (
	"context"
	"errors"
	"strings"

	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/miningreward"
	"github.com/jackc/pgx/v5"
)

// MiningRewardAllocationPreview is deliberately distinct from a settlement
// receipt. It contains no CommandID, SettlementID, issuance or inventory ID.
type MiningRewardAllocationPreview struct {
	Status, BlockInstanceID, SealDigest, Eligibility string
	ReservedAmount, TotalPower, TotalOre             int64
	Grants                                           []miningreward.Grant
}

// PreviewMiningRewardTEST reads persisted authorities in one read-only
// snapshot. It never invokes the settlement writer or reserves inventory.
func (s *Store) PreviewMiningRewardTEST(ctx context.Context, instanceID, expectedSealDigest string) (MiningRewardAllocationPreview, error) {
	var empty MiningRewardAllocationPreview
	if s == nil || s.pool == nil || !miningpower.ValidID(instanceID, 128) || len(expectedSealDigest) != 64 {
		return empty, miningpower.ErrInvalidInput
	}
	var dbName string
	if err := s.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return empty, err
	}
	if !strings.HasPrefix(dbName, "fractal_g9_test_") {
		return empty, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(context.Background())
	seal, err := loadMiningPowerSealTx(ctx, tx, instanceID)
	if err != nil {
		return empty, err
	}
	if seal.BlockInstanceID != instanceID || seal.CanonicalDigest != expectedSealDigest ||
		seal.ParticipantCount > miningpower.TestParticipantCap || seal.ActivityCount > settlementMaxActivities {
		return empty, ErrMiningRewardConflict
	}
	if _, err := validateMiningPowerSealSnapshotTx(ctx, tx, seal); err != nil {
		return empty, ErrMiningRewardInvariant
	}
	// A historical P0 seal may preserve positive power without authoritative
	// beneficiaries. Such evidence cannot be advertised as eligible allocation.
	if seal.TotalValidMiningPower > 0 && seal.Eligibility != "SETTLEMENT_INPUT" {
		return empty, ErrMiningRewardInvariant
	}
	var sourceID, bindingDigest, receiptID, displayID, ruleVersion, bindingVersion string
	var reward int64
	err = tx.QueryRow(ctx, `SELECT reservation_source_id,evidence_digest,reservation_receipt_id,
		display_block_id,g18_rule_version,binding_version,reservation_amount
		FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, instanceID).
		Scan(&sourceID, &bindingDigest, &receiptID, &displayID, &ruleVersion, &bindingVersion, &reward)
	if err != nil {
		return empty, err
	}
	actualDigest, err := miningpower.ReservationBindingEvidenceDigest(miningpower.ReservationBindingEvidence{
		Version: bindingVersion, InstanceID: instanceID, SourceID: sourceID,
		ReceiptID: receiptID, BlockID: displayID, RuleVersion: ruleVersion, Amount: reward,
	})
	if err != nil || reward <= 0 || actualDigest != bindingDigest ||
		sourceID != seal.G18ReservationSourceID || bindingDigest != seal.G18ReservationEvidenceDigest ||
		displayID != seal.DisplayBlockID || ruleVersion != seal.G18RuleVersion {
		return empty, ErrMiningRewardInvariant
	}
	var debt int64
	if err := tx.QueryRow(ctx, `SELECT recovery_debt FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&debt); err != nil {
		return empty, err
	}
	allocation, err := miningreward.AllocateTEST(reward, seal.ParticipantWeights)
	if err != nil {
		return empty, errors.Join(ErrMiningRewardInvariant, err)
	}
	preview := MiningRewardAllocationPreview{Status: "ALLOCATION_PREVIEW", BlockInstanceID: instanceID,
		SealDigest: seal.CanonicalDigest, Eligibility: "ELIGIBLE", ReservedAmount: reward,
		TotalPower: allocation.Power, TotalOre: allocation.Total, Grants: allocation.Grants}
	if allocation.Power == 0 {
		preview.Eligibility = "NO_ELIGIBLE_POWER"
	} else if debt > 0 {
		preview.Eligibility = "RECOVERY_DEBT_PAUSED"
	}
	return preview, nil
}
