package postgres

import (
	"context"
	"fmt"
	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
	"os"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"
	"time"
)

func g20Upstream(t *testing.T, s *Store) map[string]string {
	t.Helper()
	ctx := context.Background()
	rows, e := s.pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename NOT LIKE 'mining_power_%' AND tablename<>'schema_migrations' ORDER BY tablename`)
	if e != nil {
		t.Fatal(e)
	}
	tables := []string{}
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			t.Fatal(e)
		}
		tables = append(tables, name)
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
	rows.Close()
	result := map[string]string{}
	for _, table := range tables {
		var raw string
		sql := `SELECT coalesce(jsonb_agg(value ORDER BY value::text),'[]'::jsonb)::text FROM(SELECT to_jsonb(row) value FROM ` + pgx.Identifier{table}.Sanitize() + ` row) data`
		if e = s.pool.QueryRow(ctx, sql).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		result[table] = raw
	}
	return result
}

func TestG20SameBlockIDDifferentInstance(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	original := g20Accept(t, s, p, i)
	// Only the isolated fixture authority recreates G18. G20 has no FK to it.
	if _, e := s.pool.Exec(ctx, `CREATE TEMP TABLE g20_old_block AS SELECT * FROM mining_blocks; TRUNCATE mining_blocks CASCADE;
 INSERT INTO mining_blocks(block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,finalized_at,cancelled_at,reward_reserved,reward_released,reward_returned,pool_revision_at_reservation,created_at,updated_at)
 SELECT block_id,block_height,create_command_id,status,rule_version,started_at,scheduled_end_at,finalized_at,cancelled_at,reward_reserved,reward_released,reward_returned,pool_revision_at_reservation,created_at,updated_at FROM g20_old_block`); e != nil {
		t.Fatal(e)
	}
	b, e := s.LoadMiningBlock(ctx, i.BlockID)
	if e != nil || b.BlockInstanceID == i.BlockInstanceID || b.ID != i.BlockID {
		t.Fatalf("recreated block=%+v %v", b, e)
	}
	replay, e := s.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
	if e != nil || replay.Status != miningpower.StatusDuplicate || !reflect.DeepEqual(original, *replay.Original) {
		t.Fatalf("old A resolved as B: %+v %v", replay, e)
	}
	stale := g20Event(t, s, i, "old-instance-new-event")
	r, e := s.ValidateMiningActivity(ctx, p, stale, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusNotEligible || r.ReasonCode != "BLOCK_INSTANCE_NOT_FOUND" {
		t.Fatalf("deleted A validated via B: %+v %v", r, e)
	}
	session := g20Session(t, s, i)
	session.ID = "new-instance-session"
	session.BlockInstanceID = b.BlockInstanceID
	g20Insert(t, s, "mining_power_sessions", session)
	source := g20Source(t, s, i)
	source.ID = "new-instance-event"
	source.ActivityID = "mpa:" + source.ID
	source.ActivitySessionID = session.ID
	source.BlockInstanceID = b.BlockInstanceID
	next := g20RegisteredSource(t, s, source)
	g20Accept(t, s, p, next)
	for _, instance := range []string{i.BlockInstanceID, b.BlockInstanceID} {
		snap, e := s.SnapshotMiningPower(ctx, instance, miningpower.DevelopmentRuleVersion)
		if e != nil || len(snap.Activities) != 1 || len(snap.Participants) != 1 || snap.Participants[0].BlockInstanceID != instance || snap.Participants[0].ValidatedPower != 100 {
			t.Fatalf("instance isolation=%+v %v", snap, e)
		}
		report, e := s.ReconcileMiningPower(ctx, instance, miningpower.DevelopmentRuleVersion)
		if e != nil || report.Status != "PASS" {
			t.Fatalf("instance report=%+v %v", report, e)
		}
	}
}

func TestG20CrossPlayerAndVersionReplay(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g19Seed(t, s, "other-miner")
	other := miningpower.Principal{AccountID: "other-miner-account", PlayerID: "other-miner"}
	for _, committed := range []bool{false, true} {
		if committed {
			g20Accept(t, s, p, i)
		}
		r, e := s.ValidateMiningActivity(ctx, other, i, miningpower.DevelopmentRuleVersion)
		if e != nil || r.Status != miningpower.StatusNotEligible || r.Original != nil || r.AppliedPower != 0 {
			t.Fatalf("cross-player disclosure=%+v %v", r, e)
		}
	}
	r, e := s.ValidateMiningActivity(ctx, p, i, "FUTURE_RULE")
	if e != nil || r.Status != miningpower.StatusReplayed || r.AppliedPower != 0 {
		t.Fatalf("version recredit=%+v %v", r, e)
	}
	altered := i
	altered.ActivitySessionID = "another-session"
	r, e = s.ValidateMiningActivity(ctx, p, altered, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusReplayed {
		t.Fatalf("session replay=%+v %v", r, e)
	}
	altered = i
	altered.ActivityID = "different-activity"
	r, e = s.ValidateMiningActivity(ctx, p, altered, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusInvalid {
		t.Fatalf("source remap=%+v %v", r, e)
	}
	a, b := g20Counts(t, s)
	if a != 1 || b != 1 {
		t.Fatal("replay changed aggregates")
	}
}

func TestG20LifecycleOverlap(t *testing.T) {
	t.Run("G20-session-close", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		locker, e := s.pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		defer locker.Rollback(context.Background())
		if _, e = locker.Exec(ctx, `UPDATE mining_power_sessions SET data=jsonb_set(data,'{State}','"CLOSED"') WHERE session_id=$1`, i.ActivitySessionID); e != nil {
			t.Fatal(e)
		}
		accepting, e := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL")+"&application_name=g20-session-close-probe")
		if e != nil {
			t.Fatal(e)
		}
		defer accepting.Close()
		done := make(chan miningpower.ValidationResult, 1)
		errors := make(chan error, 1)
		go func() {
			r, e := accepting.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
			done <- r
			errors <- e
		}()
		blocked := false
		for !blocked && ctx.Err() == nil {
			if e = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE application_name='g20-session-close-probe' AND wait_event_type='Lock')`).Scan(&blocked); e != nil {
				t.Fatal(e)
			}
			if !blocked {
				time.Sleep(time.Millisecond)
			}
		}
		if !blocked {
			t.Fatal("acceptance did not encounter session close lock")
		}
		if e = locker.Commit(ctx); e != nil {
			t.Fatal(e)
		}
		r := <-done
		if e = <-errors; e != nil || r.Status != miningpower.StatusNotEligible {
			t.Fatalf("closed stale session accepted: %+v %v", r, e)
		}
		a, b := g20Counts(t, s)
		if a != 0 || b != 0 {
			t.Fatal("session close caused stale credit")
		}
	})
	t.Run("G18-snapshot-close", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		once := false
		s.miningPowerFailureInjector = func(stage string) error {
			if stage == "after_validation" && !once {
				once = true
				_, e := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Cancel(ctx, i.BlockID)
				return e
			}
			return nil
		}
		a := g20Accept(t, s, p, i)
		if a.ValidationSnapshot.Status != "OPEN" {
			t.Fatal("historical snapshot overwritten")
		}
		live, e := s.LoadMiningBlock(ctx, i.BlockID)
		if e != nil || live.Status != miningblock.StatusCancelled {
			t.Fatalf("external close fixture=%+v %v", live, e)
		}
		report, e := s.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
		if e != nil || report.Status != "PASS" {
			t.Fatalf("snapshot semantics=%+v %v", report, e)
		}
	})
}

