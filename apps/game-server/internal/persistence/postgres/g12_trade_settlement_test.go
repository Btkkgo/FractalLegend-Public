package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/persistence"
	"fractallegend/game-server/internal/trade"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresG12Fixture struct {
	store  *Store
	trade  *trade.Service
	ledger *ledger.Service
}

func newPostgresG12Fixture(t *testing.T) postgresG12Fixture {
	t.Helper()
	store := integrationStore(t)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	for _, account := range []string{"trade-account-a", "trade-account-b", "trade-account-c"} {
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
	if _, err := store.CreateCharacter(ctx, postgresTradeCharacter("trade-player-c", "trade-account-c", "trade-item-z", 1)); err != nil {
		t.Fatal(err)
	}
	var ids atomic.Int64
	newID := func(prefix string) string { return fmt.Sprintf("%s-g12-%d", prefix, ids.Add(1)) }
	ledgerService := ledger.NewService(store, ledger.Options{Now: func() time.Time { return postgresTradeNow }, NewID: newID})
	for _, account := range []struct{ id, owner string }{{"g12-fb-a", "trade-player-a"}, {"g12-fb-b", "trade-player-b"}, {"g12-fb-c", "trade-player-c"}} {
		if _, err := ledgerService.CreatePlayerAccount(ctx, account.id, account.owner); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ledgerService.CreateSystemAccount(ctx, postgresFixtureLiabilityAccountID, "g12-fixture-liability"); err != nil {
		t.Fatal(err)
	}
	fixture := postgresLedgerFixture{store: store, service: ledgerService}
	postgresCredit(t, fixture, "g12-fb-a", 1_000, "g12-a")
	postgresCredit(t, fixture, "g12-fb-b", 500, "g12-b")
	tradeService := trade.NewService(store, trade.Options{Now: func() time.Time { return postgresTradeNow }, NewID: newID, InventoryCapacity: 20})
	return postgresG12Fixture{store: store, trade: tradeService, ledger: ledgerService}
}

func preparePostgresG12FBOnly(t *testing.T, fixture postgresG12Fixture, id, a, b string, aFB, bFB int64) {
	t.Helper()
	ctx := context.Background()
	s, err := fixture.trade.CreateTrade(ctx, id, a, b, postgresTradeNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if aFB > 0 {
		s, err = fixture.trade.SetFBOffer(ctx, id, a, aFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	if bFB > 0 {
		s, err = fixture.trade.SetFBOffer(ctx, id, b, bFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err = fixture.trade.ConfirmTrade(ctx, id, a, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = fixture.trade.ConfirmTrade(ctx, id, b, s.Revision); err != nil {
		t.Fatal(err)
	}
}

func preparePostgresG12Trade(t *testing.T, fixture postgresG12Fixture, id string, aFB, bFB int64) {
	t.Helper()
	ctx := context.Background()
	s, err := fixture.trade.CreateTrade(ctx, id, "trade-player-a", "trade-player-b", postgresTradeNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	s, err = fixture.trade.AddItem(ctx, id, "trade-player-a", "trade-item-x", 10, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = fixture.trade.AddItem(ctx, id, "trade-player-b", "trade-item-y", 5, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if aFB > 0 {
		s, err = fixture.trade.SetFBOffer(ctx, id, "trade-player-a", aFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	if bFB > 0 {
		s, err = fixture.trade.SetFBOffer(ctx, id, "trade-player-b", bFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err = fixture.trade.ConfirmTrade(ctx, id, "trade-player-a", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.trade.ConfirmTrade(ctx, id, "trade-player-b", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
}

func TestG12PostgresMixedSettlementAndRestartReplay(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	ctx := context.Background()
	preparePostgresG12Trade(t, fixture, "g12-postgres-mixed", 125, 20)
	first, err := fixture.trade.FinalizeTrade(ctx, "g12-postgres-mixed")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.LedgerTransactionIDs) != 2 {
		t.Fatalf("result=%+v", first)
	}
	a, _ := fixture.store.LoadCharacter(ctx, "trade-player-a")
	b, _ := fixture.store.LoadCharacter(ctx, "trade-player-b")
	if len(a.Items) != 1 || a.Items[0].InstanceID != "trade-item-y" || len(b.Items) != 1 || b.Items[0].InstanceID != "trade-item-x" {
		t.Fatalf("a=%+v b=%+v", a.Items, b.Items)
	}
	if postgresBalance(t, fixture.ledger, "g12-fb-a") != 895 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 605 {
		t.Fatal("gross balances are incorrect")
	}
	restartedStore, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer restartedStore.Close()
	restarted := trade.NewService(restartedStore, trade.Options{Now: func() time.Time { return postgresTradeNow }, NewID: func(prefix string) string { return prefix + "-unused" }, InventoryCapacity: 20})
	second, err := restarted.FinalizeTrade(ctx, "g12-postgres-mixed")
	if err != nil || second.SettlementID != first.SettlementID || len(second.LedgerTransactionIDs) != 2 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	if postgresBalance(t, fixture.ledger, "g12-fb-a") != 895 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 605 {
		t.Fatal("restart replay moved funds")
	}
}

func TestG12PostgresFailureAfterLedgerWriteRollsBackItemsFBAndState(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	ctx := context.Background()
	preparePostgresG12Trade(t, fixture, "g12-postgres-rollback", 100, 0)
	fixture.store.tradeFailureInjector = func(point string) error {
		if point == trade.FailureAfterLedgerWrite {
			return errors.New("injected after ledger write")
		}
		return nil
	}
	if _, err := fixture.trade.FinalizeTrade(ctx, "g12-postgres-rollback"); !errors.Is(err, trade.ErrSettlementFailed) {
		t.Fatalf("err=%v", err)
	}
	a, _ := fixture.store.LoadCharacter(ctx, "trade-player-a")
	b, _ := fixture.store.LoadCharacter(ctx, "trade-player-b")
	loaded, _ := fixture.trade.GetTrade(ctx, "g12-postgres-rollback")
	if a.Items[0].InstanceID != "trade-item-x" || b.Items[0].InstanceID != "trade-item-y" || loaded.State != trade.StateReadyToSettle || loaded.Settlement != nil {
		t.Fatalf("a=%+v b=%+v trade=%+v", a.Items, b.Items, loaded)
	}
	if postgresBalance(t, fixture.ledger, "g12-fb-a") != 1_000 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 500 {
		t.Fatal("failed transaction changed FB")
	}
}

func TestG12PostgresAllFailureGatesRollback(t *testing.T) {
	for _, point := range []string{trade.FailureAfterItemOwnershipUpdate, trade.FailureAfterLedgerWrite, trade.FailureBeforeCompletedState, trade.FailureBeforeCommit} {
		t.Run(point, func(t *testing.T) {
			fixture := newPostgresG12Fixture(t)
			ctx := context.Background()
			id := "g12-pg-failure-" + point
			preparePostgresG12Trade(t, fixture, id, 100, 0)
			fixture.store.tradeFailureInjector = func(got string) error {
				if got == point {
					return errors.New("injected " + point)
				}
				return nil
			}
			if _, err := fixture.trade.FinalizeTrade(ctx, id); !errors.Is(err, trade.ErrSettlementFailed) {
				t.Fatalf("err=%v", err)
			}
			a, _ := fixture.store.LoadCharacter(ctx, "trade-player-a")
			b, _ := fixture.store.LoadCharacter(ctx, "trade-player-b")
			loaded, _ := fixture.trade.GetTrade(ctx, id)
			if a.Items[0].InstanceID != "trade-item-x" || b.Items[0].InstanceID != "trade-item-y" || loaded.State != trade.StateReadyToSettle || postgresBalance(t, fixture.ledger, "g12-fb-a") != 1_000 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 500 {
				t.Fatalf("a=%+v b=%+v trade=%+v", a.Items, b.Items, loaded)
			}
		})
	}
}

func TestG12PostgresBalanceUpdateFailureRollsBack(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	ctx := context.Background()
	preparePostgresG12Trade(t, fixture, "g12-pg-balance-failure", 100, 0)
	fixture.store.ledgerFailureInjector = func(point string) error {
		if point == ledger.FailureAfterFirstEntry {
			return errors.New("injected balance failure")
		}
		return nil
	}
	if _, err := fixture.trade.FinalizeTrade(ctx, "g12-pg-balance-failure"); !errors.Is(err, trade.ErrSettlementFailed) {
		t.Fatalf("err=%v", err)
	}
	if postgresBalance(t, fixture.ledger, "g12-fb-a") != 1_000 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 500 {
		t.Fatal("balance failure leaked state")
	}
	loaded, _ := fixture.trade.GetTrade(ctx, "g12-pg-balance-failure")
	if loaded.State != trade.StateReadyToSettle || loaded.Settlement != nil {
		t.Fatalf("trade=%+v", loaded)
	}
}

func TestG12PostgresConcurrentTradesCannotDoubleSpendFB(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	preparePostgresG12FBOnly(t, fixture, "g12-pg-double-1", "trade-player-a", "trade-player-b", 1_000, 0)
	preparePostgresG12FBOnly(t, fixture, "g12-pg-double-2", "trade-player-a", "trade-player-c", 1_000, 0)
	var success, insufficient atomic.Int64
	var wg sync.WaitGroup
	for _, id := range []string{"g12-pg-double-1", "g12-pg-double-2"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := fixture.trade.FinalizeTrade(context.Background(), id)
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, ledger.ErrInsufficientBalance) {
				insufficient.Add(1)
			} else {
				t.Errorf("%s: %v", id, err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || insufficient.Load() != 1 || postgresBalance(t, fixture.ledger, "g12-fb-a") != 0 {
		t.Fatalf("success=%d insufficient=%d", success.Load(), insufficient.Load())
	}
}

func TestG12PostgresOppositeDirectionTradesFinishWithoutDeadlock(t *testing.T) {
	fixture := newPostgresG12Fixture(t)
	preparePostgresG12FBOnly(t, fixture, "g12-pg-opposite-1", "trade-player-a", "trade-player-b", 100, 0)
	preparePostgresG12FBOnly(t, fixture, "g12-pg-opposite-2", "trade-player-b", "trade-player-a", 100, 0)
	var wg sync.WaitGroup
	for _, id := range []string{"g12-pg-opposite-1", "g12-pg-opposite-2"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := fixture.trade.FinalizeTrade(context.Background(), id); err != nil {
				t.Errorf("%s: %v", id, err)
			}
		}()
	}
	wg.Wait()
	if postgresBalance(t, fixture.ledger, "g12-fb-a") != 1_000 || postgresBalance(t, fixture.ledger, "g12-fb-b") != 500 {
		t.Fatal("opposite transfers changed net balances")
	}
}

func TestG12PostgresRetryableErrorsAreBoundedClassifications(t *testing.T) {
	for _, code := range []string{"40001", "40P01"} {
		if !isRetryableTradeError(&pgconn.PgError{Code: code}) {
			t.Fatalf("code %s not retryable", code)
		}
	}
	if isRetryableTradeError(&pgconn.PgError{Code: "23505"}) {
		t.Fatal("uniqueness conflict must not enter retry loop")
	}
}

func TestG12MigrationDefinesTradeFBAndReceiptColumns(t *testing.T) {
	data, err := os.ReadFile("migrations/0004_fb_trade_settlement.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, token := range []string{"player_a_fb_offer", "player_b_fb_offer", "ledger_transaction_ids", "CHECK (player_a_fb_offer >= 0)"} {
		if !strings.Contains(text, token) {
			t.Fatalf("migration missing %q", token)
		}
	}
}
