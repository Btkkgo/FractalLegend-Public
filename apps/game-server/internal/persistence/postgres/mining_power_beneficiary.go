package postgres

import (
	"context"
	"errors"
	"time"

	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

const miningBeneficiaryVersion = "G21_P0_BENEFICIARY_V1"

// captureMiningBeneficiaryTx resolves a real character/owner row. The current
// G20 PlayerID column references characters.id, but the reward authority is
// the resolved CharacterID, never a string conversion or client field.
func captureMiningBeneficiaryTx(ctx context.Context, tx pgx.Tx, p miningpower.Principal, a miningpower.ValidatedMiningActivity, at time.Time) error {
	var characterID string
	err := tx.QueryRow(ctx, `SELECT id FROM characters WHERE id=$1 AND account_id=$2 FOR SHARE`, p.PlayerID, p.AccountID).Scan(&characterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return miningpower.ErrInvariant
	}
	if err != nil {
		return err
	}
	at = miningpower.CanonicalTime(at)
	evidence := miningpower.BeneficiaryEvidence{Version: miningBeneficiaryVersion, ActivityID: a.ActivityID,
		SourceEventID: a.SourceEventID, AccountID: p.AccountID, PlayerID: p.PlayerID, CharacterID: characterID,
		BlockInstanceID: a.BlockInstanceID, AuthoritySource: "CHARACTERS_OWNER_ROW_V1", BoundAt: at}
	digest, err := miningpower.BeneficiaryEvidenceDigest(evidence)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO mining_power_beneficiary_bindings
		(binding_id,activity_id,source_event_id,block_instance_id,account_id,player_id,character_id,authority_source,binding_version,bound_at,digest)
		VALUES($1,$2,$3,$4,$5,$6,$7,'CHARACTERS_OWNER_ROW_V1',$8,$9,$10)`,
		newContributionID("mining-beneficiary"), a.ActivityID, a.SourceEventID, a.BlockInstanceID,
		p.AccountID, p.PlayerID, characterID, miningBeneficiaryVersion, at, digest)
	return err
}
