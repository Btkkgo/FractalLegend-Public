# Current Status / 当前状态

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

Updated for the sanitized public mirror on 2026-09-22. “Foundation Complete” means an accepted technical milestone, not a released product.

| Area | Status | Evidence / boundary |
|---|---|---|
| G1 World | **FOUNDATION COMPLETE** | Server-authoritative movement and collision slice |
| G2 Rendering | **FOUNDATION COMPLETE** | Browser-ready 2D atlas pipeline using Fractal override visuals |
| G3 Living World | **FOUNDATION COMPLETE** | NPC and six monsters synchronized; NPC interaction absent |
| G4 Combat | **FOUNDATION COMPLETE** | Target, normal attack, server damage, and death |
| G5 Monster AI | **FOUNDATION COMPLETE** | Deterministic state machine, chase, attack, and return |
| G6 Skills | **FOUNDATION COMPLETE** | Warrior 烈火剑法 only |
| G7 Loot / Inventory | **FOUNDATION COMPLETE** | Death drop, pickup, ownership, and 20-slot inventory |
| G8 Equipment / Stats | **FOUNDATION COMPLETE** | Equip/unequip, RuntimeStats, reconnect, and drift/concurrency tests |
| G9 Persistence | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate and full Game Server restart restore |
| G10 Trade | **FOUNDATION COMPLETE** | Item-trade domain, extended by G12; UI and markets absent |
| G11 FB Ledger | **FOUNDATION COMPLETE** | Internal server-authoritative Ledger; no wallet or blockchain adapter |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | Atomic PostgreSQL settlement with 0% fee; no Marketplace or Trade UI |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | Internal non-transferable ledger; 1:1 eligible system spend; no live producer |
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | Atomic FB + Contribution recovery, bounded partial refunds, debt and hold; no live producer |
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | Internal orchestration; no live gameplay producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | Internal material + Reputation settlement with immutable receipt; production rules and gameplay entry disabled |
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | Versioned immutable global capacity; no player ore or production ratio |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | Atomic block reward reservation, recovery debt, immutable replay; no distribution |
| G19 Black Iron Ore Migration | **FOUNDATION COMPLETE** | Reviewed identity compatibility, 1:1 inventory, immutable provenance; no issuance |
| G18.1 Stable Block Identity | **FOUNDATION COMPLETE** | Immutable BlockInstanceID |
| G20 Mining Power | **FOUNDATION COMPLETE** | SERVER authority, synthetic TEST profiles; no rewards |
| G21-P0 Settlement Prerequisites | **FOUNDATION COMPLETE** | Sealed input and immutable bindings; TEST-only accounting/issuance prerequisites, no real player ore or full settlement |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted slices do not form a public release |
| Multi-class gameplay | **PLANNED** | Warrior is the only confirmed Web runtime class |
| Boss / Party / Team Dungeon | **PLANNED** | No accepted implementation |
| Guild / Siege | **PLANNED** | No accepted implementation |
| Wallet / blockchain | **PLANNED** | No live deposit, withdrawal, or chain adapter |
| Ordinals activation | **PLANNED** | No production verification or activation |
| Player mining and reward distribution | **PLANNED** | G18 reserves block capacity only; no miners, power, tools, maps, or player ore |
| Social / voice / video / feed | **PLANNED / RESEARCH** | No accepted implementation |

### Repository topology

The private canonical repository preserves complete internal development and acceptance history. This sanitized public mirror began after G10 with a new Git history and an audited subset of project-owned source, tests, and documentation. The public history is not the original G1–G10 chronology.

### Latest closed milestone

G18 technical and manual acceptance are PASS. The accepted private PR and post-merge canonical CI each passed 6/6 jobs: Go Test 556/556, Go Race 556/556, zero skipped, Vet, Linux/Windows/macOS builds, and real independent-process Crash A–E. G18 reserves block rewards against the G17 authoritative pool. Upstream refund shortfalls create Recovery Debt; future emission and cancellation repay it first. Reconciliation checks `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`. **Distributed is zero; capacity and reservation are not player ore.** The development block rule is not a production reward or duration. No miners, mining power, tools, maps, reward allocation, player ore, or Bun migration exist.

### Release status

There is no public launch date, production deployment, mainnet economy, public Trade UI, or complete game release.

---

## 中文 — 完整对应版本

本页按 2026-09-22 的 Sanitized Public Mirror 状态更新。“Foundation Complete”表示技术里程碑已验收，不表示产品已经发布。

