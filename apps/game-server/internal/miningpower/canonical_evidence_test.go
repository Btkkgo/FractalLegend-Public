package miningpower

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestG21P0CanonicalGoldenVectors(t *testing.T) {
	stamp := time.Date(2026, 9, 28, 1, 2, 3, 456789000, time.UTC)
	beneficiary, err := BeneficiaryEvidenceDigest(BeneficiaryEvidence{
		Version: "G21_P0_BENEFICIARY_V1", ActivityID: "activity-a", SourceEventID: "source-a",
		AccountID: "account-a", PlayerID: "player-a", CharacterID: "character-a",
		BlockInstanceID: "mining-block-instance-11111111111111111111111111111111",
		AuthoritySource: "CHARACTERS_OWNER_ROW_V1", BoundAt: stamp})
	if err != nil {
		t.Fatal(err)
	}
	issuance, err := IssuanceEvidenceDigest(IssuanceEvidence{
		RuleVersion: "G21_P0_ISSUANCE_V1", SourceType: "MINING_REWARD", SourceKey: "reward-source-a",
		BlockInstanceID: "mining-block-instance-11111111111111111111111111111111",
		CharacterID:     "character-a", MaterialDefinitionID: "g19-reviewed-definition", Quantity: 7, CreatedAt: stamp})
	if err != nil {
		t.Fatal(err)
	}
	identities := []SealedActivityIdentity{{ActivityID: "activity-a", SourceEventID: "source-a"}, {ActivityID: "activity-b", SourceEventID: "source-b"}}
	activityDigest, err := AcceptedActivityIdentityDigest(identities)
	if err != nil {
		t.Fatal(err)
	}
	bindingDigest, err := BeneficiaryBindingOrderDigest([]string{beneficiary, beneficiary})
	if err != nil {
		t.Fatal(err)
	}
	seal := SettlementInputSeal{SchemaVersion: SealSchemaVersion, SealRuleVersion: SealRuleVersion,
		SealID: "seal-a", BlockInstanceID: "mining-block-instance-11111111111111111111111111111111",
		DisplayBlockID: "display-a", G18ReservationSourceID: "reservation-a", G18ReservationEvidenceDigest: "evidence-a",
		G18RuleVersion: "DEV_G18_FIXED_BLOCK_REWARD", G20RuleVersion: DevelopmentRuleVersion,
		WindowIdentity:  "mining-block-instance-11111111111111111111111111111111",
		WindowStartedAt: stamp, WindowEndedAt: stamp.Add(time.Minute), SealedAt: stamp.Add(2 * time.Minute),
		ActivityCount: 2, ParticipantCount: 2, TotalValidMiningPower: 300,
		ParticipantWeights: []SealedWeight{{PlayerID: "player-a", AccountID: "account-a", CharacterID: "character-a", Power: 100, ActivityCount: 1},
			{PlayerID: "player-b", AccountID: "account-b", CharacterID: "character-b", Power: 200, ActivityCount: 1}},
		AcceptedActivityIdentities: identities, AcceptedActivityIdentityDigest: activityDigest,
		BeneficiaryBindingDigest: bindingDigest, Eligibility: "SETTLEMENT_INPUT"}
	raw, err := CanonicalSealBytes(seal)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("empty seal")
	}
	sealDigest := DigestSealBytes(raw)
	for _, tc := range []struct{ name, got, want string }{
		{"beneficiary", beneficiary, "84c85cdbe0694882f70f25812ba16d8307d12610bcfca610f7708edcdd5c9418"},
		{"issuance", issuance, "416b5c749130efdcd7a92b596629618bbd752e4a738f6184318adb41b65608eb"},
		{"activity-order", seal.AcceptedActivityIdentityDigest, "832750c10b329dbd6a52b71c5c45254dcb7f74b332a5119f3a5e5f0ed148ac95"},
		{"binding-order", seal.BeneficiaryBindingDigest, "97470facba967d0f5904240d2232953f62b9f8c54d48586b0671ff83fc7743fc"},
		{"seal", sealDigest, "511c1e434158b334f58c182358ff3ebedbfb4ea4bba9fb5e776c4714602415b3"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s=%s want=%s", tc.name, tc.got, tc.want)
		}
	}
	otherZone := time.FixedZone("east", 8*3600)
	east := stamp.In(otherZone)
	localized, err := BeneficiaryEvidenceDigest(BeneficiaryEvidence{Version: "G21_P0_BENEFICIARY_V1", ActivityID: "activity-a", SourceEventID: "source-a",
		AccountID: "account-a", PlayerID: "player-a", CharacterID: "character-a",
		BlockInstanceID: "mining-block-instance-11111111111111111111111111111111", AuthoritySource: "CHARACTERS_OWNER_ROW_V1", BoundAt: east})
	if err != nil || localized != beneficiary {
		t.Fatalf("local timezone changed digest: %s %v", localized, err)
	}
}

