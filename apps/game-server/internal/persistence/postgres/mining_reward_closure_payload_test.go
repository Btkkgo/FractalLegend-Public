package postgres

import (
	"context"
	"errors"
	"fractallegend/game-server/internal/miningpower"
	"testing"
)

func TestG21ClosurePersistedPayloadCaps(t *testing.T) {
	for _, kind := range []string{"seal64MiB", "receipt4MiB"} {
		t.Run(kind, func(t *testing.T) {
			s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
			ctx := context.Background()
			table := "mining_power_input_seals"
			limit := settlementMaxSealBytes
			if kind == "receipt4MiB" {
				if _, e := s.SettleMiningRewardTEST(ctx, cmd); e != nil {
					t.Fatal(e)
				}
				table = "mining_reward_settlement_receipts"
				limit = settlementMaxReceiptBytes
			}
			// Corrupt persisted transport bytes, never alter reward/source authority.
			tx, e := s.pool.Begin(ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback(ctx)
			if _, e = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); e != nil {
				t.Fatal(e)
			}
			if _, e = tx.Exec(ctx, `UPDATE `+table+` SET canonical_bytes=canonical_bytes || convert_to(repeat(' ', $1-octet_length(canonical_bytes)+1),'UTF8')`, limit); e != nil {
				t.Fatal(e)
			}
			if e = tx.Commit(ctx); e != nil {
				t.Fatal(e)
			}
			var size int
			if e = s.pool.QueryRow(ctx, `SELECT octet_length(canonical_bytes) FROM `+table).Scan(&size); e != nil || size != limit+1 {
				t.Fatal(size, e)
			}
			before := rewardHash([]byte(g21ClosureSnapshot(t, s)))
			_, e = s.SettleMiningRewardTEST(ctx, cmd)
			if !errors.Is(e, miningpower.ErrInvariant) && !errors.Is(e, ErrMiningRewardInvariant) {
				t.Fatal("oversized authority replay", e)
			}
			if after := rewardHash([]byte(g21ClosureSnapshot(t, s))); after != before {
				t.Fatal("oversize rejection mutated data")
			}
			t.Logf("%s limit=%d stored=%d rejected before mutation", kind, limit, size)
		})
	}
}
