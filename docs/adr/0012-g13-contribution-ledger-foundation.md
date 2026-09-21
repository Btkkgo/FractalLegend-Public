# ADR 0012: G13 Contribution Ledger Foundation

Status: **Accepted · Technical acceptance PASS · Manual acceptance PASS**

Tracking: private canonical Issue #10 (archive reference only) / 私有 Canonical Issue #10（仅档案引用）

## English — Primary

### Context and decision

G11 owns the immutable FB Ledger; G12 uses it for atomic player Trade. Contribution is a separate, non-transferable progression and eligibility value. It is not a second FB balance or a currency. G13 establishes a credit-only V1 Contribution Ledger and a single internal coordinator that posts an authorized FB system spend and its Contribution credit in **one PostgreSQL transaction**. No browser command or client-supplied “I spent FB” claim can mint points.

### Versioned eligibility

`CONTRIBUTION_RULE_V1` grants exactly **1 Contribution Point per 1 eligible FB system-spend unit**. `EligibilityPolicy` is the only issuance rule. V1 explicitly allows the internal `SYSTEM_SERVICE` source category for the foundation's synthetic tests; no live game service producer is connected. The category cannot be inferred from an arbitrary debit or the G11 `ELIGIBLE` classification alone. A future producer must be separately authorized, validate its game rule, and use the coordinator.

Known non-eligible sources return zero Contribution: Deposit, Player Trade, Marketplace Trade, Player Transfer, Red Packet, Tip, Guild Salary, Withdrawal, Refund, Equipment/Material Recycle, Monster/Boss/Chest/Dungeon Reward, Mining Reward, Siege Reward, Guild War Reward, and Admin Adjustment. Unknown sources, unsupported rule versions, nonpositive amounts, invalid IDs, wrong FB account ownership, and mismatched posting intents fail closed. G12 player Trade at a 0% fee never calls the Contribution coordinator and credits both players zero.

### Account, entry, and source identity

`contribution_accounts` stores player ID, nonnegative balance, revision, and timestamps. Immutable `contribution_entries` store the source and source ID, eligible FB spend, credited amount, balance before/after, rule version, linked FB transaction ID, account IDs, and creation time. A database unique constraint on `(source, source_id, rule_version)` makes the business event idempotent across retries and restarts. A repeated identical request returns the stored posting; an altered amount, player, or FB account for the same source fails as a conflict. Entries link to immutable FB history.

### Atomic posting and concurrency

The coordinator uses a `READ COMMITTED` PostgreSQL transaction with a transaction-scoped source advisory lock, the existing G11 FB posting helper and deterministic FB account row locks, followed by a Contribution account row lock and revision-checked update. G11's helper performs the actual balanced FB debit/credit; G13 never creates a second FB balance. It then inserts the Contribution entry, updates the balance and revision, writes an audit event, and commits. Any error rolls back FB and Contribution together. Serialization/deadlock errors use a finite, bounded retry; idempotent source identity handles replay after an uncertain result. Row locking was selected after a 100-concurrent-spend test exposed excessive serializable snapshot conflicts; the row-lock approach passed repeated contention tests.

The migration enforces V1 source, version, positive amount, and 1:1 ratio. A database trigger verifies that each Contribution entry points to the matching posted, eligible `SYSTEM_SPEND` FB transaction with exactly one player debit and one system credit for the same amount and source reference. It rejects a forged link to an unrelated FB transaction. The entry and audit history are immutable.

### Refund boundary and reconciliation

V1 implements **credits only**; it does not implement Contribution spending or reversal. A refund or reversal of a G13-linked FB spend would otherwise leave unearned points. The shared FB posting path therefore rejects compensation of `CONTRIBUTION_SYSTEM_SPEND` until a separately designed atomic Contribution compensation exists. A refund is itself non-eligible and never earns points. Reconciliation compares each Contribution account balance with the sum of its immutable legal entries and reports mismatches without silent repair. Audit queries expose the source, rule version, amount, and linked FB transaction.

**POST-G13 HARD GATE:** The current fail-closed block on refund/reversal of a linked eligible system spend is a temporary safety measure, **not a final production design**. Before any real NPC, enhancement, manufacturing, asset activation, mining-tool, or other eligible system-spend producer is connected, FB refund/reversal and Contribution reversal/compensation must be designed and implemented atomically. A successful FB refund must not leave the original Contribution permanently credited. If those points have already been spent, the design must specify an explicit compensation, recovery, or review process. This gate is recorded here; G13 does not implement that future behavior.

### Failure and scope boundaries

Fault injection covers before FB debit, after FB debit, before/after Contribution entry, after account update, before final state, and before commit. The design must leave no partial debit, credit, entry, audit event, or balance change. The Contribution service exposes no player Transfer, Send, Trade, Deposit, Withdraw, or direct FB exchange method. G13 adds no Mining, Black Iron Ore, Recycle Migration, Marketplace, Blockchain, Wallet, Deposit/Withdrawal, Ordinals, Guild Salary, Red Packet, Tip, Siege Reward, or production game spending producer.

### Known risks

The retry budget is finite, so extreme contention can still return a safe failure without partial posting. The only V1 eligible category has no live producer; future game services require explicit eligibility review and an atomic integration. G13-linked FB refunds/reversals are deliberately blocked until an atomic Contribution compensation design is accepted.

---

## 中文 — 完整对应版本

### 背景与决策

G11 负责不可变 FB Ledger；G12 把它用于原子 Player Trade。Contribution 是独立、不可转账的成长与资格数值，不是第二套 FB Balance，也不是货币。G13 建立 V1 仅 Credit 的 Contribution Ledger，并通过一个内部 Coordinator，在**同一个 PostgreSQL Transaction** 中写入已授权的 FB System Spend 与对应 Contribution Credit。Browser Command 或客户端提交的“我消费了 FB”声明不能铸造积分。

