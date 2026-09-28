package postgres

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"testing"
	"testing/fstest"

	"fractallegend/game-server/internal/miningblock"
	"github.com/jackc/pgx/v5/pgconn"
)

// These tests catch missing generation/exposure, identity mutation, reuse of
// business-derived identity, and backfill that changes historical economics.
var g181IdentityPattern = regexp.MustCompile(`^mining-block-instance-[0-9a-f]{32}$`)

func g181Identity(t *testing.T, s *Store, blockID string) string {
	t.Helper()
	var id string
	err := s.pool.QueryRow(context.Background(), `SELECT coalesce(to_jsonb(b)->>'block_instance_id','') FROM mining_blocks b WHERE block_id=$1`, blockID).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	if !g181IdentityPattern.MatchString(id) || id == "mining-block-instance-00000000000000000000000000000000" {
		t.Fatalf("missing or invalid independent instance identity: %q", id)
	}
	return id
}

func g181ModelIdentity(t *testing.T, b miningblock.Block) string {
	t.Helper()
	if b.BlockInstanceID == "" {
		t.Fatal("G18 read model does not expose BlockInstanceID")
	}
	return b.BlockInstanceID
}

func g181Insert(t *testing.T, s *Store, id string, height int64, command string) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `INSERT INTO mining_blocks
		(block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,reward_reserved,pool_revision_at_reservation,created_at,updated_at)
		VALUES($1,$2,$3,'OPEN','DEV_G18_FIXED_BLOCK_REWARD','2026-09-22 00:00:00+00','2026-09-22 00:01:00+00',10,1,'2026-09-22 00:00:00+00','2026-09-22 00:00:00+00')`, id, height, command)
	if err != nil {
		t.Fatal(err)
	}
}

func g181SQLState(t *testing.T, err error, code, constraint string) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(err, &p) || p.Code != code || (constraint != "" && p.ConstraintName != constraint) {
		t.Fatalf("wanted SQLSTATE %s constraint %s, got %v", code, constraint, err)
	}
}

func TestG181CreateHasIdentity(t *testing.T) {
	s, svc, _, _ := g18Fund(t, 20)
	ctx := context.Background()
	first, err := svc.Create(ctx, "g181-first")
	if err != nil {
		t.Fatal(err)
	}
	id := g181Identity(t, s, first.BlockID)
	b, err := s.LoadMiningBlock(ctx, first.BlockID)
	if err != nil || g181ModelIdentity(t, b) != id {
		t.Fatalf("load did not preserve identity: %+v %v", b, err)
	}
	snapshot, err := s.SnapshotMiningBlocks(ctx)
	if err != nil || len(snapshot.Blocks) != 1 || g181ModelIdentity(t, snapshot.Blocks[0]) != id {
		t.Fatalf("snapshot lost identity: %+v %v", snapshot, err)
	}
	second, err := svc.Create(ctx, "g181-second")
	if err != nil || g181Identity(t, s, second.BlockID) == id {
		t.Fatalf("two new blocks share identity: %v", err)
	}
	if _, err = svc.Create(ctx, "g181-first"); err != nil || g181Identity(t, s, first.BlockID) != id {
		t.Fatalf("same-command replay changed identity: %v", err)
	}
}

func TestG181CreateGeneratesWithoutDatabaseDefault(t *testing.T) {
	s, svc, _, _ := g18Fund(t, 10)
	ctx := context.Background()
	if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_blocks ALTER COLUMN block_instance_id DROP DEFAULT`); err != nil {
		t.Fatal(err)
	}
	r, err := svc.Create(ctx, "g181-explicit-generation")
	if err != nil {
		t.Fatal(err)
	}
	g181Identity(t, s, r.BlockID)
}

func TestG181BusinessMutationKeepsIdentity(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"Height", `UPDATE mining_blocks SET block_height=99 WHERE block_id='g181-fixture'`},
		{"CreateCommandID", `UPDATE mining_blocks SET create_command_id='g181-changed' WHERE block_id='g181-fixture'`},
		{"BlockID", `UPDATE mining_blocks SET block_id='g181-renamed' WHERE block_id='g181-fixture'`},
		{"UnchangedIdentityAssignment", `UPDATE mining_blocks SET block_instance_id=block_instance_id WHERE block_id='g181-fixture'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := integrationStore(t)
			if err := s.Migrate(context.Background()); err != nil {
				t.Fatal(err)
			}
			g181Insert(t, s, "g181-fixture", 1, "g181-fixture-command")
			id := g181Identity(t, s, "g181-fixture")
			if _, err := s.pool.Exec(context.Background(), tc.sql); err != nil {
				t.Fatalf("existing permitted update rejected: %v", err)
			}
			blockID := "g181-fixture"
			if tc.name == "BlockID" {
				blockID = "g181-renamed"
			}
			b, err := s.LoadMiningBlock(context.Background(), blockID)
			if err != nil || g181ModelIdentity(t, b) != id {
				t.Fatalf("business mutation changed instance: %+v %v", b, err)
			}
			if tc.name == "Height" && b.Height != 99 || tc.name == "CreateCommandID" && b.CreateCommandID != "g181-changed" {
				t.Fatal("test did not mutate the business field")
			}
		})
	}
}

