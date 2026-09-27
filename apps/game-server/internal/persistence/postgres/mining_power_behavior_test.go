package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func g20Source(t *testing.T, s *Store, i miningpower.ActionIntent) miningpower.SourceEvent {
	t.Helper()
	v, e := miningPowerRead[miningpower.SourceEvent](context.Background(), s.pool, `SELECT data FROM mining_power_source_events WHERE source_event_id=$1`, i.SourceEventID)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func g20Session(t *testing.T, s *Store, i miningpower.ActionIntent) miningpower.ActivitySession {
	t.Helper()
	v, e := miningPowerRead[miningpower.ActivitySession](context.Background(), s.pool, `SELECT data FROM mining_power_sessions WHERE session_id=$1`, i.ActivitySessionID)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func g20RegisteredSource(t *testing.T, s *Store, v miningpower.SourceEvent) miningpower.ActionIntent {
	t.Helper()
	g20Insert(t, s, "mining_power_source_events", v)
	return miningpower.ActionIntent{ActivityID: v.ActivityID, SourceEventID: v.ID, ActivitySessionID: v.ActivitySessionID, BlockID: v.BlockID, BlockInstanceID: v.BlockInstanceID}
}

func TestG20RejectedContext(t *testing.T) {
	for _, name := range []string{"missing-source", "wrong-player", "wrong-account", "wrong-instance", "wrong-display", "wrong-session", "unknown-rule", "invalid-evidence", "zero-weight", "expired-source", "future-observation", "closed-session", "inactive-block", "finalized-block"} {
		t.Run(name, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			rule := miningpower.DevelopmentRuleVersion
			switch name {
			case "missing-source":
				i.SourceEventID = "missing"
				i.ActivityID = "mpa:missing"
			case "wrong-player":
				p.PlayerID = "missing-player"
			case "wrong-account":
				p.AccountID = "another-account"
			case "wrong-instance":
				i.BlockInstanceID = "mining-block-instance-ffffffffffffffffffffffffffffffff"
			case "wrong-display":
				i.BlockID = "another-display"
			case "wrong-session":
				i.ActivitySessionID = "another-session"
			case "unknown-rule":
				rule = "UNAPPROVED_VERSION"
			case "invalid-evidence", "zero-weight", "expired-source", "future-observation":
				v := g20Source(t, s, i)
				v.ID = "variant"
				v.ActivityID = "mpa:variant"
				if name == "invalid-evidence" {
					v.ServerEligibility = miningpower.StatusInvalid
				}
				if name == "zero-weight" {
					v.ActivityWeightScaled = 0
				}
				if name == "expired-source" {
					session := g20Session(t, s, i)
					v.ObservedAt = session.OpenedAt
					v.ExpiresAt = session.OpenedAt.Add(time.Microsecond)
				}
				if name == "future-observation" {
					v.ObservedAt = time.Now().UTC().Add(10 * time.Second).Truncate(time.Microsecond)
				}
				i = g20RegisteredSource(t, s, v)
			case "closed-session":
				if _, e := s.pool.Exec(ctx, `UPDATE mining_power_sessions SET data=jsonb_set(data,'{State}','"CLOSED"')`); e != nil {
					t.Fatal(e)
				}
			case "finalized-block":
				if _, e := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Finalize(ctx, i.BlockID); e != nil {
					t.Fatal(e)
				}
			case "inactive-block":
				if _, e := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Cancel(ctx, i.BlockID); e != nil {
					t.Fatal(e)
				}
			}
			r, e := s.ValidateMiningActivity(ctx, p, i, rule)
			if e != nil || r.Status == miningpower.StatusValid || r.Status == miningpower.StatusDuplicate || r.AppliedPower != 0 || r.Original != nil {
				t.Fatalf("rejection=%+v %v", r, e)
			}
			a, b := g20Counts(t, s)
			if a != 0 || b != 0 {
				t.Fatalf("rejected request wrote facts/aggregate: %d/%d", a, b)
			}
		})
	}
}

func TestG20SyntheticResolversAndLocalConstraints(t *testing.T) {
	s, _, i := g20Fixture(t)
	ctx := context.Background()
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	resolver := miningPowerProfiles{tx: tx}
	rule := miningpower.DevelopmentRuleVersion
	for _, ref := range []string{"", "MISSING", "PRODUCTION_TOOL"} {
		if _, e = resolver.ResolveTool(ctx, ref, rule); !errors.Is(e, pgx.ErrNoRows) {
			t.Fatalf("missing tool fallback: %v", e)
		}
	}
	for _, ref := range []string{"", "MISSING", "PRODUCTION_MAP"} {
		if _, e = resolver.ResolveMap(ctx, ref, rule); !errors.Is(e, pgx.ErrNoRows) {
			t.Fatalf("missing map fallback: %v", e)
		}
	}
	tool, e := resolver.ResolveTool(ctx, "TEST_ADVANCED", rule)
	if e != nil || tool.BasePowerUnits != 250 || tool.Kind != miningpower.SyntheticKind {
		t.Fatalf("synthetic tool=%+v %v", tool, e)
	}
	m, e := resolver.ResolveMap(ctx, "TEST_MAP", rule)
	if e != nil || m.ModifierScaled != miningpower.Scale || m.Kind != miningpower.SyntheticKind {
		t.Fatalf("synthetic map=%+v %v", m, e)
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	session := g20Session(t, s, i)
	for _, field := range []string{"ToolReference", "MapReference", "BlockInstanceID", "PlayerID", "AccountID", "RuleVersion"} {
		t.Run(field, func(t *testing.T) {
			raw, _ := json.Marshal(session)
			var data map[string]any
			if e := json.Unmarshal(raw, &data); e != nil {
				t.Fatal(e)
			}
			data["ID"] = "invalid-" + field
			data[field] = ""
			raw, e := json.Marshal(data)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO mining_power_sessions(data) VALUES($1)`, string(raw)); e == nil {
				t.Fatal("invalid session reference accepted")
			}
		})
	}
	for _, tool := range []miningpower.ToolProfile{{Reference: "NEG", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: -1, EfficiencyScaled: 1}, {Reference: "ZERO", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: 1, EfficiencyScaled: 0}, {Reference: "HIGH", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: 1, EfficiencyScaled: miningpower.MaxMultiplier + 1}, {Reference: "REAL", RuleVersion: rule, Kind: "PRODUCTION", BasePowerUnits: 1, EfficiencyScaled: 1}} {
		raw, e := json.Marshal(tool)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.pool.Exec(ctx, `INSERT INTO mining_power_tool_profiles(data) VALUES($1)`, string(raw)); e == nil {
			t.Fatal("invalid catalog bounds/kind accepted")
		}
	}
	for _, m := range []miningpower.MapProfile{{Reference: "ZERO", RuleVersion: rule, Kind: miningpower.SyntheticKind, ModifierScaled: 0}, {Reference: "HIGH", RuleVersion: rule, Kind: miningpower.SyntheticKind, ModifierScaled: miningpower.MaxMultiplier + 1}, {Reference: "REAL", RuleVersion: rule, Kind: "PRODUCTION", ModifierScaled: miningpower.Scale}} {
		raw, e := json.Marshal(m)
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.pool.Exec(ctx, `INSERT INTO mining_power_map_profiles(data) VALUES($1)`, string(raw)); e == nil {
			t.Fatal("invalid map bounds/kind accepted")
		}
	}
	a, b := g20Counts(t, s)
	if a != 0 || b != 0 {
		t.Fatal("profile rejection credited power")
	}
}

func TestG20ImmutableHistoryAndUnique(t *testing.T) {
	s, p, i := g20Fixture(t)
	a := g20Accept(t, s, p, i)
	ctx := context.Background()
	for _, table := range []string{"mining_power_rules", "mining_power_tool_profiles", "mining_power_map_profiles", "mining_power_source_events", "mining_power_activities"} {
		for _, operation := range []string{"UPDATE " + table + " SET data=data", "DELETE FROM " + table, "TRUNCATE " + table + " CASCADE"} {
			t.Run(operation, func(t *testing.T) {
				if _, e := s.pool.Exec(ctx, operation); e == nil {
					t.Fatal("immutable history mutation accepted")
				}
			})
		}
	}
	raw, e := json.Marshal(a)
	if e != nil {
		t.Fatal(e)
	}
	_, e = s.pool.Exec(ctx, `INSERT INTO mining_power_activities(data) VALUES($1)`, string(raw))
	var pg *pgconn.PgError
	if !errors.As(e, &pg) || pg.Code != "23505" {
		t.Fatalf("DB global uniqueness=%v", e)
	}
	for _, operation := range []string{`UPDATE mining_power_sessions SET data=jsonb_set(data,'{PlayerID}','"other"')`, `DELETE FROM mining_power_sessions`, `TRUNCATE mining_power_sessions CASCADE`} {
		if _, e = s.pool.Exec(ctx, operation); e == nil {
			t.Fatal("session binding/history mutation accepted")
		}
	}
	r, e := s.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != "PASS" {
		t.Fatalf("guards changed history: %+v %v", r, e)
	}
}

func TestG20RetryPolicy(t *testing.T) {
	for _, code := range []string{"40001", "40P01", "23505", "23514", "08006", "exhaustion", "cancelled"} {
		t.Run(code, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			attempts := 0
			failures := 0
			s.miningPowerFailureInjector = func(stage string) error {
				if stage == "after_begin" {
					attempts++
				}
				if stage != "after_activity" {
					return nil
				}
				failures++
				if code == "cancelled" {
					cancel()
					return ctx.Err()
				}
				if code == "exhaustion" {
					return &pgconn.PgError{Code: "40001"}
				}
				if (code == "40001" || code == "40P01") && failures > 2 {
					return nil
				}
				return &pgconn.PgError{Code: code}
			}
			r, e := s.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
			if code == "40001" || code == "40P01" {
				if e != nil || r.Status != miningpower.StatusValid || attempts != 3 {
					t.Fatalf("retry=%+v %v attempts=%d", r, e, attempts)
				}
				a, b := g20Counts(t, s)
				if a != 1 || b != 1 {
					t.Fatal("retry doubled/partially committed")
				}
			} else {
				if e == nil || r.AppliedPower != 0 {
					t.Fatalf("silent success=%+v %v", r, e)
				}
				want := 1
				if code == "exhaustion" {
					want = 5
					if !errors.Is(e, miningpower.ErrRetryExhausted) {
						t.Fatalf("not explicit exhaustion: %v", e)
					}
				}
				if code == "cancelled" && !errors.Is(e, context.Canceled) {
					t.Fatalf("cancellation ignored: %v", e)
				}
				if attempts != want {
					t.Fatalf("attempts=%d want=%d", attempts, want)
				}
				a, b := g20Counts(t, s)
				if a != 0 || b != 0 {
					t.Fatal("failed retry leaked writes")
				}
			}
		})
	}
}

func TestG20TransactionRollback(t *testing.T) {
	for _, stage := range []string{"before_activity", "after_activity", "after_participant", "before_commit"} {
		t.Run(stage, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			sentinel := errors.New("synthetic transaction failure")
			s.miningPowerFailureInjector = func(got string) error {
				if got == stage {
					return sentinel
				}
				return nil
			}
			if _, e := s.ValidateMiningActivity(context.Background(), p, i, miningpower.DevelopmentRuleVersion); !errors.Is(e, sentinel) {
				t.Fatalf("fault missing: %v", e)
			}
			a, b := g20Counts(t, s)
			if a != 0 || b != 0 {
				t.Fatalf("partial commit %d/%d", a, b)
			}
			s.miningPowerFailureInjector = nil
			g20Accept(t, s, p, i)
		})
	}
}

func TestG20ConcurrentValidation(t *testing.T) {
	for _, distinct := range []bool{false, true} {
		t.Run(fmt.Sprint("distinct=", distinct), func(t *testing.T) {
			s, p, i := g20Fixture(t)
			n := 100
			if distinct {
				n = 16
			}
			intents := make([]miningpower.ActionIntent, n)
			for k := range intents {
				intents[k] = i
				if distinct {
					intents[k] = g20Event(t, s, i, fmt.Sprintf("distinct-%d", k))
				}
			}
			start := make(chan struct{})
			results := make(chan miningpower.ValidationResult, n)
			errs := make(chan error, n)
			var wg sync.WaitGroup
			for _, intent := range intents {
				wg.Add(1)
				go func(intent miningpower.ActionIntent) {
					defer wg.Done()
					<-start
					r, e := s.ValidateMiningActivity(context.Background(), p, intent, miningpower.DevelopmentRuleVersion)
					results <- r
					errs <- e
				}(intent)
			}
			close(start)
			wg.Wait()
			close(results)
			close(errs)
			for e := range errs {
				if e != nil {
					t.Fatal(e)
				}
			}
			accepted, duplicate := 0, 0
			for r := range results {
				switch r.Status {
				case miningpower.StatusValid:
					accepted++
				case miningpower.StatusDuplicate:
					duplicate++
					if r.AppliedPower != 0 {
						t.Fatal("duplicate delta")
					}
				default:
					t.Fatalf("unexpected result=%+v", r)
				}
			}
			want := 1
			if distinct {
				want = n
			}
			if accepted != want || duplicate != n-want {
				t.Fatalf("statuses accepted=%d duplicate=%d", accepted, duplicate)
			}
			snap, e := s.SnapshotMiningPower(context.Background(), i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
			if e != nil || len(snap.Activities) != want || len(snap.Participants) != 1 || snap.Participants[0].ActivityCount != int64(want) || snap.Participants[0].ValidatedPower != int64(want*100) {
				t.Fatalf("aggregate=%+v %v", snap, e)
			}
		})
	}
}

func TestG20AggregateBounds(t *testing.T) {
	for _, field := range []string{"validated_power", "activity_count"} {
		t.Run(field, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			g20Accept(t, s, p, i)
			next := g20Event(t, s, i, "overflow-next")
			if _, e := s.pool.Exec(context.Background(), `UPDATE mining_power_participants SET `+field+`=$1`, int64(math.MaxInt64)); e != nil {
				t.Fatal(e)
			}
			if _, e := s.ValidateMiningActivity(context.Background(), p, next, miningpower.DevelopmentRuleVersion); !errors.Is(e, miningpower.ErrOverflow) {
				t.Fatalf("aggregate overflow=%v", e)
			}
			a, b := g20Counts(t, s)
			if a != 1 || b != 1 {
				t.Fatal("overflow leaked new activity")
			}
		})
	}
}

func TestG20G18NoLock(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	var fk int
	if e := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_constraint c JOIN pg_class child ON child.oid=c.conrelid JOIN pg_class parent ON parent.oid=c.confrelid WHERE c.contype='f' AND child.relname LIKE 'mining_power_%' AND parent.relname IN ('mining_blocks','mining_block_reservations','mining_block_entries','mining_block_receipts')`).Scan(&fk); e != nil || fk != 0 {
		t.Fatalf("G18 FK=%d %v", fk, e)
	}
	raw, e := os.ReadFile("mining_power_block.go")
	if e != nil {
		t.Fatal(e)
	}
	upper := strings.ToUpper(string(raw))
	for _, forbidden := range []string{"FOR UPDATE", "FOR NO KEY UPDATE", "FOR SHARE", "FOR KEY SHARE", "INSERT INTO", "UPDATE MINING_BLOCKS", "DELETE FROM"} {
		if strings.Contains(upper, forbidden) {
			t.Fatalf("adapter SQL violates boundary: %s", forbidden)
		}
	}
	before := g19Economy(t, s)
	for table := range before {
		if strings.HasPrefix(table, "mining_power_") {
			delete(before, table)
		}
	}
	lock, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer lock.Rollback(ctx)
	var id string
	if e = lock.QueryRow(ctx, `SELECT block_instance_id FROM mining_blocks WHERE block_instance_id=$1 FOR UPDATE`, i.BlockInstanceID).Scan(&id); e != nil {
		t.Fatal(e)
	}
	limited, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	r, e := s.ValidateMiningActivity(limited, p, i, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusValid {
		t.Fatalf("acceptance waited for upstream row lock: %+v %v", r, e)
	}
	if e = lock.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	after := g19Economy(t, s)
	for table := range after {
		if strings.HasPrefix(table, "mining_power_") {
			delete(after, table)
		}
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("G20 acceptance mutated G18/economy")
	}
}

func TestG20RestartSnapshotAndReadOnly(t *testing.T) {
	s, p, i := g20Fixture(t)
	a := g20Accept(t, s, p, i)
	ctx := context.Background()
	before, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil {
		t.Fatal(e)
	}
	reopened, e := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	if _, e = s.pool.Exec(ctx, `UPDATE mining_blocks SET block_height=block_height+7,create_command_id='changed-after-acceptance' WHERE block_instance_id=$1`, i.BlockInstanceID); e != nil {
		t.Fatal(e)
	}
	replay, e := reopened.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
	if e != nil || replay.Status != miningpower.StatusDuplicate || !reflect.DeepEqual(a, *replay.Original) {
		t.Fatalf("restart overwrote snapshot=%+v %v", replay, e)
	}
	for repeat := 0; repeat < 2; repeat++ {
		r, e := reopened.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
		if e != nil || r.Status != "PASS" {
			t.Fatalf("historical report=%+v %v", r, e)
		}
	}
	after, e := reopened.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("DB read/replay rewrote historical snapshot")
	}
	tx, e := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, `UPDATE mining_power_participants SET validated_power=0`); e == nil {
		t.Fatal("READ ONLY allowed repair")
	}
	// Actual aggregate corruption is fixture-only; reports expose it without repair.
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e = s.pool.Exec(ctx, `UPDATE mining_power_participants SET validated_power=125`); e != nil {
		t.Fatal(e)
	}
	bad, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil {
		t.Fatal(e)
	}
	r, e := s.ReconcileMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != "MISMATCH" {
		t.Fatalf("corruption hidden: %+v %v", r, e)
	}
	found := false
	for _, f := range r.Findings {
		if f.Field == "participantPower" && f.Delta != nil && *f.Delta == "25" {
			found = true
		}
	}
	if !found {
		t.Fatal("exact expected/actual/delta absent")
	}
	unchanged, e := s.SnapshotMiningPower(ctx, i.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if e != nil || !reflect.DeepEqual(bad, unchanged) {
		t.Fatal("reconciliation repaired DB")
	}
	rebuilt, e := miningpower.RebuildParticipants(bad)
	if e != nil || len(rebuilt.Participants) != 1 || rebuilt.Participants[0].ValidatedPower != 100 {
		t.Fatalf("DB immutable rebuild=%+v %v", rebuilt, e)
	}
}

func TestG20ProductionPersistenceClosed(t *testing.T) {
	s, p, i := g20Fixture(t)
	r, e := miningpower.NewService(s, miningpower.DevelopmentRuleVersion, true).Validate(context.Background(), p, i)
	if e != nil || r.Status != miningpower.StatusNotEligible || r.ReasonCode != "PRODUCTION_NOT_APPROVED" {
		t.Fatalf("production gate=%+v %v", r, e)
	}
	a, b := g20Counts(t, s)
	if a != 0 || b != 0 {
		t.Fatal("production mode wrote facts")
	}
}
