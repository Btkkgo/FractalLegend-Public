# Economy / 经济

## English — Primary

Fractal Legend separates monetary assets, progression values, reputation, and materials. These categories must not be presented as interchangeable.

| Value | Intended role | Transferability | Current status |
|---|---|---|---|
| FB | Long-term primary economic asset for trading, transfers, deposits and withdrawals | Internal player-Trade settlement is implemented | Internal Ledger and direct player-Trade settlement are **FOUNDATION COMPLETE**; real deposits and withdrawals are **NOT LIVE** |
| Contribution Points | Activation, manufacturing, advanced requirements and system progression | Not freely tradable | **PLANNED** |
| Reputation | Participation, guilds, activities and long-term behavior | Not a withdrawable currency | **PLANNED** |
| Black Iron Ore | Game material for mining, crafting and possible activation/economy rules | Material rules not finalized | **PLANNED** |
| Other materials | Crafting and progression inputs | Defined per future system | **PLANNED** |

### FB

The long-term goal is for FB to support player transfers, settlement, deposits, and withdrawals under auditable rules. G11 established an internal immutable Ledger, and G12 added atomic direct player Trade settlement with Gross postings and 0% fee. No blockchain integration or live production economy is claimed.

Real Fractal Bitcoin deposit, withdrawal, wallet signing, blockchain broadcast, confirmation, reorg handling, Marketplace, and Auction are not implemented.

### Contribution Points

Contribution is intended as non-freely-tradable progression value associated with asset activation, manufacturing, advanced requirements, and system progression. Final issuance and consumption parameters are unresolved.

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
| Contribution Points | Asset Activation、Manufacturing、Advanced Requirement 与 System Progression | 不可自由交易 | **PLANNED** |
| Reputation | Participation、Guild、活动与长期行为 | 不是可提现货币 | **PLANNED** |
| Black Iron Ore | 用于 Mining、Crafting 和可能的 Activation / Economy Rule 的游戏材料 | Material Rule 尚未确定 | **PLANNED** |
| Other Materials | Crafting 与 Progression Input | 由未来系统分别定义 | **PLANNED** |

### FB

长期目标是让 FB 在可审计规则下支持 Player Transfer、Settlement、Deposit 与 Withdrawal。G11 建立内部不可变 Ledger，G12 新增使用 Gross Posting 与 0% Fee 的原子 Direct Player Trade Settlement。本镜像不声称 Blockchain Integration 或 Live Production Economy 已上线。

真实 Fractal Bitcoin Deposit、Withdrawal、Wallet Signing、Blockchain Broadcast、Confirmation、Reorg Handling、Marketplace 和 Auction 均未实现。

### Contribution Points

Contribution 计划作为不可自由交易的成长数值，用于 Asset Activation、Manufacturing、Advanced Requirement 和 System Progression。最终 Issuance 与 Consumption Parameter 尚未确定。

### Reputation

Reputation 计划表达 Participation、Guild Activity、Event 和长期行为。它不是可提现货币。

### Black Iron Ore 与 Mining

Black Iron Ore 是游戏材料，不是 Currency。规划中的 Mining Model 为：

**Global Black Iron Emission Pool → Mining Block / Round → Validated Mining Power → Weighted Reward Distribution**

高级 Mining Tool 可以提高玩家 Mining Power，但不能增加全服总发行量。Parameter 与 Production Implementation 仍处于规划阶段。

### Trade Boundary

G10 已证明具备 Persistent Lock、Revision、Atomic Settlement、Idempotency、Concurrency Protection 和 Restart Recovery 的 Item Ownership Exchange。G12 新增 Revision-bound FB Offer 和一条原子 Item + FB Settlement Path；玩家之间手续费严格为 0%。Option A 不建立 FB Hold，因此 Confirmation 后余额可能变化；Finalize 会锁定并重新验证 Account，余额不足时完整回滚。极端竞争可能耗尽最多三次的有限 Retry，导致安全失败而不产生部分结算。正式 Trade UI、Marketplace、Auction House、Warehouse 与 Durability 仍未实现。
