package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/systemspend"
)

func TestG21AtomicSettlementPaysOnceAndReplaysExactReceipt(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	command := MiningRewardSettlementCommand{CommandID: "g21-atomic-one", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != "COMPLETED" || first.TotalOre != 10 || len(first.Grants) != 1 || first.Grants[0].CharacterID != principal.PlayerID || first.Grants[0].Quantity != 10 {
		t.Fatalf("settlement=%+v", first)
	}
	var reserved, distributed, remaining, debt int64
	if err := s.pool.QueryRow(ctx, `SELECT total_reserved,total_distributed,remaining_capacity,recovery_debt
		FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&reserved, &distributed, &remaining, &debt); err != nil {
		t.Fatal(err)
	}
	if reserved != 0 || distributed != 10 || remaining != 10 || debt != 0 {
		t.Fatalf("pool S/D/M/H=%d/%d/%d/%d", reserved, distributed, remaining, debt)
	}
	var grants, lots, stacks, consumed, receipts int
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_grants),
		(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
		(SELECT count(*) FROM mining_reward_inventory_projections),
		(SELECT count(*) FROM mining_reward_reservation_consumptions),
		(SELECT count(*) FROM mining_reward_settlement_receipts)`).Scan(&grants, &lots, &stacks, &consumed, &receipts); err != nil {
		t.Fatal(err)
	}
	if grants != 1 || lots != 1 || stacks != 1 || consumed != 1 || receipts != 1 {
		t.Fatalf("artifacts grant/lot/stack/consume/receipt=%d/%d/%d/%d/%d", grants, lots, stacks, consumed, receipts)
	}
	emissionAudit, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !emissionAudit.Balanced {
		t.Fatalf("settled emission audit=%+v %v", emissionAudit, err)
	}
	rewardAudit, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || !rewardAudit.Balanced {
		t.Fatalf("settled reward audit=%+v %v", rewardAudit, err)
	}
	replayed, err := s.SettleMiningRewardTEST(ctx, command)
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("replay=%+v first=%+v err=%v", replayed, first, err)
	}
}

func TestG21TwoBlocksSharedOwnerKeepIndependentLotsAndStacks(t *testing.T) {
	s, principal, firstIntent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	secondBirth, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Create(ctx, "g21-second-block")
	if err != nil {
		t.Fatal(err)
	}
	secondBlock, err := s.LoadMiningBlock(ctx, secondBirth.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	firstSession := g20Session(t, s, firstIntent)
	secondSession := firstSession
	secondSession.ID = "g21-second-session"
	secondSession.BlockID, secondSession.BlockInstanceID = secondBlock.ID, secondBlock.BlockInstanceID
	secondSession.OpenedAt, secondSession.ExpiresAt = secondBlock.StartedAt, secondBlock.ScheduledEndAt
	g20Insert(t, s, "mining_power_sessions", secondSession)
	secondSource := g20Source(t, s, firstIntent)
	secondSource.ID, secondSource.ActivityID = "g21-second-event", "mpa:g21-second-event"
	secondSource.BlockID, secondSource.BlockInstanceID = secondBlock.ID, secondBlock.BlockInstanceID
	secondSource.ActivitySessionID, secondSource.ExpiresAt = secondSession.ID, secondBlock.ScheduledEndAt
	secondSource.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
	g20Insert(t, s, "mining_power_source_events", secondSource)
	secondIntent := miningpower.ActionIntent{ActivityID: secondSource.ActivityID, SourceEventID: secondSource.ID,
		ActivitySessionID: secondSession.ID, BlockID: secondBlock.ID, BlockInstanceID: secondBlock.BlockInstanceID}
	var receipts []MiningRewardSettlementReceipt
	for i, intent := range []miningpower.ActionIntent{firstIntent, secondIntent} {
		g20Accept(t, s, principal, intent)
		if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
			t.Fatal(err)
		}
		seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
		if err != nil {
			t.Fatal(err)
		}
		cmd := MiningRewardSettlementCommand{
			CommandID: fmt.Sprintf("g21-shared-owner-%d", i), BlockInstanceID: intent.BlockInstanceID,
			ExpectedSealDigest: seal.CanonicalDigest}
		if i == 1 {
			spend, err := s.LoadSystemSpend(ctx, "g18-fund")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-shared-owner-refund", 15)); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardDebt) {
				t.Fatalf("second first attempt during recovery debt=%v", err)
			}
			var commands, lots int
			if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_commands),
				(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED')`).Scan(&commands, &lots); err != nil {
				t.Fatal(err)
			}
			if commands != 1 || lots != 1 {
				t.Fatalf("debt leaked second command/lot=%d/%d", commands, lots)
			}
			newSpend, err := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g21-shared-owner-repay", 15))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, newSpend.ID); err != nil {
				t.Fatal(err)
			}
			var debt int64
			if err := s.pool.QueryRow(ctx, `SELECT recovery_debt FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&debt); err != nil || debt != 0 {
				t.Fatalf("repayment debt=%d err=%v", debt, err)
			}
		}
		receipt, err := s.SettleMiningRewardTEST(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		receipts = append(receipts, receipt)
	}
	if receipts[0].SettlementID == receipts[1].SettlementID ||
		receipts[0].Grants[0].IssuanceID == receipts[1].Grants[0].IssuanceID ||
		receipts[0].Grants[0].InventoryInstanceID == receipts[1].Grants[0].InventoryInstanceID {
		t.Fatalf("shared owner reused identities: %+v %+v", receipts[0], receipts[1])
	}
	var lots, stacks, grants, consumed int
	var ore, revision int64
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
		(SELECT count(*) FROM mining_reward_inventory_projections),
		(SELECT count(*) FROM mining_reward_grants),
		(SELECT count(*) FROM mining_reward_reservation_consumptions),
		(SELECT COALESCE(sum(quantity),0) FROM character_inventory_items WHERE character_id=$1),
		(SELECT revision FROM characters WHERE id=$1)`, principal.PlayerID).
		Scan(&lots, &stacks, &grants, &consumed, &ore, &revision); err != nil {
		t.Fatal(err)
	}
	if lots != 2 || stacks != 2 || grants != 2 || consumed != 2 || ore != 20 || revision != 3 {
		t.Fatalf("two block artifacts lot/stack/grant/consume/ore/rev=%d/%d/%d/%d/%d/%d",
			lots, stacks, grants, consumed, ore, revision)
	}
}

