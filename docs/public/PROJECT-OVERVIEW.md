# Project Overview / 项目概览

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

### What is Fractal Legend?

Fractal Legend is a browser-native Legend-style MMORPG under active development for the Fractal / Bitcoin ecosystem. Its technical direction is a thin browser client connected through a protocol boundary to a server-authoritative, cross-platform Game Server and durable PostgreSQL state.

### Why does it exist?

The project explores how classic MMORPG combat, meaningful player ownership, auditable economies, and portable digital identity can coexist without making the browser, wallet, or asset metadata authoritative over game rules.

### What do players do?

The intended loop is to explore worlds, fight monsters and bosses, grow a character, learn skills, collect and equip items, trade safely, cooperate in parties and guilds, compete in siege, mine resources, activate eligible digital assets, and build a persistent social identity.

The accepted implementation currently covers a smaller technical slice: movement, rendering, living entities, combat, basic monster AI, one Warrior skill, loot, inventory, equipment, runtime stats, Character Aggregate persistence, item + FB player Trade settlement, and internal FB and Contribution ledgers. The Contribution Ledger has no live game spend producer.

### What makes it different?

1. **Server authority:** the browser sends intent; the Game Server owns validation, RNG, damage, state, ownership, and settlement.
2. **Migration with provenance:** verified Legacy identities may be preserved while uncertain behavior is labeled as a Fractal override.
3. **Economic invariants first:** ownership, locks, revision checks, atomicity, idempotency, and restart recovery precede public trading interfaces.
4. **Digital ownership with balance:** future Ordinals can contribute provenance, identity, appearance, eligibility, and collectibility without automatically granting unlimited power.
5. **Build in Public:** accepted results, failures, boundaries, tests, and retrospectives are documented.

### Why Fractal / Bitcoin?

The long-term goal is to connect game identity and selected assets to a Bitcoin-aligned ecosystem while retaining explicit game rules and server-side verification. Fractal is intended to provide an environment for higher-throughput game interactions; Bitcoin provides the broader ownership and provenance context.

This is product direction. Mainnet integration, wallets, deposits, withdrawals, confirmations, reorg handling, and production Ordinals activation are not implemented.

### Where is development today?

G1–G10 are accepted foundations from the private canonical development line and predate this mirror. G11–G16 added internal FB and Contribution ledgers, atomic direct player Trade settlement, refund recovery, eligible system spend, and recycle migration into allowed materials and non-transferable Reputation. G17 added a global Black Iron emission-capacity pool. G18 added server-authoritative Mining Blocks, atomic reward reservation, immutable receipts, and Recovery Debt / Future Offset for upstream refunds. **Capacity and reserved rewards are not player ore**; production reward, duration, player mining, and distribution remain unimplemented. G18 technical and manual acceptance passed. These milestones are not a public game release. No public launch date has been announced.

See [Current Status](CURRENT-STATUS.md), [Development Timeline](DEVELOPMENT-TIMELINE.md), and [Roadmap](ROADMAP.md).

---

## 中文 — 完整对应版本

### Fractal Legend 是什么？

Fractal Legend / 分形传奇是一款正在为 Fractal / Bitcoin 生态开发的浏览器原生传奇风格 MMORPG。技术方向是：由轻量 Browser Client 通过 Protocol Boundary 连接 Server-authoritative、跨平台 Game Server，并使用 PostgreSQL 保存持久状态。

### 为什么要做这个项目？

项目探索经典 MMORPG 战斗、有意义的玩家所有权、可审计经济和可携带数字身份如何共同存在，同时不让 Browser、Wallet 或 Asset Metadata 越过 Game Rule 的权威边界。

### 玩家要做什么？

目标循环包括探索世界、挑战怪物与 Boss、培养角色、学习技能、收集并装备物品、安全交易、参与 Party 与 Guild 协作、竞争 Siege、开采资源、激活符合条件的 Digital Asset，以及建立持久 Social Identity。

当前已验收实现只覆盖较小的技术切片：Movement、Rendering、Living Entity、Combat、Basic Monster AI、一项 Warrior Skill、Loot、Inventory、Equipment、Runtime Stats、Character Aggregate Persistence、Item + FB 玩家 Trade Settlement，以及内部 FB 与 Contribution Ledger。Contribution Ledger 尚无真实游戏消费 Producer。

### 项目有什么不同？

1. **Server Authority：**Browser 发送 Intent；Game Server 负责 Validation、RNG、Damage、State、Ownership 和 Settlement。
2. **带 Provenance 的迁移：**可以保留已验证的 Legacy Identity；不确定的行为必须标记为 Fractal Override。
3. **先建立经济不变量：**在公开交易界面之前，先建立 Ownership、Lock、Revision Check、Atomicity、Idempotency 和 Restart Recovery。
4. **兼顾 Balance 的 Digital Ownership：**未来 Ordinals 可以提供 Provenance、Identity、Appearance、Eligibility 和 Collectibility，但不会自动获得无限制战力。
5. **Build in Public：**公开记录已验收结果、失败、边界、测试和 Retrospective。

### 为什么选择 Fractal / Bitcoin？

长期目标是在保留明确 Game Rule 与服务器验证的前提下，把 Game Identity 和部分资产接入 Bitcoin-aligned 生态。Fractal 计划为更高吞吐的游戏交互提供环境；Bitcoin 提供更广泛的 Ownership 与 Provenance 背景。

以上属于产品方向。Mainnet Integration、Wallet、Deposit、Withdrawal、Confirmation、Reorg Handling 和 Production Ordinals Activation 均未实现。

### 当前开发到哪里？

G1–G10 是 Private Canonical Development Line 上的已验收 Foundation，且早于本镜像建立。G11–G16 陆续建立内部 FB 与 Contribution Ledger、原子玩家直接交易结算、退款恢复、合格系统消费，以及将物品转为允许材料和不可转让 Reputation 的回收迁移。G17 新增全服黑铁矿石发行额度池；G18 新增服务器权威 Mining Block、原子奖励预留、不可变回执，以及处理上游退款的 Recovery Debt / Future Offset。**额度和预留奖励都不是玩家矿石**；正式奖励、时长、玩家挖矿与分发仍未实现。G18 技术与人工验收已通过。这些阶段都不代表游戏公开 Release。项目尚未公布公开上线日期。

详见[当前状态](CURRENT-STATUS.md)、[开发时间线](DEVELOPMENT-TIMELINE.md)和[路线图](ROADMAP.md)。

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
