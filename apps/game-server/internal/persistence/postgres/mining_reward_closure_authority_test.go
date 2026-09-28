package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningblock"
	"fractallegend/game-server/internal/miningpower"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Corruption is introduced only by the fixture owner in a bounded transaction.
// Tested calls always run with normal triggers and cannot alter their evidence.
func g21AuthorityCorrupt(t *testing.T, s *Store, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL session_replication_role=replica`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, sql, args...); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}
func g21AuthoritySealed(t *testing.T) (*Store, miningpower.Principal, miningpower.ActionIntent, MiningRewardSettlementCommand) {
	t.Helper()
	s, p, i := g20Fixture(t)
	g19Catalog(t, s)
	g20Accept(t, s, p, i)
	if _, err := s.FinalizeMiningBlock(context.Background(), i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(context.Background(), i.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	return s, p, i, MiningRewardSettlementCommand{CommandID: "g21-authority", BlockInstanceID: i.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
}
func g21AuthorityUnchanged(t *testing.T, s *Store, before string) {
	t.Helper()
	if after := g21ClosureSnapshot(t, s); after != before {
		t.Fatal("rejected operation changed persisted table rows")
	}
}

func TestG21ClosureAuthorityBindingAndHistoricalUnbound(t *testing.T) {
	t.Run("T021_authoritative_binding", func(t *testing.T) {
		s, p, i, cmd := g21AuthoritySealed(t)
		ctx := context.Background()
		var account, player, character string
		if err := s.pool.QueryRow(ctx, `SELECT account_id,player_id,character_id FROM mining_power_beneficiary_bindings WHERE activity_id=$1`, i.ActivityID).Scan(&account, &player, &character); err != nil {
			t.Fatal(err)
		}
		r, err := s.SettleMiningRewardTEST(ctx, cmd)
		if err != nil || len(r.Grants) != 1 || r.TotalOre != 10 {
			t.Fatalf("receipt=%+v %v", r, err)
		}
		g := r.Grants[0]
		if account != p.AccountID || player != p.PlayerID || character != p.PlayerID || g.AccountID != account || g.PlayerID != player || g.CharacterID != character || g.Quantity != 10 {
			t.Fatalf("lost authority: %+v", g)
		}
	})
	t.Run("T022_historical_unbound", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		g19Catalog(t, s)
		g20Accept(t, s, p, i)
		g21AuthorityCorrupt(t, s, `DELETE FROM mining_power_beneficiary_bindings`)
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			t.Fatal(err)
		}
		seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
		if err != nil || seal.Eligibility != "NOT_SETTLEMENT_ELIGIBLE" {
			t.Fatalf("seal=%+v %v", seal, err)
		}
		before := g21ClosureSnapshot(t, s)
		_, err = s.SettleMiningRewardTEST(ctx, MiningRewardSettlementCommand{"g21-unbound", i.BlockInstanceID, seal.CanonicalDigest})
		if !errors.Is(err, ErrMiningRewardInvariant) {
			t.Fatalf("unbound payout=%v", err)
		}
		g21AuthorityUnchanged(t, s, before)
	})
	t.Run("T023_wrong_account", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		p.AccountID = "forged-account"
		before := g21ClosureSnapshot(t, s)
		r, err := s.ValidateMiningActivity(context.Background(), p, i, miningpower.DevelopmentRuleVersion)
		if err != nil || r.ReasonCode != "PLAYER_OWNER_MISMATCH" || r.Original != nil {
			t.Fatalf("wrong account=%+v %v", r, err)
		}
		g21AuthorityUnchanged(t, s, before)
	})
	t.Run("T030_owner_changed_after_seal", func(t *testing.T) {
		s, _, _, cmd := g21AuthoritySealed(t)
		g21AuthorityCorrupt(t, s, `UPDATE characters SET account_id='changed-owner' WHERE id='g20-player'`)
		before := g21ClosureSnapshot(t, s)
		_, err := s.SettleMiningRewardTEST(context.Background(), cmd)
		if !errors.Is(err, ErrMiningRewardInvariant) && !errors.Is(err, ErrMiningRewardConflict) {
			t.Fatalf("changed owner accepted: %v", err)
		}
		g21AuthorityUnchanged(t, s, before)
	})
}

func TestG21ClosureAuthorityMergedSessionsDistinctCharacters(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		powerA, powerA2, powerB int64
	}{{"T025_sessions", 2, 3, 5}, {"T027_same_account_characters", 1, 0, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			session := g20Session(t, s, i)
			source := g20Source(t, s, i)
			add := func(id, player string, power int64) {
				tool := "tool-" + id
				g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: tool, RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, BasePowerUnits: power, EfficiencyScaled: miningpower.Scale})
				ss := session
				ss.ID = id + "-session"
				ss.PlayerID = player
				ss.ToolReference = tool
				g20Insert(t, s, "mining_power_sessions", ss)
				src := source
				src.ID = id + "-event"
				src.ActivityID = "mpa:" + src.ID
				src.PlayerID = player
				src.ActivitySessionID = ss.ID
				src.ToolReference = tool
				in := g20RegisteredSource(t, s, src)
				g20Accept(t, s, miningpower.Principal{AccountID: p.AccountID, PlayerID: player}, in)
			}
			add("a1", p.PlayerID, tc.powerA)
			if tc.powerA2 > 0 {
				add("a2", p.PlayerID, tc.powerA2)
			}
			g19Seed(t, s, "other-character")
			if _, err := s.pool.Exec(ctx, `UPDATE characters SET account_id=$1 WHERE id='other-character'`, p.AccountID); err != nil {
				t.Fatal(err)
			}
			add("b", "other-character", tc.powerB)
			if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
				t.Fatal(err)
			}
			seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.SettleMiningRewardTEST(ctx, MiningRewardSettlementCommand{"g21-multi", i.BlockInstanceID, seal.CanonicalDigest})
			if err != nil {
				t.Fatal(err)
			}
			if len(r.Grants) != 2 || r.TotalOre != 10 || r.PositiveParticipantCount != 2 {
				t.Fatalf("multi=%+v", r)
			}
			for _, g := range r.Grants {
				if g.Quantity != 5 || g.AccountID != p.AccountID || g.IssuanceID == "" {
					t.Fatalf("bad grant %+v", g)
				}
			}
			if r.Grants[0].CharacterID == r.Grants[1].CharacterID || r.Grants[0].IssuanceID == r.Grants[1].IssuanceID {
				t.Fatal("merged independent characters")
			}
			if tc.powerA2 > 0 {
				for _, w := range seal.ParticipantWeights {
					if w.CharacterID == p.PlayerID && (w.Power != 5 || w.ActivityCount != 2) {
						t.Fatalf("sessions not merged %+v", w)
					}
				}
			}
			// Real G18 currently authorizes R10 only. T027's literal R2 is separately
			// corrected to real R10 in the evidence ledger; ownership is unchanged.
		})
	}
}

func TestG21ClosureAuthorityCorruptAcceptance(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"T029_forged_character", `UPDATE mining_power_beneficiary_bindings SET character_id='forged-character'`},
		{"T031_summary", `UPDATE mining_power_participants SET validated_power=validated_power+1`},
		{"T032_mixed_rule", `UPDATE mining_power_activities SET data=jsonb_set(jsonb_set(data,'{RuleVersion}','"UNAPPROVED_RULE"'),'{ValidationSnapshot,G20RuleVersion}','"UNAPPROVED_RULE"') WHERE source_event_id='second-source'`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			g20Accept(t, s, p, i)
			if tc.name == "T032_mixed_rule" {
				g20Accept(t, s, p, g20Event(t, s, i, "second-source"))
			}
			if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
				t.Fatal(err)
			}
			g21AuthorityCorrupt(t, s, tc.sql)
			before := g21ClosureSnapshot(t, s)
			_, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
			if !errors.Is(err, miningpower.ErrInvariant) {
				t.Fatalf("corrupt evidence sealed: %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	t.Run("T028_aliases_one_character", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		g20Accept(t, s, p, i)
		ss := g20Session(t, s, i)
		src := g20Source(t, s, i)
		g19Seed(t, s, "alias-player")
		if _, err := s.pool.Exec(ctx, `UPDATE characters SET account_id=$1 WHERE id='alias-player'`, p.AccountID); err != nil {
			t.Fatal(err)
		}
		ss.ID = "alias-session"
		ss.PlayerID = "alias-player"
		g20Insert(t, s, "mining_power_sessions", ss)
		src.ID = "alias-event"
		src.ActivityID = "mpa:alias-event"
		src.PlayerID = ss.PlayerID
		src.ActivitySessionID = ss.ID
		other := g20RegisteredSource(t, s, src)
		g20Accept(t, s, miningpower.Principal{AccountID: p.AccountID, PlayerID: ss.PlayerID}, other)
		var e miningpower.BeneficiaryEvidence
		err := s.pool.QueryRow(ctx, `SELECT binding_version,activity_id,source_event_id,account_id,player_id,character_id,block_instance_id,authority_source,bound_at FROM mining_power_beneficiary_bindings WHERE activity_id=$1`, other.ActivityID).Scan(&e.Version, &e.ActivityID, &e.SourceEventID, &e.AccountID, &e.PlayerID, &e.CharacterID, &e.BlockInstanceID, &e.AuthoritySource, &e.BoundAt)
		if err != nil {
			t.Fatal(err)
		}
		e.CharacterID = p.PlayerID
		digest, err := miningpower.BeneficiaryEvidenceDigest(e)
		if err != nil {
			t.Fatal(err)
		}
		g21AuthorityCorrupt(t, s, `UPDATE mining_power_beneficiary_bindings SET character_id=$1,digest=$2 WHERE activity_id=$3`, e.CharacterID, digest, e.ActivityID)
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			t.Fatal(err)
		}
		before := g21ClosureSnapshot(t, s)
		if _, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); !errors.Is(err, miningpower.ErrInvariant) {
			t.Fatalf("alias collision=%v", err)
		}
		g21AuthorityUnchanged(t, s, before)
	})
}

func TestG21ClosureAuthorityReplayConflictsAndRestart(t *testing.T) {
	s, p, i, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	before := g21ClosureSnapshot(t, s)
	for _, field := range []string{"session", "instance", "rule"} {
		t.Run("T026_T042_"+field, func(t *testing.T) {
			altered := i
			rule := miningpower.DevelopmentRuleVersion
			switch field {
			case "session":
				altered.ActivitySessionID = "different-session"
			case "instance":
				altered.BlockInstanceID = "mining-block-instance-11111111111111111111111111111111"
			case "rule":
				rule = "FUTURE_RULE"
			}
			r, err := s.ValidateMiningActivity(ctx, p, altered, rule)
			if err != nil || r.ReasonCode != "BINDING_CHANGED" || r.Original != nil || r.AppliedPower != 0 {
				t.Fatalf("replay=%+v %v", r, err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	t.Run("T044_restart_historical_rule", func(t *testing.T) {
		other, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
		if err != nil {
			t.Fatal(err)
		}
		defer other.Close()
		first, err := other.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
		if err != nil || first.Status != miningpower.StatusDuplicate || first.Original == nil {
			t.Fatalf("historical replay=%+v %v", first, err)
		}
		future, err := other.ValidateMiningActivity(ctx, p, i, "FUTURE_RULE")
		if err != nil || future.ReasonCode != "BINDING_CHANGED" || future.Original != nil {
			t.Fatalf("future replay=%+v %v", future, err)
		}
		g21AuthorityUnchanged(t, s, before)
	})
	r, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil || r.ActivityCount != 1 || r.TotalOre != 10 || len(r.Grants) != 1 {
		t.Fatalf("source replay credited twice: %+v %v", r, err)
	}
}

func TestG21ClosureAuthoritySealConcurrencyAndLateTransaction(t *testing.T) {
	t.Run("T039_identical_concurrent_seal", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		g20Accept(t, s, p, i)
		ctx := context.Background()
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			t.Fatal(err)
		}
		before := g20Upstream(t, s)
		type result struct {
			seal miningpower.SettlementInputSeal
			err  error
		}
		results := make(chan result, 2)
		var wg sync.WaitGroup
		for k := 0; k < 2; k++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				v, e := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
				results <- result{v, e}
			}()
		}
		wg.Wait()
		close(results)
		var first *miningpower.SettlementInputSeal
		for r := range results {
			if r.err != nil {
				t.Fatal(r.err)
			}
			if first == nil {
				x := r.seal
				first = &x
			} else if !reflect.DeepEqual(*first, r.seal) {
				t.Fatal("different concurrent seal")
			}
		}
		if !reflect.DeepEqual(before, g20Upstream(t, s)) {
			t.Fatal("seal changed economics")
		}
	})
	t.Run("T036_earlier_transaction_late_gate", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		begun := make(chan struct{})
		release := make(chan struct{})
		s.miningPowerFailureInjector = func(stage string) error {
			if stage == "after_begin" {
				close(begun)
				<-release
			}
			return nil
		}
		result := make(chan miningpower.ValidationResult, 1)
		errc := make(chan error, 1)
		go func() {
			r, e := s.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
			result <- r
			errc <- e
		}()
		<-begun
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			close(release)
			t.Fatal(err)
		}
		seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
		if err != nil {
			close(release)
			t.Fatal(err)
		}
		before := g21ClosureSnapshot(t, s)
		close(release)
		r := <-result
		if err := <-errc; err != nil || r.ReasonCode != "BLOCK_SEALED" {
			t.Fatalf("late BEGIN admitted=%+v %v", r, err)
		}
		g21AuthorityUnchanged(t, s, before)
		if seal.ActivityCount != 0 {
			t.Fatal("late fact included")
		}
	})
	t.Run("T048_seal_no_G18_tuple_lock", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		g20Accept(t, s, p, i)
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			t.Fatal(err)
		}
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if _, err = tx.Exec(ctx, `SELECT 1 FROM mining_blocks WHERE block_id=$1 FOR UPDATE`, i.BlockID); err != nil {
			t.Fatal(err)
		}
		limited, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if _, err := s.SealMiningPowerTEST(limited, i.BlockInstanceID); err != nil {
			t.Fatalf("sealer depends on G18 tuple lock: %v", err)
		}
		var locks int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_locks WHERE relation='mining_blocks'::regclass AND locktype='tuple'`).Scan(&locks); err != nil || locks != 0 {
			t.Fatalf("G18 tuple waiter=%d %v", locks, err)
		}
	})
}

