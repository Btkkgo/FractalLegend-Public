package postgres

import (
	"context"
	"testing"
	"time"
)

// The bulk fixture exercises the database admission gate with 100,000 valid
// source/activity identity pairs. It does not substitute for an end-to-end
// 100,000-event Seal or for a participant allocation fixture.
func TestG21ActivityAdmission100000And100001Boundary(t *testing.T) {
	s, principal, intent := g20Fixture(t)
	g20Accept(t, s, principal, intent)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	const sources = `WITH base AS (
		SELECT data FROM mining_power_source_events WHERE source_event_id=$1
	) INSERT INTO mining_power_source_events(data)
	SELECT jsonb_set(jsonb_set(base.data,'{ID}',to_jsonb('g21-cap-source-'||n)),
		'{ActivityID}',to_jsonb('mpa:g21-cap-source-'||n))
	FROM base CROSS JOIN generate_series(2,100001) AS n`
	if _, err := s.pool.Exec(ctx, sources, intent.SourceEventID); err != nil {
		t.Fatal(err)
	}
	const activities = `WITH base AS (
		SELECT data FROM mining_power_activities WHERE activity_id=$1
	) INSERT INTO mining_power_activities(data)
	SELECT jsonb_set(jsonb_set(base.data,'{SourceEventID}',to_jsonb('g21-cap-source-'||n)),
		'{ActivityID}',to_jsonb('mpa:g21-cap-source-'||n))
	FROM base CROSS JOIN generate_series(2,100000) AS n`
	started := time.Now()
	if _, err := s.pool.Exec(ctx, activities, intent.ActivityID); err != nil {
		t.Fatal(err)
	}
	var admitted, actual int
	if err := s.pool.QueryRow(ctx, `SELECT c.activity_count,
		(SELECT count(*) FROM mining_power_activities WHERE block_instance_id=$1)
		FROM mining_power_activity_admission_counts c WHERE block_instance_id=$1`, intent.BlockInstanceID).
		Scan(&admitted, &actual); err != nil {
		t.Fatal(err)
	}
	if admitted != 100000 || actual != 100000 {
		t.Fatalf("legal input boundary count=%d actual=%d", admitted, actual)
	}
	var before string
	const economics = `SELECT jsonb_build_object(
		'pool',(SELECT to_jsonb(p) FROM black_iron_emission_pools p WHERE pool_id='GLOBAL'),
		'reservation',(SELECT to_jsonb(r) FROM mining_block_reservations r WHERE block_id=$1),
		'owner',(SELECT jsonb_object_agg(id,revision) FROM characters),
		'items',(SELECT count(*) FROM character_inventory_items),
		'grants',(SELECT count(*) FROM mining_reward_grants),
		'lots',(SELECT count(*) FROM mining_reward_issuance_lots),
		'projections',(SELECT count(*) FROM mining_reward_inventory_projections),
		'consumptions',(SELECT count(*) FROM mining_reward_reservation_consumptions),
		'receipts',(SELECT count(*) FROM mining_reward_settlement_receipts),
		'completed',(SELECT count(*) FROM mining_reward_settlement_states WHERE state='COMPLETED'))::text`
	if err := s.pool.QueryRow(ctx, economics, intent.BlockID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	_, err := s.pool.Exec(ctx, `WITH base AS (
		SELECT data FROM mining_power_activities WHERE activity_id=$1
	) INSERT INTO mining_power_activities(data)
	SELECT jsonb_set(jsonb_set(base.data,'{SourceEventID}',to_jsonb('g21-cap-source-100001')),
		'{ActivityID}',to_jsonb('mpa:g21-cap-source-100001')) FROM base`, intent.ActivityID)
	if err == nil {
		t.Fatal("100001st Activity was admitted")
	}
	var after, state string
	if err := s.pool.QueryRow(ctx, economics, intent.BlockID).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1`, intent.BlockInstanceID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if before != after || state != "OPEN" {
		t.Fatalf("100001st rejection changed economic state or Seal gate: before=%s after=%s state=%s", before, after, state)
	}
	if err := s.pool.QueryRow(ctx, `SELECT activity_count FROM mining_power_activity_admission_counts WHERE block_instance_id=$1`, intent.BlockInstanceID).Scan(&admitted); err != nil || admitted != 100000 {
		t.Fatalf("rejected input changed admitted count=%d %v", admitted, err)
	}
	t.Logf("100000 admission and 100001 rejection elapsed=%s", time.Since(started))
}