func TestG181IdentityDatabaseGuards(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	g181Insert(t, s, "g181-guard", 1, "g181-guard-command")
	id := g181Identity(t, s, "g181-guard")
	for _, value := range []any{newContributionID("mining-block-instance"), nil, "", "mining-block-instance-00000000000000000000000000000000"} {
		_, err := s.pool.Exec(ctx, `UPDATE mining_blocks SET block_instance_id=$2,block_height=2 WHERE block_id=$1`, "g181-guard", value)
		g181SQLState(t, err, "55000", "")
		if g181Identity(t, s, "g181-guard") != id {
			t.Fatal("rejected update changed identity")
		}
		b, err := s.LoadMiningBlock(ctx, "g181-guard")
		if err != nil || b.Height != 1 {
			t.Fatal("rejected update leaked a business-field change")
		}
	}
	for _, tc := range []struct {
		name, code, constraint string
		value                  any
	}{
		{"duplicate", "23505", "mining_blocks_instance_id_unique", id},
		{"null", "23502", "", nil},
		{"empty", "23514", "mining_blocks_instance_id_valid", ""},
		{"zero", "23514", "mining_blocks_instance_id_valid", "mining-block-instance-00000000000000000000000000000000"},
		{"malformed", "23514", "mining_blocks_instance_id_valid", "derived-from-block-id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.pool.Exec(ctx, `INSERT INTO mining_blocks
				(block_id,block_instance_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,reward_reserved,pool_revision_at_reservation,created_at,updated_at)
				VALUES('g181-invalid',$1,2,'g181-invalid-command','OPEN','DEV_G18_FIXED_BLOCK_REWARD',now(),now()+interval '1 minute',10,1,now(),now())`, tc.value)
			g181SQLState(t, err, tc.code, tc.constraint)
		})
	}
}

func TestG181RecreatedBusinessFieldsGetNewIdentity(t *testing.T) {
	for _, reset := range []string{"DELETE FROM mining_blocks", "TRUNCATE mining_blocks CASCADE"} {
		t.Run(reset, func(t *testing.T) {
			s := integrationStore(t)
			ctx := context.Background()
			if err := s.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			g181Insert(t, s, "g181-identical-block", 1, "g181-identical-command")
			original := g181Identity(t, s, "g181-identical-block")
			before, err := s.LoadMiningBlock(ctx, "g181-identical-block")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.pool.Exec(ctx, reset); err != nil {
				if reset == "TRUNCATE mining_blocks CASCADE" {
					if g181Identity(t, s, "g181-identical-block") != original {
						t.Fatal("rejected TRUNCATE changed block identity")
					}
					return // G21 recovery history now rejects cascading TRUNCATE.
				}
				t.Fatal(err)
			}
			g181Insert(t, s, "g181-identical-block", 1, "g181-identical-command")
			if g181Identity(t, s, "g181-identical-block") == original {
				t.Fatal("recreated block reused instance identity")
			}
			after, err := s.LoadMiningBlock(ctx, "g181-identical-block")
			if err != nil || before.ID != after.ID || before.Height != after.Height || before.CreateCommandID != after.CreateCommandID || !before.StartedAt.Equal(after.StartedAt) {
				t.Fatalf("recreation did not repeat business fields: before=%+v after=%+v %v", before, after, err)
			}
		})
	}
}

func TestG181RecoveryAndNewBlockAfterRestart(t *testing.T) {
	for _, status := range []string{"OPEN", "FINALIZED", "CANCELLED", "RECOVERY_DEBT"} {
		t.Run(status, func(t *testing.T) {
			s, svc, contributions, spend := g18Fund(t, 30)
			ctx := context.Background()
			r, err := svc.Create(ctx, "g181-recovery-original")
			if err != nil {
				t.Fatal(err)
			}
			original := g181Identity(t, s, r.BlockID)
			switch status {
			case "FINALIZED":
				_, err = svc.Finalize(ctx, r.BlockID)
			case "CANCELLED":
				_, err = svc.Cancel(ctx, r.BlockID)
			case "RECOVERY_DEBT":
				_, err = contributions.RefundSystemSpend(ctx, g15Refund(spend, "g181-debt", 25))
			}
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(reopened.Close)
			if err = reopened.Migrate(ctx); err != nil {
				t.Fatal(err)
			}
			restarted := miningblock.NewService(reopened, miningblock.DevelopmentRuleVersion, false)
			b, err := restarted.Recover(ctx, r.BlockID)
			if err != nil || g181ModelIdentity(t, b) != original {
				t.Fatalf("same-block recovery lost identity: %+v %v", b, err)
			}
			if _, err = restarted.Create(ctx, "g181-recovery-original"); err != nil || g181Identity(t, reopened, r.BlockID) != original {
				t.Fatalf("restart replay replaced same block: %v", err)
			}
			if status == "RECOVERY_DEBT" {
				if _, err = restarted.Create(ctx, "g181-new-after-recovery"); !errors.Is(err, miningblock.ErrRecoveryDebt) {
					t.Fatalf("identity hardening weakened debt gate: %v", err)
				}
			} else {
				fresh, err := restarted.Create(ctx, "g181-new-after-recovery")
				if err != nil || g181Identity(t, reopened, fresh.BlockID) == original || g181Identity(t, reopened, r.BlockID) != original {
					t.Fatalf("new block after recovery did not get new identity: %v", err)
				}
			}
		})
	}
}

func g181LegacyMigrations(t *testing.T) fstest.MapFS {
	t.Helper()
	old := fstest.MapFS{}
	files, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.Name() >= "0012_" {
			continue
		}
		data, err := fs.ReadFile(migrations, "migrations/"+file.Name())
		if err != nil {
			t.Fatal(err)
		}
		old["migrations/"+file.Name()] = &fstest.MapFile{Data: data}
	}
	if len(old) != 11 {
		t.Fatalf("legacy migration count = %d", len(old))
	}
	return old
}

func g181BusinessEconomy(t *testing.T, s *Store) map[string]string {
	t.Helper()
	snapshot := g19Economy(t, s)
	var blocks string
	if err := s.pool.QueryRow(context.Background(), `SELECT coalesce(jsonb_agg(v ORDER BY v::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(b)-'block_instance_id' v FROM mining_blocks b) q`).Scan(&blocks); err != nil {
		t.Fatal(err)
	}
	snapshot["mining_blocks"] = blocks
	return snapshot
}

func TestG181MigrationBackfillPreservesRowsAndEconomy(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if err := s.MigrateFS(ctx, g181LegacyMigrations(t)); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		g181Insert(t, s, fmt.Sprintf("g181-legacy-%d", i), int64(i), fmt.Sprintf("g181-legacy-command-%d", i))
	}
	// Nonzero debt and OPEN/FINALIZED/CANCELLED legacy rows, with reservations
	// and immutable journals/receipts, exercise preservation across the upgrade.
	_, err := s.pool.Exec(ctx, `
		UPDATE mining_blocks SET status='FINALIZED',finalized_at=scheduled_end_at WHERE block_height=2;
		UPDATE mining_blocks SET status='CANCELLED',cancelled_at=scheduled_end_at,reward_returned=10 WHERE block_height=3;
		INSERT INTO mining_block_reservations(block_id,amount,released,status,created_at,updated_at)
		SELECT block_id,10,CASE WHEN status='CANCELLED' THEN 10 ELSE 0 END,CASE WHEN status='CANCELLED' THEN 'RELEASED' ELSE 'ACTIVE' END,created_at,updated_at FROM mining_blocks;
		INSERT INTO mining_block_entries(entry_id,block_id,block_height,action,capacity_delta,pool_before,pool_after,block_status_before,block_status_after,rule_version,created_at)
		SELECT 'legacy-entry-'||block_id,block_id,block_height,'OPEN',10,30,20,'','OPEN',rule_version,created_at FROM mining_blocks;
		INSERT INTO mining_block_receipts(receipt_id,entry_id,block_id,block_height,action,reward_reserved,pool_before,pool_after,block_status,rule_version,created_at)
		SELECT 'legacy-receipt-'||block_id,entry_id,block_id,block_height,'OPEN',10,30,20,'OPEN',rule_version,created_at FROM mining_block_entries;
		UPDATE black_iron_emission_pools SET total_emission_capacity=13,total_reserved=20,remaining_capacity=0,recovery_debt=7 WHERE pool_id='GLOBAL';`)
	if err != nil {
		t.Fatal(err)
	}
	before := g181BusinessEconomy(t, s)
	// This fixture tests the 0011→0012 identity backfill, rather than applying
	// future schemas that legitimately add unrelated tables to its inventory.
	identityMigration, err := fs.ReadFile(migrations, "migrations/0012_mining_block_instance_identity.sql")
	if err != nil {
		t.Fatal(err)
	}
	identityOnly := fstest.MapFS{"migrations/0012_mining_block_instance_identity.sql": {Data: identityMigration}}
	if err = s.MigrateFS(ctx, identityOnly); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, g181BusinessEconomy(t, s)) {
		t.Fatal("identity backfill changed a business/economic field or lost rows")
	}
	ids := map[string]bool{}
	for i := 1; i <= 3; i++ {
		id := g181Identity(t, s, fmt.Sprintf("g181-legacy-%d", i))
		if ids[id] {
			t.Fatal("backfill assigned duplicate identity")
		}
		ids[id] = true
	}
	var count, unique, missing int
	if err = s.pool.QueryRow(ctx, `SELECT count(*),count(DISTINCT block_instance_id),count(*) FILTER (WHERE block_instance_id IS NULL) FROM mining_blocks`).Scan(&count, &unique, &missing); err != nil || count != 3 || unique != 3 || missing != 0 {
		t.Fatalf("incomplete backfill %d/%d/%d: %v", count, unique, missing, err)
	}
	full := g19Economy(t, s)
	if err = s.MigrateFS(ctx, identityOnly); err != nil || !reflect.DeepEqual(full, g19Economy(t, s)) {
		t.Fatal("repeat migration changed persisted identity/economics")
	}
}

func TestG181FailedMigrationRollsBackIdentityBackfill(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if err := s.MigrateFS(ctx, g181LegacyMigrations(t)); err != nil {
		t.Fatal(err)
	}
	g181Insert(t, s, "g181-rollback", 1, "g181-rollback-command")
	before := g181BusinessEconomy(t, s)
	data, err := fs.ReadFile(migrations, "migrations/0012_mining_block_instance_identity.sql")
	if err != nil {
		t.Fatal(err)
	}
	broken := fstest.MapFS{"migrations/0012_mining_block_instance_identity.sql": {Data: append(data, []byte("\nSELECT 1/0;\n")...)}}
	if err = s.MigrateFS(ctx, broken); err == nil {
		t.Fatal("broken migration unexpectedly committed")
	}
	var columns, versions int
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='mining_blocks' AND column_name='block_instance_id'`).Scan(&columns); err != nil {
		t.Fatal(err)
	}
	if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version=12`).Scan(&versions); err != nil || columns != 0 || versions != 0 || !reflect.DeepEqual(before, g181BusinessEconomy(t, s)) {
		t.Fatalf("failed backfill leaked state: columns=%d versions=%d %v", columns, versions, err)
	}
	if err = s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	g181Identity(t, s, "g181-rollback")
}

func TestG181IndependentRestartReadsIdentity(t *testing.T) {
	ctx := context.Background()
	if blockID := os.Getenv("G181_VERIFY_BLOCK_ID"); blockID != "" {
		s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()
		if err = s.Migrate(ctx); err != nil {
			t.Fatal(err)
		}
		b, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Recover(ctx, blockID)
		if err != nil || g181ModelIdentity(t, b) != os.Getenv("G181_VERIFY_INSTANCE_ID") {
			t.Fatalf("independent process lost persisted identity: %+v %v", b, err)
		}
		return
	}
	s, svc, _, _ := g18Fund(t, 10)
	r, err := svc.Create(ctx, "g181-process-restart")
	if err != nil {
		t.Fatal(err)
	}
	id := g181Identity(t, s, r.BlockID)
	s.Close()
	cmd := exec.Command(os.Args[0], "-test.run=^TestG181IndependentRestartReadsIdentity$", "-test.v")
	cmd.Env = append(os.Environ(), "G181_VERIFY_BLOCK_ID="+r.BlockID, "G181_VERIFY_INSTANCE_ID="+id)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("independent restart failed: %v %s", err, out)
	}
}
