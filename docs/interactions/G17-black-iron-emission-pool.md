# G17 Black Iron Emission Pool — Curated Interaction Record

Status: **Curated G17 implementation and Stage Close record; CLOSED / PASS**

## English — Primary

### Request and decisions

The user first authorized local G17 foundation work from the verified canonical baseline while reserving commit, push, merge, public sync, and later-stage work for separate approval. Live ore issuance, Mining Block/Power/Tool/Map, Bun migration, FB or Contribution rule changes, and a final production emission ratio remained outside G17. A separate private test-isolation follow-up stayed open.

1. Reverify canonical and create a local G17 feature branch from it. The private canonical worktree stays clean. A private tracking issue records the milestone; no public issue is created.
2. Audit G14 and G15 before selecting refund behavior. G15 stores an immutable eligible spend linked to FB and Contribution records. G14 records partial refunds and reversals as immutable compensations; the original spend is retained, and net spend/consequence is reduced. Therefore G17 uses a separate negative immutable emission entry, not deletion of the original entry. It joins the G14 transaction if emission already exists; later initial application catches up earlier refunds.
3. Keep eligibility authority in G15. G17 accepts only a persisted G15 spend identity, verifies its ledger/contribution links, and calculates the amount on the server. The `DEV_G17_1_TO_1` version is a development fixture only. Production emission stays disabled pending Balance Phase.
4. Preserve conservation and exact replay. PostgreSQL uniquely binds each spend and compensation, stores UTC microsecond timestamps, returns persisted receipts, and serializes on the original FB transaction and global pool. Reconciliation checks both aggregate sums and independently derived rule amounts. A failing test exposed the latter missing check before it was added.
5. Keep test runs isolated from development runtime data without implementing the separate test-isolation follow-up. Normal and Race used separate new test/G9/G11 databases and synthetic G8 content. Both complete suites passed 511/511 with zero skips. Vet and three-platform cross-builds passed. At this earlier local review point, GitHub CI had not run because the branch was uncommitted.

### Review boundary

Only the G17 emission domain, additive persistence migration, G14 transaction hook, migration regression adjustment, focused tests, and bilingual records changed. No ore item, player balance, mining system, final ratio, or X post was produced. Human technical acceptance passed; the PR and post-merge canonical CI each passed all six jobs. The sanitized Public Mirror export followed a separate safety review. G18 was not started.

This is a curated engineering decision record, not a verbatim private conversation.

## 中文 — 完整审核版

### 请求与决策

用户先授权从已验证的 canonical 基线开展 G17 本地基础开发，Commit、Push、Merge 和公开同步等待分别批准。真实发矿、Mining Block／Power／Tool／Map、馒头迁移、FB 或 Contribution 规则修改及最终生产发行比例均不属于 G17。独立的私有测试隔离后续事项保持开放。

1. 重新核对 canonical 后，从该基线建立本地 G17 Feature Branch；Private Canonical 工作区保持干净。通过私有跟踪事项记录本里程碑，不创建公开 Issue。
2. 决定退款行为前先审计 G14 和 G15。G15 保存不可变的合格消费，并关联 FB 与 Contribution 记录。G14 将部分退款和冲正记为不可变补偿；原消费保留，净消费和后果相应减少。因此 G17 使用独立的负数不可变发行流水，不删除原流水。发行已存在时补偿与 G14 共用事务；退款先发生时，后续首次应用补记既有退款。
3. 合格资格仍由 G15 掌握。G17 只接收持久化的 G15 Spend 身份，验证 Ledger／Contribution 关联，金额由服务器计算。`DEV_G17_1_TO_1` 仅为开发 Fixture。正式发行等待 Balance Phase，当前未启用。
4. 保持守恒与精确重放。PostgreSQL 对每笔 Spend 和 Compensation 建立唯一绑定，使用 UTC 微秒时间并返回持久化回执；按原 FB Transaction 与全局发行池加锁串行化。对账同时检查汇总合计及独立计算的规则金额。一个先失败的测试揭示原本缺失的规则金额验证，随后予以修复。
5. 测试与开发 Runtime 数据隔离，但不实施独立的测试隔离后续事项。普通与 Race 分别使用新建的 Test／G9／G11 数据库和合成 G8 内容。两轮完整测试各 511/511 通过，跳过 0；Vet 和三平台交叉构建通过。在较早的本地审核节点，分支尚未 Commit，GitHub CI 尚未执行。

### 审核边界

改动仅涉及 G17 发行 Domain、增量持久化 Migration、G14 事务挂接、迁移回归测试调整、专项测试及双语记录。没有创建矿石物品、玩家余额、Mining 系统、正式比例或 X 帖文。人工技术验收通过；PR 与合并后的 canonical CI 均有六项作业通过。脱敏 Public Mirror 导出另行经过安全审查；G18 没有开始。

本文件是整理后的工程决策记录，不是私人对话的逐字稿。
