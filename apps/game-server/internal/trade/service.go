package trade

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"sort"
	"time"

	"fractallegend/game-server/internal/persistence"
)

type Options struct {
	Now               func() time.Time
	NewID             func(string) string
	InventoryCapacity int
}

type Service struct {
	repo     Repository
	now      func() time.Time
	newID    func(string) string
	capacity int
}

func NewService(repo Repository, options Options) *Service {
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	if options.NewID == nil {
		options.NewID = secureID
	}
	if options.InventoryCapacity <= 0 {
		options.InventoryCapacity = 20
	}
	return &Service{repo: repo, now: options.Now, newID: options.NewID, capacity: options.InventoryCapacity}
}

func (s *Service) event(session *Session, kind string, before State, outcome string) AuditEvent {
	return AuditEvent{TradeID: session.TradeID, Kind: kind, Revision: session.Revision, PreviousState: before, NewState: session.State, ParticipantIDs: []string{session.PlayerAID, session.PlayerBID}, OccurredAt: s.now(), Outcome: outcome}
}

func (s *Service) CreateTrade(ctx context.Context, id, playerA, playerB string, expiresAt time.Time) (Session, error) {
	now := s.now()
	if s == nil || s.repo == nil || id == "" || playerA == "" || playerB == "" || playerA == playerB || !expiresAt.After(now) {
		return Session{}, ErrInvalidTrade
	}
	value := Session{TradeID: id, PlayerAID: playerA, PlayerBID: playerB, State: StateNegotiating, Revision: 1, CreatedAt: now, UpdatedAt: now, ExpiresAt: expiresAt}
	if err := s.repo.CreateTrade(ctx, value, s.event(&value, "trade.created", "", "CREATED")); err != nil {
		return Session{}, err
	}
	return value, nil
}

func (s *Service) GetTrade(ctx context.Context, id string) (Session, error) {
	return s.repo.LoadTrade(ctx, id)
}

func participant(session *Session, actor string) (*[]OfferItem, *int64, error) {
	switch actor {
	case session.PlayerAID:
		return &session.PlayerAOffer, &session.PlayerAConfirmedRevision, nil
	case session.PlayerBID:
		return &session.PlayerBOffer, &session.PlayerBConfirmedRevision, nil
	default:
		return nil, nil, ErrInvalidParticipant
	}
}

func findItem(character persistence.CharacterAggregate, id string) (persistence.ItemInstance, bool) {
	for _, item := range character.Items {
		if item.InstanceID == id {
			return item, true
		}
	}
	return persistence.ItemInstance{}, false
}

func validateOffer(tx Transaction, session *Session, actor string, offers []OfferItem) error {
	character, err := tx.LoadCharacter(actor)
	if err != nil {
		return ErrInvalidOwnership
	}
	for _, offer := range offers {
		item, ok := findItem(character, offer.ItemInstanceID)
		if !ok || item.Location != persistence.InventoryLocation {
			return ErrInvalidOwnership
		}
		if offer.Quantity < 1 || offer.Quantity > item.Quantity {
			return ErrInvalidQuantity
		}
		if err := tx.ValidateItemLock(ItemLock{TradeID: session.TradeID, OwnerCharacterID: actor, ItemInstanceID: offer.ItemInstanceID, Quantity: offer.Quantity}); err != nil {
			return err
		}
	}
	return nil
}

func expireIfDue(tx Transaction, session *Session, now time.Time, event func(string, State, string) AuditEvent) (bool, error) {
	if session.State.terminal() || now.Before(session.ExpiresAt) {
		return false, nil
	}
	before := session.State
	session.State = StateExpired
	session.UpdatedAt = now
	if err := tx.UnlockAll(); err != nil {
		return false, err
	}
	if err := tx.AppendAudit(event("trade.expired", before, "EXPIRED")); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Service) AddAsset(ctx context.Context, tradeID, actor string, asset Asset, expectedRevision int64) (Session, error) {
	if asset.Type != "ITEM" {
		return Session{}, ErrUnsupportedAsset
	}
	return s.AddItem(ctx, tradeID, actor, asset.AssetID, asset.Quantity, expectedRevision)
}

