package postgres

import (
	"encoding/json"
	"runtime"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/miningreward"
)

func TestG21NativeAllocationReceiptVector(t *testing.T) {
	weights := []miningpower.SealedWeight{
		{CharacterID: "C", PlayerID: "C", AccountID: "account-C", Power: 3},
		{CharacterID: "B", PlayerID: "B", AccountID: "account-B", Power: 2},
		{CharacterID: "A", PlayerID: "A", AccountID: "account-A", Power: 1},
	}
	allocation, err := miningreward.AllocateTEST(10, weights)
	if err != nil || allocation.Total != 10 || len(allocation.Grants) != 3 {
		t.Fatalf("native allocation=%+v %v", allocation, err)
	}
	receipt := MiningRewardSettlementReceipt{
		SchemaVersion: "G21_V1", Status: "COMPLETED", SettlementID: "native-settlement",
		CommandID: "native-command", CommandFingerprint: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		BlockInstanceID: "mining-block-instance-11111111111111111111111111111111",
		DisplayBlockID:  "native-block", ReservationSourceID: "native-reservation",
		ReservationBindingID: "native-binding", ReservationBindingDigest: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		SealID: "native-seal", WindowID: "mining-block-instance-11111111111111111111111111111111",
		G18RuleVersion: "TEST_G18_V1", G20RuleVersion: miningpower.DevelopmentRuleVersion,
		SealDigest:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AllocationVersion: settlementAllocationVersion, ConversionVersion: settlementConversionVersion,
		ActivityCount: 3, ParticipantCount: 3, PositiveParticipantCount: 3, TotalPower: 6, ReservedAmount: 10,
		PoolNetCapacityBefore: 20, PoolNetCapacityAfter: 20,
		TotalOre: 10, PoolReservedBefore: 10, PoolReservedAfter: 0,
		PoolDistributedBefore: 0, PoolDistributedAfter: 10,
		PoolRemainingBefore: 10, PoolRemainingAfter: 10,
		PoolRevisionBefore: 2, PoolRevisionAfter: 3,
		Grants:    make([]MiningRewardGrant, 0, 3),
		CreatedAt: time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC),
	}
	var quantities []int64
	var order []string
	var sortedWeights []int64
	var remainders []int64
	var bonuses []int64
	for _, g := range allocation.Grants {
		receipt.Grants = append(receipt.Grants, MiningRewardGrant{
			PlayerID: g.PlayerID, AccountID: g.AccountID, CharacterID: g.CharacterID,
			BeneficiaryBindingID: "binding-" + g.CharacterID,
			Power:                g.Power, Quotient: g.Quotient, Remainder: g.Remainder,
			Bonus: g.Bonus, Quantity: g.Quantity,
			IssuanceID: "lot-" + g.CharacterID, InventoryInstanceID: "item-" + g.CharacterID,
			InventoryRevisionBefore: 1, InventoryRevisionAfter: 2,
			ItemRevisionAfter: 1,
		})
		quantities = append(quantities, g.Quantity)
		order = append(order, g.CharacterID)
		sortedWeights = append(sortedWeights, g.Power)
		remainders = append(remainders, g.Remainder)
		bonuses = append(bonuses, g.Bonus)
	}
	if quantities[0] != 2 || quantities[1] != 3 || quantities[2] != 5 ||
		order[0] != "A" || order[1] != "B" || order[2] != "C" || allocation.Grants[0].Bonus != 1 {
		t.Fatalf("native ranking=%+v", allocation.Grants)
	}
	raw, err := canonicalMiningRewardReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	digest := rewardHash(raw)
	const expectedReceiptDigest = "cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78"
	if digest != expectedReceiptDigest {
		t.Fatalf("native receipt digest=%s want=%s", digest, expectedReceiptDigest)
	}
	result := struct {
		Version, RemainderWinner, ReceiptDigest string
		Order                                   []string
		Weights, Remainders, Bonuses            []int64
		Quantities                              []int64
		Total                                   int64
		CanonicalReceipt                        json.RawMessage
	}{"G21_NATIVE_ALLOCATION_V2", "A", digest, order, sortedWeights, remainders, bonuses,
		quantities, allocation.Total, json.RawMessage(raw)}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("G21_NATIVE_JSON %s", encoded)
	t.Logf("G21_NATIVE_RUNTIME_JSON {\"goos\":%q,\"goarch\":%q}", runtime.GOOS, runtime.GOARCH)
}
