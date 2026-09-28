# G21 — Current accepted status / 当前已验收状态

## English — Primary

**G21 STAGE CLOSED — PASS.** The owner accepted the completed settlement foundation. Accepted settlement merge: `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`; canonical closeout revision: `37452efe5c9d529eec04c7629c483de4156dd6fb`. Public Git history remains independent.

The complete accepted repository passed **121/121 required G21 checks**, plus **9 P0_COVERED** (130 total); normal and race each **1296 PASS**, real PostgreSQL integration PASS, crash recovery **9/9**, CI **18/18**, and native Windows/Linux/macOS deterministic equality. Receipt digest: `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`.

This public subset was independently checked against fresh, isolated PostgreSQL databases: normal **1060/1060 PASS**, race **1060/1060 PASS**, zero failures or test skips; Go vet and builds for macOS arm64 / Linux amd64 / Windows amd64 PASS. Cross-compilation is a build check, not native execution; native results above belong to the accepted complete repository.

Production settlement stays **disabled**. G18 **R=10** authority and economic parameters are unchanged. G22 has not started; no release or X publication.

## 中文 — 完整审核版

**G21 STAGE CLOSED — PASS。** 用户已验收完整结算基础。已验收实现合并版本为 `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`，canonical 收尾版本为 `37452efe5c9d529eec04c7629c483de4156dd6fb`；公开仓库继续保持独立 Git 历史。

完整已验收仓库 **121/121 必需 G21 项通过**，另有 **9 P0_COVERED**（共 130 项）；普通与竞态各 **1296 PASS**，真实 PostgreSQL 集成通过、崩溃恢复 **9/9**、CI **18/18**，Windows／Linux／macOS 真实原生执行的确定性一致性通过。回执摘要为 `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`。

本公开子集使用新建隔离 PostgreSQL 库独立验证：普通 **1060/1060 PASS**、竞态 **1060/1060 PASS**，零失败、零测试跳过；Go vet 及 macOS arm64／Linux amd64／Windows amd64 构建通过。交叉编译仅作为构建检查，上述真实原生结果属于完整已验收仓库。

生产结算保持**禁用**；G18 **R=10** 权威与经济参数不变。G22 未开始，未发布 Release 或 X。

---

## Earlier milestone records / 早期里程碑记录

The following entries preserve earlier stage scope and test counts; the G21 status above is current. / 以下保留早期阶段范围与测试数；当前以 G21 状态为准。

# Public Mirror Development History / 公开镜像开发历史

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

The sanitized public mirror began **after G10** with a new Git history. G1–G10 were developed and accepted in the private canonical development repository before this mirror was created.

The table below is a public-safe retrospective. It preserves accepted results and recorded test counts without pretending that this repository's commit history represents the original chronology.

| Gate | Name | Result | Recorded tests | Public-safe summary |
|---|---|---|---|---|
| G1 | First Playable World | PASS | Acceptance and three-platform standalone PASS; no separate count recorded | Server-authoritative movement, position, bounds, speed, and collision validation |
| G2 | Rendering Pipeline | PASS | Acceptance PASS; no separate count recorded | Browser-ready 2D rendering and deterministic Fractal override atlas pipeline |
| G3 | Living World | PASS | Acceptance and three-platform standalone PASS; no separate count recorded | Authoritative NPC and six-monster world snapshot and browser synchronization |
| G4 | Combat Sandbox | PASS | Acceptance and cross-platform builds PASS; no separate count recorded | Server-owned target selection, normal attack, damage, HP, and death |
| G5 | Basic Monster AI | PASS | Acceptance PASS; no separate count recorded | Deterministic aggro, chase, attack, and return state machine |
| G6 | Skills and Character Combat | PASS | Acceptance PASS; no separate count recorded | One Warrior active-damage skill with server-owned MP, cooldown, damage, and retry protection |
| G7 | Drop, Pickup, and Inventory | PASS | G1–G6 regression, Race, standalone, and cross-platform PASS; no separate count recorded | Death drop, atomic pickup, ownership, reconnect state, and 20-slot inventory |
| G8 | Equipment Runtime and Character Stats | PASS | Combined 456/456; Node 222; Go 234; Race and three-platform builds PASS | Item-instance equipment, runtime stat modifiers, combat integration, reconnect, and drift checks |
| G9 | Persistence Foundation | PASS | Persistence 25/25; Node 222/222; Go 261/261; Combined 483/483; Race, PostgreSQL integration, restart, and three-platform builds PASS | Transactional PostgreSQL Character Aggregate with optimistic revision and restart restore |
| G10 | Trade Foundation | CLOSED / PASS | Historical Node 224/224; Trade 29/29; Go 290/290; Windows fixtures 2/2; cross-platform PASS | Item-only, server-authoritative trade with persistent locks and atomic settlement |

