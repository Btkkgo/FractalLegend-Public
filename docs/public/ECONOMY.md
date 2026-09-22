# Economy / 经济

## English — Primary

Fractal Legend separates monetary assets, progression values, reputation, and materials. These categories must not be presented as interchangeable.

| Value | Intended role | Transferability | Current status |
|---|---|---|---|
| FB | Long-term primary economic asset for trading, transfers, deposits and withdrawals | Internal player-Trade settlement is implemented | Internal Ledger and direct player-Trade settlement are **FOUNDATION COMPLETE**; real deposits and withdrawals are **NOT LIVE** |
| Contribution Points | Future activation, manufacturing, advanced requirements and system progression | Non-transferable | Internal G13–G14 Ledger and refund recovery **FOUNDATION COMPLETE**; production consumption and live issuance absent |
| Reputation | Participation, guilds, activities and long-term behavior | Non-transferable and not withdrawable | Internal G16 account and recycle output **FOUNDATION COMPLETE**; production rules and final caps absent |
| Black Iron Ore | Game material for mining, crafting and possible activation/economy rules | Material rules not finalized | **PLANNED** |
| Black Iron emission capacity | Global ceiling derived from eligible system spending | Not a player asset or transferable balance | G17 internal PostgreSQL foundation complete; no ore issued |
| Mining Block reward reservation | Capacity committed to a server-owned block | Not distributed reward or player ore | G18 internal foundation complete; production parameters undecided |
| Other materials | Crafting and progression inputs | Defined per system | G16 synthetic recycle material foundation complete; production material economy pending |

### FB

The long-term goal is for FB to support player transfers, settlement, deposits, and withdrawals under auditable rules. G11 established an internal immutable Ledger, and G12 added atomic direct player Trade settlement with Gross postings and 0% fee. No blockchain integration or live production economy is claimed.

Real Fractal Bitcoin deposit, withdrawal, wallet signing, blockchain broadcast, confirmation, reorg handling, Marketplace, and Auction are not implemented.

### Contribution Points

G13 established a non-transferable, credit-only Contribution Ledger. `CONTRIBUTION_RULE_V1` grants exactly one point per one explicitly eligible FB system-spend unit. Only the internal `SYSTEM_SERVICE` foundation category is eligible; it has no live gameplay producer. Deposit, player Trade, transfers, refunds, recycle, mining, siege, rewards, and admin adjustments earn zero; unknown sources fail closed. Eligible FB debit and Contribution credit share one PostgreSQL transaction. Contribution consumption and the future activation/manufacturing uses are not implemented.

**G14 hard gate closed:** The internal coordinator now atomically posts linked FB refund/reversal and Contribution compensation, including bounded partial refunds, recovery debt for already-used points, future-credit debt repayment, and review hold. Ordinary Ledger refunds remain fail-closed; there is no real eligible spend producer or production Contribution spend writer.

### Reputation

Reputation is intended to represent participation, guild activity, events, and sustained behavior. G16 added a non-transferable internal account and immutable recycle credit, with production rules disabled. Final caps and diminishing returns remain undecided. Recycle creates no FB, Contribution, or Black Iron Ore and returns no more than 50% of verified same-material input in the internal rule boundary.

### Black Iron Ore and mining

Black Iron Ore is a game material, not a currency. The planned mining model is:

G17 records global **emission capacity**, not Black Iron Ore held by players. G18 atomically reserves some capacity for a server-owned Mining Block, retaining it on finalization or releasing it on cancellation. An upstream refund consumes Remaining and records a shortfall as Recovery Debt; future emission and cancellation repay debt first. Reconciliation enforces `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`; G18 Distributed remains zero. `DEV_G17_1_TO_1` and `DEV_G18_FIXED_BLOCK_REWARD` are only test/development rules. Production emission ratio, block reward, and duration remain **NOT FINALIZED**.

**Global Black Iron Emission Pool → Mining Block / Round → Validated Mining Power → Weighted Reward Distribution**

Future mining tools and player power require separate design; no player mining or reward allocation is implemented in G18.

### Trade boundary

G10 proves item ownership exchange with persistent locks, revisions, atomic settlement, idempotency, concurrency protection, and restart recovery. G12 adds revision-bound FB offers and one atomic Item + FB settlement path; the player-to-player fee is exactly 0%. Option A creates no FB Hold, so balances may change after confirmation; Finalize locks and revalidates accounts, rolling back fully on insufficient funds. Extreme contention can exhaust the finite three-attempt retry budget, causing a safe failure without partial settlement. Formal Trade UI, Marketplace, Auction House, Warehouse, and Durability remain unimplemented.

---

## 中文 — 完整对应版本

Fractal Legend 将货币资产、成长数值、Reputation 和 Material 分开设计。这些类别不能被描述为可以相互替代。

