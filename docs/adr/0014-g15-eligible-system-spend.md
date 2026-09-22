# ADR 0014: G15 Eligible System Spend Orchestrator

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

## English — Primary

### Context and decision

G11 owns the FB Ledger, G13 owns Contribution earning, and G14 owns atomic refund/reversal compensation. G15 connects these foundations through an internal, server-authoritative `systemspend.Service`. A typed `Producer` registry supplies eligibility, the allowed rule version, and refund permissions. The caller supplies an operation ID, player and account identifiers, producer reference, and positive FB amount; it cannot supply eligibility, Contribution amount, or rule version. `Intent` rejects JSON decoding so a network payload cannot become an authorized server intent. There is no gameplay or browser route to this service in G15.

All real gameplay producers are disabled. Only `INTERNAL_TEST_ELIGIBLE` and `INTERNAL_TEST_NON_ELIGIBLE` may be activated for automated tests. The registry rejects attempts to activate a gameplay producer. The existing `contribution.EligibilityPolicy` is the single V1 rule resolver: one eligible FB of `SYSTEM_SERVICE` spend earns one Contribution. Each immutable `system_spends` row snapshots the producer decision, `CONTRIBUTION_RULE_V1`, original amount, refund permissions, FB transaction reference, and Contribution entry reference. A non-eligible internal spend debits FB but stores zero Contribution and no rule version. G12 player trade remains outside this pipeline with zero fee and zero Contribution.

### Transaction and replay invariant

`PostSystemSpend` uses one PostgreSQL transaction. For an eligible intent it calls the existing G13 coordinator, which posts through the G11 FB Ledger and appends the G13 Contribution credit. For a non-eligible intent it calls the existing G11 ledger writer without a Contribution credit. The same transaction inserts the immutable SystemSpend row. Database checks and the insert trigger verify the posted debit, player/account ownership, two balanced FB entries, and the matching Contribution entry when eligible. No parallel ledger or asynchronous credit exists.

An operation advisory lock serializes retries. Replaying the same operation and identical intent returns its existing spend after validating the ledger links; a changed player, account, amount, producer, reference, policy snapshot, or metadata fails closed. A unique producer reference also prevents double use. A failed transaction leaves no FB debit, Contribution credit, or SystemSpend record. An acknowledged or unacknowledged committed transaction can be found after reopening the repository and retried without a second economic effect.

### Refund, reconciliation, and limits

The immutable G15 spend points to the original FB transaction and Contribution entry required by G14. G14 checks the saved `refundable` and `partial_refund_allowed` policy for G15 spends, while historical G13 spends retain their existing behavior. Refund totals and status are derived from G14's immutable compensation rows rather than updating the spend. A 100 FB spend followed by a 40 FB refund leaves net FB spend and net Contribution entitlement of 60. If 80 of the initial 100 Contribution was synthetically consumed in a test fixture, a full refund creates 80 recovery debt; the next 50 eligible credit reduces debt to 30 before any available balance increases. No production Contribution consumption writer is introduced.

G15 reconciliation validates each spend's FB debit, Contribution credit, ownership, original rule version and amount, refund bounds, compensation total, and refund policy. It detects orphan G15 credits/debits and reuses G11/G14 account reconciliation for FB conservation and recovery debt. Mismatches are reported; immutable history is not rewritten. These checks are an internal audit path, not an automated repair or a public economy interface.

G15 does not enable NPC consumption, equipment upgrades, manufacturing, mining-tool crafting, Ordinals activation, shop purchases, chain deposits or withdrawals, or any live player economy entry. Windows runtime execution is a separate acceptance observation from Windows cross-compilation. Technical and manual acceptance passed. Stage Close authorizes reviewed commit, PR integration, Issue closure, and sanitized Public Mirror sync under their final safety gates; X publication remains manual and unauthorized.

---

## 中文 — 完整对应版本

### 背景与决策