| 领域 | 状态 | 证据 / 边界 |
|---|---|---|
| G1 World | **FOUNDATION COMPLETE** | Server-authoritative Movement 与 Collision Slice |
| G2 Rendering | **FOUNDATION COMPLETE** | 使用 Fractal Override Visual 的 Browser-ready 2D Atlas Pipeline |
| G3 Living World | **FOUNDATION COMPLETE** | 同步 NPC 与六只怪物；不含 NPC Interaction |
| G4 Combat | **FOUNDATION COMPLETE** | Target、Normal Attack、Server Damage 与 Death |
| G5 Monster AI | **FOUNDATION COMPLETE** | 确定性 State Machine、Chase、Attack 与 Return |
| G6 Skills | **FOUNDATION COMPLETE** | 仅 Warrior 烈火剑法 |
| G7 Loot / Inventory | **FOUNDATION COMPLETE** | Death Drop、Pickup、Ownership 与 20-slot Inventory |
| G8 Equipment / Stats | **FOUNDATION COMPLETE** | Equip/Unequip、RuntimeStats、Reconnect 与 Drift/Concurrency Test |
| G9 Persistence | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate 与完整 Game Server Restart Restore |
| G10 Trade | **FOUNDATION COMPLETE** | Item Trade Domain，由 G12 扩展；不含 UI 与 Market |
| G11 FB Ledger | **FOUNDATION COMPLETE** | 内部 Server-authoritative Ledger；不含 Wallet 或 Blockchain Adapter |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | PostgreSQL 原子结算、0% 手续费；不含 Marketplace 或 Trade UI |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | 内部不可转账账本；合格 System Spend 1:1；尚无真实 Producer |
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | 原子 FB + Contribution 恢复、部分退款上限、债务和 Hold；尚无真实 Producer |
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | 内部消费编排；尚无真实玩法 Producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | 内部材料与 Reputation 结算及不可变 Receipt；生产规则和玩法入口禁用 |
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | 版本化、不可变的全服额度；不发放玩家矿石，正式比例未定 |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | 原子区块奖励预留、恢复债务及不可变重放；没有分发 |
| G19 黑铁矿石迁移 | **FOUNDATION COMPLETE** | 明确身份兼容、1:1 库存及不可变来源；不发行矿石 |
| G18.1 稳定区块身份 | **FOUNDATION COMPLETE** | 不可变 BlockInstanceID |
| G20 挖矿算力 | **FOUNDATION COMPLETE** | 服务器权威、合成 TEST 输入，不结算奖励 |
| G21-P0 结算前置契约 | **FOUNDATION COMPLETE** | 封存输入与不可变绑定；仅 TEST 的记账／发行前置，不发真实玩家矿石、不实现完整结算 |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Slice 尚未组成公开 Release |
| Multi-class Gameplay | **PLANNED** | Warrior 是唯一确认的 Web Runtime Class |
| Boss / Party / Team Dungeon | **PLANNED** | 没有已验收实现 |
| Guild / Siege | **PLANNED** | 没有已验收实现 |
| Wallet / Blockchain | **PLANNED** | 没有 Live Deposit、Withdrawal 或 Chain Adapter |
| Ordinals Activation | **PLANNED** | 没有 Production Verification 或 Activation |
| 玩家挖矿与奖励分配 | **PLANNED** | G18 只预留区块容量；没有矿工、算力、工具、地图或玩家矿石 |
| Social / Voice / Video / Feed | **PLANNED / RESEARCH** | 没有已验收实现 |

### Repository Topology

Private Canonical Repository 保留完整内部开发与验收历史。本 Sanitized Public Mirror 在 G10 之后以全新 Git History 建立，仅包含经过审计的项目自有 Source、Test 与 Documentation 子集。Public History 不是原始 G1–G10 Chronology。

### 最新关闭里程碑

G18 技术与人工验收均为 PASS。获批的私有 PR 和合并后的 canonical CI 各有 6/6 作业通过：Go Test 556/556、Go Race 556/556、跳过 0、Vet、Linux/Windows/macOS 构建和真实独立进程 Crash A–E。G18 从 G17 权威池预留区块奖励；上游退款的缺口形成恢复债务，未来发行和取消释放先偿债。对账检查 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`。**已分发为零；额度和预留均不是玩家矿石。**开发区块规则不代表正式奖励或时长。没有矿工、挖矿算力、工具、地图、奖励分配、玩家矿石或馒头迁移。

### Release 状态

项目没有公开上线日期、Production Deployment、Mainnet Economy、公开 Trade UI 或完整 Game Release。

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
