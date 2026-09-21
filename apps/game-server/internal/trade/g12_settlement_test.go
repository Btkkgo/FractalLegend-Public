package trade

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fractallegend/game-server/internal/ledger"
)

func g12Service(t *testing.T) (*Service, *MemoryRepository) {
	t.Helper()
	repo := NewMemoryRepository()
	for _, value := range []struct {
		id, item string
		quantity int
	}{{"player-a", "item-x", 10}, {"player-b", "item-y", 5}, {"player-c", "item-z", 1}} {
		if err := repo.SeedCharacter(testAggregate(value.id, testItem(value.item, value.quantity))); err != nil {
			t.Fatal(err)
		}
	}
	var ids atomic.Int64
	svc := NewService(repo, Options{Now: func() time.Time { return testNow }, NewID: func(prefix string) string { return fmt.Sprintf("%s-g12-%d", prefix, ids.Add(1)) }, InventoryCapacity: 20})
	for _, account := range []ledger.LedgerAccount{
		{ID: "fb-player-a", OwnerID: "player-a", OwnerType: ledger.OwnerPlayer, Currency: ledger.CurrencyFB, Balance: 1_000, Revision: 1, CreatedAt: testNow, UpdatedAt: testNow},
		{ID: "fb-player-b", OwnerID: "player-b", OwnerType: ledger.OwnerPlayer, Currency: ledger.CurrencyFB, Balance: 500, Revision: 1, CreatedAt: testNow, UpdatedAt: testNow},
		{ID: "fb-player-c", OwnerID: "player-c", OwnerType: ledger.OwnerPlayer, Currency: ledger.CurrencyFB, Balance: 0, Revision: 1, CreatedAt: testNow, UpdatedAt: testNow},
	} {
		if err := repo.seedLedgerAccount(account); err != nil {
			t.Fatal(err)
		}
	}
	return svc, repo
}