func TestG20ExpiredCommittedReplayAndWindows(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	source := g20Source(t, s, i)
	source.ID = "short-event"
	source.ActivityID = "mpa:short-event"
	source.ExpiresAt = time.Now().UTC().Add(3 * time.Second).Truncate(time.Microsecond)
	short := g20RegisteredSource(t, s, source)
	original := g20Accept(t, s, p, short)
	timer := time.NewTimer(time.Until(source.ExpiresAt) + time.Millisecond)
	defer timer.Stop()
	<-timer.C
	r, e := s.ValidateMiningActivity(ctx, p, short, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusDuplicate || !reflect.DeepEqual(original, *r.Original) {
		t.Fatalf("expired committed replay=%+v %v", r, e)
	}
	source.ID = "expired-uncommitted"
	source.ActivityID = "mpa:expired-uncommitted"
	expired := g20RegisteredSource(t, s, source)
	r, e = s.ValidateMiningActivity(ctx, p, expired, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusExpired || r.AppliedPower != 0 {
		t.Fatalf("expired uncommitted accepted=%+v %v", r, e)
	}
	// A distinct legitimate zero-power activity is audited exactly once.
	session := g20Session(t, s, i)
	session.ID = "zero-session"
	session.ToolReference = "TEST_ZERO"
	g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: "TEST_ZERO", RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, BasePowerUnits: 0, EfficiencyScaled: miningpower.Scale})
	g20Insert(t, s, "mining_power_sessions", session)
	source = g20Source(t, s, i)
	source.ID = "zero-event"
	source.ActivityID = "mpa:zero-event"
	source.ActivitySessionID = session.ID
	source.ToolReference = session.ToolReference
	zero := g20RegisteredSource(t, s, source)
	a := g20Accept(t, s, p, zero)
	if a.ValidatedPower != 0 {
		t.Fatal("zero gained minimum-one")
	}
	empty, e := s.TotalValidatedPower(ctx, "mining-block-instance-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", miningpower.DevelopmentRuleVersion)
	if e != nil || empty != 0 {
		t.Fatalf("empty total=%d %v", empty, e)
	}
	if _, e = s.pool.Exec(ctx, `UPDATE mining_power_participants SET validated_power=9223372036854775807`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.TotalValidatedPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion); e == nil {
		t.Fatal("block total overflow ignored")
	}
}