### 版本化资格规则

`CONTRIBUTION_RULE_V1` 规定：**1 单位符合资格的 FB System Spend 产生 1 Contribution Point**。`EligibilityPolicy` 是唯一的发放规则。V1 只显式允许内部 `SYSTEM_SERVICE` 来源类别，用于 Foundation 的 Synthetic Test；尚未连接任何真实游戏服务 Producer。不能仅凭任意 Debit 或 G11 的 `ELIGIBLE` Classification 推断资格。未来 Producer 必须另行授权、验证其游戏规则并使用 Coordinator。

已知非合格来源均产生零 Contribution：Deposit、Player Trade、Marketplace Trade、Player Transfer、Red Packet、Tip、Guild Salary、Withdrawal、Refund、Equipment/Material Recycle、Monster/Boss/Chest/Dungeon Reward、Mining Reward、Siege Reward、Guild War Reward 与 Admin Adjustment。未知来源、不支持的 Rule Version、非正 Amount、无效 ID、错误的 FB Account Ownership，以及同一 Source 的不一致 Posting Intent 均默认拒绝。G12 以 0% Fee 执行的 Player Trade 不调用 Contribution Coordinator，双方积分均增加零。

### Account、Entry 与 Source Identity

`contribution_accounts` 保存 Player ID、非负 Balance、Revision 和时间戳。不可变的 `contribution_entries` 保存 Source 与 Source ID、符合资格的 FB Spend、Credit Amount、前后 Balance、Rule Version、关联 FB Transaction ID、Account ID 和创建时间。数据库对 `(source, source_id, rule_version)` 设置 Unique Constraint，使业务事件在 Retry 与 Restart 后保持幂等。完全相同的请求返回已存 Posting；同一 Source 的 Amount、Player 或 FB Account 被改动则产生 Conflict。Entry 关联不可变的 FB History。

### 原子 Posting 与并发

Coordinator 使用 `READ COMMITTED` PostgreSQL Transaction，先获取 Transaction-scoped Source Advisory Lock，再复用 G11 FB Posting Helper 和确定性的 FB Account Row Lock，随后锁定 Contribution Account Row，并以 Revision 条件更新。实际守恒的 FB Debit/Credit 由 G11 Helper 完成；G13 不建立第二套 FB Balance。之后写入 Contribution Entry、更新 Balance/Revision、写入 Audit Event 并 Commit。任一步骤出错，FB 与 Contribution 一起 Rollback。Serialization/Deadlock Error 使用有限 Retry；稳定的 Source Identity 处理结果不确定后的 Replay。100 并发消费测试曾发现 Serializable Snapshot Conflict 过多，因此选用 Row Lock 方案，并通过重复竞争测试。

Migration 对 V1 Source、Version、正 Amount 与 1:1 Ratio 做约束。数据库 Trigger 核验每条 Contribution Entry 均对应同一 Source Reference 下、已 Posted 且 Eligible 的 `SYSTEM_SPEND` FB Transaction；该 FB Transaction 必须恰有一条同额 Player Debit 与一条 System Credit。伪造对无关 FB Transaction 的引用会被拒绝。Entry 与 Audit History 不可变。

### Refund 边界与 Reconciliation

V1 **只有 Credit**，不实现 Contribution Spend 或 Reversal。若直接 Refund 或 Reverse G13 关联的 FB Spend，会留下不再有对应最终消费的积分。因此，在单独设计原子 Contribution Compensation 之前，共享 FB Posting Path 会拒绝对 `CONTRIBUTION_SYSTEM_SPEND` 的补偿。Refund 本身属于非合格来源，永远不产生积分。Reconciliation 比较每个 Contribution Account Balance 与其不可变合法 Entry 之和，报告差异，不进行静默修复。Audit Query 提供 Source、Rule Version、Amount 与关联 FB Transaction。

**POST-G13 HARD GATE：** 当前对关联合格 System Spend 的 Refund/Reversal 采用 Fail-closed Temporary Block；这是**临时安全策略，不是最终生产设计**。接入任何真实 NPC、Enhancement、Manufacturing、Asset Activation、Mining Tool 或其他合格 System Spend Producer 之前，必须先设计并实现 FB Refund/Reversal 与 Contribution Reversal/Compensation 的原子一致性。FB 退款成功后，原 Contribution 不得永久留存。如果积分已经被消费，设计必须规定明确的 Compensation、Recovery 或 Review 流程。本 ADR 记录这一强制 Gate；G13 本轮不实现未来方案。

### 失败与范围边界

Fault Injection 覆盖 FB Debit 前、FB Debit 后、Contribution Entry 前后、Account Update 后、Final State 前和 Commit 前。该设计不得留下部分 Debit、Credit、Entry、Audit Event 或 Balance Change。Contribution Service 不暴露玩家 Transfer、Send、Trade、Deposit、Withdraw 或直接兑换 FB 的方法。G13 不新增 Mining、Black Iron Ore、Recycle Migration、Marketplace、Blockchain、Wallet、Deposit/Withdrawal、Ordinals、Guild Salary、Red Packet、Tip、Siege Reward 或真实游戏消费 Producer。

### 已知风险

Retry 次数有限，极端竞争仍可能安全失败，但不会产生部分 Posting。V1 唯一合格类别尚无真实 Producer；未来游戏服务必须显式审核资格规则并进行原子集成。G13 关联的 FB Refund/Reversal 有意保持拒绝，直到原子 Contribution Compensation 设计通过验收。
