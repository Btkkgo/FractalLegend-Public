# G19 — Bun → Black Iron Ore Migration Foundation

## English — Primary

G19 establishes canonical Black Iron Ore / 黑铁矿石 (`BLACK_IRON_ORE`) as server-owned game material through explicit reviewed legacy identity compatibility. Definition, legacy and instance IDs, ownership and quantities are preserved. Migration reads locked persisted inventory and converts metadata in place at **1:1**, creating and destroying **zero units**. No client-controlled migration quantities or issuance endpoint exists. Production aliases remain empty; all tested inventories and aliases are synthetic.

One transaction freezes the catalog and commits a version+character immutable COMPLETE receipt plus per-instance provenance. Repeat/restart returns the exact receipt; precommit failure rolls back. Legacy loads and fresh compatibility saves work, stale saves and duplicate representations fail closed. Six real process-kill windows and independently restarted verifier processes cover transaction stages and before/after commit, including lost acknowledgments. No coordinated in-flight COMMIT kill is claimed; item revision overflow relies on atomic database rollback.

Bidirectional read-only reconciliation validates receipt, inventory and provenance, detecting corrupted quantities/receipts, orphan or unmarked assets and missing commitments without silent repair. Catalog/history UPDATE/DELETE/TRUNCATE protection, stale catalog snapshots, aggregate reorder and unrelated trades are tested. Ore ownership transfer/split/consumption is deferred and generic ore trades fail atomically until a reviewed provenance lifecycle exists.

Nonempty economic snapshots with finalized reservations and Recovery Debt remain identical: FB, Contribution, Reputation, Eligible Spend, Capacity, Remaining, Reserved, Distributed, Recovery Debt and immutable mining histories are unchanged. G17 remains the sole capacity authority. Existing inventory migration is separate from future emission and reservation. **New player ore issued: 0.** No production player data was read or migrated.

Human acceptance PASS. Accepted private PR and post-merge canonical CI each passed **6/6 jobs**, Test **619/619**, Race **619/619**, fail **0**, skip **0**, Vet and native Linux/Windows/macOS builds PASS. The public subset has its own independently run checks; private counts are not public test counts.

**NOT IMPLEMENTED:** Production Block Reward; Production Block Duration; Mining Power; Mining Tool; Mining Map; Mining Tool Craft NPC; Hidden Mining Material Map; Player Mining Activity; Production Ore Distribution / Ore Reward Distribution; Production Ore Issuance; Production Emission Ratio; production alias approval and ore gameplay lifecycle.

## 中文 — 完整审核版

G19 通过明确审查的旧身份兼容，将正式 Black Iron Ore／黑铁矿石（`BLACK_IRON_ORE`）定义为服务器权威游戏材料。定义、旧身份和实例 ID、所有者及数量均保留。迁移读取加锁的持久化库存，原地转换元数据，比例 **1:1**，新增及销毁数量均为 **零**。没有客户端指定迁移数量或发行端点。生产映射保持为空，所有测试库存及映射均为合成数据。

单事务冻结目录，提交版本＋角色不可变 COMPLETE 回执及逐实例来源承诺。重复／重启返回精确回执，提交前失败全部回滚。旧读取和新兼容保存可用，旧 revision 保存及重复表示拒绝。六个真实进程终止窗口和独立重启验证进程覆盖事务阶段及提交前后，包括丢失响应。不声称在正在执行的 COMMIT 内协调终止；实例 revision 溢出依靠数据库原子回滚。

双向只读对账验证回执、库存及来源，发现数量／回执破坏、孤立或无标记资产和缺失承诺，不静默修复。测试覆盖目录／历史 UPDATE／DELETE／TRUNCATE 保护、旧目录快照、聚合排序及其他物品交易。矿石所有权转移／拆分／消耗推迟，在来源生命周期获批前通用矿石交易原子拒绝。

含已完成预留和恢复债务的非空经济快照完全一致：FB、贡献、声望、合格消费、额度、Remaining、Reserved、Distributed、恢复债务及不可变挖矿历史未改变。G17 仍是唯一容量权威来源。既有库存迁移与未来发行和预留分开。**新发行玩家矿石：0。** 没有读取或迁移生产玩家数据。

人工验收 PASS。已验收私有 PR 和合并后 canonical CI 各 **6/6 作业通过**，Test **619/619**、Race **619/619**、失败 **0**、跳过 **0**、Vet 及 Linux／Windows／macOS 原生构建通过。公开子集另行独立验证，私有数量不代表公开测试数量。

**未实现：**正式区块奖励、正式区块时长、挖矿算力、挖矿工具、挖矿地图、工具制作 NPC、隐藏挖矿材料地图、玩家挖矿活动、生产矿石分发／奖励分配、生产矿石发行、生产发行比例、生产映射批准及矿石玩法生命周期。


## Public verification / 公开验证

Fresh sequential PostgreSQL test databases: Go Test **383/383 PASS**, Go Race **383/383 PASS**, failed **0**, skipped **0**; Go Vet **PASS**. All **173** tracked/nonignored files passed Secret, Personal Information, Legacy Source, Restricted Third-party Asset and local-link checks. Ten code paths are byte-identical to accepted canonical; nine approved baseline PNG files retain exact SHA-256 hashes. No new binaries, archives, oversized files, dependency or license changes. Pattern scans are complemented by explicit allowlist, source/diff and provenance review. Publication uses the account noreply identity and independent public Git history.

顺序使用各自全新 PostgreSQL 测试库：Go Test **383/383 PASS**、Go Race **383/383 PASS**、失败 **0**、跳过 **0**，Go Vet **PASS**。全部 **173** 个受追踪／非忽略文件通过秘密、个人信息、旧源码、受限素材及本地链接检查。十个代码路径与已验收 canonical 逐字节相同，九个既有批准 PNG 保持精确 SHA-256 哈希。没有新二进制、归档、超大文件、依赖或许可变化；模式扫描配合明确允许清单、源码／Diff 和来源审查。发布使用账号 noreply 身份，保留独立公开 Git 历史。

## Final Close references / 最终收尾引用

Accepted private implementation merge (plain archive reference, no private Git object): `ee0f1d1ddf42cb61d202cc734fc0e34db68d253b`. Canonical post-merge CI run `36302777395`: 6/6 PASS, Test/Race 619 each, failed/skipped 0, Vet and three native builds PASS. Public implementation export: `6f22c793c4d176f9e0886b8ce88adf6dbc9e3c59`. This later documentation-only commit records that verified export without changing tested code. Final publication HEAD is independently verified remotely and recorded in the private closure report. Human acceptance PASS; X not published; G20 not started.

已验收私有实现合并（仅纯文本归档引用，不复制私有 Git 对象）：`ee0f1d1ddf42cb61d202cc734fc0e34db68d253b`。canonical 合并后 CI `36302777395`：6/6 PASS，Test／Race 各 619，失败／跳过 0，Vet 和三项原生构建通过。公开实现导出：`6f22c793c4d176f9e0886b8ce88adf6dbc9e3c59`。后续仅文档提交记录这一已验证导出，不改变测试代码；最终公开 HEAD 另行核实远端并记录于私有收尾报告。人工验收 PASS，未发布 X，未开始 G20。
