package postgres

import (
	"context"
	"testing"
)

func TestG21ClosureReplayPreservesHistoryAndAuditRejectsItemCorruption(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"name", `UPDATE character_inventory_items SET name='corrupt'`},
		{"legacy", `UPDATE character_inventory_items SET legacy_id=legacy_id+1`},
		{"type", `UPDATE character_inventory_items SET item_type='CONSUMABLE'`},
		{"location", `UPDATE character_inventory_items SET location='EQUIPMENT',equipment_slot='WEAPON'`},
		{"owner-account", `UPDATE characters SET account_id='other-account' WHERE id='A'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
			ctx := context.Background()
			first, e := s.SettleMiningRewardTEST(ctx, cmd)
			if e != nil {
				t.Fatal(e)
			}
			g19Seed(t, s, "other")
			tx, e := s.pool.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(ctx)
			if _, e = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec(ctx, tc.sql); e != nil {
				t.Fatal(e)
			}
			if e = tx.Commit(ctx); e != nil {
				t.Fatal(e)
			}
			before := g21ClosureSnapshot(t, s)
			g21T044Exact(t, s, cmd, first)
			audit, e := s.ReconcileBlackIronEmission(ctx)
			if e == nil && audit.Balanced {
				t.Fatalf("independent audit accepted corrupt current inventory: %+v", audit)
			}
			if g21ClosureSnapshot(t, s) != before {
				t.Fatal("replay repaired corruption or changed economics")
			}
		})
	}
}
