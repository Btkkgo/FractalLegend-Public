package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestG21SettlementLockTimeoutAndCanceledContextLeaveZeroDelta(t *testing.T) {
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
	command := MiningRewardSettlementCommand{CommandID: "g21-lock-timeout", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
	var before, after string
	const snapshot = `SELECT jsonb_build_object(
		'pool',(SELECT to_jsonb(p) FROM black_iron_emission_pools p WHERE pool_id='GLOBAL'),
		'owner',(SELECT jsonb_object_agg(id,revision) FROM characters),
		'items',(SELECT count(*) FROM character_inventory_items),
		'commands',(SELECT count(*) FROM mining_reward_commands),
		'grants',(SELECT count(*) FROM mining_reward_grants),
		'lots',(SELECT count(*) FROM mining_reward_issuance_lots),
		'projections',(SELECT count(*) FROM mining_reward_inventory_projections),
		'consumptions',(SELECT count(*) FROM mining_reward_reservation_consumptions),
		'receipts',(SELECT count(*) FROM mining_reward_settlement_receipts))::text`
	if err := s.pool.QueryRow(ctx, snapshot).Scan(&before); err != nil {
		t.Fatal(err)
	}
	blocker, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	limited, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	started := time.Now()
	_, err = s.SettleMiningRewardTEST(limited, command)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Fatalf("expected lock timeout, got %v", err)
	}
	if elapsed := time.Since(started); elapsed < 4*time.Second || elapsed > 9*time.Second {
		t.Fatalf("5-second lock timeout elapsed=%s", elapsed)
	}
	if err := blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	canceled, stop := context.WithCancel(ctx)
	stop()
	if _, err := s.SettleMiningRewardTEST(canceled, command); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled caller=%v", err)
	}
	if err := s.pool.QueryRow(ctx, snapshot).Scan(&after); err != nil || after != before {
		t.Fatalf("timeout/cancellation changed state before=%s after=%s err=%v", before, after, err)
	}
	if receipt, err := s.SettleMiningRewardTEST(ctx, command); err != nil || receipt.TotalOre != 10 {
		t.Fatalf("fresh retry after timeout=%+v %v", receipt, err)
	}
}