func (s *Service) AddItem(ctx context.Context, tradeID, actor, itemID string, quantity int, expectedRevision int64) (Session, error) {
	var expired bool
	result, err := s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		if due, e := expireIfDue(tx, session, s.now(), func(k string, b State, o string) AuditEvent { return s.event(session, k, b, o) }); e != nil {
			return e
		} else if due {
			expired = true
			return nil
		}
		if session.State != StateNegotiating && session.State != StateReadyToSettle {
			return ErrTerminalState
		}
		if session.Revision != expectedRevision {
			return ErrStaleRevision
		}
		offers, _, e := participant(session, actor)
		if e != nil {
			return e
		}
		character, e := tx.LoadCharacter(actor)
		if e != nil {
			return ErrInvalidOwnership
		}
		item, ok := findItem(character, itemID)
		if !ok || item.Location != persistence.InventoryLocation {
			return ErrInvalidOwnership
		}
		if quantity < 1 || quantity > item.Quantity {
			return ErrInvalidQuantity
		}
		found := false
		for i := range *offers {
			if (*offers)[i].ItemInstanceID == itemID {
				(*offers)[i].Quantity = quantity
				found = true
				break
			}
		}
		if e = tx.LockItem(ItemLock{TradeID: tradeID, OwnerCharacterID: actor, ItemInstanceID: itemID, Quantity: quantity, CreatedAt: s.now()}); e != nil {
			return e
		}
		if !found {
			*offers = append(*offers, OfferItem{ItemInstanceID: itemID, Quantity: quantity})
		}
		session.Revision++
		session.PlayerAConfirmedRevision = 0
		session.PlayerBConfirmedRevision = 0
		before := session.State
		session.State = StateNegotiating
		session.UpdatedAt = s.now()
		return tx.AppendAudit(s.event(session, "trade.offer_changed", before, "ITEM_ADDED"))
	})
	if err != nil {
		return Session{}, err
	}
	if expired {
		return Session{}, ErrTerminalState
	}
	return result, nil
}

func (s *Service) RemoveItem(ctx context.Context, tradeID, actor, itemID string, expectedRevision int64) (Session, error) {
	var expired bool
	result, err := s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		if due, e := expireIfDue(tx, session, s.now(), func(k string, b State, o string) AuditEvent { return s.event(session, k, b, o) }); e != nil {
			return e
		} else if due {
			expired = true
			return nil
		}
		if session.State != StateNegotiating && session.State != StateReadyToSettle {
			return ErrTerminalState
		}
		if session.Revision != expectedRevision {
			return ErrStaleRevision
		}
		offers, _, e := participant(session, actor)
		if e != nil {
			return e
		}
		index := -1
		for i := range *offers {
			if (*offers)[i].ItemInstanceID == itemID {
				index = i
				break
			}
		}
		if index < 0 {
			return ErrInvalidOwnership
		}
		*offers = append((*offers)[:index], (*offers)[index+1:]...)
		if e = tx.UnlockItem(itemID); e != nil {
			return e
		}
		session.Revision++
		session.PlayerAConfirmedRevision = 0
		session.PlayerBConfirmedRevision = 0
		before := session.State
		session.State = StateNegotiating
		session.UpdatedAt = s.now()
		return tx.AppendAudit(s.event(session, "trade.offer_changed", before, "ITEM_REMOVED"))
	})
	if err == nil && expired {
		return Session{}, ErrTerminalState
	}
	return result, err
}

func (s *Service) ConfirmTrade(ctx context.Context, tradeID, actor string, revision int64) (Session, error) {
	var expired bool
	result, err := s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		if due, e := expireIfDue(tx, session, s.now(), func(k string, b State, o string) AuditEvent { return s.event(session, k, b, o) }); e != nil {
			return e
		} else if due {
			expired = true
			return nil
		}
		if session.State != StateNegotiating && session.State != StateReadyToSettle {
			return ErrTerminalState
		}
		if revision != session.Revision {
			return ErrStaleRevision
		}
		offers, confirmed, e := participant(session, actor)
		if e != nil {
			return e
		}
		if e = validateOffer(tx, session, actor, *offers); e != nil {
			return e
		}
		if *confirmed == revision {
			return nil
		}
		*confirmed = revision
		before := session.State
		if session.PlayerAConfirmedRevision == revision && session.PlayerBConfirmedRevision == revision {
			session.State = StateReadyToSettle
		}
		session.UpdatedAt = s.now()
		return tx.AppendAudit(s.event(session, "trade.confirmed", before, "CONFIRMED"))
	})
	if err != nil {
		return Session{}, err
	}
	if expired {
		return Session{}, ErrTerminalState
	}
	return result, nil
}

func (s *Service) CancelTrade(ctx context.Context, tradeID, actor string) (Session, error) {
	return s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		if _, _, err := participant(session, actor); err != nil {
			return err
		}
		if session.State == StateCancelled {
			return nil
		}
		if session.State.terminal() {
			return ErrTerminalState
		}
		before := session.State
		now := s.now()
		session.State = StateCancelled
		session.CancelledAt = &now
		session.UpdatedAt = now
		if err := tx.UnlockAll(); err != nil {
			return err
		}
		return tx.AppendAudit(s.event(session, "trade.cancelled", before, "CANCELLED"))
	})
}

func (s *Service) HandleDisconnect(ctx context.Context, tradeID, actor string) (Session, error) {
	return s.CancelTrade(ctx, tradeID, actor)
}

func (s *Service) ExpireTrade(ctx context.Context, tradeID string, at time.Time) (Session, error) {
	return s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		if session.State == StateExpired {
			return nil
		}
		if session.State.terminal() {
			return ErrTerminalState
		}
		if at.Before(session.ExpiresAt) {
			return ErrInvalidState
		}
		before := session.State
		session.State = StateExpired
		session.UpdatedAt = at
		if err := tx.UnlockAll(); err != nil {
			return err
		}
		return tx.AppendAudit(s.event(session, "trade.expired", before, "EXPIRED"))
	})
}

