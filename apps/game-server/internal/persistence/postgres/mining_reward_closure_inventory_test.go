package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"fractallegend/game-server/internal/blackiron"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/persistence"
	"github.com/jackc/pgx/v5/pgconn"
)

func g21InventoryCommand(t *testing.T, s *Store, i miningpower.ActionIntent, id string) MiningRewardSettlementCommand {
	t.Helper()
	if _, err := s.FinalizeMiningBlock(context.Background(), i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(context.Background(), i.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	return MiningRewardSettlementCommand{CommandID: id, BlockInstanceID: i.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
}

func g21InventoryActivity(t *testing.T, s *Store, base miningpower.ActionIntent, blockID, instanceID, player, suffix string) miningpower.ActionIntent {
	t.Helper()
	b, err := s.LoadMiningBlock(context.Background(), blockID)
	if err != nil {
		t.Fatal(err)
	}
	session := g20Session(t, s, base)
	session.ID = "closure-session-" + suffix
	session.PlayerID = player
	session.AccountID = player + "-account"
	session.BlockID, session.BlockInstanceID = blockID, instanceID
	session.OpenedAt, session.ExpiresAt = b.StartedAt, b.ScheduledEndAt
	g20Insert(t, s, "mining_power_sessions", session)
	source := g20Source(t, s, base)
	source.ID = "closure-event-" + suffix
	source.ActivityID = "mpa:" + source.ID
	source.PlayerID = player
	source.ActivitySessionID = session.ID
	source.BlockID, source.BlockInstanceID = blockID, instanceID
	source.ExpiresAt = b.ScheduledEndAt
	source.ObservedAt = time.Now().UTC().Truncate(time.Microsecond)
	g20Insert(t, s, "mining_power_source_events", source)
	i := miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID, ActivitySessionID: session.ID, BlockID: blockID, BlockInstanceID: instanceID}
	g20Accept(t, s, miningpower.Principal{PlayerID: player, AccountID: session.AccountID}, i)
	return i
}

// T068: losing owner locks or canonical grant order would lose revisions or
// duplicate slots when opposite participant arrival orders settle together.
func TestG21ClosureT068ConcurrentSharedOwners(t *testing.T) {
	for _, owners := range []int{1, 2} {
		t.Run(fmt.Sprintf("owners-%d", owners), func(t *testing.T) {
			s, p, i := g20Fixture(t)
			g19Catalog(t, s)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			players := []string{p.PlayerID}
			if owners == 2 {
				g19Seed(t, s, "g21-other-owner")
				players = append(players, "g21-other-owner")
			}
			g20Accept(t, s, p, i)
			if owners == 2 {
				g21InventoryActivity(t, s, i, i.BlockID, i.BlockInstanceID, players[1], "first-other")
			}
			birth, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Create(ctx, "closure-second")
			if err != nil {
				t.Fatal(err)
			}
			b, err := s.LoadMiningBlock(ctx, birth.BlockID)
			if err != nil {
				t.Fatal(err)
			}
			var second miningpower.ActionIntent
			for n := len(players) - 1; n >= 0; n-- {
				second = g21InventoryActivity(t, s, i, b.ID, b.BlockInstanceID, players[n], fmt.Sprintf("second-%d", n))
			}
			commands := []MiningRewardSettlementCommand{g21InventoryCommand(t, s, i, "closure-first"), g21InventoryCommand(t, s, second, "closure-second")}
			start := make(chan struct{})
			results := make([]MiningRewardSettlementReceipt, 2)
			errs := make([]error, 2)
			var wg sync.WaitGroup
			for n := range commands {
				wg.Add(1)
				go func(n int) {
					defer wg.Done()
					<-start
					results[n], errs[n] = s.SettleMiningRewardTEST(ctx, commands[n])
				}(n)
			}
			close(start)
			wg.Wait()
			for n, r := range results {
				if errs[n] != nil || r.Status != "COMPLETED" || r.TotalOre != 10 {
					t.Fatalf("settlement %d=%+v %v", n, r, errs[n])
				}
				for j, g := range r.Grants {
					if j > 0 && r.Grants[j-1].CharacterID >= g.CharacterID {
						t.Fatal("noncanonical owner order")
					}
					if g.InventoryRevisionAfter != g.InventoryRevisionBefore+1 {
						t.Fatalf("revision=%+v", g)
					}
				}
			}
			for _, player := range players {
				a, err := s.LoadCharacter(ctx, player)
				if err != nil || a.Character.Revision != 3 || len(a.Items) != 2 {
					t.Fatalf("owner=%+v %v", a, err)
				}
				want := 20 / owners
				if a.Items[0].Quantity+a.Items[1].Quantity != want || a.Items[0].InstanceID == a.Items[1].InstanceID || a.Items[0].SlotIndex == a.Items[1].SlotIndex {
					t.Fatalf("lost/overwritten reward=%+v", a.Items)
				}
			}
			before := g21ClosureSnapshot(t, s)
			for n, c := range commands {
				r, err := s.SettleMiningRewardTEST(ctx, c)
				if err != nil || !reflect.DeepEqual(r, results[n]) {
					t.Fatalf("replay=%+v %v", r, err)
				}
			}
			if before != g21ClosureSnapshot(t, s) {
				t.Fatal("replay changed rows")
			}
		})
	}
}

// T069: an old whole-aggregate writer must never erase committed FINAL stacks,
// including when a caller substitutes the new revision on its stale payload.
func TestG21ClosureT069StaleAndTamperedAggregateSave(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	stale, err := s.LoadCharacter(ctx, p.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := g21InventoryCommand(t, s, i, "closure-stale")
	if _, err = s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	if _, err = s.SaveCharacter(ctx, stale, stale.Character.Revision); !errors.Is(err, persistence.ErrStaleRevision) {
		t.Fatalf("stale save=%v", err)
	}
	stale.Character.Revision++
	if _, err = s.SaveCharacter(ctx, stale, stale.Character.Revision); err == nil {
		t.Fatal("forged current revision erased reward")
	}
	if before != g21ClosureSnapshot(t, s) {
		t.Fatal("rejected aggregate save changed committed rows")
	}
}

func g21LegacyRewardFixture(t *testing.T) (*Store, miningpower.Principal, miningpower.ActionIntent) {
	t.Helper()
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	a, err := s.LoadCharacter(ctx, p.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	a.Items = []persistence.ItemInstance{{InstanceID: "closure-legacy-bun", DefinitionID: "synthetic-bun-material", LegacyID: 19001, Name: "Bun", ItemType: "MATERIAL", Quantity: 7, SlotIndex: 0, Location: "INVENTORY"}}
	if _, err = s.SaveCharacter(ctx, a, a.Character.Revision); err != nil {
		t.Fatal(err)
	}
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	return s, p, i
}

// T070/T122: exercise both lock schedules with an actual first G19 conversion,
// not just a replay that already has a receipt before G21 starts.
func TestG21ClosureT070ConcurrentG19AndReward(t *testing.T) {
	for _, first := range []string{"migration", "settlement"} {
		t.Run(first+"-first", func(t *testing.T) {
			s, p, i := g21LegacyRewardFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := g21InventoryCommand(t, s, i, "closure-migration-race")
			held, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			pause := func() { once.Do(func() { close(held); <-release }) }
			if first == "migration" {
				s.blackIronFailureInjector = func(at string) error {
					if at == "before_items" {
						pause()
					}
					return nil
				}
			} else {
				s.settlementFailureInjector = func(at string) error {
					if at == "after_inventory_projection" {
						pause()
					}
					return nil
				}
			}
			var migration blackiron.Receipt
			var reward MiningRewardSettlementReceipt
			var migrationErr, rewardErr error
			var wg sync.WaitGroup
			migrate := func() { defer wg.Done(); migration, migrationErr = s.MigrateBlackIronInventory(ctx, p.PlayerID) }
			settle := func() { defer wg.Done(); reward, rewardErr = s.SettleMiningRewardTEST(ctx, cmd) }
			wg.Add(2)
			if first == "migration" {
				go migrate()
			} else {
				go settle()
			}
			select {
			case <-held:
			case <-ctx.Done():
				t.Fatal("first worker did not reach lock barrier")
			}
			if first == "migration" {
				go settle()
			} else {
				go migrate()
			}
			// Observe the second backend waiting on the common catalog lock,
			// rather than relying on goroutine scheduling as concurrency evidence.
			waiting := false
			for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
				var n int
				if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%black_iron_migration_catalog%'`).Scan(&n); err != nil {
					close(release)
					wg.Wait()
					t.Fatal(err)
				}
				if n > 0 {
					waiting = true
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			close(release)
			wg.Wait()
			if !waiting {
				t.Fatal("second worker never blocked on catalog lock")
			}
			s.blackIronFailureInjector = nil
			s.settlementFailureInjector = nil
			if migrationErr != nil || rewardErr != nil {
				t.Fatalf("migration=%v reward=%v", migrationErr, rewardErr)
			}
			if migration.PreTotal != 7 || migration.PostTotal != 7 || len(migration.Changes) != 1 || reward.TotalOre != 10 {
				t.Fatalf("migration=%+v reward=%+v", migration, reward)
			}
			restarted, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			defer restarted.Close()
			replay, err := restarted.MigrateBlackIronInventory(ctx, p.PlayerID)
			if err != nil || !reflect.DeepEqual(migration, replay) {
				t.Fatalf("migration restart replay=%+v %v", replay, err)
			}
			a, err := restarted.LoadCharacter(ctx, p.PlayerID)
			if err != nil || len(a.Items) != 2 || a.Items[0].Quantity != 7 || a.Items[1].Quantity != 10 {
				t.Fatalf("coexistence=%+v %v", a, err)
			}
			for _, audit := range []func(context.Context) (MiningRewardAudit, error){restarted.ReconcileMiningRewardIssuancePrerequisite} {
				r, e := audit(ctx)
				if e != nil || !r.Balanced {
					t.Fatalf("reward audit=%+v %v", r, e)
				}
			}
			r, e := restarted.ReconcileBlackIronInventory(ctx)
			if e != nil || !r.Balanced {
				t.Fatalf("migration audit=%+v %v", r, e)
			}
		})
	}
}

func TestG21ClosureT071T072OverflowRollsBackEveryRow(t *testing.T) {
	for _, kind := range []string{"revision", "slot"} {
		t.Run(kind, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, p, i)
			cmd := g21InventoryCommand(t, s, i, "closure-overflow")
			if kind == "revision" {
				if _, err := s.pool.Exec(ctx, `UPDATE characters SET revision=$1 WHERE id=$2`, int64(math.MaxInt64), p.PlayerID); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := s.pool.Exec(ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location) VALUES('last-slot',$1,'ordinary-material',42,'ordinary','MATERIAL',1,2147483647,'INVENTORY')`, p.PlayerID); err != nil {
					t.Fatal(err)
				}
			}
			before := g21ClosureSnapshot(t, s)
			_, err := s.SettleMiningRewardTEST(ctx, cmd)
			if kind == "revision" && !errors.Is(err, ErrMiningRewardInvariant) {
				t.Fatalf("revision overflow=%v", err)
			}
			if kind == "slot" {
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "22003" {
					t.Fatalf("slot overflow=%v", err)
				}
			}
			if before != g21ClosureSnapshot(t, s) {
				t.Fatal("overflow changed persisted rows")
			}
		})
	}
}

func TestG21ClosureT101T102RewardProvenanceCoexistsWithMigration(t *testing.T) {
	s, p, i := g21LegacyRewardFixture(t)
	ctx := context.Background()
	original, err := s.MigrateBlackIronInventory(ctx, p.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	cmd := g21InventoryCommand(t, s, i, "closure-provenance")
	reward, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.MigrateBlackIronInventory(ctx, p.PlayerID)
	if err != nil || !reflect.DeepEqual(original, replay) {
		t.Fatalf("G19 receipt mutated %+v %v", replay, err)
	}
	var aliases, migrations, assets, rewards, projections int
	if err = s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM black_iron_identity_aliases),(SELECT count(*) FROM black_iron_migration_receipts),(SELECT count(*) FROM black_iron_migration_assets),(SELECT count(*) FROM mining_reward_issuance_lots WHERE source_type='MINING_REWARD' AND status='G21_SETTLED' AND material_definition_id='synthetic-bun-material'),(SELECT count(*) FROM mining_reward_inventory_projections p JOIN mining_reward_issuance_lots l USING(issuance_id) JOIN character_inventory_items i USING(instance_id) WHERE p.quantity=10 AND i.quantity=p.quantity AND i.definition_id=l.material_definition_id AND p.instance_id NOT IN(SELECT instance_id FROM black_iron_migration_assets))`).Scan(&aliases, &migrations, &assets, &rewards, &projections); err != nil {
		t.Fatal(err)
	}
	if aliases != 1 || migrations != 1 || assets != 1 || rewards != 1 || projections != 1 || reward.TotalOre != 10 {
		t.Fatalf("source counts=%d/%d/%d/%d/%d", aliases, migrations, assets, rewards, projections)
	}
	if r, e := s.ReconcileBlackIronInventory(ctx); e != nil || !r.Balanced {
		t.Fatalf("G19=%+v %v", r, e)
	}
	if r, e := s.ReconcileMiningRewardIssuancePrerequisite(ctx); e != nil || !r.Balanced {
		t.Fatalf("G21=%+v %v", r, e)
	}
}

// T122 regression for the settlement-first fix: neither a marker nor a forged
// canonical name authorizes exclusion from G19's exact legacy conversion.
func TestG21ClosureT122FirstMigrationRejectsCorruptReward(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"lot-digest", `UPDATE mining_reward_issuance_lots SET source_digest=repeat('0',64)`},
		{"projection-quantity", `UPDATE mining_reward_inventory_projections SET quantity=quantity+1`},
		{"physical-owner", `UPDATE character_inventory_items SET character_id='closure-other-owner' WHERE instance_id IN(SELECT instance_id FROM mining_reward_inventory_projections)`},
		{"missing-lot", `DELETE FROM mining_reward_issuance_lots`},
		{"forged-canonical", `UPDATE character_inventory_items SET name='黑铁矿石' WHERE instance_id='closure-legacy-bun'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, i := g21LegacyRewardFixture(t)
			ctx := context.Background()
			g19Seed(t, s, "closure-other-owner")
			cmd := g21InventoryCommand(t, s, i, "closure-first-g19-corrupt")
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			before := g21ClosureSnapshot(t, s)
			if _, err = s.MigrateBlackIronInventory(ctx, p.PlayerID); err == nil {
				t.Fatal("first G19 migration trusted corrupt reward")
			}
			if before != g21ClosureSnapshot(t, s) {
				t.Fatal("rejected migration changed history or inventory")
			}
		})
	}
}

func TestG21ClosureT122LegacyReplayRestartReorderWithReward(t *testing.T) {
	s, p, i := g21LegacyRewardFixture(t)
	ctx := context.Background()
	a, err := s.LoadCharacter(ctx, p.PlayerID)
	if err != nil {
		t.Fatal(err)
	}
	second := a.Items[0]
	second.InstanceID = "closure-second-bun"
	second.Quantity = 3
	second.SlotIndex = 1
	a.Items = append(a.Items, second)
	if _, err = s.SaveCharacter(ctx, a, a.Character.Revision); err != nil {
		t.Fatal(err)
	}
	migration, err := s.MigrateBlackIronInventory(ctx, p.PlayerID)
	if err != nil || migration.PreTotal != 10 || migration.PostTotal != 10 || len(migration.Changes) != 2 {
		t.Fatalf("migration=%+v %v", migration, err)
	}
	cmd := g21InventoryCommand(t, s, i, "closure-reorder")
	reward, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	a, err = s.LoadCharacter(ctx, p.PlayerID)
	if err != nil || len(a.Items) != 3 {
		t.Fatalf("inventory=%+v %v", a, err)
	}
	a.Items[0], a.Items[2] = a.Items[2], a.Items[0]
	if _, err = s.SaveCharacter(ctx, a, a.Character.Revision); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	before := g21ClosureSnapshot(t, restarted)
	if r, e := restarted.MigrateBlackIronInventory(ctx, p.PlayerID); e != nil || !reflect.DeepEqual(r, migration) {
		t.Fatalf("G19 after reorder=%+v %v", r, e)
	}
	if r, e := restarted.SettleMiningRewardTEST(ctx, cmd); e != nil || !reflect.DeepEqual(r, reward) {
		t.Fatalf("G21 after reorder=%+v %v", r, e)
	}
	if r, e := restarted.ReconcileBlackIronInventory(ctx); e != nil || !r.Balanced {
		t.Fatalf("G19 audit=%+v %v", r, e)
	}
	if r, e := restarted.ReconcileMiningRewardIssuancePrerequisite(ctx); e != nil || !r.Balanced {
		t.Fatalf("G21 audit=%+v %v", r, e)
	}
	if before != g21ClosureSnapshot(t, restarted) {
		t.Fatal("reorder/restart replay changed history")
	}
}
