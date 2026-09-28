# Economy / 经济

## Current G21-P0 acceptance / 当前 G21-P0 验收

**G21-P0 Settlement Prerequisite Foundation: final human acceptance PASS; private canonical closure complete.** All 38 prerequisite scenarios passed. Native Windows, Linux and macOS produced the same canonical digest. The accepted scope adds a G20 input Seal, immutable reservation and CharacterID beneficiary bindings, Distributed/RecoveryDebt accounting, distinct `MINING_REWARD` source and inventory projection prerequisites, and a 500 participant TEST boundary. The public subset is verified independently in the [G21-P0 export record](../public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md).

These are prerequisite contracts and TEST-only writers. No production route, full reward allocation or settlement, real player ore issuance, claim, or G22 is implemented. **Full G21 implementation remains READY FOR NEXT HUMAN GATE.**

**G21-P0 结算前置契约：最终人工验收 PASS，私有 canonical 已完成收尾。** 38 项前置场景全部通过；Windows、Linux、macOS 产生相同的规范摘要。本阶段新增 G20 输入 Seal、不可变预留与 CharacterID 受益人绑定、Distributed／RecoveryDebt 记账、独立 `MINING_REWARD` 来源及库存投影前置契约，以及 500 人 TEST 边界。公开子集的独立验证记录见[G21-P0 导出记录](../public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md)。

这些是前置契约与仅限 TEST 的写入器。没有生产入口、完整奖励分配或结算、真实玩家矿石发放、领取或 G22。**完整 G21 实施仍为 READY FOR NEXT HUMAN GATE。**

## Historical G20 acceptance / 历史 G20 验收

**G20 Mining Power Foundation: final human acceptance PASS; private canonical closure complete.**

The accepted chain is **G17 Global Mining Pool → G18 Mining Block + Recovery → G18.1 Stable Block Identity → G20 Server-authoritative Mining Power**. G17 capacity/reservation is not player ore. G19 identity migration preserves existing quantities without issuance.

G20 records immutable ValidatedMiningActivity as authority and derives Participant aggregates. Stable BlockInstanceID spans sessions/events/history. G18 uses a read-only ordinary SELECT without upstream row locks or cross-domain foreign keys, saving an immutable validation snapshot. PostgreSQL SERIALIZABLE commits activity and aggregate atomically, with global SourceEvent uniqueness and at most five attempts including the first. Synthetic Tool/Map inputs are server-resolved, final client power is rejected, integer scale is 1,000,000 and one final floor division determines power. Canonical replay timestamps use `t.UTC().Round(0)`; strict receipt equality is retained. Reconciliation is read-only and never repairs unknown history.

G20 has **90/90 scenarios PASS**. Private PR and post-merge CI each have **10/10 jobs PASS**, full Normal and Race each **882/882 PASS**, zero failures/skips/data race. Native Windows/Linux/macOS T50 has 24 identical logical results with digest `f5f2e54d342d02544c54565d488aa1e096fd9208c4ea4d23b291fa2f5ceb3d60`. The public subset is tested independently; its results appear in the public export manifest rather than borrowing private counts.

This is a TEST foundation, not a public game release. Production Service remains fail-closed. **NOT IMPLEMENTED:** reward distribution, block reward settlement, ore claim, mining pool deduction, production reward/duration/tool/map rules and full G21. G20 mutates no G18, FB, Contribution, Player Ore or Mining Pool state. Social material remains an unpublished draft.

**G20 算力基础：最终人工验收 PASS，私有 canonical 收尾完成。**

已验收链条为 **G17 全服矿池 → G18 挖矿区块与恢复 → G18.1 稳定区块身份 → G20 服务器权威算力**。G17 容量／预留不等于玩家矿石；G19 身份迁移保持既有数量，不发行矿石。

G20 以不可变 ValidatedMiningActivity 为权威、Participant 为派生汇总。稳定 BlockInstanceID 贯穿会话／事件／历史。G18 普通只读 SELECT 无上游行锁或跨域外键，保存不可变验证快照。PostgreSQL SERIALIZABLE 原子提交事实与汇总；SourceEvent 全局唯一，含首次最多五次。工具／地图输入由服务器解析合成数据，拒绝客户端最终算力；整数定点比例 1,000,000，只作一次最终除法向下截断。重放时间采用 `t.UTC().Round(0)`，保留回执严格相等。对账只读，不修复未知历史。

G20 **90/90 场景通过**；私有 PR 及合并后 CI 各 **10/10 作业通过**，完整 Normal／Race 各 **882/882**，零失败、跳过、数据竞争。Windows／Linux／macOS 原生 T50 的 24 个逻辑结果完全一致，摘要如上。公开子集独立运行验证，结果在公开导出清单中记录，不借用私有数量。

这是 TEST 基础，不是公开游戏发布；生产 Service 失败关闭。**未实现：**奖励分发、区块奖励结算、矿石申领、矿池扣减、正式奖励／时长／工具／地图规则及完整 G21。G20 不修改 G18、FB、Contribution、玩家矿石或 Mining Pool。社交素材保持未发布草稿。

