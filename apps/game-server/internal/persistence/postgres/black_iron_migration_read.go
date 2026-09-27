package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"fractallegend/game-server/internal/blackiron"
	"fractallegend/game-server/internal/persistence"
	"github.com/jackc/pgx/v5"
)

func loadBlackIronAliases(ctx context.Context, tx pgx.Tx) (map[string]blackiron.Alias, error) {
	rows, err := tx.Query(ctx, `SELECT definition_id,legacy_id,legacy_name,evidence FROM black_iron_identity_aliases ORDER BY definition_id`)
	if err != nil {
		return nil, classifyError(err)
	}
	defer rows.Close()
	result := map[string]blackiron.Alias{}
	for rows.Next() {
		var a blackiron.Alias
		if err = rows.Scan(&a.DefinitionID, &a.LegacyID, &a.LegacyName, &a.Evidence); err != nil {
			return nil, classifyError(err)
		}
		result[a.DefinitionID] = a
	}
	return result, rows.Err()
}
func loadBlackIronReceipt(ctx context.Context, tx pgx.Tx, id string) (blackiron.Receipt, error) {
	var r blackiron.Receipt
	var raw []byte
	var version, character, status, kind string
	var pre, post int64
	err := tx.QueryRow(ctx, `SELECT version,character_id,status,target_kind,pre_total,post_total,receipt,created_at,completed_at FROM black_iron_migration_receipts WHERE version=$1 AND character_id=$2`, blackiron.Version, id).Scan(&version, &character, &status, &kind, &pre, &post, &raw, &r.CreatedAt, &r.CompletedAt)
	if err != nil {
		return r, err
	}
	created, completed := r.CreatedAt, r.CompletedAt
	if json.Unmarshal(raw, &r) != nil || r.Version != version || r.CharacterID != character || r.Status != status || r.TargetKind != kind || r.PreTotal != pre || r.PostTotal != post || !r.CreatedAt.Equal(created) || !r.CompletedAt.Equal(completed) {
		return blackiron.Receipt{}, blackiron.ErrMismatch
	}
	return r, nil
}

