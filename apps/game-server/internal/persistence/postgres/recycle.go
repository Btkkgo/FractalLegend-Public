package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fractallegend/game-server/internal/recycle"
	"github.com/jackc/pgx/v5"
)

func (s *Store) LoadRecycle(ctx context.Context, operationID string) (recycle.Receipt, error) {
	if s == nil || s.pool == nil {
		return recycle.Receipt{}, recycle.ErrInvalidIntent
	}
	return loadRecycle(ctx, s.pool, operationID)
}

type recycleQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func loadRecycle(ctx context.Context, q recycleQuerier, operationID string) (recycle.Receipt, error) {
	var r recycle.Receipt
	var materials []byte
	err := q.QueryRow(ctx, `SELECT operation_id,player_id,item_instance_id,item_template_id,item_revision,
        rule_id,rule_version,requested_at,materials_awarded,reputation_awarded,fb_awarded,
        contribution_awarded,black_iron_awarded,created_at FROM recycle_receipts WHERE operation_id=$1`, operationID).
		Scan(&r.OperationID, &r.PlayerID, &r.ItemInstanceID, &r.ItemTemplateID, &r.ItemRevision,
			&r.RuleID, &r.RuleVersion, &r.RequestedAt, &materials, &r.ReputationAwarded,
			&r.FBAwarded, &r.ContributionAwarded, &r.BlackIronAwarded, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return recycle.Receipt{}, recycle.ErrNotFound
	}
	if err != nil {
		return recycle.Receipt{}, err
	}
	if err = json.Unmarshal(materials, &r.MaterialsAwarded); err != nil {
		return recycle.Receipt{}, recycle.ErrReconciliation
	}
	r.RequestedAt = r.RequestedAt.UTC()
	r.CreatedAt = r.CreatedAt.UTC()
	return r, nil
}

func (s *Store) recycleFailure(at string) error {
	if s.recycleFailureInjector != nil {
		return s.recycleFailureInjector(at)
	}
	return nil
}

