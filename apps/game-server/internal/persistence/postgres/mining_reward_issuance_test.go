package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/persistence"
)

func TestG21P0SyntheticIssuanceProjectionStaleSaveAndAudit(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	stale, err := s.LoadCharacter(ctx, principal.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.ValidateMiningActivity(ctx, principal, intent, miningpower.DevelopmentRuleVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	miningBlock, err := s.LoadMiningBlock(ctx, intent.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SealMiningPowerTEST(ctx, miningBlock.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	lot, err := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-reward-source", miningBlock.BlockInstanceID, stale.Character.ID, 3)
	if err != nil || lot.SourceType != "MINING_REWARD" || lot.MaterialDefinitionID != "synthetic-bun-material" || lot.Quantity != 3 {
		t.Fatalf("lot=%+v err=%v", lot, err)
	}
	if _, err = s.SaveCharacter(ctx, stale, stale.Character.Revision); !errors.Is(err, persistence.ErrStaleRevision) {
		t.Fatalf("stale save=%v", err)
	}
	current, err := s.LoadCharacter(ctx, stale.Character.ID)
	if err != nil || current.Character.Revision != stale.Character.Revision+1 || len(current.Items) != 1 || current.Items[0].Quantity != 3 {
		t.Fatalf("projection=%+v err=%v", current, err)
	}
	replay, err := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-reward-source", miningBlock.BlockInstanceID, stale.Character.ID, 3)
	if err != nil || replay != lot {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	report, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || !report.Balanced || report.Checked != 1 {
		t.Fatalf("audit=%+v err=%v", report, err)
	}
	g19Report, err := s.ReconcileBlackIronInventory(ctx)
	if err != nil || !g19Report.Balanced {
		t.Fatalf("G19 audit=%+v err=%v", g19Report, err)
	}
}

func TestG21P0RewardKeepsExistingG19MigrationReceiptAndQuantity(t *testing.T) {
	s, _, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g19Seed(t, s, "g21-p0-migrated-miner", 5)
	original, err := s.MigrateBlackIronInventory(ctx, "g21-p0-migrated-miner")
	if err != nil {
		t.Fatal(err)
	}
	session := g20Session(t, s, intent)
	session.ID = "g21-p0-migrated-session"
	session.PlayerID = "g21-p0-migrated-miner"
	session.AccountID = "g21-p0-migrated-miner-account"
	g20Insert(t, s, "mining_power_sessions", session)
	source := g20Source(t, s, intent)
	source.ID = "g21-p0-migrated-source"
	source.ActivityID = "mpa:" + source.ID
	source.PlayerID = session.PlayerID
	source.ActivitySessionID = session.ID
	miningIntent := g20RegisteredSource(t, s, source)
	g20Accept(t, s, miningpower.Principal{AccountID: session.AccountID, PlayerID: session.PlayerID}, miningIntent)
	if _, err = s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SealMiningPowerTEST(ctx, intent.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-migrated-reward", intent.BlockInstanceID, session.PlayerID, 3); err != nil {
		t.Fatal(err)
	}
	replayed, err := s.MigrateBlackIronInventory(ctx, session.PlayerID)
	if err != nil || !reflect.DeepEqual(original, replayed) {
		t.Fatalf("G19 receipt changed: before=%+v after=%+v %v", original, replayed, err)
	}
	items, err := s.LoadCharacter(ctx, session.PlayerID)
	if err != nil || len(items.Items) != 2 || items.Items[0].Quantity != 5 || items.Items[1].Quantity != 3 {
		t.Fatalf("migration quantity changed: %+v %v", items.Items, err)
	}
	g19Report, err := s.ReconcileBlackIronInventory(ctx)
	if err != nil || !g19Report.Balanced {
		t.Fatalf("G19 audit=%+v %v", g19Report, err)
	}
	rewardReport, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || !rewardReport.Balanced {
		t.Fatalf("reward audit=%+v %v", rewardReport, err)
	}
}

func TestG21P0IssuanceAuditDetectsSourceProjectionCorruption(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"missing-projection", `ALTER TABLE mining_reward_inventory_projections DISABLE TRIGGER mining_reward_inventory_projections_immutable;
			DELETE FROM mining_reward_inventory_projections; ALTER TABLE mining_reward_inventory_projections ENABLE TRIGGER mining_reward_inventory_projections_immutable`},
		{"wrong-quantity", `ALTER TABLE mining_reward_inventory_projections DISABLE TRIGGER mining_reward_inventory_projections_immutable;
			UPDATE mining_reward_inventory_projections SET quantity=4; ALTER TABLE mining_reward_inventory_projections ENABLE TRIGGER mining_reward_inventory_projections_immutable`},
		{"wrong-character", `ALTER TABLE mining_reward_inventory_projections DISABLE TRIGGER mining_reward_inventory_projections_immutable;
			UPDATE mining_reward_inventory_projections SET character_id='other-character'; ALTER TABLE mining_reward_inventory_projections ENABLE TRIGGER mining_reward_inventory_projections_immutable`},
		{"wrong-instance", `ALTER TABLE mining_reward_inventory_projections DISABLE TRIGGER mining_reward_inventory_projections_immutable;
			UPDATE mining_reward_inventory_projections SET block_instance_id='mining-block-instance-ffffffffffffffffffffffffffffffff';
			ALTER TABLE mining_reward_inventory_projections ENABLE TRIGGER mining_reward_inventory_projections_immutable`},
		{"stale-revision", `ALTER TABLE mining_reward_inventory_projections DISABLE TRIGGER mining_reward_inventory_projections_immutable;
			UPDATE mining_reward_inventory_projections SET revision_before=revision_before+1,revision_after=revision_after+1;
			ALTER TABLE mining_reward_inventory_projections ENABLE TRIGGER mining_reward_inventory_projections_immutable`},
		{"source-digest", `ALTER TABLE mining_reward_issuance_lots DISABLE TRIGGER mining_reward_issuance_lots_immutable;
			UPDATE mining_reward_issuance_lots SET source_digest=repeat('0',64);
			ALTER TABLE mining_reward_issuance_lots ENABLE TRIGGER mining_reward_issuance_lots_immutable`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, principal, intent)
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID); err != nil {
				t.Fatal(err)
			}
			if _, err := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-corruption-source", intent.BlockInstanceID, principal.PlayerID, 3); err != nil {
				t.Fatal(err)
			}
			if _, err := s.pool.Exec(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			report, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
			if err != nil || report.Balanced || len(report.Mismatches) == 0 {
				t.Fatalf("corruption undetected: %+v %v", report, err)
			}
			if _, err = s.createMiningRewardIssuancePrerequisiteTEST(ctx, "g21-p0-corruption-source", intent.BlockInstanceID, principal.PlayerID, 3); err == nil {
				t.Fatal("corrupted issuance replay succeeded")
			}
		})
	}
}
