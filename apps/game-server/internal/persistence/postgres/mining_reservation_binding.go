package postgres

import (
	"context"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

// appendReservationBindingTx runs in the G18 OPEN transaction. The entry and
// receipt are source identities, while BlockID is retained for display only.
func appendReservationBindingTx(ctx context.Context, tx pgx.Tx, instanceID string, entry miningblock.Entry, receipt miningblock.Receipt) error {
	if entry.Action != miningblock.ActionOpen || receipt.Action != miningblock.ActionOpen ||
		entry.BlockID != receipt.BlockID || entry.CapacityDelta != receipt.RewardReserved ||
		entry.RuleVersion != receipt.RuleVersion || instanceID == "" {
		return miningblock.ErrInvariant
	}
	evidence := miningpower.ReservationBindingEvidence{
		Version: miningpower.ReservationBindingVersion, InstanceID: instanceID, SourceID: entry.ID,
		ReceiptID: receipt.ID, BlockID: entry.BlockID, RuleVersion: entry.RuleVersion, Amount: entry.CapacityDelta,
	}
	digest, err := miningpower.ReservationBindingEvidenceDigest(evidence)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO mining_reservation_instance_bindings
		(binding_id,reservation_source_id,reservation_receipt_id,block_instance_id,display_block_id,g18_rule_version,reservation_amount,binding_version,evidence_digest,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		newContributionID("mining-binding"), entry.ID, receipt.ID, instanceID, entry.BlockID,
		entry.RuleVersion, entry.CapacityDelta, miningpower.ReservationBindingVersion, digest, entry.CreatedAt)
	return err
}
