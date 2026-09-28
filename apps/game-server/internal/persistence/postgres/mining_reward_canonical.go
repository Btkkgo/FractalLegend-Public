package postgres

import (
	"bytes"
	"encoding/json"
	"strconv"
	"time"

	"fractallegend/game-server/internal/miningreward"
)

type quotedInt64 int64

func (v quotedInt64) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.FormatInt(int64(v), 10))
}
func (v *quotedInt64) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || strconv.FormatInt(n, 10) != s {
		return ErrMiningRewardInvariant
	}
	*v = quotedInt64(n)
	return nil
}

const miningRewardMicroLayout = "2006-01-02T15:04:05.000000Z"

type microTime time.Time

func (v microTime) MarshalJSON() ([]byte, error) {
	t := miningreward.CanonicalTime(time.Time(v))
	return json.Marshal(t.Format(miningRewardMicroLayout))
}
func (v *microTime) UnmarshalJSON(raw []byte) error {
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return err
	}
	t, err := time.Parse(miningRewardMicroLayout, s)
	if err != nil || t.Format(miningRewardMicroLayout) != s {
		return ErrMiningRewardInvariant
	}
	*v = microTime(miningreward.CanonicalTime(t))
	return nil
}

type miningRewardPoolWire struct {
	NetCapacity  quotedInt64 `json:"netCapacity"`
	Reserved     quotedInt64 `json:"reserved"`
	Distributed  quotedInt64 `json:"distributed"`
	Remaining    quotedInt64 `json:"remaining"`
	RecoveryDebt quotedInt64 `json:"recoveryDebt"`
	Revision     quotedInt64 `json:"revision"`
}

type miningRewardGrantWire struct {
	PlayerID                string      `json:"playerID"`
	AccountID               string      `json:"accountID"`
	CharacterID             string      `json:"characterID"`
	BeneficiaryBindingID    string      `json:"beneficiaryBindingID"`
	Power                   quotedInt64 `json:"power"`
	Quotient                quotedInt64 `json:"quotient"`
	Remainder               quotedInt64 `json:"remainder"`
	Bonus                   quotedInt64 `json:"remainderBonus"`
	Quantity                quotedInt64 `json:"finalQuantity"`
	IssuanceID              string      `json:"issuanceID"`
	InventoryInstanceID     string      `json:"inventoryInstanceID"`
	InventoryRevisionBefore quotedInt64 `json:"inventoryRevisionBefore"`
	InventoryRevisionAfter  quotedInt64 `json:"inventoryRevisionAfter"`
	ItemRevisionBefore      quotedInt64 `json:"itemRevisionBefore"`
	ItemRevisionAfter       quotedInt64 `json:"itemRevisionAfter"`
}

type miningRewardReceiptWire struct {
	SchemaVersion            string                  `json:"schemaVersion"`
	Status                   string                  `json:"status"`
	SettlementID             string                  `json:"settlementID"`
	CommandID                string                  `json:"commandID"`
	CommandFingerprint       string                  `json:"commandFingerprint"`
	BlockInstanceID          string                  `json:"blockInstanceID"`
	DisplayBlockID           string                  `json:"displayBlockID"`
	ReservationSourceID      string                  `json:"reservationSourceID"`
	ReservationBindingID     string                  `json:"reservationBindingID"`
	ReservationBindingDigest string                  `json:"reservationBindingDigest"`
	SealID                   string                  `json:"sealID"`
	SealDigest               string                  `json:"sealDigest"`
	WindowID                 string                  `json:"windowID"`
	G18RuleVersion           string                  `json:"g18RuleVersion"`
	G20RuleVersion           string                  `json:"g20RuleVersion"`
	AllocationVersion        string                  `json:"allocationRuleVersion"`
	ConversionVersion        string                  `json:"oreConversionRuleVersion"`
	ActivityCount            quotedInt64             `json:"activityCount"`
	ParticipantCount         quotedInt64             `json:"participantCount"`
	PositiveParticipantCount quotedInt64             `json:"positiveParticipantCount"`
	TotalPower               quotedInt64             `json:"totalPower"`
	ReservedAmount           quotedInt64             `json:"reservedAmount"`
	TotalOre                 quotedInt64             `json:"totalDistributedOre"`
	PoolBefore               miningRewardPoolWire    `json:"poolBefore"`
	PoolAfter                miningRewardPoolWire    `json:"poolAfter"`
	Grants                   []miningRewardGrantWire `json:"grants"`
	CreatedAt                microTime               `json:"createdAt"`
}

