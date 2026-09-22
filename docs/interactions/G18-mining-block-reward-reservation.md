# G18 — Curated Interaction Record

Status: **G18 Stage Close complete; sanitized public summary**

## English — Primary

### Request and boundaries

The user authorized a server-only Mining Block and reward-reservation foundation against the existing G17 capacity pool, with transactional PostgreSQL persistence, unique monotonic heights, immutable entries and receipts, exact idempotency, crash recovery, reconciliation, and a complete English-primary/Chinese-review record. The user prohibited player mining and ore distribution, production economic constants, G19, and work on the separate database-isolation issue. Stage Close was authorized only after technical acceptance and CI.

### Decisions and correction

1. G17 remains the single capacity authority. Create, finalize, and cancel use its row lock and a single transaction; finalize retains a reservation, while cancel atomically releases it. No intermediate status or player distribution is persisted.
2. The initial pool model could not safely compensate a refund after block capacity had been reserved. The user approved Recovery Debt / Future Offset. A refund uses Remaining first, then records debt; future emission and cancellation release repay debt before restoring Remaining. Reservations survive even for finalized blocks. New reservations fail closed while debt exists.
3. The invariant became `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`, with Distributed at zero in G18. Migration 0010 adds the necessary tables and constraints while preserving earlier ledger rows.
4. Production mode has no approved block rule and fails closed. The development-only rule fixes 10 units and one minute for tests; neither is a production reward or duration.

### Evidence and completion

An old migration-version assertion was corrected. A later test run encountered stale G9 runtime state from a reused database; the final ordinary and Race suites used independent fresh databases. Both passed 556/556 test events, zero failed and zero skipped. Independent child-process Crash A–E and commit-before-response replay were checked with new verifier processes against the same PostgreSQL database. Concurrency, refund collision, conservation, and reconciliation tests passed. The approved private merge and post-merge CI completed; the PR and canonical GitHub Actions each passed all six jobs, including 556/556 Test and Race, zero skipped, Vet, and three platform builds.

The private milestone was closed after its complete bilingual update. The public mirror copies only project-owned, allowlisted contents, retaining its independent history. Architecture and CI evidence are saved for future publication; there was no X post. The database-isolation issue stays open, and G19 has not started.

## 中文 — 完整审核版

### 请求与边界

用户授权在现有 G17 容量池上建立仅供服务器使用的 Mining Block 和奖励预留基础，要求事务性 PostgreSQL 持久化、唯一单调高度、不可变流水与回执、精确幂等、崩溃恢复、对账，以及英文主版和中文完整审核记录。用户禁止玩家挖矿及发矿、正式经济常数、G19，以及处理单独的数据库隔离事项。只有技术验收和 CI 通过后才授权 Stage Close。

### 决策与修正

1. G17 保持唯一容量权威。Create、Finalize、Cancel 使用池行锁与单个事务；Finalize 保留预留，Cancel 原子释放。没有持久化中间状态或玩家分发。
2. 初版池模型在区块容量已预留后无法安全补偿上游退款。用户批准 Recovery Debt / Future Offset：退款先用 Remaining，缺口记债；未来发行与取消释放先还债，再恢复 Remaining。已完成区块的预留也保留。债务未清时拒绝新预留。
3. 守恒式变为 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`，G18 中已分发为零。迁移 0010 增加必要表和约束，并保留旧账本行。
4. 生产模式没有获批区块规则，默认拒绝。仅供开发的规则使用 10 单位和一分钟进行测试，均不是正式奖励或时长。

### 证据与完成状态

旧迁移版本断言得到修正。后续一次运行因复用数据库中的 G9 Runtime 残留状态失败；最终普通及 Race 套件各自使用全新独立数据库，分别通过 556/556 测试事件、失败 0、跳过 0。真实独立子进程 Crash A–E 和提交后响应前重放，由新启动的验证进程连接同一 PostgreSQL 核验。并发、退款冲突、守恒及对账测试通过。获批的私有合并及合并后的 CI 已完成；PR 和 canonical GitHub Actions 各六项作业通过，包括 556/556 Test 与 Race、跳过 0、Vet 和三平台构建。

私有里程碑发布完整双语更新后已关闭。公开镜像只复制项目自有且在允许清单中的内容，保持独立历史。架构及 CI 证据留待未来发布；本轮没有发 X。数据库隔离事项继续开放，G19 尚未开始。
