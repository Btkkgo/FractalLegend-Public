# G15 — Eligible System Spend Engineering Interaction Record

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

## English — Primary

### Authorized scope and starting point

The authorized task resumed G15 after repository consolidation passed manual acceptance. The private canonical and its remote tracking branch were fetched and verified at `86ba77b9dbe39a2bfc1352c3d7842e92e2f2009b`. Two unrelated, uncommitted G14 social files were protected with a path-limited Git stash before the clean `feature/g15-eligible-system-spend` branch was created from canonical. The stash has not been applied to G15 or discarded. Bilingual GitHub Issue #14 was opened as the G15 development record; its final update and closure belong to Stage Close.

### Engineering decisions

The implementation uses G11's FB Ledger, G13's Contribution policy and atomic eligible-spend writer, and G14's refund/recovery coordinator. There is no second ledger. Server-owned Producer configuration determines eligibility, rule version, and refund policy; only internal test Producers may be active. The client cannot submit a decoded `Intent` or claim an eligibility or reward amount. Migration `0007_system_spend.sql` persists an immutable transaction-linked SystemSpend and checks its economic links at insert. The G15 coordinator serializes operation IDs, validates replay and ownership, and commits the FB debit, SystemSpend record, and optional Contribution credit together. G14 checks G15's stored refund permissions without changing historical G13 refunds. Reconciliation combines per-spend checks and existing account-level checks.

### Verification and observed constraints

Real PostgreSQL tests exercised 400 spend/refund property iterations, 100 identical concurrent calls, 100 distinct concurrent spends, 100 mixed spend/refund operations, restart/retry, seven rollback boundaries, G14 schema upgrade, full/partial refunds, recovery debt, and fail-closed requests. The complete observed counts and command outcomes are in [the G15 test report](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md). The G2 Atlas Determinism Test still needs restricted legacy material; no restricted material was imported. Windows cross-compilation and Windows runtime execution are reported separately.

### Scope held for review

No gameplay Producer, public economy endpoint, or production Contribution consumption writer was enabled. At technical acceptance, no commit, push, PR, merge, Issue closure, public mirror synchronization, X post, or G16 work had occurred. Manual acceptance then passed and authorized Stage Close under final safety gates. This record summarizes decisions and evidence; it is not a transcript.

---

## 中文 — 完整对应版本

### 授权范围与起点

仓库整理通过人工验收后，用户授权恢复 G15。已 Fetch 并核实 Private Canonical 与远端跟踪分支均为 `86ba77b9dbe39a2bfc1352c3d7842e92e2f2009b`。在从干净 Canonical 创建 `feature/g15-eligible-system-spend` 分支前，使用仅针对两个路径的 Git Stash 保护了无关的 G14 未提交社交文件。该 Stash 未应用到 G15，也未被丢弃。已创建双语 GitHub Issue #14 作为 G15 开发记录；其最终更新和关闭属于 Stage Close。

### 工程决策

实现复用 G11 FB Ledger、G13 Contribution Policy 与原子合格消费 Writer，以及 G14 退款与债务恢复 Coordinator；没有建立第二套 Ledger。服务器拥有的 Producer 配置决定资格、规则版本和退款策略；仅内部测试 Producer 可以启用。客户端不能提交可解码的 `Intent`，也不能自行声明资格或收益数量。Migration `0007_system_spend.sql` 保存关联 Transaction 的不可变 SystemSpend，并在插入时核查经济关联。G15 Coordinator 串行化 Operation ID，核验重试与归属，并在同一 Transaction 中提交 FB Debit、SystemSpend Record 及可选的 Contribution Credit。G14 对 G15 保存的退款权限进行检查，不改变历史 G13 退款。Reconciliation 结合逐笔消费检查与现有账户级检查。

### 验证与观察到的约束

真实 PostgreSQL 测试覆盖 400 轮消费/退款 Property、100 次相同请求并发、100 次不同消费并发、100 次消费/退款混合操作、重启/重试、七个回滚边界、G14 Schema Upgrade、完整/部分退款、Recovery Debt 及 Fail-Closed 请求。完整观察到的数量与命令结果记录于[G15 测试报告](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md)。G2 Atlas Determinism Test 仍依赖受限制的 Legacy 素材，本轮没有导入这些素材。Windows Cross Compile 与 Windows Runtime 执行分别报告。

### 留待评审的范围

没有启用真实 Gameplay Producer、公开经济 Endpoint 或生产用 Contribution Consumption Writer。在技术验收时，尚未进行 Commit、Push、PR、Merge、Issue 关闭、Public Mirror 同步、X 发布或 G16 开发。随后人工验收通过，并在最终安全 Gate 下授权 Stage Close。本记录整理工程决策和证据，不是聊天记录复制。
