# G18 — Mining Block and Reward Reservation Foundation

Status: **Technical and human acceptance PASS; Stage Close complete**

## English — Primary

### Development sequence

G18 began from the accepted G17 global emission pool. The first design established server-authoritative Mining Blocks with unique monotonic committed heights, atomic pool reservations, immutable block/reservation entries and receipts, exact replay, cancellation release, and a lifecycle that finalizes without allocating player rewards. Its only fixed rule, `DEV_G18_FIXED_BLOCK_REWARD`, reserves 10 test units for a one-minute development interval; it is **NOT FINAL PRODUCTION ECONOMICS**.

An upstream refund exposed a conservation gap when some capacity had already been reserved, including by a finalized block. The approved Recovery Debt / Future Offset design preserves those reservations: compensation consumes Remaining first, records any shortfall as debt, and never makes the pool negative. Future positive emission and cancellation releases pay debt before increasing Remaining. New reservations fail closed while debt exists. The extended invariant is `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`; `Distributed = 0` throughout G18. The G17 pool remains the sole capacity authority.

Migration 0010 adds the Mining Block, reservation, immutable entry/receipt, and recovery audit tables, a recovery-debt field, and updated conservation checks. It preserves existing G11–G17 ledger rows and backfills source-bound recovery audit entries for existing pool revisions. The tests exercised a fresh migration chain and a populated 0009→0010 upgrade. PostgreSQL writes canonical UTC microsecond timestamps and reads persisted values for exact replay.

### Verification record

The early full suite caught an outdated migration version assertion; after that correction, a reused runtime database left stale G9 state in another run. The final local normal and Race suites used separate fresh PostgreSQL databases and passed **556/556 test events each, zero failed, zero skipped**. The database isolation framework remains open technical debt.

Real independent child processes were terminated around Create, Finalize, and Cancel transaction boundaries. Ten subcases cover precommit rollback and after-commit-before-response replay; newly started verifier processes found no ghost block, duplicate reservation, release, finalize, or debt repayment. Concurrency, immutable history, refund collisions, reconciliation, and receipt equality passed. The accepted private PR CI and post-merge canonical CI each passed six jobs: Go Test **556/556**, Go Race **556/556**, **0 skipped**, Go Vet, and Linux/Windows/macOS builds. The real crash matrix passed in both CI suites.

The public mirror exports audited project-owned source, migration, tests, this curated record, and the [architecture decision](../adr/0017-g18-mining-block-reward-reservation.md) with independent public Git history. Architecture, Recovery Debt, conservation, crash, and CI evidence are retained for future build-in-public material. No X post was published.

### Boundaries

Finalize retains reserved capacity pending a future distribution stage; cancellation releases it and first repays any debt. Block heights are unique and monotonic, but absolute continuity is not a contract. No miner, participant, mining power, tool, map, player share, reward allocation, ore issuance or inventory, Bun migration, background scheduler, or production block reward/duration is implemented. The G17 distributed balance remains zero; G19 has not started.

## 中文 — 完整审核版

### 开发经过

G18 从已验收的 G17 全服发行池开始。初版设计建立服务器权威 Mining Block、已提交区块的唯一单调高度、原子池预留、不可变区块／预留流水与回执、精确重放、取消释放，以及完成时不分配玩家奖励的生命周期。唯一固定规则 `DEV_G18_FIXED_BLOCK_REWARD` 使用 10 个测试单位和一分钟开发时长，**不是最终生产经济参数**。

上游退款在容量已被预留、包括已完成区块持有预留时，暴露出守恒缺口。获批的 Recovery Debt / Future Offset 模型保留原预留：补偿先消耗 Remaining，缺口记债，池绝不变负。未来正发行及取消释放先偿债，再增加 Remaining。债务未清时拒绝新预留。扩展守恒式为 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`；G18 全程 `已分发 = 0`。G17 池仍是唯一容量权威来源。

迁移 0010 增加 Mining Block、预留、不可变流水／回执、恢复审计表、恢复债务字段及更新后的守恒约束。它保留既有 G11–G17 账本行，并为已有池修订补录绑定来源的恢复审计流水。测试验证全新迁移链及非空 0009→0010 升级。PostgreSQL 写入 UTC 微秒规范时间戳，重放时读取已持久化的值。

### 验证记录

早期完整测试发现旧迁移版本断言；修正后另一轮受复用 Runtime 数据库中的 G9 残留状态影响。最终本地普通及 Race 套件使用各自全新 PostgreSQL 数据库，分别通过 **556/556 测试事件、失败 0、跳过 0**。数据库隔离框架仍为开放技术债。

在 Create、Finalize、Cancel 的事务边界真实终止独立子进程。十个子场景覆盖提交前回滚和提交后响应前重放；每次由新启动的验证进程确认无幽灵区块、重复预留、释放、完成或偿债。并发、不可变历史、退款冲突、对账和回执相等性均通过。私有 PR CI 和合并后 canonical CI 各有六项作业通过：Go Test **556/556**、Go Race **556/556**、**跳过 0**、Go Vet 以及 Linux／Windows／macOS 构建。真实崩溃矩阵在两套 CI 测试中均通过。

公开镜像在独立 Git 历史中导出经审查的项目自有源码、迁移、测试、本整理记录与[架构决策](../adr/0017-g18-mining-block-reward-reservation.md)。架构、恢复债务、守恒、崩溃及 CI 证据保留为后续 Build in Public 素材。没有发布 X。

### 边界

Finalize 保留预留，等待未来独立分发阶段；Cancel 释放预留并优先偿债。区块高度唯一且单调，但不承诺绝对连续。没有实现矿工、参与者、挖矿算力、工具、地图、玩家份额、奖励分配、矿石发放或库存、馒头迁移、后台 Scheduler、正式区块奖励或时长。G17 的已分发余额仍为零，G19 尚未开始。
