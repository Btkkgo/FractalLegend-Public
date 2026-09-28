package postgres

import (
	"context"
	"encoding/json"
	"fractallegend/game-server/internal/miningpower"
	"testing"
	"time"
)

func g20Event(t *testing.T, s *Store, i miningpower.ActionIntent, id string) miningpower.ActionIntent {
	t.Helper()
	source, err := miningPowerRead[miningpower.SourceEvent](context.Background(), s.pool, `SELECT data FROM mining_power_source_events WHERE source_event_id=$1`, i.SourceEventID)
	if err != nil {
		t.Fatal(err)
	}
	source.ID = id
	source.ActivityID = "mpa:" + id
	g20Insert(t, s, "mining_power_source_events", source)
	i.SourceEventID = id
	i.ActivityID = source.ActivityID
	return i
}
func g20Counts(t *testing.T, s *Store) (int, int) {
	t.Helper()
	var a, p int
	ctx := context.Background()
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_activities`).Scan(&a); err != nil {
		t.Fatal(err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_participants`).Scan(&p); err != nil {
		t.Fatal(err)
	}
	return a, p
}
func g20Accept(t *testing.T, s *Store, p miningpower.Principal, i miningpower.ActionIntent) miningpower.ValidatedMiningActivity {
	t.Helper()
	r, e := s.ValidateMiningActivity(context.Background(), p, i, miningpower.DevelopmentRuleVersion)
	if e != nil || r.Status != miningpower.StatusValid || r.Original == nil {
		t.Fatalf("accept=%+v %v", r, e)
	}
	return *r.Original
}

func TestG20SessionBindingGuard(t *testing.T) {
	s, _, _ := g20Fixture(t)
	ctx := context.Background()
	if _, e := s.pool.Exec(ctx, `UPDATE mining_power_sessions SET data=jsonb_set(data,'{State}','"CLOSED"')`); e != nil {
		t.Fatal(e)
	}
	if _, e := s.pool.Exec(ctx, `UPDATE mining_power_sessions SET data=jsonb_set(data,'{State}','"ACTIVE"')`); e == nil {
		t.Fatal("closed session was reopened")
	}
}

func TestG20SnapshotRequiredFields(t *testing.T) {
	s, p, i := g20Fixture(t)
	a := g20Accept(t, s, p, i)
	next := g20Event(t, s, i, "g20-next")
	a.ActivityID = next.ActivityID
	a.SourceEventID = next.SourceEventID
	raw, e := json.Marshal(a)
	if e != nil {
		t.Fatal(e)
	}
	var data map[string]any
	if e = json.Unmarshal(raw, &data); e != nil {
		t.Fatal(e)
	}
	snapshot := data["ValidationSnapshot"].(map[string]any)
	snapshot["BlockInstanceID"] = nil
	snapshot["Status"] = nil
	raw, e = json.Marshal(data)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.pool.Exec(context.Background(), `INSERT INTO mining_power_activities(data) VALUES($1)`, string(raw)); e == nil {
		t.Fatal("NULL historical validation identity/state accepted")
	}
}

