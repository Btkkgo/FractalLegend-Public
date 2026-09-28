package postgres

import (
	"context"
	"testing"

	"fractallegend/game-server/internal/miningpower"
)

// An accepted fact needs persisted owner evidence; equal-looking IDs alone do
// not make the character inventory an authoritative reward destination.
func TestG21P0AcceptanceCapturesCharacterAuthority(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	result, err := s.ValidateMiningActivity(context.Background(), principal, intent, miningpower.DevelopmentRuleVersion)
	if err != nil || result.Status != miningpower.StatusValid {
		t.Fatalf("acceptance=%+v err=%v", result, err)
	}
	var account, player, character, instance, digest string
	err = s.pool.QueryRow(context.Background(), `SELECT account_id,player_id,character_id,block_instance_id,digest
		FROM mining_power_beneficiary_bindings WHERE activity_id=$1`, intent.ActivityID).
		Scan(&account, &player, &character, &instance, &digest)
	if err != nil {
		t.Fatal(err)
	}
	if account != principal.AccountID || player != principal.PlayerID || character != "g20-player" || instance != intent.BlockInstanceID || len(digest) != 64 {
		t.Fatalf("not an authoritative character binding: %q %q %q %q %q", account, player, character, instance, digest)
	}
}

func TestG21P0UnboundHistoricalActivityNotSettlementEligible(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, p, i)
	// Simulate a pre-binding accepted Activity without guessing its owner.
	if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_power_beneficiary_bindings DISABLE TRIGGER mining_power_beneficiary_bindings_immutable;
		DELETE FROM mining_power_beneficiary_bindings;
		ALTER TABLE mining_power_beneficiary_bindings ENABLE TRIGGER mining_power_beneficiary_bindings_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil || seal.Eligibility != "NOT_SETTLEMENT_ELIGIBLE" || seal.ActivityCount != 1 {
		t.Fatalf("unbound activity became eligible: %+v %v", seal, err)
	}
	if _, err = s.RebuildMiningPowerSeal(ctx, i.BlockInstanceID); err != nil {
		t.Fatalf("historical fact cannot rebuild: %v", err)
	}
}
