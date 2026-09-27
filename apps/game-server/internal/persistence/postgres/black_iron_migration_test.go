package postgres

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"io/fs"
	"math"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"fractallegend/game-server/internal/blackiron"
	"fractallegend/game-server/internal/persistence"
	"fractallegend/game-server/internal/trade"
)

func g19Seed(t *testing.T, s *Store, id string, quantities ...int) persistence.CharacterAggregate {
	t.Helper()
	ctx := context.Background()
	if err := s.CreateAccount(ctx, persistence.Account{ID: id + "-account"}); err != nil {
		t.Fatal(err)
	}
	v := persistence.CharacterAggregate{Character: persistence.Character{ID: id, AccountID: id + "-account", Name: "Synthetic G19", ClassID: "warrior", ClassName: "Warrior", Level: 1, HP: 100}, World: persistence.WorldState{MapID: "synthetic-map"}}
	for i, q := range quantities {
		v.Items = append(v.Items, persistence.ItemInstance{InstanceID: fmt.Sprintf("%s-item-%d", id, i), DefinitionID: "synthetic-bun-material", LegacyID: 19001, Name: "Bun", ItemType: "MATERIAL", Quantity: q, SlotIndex: i, Location: "INVENTORY"})
	}
	if _, err := s.CreateCharacter(ctx, v); err != nil {
		t.Fatal(err)
	}
	loaded, err := s.LoadCharacter(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}
func g19Catalog(t *testing.T, s *Store) {
	t.Helper()
	if _, err := s.pool.Exec(context.Background(), `INSERT INTO black_iron_identity_aliases(definition_id,legacy_id,legacy_name,evidence) VALUES('synthetic-bun-material',19001,'Bun','G19_SYNTHETIC_FIXTURE')`); err != nil {
		t.Fatal(err)
	}
}
func g19Fixture(t *testing.T) *Store {
	t.Helper()
	s := integrationStore(t)
	if err := s.Migrate(context.Background()); err != nil {
		t.Fatal(err)
	}
	g19Catalog(t, s)
	return s
}
func g19RequireBalanced(t *testing.T, s *Store, n int) {
	t.Helper()
	r, e := s.ReconcileBlackIronInventory(context.Background())
	if e != nil || !r.Balanced || r.Checked != n {
		t.Fatalf("report=%+v err=%v", r, e)
	}
}
func TestG19LegacyLoadConservationEmptySingleMultiplePlayersReplayRestart(t *testing.T) {
	s := g19Fixture(t)
	ctx := context.Background()
	for _, tc := range []struct {
		id    string
		q     []int
		total int64
	}{{"empty", nil, 0}, {"single", []int{1}, 1}, {"multiple", []int{2, 7, 11}, 20}, {"boundary", []int{math.MaxInt32, math.MaxInt32}, 4294967294}} {
		t.Run(tc.id, func(t *testing.T) {
			before := g19Seed(t, s, tc.id, tc.q...)
			first, e := s.MigrateBlackIronInventory(ctx, tc.id)
			if e != nil {
				t.Fatal(e)
			}
			if first.PreTotal != tc.total || first.PostTotal != tc.total || first.Status != "COMPLETE" || first.TargetKind != "BLACK_IRON_ORE" || first.Version != blackiron.Version || len(first.Changes) != len(tc.q) {
				t.Fatalf("receipt=%+v", first)
			}
			after, e := s.LoadCharacter(ctx, tc.id)
			if e != nil {
				t.Fatal(e)
			}
			want := before
			want.Items = append([]persistence.ItemInstance(nil), before.Items...)
			for i := range want.Items {
				want.Items[i].Name = "黑铁矿石"
			}
			if len(tc.q) > 0 {
				want.Character.Revision++
			}
			if !reflect.DeepEqual(after, want) {
				t.Fatalf("after=%+v want=%+v", after, want)
			}
			repeat, e := s.MigrateBlackIronInventory(ctx, tc.id)
			if e != nil || !reflect.DeepEqual(first, repeat) {
				t.Fatalf("replay=%+v err=%v", repeat, e)
			}
			reopened, e := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if e != nil {
				t.Fatal(e)
			}
			defer reopened.Close()
			restart, e := reopened.MigrateBlackIronInventory(ctx, tc.id)
			if e != nil || !reflect.DeepEqual(first, restart) {
				t.Fatalf("restart=%+v err=%v", restart, e)
			}
		})
	}
	g19RequireBalanced(t, s, 4)
}
func TestG19ConcurrentSameAndDifferentCharacters(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "parallel-a", 3)
	g19Seed(t, s, "parallel-b", 5)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 24)
	receipts := make(chan blackiron.Receipt, 24)
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := "parallel-a"
			if i%2 == 1 {
				id = "parallel-b"
			}
			r, e := s.MigrateBlackIronInventory(ctx, id)
			errs <- e
			receipts <- r
		}(i)
	}
	wg.Wait()
	close(errs)
	close(receipts)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	seen := map[string]blackiron.Receipt{}
	for r := range receipts {
		if old, ok := seen[r.CharacterID]; ok && !reflect.DeepEqual(old, r) {
			t.Fatal("nonexact concurrent replay")
		}
		seen[r.CharacterID] = r
	}
	g19RequireBalanced(t, s, 2)
}
func TestG19RollbackRetryAllBoundaries(t *testing.T) {
	for _, point := range []string{"before_items", "after_items", "after_revision", "after_receipt", "before_commit"} {
		t.Run(point, func(t *testing.T) {
			s := g19Fixture(t)
			before := g19Seed(t, s, "rollback", 4, 6)
			s.blackIronFailureInjector = func(at string) error {
				if at == point {
					return errors.New("synthetic failure")
				}
				return nil
			}
			if _, e := s.MigrateBlackIronInventory(context.Background(), "rollback"); e == nil {
				t.Fatal("failure committed")
			}
			after, e := s.LoadCharacter(context.Background(), "rollback")
			if e != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("rollback=%+v err=%v", after, e)
			}
			g19RequireBalanced(t, s, 0)
			s.blackIronFailureInjector = nil
			r, e := s.MigrateBlackIronInventory(context.Background(), "rollback")
			if e != nil || r.PostTotal != 10 {
				t.Fatalf("retry=%+v err=%v", r, e)
			}
			g19RequireBalanced(t, s, 1)
		})
	}
}
func TestG19StaleAndFreshCompatibilitySaveCannotReintroduceBun(t *testing.T) {
	s := g19Fixture(t)
	before := g19Seed(t, s, "save", 9)
	ctx := context.Background()
	if _, e := s.MigrateBlackIronInventory(ctx, "save"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.SaveCharacter(ctx, before, before.Character.Revision); !errors.Is(e, persistence.ErrStaleRevision) {
		t.Fatalf("stale=%v", e)
	}
	fresh, e := s.LoadCharacter(ctx, "save")
	if e != nil {
		t.Fatal(e)
	}
	fresh.Items[0].Name = "Bun"
	fresh.Character.EXP = 1
	if _, e = s.SaveCharacter(ctx, fresh, fresh.Character.Revision); e != nil {
		t.Fatal(e)
	}
	got, e := s.LoadCharacter(ctx, "save")
	if e != nil || got.Items[0].Name != "黑铁矿石" || got.Items[0].Quantity != 9 {
		t.Fatalf("got=%+v e=%v", got, e)
	}
	g19RequireBalanced(t, s, 1)
}
func TestG19UnknownCurrencyAndTitleAreNotGuessed(t *testing.T) {
	s := g19Fixture(t)
	v := g19Seed(t, s, "unknown", 3)
	v.Items[0].DefinitionID = "legacy-currency"
	v.Items[0].LegacyID = 49
	v.Items[0].Name = "馒头点"
	v.Items[0].ItemType = "CURRENCY"
	if _, e := s.SaveCharacter(context.Background(), v, 1); e != nil {
		t.Fatal(e)
	}
	before, e := s.LoadCharacter(context.Background(), "unknown")
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.MigrateBlackIronInventory(context.Background(), "unknown")
	if e != nil || r.PreTotal != 0 {
		t.Fatalf("r=%+v e=%v", r, e)
	}
	after, e := s.LoadCharacter(context.Background(), "unknown")
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("unknown was converted")
	}
}
func TestG19InvalidQuantityMetadataAndRevisionRejectWithoutMutation(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{{"negative", `UPDATE character_inventory_items SET quantity=-1`}, {"zero", `UPDATE character_inventory_items SET quantity=0`}, {"overflow", `UPDATE character_inventory_items SET quantity=2147483648`}, {"legacy-id", `UPDATE character_inventory_items SET legacy_id=49`}, {"type", `UPDATE character_inventory_items SET item_type='CURRENCY'`}, {"name", `UPDATE character_inventory_items SET name='馒头点'`}, {"revision-overflow", `UPDATE characters SET revision=9223372036854775807`}, {"item-revision-overflow", `UPDATE item_instance_lifecycle SET revision=9223372036854775807`}} {
		t.Run(tc.name, func(t *testing.T) {
			s := g19Fixture(t)
			g19Seed(t, s, "invalid", 1)
			_, e := s.pool.Exec(context.Background(), tc.sql)
			if tc.name == "negative" || tc.name == "zero" || tc.name == "overflow" {
				if e == nil {
					t.Fatal("database accepted invalid quantity")
				}
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.MigrateBlackIronInventory(context.Background(), "invalid"); e == nil {
				t.Fatal("invalid migration accepted")
			}
			var n int
			if e = s.pool.QueryRow(context.Background(), `SELECT count(*) FROM black_iron_migration_receipts`).Scan(&n); e != nil || n != 0 {
				t.Fatalf("receipt count=%d e=%v", n, e)
			}
		})
	}
}
func TestG19ImmutableCatalogReceiptAndReconciliationMismatch(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "audit", 5)
	ctx := context.Background()
	if _, e := s.MigrateBlackIronInventory(ctx, "audit"); e != nil {
		t.Fatal(e)
	}
	for _, sql := range []string{`UPDATE black_iron_identity_aliases SET legacy_name='other'`, `DELETE FROM black_iron_identity_aliases`, `INSERT INTO black_iron_identity_aliases VALUES('other',2,'Other','evidence')`, `UPDATE black_iron_migration_receipts SET post_total=6`, `DELETE FROM black_iron_migration_receipts`} {
		if _, e := s.pool.Exec(ctx, sql); e == nil {
			t.Fatalf("immutable state accepted %s", sql)
		}
	}
	g19RequireBalanced(t, s, 1)
	if _, e := s.pool.Exec(ctx, `ALTER TABLE character_inventory_items DISABLE TRIGGER inventory_black_iron_integrity; UPDATE character_inventory_items SET quantity=6; ALTER TABLE character_inventory_items ENABLE TRIGGER inventory_black_iron_integrity`); e != nil {
		t.Fatal(e)
	}
	r, e := s.ReconcileBlackIronInventory(ctx)
	if e != nil || r.Balanced || len(r.Mismatches) == 0 {
		t.Fatalf("mismatch=%+v e=%v", r, e)
	}
	if _, e = s.MigrateBlackIronInventory(ctx, "audit"); !errors.Is(e, blackiron.ErrMismatch) {
		t.Fatalf("corrupt replay accepted: %v", e)
	}
	var quantity int
	if e = s.pool.QueryRow(ctx, `SELECT quantity FROM character_inventory_items`).Scan(&quantity); e != nil || quantity != 6 {
		t.Fatal("reconcile silently repaired")
	}
}
func TestG19NotConfiguredMissingAndCanonicalWithoutMarkerFailClosed(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if e := s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	g19Seed(t, s, "config", 2)
	if _, e := s.MigrateBlackIronInventory(ctx, "config"); !errors.Is(e, blackiron.ErrNotConfigured) {
		t.Fatalf("unconfigured=%v", e)
	}
	g19Catalog(t, s)
	if _, e := s.MigrateBlackIronInventory(ctx, "missing"); !errors.Is(e, persistence.ErrNotFound) {
		t.Fatalf("missing=%v", e)
	}
	if _, e := s.pool.Exec(ctx, `ALTER TABLE character_inventory_items DISABLE TRIGGER inventory_black_iron_integrity; UPDATE character_inventory_items SET name='黑铁矿石'; ALTER TABLE character_inventory_items ENABLE TRIGGER inventory_black_iron_integrity`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.MigrateBlackIronInventory(ctx, "config"); !errors.Is(e, blackiron.ErrMismatch) {
		t.Fatalf("unmarked target accepted: %v", e)
	}
}

// Snapshot every economic table, including immutable histories and nonzero debt.
func g19Economy(t *testing.T, s *Store) map[string]string {
	t.Helper()
	ctx := context.Background()
	rows, e := s.pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND (tablename LIKE 'fb_%' OR tablename LIKE 'contribution_%' OR tablename LIKE 'system_spend%' OR tablename LIKE 'eligible_%' OR tablename LIKE 'black_iron_emission%' OR tablename LIKE 'mining_%' OR tablename LIKE 'reputation_%' OR tablename LIKE 'recycle_%') ORDER BY tablename`)
	if e != nil {
		t.Fatal(e)
	}
	var tables []string
	for rows.Next() {
		var n string
		if e := rows.Scan(&n); e != nil {
			t.Fatal(e)
		}
		tables = append(tables, n)
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
	rows.Close()
	out := map[string]string{}
	for _, n := range tables {
		var v string
		if e = s.pool.QueryRow(ctx, `SELECT coalesce(jsonb_agg(v ORDER BY v::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(t) v FROM `+n+` t) q`).Scan(&v); e != nil {
			t.Fatal(e)
		}
		out[n] = v
	}
	return out
}
func TestG19EconomicAndMiningIsolationWithReservedCapacityAndRecoveryDebt(t *testing.T) {
	s, mining, contributions, spend := g18Fund(t, 10)
	ctx := context.Background()
	block, e := mining.Create(ctx, "g19-existing-reservation")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = mining.Finalize(ctx, block.BlockID); e != nil {
		t.Fatal(e)
	}
	if _, e = contributions.RefundSystemSpend(ctx, g15Refund(spend, "g19-existing-refund", 10)); e != nil {
		t.Fatal(e)
	}
	g19Catalog(t, s)
	g19Seed(t, s, "economic", 3, 4)
	if _, e = s.pool.Exec(ctx, `INSERT INTO reputation_accounts(player_id,balance) VALUES('economic',7)`); e != nil {
		t.Fatal(e)
	}
	before := g19Economy(t, s)
	r, e := s.MigrateBlackIronInventory(ctx, "economic")
	if e != nil || r.PreTotal != 7 || r.PostTotal != 7 {
		t.Fatalf("r=%+v e=%v", r, e)
	}
	if !reflect.DeepEqual(before, g19Economy(t, s)) {
		t.Fatal("migration changed economics")
	}
	if _, e = s.MigrateBlackIronInventory(ctx, "economic"); e != nil {
		t.Fatal(e)
	}
	g19RequireBalanced(t, s, 1)
	if !reflect.DeepEqual(before, g19Economy(t, s)) {
		t.Fatal("replay/reconciliation changed economics")
	}
}

func TestG19TenToElevenSchemaMigrationPreservesExistingInventoryAndHistory(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	entries, e := fs.ReadDir(migrations, "migrations")
	if e != nil {
		t.Fatal(e)
	}
	for _, entry := range entries {
		if entry.Name() < "0011_" {
			raw, e := fs.ReadFile(migrations, "migrations/"+entry.Name())
			if e != nil {
				t.Fatal(e)
			}
			old["migrations/"+entry.Name()] = &fstest.MapFile{Data: raw}
		}
	}
	if e = s.MigrateFS(ctx, old); e != nil {
		t.Fatal(e)
	}
	before := g19Seed(t, s, "upgrade", 2, 8)
	economy := g19Economy(t, s)
	// This historical fixture verifies exactly 0010 -> 0011, independently of
	// later schema additions. Keep the existing eleven-version assertion.
	raw, e := fs.ReadFile(migrations, "migrations/0011_black_iron_inventory_migration.sql")
	if e != nil {
		t.Fatal(e)
	}
	old["migrations/0011_black_iron_inventory_migration.sql"] = &fstest.MapFile{Data: raw}
	if e = s.MigrateFS(ctx, old); e != nil {
		t.Fatal(e)
	}
	if e = s.MigrateFS(ctx, old); e != nil {
		t.Fatal(e)
	}
	after, e := s.LoadCharacter(ctx, "upgrade")
	if e != nil || !reflect.DeepEqual(before, after) || !reflect.DeepEqual(economy, g19Economy(t, s)) {
		t.Fatal("schema migration altered existing state")
	}
	var versions, aliases, receipts int
	for _, x := range []struct {
		sql string
		n   *int
	}{{`SELECT count(*) FROM schema_migrations`, &versions}, {`SELECT count(*) FROM black_iron_identity_aliases`, &aliases}, {`SELECT count(*) FROM black_iron_migration_receipts`, &receipts}} {
		if e = s.pool.QueryRow(ctx, x.sql).Scan(x.n); e != nil {
			t.Fatal(e)
		}
	}
	if versions != 11 || aliases != 0 || receipts != 0 {
		t.Fatalf("versions=%d aliases=%d receipts=%d", versions, aliases, receipts)
	}
	g19Catalog(t, s)
	r, e := s.MigrateBlackIronInventory(ctx, "upgrade")
	if e != nil || r.PreTotal != 10 || r.PostTotal != 10 {
		t.Fatalf("r=%+v e=%v", r, e)
	}
}

func TestG19ReconciliationDetectsDoubleRepresentationAndReceiptCorruption(t *testing.T) {
	for _, kind := range []string{"extra-instance", "forged-receipt"} {
		t.Run(kind, func(t *testing.T) {
			s := g19Fixture(t)
			g19Seed(t, s, "corrupt", 5)
			ctx := context.Background()
			if _, e := s.MigrateBlackIronInventory(ctx, "corrupt"); e != nil {
				t.Fatal(e)
			}
			if kind == "extra-instance" {
				if _, e := s.pool.Exec(ctx, `ALTER TABLE character_inventory_items DISABLE TRIGGER inventory_black_iron_integrity; INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location) VALUES('duplicate','corrupt','synthetic-bun-material',19001,'Bun','MATERIAL',5,1,'INVENTORY'); ALTER TABLE character_inventory_items ENABLE TRIGGER inventory_black_iron_integrity`); e != nil {
					t.Fatal(e)
				}
			} else {
				// Synthetic corruption only, bypassing immutability to prove the auditor.
				if _, e := s.pool.Exec(ctx, `ALTER TABLE black_iron_migration_receipts DISABLE TRIGGER black_iron_migration_receipts_immutable; UPDATE black_iron_migration_receipts SET receipt=jsonb_set(receipt,'{preTotal}','6'); ALTER TABLE black_iron_migration_receipts ENABLE TRIGGER black_iron_migration_receipts_immutable`); e != nil {
					t.Fatal(e)
				}
			}
			r, e := s.ReconcileBlackIronInventory(ctx)
			if e != nil || r.Balanced || len(r.Mismatches) == 0 {
				t.Fatalf("r=%+v e=%v", r, e)
			}
			if _, e = s.MigrateBlackIronInventory(ctx, "corrupt"); !errors.Is(e, blackiron.ErrMismatch) {
				t.Fatalf("corrupt replay=%v", e)
			}
		})
	}
}

func TestG19MigratedAssetMovementFailsClosedAndAggregateSaveStillWorks(t *testing.T) {
	s := g19Fixture(t)
	ctx := context.Background()
	g19Seed(t, s, "protected", 5)
	g19Seed(t, s, "recipient")
	if _, e := s.MigrateBlackIronInventory(ctx, "protected"); e != nil {
		t.Fatal(e)
	}
	for _, sql := range []string{`UPDATE character_inventory_items SET character_id='recipient' WHERE character_id='protected'`, `UPDATE character_inventory_items SET quantity=4 WHERE character_id='protected'`, `DELETE FROM character_inventory_items WHERE character_id='protected'`} {
		if _, e := s.pool.Exec(ctx, sql); e == nil {
			t.Fatalf("unsupported ore movement committed: %s", sql)
		}
	}
	if _, e := s.pool.Exec(ctx, `UPDATE character_inventory_items SET slot_index=3 WHERE character_id='protected'`); e != nil {
		t.Fatal(e)
	}
	v, e := s.LoadCharacter(ctx, "protected")
	if e != nil {
		t.Fatal(e)
	}
	v.Character.EXP = 9
	v.Items[0].Name = "Bun"
	if _, e = s.SaveCharacter(ctx, v, v.Character.Revision); e != nil {
		t.Fatal(e)
	}
	g19RequireBalanced(t, s, 1)
}
func TestG19CatalogOldSnapshotCannotAppendAfterFirstReceipt(t *testing.T) {
	s := g19Fixture(t)
	ctx := context.Background()
	g19Seed(t, s, "snapshot", 1)
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	var n int
	if e = tx.QueryRow(ctx, `SELECT count(*) FROM black_iron_migration_receipts`).Scan(&n); e != nil || n != 0 {
		t.Fatal(e)
	}
	if _, e = s.MigrateBlackIronInventory(ctx, "snapshot"); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `INSERT INTO black_iron_identity_aliases VALUES('late-alias',2,'Late Bun','synthetic evidence')`); e == nil {
		t.Fatal("old snapshot bypassed frozen catalog")
	}
}
func TestG19HistoryTruncateRejected(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "truncate", 1)
	ctx := context.Background()
	if _, e := s.MigrateBlackIronInventory(ctx, "truncate"); e != nil {
		t.Fatal(e)
	}
	for _, sql := range []string{`TRUNCATE black_iron_identity_aliases`, `TRUNCATE black_iron_migration_receipts CASCADE`} {
		if _, e := s.pool.Exec(ctx, sql); e == nil {
			t.Fatalf("history truncated: %s", sql)
		}
	}
	g19RequireBalanced(t, s, 1)
}
func TestG19ReconciliationRejectsUnmarkedCanonicalAsset(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "unmarked", 3)
	ctx := context.Background()
	if _, e := s.pool.Exec(ctx, `ALTER TABLE character_inventory_items DISABLE TRIGGER inventory_black_iron_integrity; UPDATE character_inventory_items SET name='黑铁矿石'; ALTER TABLE character_inventory_items ENABLE TRIGGER inventory_black_iron_integrity`); e != nil {
		t.Fatal(e)
	}
	r, e := s.ReconcileBlackIronInventory(ctx)
	if e != nil || r.Balanced || len(r.Mismatches) == 0 {
		t.Fatalf("unmarked canonical falsely balanced: %+v %v", r, e)
	}
}

func historicalMigrationsThrough(t *testing.T, last string) fstest.MapFS {
	t.Helper()
	result := fstest.MapFS{}
	files, e := fs.ReadDir(migrations, "migrations")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if file.Name() <= last {
			raw, e := fs.ReadFile(migrations, "migrations/"+file.Name())
			if e != nil {
				t.Fatal(e)
			}
			result["migrations/"+file.Name()] = &fstest.MapFile{Data: raw}
		}
	}
	return result
}
func TestG19WholePartialTradeFailAtomicallyAndUnrelatedTradePreservesOre(t *testing.T) {
	for _, amount := range []int{3, 10, 0} {
		for _, recipientMigrated := range []bool{false, true} {
			t.Run(fmt.Sprintf("amount-%d-recipient-%t", amount, recipientMigrated), func(t *testing.T) {
				s, svc := postgresTradeService(t)
				ctx := context.Background()
				g19Catalog(t, s)
				if _, e := s.pool.Exec(ctx, `UPDATE character_inventory_items SET definition_id='synthetic-bun-material',legacy_id=19001,name='Bun' WHERE instance_id='trade-item-x'`); e != nil {
					t.Fatal(e)
				}
				if _, e := s.MigrateBlackIronInventory(ctx, "trade-player-a"); e != nil {
					t.Fatal(e)
				}
				if recipientMigrated {
					if _, e := s.MigrateBlackIronInventory(ctx, "trade-player-b"); e != nil {
						t.Fatal(e)
					}
				}
				a, e := s.LoadCharacter(ctx, "trade-player-a")
				if e != nil {
					t.Fatal(e)
				}
				b, e := s.LoadCharacter(ctx, "trade-player-b")
				if e != nil {
					t.Fatal(e)
				}
				tradeState, e := svc.CreateTrade(ctx, "g19-trade", "trade-player-a", "trade-player-b", postgresTradeNow.Add(time.Hour))
				if e != nil {
					t.Fatal(e)
				}
				if amount > 0 {
					tradeState, e = svc.AddItem(ctx, tradeState.TradeID, "trade-player-a", "trade-item-x", amount, tradeState.Revision)
					if e != nil {
						t.Fatal(e)
					}
				}
				tradeState, e = svc.AddItem(ctx, tradeState.TradeID, "trade-player-b", "trade-item-y", 5, tradeState.Revision)
				if e != nil {
					t.Fatal(e)
				}
				tradeState, e = svc.ConfirmTrade(ctx, tradeState.TradeID, "trade-player-a", tradeState.Revision)
				if e != nil {
					t.Fatal(e)
				}
				tradeState, e = svc.ConfirmTrade(ctx, tradeState.TradeID, "trade-player-b", tradeState.Revision)
				if e != nil {
					t.Fatal(e)
				}
				_, e = svc.FinalizeTrade(ctx, tradeState.TradeID)
				afterA, ea := s.LoadCharacter(ctx, "trade-player-a")
				afterB, eb := s.LoadCharacter(ctx, "trade-player-b")
				if ea != nil || eb != nil {
					t.Fatal("failed aggregate load")
				}
				if amount > 0 {
					if !errors.Is(e, trade.ErrSettlementFailed) || !reflect.DeepEqual(a, afterA) || !reflect.DeepEqual(b, afterB) {
						t.Fatalf("ore trade escaped rollback: err=%v a=%+v b=%+v", e, afterA, afterB)
					}
				} else {
					if e != nil || len(afterA.Items) != 2 || afterA.Items[0].Quantity != 10 {
						t.Fatalf("unrelated trade=%v a=%+v", e, afterA)
					}
				}
				checked := 1
				if recipientMigrated {
					checked = 2
				}
				g19RequireBalanced(t, s, checked)
			})
		}
	}
}

func TestG19MissingCommitmentRejectsReplayAndReconciliation(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "missing-provenance", 3)
	ctx := context.Background()
	if _, e := s.MigrateBlackIronInventory(ctx, "missing-provenance"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.pool.Exec(ctx, `ALTER TABLE black_iron_migration_assets DISABLE TRIGGER black_iron_assets_immutable; DELETE FROM black_iron_migration_assets; ALTER TABLE black_iron_migration_assets ENABLE TRIGGER black_iron_assets_immutable`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.MigrateBlackIronInventory(ctx, "missing-provenance"); !errors.Is(e, blackiron.ErrMismatch) {
		t.Fatalf("missing provenance replay=%v", e)
	}
	r, e := s.ReconcileBlackIronInventory(ctx)
	if e != nil || r.Balanced {
		t.Fatalf("missing provenance audit=%+v %v", r, e)
	}
}

func TestG19FreshSaveCannotCreateDuplicateOreRepresentation(t *testing.T) {
	s := g19Fixture(t)
	g19Seed(t, s, "duplicate-save", 5)
	ctx := context.Background()
	if _, e := s.MigrateBlackIronInventory(ctx, "duplicate-save"); e != nil {
		t.Fatal(e)
	}
	v, e := s.LoadCharacter(ctx, "duplicate-save")
	if e != nil {
		t.Fatal(e)
	}
	extra := v.Items[0]
	extra.InstanceID = "extra-ore"
	extra.SlotIndex = 1
	extra.Name = "Bun"
	v.Items = append(v.Items, extra)
	if _, e = s.SaveCharacter(ctx, v, v.Character.Revision); e == nil {
		t.Fatal("fresh save created a second ore representation")
	}
	g19RequireBalanced(t, s, 1)
}
