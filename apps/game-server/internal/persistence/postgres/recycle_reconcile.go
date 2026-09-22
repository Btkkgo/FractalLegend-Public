package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"fractallegend/game-server/internal/recycle"
)

// ReconcileRecycle is read-only. It reports broken links; it never repairs history.
func (s *Store) ReconcileRecycle(ctx context.Context) (recycle.ReconciliationReport, error) {
	r := recycle.ReconciliationReport{Balanced: true}
	if s == nil || s.pool == nil {
		return r, recycle.ErrReconciliation
	}
	rows, err := s.pool.Query(ctx, `SELECT operation_id,rule_snapshot FROM recycle_receipts ORDER BY operation_id`)
	if err != nil {
		return r, err
	}
	type snapshot struct {
		id   string
		data []byte
	}
	var snapshots []snapshot
	for rows.Next() {
		var x snapshot
		if err = rows.Scan(&x.id, &x.data); err != nil {
			rows.Close()
			return r, err
		}
		snapshots = append(snapshots, x)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return r, err
	}
	rows.Close()
	problem := func(format string, args ...any) {
		r.Problems = append(r.Problems, fmt.Sprintf(format, args...))
		r.Balanced = false
	}
	for _, x := range snapshots {
		receipt, loadErr := s.LoadRecycle(ctx, x.id)
		if loadErr != nil {
			return r, loadErr
		}
		r.Checked++
		if receipt.FBAwarded != 0 || receipt.ContributionAwarded != 0 || receipt.BlackIronAwarded != 0 {
			problem("%s: forbidden award", x.id)
		}
		var rule recycle.Rule
		if json.Unmarshal(x.data, &rule) != nil || rule.ID != receipt.RuleID || rule.Version != receipt.RuleVersion ||
			rule.TemplateID != receipt.ItemTemplateID || rule.ReputationReward != receipt.ReputationAwarded ||
			!sameMaterials(rule.MaterialOutputs, receipt.MaterialsAwarded) {
			problem("%s: rule snapshot mismatch", x.id)
		}
		if _, validationErr := recycle.NewRegistry([]recycle.Rule{rule}); validationErr != nil {
			problem("%s: invalid rule snapshot", x.id)
		}
		var consumed, itemAbsent, reputationOK, auditOK bool
		err = s.pool.QueryRow(ctx, `SELECT
            EXISTS(SELECT 1 FROM item_instance_lifecycle WHERE instance_id=$1 AND consumed AND revision=$2),
            NOT EXISTS(SELECT 1 FROM character_inventory_items WHERE instance_id=$1),
            EXISTS(SELECT 1 FROM reputation_entries WHERE operation_id=$3 AND player_id=$4 AND amount=$5),
            EXISTS(SELECT 1 FROM recycle_audit_events WHERE operation_id=$3 AND kind='RECYCLE_COMMITTED')`,
			receipt.ItemInstanceID, receipt.ItemRevision, receipt.OperationID, receipt.PlayerID, receipt.ReputationAwarded).
			Scan(&consumed, &itemAbsent, &reputationOK, &auditOK)
		if err != nil {
			return r, err
		}
		if !consumed || !itemAbsent {
			problem("%s: consumed item mismatch", x.id)
		}
		if !reputationOK {
			problem("%s: reputation entry mismatch", x.id)
		}
		if !auditOK {
			problem("%s: audit event missing", x.id)
		}
		credits, creditErr := s.pool.Query(ctx, `SELECT material_id,quantity,inventory_instance_id FROM recycle_material_credits WHERE operation_id=$1`, x.id)
		if creditErr != nil {
			return r, creditErr
		}
		awarded := make(map[string]int64, len(receipt.MaterialsAwarded))
		for _, m := range receipt.MaterialsAwarded {
			if m.Quantity > 0 {
				awarded[m.ID] = m.Quantity
			}
		}
		for credits.Next() {
			var materialID, instanceID string
			var quantity int64
			if err = credits.Scan(&materialID, &quantity, &instanceID); err != nil {
				credits.Close()
				return r, err
			}
			if awarded[materialID] != quantity {
				problem("%s: material credit mismatch", x.id)
			}
			delete(awarded, materialID)
			var inventoryOK bool
			if err = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM character_inventory_items WHERE instance_id=$1 AND character_id=$2 AND definition_id=$3 AND item_type='MATERIAL' AND quantity=$4)`, instanceID, receipt.PlayerID, materialID, quantity).Scan(&inventoryOK); err != nil {
				credits.Close()
				return r, err
			}
			if !inventoryOK {
				problem("%s: material inventory mismatch", x.id)
			}
		}
		if err = credits.Err(); err != nil {
			credits.Close()
			return r, err
		}
		credits.Close()
		if len(awarded) != 0 {
			problem("%s: material credit missing", x.id)
		}
	}
	var orphanConsumed, orphanReputation, orphanMaterial, wrongBalance int
	for _, check := range []struct {
		query string
		dest  *int
	}{
		{`SELECT count(*) FROM item_instance_lifecycle l WHERE l.consumed AND NOT EXISTS(SELECT 1 FROM recycle_receipts r WHERE r.item_instance_id=l.instance_id)`, &orphanConsumed},
		{`SELECT count(*) FROM reputation_entries e WHERE NOT EXISTS(SELECT 1 FROM recycle_receipts r WHERE r.operation_id=e.operation_id AND r.player_id=e.player_id AND r.reputation_awarded=e.amount)`, &orphanReputation},
		{`SELECT count(*) FROM recycle_material_credits c WHERE NOT EXISTS(SELECT 1 FROM recycle_receipts r WHERE r.operation_id=c.operation_id)`, &orphanMaterial},
		{`SELECT count(*) FROM reputation_accounts a WHERE a.balance <> COALESCE((SELECT sum(e.amount) FROM reputation_entries e WHERE e.player_id=a.player_id),0)`, &wrongBalance},
	} {
		if err = s.pool.QueryRow(ctx, check.query).Scan(check.dest); err != nil {
			return r, err
		}
	}
	if orphanConsumed > 0 {
		problem("%d consumed items have no receipt", orphanConsumed)
	}
	if orphanReputation > 0 {
		problem("%d reputation entries lack a matching receipt", orphanReputation)
	}
	if orphanMaterial > 0 {
		problem("%d material credits lack a receipt", orphanMaterial)
	}
	if wrongBalance > 0 {
		problem("%d reputation account balances are inconsistent", wrongBalance)
	}
	return r, nil
}

func sameMaterials(a, b []recycle.Material) bool {
	if len(a) != len(b) {
		return false
	}
	left := map[string]int64{}
	for _, m := range a {
		left[m.ID] = m.Quantity
	}
	for _, m := range b {
		if left[m.ID] != m.Quantity {
			return false
		}
		delete(left, m.ID)
	}
	return len(left) == 0
}
