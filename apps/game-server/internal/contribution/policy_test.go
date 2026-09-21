package contribution

import (
	"errors"
	"testing"
)

func TestG13EligibilityPolicyCreditsOnlyExplicitSystemService(t *testing.T) {
	policy := EligibilityPolicy{}
	cases := []struct {
		source Source
		want   int64
	}{
		{SourceSystemService, 7},
		{SourceDeposit, 0}, {SourcePlayerTrade, 0}, {SourceMarketplaceTrade, 0},
		{SourcePlayerTransfer, 0}, {SourceRedPacket, 0}, {SourceTip, 0},
		{SourceGuildSalary, 0}, {SourceWithdrawal, 0}, {SourceRefund, 0},
		{SourceEquipmentRecycle, 0}, {SourceMaterialRecycle, 0},
		{SourceMonsterReward, 0}, {SourceBossReward, 0}, {SourceChestReward, 0},
		{SourceDungeonDrop, 0}, {SourceMiningReward, 0}, {SourceSiegeReward, 0},
		{SourceGuildWarReward, 0}, {SourceAdminAdjustment, 0},
	}
	for _, tc := range cases {
		t.Run(string(tc.source), func(t *testing.T) {
			got, err := policy.Evaluate(tc.source, 7, RuleVersionV1)
			if err != nil || got != tc.want {
				t.Fatalf("source=%s got=%d want=%d err=%v", tc.source, got, tc.want, err)
			}
		})
	}
}

func TestG13EligibilityPolicyFailsClosedOnUnknownVersionAndAmount(t *testing.T) {
	policy := EligibilityPolicy{}
	if _, err := policy.Evaluate(Source("CLIENT_CLAIM"), 10, RuleVersionV1); !errors.Is(err, ErrUnknownSource) {
		t.Fatalf("unknown source: %v", err)
	}
	if _, err := policy.Evaluate(SourceSystemService, 10, "CONTRIBUTION_RULE_V2"); !errors.Is(err, ErrRuleVersion) {
		t.Fatalf("rule version: %v", err)
	}
	if _, err := policy.Evaluate(SourceSystemService, 0, RuleVersionV1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("zero amount: %v", err)
	}
	if _, err := policy.Evaluate(SourceSystemService, -1, RuleVersionV1); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative amount: %v", err)
	}
}
