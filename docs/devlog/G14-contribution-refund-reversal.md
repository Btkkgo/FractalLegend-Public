# G14 — Contribution Refund / Reversal Atomic Compensation

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

Base: `748e3156e85832fcb95334d51bc2be03ea7f9581`

## English — Primary

### Why and what changed

G13's Contribution credit could not be safely retained after its source FB system spend was refunded, so the shared G11 FB helper blocked G13-linked refunds. G14 replaces that temporary hard gate with a private, atomic refund coordinator. Its inputs identify the original eligible FB spend, player, FB account, stable refund reference, amount, and refund/reversal type; callers cannot set the resulting Contribution balance or recovery debt. Ordinary G11 refund calls still cannot refund a G13-linked spend without the coordinator. No live eligible spend producer or browser refund command was added.

Migration `0006_contribution_refund.sql` adds separate nonnegative `recovery_debt` and `review_required` account state; immutable compensation and consumption relations; and `debt_settled` on future earning entries. Existing G13 entries retain their historical values and remain immutable. The compensation row links the original earning, original FB spend, new FB refund, original rule version, before/after available balance and debt, amount reclaimed immediately, and debt created. A deferred database trigger refuses a G13-linked FB refund with no matching Contribution compensation, even if application code is bypassed. A second trigger validates linkage and cumulative refund bounds. The G11 single-compensation rule still applies outside the G14 reference type.

### Architecture and economic safety

The coordinator serializes each refund reference and original FB spend, checks the original `ELIGIBLE` `SYSTEM_SPEND` and immutable 1:1 V1 entry, checks player/account ownership and remaining refundable amount, posts FB refund or reversal through G11, writes Contribution compensation, updates available balance/debt, and commits all effects together. One original 100 FB spend can be refunded as 100, 40, or 20+30+50; a cumulative amount above 100 is rejected. A duplicate reference reuses its committed result, while changed intent fails. A reversal follows the same compensation invariant.

If 100 Contribution was earned and 80 was synthetically used, a full refund removes the 20 available points and creates 80 recovery debt. A subsequent eligible credit of 50 reduces debt to 30 with no spendable credit. Another 50 clears the debt and makes only 20 available. This settlement occurs inside the new FB-spend/Contribution-credit transaction; it is not delayed. The internal `ValidateSpend` guard rejects spending during debt or manual review and rejects insufficient available balance. There is no production Contribution spend writer; synthetic consumption in tests demonstrates the recovery model without creating a live product feature.

Reconciliation reads a consistent PostgreSQL snapshot and checks account arithmetic, debt arithmetic, refund/compensation totals, per-original bounds, rule versions, and compensation-to-FB linkage. An anomaly is reported and marks the account for manual review; sensitive Contribution operations fail closed. The service does not guess or silently rewrite history.

### Verification and observed limits

G14 focused unit spend-guard checks passed **6/6** including subtests. G14 focused real PostgreSQL tests passed **21/21** including subtests. Two deterministic property sequences passed **200/200 iterations each**: one covers bounded refunds and duplicate replay; the other interleaves refund, debt creation, and new credits, checking recovery before spendable balance. Concurrency covered **100 identical refund calls**, **100 distinct one-unit partial refunds**, a competing 70+70 refund, and **100 concurrent credit/refund operations** with 20 concurrent reconciliation reads. Restart/reopen replay, three injected rollback points, a deferred FB-ledger commit failure after Contribution was written, uncommitted-connection-loss restart recovery, orphan-FB database rejection, new database migration, upgrade from the G13 schema, non-eligible spend/transfer isolation, cross-player rejection, unsupported historical rule rejection, and manual-review detection passed.

The full Go regression with real PostgreSQL and fresh G9/G11 runtime fixtures passed **447/447** with no reported failure. Full Go Race passed with separate fresh G9/G11 runtime databases. `go vet ./...`, Windows amd64, Linux amd64, macOS arm64 builds, TypeScript typecheck, and Web build passed. Node checks remained **95/96**: the existing G2 Atlas Determinism Test still lacks the restricted Legacy archive; all other Node checks passed, so this limitation is unchanged. An initial full Go run against a reused G9 runtime test database failed its equipment-restore test because the fixture already contained an equipped item. A fresh runtime database run passed; the initial failure is retained here rather than counted as a passing run.

### Known limitations and next gate

G14 passed technical and manual acceptance, but is not a release. `ValidateSpend` is only a server-side rule check; a future authorized production spend integration must include an atomic consumption writer and use the guard in that transaction. The internal eligible source still has no live producer. Marketplace, Mining, Blockchain, Wallet, Ordinals, Deposit, Withdrawal, Guild Salary, Red Packet, Tip, game UI changes, and G15 were not implemented. Stage Close authorizes Commit, Push, PR, Issue closure, and reviewed Public Mirror sync. X publication remains manual and unauthorized.

---

## 中文 — 完整对应版本

### 原因与变更

