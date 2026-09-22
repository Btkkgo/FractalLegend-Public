# G17 — Black Iron Emission Pool Foundation

Status: **G17 CLOSED / PASS; public-safe foundation only**

## English — Primary

### Scope and outcome

G17 adds a server-authoritative **global future Black Iron emission capacity** ledger. It does not issue ore, create a player ore balance, or change FB or Contribution rules. An internal application takes only a persisted G15 spend identity. The PostgreSQL adapter verifies that the G15 record is completed and eligible, with matching FB debit and Contribution consequence, then calculates capacity from a versioned rule. Non-eligible G15 spends and non-G15 activities add zero. The sole enabled rule is `DEV_G17_1_TO_1`, explicitly for test/development; the production ratio remains undecided and no live producer invokes this service.

The additive PostgreSQL migration creates a singleton pool, immutable source-bound entries, and immutable receipts. Entries record positive eligible spend and negative G14 refund/reversal compensation; the pool tracks gross observed spend, refunded spend, net emission capacity, and remaining capacity. The original eligible entry is never deleted. G14 refunds continue to use immutable FB/Contribution compensation; when emission has already been applied, the negative capacity entry is written in the same transaction as the refund. If the refund happened first, initial application atomically catches up the compensation. The refund semantics therefore follow the existing G14/G15 net-spend model without changing Contribution's 1:1 rule.

The global invariant is `initial capacity + signed emission entries = reserved + distributed + remaining`. Initial capacity, reserved, and distributed are zero in G17, so net emitted capacity equals remaining capacity. Read-only reconciliation recomputes the journal, validates G15/G14 source links and rule amounts, checks receipt order and pool revision, and reports mismatch without repair. A deliberately forged journal and aggregate that agreed with each other was initially missed; a regression test reproduced the gap, then rule-derived amount validation closed it.

### Verified local evidence

- G17 real-PostgreSQL tests cover first receipt, exact same-process and reopened-store replay, snapshot/reload, non-eligible and fake sources, ten concurrent duplicates, ten distinct sources, partial refund, full reversal, refund-before-application catch-up, transaction interruption, refund-wide rollback, and reconciliation corruption detection. A mutation check confirmed the refund rollback test fails if the emission compensation hook is removed.
- Final full Go suite: **511/511 PASS, 0 failed, 0 skipped**. Final full Race suite: **511/511 PASS, 0 failed, 0 skipped**. Normal and Race used separate fresh PostgreSQL test, G9 runtime, and G11 runtime databases with synthetic G8 content.
- The complete runs included G13 (43 pass events), G14 (30), G15 (26), G16 (20), G17 (10), and eight timestamp-named replay pass events, with no failures. Go vet and local Linux/amd64, Windows/amd64, and macOS/arm64 cross-builds of all Go packages and the Game Server binary passed. These are builds, not native Windows or Linux runtime tests.
- The new migration was verified after the historical G15 and G16 versions: version 7 upgraded to 8 and then 9; the complete fresh-database suite also applied version 9. No existing schema was destructively altered.

### Stage boundary

Human review granted **G17 technical acceptance and final merge approval**. The accepted private commit passed all six PR CI jobs and the merge-triggered canonical CI. This sanitized Devlog is part of the separately reviewed Public Mirror export. A separate internal test-isolation follow-up remains open and unimplemented. No X post was published. G17 includes no Mining Block, Mining Power, Mining Tool, Mining Map, Ore Distribution, player ore inventory, Bun migration, or final production emission ratio.

## 中文 — 完整审核版

### 范围与结果

G17 新增服务器权威的**全服未来黑铁矿石发行额度**账本。不实际发放矿石，不创建玩家矿石余额，也不改变 FB 或 Contribution 规则。内部应用只接收已持久化的 G15 Spend 身份。PostgreSQL Adapter 验证 G15 记录已完成且合格，存在对应 FB 扣减与 Contribution 后果，再按照版本化规则计算额度。非合格 G15 消费及非 G15 行为增加额度为零。唯一启用的规则是测试／开发专用 `DEV_G17_1_TO_1`；正式生产比例尚未决定，也没有真实玩法 Producer 调用此服务。

增量 PostgreSQL Migration 新建单例发行池、绑定来源的不可变流水和不可变回执。流水记录正数合格消费和负数 G14 退款／冲正补偿；池记录观察到的消费总额、退款总额、净发行额度及剩余额度。原合格流水永不删除。G14 继续通过不可变 FB／Contribution 补偿处理退款；如果发行已应用，负数发行流水与退款在同一事务写入。如果退款先发生，首次应用会在单个事务内补记既有补偿。因此退款语义遵循既有 G14/G15 净消费模型，不改变 Contribution 的 1:1 规则。

全局不变量为“初始额度 + 带符号发行流水 = 预留 + 已分发 + 剩余”。G17 初始额度、预留与已分发均为零，因此净发行额度等于剩余额度。只读对账重新计算流水，验证 G15/G14 来源关联与规则数量，检查回执顺序和池 Revision；发现差异只报告，不自动修复。专项测试曾发现一种漏洞：流水与汇总被同时伪造为彼此一致时，对账最初未检出。回归测试先复现，再加入基于规则的金额验证修复。

### 已验证的本地证据

- G17 真实 PostgreSQL 测试覆盖首次回执、同进程与重开 Store 的精确重放、快照／重载、非合格与伪造来源、同一来源十路并发、十个不同来源并发、部分退款、完整冲正、先退款后应用的补记、事务中断、退款整体回滚，以及对账篡改检测。故意移除发行补偿调用的变异验证证明：退款回滚测试会按预期失败。
- 最终完整 Go 测试 **511/511 PASS、失败 0、跳过 0**；最终完整 Race 测试 **511/511 PASS、失败 0、跳过 0**。普通与 Race 分别使用全新的 PostgreSQL Test、G9 Runtime 和 G11 Runtime 数据库，以及合成 G8 内容。
- 完整测试包含 G13（43 个通过事件）、G14（30）、G15（26）、G16（20）、G17（10），以及八个名称含 Timestamp 的重放通过事件，均无失败。Go vet、Linux/amd64、Windows/amd64 和 macOS/arm64 的所有 Go Package 与 Game Server Binary 本地交叉构建通过。这些是构建验证，不代表在 Windows 或 Linux 原生运行测试。
- 新迁移在历史 G15/G16 版本之后验证：Version 7 升至 8，再升至 9；完整全新数据库测试亦应用 Version 9。没有破坏性修改既有 Schema。

### 阶段边界

人工审核已确认 **G17 技术验收与最终合并批准**。获批的私有 Commit 通过 PR 的全部六项 CI，合并后的 canonical CI 也通过。本脱敏 Devlog 属于独立审核的 Public Mirror 导出。独立的内部测试隔离后续事项仍开放、未实现。没有发布 X。G17 不包含 Mining Block、Mining Power、Mining Tool、Mining Map、矿石分配、玩家矿石库存、馒头迁移或最终生产发行比例。