func TestG21ClosureAuthorityGateAndUpstreamRejections(t *testing.T) {
	t.Run("T045_lazy_gate_then_missing_history_gate", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		var n int
		ctx := context.Background()
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_acceptance_states`).Scan(&n); err != nil || n != 0 {
			t.Fatalf("gate provisioned early %d %v", n, err)
		}
		a := g20Accept(t, s, p, i)
		g21AuthorityCorrupt(t, s, `DELETE FROM mining_power_acceptance_states`)
		next := g20Event(t, s, i, "missing-gate-source")
		a.ActivityID = next.ActivityID
		a.SourceEventID = next.SourceEventID
		raw, _ := json.Marshal(a)
		before := g21ClosureSnapshot(t, s)
		_, err := s.pool.Exec(ctx, `INSERT INTO mining_power_activities(data) VALUES($1)`, raw)
		g181SQLState(t, err, "55000", "")
		g21AuthorityUnchanged(t, s, before)
	})
	for _, tc := range []struct {
		name, sql string
		sealed    bool
	}{
		{"T052_changed_R", `UPDATE mining_block_reservations SET amount=9`, true},
		{"T058_missing_source", `DELETE FROM mining_block_entries WHERE action='OPEN'`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			g19Catalog(t, s)
			g20Accept(t, s, p, i)
			if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
				t.Fatal(err)
			}
			var cmd MiningRewardSettlementCommand
			if tc.sealed {
				seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
				if err != nil {
					t.Fatal(err)
				}
				cmd = MiningRewardSettlementCommand{"g21-conflict", i.BlockInstanceID, seal.CanonicalDigest}
			}
			g21AuthorityCorrupt(t, s, tc.sql)
			before := g21ClosureSnapshot(t, s)
			var err error
			if tc.sealed {
				_, err = s.SettleMiningRewardTEST(ctx, cmd)
			} else {
				_, err = s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
			}
			if !errors.Is(err, ErrMiningRewardInvariant) && !errors.Is(err, miningpower.ErrInvariant) {
				t.Fatalf("corrupt source accepted %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	for _, cancelled := range []bool{false, true} {
		t.Run(fmt.Sprintf("T055_T056_cancelled_%v", cancelled), func(t *testing.T) {
			s, p, i := g20Fixture(t)
			g20Accept(t, s, p, i)
			ctx := context.Background()
			if cancelled {
				if _, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Cancel(ctx, i.BlockID); err != nil {
					t.Fatal(err)
				}
			}
			before := g21ClosureSnapshot(t, s)
			_, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
			if !errors.Is(err, miningpower.ErrInvalidInput) {
				t.Fatalf("nonfinal sealed: %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	t.Run("T053_T054_unique_binding", func(t *testing.T) {
		s, _, _ := g20Fixture(t)
		ctx := context.Background()
		before := g21ClosureSnapshot(t, s)
		for _, otherInstance := range []bool{false, true} {
			instance := "block_instance_id"
			source := "'different-source'"
			constraint := "mining_reservation_instance_bindings_block_instance_id_key"
			if otherInstance {
				instance = "'mining-block-instance-11111111111111111111111111111111'"
				source = "reservation_source_id"
				constraint = "mining_reservation_instance_bindings_reservation_source_id_key"
			}
			_, err := s.pool.Exec(ctx, `INSERT INTO mining_reservation_instance_bindings SELECT 'duplicate-binding',`+source+`,'different-receipt',`+instance+`,display_block_id,g18_rule_version,reservation_amount,binding_version,evidence_digest,created_at FROM mining_reservation_instance_bindings LIMIT 1`)
			g181SQLState(t, err, "23505", constraint)
			g21AuthorityUnchanged(t, s, before)
		}
	})
}

func TestG21ClosureAuthorityStrictDecoderBeforeWriter(t *testing.T) {
	s, _, i, cmd := g21AuthoritySealed(t)
	valid, _ := json.Marshal(map[string]string{"commandID": cmd.CommandID, "blockInstanceID": i.BlockInstanceID, "expectedSealDigest": cmd.ExpectedSealDigest})
	before := g21ClosureSnapshot(t, s)
	for _, key := range []string{"R", "reward", "weight", "participants", "quantity", "beneficiary", "characterID", "playerID", "accountID", "owner", "unknown"} {
		t.Run("T024_T125_"+key, func(t *testing.T) {
			raw := strings.TrimSuffix(string(valid), "}") + `,"` + key + `":"forged"}`
			_, err := s.SettleMiningRewardPayloadTEST(context.Background(), []byte(raw))
			if !errors.Is(err, miningpower.ErrInvalidInput) {
				t.Fatalf("forbidden field accepted: %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	for _, raw := range []string{strings.TrimSuffix(string(valid), "}") + `,"commandID":"duplicate"}`, strings.TrimSuffix(string(valid), "}") + `,"R":NaN}`, strings.TrimSuffix(string(valid), "}") + `,"R":Infinity}`} {
		if _, err := s.SettleMiningRewardPayloadTEST(context.Background(), []byte(raw)); !errors.Is(err, miningpower.ErrInvalidInput) {
			t.Fatalf("ambiguous payload=%v", err)
		}
		g21AuthorityUnchanged(t, s, before)
	}
}

func TestG21ClosureAuthorityReceiptCorruption(t *testing.T) {
	for _, tc := range []struct{ name, sql string }{
		{"digest", `UPDATE mining_reward_settlement_receipts SET canonical_digest=repeat('0',64)`},
		{"scalar", `UPDATE mining_reward_settlement_receipts SET total_ore=total_ore+1`},
		{"bytes", `UPDATE mining_reward_settlement_receipts SET canonical_bytes=canonical_bytes||decode('20','hex')`},
		{"missing_grant", `DELETE FROM mining_reward_grants`},
		{"missing_lot", `DELETE FROM mining_reward_issuance_lots`},
		{"missing_consumption", `DELETE FROM mining_reward_reservation_consumptions`},
		{"T066_resolved_fingerprint", `UPDATE mining_reward_commands SET fingerprint=repeat('f',64)`},
	} {
		t.Run("T129_"+tc.name, func(t *testing.T) {
			s, _, _, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
				t.Fatal(err)
			}
			g21AuthorityCorrupt(t, s, tc.sql)
			before := g21ClosureSnapshot(t, s)
			_, err := s.SettleMiningRewardTEST(ctx, cmd)
			if !errors.Is(err, ErrMiningRewardInvariant) && !errors.Is(err, ErrMiningRewardConflict) {
				t.Fatalf("corruption accepted: %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
}

func TestG21ClosureAuthorityRestrictedRole(t *testing.T) {
	s, p, i, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	if _, err := s.SettleMiningRewardTEST(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	role := fmt.Sprintf("g21_authority_probe_%d", os.Getpid())
	quoted := pgx.Identifier{role}.Sanitize()
	if _, err := s.pool.Exec(ctx, `CREATE ROLE `+quoted+` NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = s.pool.Exec(ctx, `DROP OWNED BY `+quoted); _, _ = s.pool.Exec(ctx, `DROP ROLE `+quoted) })
	if _, err := s.pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO `+quoted+`; GRANT SELECT,INSERT,UPDATE,DELETE ON ALL TABLES IN SCHEMA public TO `+quoted); err != nil {
		t.Fatal(err)
	}
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, `SET ROLE `+quoted); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, `RESET ROLE`)
	var super, createdb, createrole bool
	if err := conn.QueryRow(ctx, `SELECT rolsuper,rolcreatedb,rolcreaterole FROM pg_roles WHERE rolname=current_user`).Scan(&super, &createdb, &createrole); err != nil || super || createdb || createrole {
		t.Fatalf("privileged role %v %v %v %v", super, createdb, createrole, err)
	}
	var raw []byte
	if err := s.pool.QueryRow(ctx, `SELECT data FROM mining_power_activities LIMIT 1`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	for _, tc := range []struct {
		name, sql, code string
		args            []any
	}{
		{"T047_direct_activity", `INSERT INTO mining_power_activities(data) VALUES($1)`, "55000", []any{raw}},
		{"T047_disable_trigger", `ALTER TABLE mining_power_activities DISABLE TRIGGER ALL`, "42501", nil},
		{"T057_identity", `UPDATE mining_blocks SET block_instance_id='mining-block-instance-11111111111111111111111111111111'`, "55000", nil},
		{"T057_delete", `DELETE FROM mining_blocks`, "23503", nil},
		{"T057_truncate", `TRUNCATE mining_blocks CASCADE`, "42501", nil},
		{"T057_reuse", `INSERT INTO mining_blocks SELECT * FROM mining_blocks`, "23505", nil},
		{"T128_DDL", `ALTER TABLE mining_reward_issuance_lots DROP CONSTRAINT mining_reward_lot_settlement_shape`, "42501", nil},
		{"T128_disable_trigger", `ALTER TABLE mining_reward_issuance_lots DISABLE TRIGGER ALL`, "42501", nil},
		{"T128_owner_delete", `DELETE FROM characters WHERE id='g20-player'`, "", nil},
		{"T128_replica_bypass", `SET session_replication_role=replica`, "42501", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := conn.Exec(ctx, tc.sql, tc.args...)
			var pg *pgconn.PgError
			if !errors.As(err, &pg) || tc.code != "" && pg.Code != tc.code {
				t.Fatalf("restricted role operation=%v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	// A distinct unauthorized operator has schema access but no table grants.
	if _, err = conn.Exec(ctx, `RESET ROLE`); err != nil {
		t.Fatal(err)
	}
	deniedRole := role + "_denied"
	deniedQuoted := pgx.Identifier{deniedRole}.Sanitize()
	if _, err = s.pool.Exec(ctx, `CREATE ROLE `+deniedQuoted+` NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT; GRANT USAGE ON SCHEMA public TO `+deniedQuoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = s.pool.Exec(ctx, `DROP OWNED BY `+deniedQuoted)
		_, _ = s.pool.Exec(ctx, `DROP ROLE `+deniedQuoted)
	})
	unauthorized, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL")+"&options=-c%20role%3D"+deniedRole)
	if err != nil {
		t.Fatal(err)
	}
	defer unauthorized.Close()
	altered := cmd
	altered.CommandID = "unauthorized-operator"
	_, err = unauthorized.SettleMiningRewardTEST(ctx, altered)
	g181SQLState(t, err, "42501", "")
	for _, table := range []string{"mining_power_activities", "mining_power_beneficiary_bindings", "mining_reward_settlement_receipts", "mining_reward_issuance_lots"} {
		var n int
		err = unauthorized.pool.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n)
		g181SQLState(t, err, "42501", "")
	}
	g21AuthorityUnchanged(t, s, before)
	other := miningpower.Principal{AccountID: p.AccountID + "-forged", PlayerID: p.PlayerID}
	r, err := s.ValidateMiningActivity(ctx, other, i, miningpower.DevelopmentRuleVersion)
	if err != nil || r.Original != nil || r.ReasonCode != "PLAYER_OWNER_MISMATCH" {
		t.Fatalf("cross-owner historical leak=%+v %v", r, err)
	}
	g21AuthorityUnchanged(t, s, before)
}

// This child deliberately uses an independent process and connection. It never
// calls fixture reset. The parent provides admitted identity and file barriers.
func TestG21ClosureAuthorityProcessChild(t *testing.T) {
	if os.Getenv("G21_AUTHORITY_CHILD") != "1" {
		return
	}
	ctx := context.Background()
	s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var req struct {
		P miningpower.Principal
		I miningpower.ActionIntent
	}
	if err := json.Unmarshal([]byte(os.Getenv("G21_AUTHORITY_INPUT")), &req); err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("G21_AUTHORITY_DIR")
	id := os.Getenv("G21_AUTHORITY_ID")
	s.miningPowerFailureInjector = func(stage string) error {
		if stage == "after_validation" {
			if err := os.WriteFile(filepath.Join(dir, id+".ready"), []byte(fmt.Sprint(os.Getpid())), 0600); err != nil {
				return err
			}
			for {
				if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
					break
				}
				if _, err := os.Stat(filepath.Join(dir, id+".release")); err == nil {
					break
				}
				time.Sleep(time.Millisecond)
			}
		}
		return nil
	}
	r, err := s.ValidateMiningActivity(ctx, req.P, req.I, miningpower.DevelopmentRuleVersion)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != miningpower.StatusValid && r.ReasonCode != "BLOCK_SEALED" {
		t.Fatalf("worker result=%+v", r)
	}
	raw, _ := json.Marshal(r)
	if err := os.WriteFile(filepath.Join(dir, id+".result"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestG21ClosureAuthorityTwentyProcessesContendWithSeal(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	dir := t.TempDir()
	var workers []*exec.Cmd
	for k := 0; k < 20; k++ {
		in := g20Event(t, s, i, fmt.Sprintf("parallel-event-%02d", k))
		raw, _ := json.Marshal(struct {
			P miningpower.Principal
			I miningpower.ActionIntent
		}{p, in})
		cmd := exec.Command(os.Args[0], "-test.run=^TestG21ClosureAuthorityProcessChild$", "-test.v")
		cmd.Env = append(os.Environ(), "G21_AUTHORITY_CHILD=1", "G21_AUTHORITY_INPUT="+string(raw), "G21_AUTHORITY_DIR="+dir, fmt.Sprintf("G21_AUTHORITY_ID=%02d", k))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		workers = append(workers, cmd)
	}
	t.Cleanup(func() {
		for _, cmd := range workers {
			if cmd.ProcessState == nil {
				_ = cmd.Process.Kill()
				_ = cmd.Wait()
			}
		}
	})
	deadline := time.Now().Add(15 * time.Second)
	for {
		ready, _ := filepath.Glob(filepath.Join(dir, "*.ready"))
		if len(ready) == 20 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d workers ready", len(ready))
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "00.release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "00.result")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first admission did not commit")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	for _, cmd := range workers {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	accepted := map[string]bool{}
	for k := 0; k < 20; k++ {
		raw, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%02d.result", k)))
		if err != nil {
			t.Fatal(err)
		}
		var r miningpower.ValidationResult
		if err = json.Unmarshal(raw, &r); err != nil {
			t.Fatal(err)
		}
		if r.Status == miningpower.StatusValid {
			accepted[r.Original.SourceEventID] = true
		}
	}
	if len(accepted) != seal.ActivityCount || seal.TotalValidMiningPower != int64(len(accepted))*100 {
		t.Fatalf("admitted=%d seal=%+v", len(accepted), seal)
	}
	for _, a := range seal.AcceptedActivityIdentities {
		if !accepted[a.SourceEventID] {
			t.Fatalf("uncommitted source in seal %s", a.SourceEventID)
		}
	}
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_activities`).Scan(&count); err != nil || count != len(accepted) {
		t.Fatalf("facts=%d expected=%d %v", count, len(accepted), err)
	}
	if _, err := s.RebuildMiningPowerSeal(ctx, i.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	t.Logf("T038 independent workers=20 controller=%d committed=%d rejected=%d seal activities=%d", os.Getpid(), len(accepted), 20-len(accepted), seal.ActivityCount)
}