func TestG20MultipleMiners(t *testing.T) {
	for _, n := range []int{2, 10, 50, 100, 500} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s, _, i := g20Fixture(t)
			ctx := context.Background()
			session := g20Session(t, s, i)
			source := g20Source(t, s, i)
			principals := make([]miningpower.Principal, n)
			intents := make([]miningpower.ActionIntent, n)
			expected := int64(0)
			for k := 0; k < n; k++ {
				id := fmt.Sprintf("miner-%d", k)
				g19Seed(t, s, id)
				ss := session
				ss.ID = id + "-session"
				ss.PlayerID = id
				ss.AccountID = id + "-account"
				power := int64(100)
				if k%2 == 1 {
					ss.ToolReference = "TEST_ADVANCED"
					power = 250
				}
				g20Insert(t, s, "mining_power_sessions", ss)
				event := source
				event.ID = id + "-event"
				event.ActivityID = "mpa:" + event.ID
				event.PlayerID = id
				event.ActivitySessionID = ss.ID
				event.ToolReference = ss.ToolReference
				intents[k] = g20RegisteredSource(t, s, event)
				principals[k] = miningpower.Principal{AccountID: ss.AccountID, PlayerID: id}
				expected += power
			}
			before := g20Upstream(t, s)
			for k := n - 1; k >= 0; k-- {
				a := g20Accept(t, s, principals[k], intents[k])
				if a.ValidatedPower < 0 {
					t.Fatal("negative power")
				}
			}
			for _, k := range []int{0, n - 1} {
				r, e := s.ValidateMiningActivity(ctx, principals[k], intents[k], miningpower.DevelopmentRuleVersion)
				if e != nil || r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0 {
					t.Fatalf("miner replay=%+v %v", r, e)
				}
			}
			if n == 100 {
				var wg sync.WaitGroup
				errs := make(chan error, n)
				start := make(chan struct{})
				for k := range intents {
					wg.Add(1)
					go func(k int) {
						defer wg.Done()
						<-start
						r, e := s.ValidateMiningActivity(ctx, principals[k], intents[k], miningpower.DevelopmentRuleVersion)
						if e == nil && (r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0) {
							e = fmt.Errorf("concurrent miner replay: %+v", r)
						}
						errs <- e
					}(k)
				}
				close(start)
				wg.Wait()
				close(errs)
				for e := range errs {
					if e != nil {
						t.Fatal(e)
					}
				}
			}
			total, e := s.TotalValidatedPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
			if e != nil || total != expected {
				t.Fatalf("N=%d total=%d expected=%d %v", n, total, expected, e)
			}
			snap, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
			if e != nil || len(snap.Activities) != n || len(snap.Participants) != n {
				t.Fatalf("N=%d counts=%d/%d %v", n, len(snap.Activities), len(snap.Participants), e)
			}
			report, e := s.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
			if e != nil || report.Status != "PASS" {
				t.Fatalf("miner reconcile=%+v %v", report, e)
			}
			if !reflect.DeepEqual(before, g20Upstream(t, s)) {
				t.Fatal("mining power changed upstream assets/pool/history")
			}
		})
	}
}