func TestG21ReplayRejectsMissingGrantHistory(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-corrupt-grant", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_reward_grants DISABLE TRIGGER mining_reward_grants_immutable;
		DELETE FROM mining_reward_grants;
		ALTER TABLE mining_reward_grants ENABLE TRIGGER mining_reward_grants_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardInvariant) {
		t.Fatalf("corrupt grant replay=%v", err)
	}
}

func TestG21ReplayRejectsReceiptDigestAndIndexedScalarCorruption(t *testing.T) {
	for _, corruption := range []struct{ name, statement string }{
		{"digest", `UPDATE mining_reward_settlement_receipts SET canonical_digest=repeat('0',64)`},
		{"indexed quantity", `UPDATE mining_reward_settlement_receipts SET total_ore=total_ore+1`},
		{"indexed seal", `UPDATE mining_reward_settlement_receipts SET seal_id='forged-seal'`},
	} {
		t.Run(corruption.name, func(t *testing.T) {
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, principal, intent)
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if err != nil {
				t.Fatal(err)
			}
			cmd := MiningRewardSettlementCommand{CommandID: "g21-receipt-corrupt", BlockInstanceID: intent.BlockInstanceID,
				ExpectedSealDigest: seal.CanonicalDigest}
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_reward_settlement_receipts DISABLE TRIGGER mining_reward_receipts_immutable`); err != nil {
				t.Fatal(err)
			}
			if _, err := s.pool.Exec(ctx, corruption.statement); err != nil {
				t.Fatal(err)
			}
			if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_reward_settlement_receipts ENABLE TRIGGER mining_reward_receipts_immutable`); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardInvariant) {
				t.Fatalf("corrupt receipt replay accepted: %v", err)
			}
		})
	}
}