// The same versioned fixtures run on Windows, Linux and macOS. They call the
// production canonical digest, ordering, and seal functions without a DB.
func TestG21P0CanonicalNativeFixtures(t *testing.T) {
	const version = "G21_P0_NATIVE_FIXTURES_V1"
	stamp := time.Date(2026, 9, 28, 1, 2, 3, 456789000, time.UTC)
	instanceID := "mining-block-instance-11111111111111111111111111111111"
	reservation, err := ReservationBindingEvidenceDigest(ReservationBindingEvidence{
		Version: ReservationBindingVersion, InstanceID: instanceID, SourceID: "reservation-source-a",
		ReceiptID: "reservation-receipt-a", BlockID: "display-a", RuleVersion: "DEV_G18_FIXED_BLOCK_REWARD", Amount: 5000,
	})
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		Count                                                                             int `json:"count"`
		Seal, Beneficiary, Activity, Reservation, Issuance, FirstCharacter, LastCharacter string
	}
	results := make([]result, 0, 6)
	for _, count := range []int{2, 10, 50, 100, 500, 501} {
		t.Run(fmt.Sprintf("participants_%d", count), func(t *testing.T) {
			weights := make([]SealedWeight, 0, count)
			identities := make([]SealedActivityIdentity, 0, count)
			bindingDigests := make([]string, 0, count)
			for i := count - 1; i >= 0; i-- { // Deliberately unordered input.
				id := fmt.Sprintf("%04d", i)
				weights = append(weights, SealedWeight{PlayerID: "player-" + id, AccountID: "account-" + id,
					CharacterID: "character-" + id, Power: 3, ActivityCount: 1})
			}
			SortSealedWeights(weights)
			for i := 0; i < count; i++ {
				id := fmt.Sprintf("%04d", i)
				identities = append(identities, SealedActivityIdentity{ActivityID: "activity-" + id, SourceEventID: "source-" + id})
				binding, e := BeneficiaryEvidenceDigest(BeneficiaryEvidence{Version: "G21_P0_BENEFICIARY_V1",
					ActivityID: "activity-" + id, SourceEventID: "source-" + id, AccountID: "account-" + id,
					PlayerID: "player-" + id, CharacterID: "character-" + id, BlockInstanceID: instanceID,
					AuthoritySource: "CHARACTERS_OWNER_ROW_V1", BoundAt: stamp})
				if e != nil {
					t.Fatal(e)
				}
				bindingDigests = append(bindingDigests, binding)
			}
			activity, e := AcceptedActivityIdentityDigest(identities)
			if e != nil {
				t.Fatal(e)
			}
			beneficiary, e := BeneficiaryBindingOrderDigest(bindingDigests)
			if e != nil {
				t.Fatal(e)
			}
			issuanceEvidence := IssuanceEvidence{RuleVersion: "G21_P0_ISSUANCE_V1",
				SourceType: "MINING_REWARD", SourceKey: fmt.Sprintf("reward-%04d", count),
				BlockInstanceID: instanceID, CharacterID: "character-0000",
				MaterialDefinitionID: "g19-reviewed-definition", Quantity: int32(count), CreatedAt: stamp}
			issuance, e := IssuanceEvidenceDigest(issuanceEvidence)
			if e != nil {
				t.Fatal(e)
			}
			issuanceEvidence.CreatedAt = stamp.In(time.FixedZone("east", 8*3600))
			localizedIssuance, e := IssuanceEvidenceDigest(issuanceEvidence)
			if e != nil || localizedIssuance != issuance {
				t.Fatalf("issuance timezone changed digest: %s %v", localizedIssuance, e)
			}
			seal := SettlementInputSeal{SchemaVersion: SealSchemaVersion, SealRuleVersion: SealRuleVersion,
				SealID: fmt.Sprintf("seal-%04d", count), BlockInstanceID: instanceID, DisplayBlockID: "display-a",
				G18ReservationSourceID: "reservation-source-a", G18ReservationEvidenceDigest: reservation,
				G18RuleVersion: "DEV_G18_FIXED_BLOCK_REWARD", G20RuleVersion: DevelopmentRuleVersion,
				WindowIdentity: instanceID, WindowStartedAt: stamp, WindowEndedAt: stamp.Add(time.Minute),
				SealedAt: stamp.Add(2 * time.Minute), ActivityCount: count, ParticipantCount: count,
				TotalValidMiningPower: int64(3 * count), ParticipantWeights: weights,
				AcceptedActivityIdentities: identities, AcceptedActivityIdentityDigest: activity,
				BeneficiaryBindingDigest: beneficiary, Eligibility: "SETTLEMENT_INPUT"}
			raw, e := CanonicalSealBytes(seal)
			if count == 501 {
				if !errors.Is(e, ErrInvariant) || raw != nil {
					t.Fatalf("501 rejection: error=%v bytes=%q", e, raw)
				}
				results = append(results, result{Count: count, Seal: "REJECTED_INVARIANT", Reservation: reservation})
				return
			}
			if e != nil {
				t.Fatal(e)
			}
			if weights[0].CharacterID != "character-0000" || weights[len(weights)-1].CharacterID != fmt.Sprintf("character-%04d", count-1) {
				t.Fatalf("participant ordering: first=%s last=%s", weights[0].CharacterID, weights[len(weights)-1].CharacterID)
			}
			east := time.FixedZone("east", 8*3600)
			localized := seal
			localized.SealedAt = seal.SealedAt.In(east)
			if _, e := CanonicalSealBytes(localized); !errors.Is(e, ErrInvariant) {
				t.Fatalf("noncanonical timezone accepted: %v", e)
			}
			results = append(results, result{Count: count, Seal: DigestSealBytes(raw), Beneficiary: beneficiary,
				Activity: activity, Reservation: reservation, Issuance: issuance,
				FirstCharacter: weights[0].CharacterID, LastCharacter: weights[len(weights)-1].CharacterID})
		})
	}
	canonical, err := json.Marshal(struct {
		Version string   `json:"version"`
		Results []result `json:"results"`
	}{version, results})
	if err != nil {
		t.Fatal(err)
	}
	// The fixed expected digest is checked below and shared by every OS.
	const expected = "c50197762badfb95166d1ca81ff56baa1b69d689179055dc47f76fd06d09a436"
	got := DigestSealBytes(canonical)
	if got != expected {
		t.Errorf("native fixture digest=%s want=%s canonical=%s", got, expected, canonical)
	}
	t.Logf("G21P0_NATIVE_JSON %s", string(canonical))
	t.Logf("G21P0_NATIVE_RUNTIME_JSON {\"goos\":%q,\"goarch\":%q,\"go_version\":%q,\"fixture_count\":%d,\"digest\":%q}",
		runtime.GOOS, runtime.GOARCH, runtime.Version(), len(results), got)
}