[Devlog](../devlog/G20-mining-power-foundation.md), [public verification and allowlist](../public-sync/G20-PUBLIC-SYNC-CANDIDATE.md).

## Historical milestone records / 历史阶段记录

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

## G19 accepted update / G19 已验收更新

G19 establishes canonical Black Iron Ore / 黑铁矿石 (`BLACK_IRON_ORE`) as server-owned game material through explicit reviewed legacy identity compatibility. Definition, legacy and instance IDs, ownership and quantities are preserved. Migration reads locked persisted inventory and converts metadata in place at **1:1**, creating and destroying **zero units**. No client-controlled migration quantities or issuance endpoint exists. Production aliases remain empty; all tested inventories and aliases are synthetic.

One transaction freezes the catalog and commits a version+character immutable COMPLETE receipt plus per-instance provenance. Repeat/restart returns the exact receipt; precommit failure rolls back. Legacy loads and fresh compatibility saves work, stale saves and duplicate representations fail closed. Six real process-kill windows and independently restarted verifier processes cover transaction stages and before/after commit, including lost acknowledgments. No coordinated in-flight COMMIT kill is claimed; item revision overflow relies on atomic database rollback.

Bidirectional read-only reconciliation validates receipt, inventory and provenance, detecting corrupted quantities/receipts, orphan or unmarked assets and missing commitments without silent repair. Catalog/history UPDATE/DELETE/TRUNCATE protection, stale catalog snapshots, aggregate reorder and unrelated trades are tested. Ore ownership transfer/split/consumption is deferred and generic ore trades fail atomically until a reviewed provenance lifecycle exists.

Nonempty economic snapshots with finalized reservations and Recovery Debt remain identical: FB, Contribution, Reputation, Eligible Spend, Capacity, Remaining, Reserved, Distributed, Recovery Debt and immutable mining histories are unchanged. G17 remains the sole capacity authority. Existing inventory migration is separate from future emission and reservation. **New player ore issued: 0.** No production player data was read or migrated.

Human acceptance PASS. Accepted private PR and post-merge canonical CI each passed **6/6 jobs**, Test **619/619**, Race **619/619**, fail **0**, skip **0**, Vet and native Linux/Windows/macOS builds PASS. The public subset has its own independently run checks; private counts are not public test counts.

**NOT IMPLEMENTED:** Production Block Reward; Production Block Duration; Mining Power; Mining Tool; Mining Map; Mining Tool Craft NPC; Hidden Mining Material Map; Player Mining Activity; Production Ore Distribution / Ore Reward Distribution; Production Ore Issuance; Production Emission Ratio; production alias approval and ore gameplay lifecycle.

G19 通过明确审查的旧身份兼容，将正式 Black Iron Ore／黑铁矿石（`BLACK_IRON_ORE`）定义为服务器权威游戏材料。定义、旧身份和实例 ID、所有者及数量均保留。迁移读取加锁的持久化库存，原地转换元数据，比例 **1:1**，新增及销毁数量均为 **零**。没有客户端指定迁移数量或发行端点。生产映射保持为空，所有测试库存及映射均为合成数据。

单事务冻结目录，提交版本＋角色不可变 COMPLETE 回执及逐实例来源承诺。重复／重启返回精确回执，提交前失败全部回滚。旧读取和新兼容保存可用，旧 revision 保存及重复表示拒绝。六个真实进程终止窗口和独立重启验证进程覆盖事务阶段及提交前后，包括丢失响应。不声称在正在执行的 COMMIT 内协调终止；实例 revision 溢出依靠数据库原子回滚。

双向只读对账验证回执、库存及来源，发现数量／回执破坏、孤立或无标记资产和缺失承诺，不静默修复。测试覆盖目录／历史 UPDATE／DELETE／TRUNCATE 保护、旧目录快照、聚合排序及其他物品交易。矿石所有权转移／拆分／消耗推迟，在来源生命周期获批前通用矿石交易原子拒绝。

含已完成预留和恢复债务的非空经济快照完全一致：FB、贡献、声望、合格消费、额度、Remaining、Reserved、Distributed、恢复债务及不可变挖矿历史未改变。G17 仍是唯一容量权威来源。既有库存迁移与未来发行和预留分开。**新发行玩家矿石：0。** 没有读取或迁移生产玩家数据。

人工验收 PASS。已验收私有 PR 和合并后 canonical CI 各 **6/6 作业通过**，Test **619/619**、Race **619/619**、失败 **0**、跳过 **0**、Vet 及 Linux／Windows／macOS 原生构建通过。公开子集另行独立验证，私有数量不代表公开测试数量。

**未实现：**正式区块奖励、正式区块时长、挖矿算力、挖矿工具、挖矿地图、工具制作 NPC、隐藏挖矿材料地图、玩家挖矿活动、生产矿石分发／奖励分配、生产矿石发行、生产发行比例、生产映射批准及矿石玩法生命周期。

See / 参见 [G19 Devlog](../devlog/G19-bun-black-iron-migration.md) and [export allowlist / 导出清单](../public-sync/G19-PUBLIC-SYNC-CANDIDATE.md).