func TestG21SettlementRejectsCorruptBeneficiaryBindingDigest(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_power_beneficiary_bindings DISABLE TRIGGER mining_power_beneficiary_bindings_immutable;
		UPDATE mining_power_beneficiary_bindings SET digest=repeat('0',64);
		ALTER TABLE mining_power_beneficiary_bindings ENABLE TRIGGER mining_power_beneficiary_bindings_immutable`); err != nil {
		t.Fatal(err)
	}
	_, err = s.SettleMiningRewardTEST(ctx, MiningRewardSettlementCommand{
		CommandID: "g21-corrupt-beneficiary", BlockInstanceID: intent.BlockInstanceID,
		ExpectedSealDigest: seal.CanonicalDigest})
	if !errors.Is(err, ErrMiningRewardInvariant) {
		t.Fatalf("corrupt beneficiary was accepted: %v", err)
	}
	var grants, lots int
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_grants),
		(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED')`).Scan(&grants, &lots); err != nil {
		t.Fatal(err)
	}
	if grants != 0 || lots != 0 {
		t.Fatalf("corrupt beneficiary created grants/lots=%d/%d", grants, lots)
	}
}

func TestG21SettlementHistoryRejectsOrdinaryMutation(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO mining_reward_settlement_states(block_instance_id,seal_id,state,revision)
		VALUES($1,$2,'SEALED',1)`, intent.BlockInstanceID, seal.SealID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE mining_reward_settlement_states
		SET state='COMPLETED',revision=2,command_id='forged-command',
		settlement_id='forged-settlement',completed_at=now()
		WHERE block_instance_id=$1`, intent.BlockInstanceID); err == nil {
		t.Fatal("state skipped to completion without command, receipt, and consumption")
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-immutable", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct{ table, update string }{
		{"mining_reward_commands", "created_at"},
		{"mining_reward_settlement_receipts", "created_at"},
		{"mining_reward_grants", "created_at"},
		{"mining_reward_reservation_consumptions", "created_at"},
		{"mining_reward_issuance_lots", "created_at"},
		{"mining_reward_inventory_projections", "created_at"},
		{"black_iron_emission_recovery_entries", "created_at"},
		{"mining_power_input_seals", "sealed_at"},
		{"mining_power_beneficiary_bindings", "bound_at"},
		{"black_iron_identity_aliases", "definition_id"},
	} {
		t.Run(fixture.table, func(t *testing.T) {
			for _, sql := range []string{
				"UPDATE " + fixture.table + " SET " + fixture.update + "=" + fixture.update,
				"DELETE FROM " + fixture.table,
				"TRUNCATE " + fixture.table,
			} {
				if _, err := s.pool.Exec(ctx, sql); err == nil {
					t.Fatalf("immutable history accepted %s", sql)
				}
			}
		})
	}
	for _, sql := range []string{
		"UPDATE mining_reward_settlement_states SET state='SEALED',revision=1",
		"DELETE FROM mining_reward_settlement_states",
		"TRUNCATE mining_reward_settlement_states",
	} {
		if _, err := s.pool.Exec(ctx, sql); err == nil {
			t.Fatalf("state accepted illegal mutation %s", sql)
		}
	}
	replay, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("history changed after rejection: %+v %v", replay, err)
	}
}

