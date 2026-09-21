# ADR 0013: G14 Contribution Refund / Reversal Atomic Compensation

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

## English — Primary

### Problem and threat model

G13 could credit Contribution for an internal eligible FB system spend, but its shared FB Ledger path blocked refund and reversal. Refunding FB without reclaiming the related Contribution would let a player retain a benefit at zero net FB cost. The harder case is Contribution already consumed before the refund. Client-forged references, replay, concurrent partial refunds, cross-player requests, corrupted linkage, and changed rule versions must all fail closed. G14 supplies the compensation foundation before any live eligible spend producer is connected.

### Decision and economic invariants

`contribution.Service.RefundSystemSpend` accepts an original FB transaction ID, the expected player and FB account, a stable refund reference, a positive amount, and refund/reversal kind. It accepts no client-computed Contribution amount or debt. The PostgreSQL coordinator requires a posted `ELIGIBLE` `SYSTEM_SPEND` with the G13 Contribution reference, matching immutable Contribution entry, player ownership, original FB debit/system credit, and the original `rule_version`. `CONTRIBUTION_RULE_V1` remains 1:1; unsupported historical versions are rejected rather than recomputed with a newer rule. Non-eligible FB spends and transfers cannot enter this coordinator and continue to produce zero Contribution.

The sum of refunds for one original spend cannot exceed its eligible FB amount. Each successful refund appends an immutable `contribution_compensations` row linked to the original Contribution entry and both FB transactions. A full 100 FB refund reverses 100 Contribution of economic entitlement. A 40 FB partial refund reverses 40, leaving 60. Multiple partial refunds are serialized and bounded by the original amount. The existing G11 single-compensation constraint remains for all non-G14 references; only `CONTRIBUTION_REFUND` permits multiple bounded partial refunds.

### Recovery debt, hold, and future credits

`contribution_accounts.balance` remains nonnegative available Contribution. `recovery_debt` is a separate, nontransferable recovery obligation. When available balance is smaller than the required compensation, the coordinator reclaims all available points and creates debt for the remainder. For example, 100 earned, 80 synthetically consumed, and a full refund yields balance 0 and debt 80. Every later eligible credit settles debt **inside its original FB debit + Contribution credit transaction** before any remainder becomes available. The immutable earning entry records `debt_settled`; the original G13 earning entry is never edited. `ValidateSpend` is an internal server-side rule check: debt, review-required state, insufficient balance, and ledger drift block spending. G14 exposes no production Contribution spend writer or game purchase producer. The immutable `contribution_consumptions` relation is exercised only by a synthetic test fixture to model already-used points.

### Atomicity, idempotency, and lock order

One `READ COMMITTED` PostgreSQL transaction takes a refund-reference advisory lock, locks the original FB transaction, checks cumulative refund and linkage, posts the balanced FB refund/reversal through the existing G11 helper, locks the Contribution account, appends the compensation, updates available balance/debt/revision, and commits. FB accounts are locked in G11's sorted order. A deferred database trigger rejects any G13-linked FB refund without its matching Contribution compensation, including direct SQL that bypasses the application helper. A compensation trigger checks FB linkage and cumulative amount; immutable triggers prohibit update/delete. A duplicate reference returns the stored result without applying a second effect; conflicting intent fails. Deadlock/serialization retry is bounded and safe.

Failures after FB posting, after Contribution compensation, and before commit roll back both ledgers. Reopening the repository and replaying the same reference observes the committed result. No asynchronous debt settlement or second FB Ledger exists.

### Reconciliation and manual review

Reconciliation reads a consistent PostgreSQL snapshot. It compares available balance with earned credit minus debt settlement, synthetic consumption, and available reversal; debt with created debt minus settlement; FB refund totals with Contribution compensation; per-original totals with original eligible spend; and compensation-to-FB links and rule versions. A mismatch sets `review_required` and blocks sensitive Contribution operations. The coordinator also checks the target account and original spend before posting. It never guesses a historical repair or mutates immutable entries.

### Limits and next gate

G14 is an internal safety foundation, not a release or a live economy. `ValidateSpend` is a rule check, not a spending transaction; a future authorized spend producer must integrate it with its own atomic debit and consumption entry. The internal `SYSTEM_SERVICE` source still has no live producer. No Marketplace, Mining, Wallet, Blockchain, Ordinals, Deposit, Withdrawal, social feature, or G15 implementation is included. Manual acceptance has passed; Stage Close authorizes Commit, Push, PR, Issue closure, and reviewed Public Mirror sync. X publication remains manual and unauthorized.

---

## 中文 — 完整对应版本

### 问题与威胁模型

G13 可以为内部合格 FB System Spend 发放 Contribution，但共享 FB Ledger 路径会阻止退款和冲正。如果退还 FB 却不回收关联 Contribution，玩家便可在最终零 FB 成本下保留权益。更困难的情形是退款前 Contribution 已被使用。客户端伪造引用、重放、并发部分退款、跨玩家请求、关联数据损坏及规则版本变化都必须默认拒绝。G14 在连接任何真实合格消费 Producer 前建立补偿基础。