Private canonical archive references retained for internal verification:

- G9 documentation acceptance: `06aecadb0026cf538ef589d2ddfdaf9aea5aab67`
- G10 acceptance: `b47f7db006bf121f194bfb82808c3864c1c5f90e`
- Public overview prepared before mirror creation: `a4dae045f695904be8f564be14d42ccd33dc2bdf`
- Private overview merge: `97215c7e87c317fc28b94e1ecaf73d334748f291`

These are plain references to the private canonical archive. They are not public links and are not commits in this mirror.

After mirror creation, G11 FB Ledger, G12 atomic item + FB Trade settlement, G13 Contribution Ledger Foundation, G14 refund/reversal recovery, G15 eligible system spend, G16 recycle migration, G17 Black Iron emission capacity, and G18 Mining Block reservation were accepted for separate sanitized public export events. G11 recorded 44/44 targeted checks, 11/11 real PostgreSQL checks, and 334/334 full Go regression. G12 recorded 45/45 targeted checks, 12/12 real PostgreSQL checks, and 379/379 full Go regression. G13 recorded 41/41 targeted checks, 19/19 real PostgreSQL checks, 200/200 property iterations, 100-way contention, 420/420 full Go regression, Race, and three-platform builds. G14 added atomic Contribution refund/reversal recovery, with 27/27 focused checks, 21/21 real PostgreSQL checks, two 200-iteration property sequences, and 447/447 full Go and Race checks. G15 introduced internal eligible system spend without a live producer. G16 added internal item recycling into allowed materials and Reputation: the final private normal and Race suites each passed 492/492 with zero skips, and GitHub Actions passed 6/6 jobs after a PostgreSQL timestamp replay failure was found and repaired. Its G2 Node check was 13/14 because a restricted Asset Atlas source remains unavailable. G17 added versioned immutable global emission-capacity entries and receipts, refund compensation, conservation, and reconciliation without player ore or a production ratio. The accepted private PR and post-merge canonical CI each passed six jobs, with Go Test and Race at 511/511 and zero skipped. See the bilingual [Devlogs](../devlog/) and ADRs for scope and limits. These public commits are sanitized export events, not the original private development chronology.

G18 then added server-authoritative blocks and atomic reward reservation, discovered a refund-versus-finalized-reservation shortfall, and adopted Recovery Debt / Future Offset. Independent-process Crash A–E and replay passed. The private PR and post-merge canonical CI each passed six jobs, Go Test and Race 556/556 with zero skipped. Production block reward and duration remain undecided, and no player ore or distribution was created.

---

## 中文 — 完整对应版本

Sanitized Public Mirror 在 **G10 之后**以全新 Git History 建立。G1–G10 在本镜像创建前，已经在 Private Canonical Development Repository 中完成开发与验收。

下表是 Public-safe Retrospective。它保留已验收结果与记录的 Test Count，但不会把本仓库 Commit History 伪装成原始开发时间线。