| 数值 | 目标用途 | 可转移性 | 当前状态 |
|---|---|---|---|
| FB | 长期主要经济资产，用于交易、转账、充值和提现 | 已实现内部玩家 Trade Settlement | Internal Ledger 与 Direct Player Trade Settlement 已达到 **FOUNDATION COMPLETE**；真实充值与提现 **NOT LIVE** |
| Contribution Points | 未来的 Asset Activation、Manufacturing、Advanced Requirement 与 System Progression | 不可转账 | 内部 G13–G14 Ledger 与 Refund Recovery **FOUNDATION COMPLETE**；尚无生产用消耗与真实发放入口 |
| Reputation | Participation、Guild、活动与长期行为 | 不可转账、不可提现 | G16 内部账户与回收产出已达 **FOUNDATION COMPLETE**；没有生产规则与最终上限 |
| Black Iron Ore | 用于 Mining、Crafting 和可能的 Activation / Economy Rule 的游戏材料 | Material Rule 尚未确定 | **PLANNED** |
| 黑铁矿石发行额度 | 由合格系统消费决定的全服上限 | 不是玩家资产或可转移余额 | G17 内部 PostgreSQL 基础已完成；未发放矿石 |
| Mining Block 奖励预留 | 服务器区块所承诺的容量 | 不是已分发奖励或玩家矿石 | G18 内部基础已完成；正式参数未定 |
| Other Materials | Crafting 与 Progression Input | 由各系统定义 | G16 合成测试材料回收基础已完成；生产材料经济仍待确定 |

### FB

长期目标是让 FB 在可审计规则下支持 Player Transfer、Settlement、Deposit 与 Withdrawal。G11 建立内部不可变 Ledger，G12 新增使用 Gross Posting 与 0% Fee 的原子 Direct Player Trade Settlement。本镜像不声称 Blockchain Integration 或 Live Production Economy 已上线。

真实 Fractal Bitcoin Deposit、Withdrawal、Wallet Signing、Blockchain Broadcast、Confirmation、Reorg Handling、Marketplace 和 Auction 均未实现。

### Contribution Points

G13 建立不可转账、仅 Credit 的 Contribution Ledger。`CONTRIBUTION_RULE_V1` 对每 1 单位经显式判定合格的 FB System Spend 恰好发放 1 Point。只有内部 `SYSTEM_SERVICE` Foundation 类别合格，尚无真实游戏 Producer。Deposit、Player Trade、Transfer、Refund、Recycle、Mining、Siege、Reward 和 Admin Adjustment 均发放零分；未知来源默认拒绝。合格 FB Debit 与 Contribution Credit 在同一个 PostgreSQL Transaction 中完成。Contribution Consumption 以及未来的 Activation/Manufacturing 用途尚未实现。

**G14 硬门已关闭：** 内部 Coordinator 现可原子写入关联 FB Refund/Reversal 与 Contribution Compensation，包括有上限的部分退款、已使用积分的 Recovery Debt、未来 Credit 先偿债及 Review Hold。普通 Ledger 退款仍默认拒绝；尚无真实合格消费 Producer 或生产用 Contribution Spend Writer。

### Reputation

Reputation 计划表达 Participation、Guild Activity、Event 和长期行为。G16 新增不可转账的内部账户及不可变回收 Credit，生产规则保持禁用。最终上限与递减收益尚未决定。回收不创建 FB、Contribution 或黑铁矿石；内部规则边界中，同种材料返还不超过可核验投入的 50%。

### Black Iron Ore 与 Mining

Black Iron Ore 是游戏材料，不是 Currency。规划中的 Mining Model 为：

G17 记录全服**发行额度**，而不是玩家持有的黑铁矿石。G18 从中为服务器区块原子预留容量，完成时保留、取消时释放。上游退款先消耗 Remaining，缺口记为 Recovery Debt；未来发行和取消释放先还债。对账执行 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`；G18 的已分发为零。`DEV_G17_1_TO_1` 和 `DEV_G18_FIXED_BLOCK_REWARD` 只用于测试／开发，正式发行比例、区块奖励及时长均**尚未确定**。

**Global Black Iron Emission Pool → Mining Block / Round → Validated Mining Power → Weighted Reward Distribution**

未来挖矿工具和玩家算力需要单独设计；G18 不实现玩家挖矿或奖励分配。

### Trade Boundary

G10 已证明具备 Persistent Lock、Revision、Atomic Settlement、Idempotency、Concurrency Protection 和 Restart Recovery 的 Item Ownership Exchange。G12 新增 Revision-bound FB Offer 和一条原子 Item + FB Settlement Path；玩家之间手续费严格为 0%。Option A 不建立 FB Hold，因此 Confirmation 后余额可能变化；Finalize 会锁定并重新验证 Account，余额不足时完整回滚。极端竞争可能耗尽最多三次的有限 Retry，导致安全失败而不产生部分结算。正式 Trade UI、Marketplace、Auction House、Warehouse 与 Durability 仍未实现。
