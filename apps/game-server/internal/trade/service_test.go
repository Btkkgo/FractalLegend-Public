package trade

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"fractallegend/game-server/internal/persistence"
)

var testNow = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func testAggregate(id string, item persistence.ItemInstance) persistence.CharacterAggregate {
	return persistence.CharacterAggregate{
		Character: persistence.Character{ID: id, AccountID: "account-" + id, Name: id, ClassID: "class-warrior", ClassName: "Warrior", Level: 1, HP: 100, MP: 50, Revision: 1},
		World:     persistence.WorldState{MapID: "map-1"},
		Items:     []persistence.ItemInstance{item},
	}
}

func testItem(id string, quantity int) persistence.ItemInstance {
	return persistence.ItemInstance{InstanceID: id, DefinitionID: "canonical:" + id, LegacyID: 20, Name: id, ItemType: "MATERIAL", Quantity: quantity, SlotIndex: 0, Location: persistence.InventoryLocation}
}

func testService(t *testing.T) (*Service, *MemoryRepository) {
	t.Helper()
	repo := NewMemoryRepository()
	if err := repo.SeedCharacter(testAggregate("player-a", testItem("item-x", 10))); err != nil {
		t.Fatal(err)
	}
	if err := repo.SeedCharacter(testAggregate("player-b", testItem("item-y", 5))); err != nil {
		t.Fatal(err)
	}
	sequence := 0
	svc := NewService(repo, Options{Now: func() time.Time { return testNow }, NewID: func(prefix string) string { sequence++; return fmt.Sprintf("%s-%d", prefix, sequence) }, InventoryCapacity: 20})
	return svc, repo
}

