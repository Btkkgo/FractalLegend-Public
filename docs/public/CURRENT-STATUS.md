# Current Status / 当前状态

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
