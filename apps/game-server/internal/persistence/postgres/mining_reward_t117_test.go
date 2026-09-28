package postgres

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningreward"
)

// Receipt time is server-generated, not a command field. Equivalent receipt
// representations must retain historical command identity, bytes and zero delta;
// a different microsecond must still change the receipt's canonical digest.
func TestG21T117PrecisionBoundariesAndHistoricalReplay(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx := context.Background()
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	original, err := canonicalMiningRewardReceipt(first)
	if err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	seal, err := loadMiningPowerSealTx(ctx, tx, cmd.BlockInstanceID)
	_ = tx.Rollback(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		at   time.Time
		same bool
	}{
		{"same_microsecond_low", first.CreatedAt.Add(time.Nanosecond), true},
		{"same_microsecond_high", first.CreatedAt.Add(999 * time.Nanosecond), true},
		{"next_microsecond", first.CreatedAt.Add(time.Microsecond), false},
		{"same_instant_other_offset", first.CreatedAt.In(time.FixedZone("+0800", 8*3600)), true},
		{"different_instant", first.CreatedAt.Add(time.Second), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copy := first
			copy.CreatedAt = tc.at
			raw, err := canonicalMiningRewardReceipt(copy)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(raw, original) != tc.same || (rewardHash(raw) == rewardHash(original)) != tc.same {
				t.Fatal("timestamp identity boundary collapsed or drifted")
			}
			decoded, err := decodeCanonicalMiningRewardReceipt(raw)
			if err != nil || decoded.CreatedAt != miningreward.CanonicalTime(tc.at) {
				t.Fatalf("decoded time=%v error=%v", decoded.CreatedAt, err)
			}
			fingerprint, err := miningRewardFingerprint(cmd, seal, first.ReservationBindingDigest)
			if err != nil || fingerprint != first.CommandFingerprint {
				t.Fatalf("immutable command fingerprint changed: %v", err)
			}
			fresh, err := Open(ctx, s.pool.Config().ConnString())
			if err != nil {
				t.Fatal(err)
			}
			defer fresh.Close()
			replay, err := fresh.SettleMiningRewardTEST(ctx, cmd)
			if err != nil || !reflect.DeepEqual(first, replay) {
				t.Fatalf("historical replay changed: %v", err)
			}
			replayBytes, err := canonicalMiningRewardReceipt(replay)
			if err != nil || !bytes.Equal(replayBytes, original) {
				t.Fatal("immutable receipt changed")
			}
		})
	}
	changed := cmd
	changed.ExpectedSealDigest = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	rejected, err := s.SettleMiningRewardTEST(ctx, changed)
	if !errors.Is(err, ErrMiningRewardConflict) || rejected.SettlementID != "" {
		t.Fatalf("different historical identity accepted: %v", err)
	}
	if _, err = s.pool.Exec(ctx, `UPDATE mining_reward_settlement_receipts SET created_at=created_at+interval '1 microsecond' WHERE command_id=$1`, cmd.CommandID); err == nil {
		t.Fatal("immutable timestamp update accepted")
	}
	if g21ClosureSnapshot(t, s) != before {
		t.Fatal("replay or rejected mutation changed persisted state")
	}
}

func TestG21T117IssuanceDigestCanonicalPrecision(t *testing.T) {
	v := MiningRewardIssuance{Status: "G21_SETTLED", RuleVersion: settlementIssuanceVersion, SourceType: "MINING_REWARD", SourceKey: "fixed-source", BlockInstanceID: "fixed-instance", CharacterID: "A", MaterialDefinitionID: "ore", Quantity: 1, CreatedAt: time.Date(2026, 9, 28, 12, 0, 0, 123456001, time.UTC)}
	first, err := miningRewardSourceDigest(v)
	if err != nil {
		t.Fatal(err)
	}
	v.CreatedAt = time.Date(2026, 9, 28, 20, 0, 0, 123456999, time.FixedZone("+0800", 8*3600))
	same, err := miningRewardSourceDigest(v)
	if err != nil || same != first {
		t.Fatalf("same persisted issuance instant changed digest: %v", err)
	}
	v.CreatedAt = time.Date(2026, 9, 28, 12, 0, 0, 123457000, time.UTC)
	next, err := miningRewardSourceDigest(v)
	if err != nil || next == first {
		t.Fatalf("different microsecond lost issuance identity: %v", err)
	}
	// P0 evidence is a separate immutable prerequisite contract.
	v.Status = "P0_SYNTHETIC_ONLY"
	v.RuleVersion = rewardIssuanceRule
	v.CreatedAt = time.Date(2026, 9, 28, 12, 0, 0, 123456001, time.UTC)
	oldA, err := miningRewardSourceDigest(v)
	if err != nil {
		t.Fatal(err)
	}
	v.CreatedAt = v.CreatedAt.Add(time.Nanosecond)
	oldB, err := miningRewardSourceDigest(v)
	if err != nil || oldA == oldB {
		t.Fatal("G21 normalization changed P0 evidence precision")
	}
}