func TestG21IndependentDuplicateEconomicOriginsRejected(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	command := MiningRewardSettlementCommand{CommandID: "g21-independent-duplicates", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, command)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, sql string }{
		{"issuance-lot", `INSERT INTO mining_reward_issuance_lots
			(issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,quantity,
			material_definition_id,rule_version,created_at,source_digest,status)
			SELECT 'g21-duplicate-lot',source_type,'g21-duplicate-source',settlement_id,block_instance_id,
			character_id,quantity,material_definition_id,rule_version,created_at,source_digest,status
			FROM mining_reward_issuance_lots WHERE status='G21_SETTLED' LIMIT 1`},
		{"inventory-projection", `INSERT INTO mining_reward_inventory_projections
			(issuance_id,instance_id,character_id,block_instance_id,quantity,definition_id,
			revision_before,revision_after,created_at)
			SELECT issuance_id,'g21-duplicate-projection',character_id,block_instance_id,quantity,
			definition_id,revision_before,revision_after,created_at
			FROM mining_reward_inventory_projections LIMIT 1`},
		{"reservation-consumption", `INSERT INTO mining_reward_reservation_consumptions
			(consumption_id,settlement_id,block_instance_id,reservation_source_id,amount,pool_revision,created_at)
			SELECT 'g21-duplicate-consumption','g21-duplicate-settlement',block_instance_id,
			reservation_source_id,amount,pool_revision+1,created_at
			FROM mining_reward_reservation_consumptions LIMIT 1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.pool.Exec(ctx, tc.sql); err == nil {
				t.Fatal("duplicate economic origin was accepted")
			}
			var lots, projections, consumptions int
			if err := s.pool.QueryRow(ctx, `SELECT
				(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
				(SELECT count(*) FROM mining_reward_inventory_projections),
				(SELECT count(*) FROM mining_reward_reservation_consumptions)`).
				Scan(&lots, &projections, &consumptions); err != nil {
				t.Fatal(err)
			}
			if lots != 1 || projections != 1 || consumptions != 1 {
				t.Fatalf("rejected duplicate changed lots/projections/consumptions=%d/%d/%d", lots, projections, consumptions)
			}
			replay, err := s.SettleMiningRewardTEST(ctx, command)
			if err != nil || !reflect.DeepEqual(first, replay) {
				t.Fatalf("duplicate attempt changed receipt: %+v %v", replay, err)
			}
		})
	}
}

func TestG21IndependentItemRevisionAndCorruptReplay(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	command := MiningRewardSettlementCommand{CommandID: "g21-item-revision", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, command)
	if err != nil || len(first.Grants) != 1 || first.Grants[0].ItemRevisionBefore != 0 || first.Grants[0].ItemRevisionAfter != 1 {
		t.Fatalf("item revision receipt=%+v %v", first, err)
	}
	var persisted int64
	if err := s.pool.QueryRow(ctx, `SELECT item_revision FROM character_inventory_items WHERE instance_id=$1`,
		first.Grants[0].InventoryInstanceID).Scan(&persisted); err != nil || persisted != 1 {
		t.Fatalf("persisted item revision=%d %v", persisted, err)
	}
	// Test-only corruption bypasses the inventory guard on this one transaction.
	// Historical replay preserves the receipt; the independent audit detects the current version.
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE character_inventory_items SET item_revision=2 WHERE instance_id=$1`,
		first.Grants[0].InventoryInstanceID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	g21T044Exact(t, s, command, first)
	audit, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
	if err != nil || audit.Balanced || len(audit.Mismatches) == 0 {
		t.Fatalf("corrupt item revision audit=%+v %v", audit, err)
	}
	g21AuthorityUnchanged(t, s, before)
}

func TestG21FullSettlement500And501Boundary(t *testing.T) {
	for _, count := range []int{2, 10, 50, 100, 500, 501} {
		t.Run(fmt.Sprintf("beneficiaries-%d", count), func(t *testing.T) {
			started := time.Now()
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, principal, intent)
			baseSession := g20Session(t, s, intent)
			baseSource := g20Source(t, s, intent)
			for i := 1; i < count; i++ {
				player := fmt.Sprintf("g21-full-player-%04d", i)
				g19Seed(t, s, player)
				session := baseSession
				session.ID = fmt.Sprintf("g21-full-session-%04d", i)
				session.PlayerID = player
				session.AccountID = player + "-account"
				g20Insert(t, s, "mining_power_sessions", session)
				source := baseSource
				source.ID = fmt.Sprintf("g21-full-event-%04d", i)
				source.ActivityID = "mpa:" + source.ID
				source.PlayerID = player
				source.ActivitySessionID = session.ID
				g20Insert(t, s, "mining_power_source_events", source)
				g20Accept(t, s, miningpower.Principal{PlayerID: player, AccountID: session.AccountID},
					miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID,
						ActivitySessionID: session.ID, BlockID: intent.BlockID, BlockInstanceID: intent.BlockInstanceID})
			}
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			// Compare every persisted economic identity and every owner revision,
			// rather than relying on a few aggregate counters after rejection.
			const economicSnapshot = `SELECT jsonb_build_object(
				'pool',(SELECT to_jsonb(p) FROM black_iron_emission_pools p WHERE pool_id='GLOBAL'),
				'reservation',(SELECT to_jsonb(r) FROM mining_block_reservations r WHERE block_id=$1),
				'binding',(SELECT to_jsonb(b) FROM mining_reservation_instance_bindings b WHERE block_instance_id=$2),
				'owners',(SELECT jsonb_object_agg(id,revision) FROM characters),
				'items',(SELECT coalesce(jsonb_agg(to_jsonb(i) ORDER BY instance_id),'[]'::jsonb) FROM character_inventory_items i),
				'consumptions',(SELECT count(*) FROM mining_reward_reservation_consumptions),
				'grants',(SELECT count(*) FROM mining_reward_grants),
				'lots',(SELECT count(*) FROM mining_reward_issuance_lots),
				'stacks',(SELECT count(*) FROM mining_reward_inventory_projections),
				'receipts',(SELECT count(*) FROM mining_reward_settlement_receipts),
				'commands',(SELECT count(*) FROM mining_reward_commands),
				'completed',(SELECT count(*) FROM mining_reward_settlement_states WHERE state='COMPLETED'))::text`
			var beforeEconomic string
			if count == 501 {
				if err := s.pool.QueryRow(ctx, economicSnapshot, intent.BlockID, intent.BlockInstanceID).Scan(&beforeEconomic); err != nil {
					t.Fatal(err)
				}
			}
			seal, sealErr := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if count == 501 {
				if sealErr == nil {
					t.Fatal("501 beneficiary Seal succeeded")
				}
				var reserved, distributed, remaining, debt, revision int64
				var grants, lots, stacks, receipts, completed int
				if err := s.pool.QueryRow(ctx, `SELECT
					(SELECT total_reserved FROM black_iron_emission_pools),
					(SELECT total_distributed FROM black_iron_emission_pools),
					(SELECT remaining_capacity FROM black_iron_emission_pools),
					(SELECT recovery_debt FROM black_iron_emission_pools),
					(SELECT revision FROM black_iron_emission_pools),
					(SELECT count(*) FROM mining_reward_grants),
					(SELECT count(*) FROM mining_reward_issuance_lots),
					(SELECT count(*) FROM mining_reward_inventory_projections),
					(SELECT count(*) FROM mining_reward_settlement_receipts),
					(SELECT count(*) FROM mining_reward_settlement_states WHERE state='COMPLETED')`).
					Scan(&reserved, &distributed, &remaining, &debt, &revision, &grants, &lots, &stacks, &receipts, &completed); err != nil {
					t.Fatal(err)
				}
				if reserved != 10 || distributed != 0 || remaining != 10 || debt != 0 || revision != 2 ||
					grants != 0 || lots != 0 || stacks != 0 || receipts != 0 || completed != 0 {
					t.Fatalf("501 pool S/D/M/H/rev=%d/%d/%d/%d/%d artifacts=%d/%d/%d/%d/%d",
						reserved, distributed, remaining, debt, revision, grants, lots, stacks, receipts, completed)
				}
				var afterEconomic string
				if err := s.pool.QueryRow(ctx, economicSnapshot, intent.BlockID, intent.BlockInstanceID).Scan(&afterEconomic); err != nil {
					t.Fatal(err)
				}
				if afterEconomic != beforeEconomic {
					t.Fatalf("501 rejection changed pool, reservation, consumption, issuance, inventory, receipt, command, state, or owner revisions:\nbefore=%s\nafter=%s", beforeEconomic, afterEconomic)
				}
				t.Logf("501 rejected before settlement in %s", time.Since(started))
				return
			}
			if sealErr != nil || seal.ParticipantCount != count {
				t.Fatalf("500 seal=%+v %v", seal, sealErr)
			}
			receipt, err := s.SettleMiningRewardTEST(ctx, MiningRewardSettlementCommand{
				CommandID: "g21-full-500", BlockInstanceID: intent.BlockInstanceID,
				ExpectedSealDigest: seal.CanonicalDigest})
			if err != nil || receipt.Status != "COMPLETED" || receipt.TotalOre != 10 || len(receipt.Grants) != count {
				t.Fatalf("500 settlement status=%s total=%d grants=%d err=%v", receipt.Status, receipt.TotalOre, len(receipt.Grants), err)
			}
			var sum int64
			var positive int
			for i, g := range receipt.Grants {
				if i > 0 && receipt.Grants[i-1].CharacterID >= g.CharacterID {
					t.Fatal("500 grants lost CharacterID order")
				}
				sum += g.Quantity
				if g.Quantity > 0 {
					positive++
					if g.InventoryRevisionAfter != g.InventoryRevisionBefore+1 || g.ItemRevisionBefore != 0 || g.ItemRevisionAfter != 1 {
						t.Fatalf("positive grant revision=%+v", g)
					}
				} else if g.IssuanceID != "" || g.InventoryInstanceID != "" ||
					g.InventoryRevisionAfter != g.InventoryRevisionBefore || g.ItemRevisionBefore != 0 || g.ItemRevisionAfter != 0 {
					t.Fatalf("zero grant changed inventory=%+v", g)
				}
			}
			if sum != 10 || positive != min(count, 10) {
				t.Fatalf("500 conservation sum=%d positive=%d", sum, positive)
			}
			var grants, lots, stacks, consumed int
			if err := s.pool.QueryRow(ctx, `SELECT
				(SELECT count(*) FROM mining_reward_grants),
				(SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
				(SELECT count(*) FROM mining_reward_inventory_projections),
				(SELECT count(*) FROM mining_reward_reservation_consumptions)`).
				Scan(&grants, &lots, &stacks, &consumed); err != nil {
				t.Fatal(err)
			}
			if grants != count || lots != min(count, 10) || stacks != min(count, 10) || consumed != 1 {
				t.Fatalf("500 artifacts=%d/%d/%d/%d", grants, lots, stacks, consumed)
			}
			var receipts, completed, accepted, zeroArtifacts int
			if err := s.pool.QueryRow(ctx, `SELECT
              (SELECT count(*) FROM mining_reward_settlement_receipts),
              (SELECT count(*) FROM mining_reward_settlement_states WHERE state='COMPLETED'),
              (SELECT count(*) FROM mining_power_activities),
              (SELECT count(*) FROM mining_reward_grants g JOIN characters c ON c.id=g.character_id
                WHERE g.quantity=0 AND (c.revision<>1 OR EXISTS(SELECT 1 FROM character_inventory_items i WHERE i.character_id=c.id)))`).Scan(&receipts, &completed, &accepted, &zeroArtifacts); err != nil {
				t.Fatal(err)
			}
			if receipts != 1 || completed != 1 || accepted != count || zeroArtifacts != 0 {
				t.Fatalf("receipt/completed/accepted/zero artifacts=%d/%d/%d/%d", receipts, completed, accepted, zeroArtifacts)
			}
			beforeReplay := g21ClosureSnapshot(t, s)
			replay, err := s.SettleMiningRewardTEST(ctx, MiningRewardSettlementCommand{CommandID: receipt.CommandID, BlockInstanceID: receipt.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest})
			if err != nil || !reflect.DeepEqual(receipt, replay) || g21ClosureSnapshot(t, s) != beforeReplay {
				t.Fatalf("scale replay changed state or receipt: %v", err)
			}
			r, e := s.ReconcileBlackIronEmission(ctx)
			if e != nil || !r.Balanced {
				t.Fatalf("emission conservation %+v %v", r, e)
			}
			ir, e := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
			if e != nil || !ir.Balanced {
				t.Fatalf("inventory conservation %+v %v", ir, e)
			}
			raw, err := canonicalMiningRewardReceipt(receipt)
			if err != nil || len(raw) > settlementMaxReceiptBytes {
				t.Fatal("receipt payload", err)
			}
			t.Logf("N=%d R=10 grants=%d positive=%d zero=%d lots=%d stacks=%d sum=%d receipt=1 completed=1 consumption=1 replay=ZERO bytes=%d duration=%s", count, grants, positive, count-positive, lots, stacks, sum, len(raw), time.Since(started))
		})
	}
}