| Gate | 名称 | 结果 | 已记录测试 | Public-safe 摘要 |
|---|---|---|---|---|
| G1 | First Playable World | PASS | Acceptance 与三平台 Standalone PASS；未单独记录计数 | Server-authoritative Movement、Position、Bounds、Speed 与 Collision Validation |
| G2 | Rendering Pipeline | PASS | Acceptance PASS；未单独记录计数 | Browser-ready 2D Rendering 与确定性的 Fractal Override Atlas Pipeline |
| G3 | Living World | PASS | Acceptance 与三平台 Standalone PASS；未单独记录计数 | 权威 NPC、六只怪物 World Snapshot 与 Browser Synchronization |
| G4 | Combat Sandbox | PASS | Acceptance 与 Cross-platform Build PASS；未单独记录计数 | Server-owned Target Selection、Normal Attack、Damage、HP 与 Death |
| G5 | Basic Monster AI | PASS | Acceptance PASS；未单独记录计数 | 确定性的 Aggro、Chase、Attack 与 Return State Machine |
| G6 | Skills and Character Combat | PASS | Acceptance PASS；未单独记录计数 | 一项 Warrior Active-damage Skill，MP、Cooldown、Damage 与 Retry Protection 由服务器拥有 |
| G7 | Drop, Pickup, and Inventory | PASS | G1–G6 Regression、Race、Standalone 与 Cross-platform PASS；未单独记录计数 | Death Drop、Atomic Pickup、Ownership、Reconnect State 与 20-slot Inventory |
| G8 | Equipment Runtime and Character Stats | PASS | Combined 456/456；Node 222；Go 234；Race 与三平台 Build PASS | Item-instance Equipment、Runtime Stat Modifier、Combat Integration、Reconnect 与 Drift Check |
| G9 | Persistence Foundation | PASS | Persistence 25/25；Node 222/222；Go 261/261；Combined 483/483；Race、PostgreSQL Integration、Restart 与三平台 Build PASS | Transactional PostgreSQL Character Aggregate、Optimistic Revision 与 Restart Restore |
| G10 | Trade Foundation | CLOSED / PASS | Historical Node 224/224；Trade 29/29；Go 290/290；Windows Fixture 2/2；Cross-platform PASS | 仅 Item、Server-authoritative Trade，包含 Persistent Lock 与 Atomic Settlement |

为内部验证保留的 Private Canonical Archive Reference：

- G9 Documentation Acceptance：`06aecadb0026cf538ef589d2ddfdaf9aea5aab67`
- G10 Acceptance：`b47f7db006bf121f194bfb82808c3864c1c5f90e`
- Public Mirror 建立前准备的 Public Overview：`a4dae045f695904be8f564be14d42ccd33dc2bdf`
- Private Overview Merge：`97215c7e87c317fc28b94e1ecaf73d334748f291`

这些只是 Private Canonical Archive 的纯文本 Reference，不是 Public Link，也不是本镜像中的 Commit。

镜像建立后，G11 FB Ledger、G12 原子 Item + FB Trade Settlement、G13 Contribution Ledger Foundation、G14 Refund/Reversal Recovery、G15 Eligible System Spend、G16 Recycle Migration、G17 黑铁矿石发行额度与 G18 Mining Block 预留分别通过验收，并以独立的脱敏公开事件导出。G11 记录专项检查 44/44、真实 PostgreSQL 检查 11/11、Go 完整回归 334/334。G12 记录专项检查 45/45、真实 PostgreSQL 检查 12/12、Go 完整回归 379/379。G13 记录专项检查 41/41、真实 PostgreSQL 检查 19/19、Property Iteration 200/200、100 并发竞争、Go 完整回归 420/420、Race 与三平台 Build。G14 新增原子 Contribution Refund/Reversal Recovery，专项检查 27/27、真实 PostgreSQL 检查 21/21、两组各 200 轮 Property，以及各 447/447 的 Go 全量和 Race。G15 建立尚无真实 Producer 的内部合格系统消费。G16 新增将物品内部回收为允许材料与 Reputation 的基础：Private 最终普通测试与 Race 各为 492/492、0 跳过，PostgreSQL 时间重放故障被发现并修复后，GitHub Actions 6/6 Jobs 通过。其 G2 Node 因受限 Asset Atlas 来源不可用仍为 13/14。G17 新增版本化、不可变的全服发行额度流水和回执、退款补偿、守恒与对账，不发放玩家矿石，也未确定正式比例。获批的私有 PR 与合并后的 canonical CI 各有六项作业通过，Go Test 和 Race 各为 511/511、跳过 0。具体范围与限制详见双语 [Devlog](../devlog/) 和 ADR。这些 Public Commit 是脱敏导出事件，不是原始 Private Development Chronology。

随后 G18 新增服务器权威区块与原子奖励预留，发现上游退款与已完成区块预留间的缺口，并采用 Recovery Debt / Future Offset。真实独立进程 Crash A–E 与重放通过。私有 PR 和合并后 canonical CI 各六项作业通过，Go Test 与 Race 各 556/556、跳过 0。正式区块奖励与时长仍未决定，没有发放玩家矿石或进行分配。

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
