package miningpower

import (
	"errors"
	"strings"
	"testing"
)

func TestG20IntentValidation(t *testing.T) {
	legal := `{"activityId":"mpa:event-1","sourceEventId":"event-1","activitySessionId":"session-1","blockId":"block-1","blockInstanceId":"instance-1"}`
	intent, err := DecodeActionIntent([]byte(legal))
	if err != nil || intent.SourceEventID != "event-1" || intent.ActivityID != "mpa:event-1" {
		t.Fatalf("legal intent=%+v err=%v", intent, err)
	}
	for _, field := range []string{"miningPower", "power", "finalPower", "effectivePower", "participantPower", "blockPowerShare", "rewardShare", "FinalMiningPower", "EffectiveMiningPower", "PlayerShare", "TotalMiningPower", "FINAL_POWER", "mining_power"} {
		t.Run("T19/"+field, func(t *testing.T) {
			_, err := DecodeActionIntent([]byte(strings.TrimSuffix(legal, "}") + `,"` + field + `":999999999}`))
			if !errors.Is(err, ErrClientPowerForbidden) {
				t.Fatalf("forged final power not explicitly rejected: %v", err)
			}
		})
	}
	for _, tc := range []struct{ name, extra string }{
		{"T15/fake", `"miningPower":999999999`},
		{"T16/extreme", `"finalPower":999999999999999999999999999999999999999`},
		{"T17/negative", `"miningPower":-1`},
		{"T18/nan", `"miningPower":"NaN"`},
		{"T18/infinity", `"miningPower":"Infinity"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeActionIntent([]byte(strings.TrimSuffix(legal, "}") + "," + tc.extra + "}")); !errors.Is(err, ErrClientPowerForbidden) {
				t.Fatalf("final power accepted or silently ignored: %v", err)
			}
		})
	}
	for _, field := range []string{"ToolBasePower", "Efficiency", "ToolEfficiency", "ValidMiningActivity", "ActivityWeight", "MapModifier", "ruleVersion", "playerId", "clientTimestamp"} {
		t.Run("T20/"+field, func(t *testing.T) {
			if _, err := DecodeActionIntent([]byte(strings.TrimSuffix(legal, "}") + `,"` + field + `":123}`)); err == nil {
				t.Fatal("forged authoritative input accepted")
			}
		})
	}
	for _, tc := range []struct{ name, payload string }{
		{"duplicate", strings.TrimSuffix(legal, "}") + `,"blockId":"block-2"}`},
		{"unknown", strings.TrimSuffix(legal, "}") + `,"extra":1}`},
		{"case", strings.Replace(legal, "blockId", "BlockId", 1)},
		{"nested", strings.Replace(legal, `"block-1"`, `{"block":"block-1"}`, 1)},
		{"nonstring", strings.Replace(legal, `"block-1"`, `123`, 1)},
		{"trailing", legal + `{}`}, {"array", "[" + legal + "]"},
		{"missing", `{"sourceEventId":"event-1"}`}, {"malformed", `{`},
		{"oversize", strings.Repeat(" ", 4097) + legal},
		{"rawnan", strings.TrimSuffix(legal, "}") + `,"x":NaN}`},
		{"rawinf", strings.TrimSuffix(legal, "}") + `,"x":Infinity}`},
	} {
		t.Run("T21/"+tc.name, func(t *testing.T) {
			if _, err := DecodeActionIntent([]byte(tc.payload)); err == nil {
				t.Fatal("malformed intent accepted")
			}
		})
	}
}

func TestG20IdentityBinding(t *testing.T) {
	base := ActionIntent{ActivityID: "mpa:event-1", SourceEventID: "event-1", ActivitySessionID: "session-1", BlockID: "block-1", BlockInstanceID: "instance-1"}
	if err := ValidateIntent(base); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "event/1", "事件", strings.Repeat("a", 97), "x\x00y"} {
		t.Run("T14/"+value, func(t *testing.T) {
			bad := base
			bad.SourceEventID = value
			bad.ActivityID = "mpa:" + value
			if ValidateIntent(bad) == nil {
				t.Fatal("invalid source identity accepted")
			}
		})
	}
	bad := base
	bad.ActivityID = "another-id"
	if !errors.Is(ValidateIntent(bad), ErrInvalidIntent) {
		t.Fatal("T24 noncanonical mapping accepted")
	}
}