func TestG20MigrationPreservesEconomy(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	old := g181LegacyMigrations(t)
	raw, e := os.ReadFile("migrations/0012_mining_block_instance_identity.sql")
	if e != nil {
		t.Fatal(e)
	}
	// Apply the previous accepted schema boundary first, then the new migration.
	old["migrations/0012_mining_block_instance_identity.sql"] = &fstest.MapFile{Data: raw}
	if e = s.MigrateFS(ctx, old); e != nil {
		t.Fatal(e)
	}
	g19Catalog(t, s)
	g19Seed(t, s, "preserve-ore", 3)
	if _, e = s.MigrateBlackIronInventory(ctx, "preserve-ore"); e != nil {
		t.Fatal(e)
	}
	before := g20Upstream(t, s)
	ddl, e := os.ReadFile("migrations/0013_mining_power_foundation.sql")
	if e != nil {
		t.Fatal(e)
	}
	bad := fstest.MapFS{"migrations/0013_mining_power_foundation.sql": &fstest.MapFile{Data: append(append([]byte{}, ddl...), []byte("; INVALID SQL;")...)}}
	if e = s.MigrateFS(ctx, bad); e == nil {
		t.Fatal("failed G20 migration succeeded")
	}
	var leaked int
	if e = s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM schema_migrations WHERE version=13)+(SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename LIKE 'mining_power_%')`).Scan(&leaked); e != nil || leaked != 0 || !reflect.DeepEqual(before, g20Upstream(t, s)) {
		t.Fatalf("migration DDL/registry/data leaked: %d %v", leaked, e)
	}
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(before, g20Upstream(t, s)) {
		t.Fatal("additive migration changed prior data")
	}
	for _, name := range []string{"mining_power_rules", "mining_power_tool_profiles", "mining_power_map_profiles", "mining_power_sessions", "mining_power_source_events", "mining_power_activities", "mining_power_participants"} {
		var count int
		if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM `+name).Scan(&count); e != nil || count != 0 {
			t.Fatalf("migration seeded %s=%d %v", name, count, e)
		}
	}
}

func TestG20EconomyIsolation(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	mining := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false)
	finalized, e := mining.Create(ctx, "g20-existing-finalized")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = mining.Finalize(ctx, finalized.BlockID); e != nil {
		t.Fatal(e)
	}
	spend, e := s.LoadSystemSpend(ctx, "g18-fund")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g20-existing-debt", 15)); e != nil {
		t.Fatal(e)
	}
	g19Catalog(t, s)
	g19Seed(t, s, "existing-ore", 3, 4)
	if _, e = s.MigrateBlackIronInventory(ctx, "existing-ore"); e != nil {
		t.Fatal(e)
	}
	g19Seed(t, s, "recycle-miner")
	if _, e = s.pool.Exec(ctx, `INSERT INTO character_inventory_items(instance_id,character_id,definition_id,legacy_id,name,item_type,quantity,slot_index,location) VALUES('g20-recycle-sword','recycle-miner','item-mafa-tulong',411,'TEST synthetic sword','EQUIPMENT',1,0,'INVENTORY')`); e != nil {
		t.Fatal(e)
	}
	recycleIntent := g16Intent("g20-recycle-sword", "g20-existing-reputation")
	recycleIntent.PlayerID = "recycle-miner"
	if _, e = g16Service(t, s).Recycle(ctx, recycleIntent); e != nil {
		t.Fatal(e)
	}
	var debt int64
	if e = s.pool.QueryRow(ctx, `SELECT recovery_debt FROM black_iron_emission_pools`).Scan(&debt); e != nil || debt <= 0 {
		t.Fatalf("nonzero recovery debt absent: %d %v", debt, e)
	}
	g20Insert(t, s, "mining_power_map_profiles", miningpower.MapProfile{Reference: "TEST_BOOST", RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, ModifierScaled: 1500000})
	ss := g20Session(t, s, i)
	ss.ID = "advanced-session"
	ss.ToolReference = "TEST_ADVANCED"
	ss.MapReference = "TEST_BOOST"
	g20Insert(t, s, "mining_power_sessions", ss)
	event := g20Source(t, s, i)
	event.ID = "advanced-event"
	event.ActivityID = "mpa:advanced-event"
	event.ActivitySessionID = ss.ID
	event.ToolReference = ss.ToolReference
	event.MapReference = ss.MapReference
	advanced := g20RegisteredSource(t, s, event)
	before := g20Upstream(t, s)
	basic := g20Accept(t, s, p, i)
	strong := g20Accept(t, s, p, advanced)
	if basic.ValidatedPower != 100 || strong.ValidatedPower != 375 {
		t.Fatalf("synthetic profiles not server-derived: %d/%d", basic.ValidatedPower, strong.ValidatedPower)
	}
	if !reflect.DeepEqual(before, g20Upstream(t, s)) {
		t.Fatal("G20 changed FB/Contribution/Reputation/Ore/G19/Pool/Reward/Debt or upstream history")
	}
	r, e := s.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != "PASS" {
		t.Fatalf("economic fixture power report=%+v %v", r, e)
	}
}
