package recycle

import (
	"encoding/json"
	"errors"
	"testing"
	"testing/quick"
	"time"
)

func fixtureRule(input, output int64) Rule {
	return Rule{ID: "g16-test", Version: "v1", TemplateID: "item-mafa-tulong", Enabled: true,
		Recyclable: true, VerifiedInput: []Material{{MaterialRecycleScrap, input}},
		MaterialOutputs: []Material{{MaterialRecycleScrap, output}}, ReputationReward: 2,
		MaxReturnBasisPoints: 5000, InputProvenance: "G16_TEST_FIXTURE",
		AntiFarm:  AntiFarmPolicy{PeriodCapPolicy: "UNDECIDED", DiminishingReturnPolicy: "UNDECIDED"},
		CreatedAt: time.Date(2026, 9, 22, 0, 0, 0, 0, time.UTC)}
}

func TestReturnRatioPropertyAndForbiddenOutputs(t *testing.T) {
	property := func(raw uint16) bool {
		input := int64(raw%1000) + 1
		for output := int64(0); output <= input/2; output++ {
			if _, err := NewRegistry([]Rule{fixtureRule(input, output)}); err != nil {
				return false
			}
		}
		_, err := NewRegistry([]Rule{fixtureRule(input, input/2+1)})
		return errors.Is(err, ErrInvalidRule)
	}
	if err := quick.Check(property, &quick.Config{MaxCount: 50}); err != nil {
		t.Fatal(err)
	}
	rule := fixtureRule(10, 5)
	rule.MaterialOutputs[0].ID = "BLACK_IRON_ORE"
	if _, err := NewRegistry([]Rule{rule}); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("ore rule accepted: %v", err)
	}
	rule = fixtureRule(10, 5)
	rule.MaterialOutputs[0].ID = "UNREGISTERED_MATERIAL"
	if _, err := NewRegistry([]Rule{rule}); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("unknown material accepted: %v", err)
	}
}

func TestRuleFailClosedAndIntentBoundary(t *testing.T) {
	rule := fixtureRule(10, 5)
	registry, err := NewRegistry([]Rule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve("missing"); !errors.Is(err, ErrUnknownRule) {
		t.Fatal(err)
	}
	rule.Enabled = false
	registry, err = NewRegistry([]Rule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registry.Resolve(rule.ID); !errors.Is(err, ErrDisabledRule) {
		t.Fatal(err)
	}
	rule = fixtureRule(10, 5)
	rule.AntiFarm.PeriodCapPolicy = ""
	if _, err := NewRegistry([]Rule{rule}); !errors.Is(err, ErrInvalidRule) {
		t.Fatalf("missing anti-farm boundary accepted: %v", err)
	}
	var intent Intent
	if err := json.Unmarshal([]byte(`{"OperationID":"untrusted","FBAward":100}`), &intent); !errors.Is(err, ErrClientIntent) {
		t.Fatalf("client intent accepted: %v", err)
	}
}
