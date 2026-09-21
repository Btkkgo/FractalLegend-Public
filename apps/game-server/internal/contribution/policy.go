package contribution

import "errors"

var (
	ErrUnknownSource = errors.New("unknown contribution source")
	ErrRuleVersion   = errors.New("unsupported contribution rule version")
	ErrInvalidAmount = errors.New("invalid contribution amount")
)

type Source string

const (
	RuleVersionV1 = "CONTRIBUTION_RULE_V1"

	// This is an internal source category, not a live game spending endpoint.
	SourceSystemService    Source = "SYSTEM_SERVICE"
	SourceDeposit          Source = "DEPOSIT"
	SourcePlayerTrade      Source = "PLAYER_TRADE"
	SourceMarketplaceTrade Source = "MARKETPLACE_TRADE"
	SourcePlayerTransfer   Source = "PLAYER_TRANSFER"
	SourceRedPacket        Source = "RED_PACKET"
	SourceTip              Source = "TIP"
	SourceGuildSalary      Source = "GUILD_SALARY"
	SourceWithdrawal       Source = "WITHDRAWAL"
	SourceRefund           Source = "REFUND"
	SourceEquipmentRecycle Source = "EQUIPMENT_RECYCLE"
	SourceMaterialRecycle  Source = "MATERIAL_RECYCLE"
	SourceMonsterReward    Source = "MONSTER_REWARD"
	SourceBossReward       Source = "BOSS_REWARD"
	SourceChestReward      Source = "CHEST_REWARD"
	SourceDungeonDrop      Source = "DUNGEON_DROP"
	SourceMiningReward     Source = "MINING_REWARD"
	SourceSiegeReward      Source = "SIEGE_REWARD"
	SourceGuildWarReward   Source = "GUILD_WAR_REWARD"
	SourceAdminAdjustment  Source = "ADMIN_ADJUSTMENT"
)

// EligibilityPolicy is the sole V1 authority for Contribution issuance.
// Known non-spend sources return zero; unknown sources and rule versions fail closed.
type EligibilityPolicy struct{}

func (EligibilityPolicy) Evaluate(source Source, eligibleFBSpend int64, ruleVersion string) (int64, error) {
	if ruleVersion != RuleVersionV1 {
		return 0, ErrRuleVersion
	}
	if eligibleFBSpend <= 0 {
		return 0, ErrInvalidAmount
	}
	switch source {
	case SourceSystemService:
		return eligibleFBSpend, nil
	case SourceDeposit, SourcePlayerTrade, SourceMarketplaceTrade, SourcePlayerTransfer,
		SourceRedPacket, SourceTip, SourceGuildSalary, SourceWithdrawal, SourceRefund,
		SourceEquipmentRecycle, SourceMaterialRecycle, SourceMonsterReward, SourceBossReward,
		SourceChestReward, SourceDungeonDrop, SourceMiningReward, SourceSiegeReward,
		SourceGuildWarReward, SourceAdminAdjustment:
		return 0, nil
	default:
		return 0, ErrUnknownSource
	}
}
