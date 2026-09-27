# Roadmap / 路线图

## English — Primary

The roadmap is milestone-driven. It does not promise dates.

### Completed foundations

- G1–G8: world, rendering, living entities, combat, AI, one Warrior skill, loot, inventory, equipment, and runtime stats.
- G9: PostgreSQL Character Aggregate persistence and restart restore.
- G10: secure item Trade Foundation.
- G11: server-authoritative internal FB Ledger Foundation with double-entry conservation, immutable history, idempotency, concurrency protection, reversal, reconciliation, and restart persistence.
- G12: atomic item + FB player Trade settlement with revision-bound offers, Gross postings, 0% fee, rollback, and restart replay.
- G13: internal non-transferable Contribution Ledger with versioned 1:1 eligible-system-spend rule, atomic FB linkage, idempotency, rollback, reconciliation, and no live producer.
- G14: atomic linked FB refund/reversal and Contribution compensation with bounded partial refunds, recovery debt, future-credit debt repayment, and review hold; no live producer.
- G15: internal Eligible System Spend orchestration over the existing ledgers; no live gameplay producer.
- G16: internal Recycle Migration Foundation with atomic item consumption, allowed material and non-transferable Reputation output, immutable canonical receipt, and no production rule or gameplay entry.
- G17: internal global Black Iron emission-capacity pool with versioned immutable entries and receipts, refund compensation, and reconciliation; no player ore or production emission ratio.
- G18: server-authoritative Mining Blocks, atomic reservation against the G17 pool, cancellation release, immutable receipts, crash recovery, and Recovery Debt / Future Offset; no reward distribution or player ore.

### Next

- Plan later economy and blockchain adapters as separately reviewed milestones; no live deposit, withdrawal, wallet, or chain feature exists.
- Continue the browser product experience using accepted server authority.
- The G14 refund/compensation hard gate is closed. A real eligible system-spend producer, production Contribution spending, and manufacturing use remain separate future work.
- G16 completed only the internal recycle foundation. Production recycle rules, Reputation caps, gameplay integration, and Mining require separate review.
- G17 completed capacity accounting; G18 completed Mining Block reservation only. Production reward and duration, power, tools, reward distribution, and player ore require separate review; G19 is now accepted; see the G19 update below.

### Future

- Full browser gameplay and broader class/skill progression.
- Bosses, parties, and team dungeons.
- Guild, guild roles, reputation, missions, wars, territory, and siege.
- Player mining using the accepted global capacity and block reservation foundations, pending future power, tool, and distribution rules.
- Wallet and domain identity.
- Ordinals ownership, metadata verification, mapping, and activation.
- Fractal / Bitcoin deposit, withdrawal, confirmation, and reorg handling.
- Marketplace and Auction beyond the accepted direct item + FB Trade settlement.
- Profiles, follows, feeds, guild/party chat, voice, video, tips, and red packets.
- Admin, security hardening, performance, mobile support, and Beta readiness.

Siege is intended to reward equipment, skills, titles, visual prestige, and other non-monetary privileges rather than directly emitting FB or Contribution, reducing permanent compounding advantages. This is a product principle, not an implemented rule.

---

## 中文 — 完整对应版本

Roadmap 按里程碑推进，不承诺日期。

### 已完成 Foundation

- G1–G8：World、Rendering、Living Entity、Combat、AI、一项 Warrior Skill、Loot、Inventory、Equipment 与 Runtime Stats。
- G9：PostgreSQL Character Aggregate Persistence 与 Restart Restore。
- G10：安全 Item Trade Foundation。
- G11：Server-authoritative Internal FB Ledger Foundation，包含 Double-entry Conservation、Immutable History、Idempotency、Concurrency Protection、Reversal、Reconciliation 与 Restart Persistence。
- G12：原子 Item + FB 玩家 Trade Settlement，包含 Revision-bound Offer、Gross Posting、0% Fee、Rollback 与 Restart Replay。
- G13：内部不可转账的 Contribution Ledger，包含版本化 1:1 合格 System Spend Rule、原子 FB 关联、Idempotency、Rollback 与 Reconciliation；尚无真实 Producer。
- G14：原子关联 FB Refund/Reversal 与 Contribution Compensation，支持有上限的部分退款、Recovery Debt、未来 Credit 先偿债与 Review Hold；尚无真实 Producer。
- G15：在现有账本上建立内部 Eligible System Spend 编排；尚无真实玩法 Producer。
- G16：内部 Recycle Migration Foundation，包括原子物品消费、允许材料与不可转让 Reputation 产出、不可变规范 Receipt；没有生产规则或玩法入口。
- G17：内部全服黑铁矿石发行额度池，包含版本化不可变流水与回执、退款补偿和对账；不发放玩家矿石，正式发行比例未定。
- G18：服务器权威 Mining Block、针对 G17 池的原子预留、取消释放、不可变回执、崩溃恢复与 Recovery Debt / Future Offset；不分配奖励或发放玩家矿石。

### 下一步

- 把后续 Economy 与 Blockchain Adapter 作为独立审核阶段规划；目前没有真实 Deposit、Withdrawal、Wallet 或 Chain 功能。
- 继续以已验收 Server Authority 扩展 Browser Product Experience。
- G14 Refund/Compensation 硬门已关闭。真实合格 System Spend Producer、生产用 Contribution 消费与 Manufacturing 用途仍须作为独立未来工作。
- G16 只完成内部回收基础。生产回收规则、Reputation 上限、玩法集成及 Mining 都需要分别审查。
- G17 完成额度记账；G18 只完成 Mining Block 预留。正式奖励与时长、Power、Tool、奖励分配与玩家矿石均需单独审核；G19 现已验收，见下方 G19 更新。

### 未来

- 完整 Browser Gameplay 与更广泛 Class / Skill Progression。
- Boss、Party 与 Team Dungeon。
- Guild、Guild Role、Reputation、Mission、War、Territory 与 Siege。
- 基于已验收的全服额度和区块预留基础规划未来玩家挖矿；Power、Tool 与分发规则仍待规划。
- Wallet 与 Domain Identity。
- Ordinals Ownership、Metadata Verification、Mapping 与 Activation。
- Fractal / Bitcoin Deposit、Withdrawal、Confirmation 与 Reorg Handling。
- 已验收的 Direct Item + FB Trade Settlement 以外的 Marketplace 与 Auction。
- Profile、Follow、Feed、Guild/Party Chat、Voice、Video、Tip 与 Red Packet。
- Admin、Security Hardening、Performance、Mobile Support 与 Beta Readiness。

Siege 计划奖励 Equipment、Skill、Title、Visual Prestige 与其他非货币 Privilege，而不是直接发行 FB 或 Contribution，以减少永久滚雪球优势。这是产品原则，不是已实现规则。

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
