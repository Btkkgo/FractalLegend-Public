package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"fractallegend/game-server/internal/miningpower"
)

func TestG21StrictCommandDecodeRejectsForbiddenAndAmbiguousPayloads(t *testing.T) {
	const valid = `{"commandID":"g21-strict","blockInstanceID":"mining-block-instance-11111111111111111111111111111111","expectedSealDigest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`
	command, err := DecodeMiningRewardCommandTEST([]byte(valid))
	if err != nil || command.CommandID != "g21-strict" {
		t.Fatalf("valid strict command=%+v %v", command, err)
	}
	for name, raw := range map[string]string{
		"missing":          `{"commandID":"g21-strict"}`,
		"duplicate":        strings.Replace(valid, `"commandID":"g21-strict"`, `"commandID":"g21-strict","commandID":"other"`, 1),
		"reward":           strings.Replace(valid, `"commandID":`, `"reward":500,"commandID":`, 1),
		"weight":           strings.Replace(valid, `"commandID":`, `"weight":1,"commandID":`, 1),
		"quantity":         strings.Replace(valid, `"commandID":`, `"quantity":500,"commandID":`, 1),
		"participants":     strings.Replace(valid, `"commandID":`, `"participants":[],"commandID":`, 1),
		"beneficiary":      strings.Replace(valid, `"commandID":`, `"beneficiary":"A","commandID":`, 1),
		"numeric-id":       strings.Replace(valid, `"commandID":"g21-strict"`, `"commandID":42`, 1),
		"null-id":          strings.Replace(valid, `"commandID":"g21-strict"`, `"commandID":null`, 1),
		"upper-digest":     strings.Replace(valid, strings.Repeat("a", 64), strings.Repeat("A", 64), 1),
		"invalid-instance": strings.Replace(valid, "mining-block-instance-11111111111111111111111111111111", "some-other-instance", 1),
		"trailing":         valid + `{}`,
		"array":            `[]`,
		"nan":              strings.Replace(valid, `"commandID":`, `"reward":NaN,"commandID":`, 1),
		"infinity":         strings.Replace(valid, `"commandID":`, `"reward":Infinity,"commandID":`, 1),
		"oversize":         valid + strings.Repeat(" ", miningRewardMaxCommandBytes),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeMiningRewardCommandTEST([]byte(raw)); !errors.Is(err, miningpower.ErrInvalidInput) {
				t.Fatalf("payload accepted or wrong error: %v", err)
			}
		})
	}
}

func TestG21StrictPayloadSettlementUsesServerAuthorities(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, principal, intent)
	if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{
		"commandID": "g21-strict-real", "blockInstanceID": intent.BlockInstanceID,
		"expectedSealDigest": seal.CanonicalDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	invalid := append([]byte(nil), raw[:len(raw)-1]...)
	invalid = append(invalid, []byte(`,"reward":500}`)...)
	if _, err := s.SettleMiningRewardPayloadTEST(ctx, invalid); !errors.Is(err, miningpower.ErrInvalidInput) {
		t.Fatalf("forbidden reward payload=%v", err)
	}
	var before int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reward_commands`).Scan(&before); err != nil || before != 0 {
		t.Fatalf("invalid payload reached transaction count=%d %v", before, err)
	}
	first, err := s.SettleMiningRewardPayloadTEST(ctx, raw)
	if err != nil || first.TotalOre != 10 || first.Status != "COMPLETED" {
		t.Fatalf("strict settlement=%+v %v", first, err)
	}
	replay, err := s.SettleMiningRewardPayloadTEST(ctx, raw)
	if err != nil || !reflect.DeepEqual(first, replay) {
		t.Fatalf("strict replay=%+v %v", replay, err)
	}
}