func (s *Service) IsItemLocked(ctx context.Context, itemID string) (bool, error) {
	return s.repo.IsItemLocked(ctx, itemID)
}
func (s *Service) ValidateItemMutation(ctx context.Context, itemID string) error {
	locked, err := s.IsItemLocked(ctx, itemID)
	if err != nil {
		return err
	}
	if locked {
		return ErrItemLocked
	}
	return nil
}

func (s *Service) FinalizeTrade(ctx context.Context, tradeID string) (SettlementResult, error) {
	var result SettlementResult
	var attempted Session
	var expired bool
	_, err := s.repo.Transact(ctx, tradeID, func(tx Transaction) error {
		session := tx.Session()
		attempted = cloneSession(*session)
		if session.State == StateCompleted && session.Settlement != nil {
			result = *session.Settlement
			return nil
		}
		if session.State.terminal() {
			return ErrTerminalState
		}
		if due, e := expireIfDue(tx, session, s.now(), func(k string, b State, o string) AuditEvent { return s.event(session, k, b, o) }); e != nil {
			return e
		} else if due {
			expired = true
			return nil
		}
		if session.State != StateReadyToSettle || session.PlayerAConfirmedRevision != session.Revision || session.PlayerBConfirmedRevision != session.Revision {
			return ErrNotReady
		}
		if err := validateOffer(tx, session, session.PlayerAID, session.PlayerAOffer); err != nil {
			return err
		}
		if err := validateOffer(tx, session, session.PlayerBID, session.PlayerBOffer); err != nil {
			return err
		}
		if err := tx.AppendAudit(s.event(session, "trade.settlement_started", session.State, "STARTED")); err != nil {
			return err
		}
		a, err := tx.LoadCharacter(session.PlayerAID)
		if err != nil {
			return err
		}
		b, err := tx.LoadCharacter(session.PlayerBID)
		if err != nil {
			return err
		}
		if err = s.transferOffers(&a, &b, session.PlayerAOffer); err != nil {
			return err
		}
		if err = s.transferOffers(&b, &a, session.PlayerBOffer); err != nil {
			return err
		}
		if len(a.Items) > s.capacity || len(b.Items) > s.capacity {
			return ErrInventoryCapacity
		}
		normalizeSlots(&a)
		normalizeSlots(&b)
		if err = tx.SaveCharacter(a); err != nil {
			return err
		}
		if err = tx.SaveCharacter(b); err != nil {
			return err
		}
		now := s.now()
		result = SettlementResult{TradeID: session.TradeID, SettlementID: s.newID("settlement"), CompletedAt: now}
		before := session.State
		session.State = StateCompleted
		session.CompletedAt = &now
		session.UpdatedAt = now
		session.Settlement = &result
		if err = tx.UnlockAll(); err != nil {
			return err
		}
		return tx.AppendAudit(s.event(session, "trade.completed", before, "COMPLETED"))
	})
	if err != nil {
		classified := settlementError(err)
		if attempted.TradeID != "" && (attempted.State == StateReadyToSettle) {
			_ = s.repo.RecordSettlementFailure(ctx, tradeID, s.event(&attempted, "trade.settlement_failed", attempted.State, "ROLLED_BACK"))
		}
		return SettlementResult{}, classified
	}
	if expired {
		return SettlementResult{}, ErrTerminalState
	}
	return result, nil
}

func (s *Service) transferOffers(source, destination *persistence.CharacterAggregate, offers []OfferItem) error {
	for _, offer := range offers {
		index := -1
		for i := range source.Items {
			if source.Items[i].InstanceID == offer.ItemInstanceID {
				index = i
				break
			}
		}
		if index < 0 {
			return ErrInvalidOwnership
		}
		item := source.Items[index]
		if item.Location != persistence.InventoryLocation {
			return ErrInvalidOwnership
		}
		if offer.Quantity < 1 || offer.Quantity > item.Quantity {
			return ErrInvalidQuantity
		}
		if offer.Quantity == item.Quantity {
			source.Items = append(source.Items[:index], source.Items[index+1:]...)
			item.Quantity = offer.Quantity
			destination.Items = append(destination.Items, item)
		} else {
			source.Items[index].Quantity -= offer.Quantity
			item.InstanceID = s.newID("item")
			item.Quantity = offer.Quantity
			destination.Items = append(destination.Items, item)
		}
	}
	return nil
}

func normalizeSlots(character *persistence.CharacterAggregate) {
	sort.SliceStable(character.Items, func(i, j int) bool { return character.Items[i].SlotIndex < character.Items[j].SlotIndex })
	for index := range character.Items {
		character.Items[index].SlotIndex = index
	}
}

func secureID(prefix string) string {
	value := make([]byte, 16)
	if _, err := cryptorand.Read(value); err != nil {
		panic("secure random unavailable")
	}
	return prefix + "-" + hex.EncodeToString(value)
}
