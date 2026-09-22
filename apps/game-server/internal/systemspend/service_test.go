package systemspend

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"

	"fractallegend/game-server/internal/contribution"
)

type captureRepository struct{ last ResolvedIntent }

func (r *captureRepository) PostSystemSpend(_ context.Context, value ResolvedIntent) (SystemSpend, error) {
	r.last = value
	return SystemSpend{OperationID: value.OperationID, FBAmount: value.FBAmount, ContributionAmount: value.ContributionAmount, Eligible: value.Eligible, RuleVersion: value.RuleVersion}, nil
}
func (*captureRepository) LoadSystemSpend(context.Context, string) (SystemSpend, error) {
	return SystemSpend{}, ErrNotFound
}
func (*captureRepository) ReconcileSystemSpends(context.Context) (ReconciliationReport, error) {
	return ReconciliationReport{Balanced: true}, nil
}

func testRegistry(t *testing.T) Registry {
	t.Helper()
	registry, err := NewRegistry([]Producer{
		{Type: ProducerInternalTestEligible, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Refundable: true, PartialRefundAllowed: true, Active: true, Description: "internal eligible test"},
		{Type: ProducerInternalTestNonEligible, Active: true, Description: "internal non-eligible test"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func testIntent() Intent {
	return Intent{OperationID: "operation-1", PlayerID: "player-1", PlayerFBAccountID: "fb-player-1", SystemFBAccountID: "fb-system", ProducerType: ProducerInternalTestEligible, ProducerReference: "reference-1", FBAmount: 100}
}

func TestG15RegistryDisablesAllGameplayProducers(t *testing.T) {
	registry := ProductionRegistry()
	for _, kind := range []ProducerType{ProducerNPCService, ProducerEquipmentUpgrade, ProducerManufacturing, ProducerMiningToolCraft, ProducerOrdinalsActivation, ProducerShop} {
		if _, err := registry.Resolve(kind); !errors.Is(err, ErrDisabledProducer) {
			t.Fatalf("producer=%s err=%v", kind, err)
		}
	}
	if _, err := NewRegistry([]Producer{{Type: ProducerNPCService, Eligible: true, AllowedRuleVersion: contribution.RuleVersionV1, Active: true, Description: "forged activation"}}); !errors.Is(err, ErrDisabledProducer) {
		t.Fatalf("activated gameplay producer: %v", err)
	}
	if _, err := registry.Resolve(ProducerType("CLIENT_FORGED")); !errors.Is(err, ErrUnknownProducer) {
		t.Fatalf("unknown producer: %v", err)
	}
	if _, err := NewRegistry([]Producer{{Type: ProducerInternalTestNonEligible, Refundable: true, Active: true, Description: "invalid non-eligible refund"}}); !errors.Is(err, ErrInvalidIntent) {
		t.Fatalf("non-eligible refund policy accepted: %v", err)
	}
}

func TestG15RejectsClientClaimsAndInvalidReferences(t *testing.T) {
	var decoded Intent
	if err := json.Unmarshal([]byte(`{"OperationID":"operation-1","Eligible":true,"ContributionAmount":999,"RuleVersion":"CONTRIBUTION_RULE_V1"}`), &decoded); !errors.Is(err, ErrClientIntent) {
		t.Fatalf("client JSON accepted: %v", err)
	}
	repo := &captureRepository{}
	service := NewService(repo, testRegistry(t))
	for _, value := range []Intent{
		{OperationID: "bad' OR 1=1", PlayerID: "player-1", PlayerFBAccountID: "fb-player-1", SystemFBAccountID: "fb-system", ProducerType: ProducerInternalTestEligible, ProducerReference: "ref", FBAmount: 1},
		{OperationID: "operation-1", PlayerID: "player-1", PlayerFBAccountID: "fb-player-1", SystemFBAccountID: "fb-system", ProducerType: ProducerInternalTestEligible, ProducerReference: "", FBAmount: 1},
		{OperationID: "operation-1", PlayerID: "player-1", PlayerFBAccountID: "fb-player-1", SystemFBAccountID: "fb-system", ProducerType: ProducerInternalTestEligible, ProducerReference: "ref", FBAmount: 0},
		{OperationID: "operation-1", PlayerID: "player-1", PlayerFBAccountID: "fb-player-1", SystemFBAccountID: "fb-player-1", ProducerType: ProducerInternalTestEligible, ProducerReference: "ref", FBAmount: 1},
	} {
		if _, err := service.Post(context.Background(), value); !errors.Is(err, ErrInvalidIntent) {
			t.Fatalf("intent=%+v err=%v", value, err)
		}
	}
	unknown := testIntent()
	unknown.ProducerType = ProducerType("CLIENT_FORGED")
	if _, err := service.Post(context.Background(), unknown); !errors.Is(err, ErrUnknownProducer) {
		t.Fatalf("unknown producer: %v", err)
	}
}

func TestG15RuleResolutionProperty400(t *testing.T) {
	repo := &captureRepository{}
	service := NewService(repo, testRegistry(t))
	random := rand.New(rand.NewSource(1515))
	var totalFB, totalContribution int64
	for i := 0; i < 400; i++ {
		intent := testIntent()
		intent.OperationID = "property-" + stringID(i)
		intent.ProducerReference = "reference-" + stringID(i)
		intent.FBAmount = 1 + random.Int63n(10_000)
		if i%4 == 0 {
			intent.ProducerType = ProducerInternalTestNonEligible
		}
		value, err := service.Post(context.Background(), intent)
		if err != nil {
			t.Fatal(err)
		}
		if value.FBAmount != intent.FBAmount || value.ContributionAmount < 0 || value.ContributionAmount > value.FBAmount {
			t.Fatalf("iteration=%d value=%+v", i, value)
		}
		if intent.ProducerType == ProducerInternalTestEligible {
			if !value.Eligible || value.ContributionAmount != value.FBAmount || value.RuleVersion != contribution.RuleVersionV1 {
				t.Fatalf("eligible iteration=%d value=%+v", i, value)
			}
		} else if value.Eligible || value.ContributionAmount != 0 || value.RuleVersion != "" {
			t.Fatalf("non-eligible iteration=%d value=%+v", i, value)
		}
		totalFB += value.FBAmount
		totalContribution += value.ContributionAmount
	}
	if totalContribution <= 0 || totalContribution >= totalFB {
		t.Fatalf("aggregate contribution=%d FB=%d", totalContribution, totalFB)
	}
}

func stringID(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}