func createTrade(t *testing.T, svc *Service, id string) Session {
	t.Helper()
	s, err := svc.CreateTrade(context.Background(), id, "player-a", "player-b", testNow.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func readyTrade(t *testing.T, svc *Service, id string, quantity int) Session {
	t.Helper()
	s := createTrade(t, svc, id)
	s, err := svc.AddItem(context.Background(), id, "player-a", "item-x", quantity, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.AddItem(context.Background(), id, "player-b", "item-y", 5, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.ConfirmTrade(context.Background(), id, "player-a", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.ConfirmTrade(context.Background(), id, "player-b", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if s.State != StateReadyToSettle {
		t.Fatalf("state=%s", s.State)
	}
	return s
}

func ownerHas(t *testing.T, repo *MemoryRepository, owner, itemID string, quantity int) {
	t.Helper()
	c, err := repo.Character(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range c.Items {
		if item.InstanceID == itemID && item.Quantity == quantity {
			return
		}
	}
	t.Fatalf("%s does not own %s quantity %d: %+v", owner, itemID, quantity, c.Items)
}

func TestBasicTradeTransfersBothItemsAtomically(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-basic", 10)
	result, err := svc.FinalizeTrade(context.Background(), "trade-basic")
	if err != nil || result.SettlementID == "" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	ownerHas(t, repo, "player-a", "item-y", 5)
	ownerHas(t, repo, "player-b", "item-x", 10)
}

func TestOfferChangeInvalidatesBothConfirmations(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-revision")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 2, s.Revision)
	s, _ = svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	s, err := svc.AddItem(context.Background(), s.TradeID, "player-b", "item-y", 1, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if s.PlayerAConfirmedRevision != 0 || s.PlayerBConfirmedRevision != 0 || s.Revision != 3 {
		t.Fatalf("session=%+v", s)
	}
	if _, err = svc.FinalizeTrade(context.Background(), s.TradeID); !errors.Is(err, ErrNotReady) {
		t.Fatalf("err=%v", err)
	}
}

func TestDoubleFinalizeReturnsOneSettlement(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-double", 10)
	first, err := svc.FinalizeTrade(context.Background(), "trade-double")
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.FinalizeTrade(context.Background(), "trade-double")
	if err != nil || second != first {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	ownerHas(t, repo, "player-b", "item-x", 10)
}

func TestConcurrentTradesCannotLockSameItem(t *testing.T) {
	svc, _ := testService(t)
	a := createTrade(t, svc, "trade-lock-1")
	b := createTrade(t, svc, "trade-lock-2")
	if _, err := svc.AddItem(context.Background(), a.TradeID, "player-a", "item-x", 1, a.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddItem(context.Background(), b.TradeID, "player-a", "item-x", 1, b.Revision); !errors.Is(err, ErrItemLocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestCancelReleasesLocksWithoutChangingOwnership(t *testing.T) {
	svc, repo := testService(t)
	s := createTrade(t, svc, "trade-cancel")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	if _, err := svc.CancelTrade(context.Background(), s.TradeID, "player-a"); err != nil {
		t.Fatal(err)
	}
	locked, _ := svc.IsItemLocked(context.Background(), "item-x")
	if locked {
		t.Fatal("lock remains")
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
	if _, err := svc.CancelTrade(context.Background(), s.TradeID, "player-a"); err != nil {
		t.Fatalf("idempotent cancel: %v", err)
	}
}

func TestExpireReleasesLocksAndPreventsSettlement(t *testing.T) {
	svc, repo := testService(t)
	s := createTrade(t, svc, "trade-expire")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	if _, err := svc.ExpireTrade(context.Background(), s.TradeID, testNow.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	locked, _ := svc.IsItemLocked(context.Background(), "item-x")
	if locked {
		t.Fatal("lock remains")
	}
	if _, err := svc.FinalizeTrade(context.Background(), s.TradeID); !errors.Is(err, ErrTerminalState) {
		t.Fatalf("err=%v", err)
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
}

func TestExpiredTradeCannotRemoveOfferAndReleasesLock(t *testing.T) {
	svc, repo := testService(t)
	s := createTrade(t, svc, "trade-expired-mutation")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	expiredService := NewService(repo, Options{Now: func() time.Time { return testNow.Add(2 * time.Hour) }, NewID: func(prefix string) string { return prefix + "-expired" }, InventoryCapacity: 20})
	if _, err := expiredService.RemoveItem(context.Background(), s.TradeID, "player-a", "item-x", s.Revision); !errors.Is(err, ErrTerminalState) {
		t.Fatalf("err=%v", err)
	}
	locked, _ := expiredService.IsItemLocked(context.Background(), "item-x")
	loaded, _ := expiredService.GetTrade(context.Background(), s.TradeID)
	if locked || loaded.State != StateExpired {
		t.Fatalf("locked=%v state=%s", locked, loaded.State)
	}
}

func TestLockedItemMutationGateRejectsItem(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-mutation")
	if _, err := svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision); err != nil {
		t.Fatal(err)
	}
	if err := svc.ValidateItemMutation(context.Background(), "item-x"); !errors.Is(err, ErrItemLocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestInvalidOwnershipIsRejected(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-owner")
	if _, err := svc.AddItem(context.Background(), s.TradeID, "player-a", "item-y", 1, s.Revision); !errors.Is(err, ErrInvalidOwnership) {
		t.Fatalf("err=%v", err)
	}
}

func TestInjectedSettlementFailureRollsBackBothInventories(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-fail", 10)
	repo.InjectFailure(FailureAfterFirstCharacterWrite, errors.New("synthetic transaction failure"))
	if _, err := svc.FinalizeTrade(context.Background(), "trade-fail"); !errors.Is(err, ErrSettlementFailed) {
		t.Fatalf("err=%v", err)
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
	ownerHas(t, repo, "player-b", "item-y", 5)
	s, _ := svc.GetTrade(context.Background(), "trade-fail")
	if s.State != StateReadyToSettle || s.Settlement != nil {
		t.Fatalf("session=%+v", s)
	}
}

func TestRestartBeforeSettlementRestoresTradeAndLocks(t *testing.T) {
	svc, repo := testService(t)
	s := createTrade(t, svc, "trade-restart-before")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	restarted := NewService(repo, Options{Now: func() time.Time { return testNow }, NewID: func(prefix string) string { return prefix + "-restart" }, InventoryCapacity: 20})
	loaded, err := restarted.GetTrade(context.Background(), s.TradeID)
	if err != nil || loaded.Revision != s.Revision || loaded.State != StateNegotiating {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	locked, _ := restarted.IsItemLocked(context.Background(), "item-x")
	if !locked {
		t.Fatal("persistent lock lost")
	}
}

func TestRestartAfterSettlementKeepsFinalizeIdempotent(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-restart-after", 10)
	first, err := svc.FinalizeTrade(context.Background(), "trade-restart-after")
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewService(repo, Options{Now: func() time.Time { return testNow }, NewID: func(prefix string) string { return prefix + "-restart" }, InventoryCapacity: 20})
	second, err := restarted.FinalizeTrade(context.Background(), "trade-restart-after")
	if err != nil || second != first {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestStaleRevisionConfirmationIsRejected(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-stale")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	if _, err := svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision-1); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("err=%v", err)
	}
}

func TestDuplicateConfirmationIsIdempotent(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-confirm")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	first, err := svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	if err != nil || second.Revision != first.Revision || second.PlayerAConfirmedRevision != first.PlayerAConfirmedRevision {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
}

func TestCancelCompletedTradeIsRejected(t *testing.T) {
	svc, _ := testService(t)
	readyTrade(t, svc, "trade-completed", 10)
	if _, err := svc.FinalizeTrade(context.Background(), "trade-completed"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CancelTrade(context.Background(), "trade-completed", "player-a"); !errors.Is(err, ErrTerminalState) {
		t.Fatalf("err=%v", err)
	}
}

func TestPersistenceReloadPreservesSessionFields(t *testing.T) {
	svc, repo := testService(t)
	s := readyTrade(t, svc, "trade-reload", 3)
	loaded, err := repo.LoadTrade(context.Background(), s.TradeID)
	if err != nil || loaded.TradeID != s.TradeID || loaded.PlayerAID != s.PlayerAID || loaded.PlayerBID != s.PlayerBID || loaded.Revision != s.Revision || loaded.State != s.State || len(loaded.PlayerAOffer) != 1 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
}

func TestPartialStackTransfersOnlyOfferedQuantity(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-stack", 3)
	if _, err := svc.FinalizeTrade(context.Background(), "trade-stack"); err != nil {
		t.Fatal(err)
	}
	ownerHas(t, repo, "player-a", "item-x", 7)
	b, _ := repo.Character(context.Background(), "player-b")
	found := false
	for _, item := range b.Items {
		found = found || (item.DefinitionID == "canonical:item-x" && item.Quantity == 3 && item.InstanceID != "item-x")
	}
	if !found {
		t.Fatalf("split stack missing: %+v", b.Items)
	}
}

func TestInvalidStackQuantitiesAreRejected(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-quantity")
	for _, quantity := range []int{0, -1, 11} {
		if _, err := svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", quantity, s.Revision); !errors.Is(err, ErrInvalidQuantity) {
			t.Fatalf("quantity=%d err=%v", quantity, err)
		}
	}
}

func TestInventoryCapacityFailureLeavesBothSidesUnchanged(t *testing.T) {
	svc, repo := testService(t)
	b, _ := repo.Character(context.Background(), "player-b")
	for i := 1; i < 20; i++ {
		item := testItem(fmt.Sprintf("b-full-%d", i), 1)
		item.SlotIndex = i
		b.Items = append(b.Items, item)
	}
	if err := repo.ReplaceCharacter(b); err != nil {
		t.Fatal(err)
	}
	s := createTrade(t, svc, "trade-capacity")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 3, s.Revision)
	s, _ = svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	s, _ = svc.ConfirmTrade(context.Background(), s.TradeID, "player-b", s.Revision)
	if _, err := svc.FinalizeTrade(context.Background(), s.TradeID); !errors.Is(err, ErrInventoryCapacity) {
		t.Fatalf("err=%v", err)
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
}

func TestDisconnectCancelsNegotiatingTrade(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-disconnect")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	if _, err := svc.HandleDisconnect(context.Background(), s.TradeID, "player-b"); err != nil {
		t.Fatal(err)
	}
	loaded, _ := svc.GetTrade(context.Background(), s.TradeID)
	if loaded.State != StateCancelled {
		t.Fatalf("state=%s", loaded.State)
	}
}

func TestConcurrentFinalizeCompletesExactlyOnce(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-concurrent-finalize", 10)
	var wg sync.WaitGroup
	results := make(chan SettlementResult, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := svc.FinalizeTrade(context.Background(), "trade-concurrent-finalize")
			results <- r
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	var settlement string
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	for result := range results {
		if settlement == "" {
			settlement = result.SettlementID
		}
		if result.SettlementID != settlement {
			t.Fatalf("settlements differ: %s %s", settlement, result.SettlementID)
		}
	}
	ownerHas(t, repo, "player-b", "item-x", 10)
}

func TestRemoveItemAdvancesRevisionUnlocksAndClearsConfirmations(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-remove")
	s, _ = svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
	s, _ = svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	s, err := svc.RemoveItem(context.Background(), s.TradeID, "player-a", "item-x", s.Revision)
	if err != nil || s.Revision != 3 || s.PlayerAConfirmedRevision != 0 {
		t.Fatalf("session=%+v err=%v", s, err)
	}
	locked, _ := svc.IsItemLocked(context.Background(), "item-x")
	if locked {
		t.Fatal("removed item remains locked")
	}
}

func TestAuditEventsRecordTransitionsWithoutSecrets(t *testing.T) {
	svc, repo := testService(t)
	readyTrade(t, svc, "trade-audit", 10)
	if _, err := svc.FinalizeTrade(context.Background(), "trade-audit"); err != nil {
		t.Fatal(err)
	}
	events, err := repo.AuditEvents(context.Background(), "trade-audit")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"trade.created": false, "trade.offer_changed": false, "trade.confirmed": false, "trade.settlement_started": false, "trade.completed": false}
	for _, event := range events {
		if _, ok := want[event.Kind]; ok {
			want[event.Kind] = true
		}
		if event.TradeID == "" || len(event.ParticipantIDs) != 2 {
			t.Fatalf("event=%+v", event)
		}
	}
	for kind, found := range want {
		if !found {
			t.Fatalf("missing %s: %+v", kind, events)
		}
	}
}

func TestAuditSequenceRemainsStableAcrossTransactions(t *testing.T) {
	svc, repo := testService(t)
	s := createTrade(t, svc, "trade-audit-sequence")
	before, _ := repo.AuditEvents(context.Background(), s.TradeID)
	if len(before) != 1 {
		t.Fatalf("before=%+v", before)
	}
	if _, err := svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision); err != nil {
		t.Fatal(err)
	}
	after, _ := repo.AuditEvents(context.Background(), s.TradeID)
	if len(after) != 2 || after[0].Sequence != before[0].Sequence || after[1].Sequence <= after[0].Sequence {
		t.Fatalf("before=%+v after=%+v", before, after)
	}
}

func TestUnsupportedAssetTypeFailsClosed(t *testing.T) {
	svc, _ := testService(t)
	s := createTrade(t, svc, "trade-asset")
	if _, err := svc.AddAsset(context.Background(), s.TradeID, "player-a", Asset{Type: "FB", AssetID: "fake", Quantity: 1}, s.Revision); !errors.Is(err, ErrUnsupportedAsset) {
		t.Fatalf("err=%v", err)
	}
}