G13 发放的 Contribution 在其来源 FB System Spend 退款后不能安全保留，因此共享 G11 FB Helper 原本阻止 G13 关联退款。G14 用私有的原子退款 Coordinator 替代这道临时硬门。输入明确原始合格 FB 消费、玩家、FB Account、稳定的退款 Reference、金额及 Refund/Reversal 类型；调用方不能设定最终 Contribution Balance 或 Recovery Debt。普通 G11 Refund 调用仍不能绕过 Coordinator 对 G13 关联消费退款。本轮没有新增真实合格消费 Producer 或 Browser Refund Command。

Migration `0006_contribution_refund.sql` 为 Account 新增独立且非负的 `recovery_debt` 与 `review_required`，新增不可变的补偿和 Consumption 关系，并在未来收益 Entry 上新增 `debt_settled`。既有 G13 Entry 保留历史数值且继续不可变。补偿行关联原收益、原 FB 消费、新 FB 退款、原始规则版本、前后可用余额和债务、立即回收数量及新建债务。Deferred Database Trigger 即使在绕过应用代码时，也会拒绝缺少匹配 Contribution 补偿的 G13 关联 FB 退款。另一 Trigger 校验关联关系和累计退款上限。G11 单次补偿规则在 G14 Reference Type 之外保持不变。

### 架构与经济安全

Coordinator 对每个退款 Reference 和原 FB 消费串行化，检查原始 `ELIGIBLE` `SYSTEM_SPEND` 与不可变的 V1 1:1 Entry，核验玩家/Account 归属和剩余可退款金额，通过 G11 写入 FB Refund/Reversal，再写入 Contribution 补偿、更新可用余额与债务，最后统一 Commit。原始 100 FB 消费可以退款 100、退款 40，或分为 20+30+50；累计超过 100 会被拒绝。重复 Reference 返回已提交结果；修改 Intent 则失败。Reversal 遵守同一补偿不变量。

如果原先获得 100 Contribution，且 Synthetic Fixture 已使用 80，完整退款会收回剩余可用的 20 并建立 80 Recovery Debt。随后新获得 50 合格 Contribution 时，债务降为 30，可用余额仍为零；再次获得 50 后债务清零，只有剩余 20 变为可用。这些偿还发生在新的 FB Spend/Contribution Credit Transaction 内部，不采用延迟任务。内部 `ValidateSpend` Guard 在存在债务或需人工复核时拒绝消费，也拒绝超出可用余额的请求。尚无生产用 Contribution Spend Writer；测试中的 Synthetic Consumption 只验证恢复模型，不构成真实产品功能。

Reconciliation 读取一致的 PostgreSQL Snapshot，检查 Account 算术、债务算术、FB 退款与 Contribution 补偿总额、每笔原始消费的上限、规则版本及补偿到 FB 的关联。异常会被报告并把 Account 标记为需要人工复核；敏感 Contribution 操作默认拒绝。Service 不猜测修复，也不静默重写历史。

### 验证与观察到的限制

G14 专项消费 Guard 单元检查 **6/6** 通过，包含子测试。G14 专项真实 PostgreSQL 检查 **21/21** 通过，包含子测试。两组确定性 Property Sequence 各 **200/200** 轮通过：一组覆盖退款上限与重复请求重放；另一组交错执行退款、债务创建和新收益，检查先偿债后增加可用余额。并发覆盖 **100 次相同退款调用**、**100 次不同 Reference 的 1 单位部分退款**、竞争性的 70+70 退款，以及 **100 次并发 Credit/Refund 操作**和 20 次并发 Reconciliation Read。Restart/Reopen 重放、三个故障回滚点、Contribution 写入后的 FB Ledger 延迟提交失败、未提交事务断线后的重启恢复、数据库拒绝孤立 FB 退款、全新建库 Migration、从 G13 Schema 升级、非合格消费/Transfer 隔离、跨玩家拒绝、不支持的历史规则版本拒绝及人工复核检测均通过。

使用真实 PostgreSQL 和全新 G9/G11 Runtime Fixture 的 Go 完整回归 **447/447** 通过，未报告失败。完整 Go Race 使用另外两套全新的 G9/G11 Runtime 数据库并通过。`go vet ./...`、Windows amd64、Linux amd64、macOS arm64 Build、TypeScript Typecheck 与 Web Build 通过。Node 检查仍为 **95/96**：既有 G2 Atlas Determinism Test 缺少受限制 Legacy Archive；其他 Node 检查均通过，因此这项限制没有恶化。最初一次 Go 完整回归复用了 G9 Runtime 测试数据库，其 Fixture 中已经有已装备 Item，导致装备恢复测试失败；换全新 Runtime 数据库后通过。本日志保留最初失败，不把它计入通过记录。

### 已知限制与下一道 Gate

G14 已通过技术与人工验收，但不代表 Release。`ValidateSpend` 只是服务端规则检查；未来经单独授权的生产消费集成必须加入原子 Consumption Writer，并在同一 Transaction 中使用 Guard。内部合格 Source 仍没有真实 Producer。本轮没有实现 Marketplace、Mining、Blockchain、Wallet、Ordinals、Deposit、Withdrawal、Guild Salary、Red Packet、Tip、游戏 UI 变更或 G15。Stage Close 已授权 Commit、Push、PR、Issue 关闭及经审查的 Public Mirror Sync。X 发布仍由用户手动决定，本轮未获授权。
