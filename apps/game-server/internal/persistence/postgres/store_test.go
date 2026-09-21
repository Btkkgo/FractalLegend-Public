package postgres

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"fractallegend/game-server/internal/persistence"
)

func postgresAggregate() persistence.CharacterAggregate {
	return persistence.CharacterAggregate{
		Character: persistence.Character{ID: "character-postgres", AccountID: "account-postgres", Name: "Durable Warrior", ClassID: "class-warrior", ClassName: "战士", Level: 1, EXP: 10, HP: 91, MP: 27},
		World:     persistence.WorldState{MapID: "map-current", X: 4, Y: 6},
		Items: []persistence.ItemInstance{
			{InstanceID: "item-loot", DefinitionID: "item-awakening", LegacyID: 20, Name: "觉醒石", ItemType: "MATERIAL", Quantity: 1, SlotIndex: 0, Location: persistence.InventoryLocation},
			{InstanceID: "item-equipped", DefinitionID: "item-mafa-tulong", LegacyID: 411, Name: "玛法屠龙", ItemType: "EQUIPMENT", Quantity: 1, SlotIndex: 1, Location: persistence.EquipmentLocation, EquipmentSlot: "WEAPON"},
		},
		Equipment: []persistence.EquippedItem{{Slot: "WEAPON", ItemInstanceID: "item-equipped"}},
		Skills:    []persistence.SkillState{{SkillID: "skill-liehuo", Learned: true, Level: 1}},
	}
}

func integrationStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("FRACTAL_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("FRACTAL_TEST_DATABASE_URL is not configured")
	}
	if !strings.Contains(url, "fractal_g9_test") {
		t.Fatal("integration database name must contain fractal_g9_test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	if _, err = store.pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestPostgresMigrationIsRepeatableAndRejectsFailedVersion(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	bad := fstest.MapFS{"migrations/9999_broken.sql": {Data: []byte("CREATE TABLE migration_should_rollback(id bigint); INVALID SQL;")}}
	if err := store.MigrateFS(ctx, bad); err == nil {
		t.Fatal("broken migration succeeded")
	}
	var versionCount, tableCount int
	if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations WHERE version=9999").Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if err := store.pool.QueryRow(ctx, "SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='migration_should_rollback'").Scan(&tableCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 0 || tableCount != 0 {
		t.Fatalf("failed migration leaked state: version=%d table=%d", versionCount, tableCount)
	}
	if _, err := fs.Stat(migrations, "migrations/0001_initial_persistence.sql"); err != nil {
		t.Fatalf("versioned migration missing: %v", err)
	}
	if _, err := fs.Stat(migrations, "migrations/0002_trade_foundation.sql"); err != nil {
		t.Fatalf("trade migration missing: %v", err)
	}
}

func TestPostgresRoundTripsAggregateAndRejectsStaleSave(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	created := time.Date(2026, 9, 20, 11, 0, 0, 0, time.UTC)
	if err := store.CreateAccount(ctx, persistence.Account{ID: "account-postgres", CreatedAt: created, UpdatedAt: created}); err != nil {
		t.Fatal(err)
	}
	account, err := store.LoadAccount(ctx, "account-postgres")
	if err != nil || account.ID != "account-postgres" || !account.CreatedAt.Equal(created) {
		t.Fatalf("account=%#v error=%v", account, err)
	}
	value := postgresAggregate()
	revision, err := store.CreateCharacter(ctx, value)
	if err != nil || revision != 1 {
		t.Fatalf("create revision=%d error=%v", revision, err)
	}
	loaded, err := store.LoadCharacter(ctx, value.Character.ID)
	if err != nil || loaded.Character.Revision != 1 || loaded.World != value.World || len(loaded.Items) != 2 || len(loaded.Equipment) != 1 || len(loaded.Skills) != 1 {
		t.Fatalf("loaded=%#v error=%v", loaded, err)
	}
	stale := loaded
	loaded.Character.EXP = 20
	loaded.World.X = 8
	next, err := store.SaveCharacter(ctx, loaded, 1)
	if err != nil || next != 2 {
		t.Fatalf("save revision=%d error=%v", next, err)
	}
	stale.Character.EXP = 999
	if _, err = store.SaveCharacter(ctx, stale, 1); !errors.Is(err, persistence.ErrStaleRevision) {
		t.Fatalf("stale error=%v", err)
	}
	loaded, _ = store.LoadCharacter(ctx, value.Character.ID)
	if loaded.Character.EXP != 20 || loaded.World.X != 8 || loaded.Character.Revision != 2 {
		t.Fatalf("stale writer changed data: %#v", loaded)
	}
}

func TestPostgresChildFailureRollsBackWholeAggregate(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, persistence.Account{ID: "account-postgres"}); err != nil {
		t.Fatal(err)
	}
	value := postgresAggregate()
	if _, err := store.CreateCharacter(ctx, value); err != nil {
		t.Fatal(err)
	}
	value.Character.EXP = 777
	value.Items[0].Name = strings.Repeat("x", 129)
	if _, err := store.SaveCharacter(ctx, value, 1); err == nil {
		t.Fatal("constraint failure did not abort save")
	}
	loaded, err := store.LoadCharacter(ctx, value.Character.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Character.EXP != 10 || loaded.Character.Revision != 1 || loaded.Items[0].Name != "觉醒石" {
		t.Fatalf("partial aggregate escaped rollback: %#v", loaded)
	}
}

func TestPostgresRejectsPartialAggregateLoad(t *testing.T) {
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateAccount(ctx, persistence.Account{ID: "account-postgres"}); err != nil {
		t.Fatal(err)
	}
	value := postgresAggregate()
	if _, err := store.CreateCharacter(ctx, value); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, "DELETE FROM character_equipment WHERE character_id=$1", value.Character.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadCharacter(ctx, value.Character.ID); !errors.Is(err, persistence.ErrInvalidAggregate) {
		t.Fatalf("partial load error=%v", err)
	}
}

func TestPostgresUnavailableFailsClosedWithoutDSNDisclosure(t *testing.T) {
	userValue, credentialValue, databaseValue := "redaction-user", "redaction-credential", "redaction-database"
	secretURL := strings.Join([]string{"postgres", "://", userValue, ":", credentialValue, "@127.0.0.1:1/", databaseValue, "?connect_timeout=1"}, "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Open(ctx, secretURL)
	if !errors.Is(err, persistence.ErrUnavailable) {
		t.Fatalf("error=%v", err)
	}
	if strings.Contains(err.Error(), userValue) || strings.Contains(err.Error(), credentialValue) || strings.Contains(err.Error(), databaseValue) {
		t.Fatalf("database URL leaked: %v", err)
	}
}
