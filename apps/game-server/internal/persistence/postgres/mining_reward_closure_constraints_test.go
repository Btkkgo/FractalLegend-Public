package postgres

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"fractallegend/game-server/internal/recycle"
	"fractallegend/game-server/internal/trade"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func g21ConstraintFixture(t *testing.T) (*Store, MiningRewardSettlementCommand, MiningRewardSettlementReceipt) {
	t.Helper()
	s, p, i := g20Fixture(t)
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	cmd := g21InventoryCommand(t, s, i, "closure-constraint")
	r, e := s.SettleMiningRewardTEST(context.Background(), cmd)
	if e != nil {
		t.Fatal(e)
	}
	return s, cmd, r
}

func TestG21ClosureT101IndependentRewardCreatesNoMigrationReceipt(t *testing.T) {
	s, _, r := g21ConstraintFixture(t)
	var aliases, receipts, assets, linked int
	err := s.pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM black_iron_identity_aliases),
		(SELECT count(*) FROM black_iron_migration_receipts),(SELECT count(*) FROM black_iron_migration_assets),
		(SELECT count(*) FROM mining_reward_issuance_lots l JOIN mining_reward_inventory_projections p USING(issuance_id)
		JOIN character_inventory_items i USING(instance_id) JOIN black_iron_identity_aliases a ON a.definition_id=i.definition_id
		WHERE l.source_type='MINING_REWARD' AND l.status='G21_SETTLED' AND l.material_definition_id=a.definition_id
		AND i.quantity=10 AND p.quantity=10 AND l.quantity=10)`).Scan(&aliases, &receipts, &assets, &linked)
	if err != nil || aliases != 1 || receipts != 0 || assets != 0 || linked != 1 || r.TotalOre != 10 {
		t.Fatalf("independent source=%d/%d/%d/%d %v", aliases, receipts, assets, linked, err)
	}
}

// T103/T104/T106: each ordinary SQL statement must be rejected atomically;
// independent identities avoid confusing a primary-key error with source guards.
func TestG21ClosureT103T104T106SourceAndMovementGuards(t *testing.T) {
	s, cmd, receipt := g21ConstraintFixture(t)
	ctx := context.Background()
	g19Seed(t, s, "closure-different-owner")
	if _, err := s.MigrateBlackIronInventory(ctx, receipt.Grants[0].CharacterID); err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, sql, code string }{
		{"migration-source", `INSERT INTO mining_reward_issuance_lots SELECT 'forged-lot','MIGRATION','forged-source','forged-settlement',block_instance_id,character_id,quantity,material_definition_id,rule_version,created_at,source_digest,status FROM mining_reward_issuance_lots LIMIT 1`, "23514"},
		{"dual-origin", `INSERT INTO black_iron_migration_assets(instance_id,version,character_id,definition_id,legacy_id,quantity) SELECT i.instance_id,'G19_BLACK_IRON_V1',i.character_id,i.definition_id,i.legacy_id,i.quantity FROM character_inventory_items i JOIN mining_reward_inventory_projections p USING(instance_id)`, "23514"},
		{"duplicate-lot-character", `INSERT INTO mining_reward_issuance_lots SELECT 'duplicate-lot',source_type,'duplicate-source',settlement_id,block_instance_id,character_id,quantity,material_definition_id,rule_version,created_at,source_digest,status FROM mining_reward_issuance_lots LIMIT 1`, "23505"},
		{"duplicate-stack", `INSERT INTO mining_reward_inventory_projections SELECT 'other-issuance',instance_id,character_id,block_instance_id,quantity,definition_id,revision_before,revision_after,created_at FROM mining_reward_inventory_projections LIMIT 1`, "23505"},
		{"delete", `DELETE FROM character_inventory_items`, "23514"},
		{"split", `UPDATE character_inventory_items SET quantity=quantity-1`, "23514"},
		{"move-owner", `UPDATE character_inventory_items SET character_id='closure-different-owner'`, "23514"},
		{"move-location", `UPDATE character_inventory_items SET location='EQUIPMENT',equipment_slot='WEAPON'`, "23514"},
		{"change-definition", `UPDATE character_inventory_items SET definition_id='ordinary-material'`, "23514"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := g21ClosureSnapshot(t, s)
			_, err := s.pool.Exec(ctx, tc.sql)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || pg.Code != tc.code {
				t.Fatalf("guard error=%v want SQLSTATE %s", err, tc.code)
			}
			if before != g21ClosureSnapshot(t, s) {
				t.Fatal("guard rejection changed rows")
			}
			replay, e := s.SettleMiningRewardTEST(ctx, cmd)
			if e != nil || !reflect.DeepEqual(replay, receipt) {
				t.Fatalf("FINAL changed=%+v %v", replay, e)
			}
		})
	}
}

// T105: privileged corruption is fixture preparation only. Audit/replay runs
// with ordinary triggers restored. Audit detects all mismatches; exact replay
// depends only on immutable history. Neither operation repairs current data.
func TestG21ClosureT105CorruptFinalProjectionReadOnlyDetection(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"missing-lot", `DELETE FROM mining_reward_issuance_lots`},
		{"missing-projection", `DELETE FROM mining_reward_inventory_projections`},
		{"missing-stack", `DELETE FROM character_inventory_items`},
		{"projection-owner", `UPDATE mining_reward_inventory_projections SET character_id='closure-different-owner'`},
		{"inventory-owner", `UPDATE character_inventory_items SET character_id='closure-different-owner'`},
		{"quantity", `UPDATE character_inventory_items SET quantity=quantity+1`},
		{"definition", `UPDATE character_inventory_items SET definition_id='forged-definition'`},
		{"item-revision", `UPDATE character_inventory_items SET item_revision=2`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, cmd, original := g21ConstraintFixture(t)
			ctx := context.Background()
			g19Seed(t, s, "closure-different-owner")
			tx, err := s.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec(ctx, tc.sql); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			before := g21ClosureSnapshot(t, s)
			report, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
			if err != nil || report.Balanced || len(report.Mismatches) == 0 {
				t.Errorf("audit accepted corrupt %s: %+v %v", tc.name, report, err)
			}
			if tc.name == "missing-lot" || tc.name == "missing-projection" || tc.name == "projection-owner" {
				if _, err = s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardInvariant) {
					t.Errorf("replay accepted immutable history corruption: %v", err)
				}
			} else {
				g21T044Exact(t, s, cmd, original)
			}
			if before != g21ClosureSnapshot(t, s) {
				t.Fatal("audit/replay repaired corrupt data")
			}
		})
	}
}

func TestG21ClosureT103RewardOriginCannotClaimMigratedStack(t *testing.T) {
	s, p, i := g21LegacyRewardFixture(t)
	ctx := context.Background()
	if _, err := s.MigrateBlackIronInventory(ctx, p.PlayerID); err != nil {
		t.Fatal(err)
	}
	cmd := g21InventoryCommand(t, s, i, "closure-other-origin")
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	_, err := s.pool.Exec(ctx, `INSERT INTO mining_reward_issuance_lots(issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,quantity,material_definition_id,rule_version,created_at,source_digest,status)
		SELECT 'forged-p0-lot','MINING_REWARD','forged-p0-source','forged-final-settlement',block_instance_id,character_id,7,material_definition_id,'G21_TEST_ISSUANCE_V1',created_at,source_digest,'G21_SETTLED' FROM mining_reward_issuance_lots LIMIT 1;
		INSERT INTO mining_reward_inventory_projections(issuance_id,instance_id,character_id,block_instance_id,quantity,definition_id,revision_before,revision_after,created_at)
		SELECT 'forged-p0-lot','closure-legacy-bun',character_id,block_instance_id,7,material_definition_id,1,2,created_at FROM mining_reward_issuance_lots WHERE issuance_id='forged-p0-lot'`)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23514" {
		t.Fatalf("reverse dual source=%v", err)
	}
	if before != g21ClosureSnapshot(t, s) {
		t.Fatal("rejected reverse source insertion changed rows")
	}
}

func TestG21ClosureT106TradeAndRecycleCannotConsumeReward(t *testing.T) {
	for _, amount := range []int{3, 10} {
		t.Run(fmt.Sprintf("trade-%d", amount), func(t *testing.T) {
			s, cmd, reward := g21ConstraintFixture(t)
			ctx := context.Background()
			owner := reward.Grants[0].CharacterID
			other := "closure-trade-other"
			g19Seed(t, s, other)
			a, err := s.LoadCharacter(ctx, owner)
			if err != nil {
				t.Fatal(err)
			}
			b, err := s.LoadCharacter(ctx, other)
			if err != nil {
				t.Fatal(err)
			}
			var sequence int
			svc := trade.NewService(s, trade.Options{Now: func() time.Time { return postgresTradeNow }, NewID: func(prefix string) string { sequence++; return fmt.Sprintf("%s-closure-%d", prefix, sequence) }, InventoryCapacity: 20})
			state, err := svc.CreateTrade(ctx, "closure-trade", owner, other, postgresTradeNow.Add(time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			state, err = svc.AddItem(ctx, state.TradeID, owner, reward.Grants[0].InventoryInstanceID, amount, state.Revision)
			if err != nil {
				t.Fatal(err)
			}
			state, err = svc.ConfirmTrade(ctx, state.TradeID, owner, state.Revision)
			if err != nil {
				t.Fatal(err)
			}
			state, err = svc.ConfirmTrade(ctx, state.TradeID, other, state.Revision)
			if err != nil {
				t.Fatal(err)
			}
			economy := g19Economy(t, s)
			if _, err = svc.FinalizeTrade(ctx, state.TradeID); !errors.Is(err, trade.ErrSettlementFailed) {
				t.Fatalf("reward trade=%v", err)
			}
			afterA, ea := s.LoadCharacter(ctx, owner)
			afterB, eb := s.LoadCharacter(ctx, other)
			if ea != nil || eb != nil || !reflect.DeepEqual(a, afterA) || !reflect.DeepEqual(b, afterB) || !reflect.DeepEqual(economy, g19Economy(t, s)) {
				t.Fatal("failed trade changed inventory/economy")
			}
			if r, e := s.SettleMiningRewardTEST(ctx, cmd); e != nil || !reflect.DeepEqual(r, reward) {
				t.Fatalf("trade changed FINAL=%+v %v", r, e)
			}
		})
	}
	t.Run("recycle", func(t *testing.T) {
		s, _, r := g21ConstraintFixture(t)
		ctx := context.Background()
		rule := g16Rule()
		rule.TemplateID = "synthetic-bun-material"
		registry, err := recycle.NewRegistry([]recycle.Rule{rule})
		if err != nil {
			t.Fatal(err)
		}
		intent := g16Intent(r.Grants[0].InventoryInstanceID, "closure-recycle")
		intent.PlayerID = r.Grants[0].CharacterID
		before := g21ClosureSnapshot(t, s)
		if _, err = recycle.NewService(s, registry).Recycle(ctx, intent); !errors.Is(err, recycle.ErrLocked) {
			t.Fatalf("reward recycle=%v", err)
		}
		if before != g21ClosureSnapshot(t, s) {
			t.Fatal("rejected recycle changed rows")
		}
	})
}

func TestG21ClosureT103P0AndFinalOriginsAreExclusive(t *testing.T) {
	t.Run("P0-after-FINAL-helper", func(t *testing.T) {
		s, cmd, r := g21ConstraintFixture(t)
		before := g21ClosureSnapshot(t, s)
		if _, err := s.createMiningRewardIssuancePrerequisiteTEST(context.Background(), "closure-forbidden-p0", cmd.BlockInstanceID, r.Grants[0].CharacterID, 1); err == nil {
			t.Fatal("P0 helper issued after FINAL")
		}
		if before != g21ClosureSnapshot(t, s) {
			t.Fatal("rejected P0 changed FINAL/economy")
		}
	})
	t.Run("P0-after-FINAL-SQL", func(t *testing.T) {
		s, _, _ := g21ConstraintFixture(t)
		before := g21ClosureSnapshot(t, s)
		_, err := s.pool.Exec(context.Background(), `INSERT INTO mining_reward_issuance_lots SELECT 'extra-p0-lot','MINING_REWARD','extra-p0-source','G21_P0_TEST_PLACEHOLDER',block_instance_id,character_id,1,material_definition_id,'G21_P0_ISSUANCE_V1',created_at,source_digest,'P0_SYNTHETIC_ONLY' FROM mining_reward_issuance_lots LIMIT 1`)
		if err == nil {
			t.Fatal("direct SQL P0 origin accepted after FINAL")
		}
		if before != g21ClosureSnapshot(t, s) {
			t.Fatal("rejected SQL origin changed rows")
		}
	})
	t.Run("FINAL-after-P0", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		g19Catalog(t, s)
		g20Accept(t, s, p, i)
		cmd := g21InventoryCommand(t, s, i, "closure-p0-first")
		if _, err := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "closure-p0-first", i.BlockInstanceID, p.PlayerID, 1); err != nil {
			t.Fatal(err)
		}
		before := g21ClosureSnapshot(t, s)
		if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardConflict) {
			t.Fatalf("FINAL after P0=%v", err)
		}
		_, err := s.pool.Exec(ctx, `INSERT INTO mining_reward_issuance_lots SELECT 'extra-final-lot','MINING_REWARD','extra-final-source','extra-final-settlement',block_instance_id,character_id,10,material_definition_id,'G21_TEST_ISSUANCE_V1',created_at,source_digest,'G21_SETTLED' FROM mining_reward_issuance_lots LIMIT 1`)
		if err == nil {
			t.Fatal("direct SQL FINAL origin accepted after P0")
		}
		if before != g21ClosureSnapshot(t, s) {
			t.Fatal("rejected FINAL changed rows")
		}
	})
}

func TestG21ClosureT107HistoryMutationsPreserveAllRows(t *testing.T) {
	s, _, _ := g21ConstraintFixture(t)
	ctx := context.Background()
	if _, err := s.pool.Exec(ctx, `UPDATE black_iron_migration_catalog SET frozen=true`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ table, column string }{
		{"mining_power_input_seals", "sealed_at"}, {"mining_reservation_instance_bindings", "reservation_amount"}, {"mining_power_beneficiary_bindings", "bound_at"},
		{"mining_reward_commands", "created_at"}, {"mining_reward_settlement_receipts", "created_at"}, {"mining_reward_grants", "created_at"},
		{"mining_reward_issuance_lots", "created_at"}, {"mining_reward_inventory_projections", "created_at"}, {"mining_reward_reservation_consumptions", "created_at"},
		{"black_iron_identity_aliases", "definition_id"}, {"black_iron_migration_catalog", "frozen"}, {"black_iron_emission_recovery_entries", "created_at"},
		{"mining_reward_origin_modes", "origin_mode"},
	} {
		t.Run(tc.table, func(t *testing.T) {
			update := fmt.Sprintf("UPDATE %s SET %s=%s", tc.table, tc.column, tc.column)
			if tc.table == "black_iron_migration_catalog" {
				update = "UPDATE black_iron_migration_catalog SET frozen=false"
			}
			if tc.table == "mining_reward_origin_modes" {
				update = "UPDATE mining_reward_origin_modes SET origin_mode='P0_SYNTHETIC_ONLY'"
			}
			for _, sql := range []string{update, "DELETE FROM " + tc.table, "TRUNCATE " + tc.table + " CASCADE"} {
				before := g21ClosureSnapshot(t, s)
				if _, err := s.pool.Exec(ctx, sql); err == nil {
					t.Fatalf("history mutation succeeded: %s", sql)
				}
				if before != g21ClosureSnapshot(t, s) {
					t.Fatalf("history changed: %s", sql)
				}
			}
		})
	}
}

func TestG21ClosureT103ConcurrentOriginsRejectStaleSQLWriter(t *testing.T) {
	for _, first := range []string{"P0", "FINAL"} {
		for _, isolation := range []pgx.TxIsoLevel{pgx.ReadCommitted, pgx.RepeatableRead} {
			t.Run(first+"/"+string(isolation), func(t *testing.T) {
				s, p, i := g20Fixture(t)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				g19Catalog(t, s)
				g20Accept(t, s, p, i)
				cmd := g21InventoryCommand(t, s, i, "closure-origin-race")
				tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: isolation})
				if err != nil {
					t.Fatal(err)
				}
				defer tx.Rollback(ctx)
				var count int
				if err = tx.QueryRow(ctx, `SELECT count(*) FROM mining_reward_origin_modes`).Scan(&count); err != nil || count != 0 {
					t.Fatalf("initial origin snapshot=%d %v", count, err)
				}
				held, release := make(chan struct{}), make(chan struct{})
				firstResult := make(chan error, 1)
				if first == "P0" {
					s.prerequisiteFailureInjector = func(at string) error {
						if at == "issuance_before_commit" {
							close(held)
							<-release
						}
						return nil
					}
					go func() {
						_, e := s.createMiningRewardIssuancePrerequisiteTEST(ctx, "closure-race-p0", i.BlockInstanceID, p.PlayerID, 1)
						firstResult <- e
					}()
				} else {
					s.settlementFailureInjector = func(at string) error {
						if at == "before_commit" {
							close(held)
							<-release
						}
						return nil
					}
					go func() { _, e := s.SettleMiningRewardTEST(ctx, cmd); firstResult <- e }()
				}
				select {
				case <-held:
				case <-ctx.Done():
					t.Fatal("first origin missed commit barrier")
				}
				status, version, settlement := "G21_SETTLED", "G21_TEST_ISSUANCE_V1", "closure-probe-final"
				if first == "FINAL" {
					status, version, settlement = "P0_SYNTHETIC_ONLY", "G21_P0_ISSUANCE_V1", "G21_P0_TEST_PLACEHOLDER"
				}
				probeResult := make(chan error, 1)
				go func() {
					_, e := tx.Exec(ctx, `INSERT INTO mining_reward_issuance_lots(issuance_id,source_type,source_key,settlement_id,block_instance_id,character_id,quantity,material_definition_id,rule_version,created_at,source_digest,status) VALUES('closure-probe-lot','MINING_REWARD','closure-probe-source',$1,$2,$3,1,'synthetic-bun-material',$4,now(),repeat('0',64),$5)`, settlement, i.BlockInstanceID, p.PlayerID, version, status)
					probeResult <- e
				}()
				waiting := false
				for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
					var n int
					if err = s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE 'INSERT INTO mining_reward_issuance_lots%'`).Scan(&n); err != nil {
						break
					}
					if n > 0 {
						waiting = true
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
				close(release)
				firstErr := <-firstResult
				probeErr := <-probeResult
				_ = tx.Rollback(ctx)
				s.prerequisiteFailureInjector = nil
				s.settlementFailureInjector = nil
				if firstErr != nil || !waiting {
					t.Fatalf("first=%v observed SQL conflict=%t", firstErr, waiting)
				}
				var pg *pgconn.PgError
				want := "23514"
				if isolation == pgx.RepeatableRead {
					want = "40001"
				}
				if !errors.As(probeErr, &pg) || pg.Code != want {
					t.Fatalf("competing origin=%v want %s", probeErr, want)
				}
				var modes, lots int
				if err = s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_origin_modes),(SELECT count(*) FROM mining_reward_issuance_lots)`).Scan(&modes, &lots); err != nil || modes != 1 || lots != 1 {
					t.Fatalf("origin/lot counts=%d/%d %v", modes, lots, err)
				}
				if r, e := s.ReconcileMiningRewardIssuancePrerequisite(ctx); e != nil || !r.Balanced {
					t.Fatalf("winner audit=%+v %v", r, e)
				}
			})
		}
	}
}