func TestG21PostSettlementRefundKeepsOreAndReplayDespiteDebt(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-refund-completed", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	spend, err := s.LoadSystemSpend(ctx, "g18-fund")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-refund-after-pay", 15)); err != nil {
		t.Fatal(err)
	}
	var capacity, reserved, distributed, remaining, debt, ore int64
	if err := s.pool.QueryRow(ctx, `SELECT total_emission_capacity,total_reserved,total_distributed,
		remaining_capacity,recovery_debt FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).
		Scan(&capacity, &reserved, &distributed, &remaining, &debt); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT sum(quantity) FROM character_inventory_items WHERE character_id=$1`, principal.PlayerID).Scan(&ore); err != nil {
		t.Fatal(err)
	}
	if capacity != 5 || reserved != 0 || distributed != 10 || remaining != 0 || debt != 5 || ore != 10 {
		t.Fatalf("refund C/S/D/M/H/Ore=%d/%d/%d/%d/%d/%d", capacity, reserved, distributed, remaining, debt, ore)
	}
	replay, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("refund replay=%+v first=%+v %v", replay, first, err)
	}
	report, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("refund recovery=%+v %v", report, err)
	}
	newSpend, err := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g21-capacity-repay", 7))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, newSpend.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT remaining_capacity,recovery_debt FROM black_iron_emission_pools
		WHERE pool_id='GLOBAL'`).Scan(&remaining, &debt); err != nil {
		t.Fatal(err)
	}
	if remaining != 2 || debt != 0 {
		t.Fatalf("repayment remaining/debt=%d/%d", remaining, debt)
	}
	report, err = s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("repayment recovery=%+v %v", report, err)
	}
}

func TestG21RecoveryDebtExactRemainingRefund(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-exact-refund-remaining", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	// The existing G18 block keeps its authoritative R=10. Additional G15
	// capacity establishes the precise C100/S0/D10/M90/H0 test baseline.
	newSpend, err := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g21-exact-capacity-80", 80))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, newSpend.ID); err != nil {
		t.Fatal(err)
	}
	var capacity, reserved, distributed, remaining, debt int64
	readPool := func() {
		t.Helper()
		if err := s.pool.QueryRow(ctx, `SELECT total_emission_capacity,total_reserved,total_distributed,remaining_capacity,recovery_debt
			FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&capacity, &reserved, &distributed, &remaining, &debt); err != nil {
			t.Fatal(err)
		}
	}
	readPool()
	if capacity != 100 || reserved != 0 || distributed != 10 || remaining != 90 || debt != 0 {
		t.Fatalf("before refund C/S/D/M/H=%d/%d/%d/%d/%d", capacity, reserved, distributed, remaining, debt)
	}
	spend, err := s.LoadSystemSpend(ctx, "g18-fund")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-exact-refund-20", 20)); err != nil {
		t.Fatal(err)
	}
	readPool()
	if capacity != 80 || reserved != 0 || distributed != 10 || remaining != 70 || debt != 0 {
		t.Fatalf("after refund C/S/D/M/H=%d/%d/%d/%d/%d", capacity, reserved, distributed, remaining, debt)
	}
	var before, after string
	const snapshot = `SELECT jsonb_build_object(
		'pool',(SELECT to_jsonb(p) FROM black_iron_emission_pools p WHERE pool_id='GLOBAL'),
		'lot',(SELECT jsonb_agg(to_jsonb(l)) FROM mining_reward_issuance_lots l WHERE status='G21_SETTLED'),
		'item',(SELECT jsonb_agg(to_jsonb(i)) FROM character_inventory_items i),
		'grant',(SELECT jsonb_agg(to_jsonb(g)) FROM mining_reward_grants g),
		'recovery',(SELECT count(*) FROM black_iron_emission_recovery_entries))::text`
	if err := s.pool.QueryRow(ctx, snapshot).Scan(&before); err != nil {
		t.Fatal(err)
	}
	replay, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("completed replay under refund=%+v %v", replay, err)
	}
	if err := s.pool.QueryRow(ctx, snapshot).Scan(&after); err != nil || after != before {
		t.Fatalf("replay changed economy or inventory: before=%s after=%s err=%v", before, after, err)
	}
	if report, err := s.ReconcileBlackIronEmission(ctx); err != nil || !report.Balanced {
		t.Fatalf("recovery reconcile=%+v %v", report, err)
	}
}

