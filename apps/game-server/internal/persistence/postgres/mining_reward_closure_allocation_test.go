package postgres

import (
	"context"
	"fmt"
	"fractallegend/game-server/internal/emission"
	"fractallegend/game-server/internal/miningpower"
	"fractallegend/game-server/internal/miningreward"
	"fractallegend/game-server/internal/systemspend"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Sources and tool profiles are G20 TEST inputs, never G18 reward authority.
func g21ClosureWeightFixture(t *testing.T, names []string, powers []int64) (*Store, MiningRewardSettlementCommand) {
	t.Helper()
	s, _, intent := g20Fixture(t)
	ctx := context.Background()
	g19Catalog(t, s)
	baseSession := g20Session(t, s, intent)
	baseSource := g20Source(t, s, intent)
	for i, name := range names {
		g19Seed(t, s, name)
		ref := fmt.Sprintf("TEST_CLOSURE_%d", i)
		base, eff := powers[i], int64(miningpower.Scale)
		if base == 0 {
			base = 1
			eff = 1
		}
		g20Insert(t, s, "mining_power_tool_profiles", miningpower.ToolProfile{Reference: ref, RuleVersion: miningpower.DevelopmentRuleVersion, Kind: miningpower.SyntheticKind, BasePowerUnits: base, EfficiencyScaled: eff})
		session := baseSession
		session.ID = fmt.Sprintf("closure-session-%d", i)
		session.PlayerID = name
		session.AccountID = name + "-account"
		session.ToolReference = ref
		g20Insert(t, s, "mining_power_sessions", session)
		source := baseSource
		source.ID = fmt.Sprintf("closure-source-%d", i)
		source.ActivityID = "mpa:" + source.ID
		source.PlayerID = name
		source.ActivitySessionID = session.ID
		source.ToolReference = ref
		a := g20Accept(t, s, miningpower.Principal{PlayerID: name, AccountID: session.AccountID}, g20RegisteredSource(t, s, source))
		if a.ValidatedPower != powers[i] {
			t.Fatalf("power=%d want=%d", a.ValidatedPower, powers[i])
		}
	}
	if _, e := s.FinalizeMiningBlock(ctx, intent.BlockID); e != nil {
		t.Fatal(e)
	}
	seal, e := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
	if e != nil {
		t.Fatal(e)
	}
	return s, MiningRewardSettlementCommand{CommandID: "g21-weight-closure", BlockInstanceID: intent.BlockInstanceID, ExpectedSealDigest: seal.CanonicalDigest}
}

func TestG21ClosurePersistedAllocationVectors(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ids           []string
		weights, want []int64
	}{
		{"T001_exact_R10", []string{"B", "A"}, []int64{3, 2}, []int64{4, 6}},
		{"T011_wide_product_R10", []string{"B", "A"}, []int64{math.MaxInt64 / 2, math.MaxInt64 / 2}, []int64{5, 5}},
		{"T012_power_MaxInt64_R10", []string{"B", "A"}, []int64{1, math.MaxInt64 - 1}, []int64{10, 0}},
		{"T006_zero", []string{"B", "A"}, []int64{2, 0}, []int64{0, 10}},
		{"T008_three_zero", []string{"C", "A", "B"}, []int64{0, 0, 0}, nil},
		{"T018_case_bytes", []string{"a", "A", "Z"}, []int64{1, 1, 1}, []int64{4, 3, 3}},
		{"T019_numeric_bytes", []string{"2", "10", "3"}, []int64{1, 1, 1}, []int64{4, 3, 3}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, cmd := g21ClosureWeightFixture(t, tc.ids, tc.weights)
			ctx := context.Background()
			before := g21ClosureSnapshot(t, s)
			receipt, e := s.SettleMiningRewardTEST(ctx, cmd)
			if e != nil {
				t.Fatal(e)
			}
			if tc.want == nil {
				seal, e := s.LoadMiningPowerSeal(ctx, cmd.BlockInstanceID)
				if e != nil || seal.ActivityCount != 3 || seal.ParticipantCount != 3 || seal.TotalValidMiningPower != 0 || receipt.Status != "SEALED_NO_ELIGIBLE_POWER" || g21NoPowerEconomicSnapshot(g21ClosureSnapshot(t, s)) != g21NoPowerEconomicSnapshot(before) {
					t.Fatalf("zero-power seal=%+v receipt=%+v err=%v", seal, receipt, e)
				}
				var commands, states int
				if e := s.pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM mining_reward_commands WHERE status='SEALED_NO_ELIGIBLE_POWER'),(SELECT count(*) FROM mining_reward_settlement_states WHERE state='SEALED_NO_ELIGIBLE_POWER')`).Scan(&commands, &states); e != nil || commands != 1 || states != 1 {
					t.Fatal("no-power disposition", e)
				}
				after := g21ClosureSnapshot(t, s)
				again, e := s.SettleMiningRewardTEST(ctx, cmd)
				if e != nil || !reflect.DeepEqual(receipt, again) || g21ClosureSnapshot(t, s) != after {
					t.Fatal("no-power replay", e)
				}
				return
			}
			if receipt.ReservedAmount != 10 || receipt.TotalOre != 10 || len(receipt.Grants) != len(tc.want) {
				t.Fatalf("receipt=%+v", receipt)
			}
			for i, want := range tc.want {
				g := receipt.Grants[i]
				if g.Quantity != want {
					t.Fatalf("grant=%+v want=%d", g, want)
				}
				var revision int64
				var stacks, lots int
				if e := s.pool.QueryRow(ctx, `SELECT revision,(SELECT count(*) FROM character_inventory_items WHERE character_id=$1),(SELECT count(*) FROM mining_reward_issuance_lots WHERE character_id=$1) FROM characters WHERE id=$1`, g.CharacterID).Scan(&revision, &stacks, &lots); e != nil {
					t.Fatal(e)
				}
				if want == 0 && (revision != 1 || stacks != 0 || lots != 0) {
					t.Fatal("zero creates artifact")
				}
				if want > 0 && (revision != 2 || stacks != 1 || lots != 1 || g.ItemRevisionAfter != 1) {
					t.Fatal("positive version/artifact mismatch")
				}
			}
			paid := g21ClosureSnapshot(t, s)
			again, e := s.SettleMiningRewardTEST(ctx, cmd)
			if e != nil || !reflect.DeepEqual(receipt, again) || paid != g21ClosureSnapshot(t, s) {
				t.Fatalf("replay=%v", e)
			}
		})
	}
}

func TestG21ClosureSixPersistedPermutations(t *testing.T) {
	var expected []int64
	for _, order := range [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			ids := []string{"A", "B", "C"}
			w := []int64{1, 2, 3}
			s, cmd := g21ClosureWeightFixture(t, []string{ids[order[0]], ids[order[1]], ids[order[2]]}, []int64{w[order[0]], w[order[1]], w[order[2]]})
			r, e := s.SettleMiningRewardTEST(context.Background(), cmd)
			if e != nil {
				t.Fatal(e)
			}
			var got []int64
			for i, g := range r.Grants {
				if g.CharacterID != ids[i] {
					t.Fatal("order")
				}
				got = append(got, g.Quantity)
			}
			if expected == nil {
				expected = []int64{2, 3, 5}
			}
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("permutation=%v", got)
			}
		})
	}
}

func TestG21ClosureTimezoneStrictReplay(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"A"}, []int64{1})
	ctx := context.Background()
	first, e := s.SettleMiningRewardTEST(ctx, cmd)
	if e != nil {
		t.Fatal(e)
	}
	before := g21ClosureSnapshot(t, s)
	raw, e := canonicalMiningRewardReceipt(first)
	if e != nil {
		t.Fatal(e)
	}
	if first.CreatedAt.Location() != time.UTC || first.CreatedAt.Nanosecond()%1000 != 0 || first.CreatedAt != first.CreatedAt.Round(0) {
		t.Fatal("noncanonical time")
	}
	for _, zone := range []string{"UTC", "Asia/Shanghai", "America/New_York"} {
		t.Run(zone, func(t *testing.T) {
			loc, e := time.LoadLocation(zone)
			if e != nil {
				t.Fatal(e)
			}
			copy := first
			copy.CreatedAt = first.CreatedAt.In(loc)
			encoded, e := canonicalMiningRewardReceipt(copy)
			if e != nil || string(encoded) != string(raw) {
				t.Fatal("timezone changed canonical bytes", e)
			}
			fresh, e := Open(ctx, s.pool.Config().ConnString())
			if e != nil {
				t.Fatal(e)
			}
			defer fresh.Close()
			replay, e := fresh.SettleMiningRewardTEST(ctx, cmd)
			if e != nil || !reflect.DeepEqual(first, replay) {
				t.Fatal("new connection replay", e)
			}
		})
	}
	// Runtime monotonic clocks and submicrosecond input have one canonical wire time.
	for _, clock := range []time.Time{
		time.Now(),
		time.Date(2026, 9, 28, 12, 0, 0, 123456789, time.UTC),
	} {
		c := first
		c.CreatedAt = clock
		encoded, e := canonicalMiningRewardReceipt(c)
		if e != nil {
			t.Fatal(e)
		}
		decoded, e := decodeCanonicalMiningRewardReceipt(encoded)
		if e != nil || decoded.CreatedAt != miningreward.CanonicalTime(clock) {
			t.Fatalf("clock normalization got=%s input=%s err=%v", decoded.CreatedAt, clock, e)
		}
	}
	if g21ClosureSnapshot(t, s) != before {
		t.Fatal("timezone replay changed database")
	}
}

// Only the specified immutable non-economic no-power disposition may be new.
func g21NoPowerEconomicSnapshot(raw string) string {
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, "mining_reward_commands:") && !strings.HasPrefix(line, "mining_reward_settlement_states:") {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func TestG21ClosureExactPoolAndHundredReplays(t *testing.T) {
	s, cmd := g21ClosureWeightFixture(t, []string{"B", "A"}, []int64{1, 1})
	ctx := context.Background()
	spend, e := systemspend.NewService(s, g15Registry(t)).Post(ctx, g15Intent("g21-closure-pool-funding", 80))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = emission.NewService(s, emission.DevelopmentRuleVersion).Apply(ctx, spend.ID); e != nil {
		t.Fatal(e)
	}
	r, e := s.SettleMiningRewardTEST(ctx, cmd)
	if e != nil {
		t.Fatal(e)
	}
	if r.PoolNetCapacityBefore != 100 || r.PoolNetCapacityAfter != 100 || r.PoolReservedBefore != 10 || r.PoolReservedAfter != 0 || r.PoolDistributedBefore != 0 || r.PoolDistributedAfter != 10 || r.PoolRemainingBefore != 90 || r.PoolRemainingAfter != 90 || r.PoolDebtBefore != 0 || r.PoolDebtAfter != 0 || r.PoolRevisionAfter != r.PoolRevisionBefore+1 || len(r.Grants) != 2 || r.Grants[0].Quantity != 5 || r.Grants[1].Quantity != 5 {
		t.Fatalf("exact pool PAY oracle %+v", r)
	}
	before := g21ClosureSnapshot(t, s)
	for i := 0; i < 100; i++ {
		again, e := s.SettleMiningRewardTEST(ctx, cmd)
		if e != nil || !reflect.DeepEqual(r, again) {
			t.Fatalf("replay%d %v", i, e)
		}
	}
	if before != g21ClosureSnapshot(t, s) {
		t.Fatal("100 replays changed persisted rows")
	}
}
