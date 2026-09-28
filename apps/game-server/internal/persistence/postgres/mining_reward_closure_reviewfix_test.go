package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestG21ClosureReviewOwnerLockDeadlockRetries(t *testing.T) {
	s, cmd := closureTimeoutFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	config := s.pool.Config()
	counter := &closureTransactionCounter{}
	config.ConnConfig.Tracer = counter
	config.ConnConfig.RuntimeParams["deadlock_timeout"] = "50ms"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	worker := &Store{pool: pool}
	blocker, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(context.Background())
	if _, err = blocker.Exec(ctx, `SET LOCAL deadlock_timeout='5s'; SELECT 1 FROM characters WHERE id='g20-player' FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	type result struct {
		receipt MiningRewardSettlementReceipt
		err     error
	}
	done := make(chan result, 1)
	go func() { r, e := worker.SettleMiningRewardTEST(ctx, cmd); done <- result{r, e} }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waits int
		err = s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'SELECT account_id,revision FROM characters%'`).Scan(&waits)
		if err != nil {
			t.Fatal(err)
		}
		if waits == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("settlement never blocked on owner")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// The settlement owns GLOBAL and waits for our owner. Closing the cycle here
	// forces its 50ms deadlock detector (versus this backend's5s) to raise40P01.
	if _, err = blocker.Exec(ctx, `SELECT 1 FROM black_iron_emission_pools WHERE pool_id='GLOBAL' FOR UPDATE`); err != nil {
		t.Fatalf("wrong deadlock victim: %v", err)
	}
	if err = blocker.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	got := <-done
	if got.err != nil || got.receipt.TotalOre != 10 {
		t.Fatalf("owner-lock deadlock did not retry: receipt=%+v err=%v begins=%d", got.receipt, got.err, counter.begins.Load())
	}
	if n := counter.begins.Load(); n != 2 {
		t.Fatalf("real owner deadlock attempts=%d want2", n)
	}
	closureReconcile(t, s)
}

func TestG21ClosureReviewBindingQuerySQLStateRetries(t *testing.T) {
	s, cmd := closureTimeoutFixture(t)
	// A test-only view preserves every authority row and injects a genuine
	// PostgreSQL40001 specifically when the late beneficiary aggregate reads IDs.
	closureSQL(t, s, `DROP SEQUENCE IF EXISTS closure_binding_attempts; CREATE SEQUENCE closure_binding_attempts;
 CREATE OR REPLACE FUNCTION closure_binding_read(v text) RETURNS text LANGUAGE plpgsql AS $$ BEGIN
 IF position('SELECT count(*),min(binding_id)' IN current_query())>0 AND nextval('closure_binding_attempts')=1 THEN
 RAISE EXCEPTION 'forced beneficiary read serialization failure' USING ERRCODE='40001'; END IF; RETURN v; END $$;
 ALTER TABLE mining_power_beneficiary_bindings RENAME TO closure_original_beneficiary_bindings;
 CREATE VIEW mining_power_beneficiary_bindings AS SELECT closure_binding_read(binding_id) AS binding_id,activity_id,source_event_id,block_instance_id,account_id,player_id,character_id,authority_source,binding_version,bound_at,digest FROM closure_original_beneficiary_bindings`)
	defer closureSQL(t, s, `DROP VIEW mining_power_beneficiary_bindings; ALTER TABLE closure_original_beneficiary_bindings RENAME TO mining_power_beneficiary_bindings`)
	r, err := s.SettleMiningRewardTEST(context.Background(), cmd)
	if err != nil || r.TotalOre != 10 {
		t.Fatalf("beneficiary SQLSTATE not retried: %+v %v", r, err)
	}
	var attempts int
	if err = s.pool.QueryRow(context.Background(), `SELECT last_value FROM closure_binding_attempts`).Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("binding attempts=%d err=%v", attempts, err)
	}
	closureReconcile(t, s)
}

func TestG21ClosureReviewPreviewRejectsHistoricalUnboundPower(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica; DELETE FROM mining_power_beneficiary_bindings`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil || seal.TotalValidMiningPower <= 0 || seal.Eligibility != "NOT_SETTLEMENT_ELIGIBLE" {
		t.Fatalf("historical seal=%+v %v", seal, err)
	}
	before := closureSnapshot(t, s)
	preview, err := s.PreviewMiningRewardTEST(ctx, i.BlockInstanceID, seal.CanonicalDigest)
	if !errors.Is(err, ErrMiningRewardInvariant) || preview.Eligibility == "ELIGIBLE" {
		t.Fatalf("unbound preview advertised authority: %+v %v", preview, err)
	}
	closureZero(t, s, before)
}