func TestG21ConcurrentRefundAndSettlementSerializeOnPool(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	spend, err := s.LoadSystemSpend(ctx, "g18-fund")
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-race-refund", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	start := make(chan struct{})
	var settlementErr, refundErr error
	var settled MiningRewardSettlementReceipt
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		settled, settlementErr = s.SettleMiningRewardTEST(ctx, cmd)
	}()
	go func() {
		defer group.Done()
		<-start
		_, refundErr = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-race-refund-source", 15))
	}()
	close(start)
	group.Wait()
	if refundErr != nil {
		t.Fatalf("refund=%v", refundErr)
	}
	var capacity, reserved, distributed, remaining, debt, lots int64
	if err := s.pool.QueryRow(ctx, `SELECT total_emission_capacity,total_reserved,total_distributed,
		remaining_capacity,recovery_debt FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).
		Scan(&capacity, &reserved, &distributed, &remaining, &debt); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'`).Scan(&lots); err != nil {
		t.Fatal(err)
	}
	if capacity != 5 || remaining != 0 || debt != 5 {
		t.Fatalf("refund/settle conservation C/M/H=%d/%d/%d", capacity, remaining, debt)
	}
	if settlementErr == nil {
		if settled.Status != "COMPLETED" || reserved != 0 || distributed != 10 || lots != 1 {
			t.Fatalf("settle-first=%+v pool S/D=%d/%d lots=%d", settled, reserved, distributed, lots)
		}
	} else if !errors.Is(settlementErr, ErrMiningRewardDebt) || reserved != 10 || distributed != 0 || lots != 0 {
		t.Fatalf("refund-first err=%v pool S/D=%d/%d lots=%d", settlementErr, reserved, distributed, lots)
	}
	report, err := s.ReconcileBlackIronEmission(ctx)
	if err != nil || !report.Balanced {
		t.Fatalf("race recovery=%+v %v", report, err)
	}
}