func g20Insert(t *testing.T, s *Store, table string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	// Table names are closed literals owned by this isolated fixture helper.
	if _, err = s.pool.Exec(context.Background(), `INSERT INTO `+table+`(data) VALUES($1::jsonb)`, string(raw)); err != nil {
		t.Fatal(err)
	}
}
func g20Fixture(t *testing.T) (*Store, miningpower.Principal, miningpower.ActionIntent) {
	t.Helper()
	s, svc := g20Fund(t, 20)
	ctx := context.Background()
	receipt, err := svc.Create(ctx, "g20-fixture")
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.LoadMiningBlock(ctx, receipt.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	g19Seed(t, s, "g20-player")
	rule := miningpower.DevelopmentRuleVersion
	g20Insert(t, s, "mining_power_rules", miningpower.DevelopmentManifest())
	g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: "TEST_BASIC", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: 100, EfficiencyScaled: miningpower.Scale})
	g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: "TEST_ADVANCED", RuleVersion: rule, Kind: miningpower.SyntheticKind, BasePowerUnits: 250, EfficiencyScaled: miningpower.Scale})
	g20Insert(t, s, "mining_power_map_profiles", miningpower.MapProfile{Reference: "TEST_MAP", RuleVersion: rule, Kind: miningpower.SyntheticKind, ModifierScaled: miningpower.Scale})
	now := time.Now().UTC().Truncate(time.Microsecond)
	session := miningpower.ActivitySession{ID: "g20-session", PlayerID: "g20-player", AccountID: "g20-player-account", BlockID: b.ID, BlockInstanceID: b.BlockInstanceID, ToolReference: "TEST_BASIC", MapReference: "TEST_MAP", RuleVersion: rule, State: "ACTIVE", OpenedAt: b.StartedAt, ExpiresAt: b.ScheduledEndAt, CreatedAt: now, UpdatedAt: now}
	g20Insert(t, s, "mining_power_sessions", session)
	source := miningpower.SourceEvent{ID: "g20-event", ActivityID: "mpa:g20-event", PlayerID: session.PlayerID, BlockID: b.ID, BlockInstanceID: b.BlockInstanceID, ActivitySessionID: session.ID, ToolReference: session.ToolReference, MapReference: session.MapReference, RuleVersion: rule, ObservedAt: now, ExpiresAt: b.ScheduledEndAt, ActivityWeightScaled: miningpower.Scale, ServerEligibility: miningpower.StatusValid, EvidenceKind: miningpower.SyntheticEvidence}
	g20Insert(t, s, "mining_power_source_events", source)
	return s, miningpower.Principal{AccountID: session.AccountID, PlayerID: session.PlayerID}, miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID, ActivitySessionID: session.ID, BlockID: b.ID, BlockInstanceID: b.BlockInstanceID}
}

func TestG20ActivityValidation(t *testing.T) {
	s, p, intent := g20Fixture(t)
	repo, ok := any(s).(miningpower.Repository)
	if !ok {
		t.Fatal("G20 durable repository missing")
	}
	result, err := repo.ValidateMiningActivity(context.Background(), p, intent, miningpower.DevelopmentRuleVersion)
	if err != nil || result.Status != miningpower.StatusValid || result.AppliedPower != 100 || result.Original == nil {
		t.Fatalf("acceptance=%+v %v", result, err)
	}
	repeat, err := repo.ValidateMiningActivity(context.Background(), p, intent, miningpower.DevelopmentRuleVersion)
	if err != nil || repeat.Status != miningpower.StatusDuplicate || repeat.AppliedPower != 0 {
		t.Fatalf("replay=%+v %v", repeat, err)
	}
	report, err := repo.ReconcileMiningPower(context.Background(), intent.BlockInstanceID, miningpower.DevelopmentRuleVersion)
	if err != nil || report.Status != "PASS" {
		t.Fatalf("report=%+v %v", report, err)
	}
}

func TestG20SchemaConstraints(t *testing.T) {
	s := integrationStore(t)
	ctx := context.Background()
	if err := s.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN
		('mining_power_rules','mining_power_tool_profiles','mining_power_map_profiles','mining_power_sessions',
		'mining_power_source_events','mining_power_activities','mining_power_participants')`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 7 {
		t.Fatalf("G20 requires seven owned tables, got %d", count)
	}
}

func TestG20AdapterBoundary(t *testing.T) {
	s, svc := g20Fund(t, 20)
	ctx := context.Background()
	receipt, err := svc.Create(ctx, "g20-adapter")
	if err != nil {
		t.Fatal(err)
	}
	block, err := s.LoadMiningBlock(ctx, receipt.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	adapter, ok := any(s).(miningpower.MiningBlockContext)
	if !ok {
		t.Fatal("G18 read-only adapter missing")
	}
	got, err := adapter.ReadBlock(ctx, block.BlockInstanceID)
	if err != nil || got.ID != block.ID {
		t.Fatalf("read adapter=%+v %v", got, err)
	}
}