### 决策与经济不变量

`contribution.Service.RefundSystemSpend` 接收原 FB Transaction ID、预期玩家与 FB Account、稳定的退款 Reference、正数金额及 Refund/Reversal 类型；不接受客户端计算的 Contribution 数量或债务。PostgreSQL Coordinator 要求原交易是已入账的 `ELIGIBLE` `SYSTEM_SPEND`，使用 G13 Contribution Reference，并匹配不可变 Contribution Entry、玩家归属、原始 FB 扣款与 System 入账，以及原始 `rule_version`。`CONTRIBUTION_RULE_V1` 仍为 1:1；不支持的历史版本会被拒绝，不能用新规则重算。非合格 FB 消费和 Transfer 不能进入此 Coordinator，Contribution 增量继续为零。

同一原始消费的累计退款不得超过其合格 FB 金额。每次成功退款都会追加不可变的 `contribution_compensations` 行，关联原 Contribution Entry 和两笔 FB Transaction。完整退回 100 FB 将冲正 100 Contribution 经济权益；部分退款 40 FB 则冲正 40，留下 60。多次部分退款会串行化，并以原始金额为上限。G11 原有单次补偿约束对所有非 G14 Reference 保持不变；只有 `CONTRIBUTION_REFUND` 允许有总额上限的多次部分退款。

### 恢复债务、消费冻结与未来收益

`contribution_accounts.balance` 始终表示非负的可用 Contribution。`recovery_debt` 是独立、不可转让的恢复义务。可用余额不足以覆盖补偿时，Coordinator 收回全部可用积分，剩余部分记为债务。例如获得 100、在 Synthetic Fixture 中已使用 80、随后完整退款，结果为余额 0、债务 80。以后的每笔合格 Credit 都在其原本的 FB Debit + Contribution Credit Transaction **内部**先偿债，剩余部分才进入可用余额。不可变的收益 Entry 记录 `debt_settled`；G13 原始收益 Entry 永不修改。`ValidateSpend` 是内部服务端规则检查：存在债务、需要人工复核、余额不足或账本漂移时均阻止消费。G14 不暴露生产用 Contribution Spend Writer 或游戏购买 Producer。不可变的 `contribution_consumptions` 关系仅由 Synthetic Test Fixture 使用，以模拟此前已用积分。

### 原子性、幂等与锁顺序

单个 `READ COMMITTED` PostgreSQL Transaction 依次取得退款 Reference Advisory Lock、锁定原 FB Transaction、核验累计退款和关联关系、通过现有 G11 Helper 写入守恒的 FB Refund/Reversal、锁定 Contribution Account、追加补偿记录、更新可用余额/债务/Revision，最后 Commit。FB Account 按 G11 的排序顺序加锁。Deferred Database Trigger 拒绝任何缺少匹配 Contribution 补偿的 G13 关联 FB 退款，包括绕过应用 Helper 的直接 SQL。补偿 Trigger 检查 FB 关联及累计金额；不可变 Trigger 禁止更新和删除。重复 Reference 返回已存结果，不再次改变经济状态；同一 Reference 的冲突 Intent 会失败。Deadlock/Serialization Retry 有界且安全。

FB Posting 后、Contribution Compensation 后及 Commit 前的故障都使两个账本一起回滚。重新打开 Repository 后使用同一 Reference 重放，会看到已提交的结果。不使用异步偿债，也不建立第二套 FB Ledger。

### 对账与人工复核

Reconciliation 读取一致的 PostgreSQL Snapshot。它对比可用余额与收益减去偿债、Synthetic Consumption 和可用余额冲正后的数值；对比债务创建与偿还；对比 FB 退款总额与 Contribution 补偿；核查每笔原始消费的总额上限、补偿到 FB 的关联以及规则版本。出现不一致时设置 `review_required` 并阻止敏感 Contribution 操作。Coordinator 在入账前也检查目标 Account 和原始消费。系统不会猜测历史修复，也不会修改不可变 Entry。

### 限制与下一个 Gate

G14 是内部安全基础，不是 Release 或真实经济上线。`ValidateSpend` 是规则检查，不是消费 Transaction；未来经单独授权的 Spend Producer 必须把它和自身原子扣减及 Consumption Entry 集成。内部 `SYSTEM_SERVICE` 来源仍没有真实 Producer。本轮不包含 Marketplace、Mining、Wallet、Blockchain、Ordinals、Deposit、Withdrawal、社交功能或 G15 实现。人工验收已通过；Stage Close 已授权 Commit、Push、PR、Issue 关闭及经审查的 Public Mirror Sync。X 发布仍由用户手动决定，本轮未获授权。