func TestG21ZeroPowerKeepsReservationAndInventoryUntouched(t *testing.T) {
	s, _, intent := g20Fixture(t)
	ctx := context.Background()
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil || seal.TotalValidMiningPower != 0 {
		t.Fatalf("seal=%+v %v", seal, err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-zero-power", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || first.Status != "SEALED_NO_ELIGIBLE_POWER" {
		t.Fatalf("no-power=%+v %v", first, err)
	}
	replayed, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatalf("no-power replay=%+v %v", replayed, err)
	}
	var reserved, distributed, stacks, lots, grants, consumed, receipts int
	if err := s.pool.QueryRow(ctx, `SELECT
		(SELECT total_reserved FROM black_iron_emission_pools),
		(SELECT total_distributed FROM black_iron_emission_pools),
		(SELECT count(*) FROM mining_reward_inventory_projections),
		(SELECT count(*) FROM mining_reward_issuance_lots),
		(SELECT count(*) FROM mining_reward_grants),
		(SELECT count(*) FROM mining_reward_reservation_consumptions),
		(SELECT count(*) FROM mining_reward_settlement_receipts)`).
		Scan(&reserved, &distributed, &stacks, &lots, &grants, &consumed, &receipts); err != nil {
		t.Fatal(err)
	}
	if reserved != 10 || distributed != 0 || stacks != 0 || lots != 0 || grants != 0 || consumed != 0 || receipts != 0 {
		t.Fatalf("zero-power effects S/D/stack/lot/grant/consume/receipt=%d/%d/%d/%d/%d/%d/%d",
			reserved, distributed, stacks, lots, grants, consumed, receipts)
	}
}

