package postgres

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"fractallegend/game-server/internal/miningpower"
)

func TestG21P0SealParticipantBoundaries(t *testing.T) {
	for _, count := range []int{2, 10, 50, 100, 500, 501} {
		t.Run(fmt.Sprintf("participants-%d", count), func(t *testing.T) {
			start := time.Now()
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			g20Accept(t, s, principal, intent)
			baseSession := g20Session(t, s, intent)
			baseSource := g20Source(t, s, intent)
			writes := 0
			for i := 1; i < count; i++ {
				player := fmt.Sprintf("g21-p0-player-%04d", i)
				g19Seed(t, s, player)
				writes += 3
				session := baseSession
				session.ID = fmt.Sprintf("g21-p0-session-%04d", i)
				session.PlayerID = player
				session.AccountID = player + "-account"
				g20Insert(t, s, "mining_power_sessions", session)
				source := baseSource
				source.ID = fmt.Sprintf("g21-p0-event-%04d", i)
				source.ActivityID = "mpa:" + source.ID
				source.PlayerID = player
				source.ActivitySessionID = session.ID
				g20Insert(t, s, "mining_power_source_events", source)
				writes += 2
				p := miningpower.Principal{PlayerID: player, AccountID: session.AccountID}
				in := miningpower.ActionIntent{ActivityID: source.ActivityID, SourceEventID: source.ID,
					ActivitySessionID: session.ID, BlockID: intent.BlockID, BlockInstanceID: intent.BlockInstanceID}
				g20Accept(t, s, p, in)
				writes += 3 // Activity, beneficiary binding, participant aggregate.
			}
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if count == 501 {
				if err == nil {
					t.Fatal("501 participants sealed")
				}
				var state string
				if e := s.pool.QueryRow(ctx, `SELECT state FROM mining_power_acceptance_states WHERE block_instance_id=$1`, intent.BlockInstanceID).Scan(&state); e != nil || state != "OPEN" {
					t.Fatalf("501 changed state=%s %v", state, e)
				}
				var poolDistributed int64
				if e := s.pool.QueryRow(ctx, `SELECT total_distributed FROM black_iron_emission_pools`).Scan(&poolDistributed); e != nil || poolDistributed != 0 {
					t.Fatalf("501 economic effect=%d %v", poolDistributed, e)
				}
				return
			}
			if err != nil || seal.ParticipantCount != count || seal.ActivityCount != count || seal.TotalValidMiningPower != int64(100*count) || len(seal.ParticipantWeights) != count {
				t.Fatalf("seal=%+v err=%v", seal, err)
			}
			for i := 1; i < len(seal.ParticipantWeights); i++ {
				if seal.ParticipantWeights[i-1].PlayerID >= seal.ParticipantWeights[i].PlayerID {
					t.Fatal("unstable participant order")
				}
			}
			if len(seal.BeneficiaryBindingDigest) != 64 || len(seal.AcceptedActivityIdentityDigest) != 64 || len(seal.CanonicalDigest) != 64 {
				t.Fatal("missing canonical digests")
			}
			rebuilt, e := s.RebuildMiningPowerSeal(ctx, intent.BlockInstanceID)
			if e != nil || !reflect.DeepEqual(seal, rebuilt) {
				t.Fatalf("rebuild=%+v %v", rebuilt, e)
			}
			reopened, e := Open(ctx, os.Getenv("FRACTAL_TEST_DATABASE_URL"))
			if e != nil {
				t.Fatal(e)
			}
			defer reopened.Close()
			restarted, e := reopened.RebuildMiningPowerSeal(ctx, intent.BlockInstanceID)
			if e != nil || !reflect.DeepEqual(seal, restarted) {
				t.Fatalf("restart=%+v %v", restarted, e)
			}
			t.Logf("participants=%d elapsed=%s approximate_fixture_and_accept_writes=%d", count, time.Since(start), writes)
		})
	}
}
