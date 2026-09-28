package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"fractallegend/game-server/internal/blackiron"
	"fractallegend/game-server/internal/persistence"
	"github.com/jackc/pgx/v5"
)

// MigrateBlackIronInventory is an internal server operation, not a client
// command. Quantities come exclusively from locked persisted inventory.
// Version + character identity is the deterministic replay key.
func (s *Store) MigrateBlackIronInventory(ctx context.Context, characterID string) (blackiron.Receipt, error) {
	if s == nil || s.pool == nil {
		return blackiron.Receipt{}, persistence.ErrUnavailable
	}
	if strings.TrimSpace(characterID) == "" || len(characterID) > 128 {
		return blackiron.Receipt{}, blackiron.ErrInvalid
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	defer tx.Rollback(ctx)
	var frozen bool
	if err = tx.QueryRow(ctx, "SELECT frozen FROM black_iron_migration_catalog WHERE singleton FOR UPDATE").Scan(&frozen); err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	var revision int64
	if err = tx.QueryRow(ctx, `SELECT revision FROM characters WHERE id=$1 FOR UPDATE`, characterID).Scan(&revision); errors.Is(err, pgx.ErrNoRows) {
		return blackiron.Receipt{}, persistence.ErrNotFound
	} else if err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	aliases, err := loadBlackIronAliases(ctx, tx)
	if err != nil {
		return blackiron.Receipt{}, err
	}
	if len(aliases) == 0 {
		return blackiron.Receipt{}, blackiron.ErrNotConfigured
	}
	existing, err := loadBlackIronReceipt(ctx, tx, characterID)
	if err == nil {
		if err = checkBlackIronReceipt(ctx, tx, existing, aliases); err != nil {
			return blackiron.Receipt{}, err
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return blackiron.Receipt{}, err
	}
	rewardInstances, err := validatedBlackIronRewardInstances(ctx, tx, characterID)
	if err != nil {
		return blackiron.Receipt{}, err
	}
	// Only actual legacy state may be converted; an unmarked canonical record
	// cannot be guessed to be an already completed migration.
	rows, err := tx.Query(ctx, `SELECT instance_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location
 FROM character_inventory_items WHERE character_id=$1 ORDER BY slot_index,instance_id FOR UPDATE`, characterID)
	if err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	receipt := blackiron.Receipt{Version: blackiron.Version, CharacterID: characterID, TargetKind: blackiron.Kind, Status: "COMPLETE", RevisionBefore: revision, RevisionAfter: revision, Changes: []blackiron.Change{}}
	for rows.Next() {
		var a blackiron.Asset
		if err = rows.Scan(&a.InstanceID, &a.DefinitionID, &a.LegacyID, &a.Name, &a.ItemType, &a.Quantity, &a.SlotIndex, &a.Location); err != nil {
			rows.Close()
			return blackiron.Receipt{}, classifyError(err)
		}
		alias, ok := aliases[a.DefinitionID]
		if !ok {
			continue
		}
		if a.Name == blackiron.Name {
			if rewardInstances[a.InstanceID] {
				continue
			}
			rows.Close()
			return blackiron.Receipt{}, blackiron.ErrMismatch
		}
		target, e := blackiron.Normalize(a, alias)
		if e != nil {
			rows.Close()
			return blackiron.Receipt{}, e
		}
		receipt.PreTotal, e = blackiron.AddQuantity(receipt.PreTotal, a.Quantity)
		if e != nil {
			rows.Close()
			return blackiron.Receipt{}, e
		}
		receipt.PostTotal, e = blackiron.AddQuantity(receipt.PostTotal, target.Quantity)
		if e != nil {
			rows.Close()
			return blackiron.Receipt{}, e
		}
		receipt.Changes = append(receipt.Changes, blackiron.Change{Source: a, Target: target})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	if len(receipt.Changes) > 0 {
		if revision == math.MaxInt64 {
			return blackiron.Receipt{}, blackiron.ErrInvalid
		}
		receipt.RevisionAfter++
	}
	if err = s.blackIronFail("before_items"); err != nil {
		return blackiron.Receipt{}, err
	}
	for _, change := range receipt.Changes {
		result, e := tx.Exec(ctx, `UPDATE character_inventory_items SET name=$1 WHERE instance_id=$2 AND character_id=$3`, blackiron.Name, change.Source.InstanceID, characterID)
		if e != nil {
			return blackiron.Receipt{}, classifyError(e)
		}
		if result.RowsAffected() != 1 {
			return blackiron.Receipt{}, blackiron.ErrMismatch
		}
	}
	if err = s.blackIronFail("after_items"); err != nil {
		return blackiron.Receipt{}, err
	}
	if len(receipt.Changes) > 0 {
		if _, err = tx.Exec(ctx, `UPDATE characters SET revision=$1,updated_at=now() WHERE id=$2`, receipt.RevisionAfter, characterID); err != nil {
			return blackiron.Receipt{}, classifyError(err)
		}
	}
	if err = s.blackIronFail("after_revision"); err != nil {
		return blackiron.Receipt{}, err
	}
	// PostgreSQL timestamps have microsecond precision: replay is byte-stable.
	receipt.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	receipt.CompletedAt = receipt.CreatedAt
	raw, err := json.Marshal(receipt)
	if err != nil {
		return blackiron.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO black_iron_migration_receipts(version,character_id,status,target_kind,pre_total,post_total,receipt,created_at,completed_at)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, receipt.Version, receipt.CharacterID, receipt.Status, receipt.TargetKind, receipt.PreTotal, receipt.PostTotal, raw, receipt.CreatedAt, receipt.CompletedAt); err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	for _, change := range receipt.Changes {
		if _, err = tx.Exec(ctx, `INSERT INTO black_iron_migration_assets(instance_id,version,character_id,definition_id,legacy_id,quantity) VALUES($1,$2,$3,$4,$5,$6)`, change.Target.InstanceID, receipt.Version, characterID, change.Target.DefinitionID, change.Target.LegacyID, change.Target.Quantity); err != nil {
			return blackiron.Receipt{}, classifyError(err)
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE black_iron_migration_catalog SET frozen=true WHERE singleton`); err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	if err = s.blackIronFail("after_receipt"); err != nil {
		return blackiron.Receipt{}, err
	}
	if err = checkBlackIronReceipt(ctx, tx, receipt, aliases); err != nil {
		return blackiron.Receipt{}, err
	}
	if err = s.blackIronFail("before_commit"); err != nil {
		return blackiron.Receipt{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return blackiron.Receipt{}, classifyError(err)
	}
	// Test seam also models a lost acknowledgement after a successful commit.
	if err = s.blackIronFail("after_commit"); err != nil {
		return blackiron.Receipt{}, err
	}
	return receipt, nil
}

// A first G19 conversion can race after a committed G21 reward. Only exact,
// independently committed reward provenance may be excluded from conversion.
// Merely finding a projection marker must never legitimize a corrupt stack.
func validatedBlackIronRewardInstances(ctx context.Context, tx pgx.Tx, characterID string) (map[string]bool, error) {
	instances := map[string]bool{}
	present, err := hasMiningRewardProvenanceTx(ctx, tx)
	if err != nil || !present {
		return instances, err
	}
	rows, err := tx.Query(ctx, `SELECT p.instance_id,l.source_key FROM mining_reward_inventory_projections p
		JOIN mining_reward_issuance_lots l USING(issuance_id) WHERE p.character_id=$1 ORDER BY p.instance_id`, characterID)
	if err != nil {
		return nil, err
	}
	type source struct{ instance, key string }
	var sources []source
	for rows.Next() {
		var v source
		if err = rows.Scan(&v.instance, &v.key); err != nil {
			rows.Close()
			return nil, err
		}
		sources = append(sources, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, source := range sources {
		lot, err := loadMiningRewardIssuanceTx(ctx, tx, source.key)
		if err != nil {
			return nil, err
		}
		digest, err := miningRewardSourceDigest(lot)
		if err != nil || digest != lot.SourceDigest || lot.SourceType != "MINING_REWARD" || lot.CharacterID != characterID {
			return nil, blackiron.ErrMismatch
		}
		validOrigin := lot.Status == "P0_SYNTHETIC_ONLY" && lot.RuleVersion == rewardIssuanceRule && lot.SettlementID == "G21_P0_TEST_PLACEHOLDER"
		if lot.Status == "G21_SETTLED" && lot.RuleVersion == settlementIssuanceVersion {
			var command string
			if err = tx.QueryRow(ctx, `SELECT command_id FROM mining_reward_settlement_receipts WHERE settlement_id=$1`, lot.SettlementID).Scan(&command); err != nil {
				return nil, blackiron.ErrMismatch
			}
			if _, err = loadMiningRewardReceiptTx(ctx, tx, command); err != nil {
				return nil, blackiron.ErrMismatch
			}
			validOrigin = true
		}
		if !validOrigin {
			return nil, blackiron.ErrMismatch
		}
		var exact bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM mining_reward_inventory_projections p
			JOIN character_inventory_items i ON i.instance_id=p.instance_id
			JOIN characters c ON c.id=i.character_id
			JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id
			WHERE p.issuance_id=$1 AND p.instance_id=$2 AND p.character_id=$3 AND i.character_id=$3
			AND p.block_instance_id=$4 AND p.quantity=$5 AND i.quantity=$5
			AND p.definition_id=$6 AND i.definition_id=$6 AND i.legacy_id=a.legacy_id
			AND i.name='黑铁矿石' AND i.item_type='MATERIAL' AND i.location='INVENTORY' AND i.equipment_slot IS NULL
			AND c.revision>=p.revision_after AND p.revision_after=p.revision_before+1
			AND NOT EXISTS(SELECT 1 FROM black_iron_migration_assets m WHERE m.instance_id=i.instance_id))`,
			lot.ID, source.instance, characterID, lot.BlockInstanceID, lot.Quantity, lot.MaterialDefinitionID).Scan(&exact)
		if err != nil || !exact {
			return nil, blackiron.ErrMismatch
		}
		instances[source.instance] = true
	}
	return instances, nil
}

func (s *Store) blackIronFail(at string) error {
	if s.blackIronFailureInjector != nil {
		return s.blackIronFailureInjector(at)
	}
	return nil
}
