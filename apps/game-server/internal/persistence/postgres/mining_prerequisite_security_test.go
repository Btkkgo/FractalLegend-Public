package postgres

import (
	"context"
	"testing"
)

func TestG21P0ImmutableEvidenceCannotBeUpdatedDeletedOrTruncated(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-immutable-reward", i.BlockInstanceID, p.PlayerID, 3); err != nil {
		t.Fatal(err)
	}
	for _, fact := range []struct{ table, key string }{
		{"mining_reservation_instance_bindings", "binding_id"}, {"mining_power_beneficiary_bindings", "binding_id"},
		{"mining_power_input_seals", "seal_id"}, {"mining_reward_issuance_lots", "issuance_id"},
		{"mining_reward_inventory_projections", "issuance_id"},
	} {
		for _, statement := range []string{"UPDATE " + fact.table + " SET " + fact.key + "=" + fact.key,
			"DELETE FROM " + fact.table, "TRUNCATE " + fact.table} {
			if _, err := s.pool.Exec(ctx, statement); err == nil {
				t.Fatalf("immutable evidence changed: %s", statement)
			}
		}
	}
	if _, err := s.RebuildMiningPowerSeal(ctx, i.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	report, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("evidence changed: %+v %v", report, err)
	}
}
