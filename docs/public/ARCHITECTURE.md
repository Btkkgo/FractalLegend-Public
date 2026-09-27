# Architecture / 架构

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