func TestG21DifferentCommandSameInstanceAndChangedFingerprintReject(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-first-command", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	other := cmd
	other.CommandID = "g21-second-command"
	if _, err := s.SettleMiningRewardTEST(ctx, other); !errors.Is(err, ErrMiningRewardConflict) {
		t.Fatalf("second command=%v", err)
	}
	changed := cmd
	changed.ExpectedSealDigest = "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := s.SettleMiningRewardTEST(ctx, changed); !errors.Is(err, ErrMiningRewardConflict) {
		t.Fatalf("changed command=%v", err)
	}
}

func TestG21ConcurrentExactCommandPaysOnce(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := MiningRewardSettlementCommand{CommandID: "g21-concurrent-command", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	const workers = 10
	results := make([]MiningRewardSettlementReceipt, workers)
	errs := make([]error, workers)
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			results[i], errs[i] = s.SettleMiningRewardTEST(ctx, cmd)
		}(i)
	}
	group.Wait()
	for i := range results {
		if errs[i] != nil || !reflect.DeepEqual(results[i], results[0]) {
			t.Fatalf("worker %d result=%+v err=%v", i, results[i], errs[i])
		}
	}
	var lots, consumed int
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_issuance_lots WHERE status='G21_SETTLED'),
		(SELECT count(*) FROM mining_reward_reservation_consumptions)`).Scan(&lots, &consumed); err != nil {
		t.Fatal(err)
	}
	if lots != 1 || consumed != 1 {
		t.Fatalf("duplicate economic effects lots=%d consumed=%d", lots, consumed)
	}
}