// Historical receipt + current inventory are checked, never automatically fixed.
// G19 does not implement ore consumption or transfer lifecycle; those later
// features must extend the audit before legitimately changing migrated assets.
func checkBlackIronReceipt(ctx context.Context, tx pgx.Tx, r blackiron.Receipt, aliases map[string]blackiron.Alias) error {
	if r.Version != blackiron.Version || r.Status != "COMPLETE" || r.TargetKind != blackiron.Kind || r.CharacterID == "" || r.Changes == nil || r.RevisionBefore < 1 || r.RevisionAfter < r.RevisionBefore || r.CreatedAt.IsZero() || r.CompletedAt.Before(r.CreatedAt) {
		return blackiron.ErrMismatch
	}
	wantRevision := r.RevisionBefore
	if len(r.Changes) > 0 {
		if wantRevision == int64(^uint64(0)>>1) {
			return blackiron.ErrMismatch
		}
		wantRevision++
	}
	if r.RevisionAfter != wantRevision {
		return blackiron.ErrMismatch
	}
	seen := map[string]bool{}
	var pre, post int64
	for _, c := range r.Changes {
		alias, ok := aliases[c.Source.DefinitionID]
		if !ok || seen[c.Source.InstanceID] || c.Source.Name != alias.LegacyName {
			return blackiron.ErrMismatch
		}
		seen[c.Source.InstanceID] = true
		var commitmentVersion, commitmentOwner, commitmentDefinition string
		var commitmentLegacy int
		var commitmentQuantity int64
		e := tx.QueryRow(ctx, `SELECT version,character_id,definition_id,legacy_id,quantity FROM black_iron_migration_assets WHERE instance_id=$1`, c.Source.InstanceID).Scan(&commitmentVersion, &commitmentOwner, &commitmentDefinition, &commitmentLegacy, &commitmentQuantity)
		if e != nil || commitmentVersion != r.Version || commitmentOwner != r.CharacterID || commitmentDefinition != c.Target.DefinitionID || commitmentLegacy != c.Target.LegacyID || commitmentQuantity != c.Target.Quantity {
			return blackiron.ErrMismatch
		}

		want, e := blackiron.Normalize(c.Source, alias)
		if e != nil || want != c.Target {
			return blackiron.ErrMismatch
		}
		pre, e = blackiron.AddQuantity(pre, c.Source.Quantity)
		if e != nil {
			return blackiron.ErrMismatch
		}
		post, e = blackiron.AddQuantity(post, c.Target.Quantity)
		if e != nil {
			return blackiron.ErrMismatch
		}
		var a blackiron.Asset
		var owner string
		e = tx.QueryRow(ctx, `SELECT instance_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location,character_id FROM character_inventory_items WHERE instance_id=$1`, c.Target.InstanceID).Scan(&a.InstanceID, &a.DefinitionID, &a.LegacyID, &a.Name, &a.ItemType, &a.Quantity, &a.SlotIndex, &a.Location, &owner)
		a.SlotIndex = c.Target.SlotIndex // Ordinary inventory reordering is not asset conversion.
		if e != nil || a != c.Target || owner != r.CharacterID {
			return blackiron.ErrMismatch
		}
	}
	if pre != post || pre != r.PreTotal || post != r.PostTotal {
		return blackiron.ErrMismatch
	}
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT revision FROM characters WHERE id=$1`, r.CharacterID).Scan(&revision); err != nil {
		return classifyError(err)
	}
	if revision < r.RevisionAfter {
		return blackiron.ErrMismatch
	}
	// Also detect added mapped assets and legacy/canonical double representation.
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM character_inventory_items i JOIN black_iron_identity_aliases a ON i.definition_id=a.definition_id WHERE i.character_id=$1`, r.CharacterID).Scan(&count); err != nil {
		return classifyError(err)
	}
	if count != len(r.Changes) {
		return blackiron.ErrMismatch
	}
	return nil
}
func (s *Store) ReconcileBlackIronInventory(ctx context.Context) (blackiron.Report, error) {
	r := blackiron.Report{Balanced: true, Mismatches: []string{}}
	if s == nil || s.pool == nil {
		return r, persistence.ErrUnavailable
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return r, classifyError(err)
	}
	defer tx.Rollback(ctx)
	aliases, err := loadBlackIronAliases(ctx, tx)
	if err != nil {
		return r, err
	}
	rows, err := tx.Query(ctx, `SELECT character_id FROM black_iron_migration_receipts WHERE version=$1 ORDER BY character_id`, blackiron.Version)
	if err != nil {
		return r, classifyError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return r, classifyError(err)
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return r, classifyError(err)
	}
	for _, id := range ids {
		r.Checked++
		receipt, e := loadBlackIronReceipt(ctx, tx, id)
		if e == nil {
			e = checkBlackIronReceipt(ctx, tx, receipt, aliases)
		}
		if e != nil {
			r.Balanced = false
			r.Mismatches = append(r.Mismatches, fmt.Sprintf("%s: migration invariant mismatch", id))
		}
	}
	// Reverse coverage: canonical inventory requires immutable migration provenance.
	orphanRows, e := tx.Query(ctx, `SELECT i.instance_id FROM character_inventory_items i JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id LEFT JOIN black_iron_migration_assets m ON m.instance_id=i.instance_id LEFT JOIN black_iron_migration_receipts r ON r.version=m.version AND r.character_id=m.character_id WHERE i.name=$1 AND (m.instance_id IS NULL OR r.character_id IS NULL) ORDER BY i.instance_id`, blackiron.Name)
	if e != nil {
		return r, classifyError(e)
	}
	for orphanRows.Next() {
		var id string
		if e = orphanRows.Scan(&id); e != nil {
			orphanRows.Close()
			return r, classifyError(e)
		}
		r.Balanced = false
		r.Mismatches = append(r.Mismatches, fmt.Sprintf("%s: canonical asset has no migration provenance", id))
	}
	e = orphanRows.Err()
	orphanRows.Close()
	if e != nil {
		return r, classifyError(e)
	}
	return r, nil
}