func g12Ready(t *testing.T, svc *Service, id string, aFB, bFB int64, withItems bool) Session {
	t.Helper()
	s := createTrade(t, svc, id)
	var err error
	if withItems {
		s, err = svc.AddItem(context.Background(), id, "player-a", "item-x", 10, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
		s, err = svc.AddItem(context.Background(), id, "player-b", "item-y", 5, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	if aFB != 0 {
		s, err = svc.SetFBOffer(context.Background(), id, "player-a", aFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	if bFB != 0 {
		s, err = svc.SetFBOffer(context.Background(), id, "player-b", bFB, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
	}
	s, err = svc.ConfirmTrade(context.Background(), id, "player-a", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	s, err = svc.ConfirmTrade(context.Background(), id, "player-b", s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func fbBalance(t *testing.T, repo *MemoryRepository, owner string) int64 {
	t.Helper()
	account, err := repo.ledgerAccountByOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	return account.Balance
}

func TestG12FBOfferInvalidatesConfirmationsAndAdvancesRevision(t *testing.T) {
	svc, _ := g12Service(t)
	s := createTrade(t, svc, "g12-revision")
	s, _ = svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision)
	updated, err := svc.SetFBOffer(context.Background(), s.TradeID, "player-b", 25, s.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != s.Revision+1 || updated.PlayerAConfirmedRevision != 0 || updated.PlayerBConfirmedRevision != 0 || updated.PlayerBFBOffer != 25 {
		t.Fatalf("session=%+v", updated)
	}
}

func TestG12RejectsNegativeAndOverflowingFBOffers(t *testing.T) {
	svc, _ := g12Service(t)
	s := createTrade(t, svc, "g12-invalid-offer")
	if _, err := svc.SetFBOffer(context.Background(), s.TradeID, "player-a", -1, s.Revision); !errors.Is(err, ledger.ErrInvalidAmount) {
		t.Fatalf("negative err=%v", err)
	}
	if _, err := svc.SetFBOffer(context.Background(), s.TradeID, "player-a", math.MaxInt64, s.Revision); err != nil {
		t.Fatalf("max int64 should be a valid offer representation: %v", err)
	}
}

func TestG12FBOnlyGrossSettlementHasZeroFee(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-fb-only", 125, 20, false)
	result, err := svc.FinalizeTrade(context.Background(), "g12-fb-only")
	if err != nil {
		t.Fatal(err)
	}
	if got := fbBalance(t, repo, "player-a"); got != 895 {
		t.Fatalf("player-a balance=%d", got)
	}
	if got := fbBalance(t, repo, "player-b"); got != 605 {
		t.Fatalf("player-b balance=%d", got)
	}
	if len(result.LedgerTransactionIDs) != 2 {
		t.Fatalf("ledger ids=%v", result.LedgerTransactionIDs)
	}
	for _, id := range result.LedgerTransactionIDs {
		tx, loadErr := repo.ledgerTransaction(context.Background(), id)
		if loadErr != nil || len(tx.Entries) != 2 || tx.Entries[0].Amount+tx.Entries[1].Amount != 0 {
			t.Fatalf("tx=%+v err=%v", tx, loadErr)
		}
	}
}

func TestG12MixedItemAndFBSettlementIsAtomicAndReceipted(t *testing.T) {
	svc, repo := g12Service(t)
	ready := g12Ready(t, svc, "g12-mixed", 200, 0, true)
	result, err := svc.FinalizeTrade(context.Background(), ready.TradeID)
	if err != nil {
		t.Fatal(err)
	}
	ownerHas(t, repo, "player-a", "item-y", 5)
	ownerHas(t, repo, "player-b", "item-x", 10)
	if fbBalance(t, repo, "player-a") != 800 || fbBalance(t, repo, "player-b") != 700 {
		t.Fatal("unexpected FB balances")
	}
	receipt, err := svc.TradeReceipt(context.Background(), ready.TradeID)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != StateCompleted || receipt.SettlementID != result.SettlementID || receipt.PlayerAFBOffer != 200 || len(receipt.LedgerTransactionIDs) != 1 {
		t.Fatalf("receipt=%+v result=%+v", receipt, result)
	}
}

func TestG12InsufficientFBAtFinalizeRollsBackItemsAndLedger(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-insufficient", 1_000, 0, true)
	account, _ := repo.ledgerAccountByOwner(context.Background(), "player-a")
	account.Balance = 999
	if err := repo.replaceLedgerAccount(account); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeTrade(context.Background(), "g12-insufficient"); !errors.Is(err, ledger.ErrInsufficientBalance) {
		t.Fatalf("err=%v", err)
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
	ownerHas(t, repo, "player-b", "item-y", 5)
	if fbBalance(t, repo, "player-a") != 999 || fbBalance(t, repo, "player-b") != 500 {
		t.Fatal("balances changed after failed settlement")
	}
	loaded, _ := svc.GetTrade(context.Background(), "g12-insufficient")
	if loaded.State != StateReadyToSettle || loaded.Settlement != nil {
		t.Fatalf("trade=%+v", loaded)
	}
}

func TestG12DuplicateFinalizeAndRestartMoveFundsOnce(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-replay", 75, 0, false)
	first, err := svc.FinalizeTrade(context.Background(), "g12-replay")
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewService(repo, Options{Now: func() time.Time { return testNow }, NewID: func(prefix string) string { return prefix + "-should-not-be-used" }, InventoryCapacity: 20})
	second, err := restarted.FinalizeTrade(context.Background(), "g12-replay")
	if err != nil || second.SettlementID != first.SettlementID || len(second.LedgerTransactionIDs) != 1 {
		t.Fatalf("first=%+v second=%+v err=%v", first, second, err)
	}
	if fbBalance(t, repo, "player-a") != 925 || fbBalance(t, repo, "player-b") != 575 {
		t.Fatal("duplicate finalize moved funds twice")
	}
}

func TestG12CancelAndExpireNeverMoveFB(t *testing.T) {
	for _, tc := range []struct {
		name string
		end  func(*Service, string) error
	}{
		{"cancel", func(s *Service, id string) error {
			_, err := s.CancelTrade(context.Background(), id, "player-a")
			return err
		}},
		{"expire", func(s *Service, id string) error {
			_, err := s.ExpireTrade(context.Background(), id, testNow.Add(2*time.Hour))
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := g12Service(t)
			s := createTrade(t, svc, "g12-"+tc.name)
			s, _ = svc.SetFBOffer(context.Background(), s.TradeID, "player-a", 100, s.Revision)
			if err := tc.end(svc, s.TradeID); err != nil {
				t.Fatal(err)
			}
			if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != 500 {
				t.Fatal("terminal transition moved FB")
			}
		})
	}
}

func TestG12FailureAfterLedgerWriteRollsBackCombinedSettlement(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-ledger-failure", 100, 0, true)
	repo.InjectFailure(FailureAfterLedgerWrite, errors.New("synthetic failure after ledger write"))
	if _, err := svc.FinalizeTrade(context.Background(), "g12-ledger-failure"); !errors.Is(err, ErrSettlementFailed) {
		t.Fatalf("err=%v", err)
	}
	ownerHas(t, repo, "player-a", "item-x", 10)
	ownerHas(t, repo, "player-b", "item-y", 5)
	if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != 500 {
		t.Fatal("rollback did not restore FB")
	}
}

func TestG12ConfirmationRejectsInsufficientCurrentBalance(t *testing.T) {
	svc, _ := g12Service(t)
	s := createTrade(t, svc, "g12-confirm-insufficient")
	s, _ = svc.SetFBOffer(context.Background(), s.TradeID, "player-a", 1_001, s.Revision)
	if _, err := svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision); !errors.Is(err, ledger.ErrInsufficientBalance) {
		t.Fatalf("err=%v", err)
	}
}

func TestG12ReceiverOverflowRollsBack(t *testing.T) {
	svc, repo := g12Service(t)
	account, _ := repo.ledgerAccountByOwner(context.Background(), "player-b")
	account.Balance = math.MaxInt64
	if err := repo.replaceLedgerAccount(account); err != nil {
		t.Fatal(err)
	}
	g12Ready(t, svc, "g12-overflow", 1, 0, false)
	if _, err := svc.FinalizeTrade(context.Background(), "g12-overflow"); !errors.Is(err, ledger.ErrAmountOverflow) {
		t.Fatalf("err=%v", err)
	}
	if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != math.MaxInt64 {
		t.Fatal("overflow leaked mutation")
	}
}

func TestG12ConcurrentTradesCannotDoubleSpendFB(t *testing.T) {
	svc, repo := g12Service(t)
	ready := func(id, counterparty string) {
		s, err := svc.CreateTrade(context.Background(), id, "player-a", counterparty, testNow.Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		s, err = svc.SetFBOffer(context.Background(), id, "player-a", 1_000, s.Revision)
		if err != nil {
			t.Fatal(err)
		}
		s, err = svc.ConfirmTrade(context.Background(), id, "player-a", s.Revision)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = svc.ConfirmTrade(context.Background(), id, counterparty, s.Revision); err != nil {
			t.Fatal(err)
		}
	}
	ready("g12-double-spend-1", "player-b")
	ready("g12-double-spend-2", "player-c")
	var successes atomic.Int64
	var insufficient atomic.Int64
	var wg sync.WaitGroup
	for _, id := range []string{"g12-double-spend-1", "g12-double-spend-2"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.FinalizeTrade(context.Background(), id)
			if err == nil {
				successes.Add(1)
			} else if errors.Is(err, ledger.ErrInsufficientBalance) {
				insufficient.Add(1)
			} else {
				t.Errorf("unexpected err=%v", err)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 || insufficient.Load() != 1 || fbBalance(t, repo, "player-a") != 0 {
		t.Fatalf("success=%d insufficient=%d balance=%d", successes.Load(), insufficient.Load(), fbBalance(t, repo, "player-a"))
	}
}

func TestG12AllSettlementFailureGatesRollback(t *testing.T) {
	for _, point := range []string{FailureAfterItemOwnershipUpdate, FailureDuringBalanceUpdate, FailureAfterLedgerWrite, FailureBeforeCompletedState, FailureBeforeCommit} {
		t.Run(point, func(t *testing.T) {
			svc, repo := g12Service(t)
			id := "g12-failure-" + point
			g12Ready(t, svc, id, 100, 0, true)
			repo.InjectFailure(point, errors.New("injected "+point))
			if _, err := svc.FinalizeTrade(context.Background(), id); !errors.Is(err, ErrSettlementFailed) {
				t.Fatalf("err=%v", err)
			}
			ownerHas(t, repo, "player-a", "item-x", 10)
			ownerHas(t, repo, "player-b", "item-y", 5)
			if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != 500 {
				t.Fatal("FB mutation escaped rollback")
			}
			loaded, _ := svc.GetTrade(context.Background(), id)
			if loaded.State != StateReadyToSettle || loaded.Settlement != nil {
				t.Fatalf("trade=%+v", loaded)
			}
		})
	}
}

func TestG12GrossLedgerReferencesAreStableAndUnique(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-reference", 40, 10, false)
	result, err := svc.FinalizeTrade(context.Background(), "g12-reference")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"g12-reference:A_TO_B": false, "g12-reference:B_TO_A": false}
	for _, id := range result.LedgerTransactionIDs {
		tx, _ := repo.ledgerTransaction(context.Background(), id)
		if tx.Reference.Type != "PLAYER_TRADE" {
			t.Fatalf("reference=%+v", tx.Reference)
		}
		if _, ok := want[tx.Reference.ID]; !ok {
			t.Fatalf("reference=%+v", tx.Reference)
		}
		want[tx.Reference.ID] = true
	}
	for key, seen := range want {
		if !seen {
			t.Fatalf("missing reference %s", key)
		}
	}
}

func TestG12PropertyConservesFBItemsAndFailedTrades(t *testing.T) {
	rng := rand.New(rand.NewSource(12))
	for iteration := 0; iteration < 200; iteration++ {
		svc, repo := g12Service(t)
		aOffer := int64(rng.Intn(1_001))
		bOffer := int64(rng.Intn(501))
		withItems := rng.Intn(2) == 1
		id := fmt.Sprintf("g12-property-%d", iteration)
		g12Ready(t, svc, id, aOffer, bOffer, withItems)
		initialTotal := int64(1_500)
		result, err := svc.FinalizeTrade(context.Background(), id)
		if err != nil {
			t.Fatalf("iteration=%d err=%v", iteration, err)
		}
		if result.TradeID != id || fbBalance(t, repo, "player-a") < 0 || fbBalance(t, repo, "player-b") < 0 || fbBalance(t, repo, "player-a")+fbBalance(t, repo, "player-b") != initialTotal {
			t.Fatalf("iteration=%d conservation failed", iteration)
		}
		if withItems {
			ownerHas(t, repo, "player-a", "item-y", 5)
			ownerHas(t, repo, "player-b", "item-x", 10)
		}
		if second, secondErr := svc.FinalizeTrade(context.Background(), id); secondErr != nil || second.SettlementID != result.SettlementID {
			t.Fatalf("iteration=%d replay=%+v err=%v", iteration, second, secondErr)
		}
	}
}

func TestG12HundredConcurrentSettlementsConserveSupply(t *testing.T) {
	svc, repo := g12Service(t)
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("g12-concurrent-%03d", i)
		g12Ready(t, svc, id, 10, 0, false)
	}
	var successes atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("g12-concurrent-%03d", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.FinalizeTrade(context.Background(), id); err != nil {
				t.Errorf("%s: %v", id, err)
			} else {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 100 || fbBalance(t, repo, "player-a") != 0 || fbBalance(t, repo, "player-b") != 1_500 {
		t.Fatalf("success=%d a=%d b=%d", successes.Load(), fbBalance(t, repo, "player-a"), fbBalance(t, repo, "player-b"))
	}
}

func TestG12ProductionServiceExportsNoClientForgedCompletionOrBalanceMutation(t *testing.T) {
	forbidden := map[string]bool{"SetState": true, "SetCompleted": true, "SetBalance": true, "CreditFB": true, "DebitFB": true, "ApplyClientSettlement": true}
	typeOf := reflect.TypeOf(&Service{})
	for i := 0; i < typeOf.NumMethod(); i++ {
		if forbidden[typeOf.Method(i).Name] {
			t.Fatalf("forbidden production method %s", typeOf.Method(i).Name)
		}
	}
}

func TestG12ItemForItemSettlementRemainsCompatible(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-item-only", 0, 0, true)
	result, err := svc.FinalizeTrade(context.Background(), "g12-item-only")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.LedgerTransactionIDs) != 0 {
		t.Fatalf("ledger ids=%v", result.LedgerTransactionIDs)
	}
	ownerHas(t, repo, "player-a", "item-y", 5)
	ownerHas(t, repo, "player-b", "item-x", 10)
}

func TestG12BothSidesItemAndFBSettleGross(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-bilateral-mixed", 125, 20, true)
	result, err := svc.FinalizeTrade(context.Background(), "g12-bilateral-mixed")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.LedgerTransactionIDs) != 2 || fbBalance(t, repo, "player-a") != 895 || fbBalance(t, repo, "player-b") != 605 {
		t.Fatalf("result=%+v", result)
	}
	ownerHas(t, repo, "player-a", "item-y", 5)
	ownerHas(t, repo, "player-b", "item-x", 10)
}

func TestG12StaleRevisionCannotConfirmFBOffer(t *testing.T) {
	svc, _ := g12Service(t)
	s := createTrade(t, svc, "g12-stale-revision")
	s, _ = svc.SetFBOffer(context.Background(), s.TradeID, "player-a", 10, s.Revision)
	if _, err := svc.ConfirmTrade(context.Background(), s.TradeID, "player-a", s.Revision-1); !errors.Is(err, ErrStaleRevision) {
		t.Fatalf("err=%v", err)
	}
}

func TestG12OwnershipChangeBeforeFinalizeRollsBackFB(t *testing.T) {
	svc, repo := g12Service(t)
	g12Ready(t, svc, "g12-owner-change", 100, 0, true)
	a, _ := repo.Character(context.Background(), "player-a")
	a.Items = nil
	if err := repo.ReplaceCharacter(a); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.FinalizeTrade(context.Background(), "g12-owner-change"); !errors.Is(err, ErrInvalidOwnership) {
		t.Fatalf("err=%v", err)
	}
	if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != 500 {
		t.Fatal("FB changed on invalid ownership")
	}
}

func TestG12PersistentItemLockRejectsSecondTrade(t *testing.T) {
	svc, _ := g12Service(t)
	first := createTrade(t, svc, "g12-item-lock-1")
	second := createTrade(t, svc, "g12-item-lock-2")
	if _, err := svc.AddItem(context.Background(), first.TradeID, "player-a", "item-x", 1, first.Revision); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddItem(context.Background(), second.TradeID, "player-a", "item-x", 1, second.Revision); !errors.Is(err, ErrItemLocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestG12ConcurrentItemOffersHaveExactlyOneLockWinner(t *testing.T) {
	svc, _ := g12Service(t)
	first := createTrade(t, svc, "g12-item-race-1")
	second := createTrade(t, svc, "g12-item-race-2")
	var success, locked atomic.Int64
	var wg sync.WaitGroup
	for _, s := range []Session{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := svc.AddItem(context.Background(), s.TradeID, "player-a", "item-x", 1, s.Revision)
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, ErrItemLocked) {
				locked.Add(1)
			} else {
				t.Errorf("err=%v", err)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || locked.Load() != 1 {
		t.Fatalf("success=%d locked=%d", success.Load(), locked.Load())
	}
}

func TestG12SelfTradeIsRejected(t *testing.T) {
	svc, _ := g12Service(t)
	if _, err := svc.CreateTrade(context.Background(), "g12-self", "player-a", "player-a", testNow.Add(time.Hour)); !errors.Is(err, ErrInvalidTrade) {
		t.Fatalf("err=%v", err)
	}
}

func TestG12ExpiredTradeStateSurvivesServiceRestart(t *testing.T) {
	svc, repo := g12Service(t)
	s := createTrade(t, svc, "g12-expiry-restart")
	s, _ = svc.SetFBOffer(context.Background(), s.TradeID, "player-a", 100, s.Revision)
	if _, err := svc.ExpireTrade(context.Background(), s.TradeID, testNow.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	restarted := NewService(repo, Options{Now: func() time.Time { return testNow.Add(2 * time.Hour) }})
	loaded, err := restarted.GetTrade(context.Background(), s.TradeID)
	if err != nil || loaded.State != StateExpired {
		t.Fatalf("trade=%+v err=%v", loaded, err)
	}
	if _, err = restarted.FinalizeTrade(context.Background(), s.TradeID); !errors.Is(err, ErrTerminalState) {
		t.Fatalf("err=%v", err)
	}
}

func TestG12DisconnectCancelsWithoutFBMovement(t *testing.T) {
	svc, repo := g12Service(t)
	s := createTrade(t, svc, "g12-disconnect")
	s, _ = svc.SetFBOffer(context.Background(), s.TradeID, "player-a", 100, s.Revision)
	if _, err := svc.HandleDisconnect(context.Background(), s.TradeID, "player-b"); err != nil {
		t.Fatal(err)
	}
	if fbBalance(t, repo, "player-a") != 1_000 || fbBalance(t, repo, "player-b") != 500 {
		t.Fatal("disconnect moved FB")
	}
}

func TestG12ReceiptUnavailableBeforeCompletion(t *testing.T) {
	svc, _ := g12Service(t)
	s := createTrade(t, svc, "g12-receipt-pending")
	if _, err := svc.TradeReceipt(context.Background(), s.TradeID); !errors.Is(err, ErrNotReady) {
		t.Fatalf("err=%v", err)
	}
}