// PostRecycle is internal only. No gameplay route calls it in G16.
func (s *Store) PostRecycle(ctx context.Context, intent recycle.Intent, rule recycle.Rule) (recycle.Receipt, error) {
	if s == nil || s.pool == nil || !intent.Valid() {
		return recycle.Receipt{}, recycle.ErrInvalidIntent
	}
	checked, err := recycle.NewRegistry([]recycle.Rule{rule})
	if err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = checked.Resolve(intent.RuleID); err != nil {
		return recycle.Receipt{}, err
	}
	requestedAt := intent.RequestedAt.UTC().Truncate(time.Microsecond)
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return recycle.Receipt{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "g16-operation:"+intent.OperationID); err != nil {
		return recycle.Receipt{}, err
	}
	previous, err := loadRecycle(ctx, tx, intent.OperationID)
	if err == nil {
		if previous.PlayerID != intent.PlayerID || previous.ItemInstanceID != intent.ItemInstanceID ||
			previous.ItemRevision != intent.ExpectedItemRevision || previous.RuleID != intent.RuleID ||
			!previous.RequestedAt.Equal(requestedAt) {
			return recycle.Receipt{}, recycle.ErrConflict
		}
		return previous, nil
	}
	if !errors.Is(err, recycle.ErrNotFound) {
		return recycle.Receipt{}, err
	}
	var owner string
	if err = tx.QueryRow(ctx, `SELECT id FROM characters WHERE id=$1 FOR UPDATE`, intent.PlayerID).Scan(&owner); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return recycle.Receipt{}, recycle.ErrOwnership
		}
		return recycle.Receipt{}, err
	}
	var itemOwner, templateID, itemType, location string
	var revision int64
	var quantity, slotIndex int
	err = tx.QueryRow(ctx, `SELECT i.character_id,i.definition_id,i.item_type,i.location,i.quantity,i.slot_index,l.revision
        FROM character_inventory_items i JOIN item_instance_lifecycle l ON l.instance_id=i.instance_id
        WHERE i.instance_id=$1 FOR UPDATE OF i,l`, intent.ItemInstanceID).
		Scan(&itemOwner, &templateID, &itemType, &location, &quantity, &slotIndex, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		var consumed bool
		queryErr := tx.QueryRow(ctx, `SELECT consumed FROM item_instance_lifecycle WHERE instance_id=$1`, intent.ItemInstanceID).Scan(&consumed)
		if queryErr == nil && consumed {
			return recycle.Receipt{}, recycle.ErrConsumed
		}
		return recycle.Receipt{}, recycle.ErrNotFound
	}
	if err != nil {
		return recycle.Receipt{}, err
	}
	if itemOwner != intent.PlayerID {
		return recycle.Receipt{}, recycle.ErrOwnership
	}
	if err = s.recycleFailure("after_ownership"); err != nil {
		return recycle.Receipt{}, err
	}
	if revision != intent.ExpectedItemRevision {
		return recycle.Receipt{}, recycle.ErrRevision
	}
	if itemType != "EQUIPMENT" || quantity != 1 || location != "INVENTORY" || templateID != rule.TemplateID || slotIndex < 0 {
		return recycle.Receipt{}, recycle.ErrLocked
	}
	var locked bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM trade_item_locks WHERE item_instance_id=$1)
        OR EXISTS(SELECT 1 FROM trade_offers o JOIN trade_sessions t ON t.trade_id=o.trade_id
            WHERE o.item_instance_id=$1 AND t.state IN ('NEGOTIATING','READY_TO_SETTLE'))
        OR EXISTS(SELECT 1 FROM character_equipment WHERE item_instance_id=$1)`, intent.ItemInstanceID).Scan(&locked)
	if err != nil {
		return recycle.Receipt{}, err
	}
	if locked {
		return recycle.Receipt{}, recycle.ErrLocked
	}
	if err = s.recycleFailure("after_rule_resolution"); err != nil {
		return recycle.Receipt{}, err
	}
	if err = s.recycleFailure("before_item_consume"); err != nil {
		return recycle.Receipt{}, err
	}
	command, err := tx.Exec(ctx, `UPDATE item_instance_lifecycle SET consumed=true WHERE instance_id=$1 AND NOT consumed`, intent.ItemInstanceID)
	if err != nil {
		return recycle.Receipt{}, err
	}
	if command.RowsAffected() != 1 {
		return recycle.Receipt{}, recycle.ErrConsumed
	}
	command, err = tx.Exec(ctx, `DELETE FROM character_inventory_items WHERE instance_id=$1 AND character_id=$2`, intent.ItemInstanceID, intent.PlayerID)
	if err != nil {
		return recycle.Receipt{}, err
	}
	if command.RowsAffected() != 1 {
		return recycle.Receipt{}, recycle.ErrConsumed
	}
	if _, err = tx.Exec(ctx, `UPDATE characters SET revision=revision+1,updated_at=now() WHERE id=$1`, intent.PlayerID); err != nil {
		return recycle.Receipt{}, err
	}
	if err = s.recycleFailure("after_item_consume"); err != nil {
		return recycle.Receipt{}, err
	}
	now := time.Now().UTC()
	r := recycle.Receipt{OperationID: intent.OperationID, PlayerID: intent.PlayerID,
		ItemInstanceID: intent.ItemInstanceID, ItemTemplateID: templateID, ItemRevision: revision,
		RuleID: rule.ID, RuleVersion: rule.Version, RequestedAt: requestedAt,
		MaterialsAwarded: rule.MaterialOutputs, ReputationAwarded: rule.ReputationReward, CreatedAt: now}
	for _, output := range r.MaterialsAwarded {
		if output.Quantity == 0 {
			continue
		}
		var nextSlot int
		if err = tx.QueryRow(ctx, `SELECT COALESCE(max(slot_index),-1)+1 FROM character_inventory_items WHERE character_id=$1`, intent.PlayerID).Scan(&nextSlot); err != nil {
			return recycle.Receipt{}, err
		}
		id := fmt.Sprintf("g16-%x", sha256.Sum256([]byte(intent.OperationID+":"+output.ID)))
		if _, err = tx.Exec(ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location)
            VALUES($1,$2,$3,0,$4,'MATERIAL',$5,$6,'INVENTORY')`, id, intent.PlayerID, output.ID, output.ID, output.Quantity, nextSlot); err != nil {
			return recycle.Receipt{}, err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO recycle_material_credits(operation_id,material_id,quantity,inventory_instance_id,created_at) VALUES($1,$2,$3,$4,$5)`, intent.OperationID, output.ID, output.Quantity, id, now); err != nil {
			return recycle.Receipt{}, err
		}
	}
	if err = s.recycleFailure("after_material_credit"); err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO reputation_accounts(player_id,balance) VALUES($1,0) ON CONFLICT(player_id) DO NOTHING`, intent.PlayerID); err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE reputation_accounts SET balance=balance+$2,revision=revision+1 WHERE player_id=$1`, intent.PlayerID, r.ReputationAwarded); err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO reputation_entries(operation_id,player_id,amount,created_at) VALUES($1,$2,$3,$4)`, intent.OperationID, intent.PlayerID, r.ReputationAwarded, now); err != nil {
		return recycle.Receipt{}, err
	}
	if err = s.recycleFailure("after_reputation_credit"); err != nil {
		return recycle.Receipt{}, err
	}
	materials, err := r.MaterialJSON()
	if err != nil {
		return recycle.Receipt{}, err
	}
	snapshot, err := json.Marshal(rule)
	if err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO recycle_receipts(operation_id,player_id,item_instance_id,item_template_id,item_revision,rule_id,rule_version,rule_snapshot,requested_at,materials_awarded,reputation_awarded,created_at)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, r.OperationID, r.PlayerID, r.ItemInstanceID, r.ItemTemplateID, r.ItemRevision, r.RuleID, r.RuleVersion, snapshot, r.RequestedAt, materials, r.ReputationAwarded, now); err != nil {
		return recycle.Receipt{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO recycle_audit_events(operation_id,kind,occurred_at) VALUES($1,'RECYCLE_COMMITTED',$2)`, r.OperationID, now); err != nil {
		return recycle.Receipt{}, err
	}
	if err = s.recycleFailure("after_receipt"); err != nil {
		return recycle.Receipt{}, err
	}
	if err = s.recycleFailure("before_commit"); err != nil {
		return recycle.Receipt{}, err
	}
	persisted, err := loadRecycle(ctx, tx, r.OperationID)
	if err != nil {
		return recycle.Receipt{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return recycle.Receipt{}, err
	}
	return persisted, nil
}
