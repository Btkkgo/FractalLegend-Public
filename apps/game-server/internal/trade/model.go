package trade

import (
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("trade not found")
	ErrInvalidTrade       = errors.New("invalid trade")
	ErrInvalidParticipant = errors.New("invalid trade participant")
	ErrInvalidState       = errors.New("invalid trade state")
	ErrTerminalState      = errors.New("trade is in a terminal state")
	ErrStaleRevision      = errors.New("stale trade revision")
	ErrInvalidOwnership   = errors.New("item ownership is invalid")
	ErrInvalidQuantity    = errors.New("item quantity is invalid")
	ErrItemLocked         = errors.New("item is locked by another trade")
	ErrNotReady           = errors.New("trade is not ready to settle")
	ErrInventoryCapacity  = errors.New("inventory capacity exceeded")
	ErrSettlementFailed   = errors.New("trade settlement failed")
	ErrUnsupportedAsset   = errors.New("unsupported trade asset")
	ErrRetryExhausted     = errors.New("trade settlement retry exhausted")
)

type State string

const (
	StateNegotiating   State = "NEGOTIATING"
	StateReadyToSettle State = "READY_TO_SETTLE"
	StateCompleted     State = "COMPLETED"
	StateCancelled     State = "CANCELLED"
	StateExpired       State = "EXPIRED"
)

func (s State) terminal() bool {
	return s == StateCompleted || s == StateCancelled || s == StateExpired
}

type Asset struct {
	Type     string
	AssetID  string
	Quantity int
}

type OfferItem struct {
	ItemInstanceID string
	Quantity       int
}

type SettlementResult struct {
	TradeID              string
	SettlementID         string
	LedgerTransactionIDs []string
	CompletedAt          time.Time
}

type Session struct {
	TradeID                  string
	PlayerAID                string
	PlayerBID                string
	State                    State
	Revision                 int64
	PlayerAOffer             []OfferItem
	PlayerBOffer             []OfferItem
	PlayerAFBOffer           int64
	PlayerBFBOffer           int64
	PlayerAConfirmedRevision int64
	PlayerBConfirmedRevision int64
	CreatedAt                time.Time
	UpdatedAt                time.Time
	ExpiresAt                time.Time
	CompletedAt              *time.Time
	CancelledAt              *time.Time
	Settlement               *SettlementResult
}

type ItemLock struct {
	TradeID, OwnerCharacterID, ItemInstanceID string
	Quantity                                  int
	CreatedAt                                 time.Time
}

type AuditEvent struct {
	Sequence             int64
	TradeID              string
	Kind                 string
	Revision             int64
	PreviousState        State
	NewState             State
	ParticipantIDs       []string
	OccurredAt           time.Time
	Outcome              string
	LedgerTransactionIDs []string
}

type TradeReceipt struct {
	TradeID              string
	SettlementID         string
	PlayerAID            string
	PlayerBID            string
	PlayerAOffer         []OfferItem
	PlayerBOffer         []OfferItem
	PlayerAFBOffer       int64
	PlayerBFBOffer       int64
	Revision             int64
	Status               State
	LedgerTransactionIDs []string
	CompletedAt          time.Time
}

func cloneSession(value Session) Session {
	value.PlayerAOffer = append([]OfferItem(nil), value.PlayerAOffer...)
	value.PlayerBOffer = append([]OfferItem(nil), value.PlayerBOffer...)
	if value.CompletedAt != nil {
		v := *value.CompletedAt
		value.CompletedAt = &v
	}
	if value.CancelledAt != nil {
		v := *value.CancelledAt
		value.CancelledAt = &v
	}
	if value.Settlement != nil {
		v := *value.Settlement
		v.LedgerTransactionIDs = append([]string(nil), value.Settlement.LedgerTransactionIDs...)
		value.Settlement = &v
	}
	return value
}
