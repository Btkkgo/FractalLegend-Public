package postgres

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
)

const rewardIssuanceRule = "G21_P0_ISSUANCE_V1"

type MiningRewardIssuance struct {
	ID, SourceType, SourceKey, SettlementID, BlockInstanceID, CharacterID string
	Quantity                                                              int32
	MaterialDefinitionID, RuleVersion                                     string
	CreatedAt                                                             time.Time
	SourceDigest, Status                                                  string
}

type MiningRewardAudit struct {
	Balanced   bool
	Checked    int
	Mismatches []string
}

func miningRewardSourceDigest(v MiningRewardIssuance) (string, error) {
	return miningpower.IssuanceEvidenceDigest(miningpower.IssuanceEvidence{
		RuleVersion: v.RuleVersion, SourceType: v.SourceType, SourceKey: v.SourceKey,
		BlockInstanceID: v.BlockInstanceID, CharacterID: v.CharacterID, MaterialDefinitionID: v.MaterialDefinitionID,
		Quantity: v.Quantity, CreatedAt: v.CreatedAt})
}

func loadMiningRewardIssuanceTx(ctx context.Context, tx pgx.Tx, key string) (MiningRewardIssuance, error) {
	var v MiningRewardIssuance
	err := tx.QueryRow(ctx, `SELECT issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,
		quantity,material_definition_id,rule_version,created_at,source_digest,status
		FROM mining_reward_issuance_lots WHERE source_key=$1`, key).
		Scan(&v.ID, &v.SourceType, &v.SourceKey, &v.SettlementID, &v.BlockInstanceID, &v.CharacterID,
			&v.Quantity, &v.MaterialDefinitionID, &v.RuleVersion, &v.CreatedAt, &v.SourceDigest, &v.Status)
	v.CreatedAt = miningpower.CanonicalTime(v.CreatedAt)
	return v, err
}

// This fixture is intentionally unexported, has no route, and requires an
// isolated test database. It demonstrates the future atomic origin/projection
// contract without implementing allocation or a real reward grant.
func (s *Store) createMiningRewardIssuancePrerequisiteTEST(ctx context.Context, sourceKey, instanceID, characterID string, quantity int32) (MiningRewardIssuance, error) {
	var zero MiningRewardIssuance
	if s == nil || s.pool == nil || !validMiningBlockID(sourceKey) || !miningpower.ValidID(instanceID, 128) ||
		!validMiningBlockID(characterID) || quantity <= 0 {
		return zero, miningpower.ErrInvalidInput
	}
	var dbName string
	if err := s.pool.QueryRow(ctx, `SELECT current_database()`).Scan(&dbName); err != nil {
		return zero, err
	}
	if !strings.HasPrefix(dbName, "fractal_g9_test_") {
		return zero, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return zero, err
	}
	defer tx.Rollback(context.Background())
	prior, err := loadMiningRewardIssuanceTx(ctx, tx, sourceKey)
	if err == nil {
		if prior.BlockInstanceID != instanceID || prior.CharacterID != characterID || prior.Quantity != quantity {
			return zero, miningpower.ErrInvariant
		}
		digest, digestErr := miningRewardSourceDigest(prior)
		if digestErr != nil || digest != prior.SourceDigest {
			return zero, miningpower.ErrInvariant
		}
		var projected int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_inventory_projections p
			JOIN character_inventory_items i ON i.instance_id=p.instance_id
			JOIN characters c ON c.id=p.character_id AND c.revision>=p.revision_after
			WHERE p.issuance_id=$1 AND p.character_id=$2 AND p.block_instance_id=$3 AND
			p.quantity=$4 AND i.character_id=p.character_id AND i.quantity=p.quantity AND
			i.definition_id=p.definition_id AND p.definition_id=$5`, prior.ID, characterID, instanceID, quantity, prior.MaterialDefinitionID).Scan(&projected); err != nil || projected != 1 {
			return zero, miningpower.ErrInvariant
		}
		return prior, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return zero, err
	}
	var sealCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_power_input_seals WHERE block_instance_id=$1 AND eligibility='SETTLEMENT_INPUT'`, instanceID).Scan(&sealCount); err != nil {
		return zero, err
	}
	if sealCount != 1 {
		return zero, miningpower.ErrInvalidInput
	}
	var beneficiaryCount int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_power_beneficiary_bindings
		WHERE block_instance_id=$1 AND character_id=$2`, instanceID, characterID).Scan(&beneficiaryCount); err != nil {
		return zero, err
	}
	if beneficiaryCount == 0 {
		return zero, miningpower.ErrInvalidInput
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM characters WHERE id=$1 FOR UPDATE`, characterID).Scan(&revision); err != nil {
		return zero, err
	}
	if revision == math.MaxInt64 {
		return zero, miningpower.ErrInvariant
	}
	var definition string
	var legacyID, aliasCount int32
	if err = tx.QueryRow(ctx, `SELECT min(definition_id),min(legacy_id),count(*) FROM black_iron_identity_aliases`).Scan(&definition, &legacyID, &aliasCount); err != nil {
		return zero, err
	}
	// G19 can hold multiple reviewed legacy aliases; P0 must not guess which
	// material definition is canonical when authority is ambiguous.
	if aliasCount != 1 || definition == "" {
		return zero, miningpower.ErrInvalidInput
	}
	var slot int32
	if err = tx.QueryRow(ctx, `SELECT COALESCE(max(slot_index)+1,0) FROM character_inventory_items WHERE character_id=$1`, characterID).Scan(&slot); err != nil {
		return zero, err
	}
	now := miningpower.CanonicalTime(time.Now().UTC().Truncate(time.Microsecond))
	v := MiningRewardIssuance{ID: newContributionID("mining-reward-issuance"), SourceType: "MINING_REWARD",
		SourceKey: sourceKey, SettlementID: "G21_P0_TEST_PLACEHOLDER", BlockInstanceID: instanceID,
		CharacterID: characterID, Quantity: quantity, MaterialDefinitionID: definition, RuleVersion: rewardIssuanceRule,
		CreatedAt: now, Status: "P0_SYNTHETIC_ONLY"}
	v.SourceDigest, err = miningRewardSourceDigest(v)
	if err != nil {
		return zero, err
	}
	instance := newContributionID("mining-reward-item")
	_, err = tx.Exec(ctx, `INSERT INTO mining_reward_issuance_lots
		(issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,quantity,
		 material_definition_id,rule_version,created_at,source_digest,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, v.ID, v.SourceType, v.SourceKey, v.SettlementID,
		v.BlockInstanceID, v.CharacterID, v.Quantity, v.MaterialDefinitionID, v.RuleVersion, v.CreatedAt, v.SourceDigest, v.Status)
	if err != nil {
		return zero, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO mining_reward_inventory_projections
		(issuance_id,instance_id,character_id,block_instance_id,quantity,definition_id,revision_before,revision_after,created_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, v.ID, instance, characterID, instanceID, quantity, definition, revision, revision+1, now)
	if err != nil {
		return zero, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO character_inventory_items
		(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location)
		VALUES($1,$2,$3,$4,'黑铁矿石','MATERIAL',$5,$6,'INVENTORY')`, instance, characterID, definition, legacyID, quantity, slot)
	if err != nil {
		return zero, err
	}
	_, err = tx.Exec(ctx, `UPDATE characters SET revision=revision+1,updated_at=$2 WHERE id=$1 AND revision=$3`, characterID, now, revision)
	if err != nil {
		return zero, err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("issuance_before_commit"); err != nil {
			return zero, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return zero, err
	}
	if s.prerequisiteFailureInjector != nil {
		if err = s.prerequisiteFailureInjector("issuance_after_commit_before_response"); err != nil {
			return zero, err
		}
	}
	return v, nil
}

