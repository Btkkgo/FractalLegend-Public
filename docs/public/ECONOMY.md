# Economy / 经济

## English — Primary

Fractal Legend separates monetary assets, progression values, reputation, and materials. These categories must not be presented as interchangeable.

| Value | Intended role | Transferability | Current status |
|---|---|---|---|
| FB | Long-term primary economic asset for trading, transfers, deposits and withdrawals | Internal player-Trade settlement is implemented | Internal Ledger and direct player-Trade settlement are **FOUNDATION COMPLETE**; real deposits and withdrawals are **NOT LIVE** |
| Contribution Points | Future activation, manufacturing, advanced requirements and system progression | Non-transferable | Internal G13 credit-only Ledger **FOUNDATION COMPLETE**; consumption and live issuance absent |
| Reputation | Participation, guilds, activities and long-term behavior | Not a withdrawable currency | **PLANNED** |
| Black Iron Ore | Game material for mining, crafting and possible activation/economy rules | Material rules not finalized | **PLANNED** |
| Other materials | Crafting and progression inputs | Defined per future system | **PLANNED** |

### FB

The long-term goal is for FB to support player transfers, settlement, deposits, and withdrawals under auditable rules. G11 established an internal immutable Ledger, and G12 added atomic direct player Trade settlement with Gross postings and 0% fee. No blockchain integration or live production economy is claimed.

Real Fractal Bitcoin deposit, withdrawal, wallet signing, blockchain broadcast, confirmation, reorg handling, Marketplace, and Auction are not implemented.

### Contribution Points

G13 established a non-transferable, credit-only Contribution Ledger. `CONTRIBUTION_RULE_V1` grants exactly one point per one explicitly eligible FB system-spend unit. Only the internal `SYSTEM_SERVICE` foundation category is eligible; it has no live gameplay producer. Deposit, player Trade, transfers, refunds, recycle, mining, siege, rewards, and admin adjustments earn zero; unknown sources fail closed. Eligible FB debit and Contribution credit share one PostgreSQL transaction. Contribution consumption and the future activation/manufacturing uses are not implemented.

**POST-G13 HARD GATE:** The current block on refund/reversal of a linked eligible spend is temporary fail-closed safety behavior, not a final production design. Atomic FB refund/reversal and Contribution reversal/compensation, including recovery or review if points were already spent, must be implemented before any real eligible spend producer goes live.

### Reputation

Reputation is intended to represent participation, guild activity, events, and sustained behavior. It is not a withdrawable currency.

### Black Iron Ore and mining

Black Iron Ore is a game material, not a currency. The planned mining model is:

**Global Black Iron Emission Pool → Mining Block / Round → Validated Mining Power → Weighted Reward Distribution**

Higher-level mining tools may increase a player's Mining Power, but they must not increase total server issuance. Parameters and production implementation remain planned.

### Trade boundary

G10 proves item ownership exchange with persistent locks, revisions, atomic settlement, idempotency, concurrency protection, and restart recovery. G12 adds revision-bound FB offers and one atomic Item + FB settlement path; the player-to-player fee is exactly 0%. Option A creates no FB Hold, so balances may change after confirmation; Finalize locks and revalidates accounts, rolling back fully on insufficient funds. Extreme contention can exhaust the finite three-attempt retry budget, causing a safe failure without partial settlement. Formal Trade UI, Marketplace, Auction House, Warehouse, and Durability remain unimplemented.

---

## 中文 — 完整对应版本

Fractal Legend 将货币资产、成长数值、Reputation 和 Material 分开设计。这些类别不能被描述为可以相互替代。

| 数值 | 目标用途 | 可转移性 | 当前状态 |
|---|---|---|---|
| FB | 长期主要经济资产，用于交易、转账、充值和提现 | 已实现内部玩家 Trade Settlement | Internal Ledger 与 Direct Player Trade Settlement 已达到 **FOUNDATION COMPLETE**；真实充值与提现 **NOT LIVE** |
| Contribution Points | 未来的 Asset Activation、Manufacturing、Advanced Requirement 与 System Progression | 不可转账 | 内部 G13 仅 Credit 的 Ledger **FOUNDATION COMPLETE**；尚无消耗与真实发放入口 |
| Reputation | Participation、Guild、活动与长期行为 | 不是可提现货币 | **PLANNED** |
| Black Iron Ore | 用于 Mining、Crafting 和可能的 Activation / Economy Rule 的游戏材料 | Material Rule 尚未确定 | **PLANNED** |
| Other Materials | Crafting 与 Progression Input | 由未来系统分别定义 | **PLANNED** |

### FB

长期目标是让 FB 在可审计规则下支持 Player Transfer、Settlement、Deposit 与 Withdrawal。G11 建立内部不可变 Ledger，G12 新增使用 Gross Posting 与 0% Fee 的原子 Direct Player Trade Settlement。本镜像不声称 Blockchain Integration 或 Live Production Economy 已上线。

真实 Fractal Bitcoin Deposit、Withdrawal、Wallet Signing、Blockchain Broadcast、Confirmation、Reorg Handling、Marketplace 和 Auction 均未实现。

### Contribution Points

G13 建立不可转账、仅 Credit 的 Contribution Ledger。`CONTRIBUTION_RULE_V1` 对每 1 单位经显式判定合格的 FB System Spend 恰好发放 1 Point。只有内部 `SYSTEM_SERVICE` Foundation 类别合格，尚无真实游戏 Producer。Deposit、Player Trade、Transfer、Refund、Recycle、Mining、Siege、Reward 和 Admin Adjustment 均发放零分；未知来源默认拒绝。合格 FB Debit 与 Contribution Credit 在同一个 PostgreSQL Transaction 中完成。Contribution Consumption 以及未来的 Activation/Manufacturing 用途尚未实现。

**POST-G13 HARD GATE：** 当前对关联合格消费的 Refund/Reversal Block 是临时 Fail-closed 安全行为，不是最终生产设计。任何真实合格消费 Producer 上线以前，必须实现原子 FB Refund/Reversal 和 Contribution Reversal/Compensation；积分已被消费时还需 Recovery 或 Review 流程。

### Reputation

Reputation 计划表达 Participation、Guild Activity、Event 和长期行为。它不是可提现货币。

### Black Iron Ore 与 Mining

Black Iron Ore 是游戏材料，不是 Currency。规划中的 Mining Model 为：

**Global Black Iron Emission Pool → Mining Block / Round → Validated Mining Power → Weighted Reward Distribution**

高级 Mining Tool 可以提高玩家 Mining Power，但不能增加全服总发行量。Parameter 与 Production Implementation 仍处于规划阶段。

### Trade Boundary

G10 已证明具备 Persistent Lock、Revision、Atomic Settlement、Idempotency、Concurrency Protection 和 Restart Recovery 的 Item Ownership Exchange。G12 新增 Revision-bound FB Offer 和一条原子 Item + FB Settlement Path；玩家之间手续费严格为 0%。Option A 不建立 FB Hold，因此 Confirmation 后余额可能变化；Finalize 会锁定并重新验证 Account，余额不足时完整回滚。极端竞争可能耗尽最多三次的有限 Retry，导致安全失败而不产生部分结算。正式 Trade UI、Marketplace、Auction House、Warehouse 与 Durability 仍未实现。
