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

# Architecture / 架构

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

### Runtime direction

Browser → player intent and rendering → Protocol / API boundary → validated commands and authoritative events → Cross-platform Fractal Game Server → repositories, transactions and revisions → PostgreSQL

The browser is untrusted. It renders authoritative state and sends intent; it does not decide movement legality, damage, RNG, death, item ownership, equipment modifiers, balances, trade completion, or persistent state.

The Game Server owns world simulation, map and movement rules, monsters, combat, skills, items, inventory, equipment, character stats, and trade rules. PostgreSQL owns durable records under transactions and optimistic revisions.

### Accepted persistence boundary

The G9 Character Aggregate stores character identity, account ownership, class, level, EXP, HP/MP, safe world position, inventory item instances, equipment relations, and learned skill state. Invalid restore positions fall back to a reviewed safe spawn. RuntimeStats are recalculated rather than trusted from a client.

### Accepted trade boundary

G10 adds a transport-independent Trade Domain. Offer revisions invalidate prior confirmations. Persistent locks prevent the same item from entering two active trades. Both Character Aggregates settle in one transaction, and duplicate Finalize calls return one durable result.

### Future modules

The accepted G17 accounting chain is **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**. G18 extends it with **G17 pool → Mining Block → atomic reward reservation → immutable receipt → finalize retaining reservation or cancel releasing it**. Upstream refund shortfalls become Recovery Debt; future emission and cancellation repay it first. PostgreSQL serializes mutations and stores canonical UTC microsecond receipts. The invariant is `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`; Distributed is zero in G18. **Emission capacity and reserved reward are not player ore.** Production reward/duration, miners, power, distribution, and player ore remain unimplemented.

Production economy, wallet, blockchain adapters, asset/Ordinals verification, player mining and allocation, guilds, social, media, administration, and analytics remain separate future ownership domains. They must not bypass the Game Server or database invariants.

### Cross-platform direction

The future Game Server supports Windows, Linux, and macOS build targets. The accepted G8–G10 line verified those targets. The Legacy Windows stack is a historical content and migration source, not a required production server platform.

### Repository topology note

The sanitized public mirror began after G10 with an independent Git history. It publishes an audited subset of project-owned domains and documentation; it does not copy or reconstruct the private canonical repository history.

---

## 中文 — 完整对应版本

### Runtime 方向

Browser → Player Intent 与 Rendering → Protocol / API Boundary → Validated Command 与 Authoritative Event → Cross-platform Fractal Game Server → Repository、Transaction 与 Revision → PostgreSQL

Browser 不受信任。它渲染 Authoritative State 并发送 Intent；它不能决定 Movement Legality、Damage、RNG、Death、Item Ownership、Equipment Modifier、Balance、Trade Completion 或 Persistent State。

Game Server 负责 World Simulation、Map 与 Movement Rule、Monster、Combat、Skill、Item、Inventory、Equipment、Character Stats 和 Trade Rule。PostgreSQL 在 Transaction 与 Optimistic Revision 下保存 Durable Record。

### 已验收 Persistence Boundary

G9 Character Aggregate 保存 Character Identity、Account Ownership、Class、Level、EXP、HP/MP、安全 World Position、Inventory ItemInstance、Equipment Relation 和 Learned Skill State。无效 Restore Position 会回退到已审核的 Safe Spawn。RuntimeStats 会重新计算，而不会信任 Client。

### 已验收 Trade Boundary

G10 增加独立于 Transport 的 Trade Domain。Offer Revision 会让旧 Confirmation 失效。Persistent Lock 阻止同一 Item 进入两笔 Active Trade。双方 Character Aggregate 在一个 Transaction 中结算，重复 Finalize Call 返回同一个持久结果。

### 未来模块

已验收的 G17 记账链为 **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**。G18 增加 **G17 池 → Mining Block → 原子奖励预留 → 不可变回执 → 保留预留的完成或释放预留的取消**。上游退款缺口形成 Recovery Debt，未来发行及取消先偿债。PostgreSQL 串行处理池变化并保存 UTC 微秒规范回执。守恒式为 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`；G18 的已分发为零。**发行额度和预留奖励不等于玩家矿石。**正式奖励／时长、矿工、算力、分配及玩家矿石仍未实现。

正式 Economy、Wallet、Blockchain Adapter、Asset/Ordinals Verification、玩家挖矿及分配、Guild、Social、Media、Administration 和 Analytics 都属于独立的未来 Ownership Domain。它们不能绕过 Game Server 或 Database Invariant。

### Cross-platform 方向

未来 Game Server 支持 Windows、Linux 与 macOS Build Target。已验收 G8–G10 开发线验证了这些目标。Legacy Windows Stack 只是历史内容与 Migration Source，不是 Production Server 的必需平台。

### Repository Topology 说明

Sanitized Public Mirror 在 G10 之后以独立 Git History 建立。它公开经过审计的项目自有 Domain 与 Documentation 子集，不复制或重建 Private Canonical Repository History。

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