// Read-only source and projection audit; no repair or reward mutation.
func (s *Store) ReconcileMiningRewardIssuancePrerequisite(ctx context.Context) (MiningRewardAudit, error) {
	r := MiningRewardAudit{Balanced: true, Mismatches: []string{}}
	if s == nil || s.pool == nil {
		return r, miningpower.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return r, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT source_key FROM mining_reward_issuance_lots ORDER BY source_key`)
	if err != nil {
		return r, err
	}
	var keys []string
	for rows.Next() {
		var key string
		if err = rows.Scan(&key); err != nil {
			rows.Close()
			return r, err
		}
		keys = append(keys, key)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return r, err
	}
	for _, key := range keys {
		r.Checked++
		v, e := loadMiningRewardIssuanceTx(ctx, tx, key)
		if e != nil {
			return r, e
		}
		digest, e := miningRewardSourceDigest(v)
		if e != nil {
			return r, e
		}
		if digest != v.SourceDigest || v.SourceType != "MINING_REWARD" || v.RuleVersion != rewardIssuanceRule || v.Status != "P0_SYNTHETIC_ONLY" {
			r.Balanced = false
			r.Mismatches = append(r.Mismatches, key+": source digest or type")
		}
		var instance, owner, block, definition, itemDefinition, name, itemType, location string
		var quantity, itemQuantity, legacyID, aliasLegacy int32
		var before, after, current int64
		e = tx.QueryRow(ctx, `SELECT p.instance_id,p.character_id,p.block_instance_id,p.definition_id,p.quantity,
			p.revision_before,p.revision_after,i.definition_id,i.quantity,i.legacy_id,i.name,i.item_type,i.location,
			c.revision,a.legacy_id FROM mining_reward_inventory_projections p
			LEFT JOIN character_inventory_items i ON i.instance_id=p.instance_id
			LEFT JOIN characters c ON c.id=p.character_id
			LEFT JOIN black_iron_identity_aliases a ON a.definition_id=p.definition_id
			WHERE p.issuance_id=$1`, v.ID).Scan(&instance, &owner, &block, &definition, &quantity,
			&before, &after, &itemDefinition, &itemQuantity, &legacyID, &name, &itemType, &location, &current, &aliasLegacy)
		if e != nil || instance == "" || owner != v.CharacterID || block != v.BlockInstanceID ||
			definition != v.MaterialDefinitionID || quantity != v.Quantity || itemDefinition != definition ||
			itemQuantity != quantity || legacyID != aliasLegacy || name != "黑铁矿石" || itemType != "MATERIAL" ||
			location != "INVENTORY" || after != before+1 || current < after {
			r.Balanced = false
			r.Mismatches = append(r.Mismatches, key+": source/projection/inventory mismatch")
		}
	}
	var extras int
	err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_inventory_projections p
		LEFT JOIN mining_reward_issuance_lots l ON l.issuance_id=p.issuance_id
		WHERE l.issuance_id IS NULL`).Scan(&extras)
	if err != nil {
		return r, err
	}
	if extras != 0 {
		r.Balanced = false
		r.Mismatches = append(r.Mismatches, "extra projection")
	}
	return r, tx.Commit(ctx)
}
