package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"fractallegend/game-server/internal/persistence"
	"fractallegend/game-server/internal/trade"
)

var postgresTradeNow = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func postgresTradeCharacter(id, account, itemID string, quantity int) persistence.CharacterAggregate {
	return persistence.CharacterAggregate{
		Character: persistence.Character{ID: id, AccountID: account, Name: id, ClassID: "class-warrior", ClassName: "Warrior", Level: 1, HP: 100, MP: 50},
		World:     persistence.WorldState{MapID: "map-1"},
		Items:     []persistence.ItemInstance{{InstanceID: itemID, DefinitionID: "canonical:" + itemID, LegacyID: 20, Name: itemID, ItemType: "MATERIAL", Quantity: quantity, SlotIndex: 0, Location: persistence.InventoryLocation}},
	}
}

func postgresTradeService(t *testing.T) (*Store, *trade.Service) {
	t.Helper()
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, account := range []string{"trade-account-a", "trade-account-b"} {
		if err := store.CreateAccount(ctx, persistence.Account{ID: account}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateCharacter(ctx, postgresTradeCharacter("trade-player-a", "trade-account-a", "trade-item-x", 10)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateCharacter(ctx, postgresTradeCharacter("trade-player-b", "trade-account-b", "trade-item-y", 5)); err != nil {
		t.Fatal(err)
	}
	sequence := 0
	svc := trade.NewService(store, trade.Options{Now: func() time.Time { return postgresTradeNow }, NewID: func(prefix string) string { sequence++; return prefix + "-postgres-" + string(rune('0'+sequence)) }, InventoryCapacity: 20})
	return store, svc
}

func preparePostgresTrade(t *testing.T, svc *trade.Service, id string) {
	t.Helper()
	ctx := context.Background()
	s, err := svc.CreateTrade(ctx, id, "trade-player-a", "trade-player-b", postgresTradeNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.AddItem(ctx, id, "trade-player-a", "trade-item-x", 3, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.AddItem(ctx, id, "trade-player-b", "trade-item-y", 5, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.ConfirmTrade(ctx, id, "trade-player-a", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = svc.ConfirmTrade(ctx, id, "trade-player-b", s.Revision); err != nil {
		t.Fatal(err)
	}
}

func TestPostgresTradePersistsAcrossRestartAndFinalizesIdempotently(t *testing.T) {
	store, svc := postgresTradeService(t)
	ctx := context.Background()
	preparePostgresTrade(t, svc, "postgres-restart")
	restartedStore, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer restartedStore.Close()
	restarted := trade.NewService(restartedStore, trade.Options{Now: func() time.Time { return postgresTradeNow }, NewID: func(prefix string) string { return prefix + "-restart" }, InventoryCapacity: 20})
	loaded, err := restarted.GetTrade(ctx, "postgres-restart")
	if err != nil || loaded.State != trade.StateReadyToSettle {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	first, err := restarted.FinalizeTrade(ctx, loaded.TradeID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := restarted.FinalizeTrade(ctx, loaded.TradeID)
	if err != nil || second.TradeID != first.TradeID || second.SettlementID != first.SettlementID || !second.CompletedAt.Equal(first.CompletedAt) {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	a, _ := store.LoadCharacter(ctx, "trade-player-a")
	b, _ := store.LoadCharacter(ctx, "trade-player-b")
	if len(a.Items) != 2 || len(b.Items) != 1 || b.Items[0].Quantity != 3 {
		t.Fatalf("a=%+v b=%+v", a.Items, b.Items)
	}
}

func TestPostgresTradeFailureInjectionRollsBackRealTransaction(t *testing.T) {
	store, svc := postgresTradeService(t)
	ctx := context.Background()
	preparePostgresTrade(t, svc, "postgres-rollback")
	store.tradeFailureInjector = func(point string) error {
		if point == trade.FailureAfterFirstCharacterWrite {
			return errors.New("synthetic PostgreSQL transaction failure")
		}
		return nil
	}
	if _, err := svc.FinalizeTrade(ctx, "postgres-rollback"); !errors.Is(err, trade.ErrSettlementFailed) {
		t.Fatalf("err=%v", err)
	}
	a, _ := store.LoadCharacter(ctx, "trade-player-a")
	b, _ := store.LoadCharacter(ctx, "trade-player-b")
	loaded, _ := svc.GetTrade(ctx, "postgres-rollback")
	if len(a.Items) != 1 || a.Items[0].Quantity != 10 || len(b.Items) != 1 || b.Items[0].Quantity != 5 || loaded.State != trade.StateReadyToSettle {
		t.Fatalf("a=%+v b=%+v trade=%+v", a.Items, b.Items, loaded)
	}
}

func TestPostgresTradePersistentUniqueLockRejectsConcurrentTrade(t *testing.T) {
	_, svc := postgresTradeService(t)
	ctx := context.Background()
	first, _ := svc.CreateTrade(ctx, "postgres-lock-1", "trade-player-a", "trade-player-b", postgresTradeNow.Add(time.Hour))
	second, _ := svc.CreateTrade(ctx, "postgres-lock-2", "trade-player-a", "trade-player-b", postgresTradeNow.Add(time.Hour))
	if _, err := svc.AddItem(ctx, first.TradeID, first.PlayerAID, "trade-item-x", 1, first.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddItem(ctx, second.TradeID, second.PlayerAID, "trade-item-x", 1, second.Revision); !errors.Is(err, trade.ErrItemLocked) {
		t.Fatalf("err=%v", err)
	}
}
