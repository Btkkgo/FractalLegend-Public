package postgres

import (
	"context"
	"reflect"
	"testing"

	"fractallegend/game-server/internal/contribution"
)

func TestG21PreviewReadsOnlyForEligibleDebtAndZeroPower(t *testing.T) {
	for _, scenario := range []string{"eligible", "debt", "zero-power"} {
		t.Run(scenario, func(t *testing.T) {
			s, principal, intent := g20Fixture(t)
			ctx := context.Background()
			if scenario != "zero-power" {
				g20Accept(t, s, principal, intent)
			}
			if scenario == "debt" {
				spend, err := s.LoadSystemSpend(ctx, "g18-fund")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := contribution.NewService(s).RefundSystemSpend(ctx, g15Refund(spend, "g21-preview-debt", 15)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.FinalizeMiningBlock(ctx, intent.BlockID); err != nil {
				t.Fatal(err)
			}
			seal, err := s.SealMiningPowerTEST(ctx, intent.BlockInstanceID)
			if err != nil {
				t.Fatal(err)
			}
			before := g21ClosureSnapshot(t, s)
			first, err := s.PreviewMiningRewardTEST(ctx, intent.BlockInstanceID, seal.CanonicalDigest)
			if err != nil {
				t.Fatal(err)
			}
			second, err := s.PreviewMiningRewardTEST(ctx, intent.BlockInstanceID, seal.CanonicalDigest)
			if err != nil || !reflect.DeepEqual(first, second) {
				t.Fatalf("preview replay=%+v first=%+v %v", second, first, err)
			}
			if after := g21ClosureSnapshot(t, s); after != before {
				t.Fatal("preview mutated complete database snapshot")
			}
			if first.Status != "ALLOCATION_PREVIEW" || first.BlockInstanceID != intent.BlockInstanceID ||
				first.ReservedAmount != 10 || first.SealDigest != seal.CanonicalDigest {
				t.Fatalf("preview shape=%+v", first)
			}
			switch scenario {
			case "eligible":
				if first.Eligibility != "ELIGIBLE" || first.TotalOre != 10 || len(first.Grants) != 1 {
					t.Fatalf("eligible preview=%+v", first)
				}
			case "debt":
				if first.Eligibility != "RECOVERY_DEBT_PAUSED" || first.TotalOre != 10 || len(first.Grants) != 1 {
					t.Fatalf("debt preview=%+v", first)
				}
			case "zero-power":
				if first.Eligibility != "NO_ELIGIBLE_POWER" || first.TotalOre != 0 || len(first.Grants) != 0 {
					t.Fatalf("zero-power preview=%+v", first)
				}
			}
		})
	}
}
