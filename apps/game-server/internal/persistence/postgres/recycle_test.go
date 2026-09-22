package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"fractallegend/game-server/internal/persistence"
	"fractallegend/game-server/internal/recycle"
)

var g16RequestedAt = time.Date(2026, 9, 22, 1, 0, 0, 0, time.UTC)

func g16Rule() recycle.Rule {
	return recycle.Rule{ID: "G16_INTERNAL_TEST", Version: "G16_RULE_V1", TemplateID: "item-mafa-tulong",
		Enabled: true, Recyclable: true,
		VerifiedInput:    []recycle.Material{{ID: recycle.MaterialRecycleScrap, Quantity: 10}},
		MaterialOutputs:  []recycle.Material{{ID: recycle.MaterialRecycleScrap, Quantity: 5}},
		ReputationReward: 2, MaxReturnBasisPoints: 5000, InputProvenance: "G16_TEST_FIXTURE",
		AntiFarm:  recycle.AntiFarmPolicy{PeriodCapPolicy: "UNDECIDED", DiminishingReturnPolicy: "UNDECIDED"},
		CreatedAt: g16RequestedAt}
}

func g16Service(t *testing.T, store *Store) *recycle.Service {
	t.Helper()
	registry, err := recycle.NewRegistry([]recycle.Rule{g16Rule()})
	if err != nil {
		t.Fatal(err)
	}
	return recycle.NewService(store, registry)
}

func g16Fixture(t *testing.T, count int) *Store {
	t.Helper()
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, persistence.Account{ID: "g16-account"}); err != nil {
		t.Fatal(err)
	}
	value := persistence.CharacterAggregate{Character: persistence.Character{ID: "g16-player", AccountID: "g16-account", Name: "G16 Test Player", ClassID: "warrior", ClassName: "Warrior", Level: 1, HP: 100, MP: 20},
		World: persistence.WorldState{MapID: "test-map"}}
	for i := 0; i < count; i++ {
		value.Items = append(value.Items, persistence.ItemInstance{InstanceID: fmt.Sprintf("g16-item-%03d", i), DefinitionID: "item-mafa-tulong", Name: "Test Sword", ItemType: "EQUIPMENT", Quantity: 1, SlotIndex: i, Location: persistence.InventoryLocation})
	}
	if _, err := store.CreateCharacter(ctx, value); err != nil {
		t.Fatal(err)
	}
	return store
}

func g16Intent(item, operation string) recycle.Intent {
	return recycle.Intent{OperationID: operation, PlayerID: "g16-player", ItemInstanceID: item,
		ExpectedItemRevision: 1, RuleID: "G16_INTERNAL_TEST", RequestedAt: g16RequestedAt}
}

