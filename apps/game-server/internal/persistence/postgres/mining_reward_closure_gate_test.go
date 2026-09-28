package postgres

import (
	"context"
	"errors"
	"fmt"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type g21GateTrace struct{ queries []string }

func (g *g21GateTrace) TraceQueryStart(ctx context.Context, _ *pgx.Conn, d pgx.TraceQueryStartData) context.Context {
	g.queries = append(g.queries, d.SQL)
	return ctx
}
func (g *g21GateTrace) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestG21ClosureProductionDatabaseGateNoWriter(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx := context.Background()
	source := s.pool.Config().ConnConfig.Database
	cfg := s.pool.Config().Copy()
	s.Close()
	adminCfg := cfg.ConnConfig.Copy()
	adminCfg.Database = "postgres"
	admin, e := pgx.ConnectConfig(ctx, adminCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close(ctx)
	name := fmt.Sprintf("fractal_g21_disabled_%d", time.Now().UnixNano())
	if _, e = admin.Exec(ctx, `CREATE DATABASE `+pgx.Identifier{name}.Sanitize()+` TEMPLATE `+pgx.Identifier{source}.Sanitize()); e != nil {
		t.Fatal(e)
	}
	defer func() {
		if _, e := admin.Exec(ctx, `DROP DATABASE `+pgx.Identifier{name}.Sanitize()); e != nil {
			t.Error(e)
		}
	}()
	cfg.ConnConfig.Database = name
	trace := &g21GateTrace{}
	cfg.ConnConfig.Tracer = trace
	pool, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer pool.Close()
	disabled := &Store{pool: pool}
	before := g21ClosureSnapshot(t, disabled)
	for _, op := range []string{"seal", "preview", "settle"} {
		trace.queries = nil
		switch op {
		case "seal":
			_, e = disabled.SealMiningPowerTEST(ctx, cmd.BlockInstanceID)
		case "preview":
			_, e = disabled.PreviewMiningRewardTEST(ctx, cmd.BlockInstanceID, cmd.ExpectedSealDigest)
		case "settle":
			_, e = disabled.SettleMiningRewardTEST(ctx, cmd)
		}
		if !errors.Is(e, miningpower.ErrUnavailable) {
			t.Fatalf("%s production gate=%v", op, e)
		}
		if len(trace.queries) != 1 || trace.queries[0] != "SELECT current_database()" {
			t.Fatalf("%s reached writer: %v", op, trace.queries)
		}
	}
	if g21ClosureSnapshot(t, disabled) != before {
		t.Fatal("disabled database mutated")
	}
}

func TestG21ClosureNoProductionSettlementRoute(t *testing.T) {
	// The implementation has a TEST database guard, no production service flag.
	// Audit executable call sites rather than treating a flag as writer evidence.
	calls := map[string]bool{"SettleMiningRewardTEST": true, "SettleMiningRewardPayloadTEST": true, "PreviewMiningRewardTEST": true, "SealMiningPowerTEST": true}
	for _, root := range []string{"../../../cmd", "../.."} {
		e := filepath.WalkDir(root, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.Contains(filepath.ToSlash(path), "persistence/postgres/") {
				return nil
			}
			file, e := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if e != nil {
				return e
			}
			ast.Inspect(file, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok {
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok && calls[sel.Sel.Name] {
						t.Errorf("production caller %s in %s", sel.Sel.Name, path)
					}
				}
				return true
			})
			return nil
		})
		if e != nil {
			t.Fatal(e)
		}
	}
}