func miningRewardReceiptToWire(r MiningRewardSettlementReceipt) miningRewardReceiptWire {
	grants := make([]miningRewardGrantWire, 0, len(r.Grants))
	for _, g := range r.Grants {
		grants = append(grants, miningRewardGrantWire{
			PlayerID: g.PlayerID, AccountID: g.AccountID, CharacterID: g.CharacterID, BeneficiaryBindingID: g.BeneficiaryBindingID,
			Power: quotedInt64(g.Power), Quotient: quotedInt64(g.Quotient), Remainder: quotedInt64(g.Remainder),
			Bonus: quotedInt64(g.Bonus), Quantity: quotedInt64(g.Quantity), IssuanceID: g.IssuanceID,
			InventoryInstanceID: g.InventoryInstanceID, InventoryRevisionBefore: quotedInt64(g.InventoryRevisionBefore),
			InventoryRevisionAfter: quotedInt64(g.InventoryRevisionAfter), ItemRevisionBefore: quotedInt64(g.ItemRevisionBefore),
			ItemRevisionAfter: quotedInt64(g.ItemRevisionAfter),
		})
	}
	return miningRewardReceiptWire{
		SchemaVersion: r.SchemaVersion, Status: r.Status, SettlementID: r.SettlementID, CommandID: r.CommandID,
		CommandFingerprint: r.CommandFingerprint, BlockInstanceID: r.BlockInstanceID, DisplayBlockID: r.DisplayBlockID,
		ReservationSourceID: r.ReservationSourceID, ReservationBindingID: r.ReservationBindingID,
		ReservationBindingDigest: r.ReservationBindingDigest, SealID: r.SealID, SealDigest: r.SealDigest,
		WindowID: r.WindowID, G18RuleVersion: r.G18RuleVersion, G20RuleVersion: r.G20RuleVersion,
		AllocationVersion: r.AllocationVersion, ConversionVersion: r.ConversionVersion,
		ActivityCount: quotedInt64(r.ActivityCount), ParticipantCount: quotedInt64(r.ParticipantCount),
		PositiveParticipantCount: quotedInt64(r.PositiveParticipantCount), TotalPower: quotedInt64(r.TotalPower),
		ReservedAmount: quotedInt64(r.ReservedAmount), TotalOre: quotedInt64(r.TotalOre),
		PoolBefore: miningRewardPoolWire{quotedInt64(r.PoolNetCapacityBefore), quotedInt64(r.PoolReservedBefore),
			quotedInt64(r.PoolDistributedBefore), quotedInt64(r.PoolRemainingBefore), quotedInt64(r.PoolDebtBefore), quotedInt64(r.PoolRevisionBefore)},
		PoolAfter: miningRewardPoolWire{quotedInt64(r.PoolNetCapacityAfter), quotedInt64(r.PoolReservedAfter),
			quotedInt64(r.PoolDistributedAfter), quotedInt64(r.PoolRemainingAfter), quotedInt64(r.PoolDebtAfter), quotedInt64(r.PoolRevisionAfter)},
		Grants: grants, CreatedAt: microTime(r.CreatedAt),
	}
}

func canonicalMiningRewardReceipt(r MiningRewardSettlementReceipt) ([]byte, error) {
	return json.Marshal(miningRewardReceiptToWire(r))
}

func decodeCanonicalMiningRewardReceipt(raw []byte) (MiningRewardSettlementReceipt, error) {
	var w miningRewardReceiptWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return MiningRewardSettlementReceipt{}, err
	}
	r := MiningRewardSettlementReceipt{
		SchemaVersion: w.SchemaVersion, Status: w.Status, SettlementID: w.SettlementID, CommandID: w.CommandID,
		CommandFingerprint: w.CommandFingerprint, BlockInstanceID: w.BlockInstanceID, DisplayBlockID: w.DisplayBlockID,
		ReservationSourceID: w.ReservationSourceID, ReservationBindingID: w.ReservationBindingID,
		ReservationBindingDigest: w.ReservationBindingDigest, SealID: w.SealID, SealDigest: w.SealDigest,
		WindowID: w.WindowID, G18RuleVersion: w.G18RuleVersion, G20RuleVersion: w.G20RuleVersion,
		AllocationVersion: w.AllocationVersion, ConversionVersion: w.ConversionVersion,
		ActivityCount: int64(w.ActivityCount), ParticipantCount: int64(w.ParticipantCount),
		PositiveParticipantCount: int64(w.PositiveParticipantCount), TotalPower: int64(w.TotalPower),
		ReservedAmount: int64(w.ReservedAmount), TotalOre: int64(w.TotalOre),
		PoolNetCapacityBefore: int64(w.PoolBefore.NetCapacity), PoolNetCapacityAfter: int64(w.PoolAfter.NetCapacity),
		PoolReservedBefore: int64(w.PoolBefore.Reserved), PoolReservedAfter: int64(w.PoolAfter.Reserved),
		PoolDistributedBefore: int64(w.PoolBefore.Distributed), PoolDistributedAfter: int64(w.PoolAfter.Distributed),
		PoolRemainingBefore: int64(w.PoolBefore.Remaining), PoolRemainingAfter: int64(w.PoolAfter.Remaining),
		PoolDebtBefore: int64(w.PoolBefore.RecoveryDebt), PoolDebtAfter: int64(w.PoolAfter.RecoveryDebt),
		PoolRevisionBefore: int64(w.PoolBefore.Revision), PoolRevisionAfter: int64(w.PoolAfter.Revision),
		Grants: make([]MiningRewardGrant, 0, len(w.Grants)), CreatedAt: time.Time(w.CreatedAt),
	}
	for _, g := range w.Grants {
		r.Grants = append(r.Grants, MiningRewardGrant{
			PlayerID: g.PlayerID, AccountID: g.AccountID, CharacterID: g.CharacterID,
			BeneficiaryBindingID: g.BeneficiaryBindingID, Power: int64(g.Power), Quotient: int64(g.Quotient),
			Remainder: int64(g.Remainder), Bonus: int64(g.Bonus), Quantity: int64(g.Quantity),
			IssuanceID: g.IssuanceID, InventoryInstanceID: g.InventoryInstanceID,
			InventoryRevisionBefore: int64(g.InventoryRevisionBefore), InventoryRevisionAfter: int64(g.InventoryRevisionAfter),
			ItemRevisionBefore: int64(g.ItemRevisionBefore), ItemRevisionAfter: int64(g.ItemRevisionAfter),
		})
	}
	encoded, err := canonicalMiningRewardReceipt(r)
	if err != nil || !bytes.Equal(raw, encoded) {
		return MiningRewardSettlementReceipt{}, ErrMiningRewardInvariant
	}
	return r, nil
}
