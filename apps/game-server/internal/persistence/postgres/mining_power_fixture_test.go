package postgres

import (
	"context"
	"fmt"
	"fractallegend/game-server/internal/contribution"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/ledger"
	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/systemspend"
	"github.com/jackc/pgx/v5"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// G20 fixtures are serial within one isolated test database. Reuse the fully
// migrated schema rather than rebuilding all thirteen versions for every case.
// Only reset preparation bypasses triggers, in a transaction-local setting;
// the tested service/guards run after commit with session_replication_role=origin.
// Migration-specific tests still drop the schema and execute real migration DDL.
func g20FreshStore(t *testing.T) *Store {
	t.Helper()
	url := os.Getenv("FRACTAL_TEST_DATABASE_URL")
	if !strings.Contains(url, "fractal_g9_test") {
		t.Fatal("G20 fixture requires isolated fractal_g9_test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, e := Open(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(s.Close)
	if e = s.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback(ctx)
	rows, e := tx.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations' ORDER BY tablename`)
	if e != nil {
		t.Fatal(e)
	}
	var names []string
	for rows.Next() {
		var n string
		if e = rows.Scan(&n); e != nil {
			t.Fatal(e)
		}
		names = append(names, pgx.Identifier{"public", n}.Sanitize())
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, "TRUNCATE "+strings.Join(names, ",")+" RESTART IDENTITY CASCADE"); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(ctx, `SET LOCAL session_replication_role=origin; INSERT INTO black_iron_emission_pools(pool_id,created_at,updated_at) VALUES('GLOBAL',now(),now()); INSERT INTO black_iron_migration_catalog(singleton) VALUES(true)`); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	var mode string
	if e = s.pool.QueryRow(ctx, `SHOW session_replication_role`).Scan(&mode); e != nil || mode != "origin" {
		t.Fatalf("fixture leaked trigger bypass: %s %v", mode, e)
	}
	return s
}

func g20Fund(t *testing.T, amount int64) (*Store, *miningblock.Service) {
	t.Helper()
	s := g20FreshStore(t)
	ctx := context.Background()
	var sequence atomic.Int64
	options := ledger.Options{Now: func() time.Time { return postgresLedgerNow }, NewID: func(prefix string) string { return fmt.Sprintf("%s-postgres-%d", prefix, sequence.Add(1)) }}
	fb := ledger.NewService(s, options)
	for _, a := range []struct{ id, owner string }{{"pg-fb-a", "pg-player-a"}, {"pg-fb-b", "pg-player-b"}} {
		if _, e := fb.CreatePlayerAccount(ctx, a.id, a.owner); e != nil {
			t.Fatal(e)
		}
	}
	for _, a := range []struct{ id, owner string }{{"pg-fb-system", "pg-system-spend"}, {postgresFixtureLiabilityAccountID, "pg-test-fixture-liability"}} {
		if _, e := fb.CreateSystemAccount(ctx, a.id, a.owner); e != nil {
			t.Fatal(e)
		}
	}
	fixture := postgresLedgerFixture{store: s, service: fb}
	postgresCredit(t, fixture, "pg-fb-a", 10000, "g13-start-a")
	postgresCredit(t, fixture, "pg-fb-b", 10000, "g13-start-b")
	contributions := contribution.NewService(s)
	for _, player := range []string{"pg-player-a", "pg-player-b"} {
		if _, e := contributions.CreateAccount(ctx, player); e != nil {
			t.Fatal(e)
		}
	}
	spend, e := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g18-fund", amount))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID); e != nil {
		t.Fatal(e)
	}
	return s, miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false)
}

func TestG20FixtureResetRetainsGuards(t *testing.T) {
	s, p, i := g20Fixture(t)
	g20Accept(t, s, p, i)
	next, _, _ := g20Fixture(t)
	a, b := g20Counts(t, next)
	if a != 0 || b != 0 {
		t.Fatal("prior fixture facts leaked")
	}
	var n int
	if e := next.pool.QueryRow(context.Background(), `SELECT count(*) FROM schema_migrations`).Scan(&n); e != nil || n != 14 {
		t.Fatalf("schema migration lost: %d %v", n, e)
	}
	if _, e := next.pool.Exec(context.Background(), `UPDATE mining_power_source_events SET data=data`); e == nil {
		t.Fatal("fixture reset disabled immutable guards")
	}
	var reserved int64
	if e := next.pool.QueryRow(context.Background(), `SELECT total_reserved FROM black_iron_emission_pools WHERE pool_id='GLOBAL'`).Scan(&reserved); e != nil || reserved != 10 {
		t.Fatalf("real reservation fixture=%d %v", reserved, e)
	}
}