G11 管理 FB Ledger，G13 管理 Contribution 收益，G14 管理原子化退款与冲正补偿。G15 通过内部、由服务器权威控制的 `systemspend.Service` 连接这些基础。Typed `Producer` Registry 提供资格、允许的规则版本和退款权限。调用方只提供 Operation ID、玩家与账户标识、Producer Reference 和正数 FB 金额；不能提供资格、Contribution 数量或规则版本。`Intent` 拒绝 JSON 解码，使网络请求无法直接成为获授权的服务端 Intent。G15 没有将该服务连接到 Gameplay 或浏览器路由。

所有真实 Gameplay Producer 均处于 Disabled 状态。仅 `INTERNAL_TEST_ELIGIBLE` 和 `INTERNAL_TEST_NON_ELIGIBLE` 可以为自动测试启用。Registry 会拒绝启用真实 Gameplay Producer。现有 `contribution.EligibilityPolicy` 是 V1 唯一规则解析器：`SYSTEM_SERVICE` 的每 1 单位合格 FB 消费产生 1 Contribution。每条不可变的 `system_spends` 记录保存 Producer 决策、`CONTRIBUTION_RULE_V1`、原始金额、退款权限、FB Transaction Reference 和 Contribution Entry Reference。非合格内部消费会扣除 FB，但保存零 Contribution 和空规则版本。G12 玩家交易仍在该流程之外，手续费和 Contribution 均为零。

### 事务与重试不变量

`PostSystemSpend` 使用一个 PostgreSQL Transaction。合格 Intent 调用现有 G13 Coordinator，由其通过 G11 FB Ledger 记账并追加 G13 Contribution Credit。非合格 Intent 调用现有 G11 Ledger Writer，不发放 Contribution。同一 Transaction 插入不可变的 SystemSpend 记录。数据库约束及插入 Trigger 核验已入账的扣款、玩家与账户归属、两条守恒的 FB Entry，以及合格消费对应的 Contribution Entry。没有平行 Ledger 或异步 Credit。

Operation Advisory Lock 对重试进行串行化。同一 Operation 与相同 Intent 的重放，在验证 Ledger 关联后返回原有 Spend；玩家、账户、金额、Producer、Reference、策略快照或 Metadata 改变时默认拒绝。唯一 Producer Reference 也防止重复使用。失败的 Transaction 不留下 FB Debit、Contribution Credit 或 SystemSpend Record。已经 Commit 但响应丢失的 Transaction 可在重新打开 Repository 后找到；重试不会产生第二次经济效果。

### 退款、对账与边界

不可变的 G15 Spend 指向 G14 所需的原始 FB Transaction 与 Contribution Entry。G14 对 G15 Spend 检查保存的 `refundable` 与 `partial_refund_allowed` 策略，历史 G13 Spend 则保持原有行为。退款总额和状态由 G14 不可变 Compensation Row 推导，不更新 Spend。100 FB 消费后退款 40 FB，最终净 FB 消费与净 Contribution 权益均为 60。如果测试 Fixture 模拟先使用最初 100 Contribution 中的 80，完整退款会产生 80 Recovery Debt；下一次 50 合格 Credit 会先把债务降至 30，可用余额不会增加。本阶段没有引入生产用 Contribution Consumption Writer。

G15 Reconciliation 对每笔 Spend 核查 FB Debit、Contribution Credit、归属、原始规则版本与金额、退款上限、补偿总额和退款策略。它发现孤立的 G15 Credit/Debit，并复用 G11/G14 的账户级对账检查 FB 守恒与 Recovery Debt。异常会被报告，不会改写不可变历史。这是一条内部审计路径，不是自动修复或公开经济入口。

G15 不启用 NPC 消费、装备强化、制造、Mining Tool Craft、Ordinals Activation、商店购买、链上充值或提现，也不启用真实玩家经济入口。Windows Runtime 执行结果必须与 Windows Cross Compile 分开记录。技术验收与人工验收均已通过。Stage Close 授权在最终安全 Gate 下执行经审查的 Commit、PR 集成、Issue 关闭和脱敏 Public Mirror 同步；X 发布仍须用户手动决定，本轮未获授权。
