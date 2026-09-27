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

func (s *Store) blackIronFail(at string) error {
	if s.blackIronFailureInjector != nil {
		return s.blackIronFailureInjector(at)
	}
	return nil
}
