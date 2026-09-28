package postgres

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"fractallegend/game-server/internal/miningpower"
)

// A new reservation must carry its concrete instance evidence in the same
// committed create operation; a display BlockID alone is insufficient.
func TestG21P0ReservationBindsNewInstance(t *testing.T) {
	s, service := g20Fund(t, 20)
	ctx := context.Background()
	receipt, err := service.Create(ctx, "g21-p0-binding-create")
	if err != nil {
		t.Fatal(err)
	}
	block, err := s.LoadMiningBlock(ctx, receipt.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	var instance, source, rule string
	var amount int64
	err = s.pool.QueryRow(ctx, `SELECT block_instance_id,reservation_source_id,g18_rule_version,reservation_amount
		FROM mining_reservation_instance_bindings WHERE block_instance_id=$1`, block.BlockInstanceID).
		Scan(&instance, &source, &rule, &amount)
	if err != nil {
		t.Fatal(err)
	}
	if instance != block.BlockInstanceID || source == "" || rule != block.RuleVersion || amount != receipt.RewardReserved {
		t.Fatalf("binding disagrees with authoritative reservation: instance=%q source=%q rule=%q amount=%d", instance, source, rule, amount)
	}
}

func TestG21P0HistoricalUnboundReservationFailsClosed(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	old := fstest.MapFS{}
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() < "0014_" {
			raw, e := fs.ReadFile(migrations, "migrations/"+entry.Name())
			if e != nil {
				t.Fatal(e)
			}
			old["migrations/"+entry.Name()] = &fstest.MapFile{Data: raw}
		}
	}
	if err = s.MigrateFS(ctx, old); err != nil {
		t.Fatal(err)
	}
	g181Insert(t, s, "g21-p0-unbound", 1, "g21-p0-unbound-command")
	var instance string
	if err := s.pool.QueryRow(ctx, `SELECT block_instance_id FROM mining_blocks WHERE block_id='g21-p0-unbound'`).Scan(&instance); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `UPDATE mining_blocks SET status='FINALIZED',finalized_at=now(),updated_at=now() WHERE block_id='g21-p0-unbound'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO mining_block_reservations(block_id,amount,status,created_at,updated_at)
		VALUES('g21-p0-unbound',10,'ACTIVE',now(),now())`); err != nil {
		t.Fatal(err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var bindings int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_reservation_instance_bindings`).Scan(&bindings); err != nil || bindings != 0 {
		t.Fatalf("historical binding was inferred: %d %v", bindings, err)
	}
	_, err = s.SealMiningPowerTEST(ctx, instance)
	if !errors.Is(err, miningpower.ErrInvalidInput) {
		t.Fatalf("unbound reservation sealed: %v", err)
	}
	var n int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_input_seals`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("unbound seal created=%d %v", n, err)
	}
}
