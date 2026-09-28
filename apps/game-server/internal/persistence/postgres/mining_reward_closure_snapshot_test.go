package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
)

// Every persisted row, including recovery, command and settlement state, is
// compared. Sorting complete JSON rows makes the oracle independent of scans.
func g21ClosureSnapshot(t *testing.T, s *Store) string {
	t.Helper()
	ctx := context.Background()
	rows, err := s.pool.Query(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	for _, name := range names {
		var raw string
		err = s.pool.QueryRow(ctx, `SELECT COALESCE(jsonb_agg(r ORDER BY r::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(t) r FROM `+pgx.Identifier{"public", name}.Sanitize()+` t) q`).Scan(&raw)
		if err != nil {
			t.Fatal(err)
		}
		out.WriteString(name + ":" + raw + "\n")
	}
	return out.String()
}