func TestG21ClosureAuthorityBindingCreationAndFingerprint(t *testing.T) {
	s, p, i, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	var source, receipt, instance, digest string
	var amount int64
	if err := s.pool.QueryRow(ctx, `SELECT b.reservation_source_id,b.reservation_receipt_id,b.block_instance_id,b.reservation_amount,b.evidence_digest FROM mining_reservation_instance_bindings b JOIN mining_block_entries e ON e.entry_id=b.reservation_source_id AND e.action='OPEN' JOIN mining_block_receipts r ON r.receipt_id=b.reservation_receipt_id AND r.entry_id=e.entry_id JOIN mining_blocks m ON m.block_instance_id=b.block_instance_id WHERE m.block_id=$1 AND m.status='FINALIZED'`, i.BlockID).Scan(&source, &receipt, &instance, &amount, &digest); err != nil || instance != i.BlockInstanceID || amount != 10 || len(digest) != 64 {
		t.Fatalf("T049 actual authority binding %s/%s/%s/%d/%s %v", source, receipt, instance, amount, digest, err)
	}
	first, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	second, err := miningblock.NewService(s, miningblock.DevelopmentRuleVersion, false).Create(ctx, "g21-independent-instance")
	if err != nil {
		t.Fatal(err)
	}
	secondBlock, err := s.LoadMiningBlock(ctx, second.BlockID)
	if err != nil {
		t.Fatal(err)
	}
	if secondBlock.BlockInstanceID == i.BlockInstanceID {
		t.Fatal("T051 independent instance collided")
	}
	if _, err = s.FinalizeMiningBlock(ctx, second.BlockID); err != nil {
		t.Fatal(err)
	}
	secondSeal, err := s.SealMiningPowerTEST(ctx, secondBlock.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	altered := cmd
	altered.BlockInstanceID = secondBlock.BlockInstanceID
	altered.ExpectedSealDigest = secondSeal.CanonicalDigest
	if _, err := s.SettleMiningRewardTEST(ctx, altered); !errors.Is(err, ErrMiningRewardConflict) {
		t.Fatalf("T064 changed valid instance=%v", err)
	}
	g21AuthorityUnchanged(t, s, before)
	altered = cmd
	altered.ExpectedSealDigest = strings.Repeat("f", 64)
	if _, err := s.SettleMiningRewardTEST(ctx, altered); !errors.Is(err, ErrMiningRewardConflict) {
		t.Fatalf("T066 changed requested seal=%v", err)
	}
	g21AuthorityUnchanged(t, s, before)
	for _, sql := range []string{`UPDATE mining_blocks SET block_id='reused-display' WHERE block_id=$1`, `DELETE FROM mining_blocks WHERE block_id=$1`} {
		if _, err := s.pool.Exec(ctx, sql, i.BlockID); err == nil {
			t.Fatalf("T051 protected display changed: %s", sql)
		}
		g21AuthorityUnchanged(t, s, before)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for k := 0; k < 2; k++ {
		wg.Add(1)
		go func(k int) {
			defer wg.Done()
			_, err := s.pool.Exec(ctx, `INSERT INTO mining_reward_reservation_consumptions(consumption_id,settlement_id,block_instance_id,reservation_source_id,amount,pool_revision,created_at) VALUES($1,$2,$3,$4,10,$5,now())`, fmt.Sprintf("source-conflict-%d", k), fmt.Sprintf("source-settlement-%d", k), secondBlock.BlockInstanceID, first.ReservationSourceID, first.PoolRevisionAfter+int64(k)+1)
			errs <- err
		}(k)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		g181SQLState(t, err, "23505", "mining_reward_reservation_consumption_reservation_source_id_key")
	}
	g21AuthorityUnchanged(t, s, before)
	_ = p
}

func TestG21ClosureAuthoritySealAndRuleCorruption(t *testing.T) {
	for _, field := range []string{"G20RuleVersion", "AllocationVersion", "ConversionVersion"} {
		t.Run("T065_"+field, func(t *testing.T) {
			s, _, _, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			r, err := s.SettleMiningRewardTEST(ctx, cmd)
			if err != nil {
				t.Fatal(err)
			}
			switch field {
			case "G20RuleVersion":
				r.G20RuleVersion = "OTHER_G20_RULE"
			case "AllocationVersion":
				r.AllocationVersion = "OTHER_ALLOCATION_RULE"
			case "ConversionVersion":
				r.ConversionVersion = "OTHER_CONVERSION_RULE"
			}
			raw, err := canonicalMiningRewardReceipt(r)
			if err != nil {
				t.Fatal(err)
			}
			g21AuthorityCorrupt(t, s, `UPDATE mining_reward_settlement_receipts SET canonical_bytes=$1,canonical_digest=$2`, raw, rewardHash(raw))
			before := g21ClosureSnapshot(t, s)
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardInvariant) && !errors.Is(err, ErrMiningRewardConflict) {
				t.Fatalf("rule changed successful replay=%v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
	for _, sql := range []string{`UPDATE mining_power_input_seals SET canonical_digest=repeat('0',64)`, `UPDATE mining_power_input_seals SET canonical_bytes=canonical_bytes||decode('20','hex')`, `UPDATE mining_reservation_instance_bindings SET evidence_digest=repeat('0',64)`} {
		t.Run("T040_"+fmt.Sprint(len(sql)), func(t *testing.T) {
			s, _, i, _ := g21AuthoritySealed(t)
			g21AuthorityCorrupt(t, s, sql)
			before := g21ClosureSnapshot(t, s)
			if _, err := s.RebuildMiningPowerSeal(context.Background(), i.BlockInstanceID); !errors.Is(err, miningpower.ErrInvariant) {
				t.Fatalf("corrupt seal rebuilt: %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
		})
	}
}

func TestG21ClosureAuthorityInvalidReservationBinding(t *testing.T) {
	for _, amount := range []int64{0, -1} {
		t.Run(fmt.Sprintf("T010_R%d", amount), func(t *testing.T) {
			s, _, _, cmd := g21AuthoritySealed(t)
			ctx := context.Background()
			// The ordinary database authority rejects invalid R before it can exist.
			before := g21ClosureSnapshot(t, s)
			_, err := s.pool.Exec(ctx, `UPDATE mining_reservation_instance_bindings SET reservation_amount=$1`, amount)
			if err == nil {
				t.Fatal("immutable reservation changed")
			}
			g21AuthorityUnchanged(t, s, before)
			// Corruption oracle bypasses the positive check only in this disposable DB.
			if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_reservation_instance_bindings DROP CONSTRAINT mining_reservation_instance_bindings_reservation_amount_check`); err != nil {
				t.Fatal(err)
			}
			g21AuthorityCorrupt(t, s, `UPDATE mining_reservation_instance_bindings SET reservation_amount=$1`, amount)
			before = g21ClosureSnapshot(t, s)
			if _, err := s.SettleMiningRewardTEST(ctx, cmd); !errors.Is(err, ErrMiningRewardInvariant) {
				t.Fatalf("invalid R accepted %v", err)
			}
			g21AuthorityUnchanged(t, s, before)
			// Restore the constraint for the next reusable fixture only after deleting
			// the deliberately damaged row, with all triggers restored at commit.
			g21AuthorityCorrupt(t, s, `DELETE FROM mining_reservation_instance_bindings`)
			if _, err := s.pool.Exec(ctx, `ALTER TABLE mining_reservation_instance_bindings ADD CONSTRAINT mining_reservation_instance_bindings_reservation_amount_check CHECK(reservation_amount>0)`); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestG21ClosureAuthoritySealRollbackAndAdmission(t *testing.T) {
	t.Run("T034_acceptance_rollback", func(t *testing.T) {
		s, p, i := g20Fixture(t)
		ctx := context.Background()
		reached := make(chan struct{})
		release := make(chan struct{})
		sentinel := errors.New("abort accepted fact")
		s.miningPowerFailureInjector = func(stage string) error {
			if stage == "after_activity" {
				close(reached)
				<-release
				return sentinel
			}
			return nil
		}
		done := make(chan error, 1)
		go func() { _, err := s.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion); done <- err }()
		<-reached
		if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
			close(release)
			t.Fatal(err)
		}
		type sealed struct {
			seal miningpower.SettlementInputSeal
			err  error
		}
		seals := make(chan sealed, 1)
		go func() { v, e := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); seals <- sealed{v, e} }()
		select {
		case r := <-seals:
			close(release)
			t.Fatalf("seal passed uncommitted admission %+v", r)
		case <-time.After(50 * time.Millisecond):
		}
		close(release)
		if err := <-done; !errors.Is(err, sentinel) {
			t.Fatal(err)
		}
		r := <-seals
		if r.err != nil || r.seal.ActivityCount != 0 || r.seal.TotalValidMiningPower != 0 {
			t.Fatalf("rolled back fact included %+v", r)
		}
		if a, b := g20Counts(t, s); a != 0 || b != 0 {
			t.Fatalf("acceptance rollback left facts %d/%d", a, b)
		}
	})
	for _, rollback := range []bool{false, true} {
		t.Run(fmt.Sprintf("T035_T037_seal_rollback_%v", rollback), func(t *testing.T) {
			s, p, i := g20Fixture(t)
			ctx := context.Background()
			validated := make(chan struct{})
			acceptRelease := make(chan struct{})
			sealReached := make(chan struct{})
			sealRelease := make(chan struct{})
			sentinel := errors.New("abort seal")
			s.miningPowerFailureInjector = func(stage string) error {
				if stage == "after_validation" {
					close(validated)
					<-acceptRelease
				}
				return nil
			}
			s.prerequisiteFailureInjector = func(stage string) error {
				if stage == "seal_before_commit" {
					close(sealReached)
					<-sealRelease
					if rollback {
						return sentinel
					}
				}
				return nil
			}
			results := make(chan miningpower.ValidationResult, 1)
			errs := make(chan error, 1)
			go func() {
				r, e := s.ValidateMiningActivity(ctx, p, i, miningpower.DevelopmentRuleVersion)
				results <- r
				errs <- e
			}()
			<-validated
			if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
				t.Fatal(err)
			}
			sealed := make(chan error, 1)
			go func() { _, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); sealed <- err }()
			<-sealReached
			close(acceptRelease)
			select {
			case r := <-results:
				close(sealRelease)
				t.Fatalf("accept bypassed exclusive gate %+v", r)
			case <-time.After(50 * time.Millisecond):
			}
			close(sealRelease)
			sealErr := <-sealed
			r := <-results
			err := <-errs
			if rollback {
				if !errors.Is(sealErr, sentinel) || err != nil || r.Status != miningpower.StatusValid {
					t.Fatalf("rollback resume %+v %v seal=%v", r, err, sealErr)
				}
				var n int
				if e := s.pool.QueryRow(ctx, `SELECT count(*) FROM mining_power_input_seals`).Scan(&n); e != nil || n != 0 {
					t.Fatalf("orphan seal %d %v", n, e)
				}
			} else if sealErr != nil || err != nil || r.ReasonCode != "BLOCK_SEALED" {
				t.Fatalf("sealed gate admitted %+v %v seal=%v", r, err, sealErr)
			}
		})
	}
}

func TestG21ClosureAuthorityMissingGateAfterHistory(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, p, i)
	next := g20Event(t, s, i, "later-without-gate")
	g21AuthorityCorrupt(t, s, `DELETE FROM mining_power_acceptance_states`)
	before := g21ClosureSnapshot(t, s)
	r, err := s.ValidateMiningActivity(ctx, p, next, miningpower.DevelopmentRuleVersion)
	if err == nil && r.Status == miningpower.StatusValid {
		t.Fatalf("T045 missing historical gate was silently repaired, newly accepted source %s", next.SourceEventID)
	}
	g21AuthorityUnchanged(t, s, before)
}

func TestG21ClosureAuthorityConfiguredRuleRestart(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, p, i)
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); err != nil {
		t.Fatal(err)
	}
	before := g21ClosureSnapshot(t, s)
	restarted, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	svc := miningpower.NewService(restarted, "FUTURE_RULE", false)
	r, err := svc.Validate(ctx, p, i)
	if err != nil || r.Status != miningpower.StatusReplayed || r.ReasonCode != "BINDING_CHANGED" || r.Original != nil || r.AppliedPower != 0 {
		t.Fatalf("configured changed-rule replay violated existing G20 fingerprint contract: %+v %v", r, err)
	}
	t.Log("G20 activity replay retains BINDING_CHANGED; completed G21 settlement replay is independently covered by TestG21T044CompletedReplayPrecedence")
	g21AuthorityUnchanged(t, s, before)
}

func TestG21ClosureAuthorityStrictInvalidUTF8AndSurrogates(t *testing.T) {
	s, _, i, cmd := g21AuthoritySealed(t)
	valid, _ := json.Marshal(map[string]string{"commandID": cmd.CommandID, "blockInstanceID": i.BlockInstanceID, "expectedSealDigest": cmd.ExpectedSealDigest})
	before := g21ClosureSnapshot(t, s)
	for _, value := range []string{string([]byte{0xff}), `\ud800`, `\udfff`, `\ud800x`} {
		raw := strings.Replace(string(valid), cmd.CommandID, value, 1)
		if _, err := s.SettleMiningRewardPayloadTEST(context.Background(), []byte(raw)); !errors.Is(err, miningpower.ErrInvalidInput) {
			t.Fatalf("malformed UTF accepted: %v", err)
		}
		g21AuthorityUnchanged(t, s, before)
	}
}

func TestG21ClosureAuthoritySealMissingHistoricalGate(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, p, i)
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	g21AuthorityCorrupt(t, s, `DELETE FROM mining_power_acceptance_states`)
	before := g21ClosureSnapshot(t, s)
	if _, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID); !errors.Is(err, miningpower.ErrInvariant) {
		t.Fatalf("T045 seal silently recreated missing historical gate: %v", err)
	}
	g21AuthorityUnchanged(t, s, before)
}

func TestG21ClosureAuthoritySettlementProcessChild(t *testing.T) {
	if os.Getenv("G21_AUTHORITY_SETTLE_CHILD") != "1" {
		return
	}
	ctx := context.Background()
	id := os.Getenv("G21_AUTHORITY_ID")
	dir := os.Getenv("G21_AUTHORITY_DIR")
	s, err := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL")+"&application_name=g21-authority-settle-"+id)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var cmd MiningRewardSettlementCommand
	if err := json.Unmarshal([]byte(os.Getenv("G21_AUTHORITY_INPUT")), &cmd); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("G21_AUTHORITY_SETTLE_HOLD") == "1" {
		s.settlementFailureInjector = func(point string) error {
			if point != "before_issuance" {
				return nil
			}
			if err := os.WriteFile(filepath.Join(dir, "active"), []byte(fmt.Sprint(os.Getpid())), 0600); err != nil {
				return err
			}
			deadline := time.Now().Add(15 * time.Second)
			for {
				if _, err := os.Stat(filepath.Join(dir, "release")); err == nil {
					return nil
				}
				if time.Now().After(deadline) {
					return errors.New("settlement release timeout")
				}
				time.Sleep(time.Millisecond)
			}
		}
	}
	before := g21ClosureSnapshot(t, s)
	r, err := s.SettleMiningRewardTEST(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("G21_AUTHORITY_SETTLE_VERIFY") == "1" {
		g21AuthorityUnchanged(t, s, before)
		audit, err := s.ReconcileBlackIronEmission(ctx)
		if err != nil || !audit.Balanced {
			t.Fatalf("independent verifier emission %+v %v", audit, err)
		}
		reward, err := s.ReconcileMiningRewardIssuancePrerequisite(ctx)
		if err != nil || !reward.Balanced {
			t.Fatalf("independent verifier issuance %+v %v", reward, err)
		}
	}
	raw, err := canonicalMiningRewardReceipt(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".receipt"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestG21ClosureAuthorityTenSettlementProcesses(t *testing.T) {
	s, _, _, cmd := g21AuthoritySealed(t)
	ctx := context.Background()
	dir := t.TempDir()
	raw, _ := json.Marshal(cmd)
	var workers []*exec.Cmd
	start := func(id string, hold, verify bool) *exec.Cmd {
		c := exec.Command(os.Args[0], "-test.run=^TestG21ClosureAuthoritySettlementProcessChild$", "-test.v")
		c.Env = append(os.Environ(), "G21_AUTHORITY_SETTLE_CHILD=1", "G21_AUTHORITY_INPUT="+string(raw), "G21_AUTHORITY_DIR="+dir, "G21_AUTHORITY_ID="+id)
		if hold {
			c.Env = append(c.Env, "G21_AUTHORITY_SETTLE_HOLD=1")
		}
		if verify {
			c.Env = append(c.Env, "G21_AUTHORITY_SETTLE_VERIFY=1")
		}
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		workers = append(workers, c)
		return c
	}
	t.Cleanup(func() {
		for _, c := range workers {
			if c.ProcessState == nil {
				_ = c.Process.Kill()
				_ = c.Wait()
			}
		}
	})
	start("00", true, false)
	deadline := time.Now().Add(15 * time.Second)
	for {
		if _, err := os.Stat(filepath.Join(dir, "active")); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first process did not reach transaction barrier")
		}
		time.Sleep(time.Millisecond)
	}
	for k := 1; k < 10; k++ {
		start(fmt.Sprintf("%02d", k), false, false)
	}
	for {
		var waiting int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE application_name LIKE 'g21-authority-settle-%' AND wait_event_type='Lock'`).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting == 9 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected nine real DB lock waiters, got %d", waiting)
		}
		time.Sleep(time.Millisecond)
	}
	if err := os.WriteFile(filepath.Join(dir, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, c := range workers {
		if err := c.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	var original string
	for k := 0; k < 10; k++ {
		value, err := os.ReadFile(filepath.Join(dir, fmt.Sprintf("%02d.receipt", k)))
		if err != nil {
			t.Fatal(err)
		}
		if k == 0 {
			original = string(value)
		} else if original != string(value) {
			t.Fatalf("process %d returned different canonical receipt", k)
		}
	}
	var commands, receipts, lots, grants, consumes int
	var ore int64
	if err := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_commands),(SELECT count(*) FROM mining_reward_settlement_receipts),(SELECT count(*) FROM mining_reward_issuance_lots),(SELECT count(*) FROM mining_reward_grants),(SELECT count(*) FROM mining_reward_reservation_consumptions),(SELECT sum(quantity) FROM character_inventory_items)`).Scan(&commands, &receipts, &lots, &grants, &consumes, &ore); err != nil || commands != 1 || receipts != 1 || lots != 1 || grants != 1 || consumes != 1 || ore != 10 {
		t.Fatalf("duplicated economic artifacts %d/%d/%d/%d/%d/%d %v", commands, receipts, lots, grants, consumes, ore, err)
	}
	verifier := start("verifier", false, true)
	if err := verifier.Wait(); err != nil {
		t.Fatal(err)
	}
	verified, err := os.ReadFile(filepath.Join(dir, "verifier.receipt"))
	if err != nil || string(verified) != original {
		t.Fatalf("independent restart receipt mismatch %v", err)
	}
	t.Logf("T062 ten process PIDs=%v; nine observed advisory-lock waiters; verifier PID=%d; exact receipt bytes=%d", func() []int {
		ids := []int{}
		for _, c := range workers[:10] {
			ids = append(ids, c.Process.Pid)
		}
		return ids
	}(), verifier.Process.Pid, len(original))
}

func TestG21ClosureAuthorityExpiredReplayAndOldTimeSource(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	source := g20Source(t, s, i)
	source.ID = "short-authority-source"
	source.ActivityID = "mpa:" + source.ID
	source.ExpiresAt = time.Now().UTC().Add(time.Second).Truncate(time.Microsecond)
	short := g20RegisteredSource(t, s, source)
	accepted := g20Accept(t, s, p, short)
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(source.ExpiresAt) + time.Millisecond)
	before := g21ClosureSnapshot(t, s)
	r, err := s.ValidateMiningActivity(ctx, p, short, miningpower.DevelopmentRuleVersion)
	if err != nil || r.Status != miningpower.StatusDuplicate || r.AppliedPower != 0 || r.Original == nil || !reflect.DeepEqual(accepted, *r.Original) {
		t.Fatalf("T041 expired sealed replay %+v %v", r, err)
	}
	g21AuthorityUnchanged(t, s, before)
	fresh := g20Event(t, s, i, "new-source-old-observation")
	before = g21ClosureSnapshot(t, s)
	r, err = s.ValidateMiningActivity(ctx, p, fresh, miningpower.DevelopmentRuleVersion)
	if err != nil || r.ReasonCode != "BLOCK_SEALED" || r.Original != nil || r.AppliedPower != 0 {
		t.Fatalf("T043 old observed new source %+v %v", r, err)
	}
	g21AuthorityUnchanged(t, s, before)
	again, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if err != nil || !reflect.DeepEqual(seal, again) {
		t.Fatalf("seal changed after expired replay %v", err)
	}
}

func TestG21ClosureAuthorityMixedRuleCannotBeFilteredOut(t *testing.T) {
	s, p, i := g20Fixture(t)
	ctx := context.Background()
	g20Accept(t, s, p, i)
	session := g20Session(t, s, i)
	session.ID = "mixed-rule-session"
	g20Insert(t, s, "mining_power_sessions", session)
	source := g20Source(t, s, i)
	source.ID = "mixed-rule-source"
	source.ActivityID = "mpa:" + source.ID
	source.ActivitySessionID = session.ID
	other := g20RegisteredSource(t, s, source)
	g20Accept(t, s, p, other)
	if _, err := s.FinalizeMiningBlock(ctx, i.BlockID); err != nil {
		t.Fatal(err)
	}
	g21AuthorityCorrupt(t, s, `UPDATE mining_power_activities SET data=jsonb_set(jsonb_set(data,'{RuleVersion}','"UNAPPROVED_RULE"'),'{ValidationSnapshot,G20RuleVersion}','"UNAPPROVED_RULE"') WHERE source_event_id='mixed-rule-source'; UPDATE mining_power_participants SET rule_version='UNAPPROVED_RULE' WHERE session_id='mixed-rule-session'; UPDATE mining_power_sessions SET data=jsonb_set(data,'{RuleVersion}','"UNAPPROVED_RULE"') WHERE session_id='mixed-rule-session'; UPDATE mining_power_source_events SET data=jsonb_set(data,'{RuleVersion}','"UNAPPROVED_RULE"') WHERE source_event_id='mixed-rule-source'`)
	before := g21ClosureSnapshot(t, s)
	seal, err := s.SealMiningPowerTEST(ctx, i.BlockInstanceID)
	if !errors.Is(err, miningpower.ErrInvariant) {
		t.Fatalf("T032 unapproved accepted rule silently excluded from seal activities=%d err=%v", seal.ActivityCount, err)
	}
	g21AuthorityUnchanged(t, s, before)
}