func g16Counts(t *testing.T, store *Store) (item, material, reputation, receipt, audit int) {
	t.Helper()
	ctx := context.Background()
	for _, x := range []struct {
		query  string
		target *int
	}{
		{`SELECT count(*) FROM character_inventory_items WHERE item_type='EQUIPMENT'`, &item},
		{`SELECT count(*) FROM recycle_material_credits`, &material},
		{`SELECT count(*) FROM reputation_entries`, &reputation},
		{`SELECT count(*) FROM recycle_receipts`, &receipt},
		{`SELECT count(*) FROM recycle_audit_events`, &audit},
	} {
		if err := store.pool.QueryRow(ctx, x.query).Scan(x.target); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func requireExactG16Receipt(t *testing.T, first, loaded recycle.Receipt) {
	t.Helper()
	if !reflect.DeepEqual(first, loaded) {
		t.Fatalf("immutable receipt changed: first=%+v loaded=%+v", first, loaded)
	}
	firstJSON, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	loadedJSON, err := json.Marshal(loaded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, loadedJSON) {
		t.Fatalf("serialized receipt changed: first=%s loaded=%s", firstJSON, loadedJSON)
	}
}

func TestG16PostgresSuccessfulRecycleReplayConflictRestartAndSnapshot(t *testing.T) {
	store := g16Fixture(t, 1)
	ctx := context.Background()
	service := g16Service(t, store)
	intent := g16Intent("g16-item-000", "g16-success")
	intent.RequestedAt = intent.RequestedAt.Add(123 * time.Nanosecond)
	first, err := service.Recycle(ctx, intent)
	if err != nil {
		t.Fatal(err)
	}
	if first.ReputationAwarded != 2 || len(first.MaterialsAwarded) != 1 || first.MaterialsAwarded[0].Quantity != 5 || first.FBAwarded != 0 || first.ContributionAwarded != 0 || first.BlackIronAwarded != 0 {
		t.Fatalf("receipt=%+v", first)
	}
	if item, material, reputation, receipt, audit := g16Counts(t, store); item != 0 || material != 1 || reputation != 1 || receipt != 1 || audit != 1 {
		t.Fatalf("counts=%d %d %d %d %d", item, material, reputation, receipt, audit)
	}
	var balance int64
	if err := store.pool.QueryRow(ctx, `SELECT balance FROM reputation_accounts WHERE player_id='g16-player'`).Scan(&balance); err != nil || balance != 2 {
		t.Fatalf("balance=%d err=%v", balance, err)
	}
	loaded, err := store.LoadRecycle(ctx, intent.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	requireExactG16Receipt(t, first, loaded)
	replay, err := service.Recycle(ctx, intent)
	if err != nil {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	requireExactG16Receipt(t, first, replay)
	for _, change := range []func(*recycle.Intent){
		func(i *recycle.Intent) { i.PlayerID = "another-player" },
		func(i *recycle.Intent) { i.ItemInstanceID = "another-item" },
		func(i *recycle.Intent) { i.RuleID = "other-rule" },
		func(i *recycle.Intent) { i.ExpectedItemRevision++ },
		func(i *recycle.Intent) { i.RequestedAt = i.RequestedAt.Add(time.Microsecond) },
	} {
		changed := intent
		change(&changed)
		if _, err := service.Recycle(ctx, changed); !errors.Is(err, recycle.ErrConflict) {
			t.Fatalf("conflict=%+v err=%v", changed, err)
		}
	}
	reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(reopened.Close)
	modifiedRule := g16Rule()
	modifiedRule.Enabled = false
	registry, err := recycle.NewRegistry([]recycle.Rule{modifiedRule})
	if err != nil {
		t.Fatal(err)
	}
	restarted := recycle.NewService(reopened, registry)
	afterRestart, err := restarted.Recycle(ctx, intent)
	if err != nil {
		t.Fatalf("restart=%+v err=%v", afterRestart, err)
	}
	requireExactG16Receipt(t, first, afterRestart)
	if report, err := restarted.Reconcile(ctx); err != nil || !report.Balanced || report.Checked != 1 {
		t.Fatalf("reconcile=%+v err=%v", report, err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE recycle_receipts SET rule_version='forged' WHERE operation_id='g16-success'`); err == nil {
		t.Fatal("immutable receipt changed")
	}
	if _, err := service.Recycle(ctx, g16Intent("g16-item-000", "second-operation")); !errors.Is(err, recycle.ErrConsumed) {
		t.Fatalf("second recycle=%v", err)
	}
}

func TestG16PostgresRejectsOwnershipRevisionLocksUnknownAndDisabledRules(t *testing.T) {
	store := g16Fixture(t, 2)
	ctx := context.Background()
	service := g16Service(t, store)
	intent := g16Intent("g16-item-000", "reject")
	intent.PlayerID = "not-owner"
	if _, err := service.Recycle(ctx, intent); !errors.Is(err, recycle.ErrOwnership) {
		t.Fatalf("ownership=%v", err)
	}
	intent = g16Intent("g16-item-000", "reject")
	intent.ExpectedItemRevision = 2
	if _, err := service.Recycle(ctx, intent); !errors.Is(err, recycle.ErrRevision) {
		t.Fatalf("revision=%v", err)
	}
	intent = g16Intent("g16-item-000", "reject")
	intent.RuleID = "missing"
	if _, err := service.Recycle(ctx, intent); !errors.Is(err, recycle.ErrUnknownRule) {
		t.Fatalf("unknown=%v", err)
	}
	rule := g16Rule()
	rule.Enabled = false
	registry, err := recycle.NewRegistry([]recycle.Rule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recycle.NewService(store, registry).Recycle(ctx, g16Intent("g16-item-000", "reject")); !errors.Is(err, recycle.ErrDisabledRule) {
		t.Fatalf("disabled=%v", err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE character_inventory_items SET location='EQUIPMENT',equipment_slot='WEAPON' WHERE instance_id='g16-item-000'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO character_equipment(character_id,slot,item_instance_id) VALUES('g16-player','WEAPON','g16-item-000')`); err != nil {
		t.Fatal(err)
	}
	intent = g16Intent("g16-item-000", "equipped")
	intent.ExpectedItemRevision = 2
	if _, err := service.Recycle(ctx, intent); !errors.Is(err, recycle.ErrLocked) {
		t.Fatalf("equipped=%v", err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO trade_sessions(trade_id,player_a_id,player_b_id,state,revision,created_at,updated_at,expires_at)
        VALUES('g16-trade','g16-player','g16-player-2','NEGOTIATING',1,now(),now(),now()+interval '1 hour')`); err == nil {
		t.Fatal("trade fixture unexpectedly accepted missing player")
	}
	second := persistence.CharacterAggregate{Character: persistence.Character{ID: "g16-player-2", AccountID: "g16-account", Name: "Other", ClassID: "warrior", ClassName: "Warrior", Level: 1, HP: 100, MP: 20}, World: persistence.WorldState{MapID: "test-map"}}
	if _, err := store.CreateCharacter(ctx, second); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO trade_sessions(trade_id,player_a_id,player_b_id,state,revision,created_at,updated_at,expires_at)
        VALUES('g16-trade','g16-player','g16-player-2','NEGOTIATING',1,now(),now(),now()+interval '1 hour')`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO trade_item_locks(item_instance_id,trade_id,owner_character_id,quantity,created_at) VALUES('g16-item-001','g16-trade','g16-player',1,now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Recycle(ctx, g16Intent("g16-item-001", "locked")); !errors.Is(err, recycle.ErrLocked) {
		t.Fatalf("trade lock=%v", err)
	}
	if item, material, reputation, receipt, audit := g16Counts(t, store); item != 2 || material != 0 || reputation != 0 || receipt != 0 || audit != 0 {
		t.Fatalf("rejections changed state=%d %d %d %d %d", item, material, reputation, receipt, audit)
	}
}

func TestG16PostgresRollbackAtAllBoundaries(t *testing.T) {
	for _, point := range []string{"after_ownership", "after_rule_resolution", "before_item_consume", "after_item_consume", "after_material_credit", "after_reputation_credit", "after_receipt", "before_commit"} {
		t.Run(point, func(t *testing.T) {
			store := g16Fixture(t, 1)
			service := g16Service(t, store)
			store.recycleFailureInjector = func(at string) error {
				if at == point {
					return errors.New("injected G16 rollback")
				}
				return nil
			}
			intent := g16Intent("g16-item-000", "rollback-"+point)
			if _, err := service.Recycle(context.Background(), intent); err == nil {
				t.Fatal("injection committed")
			}
			if item, material, reputation, receipt, audit := g16Counts(t, store); item != 1 || material != 0 || reputation != 0 || receipt != 0 || audit != 0 {
				t.Fatalf("partial settlement=%d %d %d %d %d", item, material, reputation, receipt, audit)
			}
			store.recycleFailureInjector = nil
			if _, err := service.Recycle(context.Background(), intent); err != nil {
				t.Fatalf("retry=%v", err)
			}
		})
	}
}

func TestG16PostgresConcurrentSameAndDistinctItems(t *testing.T) {
	t.Run("same_item", func(t *testing.T) {
		store := g16Fixture(t, 1)
		service := g16Service(t, store)
		var wg sync.WaitGroup
		errs := make(chan error, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := service.Recycle(context.Background(), g16Intent("g16-item-000", fmt.Sprintf("same-%03d", i)))
				errs <- err
			}(i)
		}
		wg.Wait()
		close(errs)
		success, consumed := 0, 0
		for err := range errs {
			if err == nil {
				success++
			} else if errors.Is(err, recycle.ErrConsumed) {
				consumed++
			} else {
				t.Errorf("unexpected error: %v", err)
			}
		}
		if success != 1 || consumed != 99 {
			t.Fatalf("success=%d consumed=%d", success, consumed)
		}
		if item, material, reputation, receipt, audit := g16Counts(t, store); item != 0 || material != 1 || reputation != 1 || receipt != 1 || audit != 1 {
			t.Fatalf("counts=%d %d %d %d %d", item, material, reputation, receipt, audit)
		}
	})
	t.Run("distinct_items", func(t *testing.T) {
		store := g16Fixture(t, 100)
		service := g16Service(t, store)
		var wg sync.WaitGroup
		errs := make(chan error, 100)
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := service.Recycle(context.Background(), g16Intent(fmt.Sprintf("g16-item-%03d", i), fmt.Sprintf("distinct-%03d", i)))
				errs <- err
			}(i)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			if err != nil {
				t.Errorf("distinct item: %v", err)
			}
		}
		if item, material, reputation, receipt, audit := g16Counts(t, store); item != 0 || material != 100 || reputation != 100 || receipt != 100 || audit != 100 {
			t.Fatalf("counts=%d %d %d %d %d", item, material, reputation, receipt, audit)
		}
		if report, err := service.Reconcile(context.Background()); err != nil || !report.Balanced || report.Checked != 100 {
			t.Fatalf("reconcile=%+v err=%v", report, err)
		}
	})
}

func TestG16PostgresReconciliationDetectsCorruption(t *testing.T) {
	store := g16Fixture(t, 1)
	service := g16Service(t, store)
	if _, err := service.Recycle(context.Background(), g16Intent("g16-item-000", "reconcile")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(context.Background(), `UPDATE reputation_accounts SET balance=balance+1 WHERE player_id='g16-player'`); err != nil {
		t.Fatal(err)
	}
	if report, err := service.Reconcile(context.Background()); err != nil || report.Balanced {
		t.Fatalf("missed reputation corruption=%+v err=%v", report, err)
	}
	if _, err := store.pool.Exec(context.Background(), `UPDATE reputation_accounts SET balance=balance-1 WHERE player_id='g16-player'`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(context.Background(), `DELETE FROM character_inventory_items WHERE item_type='MATERIAL'`); err != nil {
		t.Fatal(err)
	}
	if report, err := service.Reconcile(context.Background()); err != nil || report.Balanced {
		t.Fatalf("missed inventory corruption=%+v err=%v", report, err)
	}
}

func TestG16RejectsDuplicateHistoryAndDetectsConsumedItemWithoutReceipt(t *testing.T) {
	store := g16Fixture(t, 1)
	ctx := context.Background()
	service := g16Service(t, store)
	if _, err := service.Recycle(ctx, g16Intent("g16-item-000", "history")); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO recycle_receipts(operation_id,player_id,item_instance_id,item_template_id,item_revision,rule_id,rule_version,rule_snapshot,requested_at,materials_awarded,reputation_awarded,created_at)
		SELECT 'duplicate-history',player_id,item_instance_id,item_template_id,item_revision,rule_id,rule_version,rule_snapshot,requested_at,materials_awarded,reputation_awarded,created_at FROM recycle_receipts WHERE operation_id='history'`); err == nil {
		t.Fatal("duplicate item receipt accepted")
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO reputation_entries(operation_id,player_id,amount,created_at) VALUES('history','g16-player',2,now())`); err == nil {
		t.Fatal("duplicate reputation entry accepted")
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO item_instance_lifecycle(instance_id,revision,consumed) VALUES('orphan-consumed',1,true)`); err != nil {
		t.Fatal(err)
	}
	if report, err := service.Reconcile(ctx); err != nil || report.Balanced {
		t.Fatalf("orphan consumed item undetected=%+v err=%v", report, err)
	}
}

func TestG16ItemRevisionSurvivesAggregateRewriteAndConsumedIDCannotReturn(t *testing.T) {
	store := g16Fixture(t, 1)
	ctx := context.Background()
	aggregate, err := store.LoadCharacter(ctx, "g16-player")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveCharacter(ctx, aggregate, aggregate.Character.Revision); err != nil {
		t.Fatal(err)
	}
	service := g16Service(t, store)
	if _, err := service.Recycle(ctx, g16Intent("g16-item-000", "stale")); !errors.Is(err, recycle.ErrRevision) {
		t.Fatalf("stale item revision=%v", err)
	}
	var revision int64
	if err := store.pool.QueryRow(ctx, `SELECT revision FROM item_instance_lifecycle WHERE instance_id='g16-item-000'`).Scan(&revision); err != nil || revision != 2 {
		t.Fatalf("revision=%d err=%v", revision, err)
	}
	intent := g16Intent("g16-item-000", "current")
	intent.ExpectedItemRevision = revision
	if _, err := service.Recycle(ctx, intent); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location)
		VALUES('g16-item-000','g16-player','item-mafa-tulong',0,'forged','EQUIPMENT',1,0,'INVENTORY')`); err == nil {
		t.Fatal("consumed instance reentered inventory")
	}
}

func TestG16HundredDuplicateOperationRetriesOneReceipt(t *testing.T) {
	store := g16Fixture(t, 1)
	service := g16Service(t, store)
	intent := g16Intent("g16-item-000", "duplicate-operation")
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Recycle(context.Background(), intent); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("duplicate request=%v", err)
		}
	}
	if item, material, reputation, receipt, audit := g16Counts(t, store); item != 0 || material != 1 || reputation != 1 || receipt != 1 || audit != 1 {
		t.Fatalf("counts=%d %d %d %d %d", item, material, reputation, receipt, audit)
	}
}

func TestG16HundredDifferentPlayersSettleIndependently(t *testing.T) {
	store := g16Fixture(t, 0)
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		player := fmt.Sprintf("g16-independent-player-%03d", i)
		item := fmt.Sprintf("g16-independent-item-%03d", i)
		value := persistence.CharacterAggregate{Character: persistence.Character{ID: player, AccountID: "g16-account", Name: player, ClassID: "warrior", ClassName: "Warrior", Level: 1, HP: 100, MP: 20},
			World: persistence.WorldState{MapID: "test-map"}, Items: []persistence.ItemInstance{{InstanceID: item, DefinitionID: "item-mafa-tulong", Name: "Test Sword", ItemType: "EQUIPMENT", Quantity: 1, SlotIndex: 0, Location: persistence.InventoryLocation}}}
		if _, err := store.CreateCharacter(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	service := g16Service(t, store)
	var wg sync.WaitGroup
	errs := make(chan error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			intent := g16Intent(fmt.Sprintf("g16-independent-item-%03d", i), fmt.Sprintf("independent-%03d", i))
			intent.PlayerID = fmt.Sprintf("g16-independent-player-%03d", i)
			_, err := service.Recycle(ctx, intent)
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Errorf("independent player=%v", err)
		}
	}
	if item, material, reputation, receipt, audit := g16Counts(t, store); item != 0 || material != 100 || reputation != 100 || receipt != 100 || audit != 100 {
		t.Fatalf("counts=%d %d %d %d %d", item, material, reputation, receipt, audit)
	}
}

func TestG16MigrationUpgradesG15Schema(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	legacy := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "0008_" {
			continue
		}
		data, readErr := fs.ReadFile(migrations, "migrations/"+entry.Name())
		if readErr != nil {
			t.Fatal(readErr)
		}
		legacy["migrations/"+entry.Name()] = &fstest.MapFile{Data: data}
	}
	if err := store.MigrateFS(ctx, legacy); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := store.pool.QueryRow(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&before); err != nil || before != 7 {
		t.Fatalf("G15 version=%d err=%v", before, err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var after int
	if err := store.pool.QueryRow(ctx, `SELECT max(version) FROM schema_migrations`).Scan(&after); err != nil || after != 8 {
		t.Fatalf("G16 version=%d err=%v", after, err)
	}
}
