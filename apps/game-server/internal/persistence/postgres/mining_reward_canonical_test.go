package postgres

import (
	"bytes"
	"testing"
	"time"
)

func TestG21CanonicalReceiptQuotesIntegersAndUsesFixedUTCPrecision(t *testing.T) {
	receipt := MiningRewardSettlementReceipt{
		SchemaVersion: "G21_V1", Status: "COMPLETED", SettlementID: "settlement-fixed",
		CommandID: "command-fixed", BlockInstanceID: "mining-block-instance-11111111111111111111111111111111",
		ReservationSourceID: "reservation-fixed", SealID: "seal-fixed",
		SealDigest:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		AllocationVersion: settlementAllocationVersion, ConversionVersion: settlementConversionVersion,
		TotalOre: 10, PoolReservedBefore: 10, PoolReservedAfter: 0,
		PoolDistributedBefore: 0, PoolDistributedAfter: 10,
		PoolRemainingBefore: 10, PoolRemainingAfter: 10,
		PoolRevisionBefore: 2, PoolRevisionAfter: 3,
		Grants:    []MiningRewardGrant{{CharacterID: "A", Power: 1, Quantity: 10}},
		CreatedAt: time.Date(2026, 9, 28, 2, 3, 4, 5000, time.FixedZone("+8", 8*3600)),
	}
	raw, err := canonicalMiningRewardReceipt(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range [][]byte{[]byte(`"totalDistributedOre":"10"`), []byte(`"finalQuantity":"10"`), []byte(`"createdAt":"2026-09-27T18:03:04.000005Z"`)} {
		if !bytes.Contains(raw, needle) {
			t.Fatalf("missing canonical field %q in %s", needle, raw)
		}
	}
	decoded, err := decodeCanonicalMiningRewardReceipt(raw)
	if err != nil || decoded.TotalOre != receipt.TotalOre || decoded.Grants[0].Quantity != 10 {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
}
