# G14 — Contribution Refund / Reversal Interaction Record

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

This is a curated public engineering decision record, not a private conversation transcript.

## English — Primary

### Authorization and boundaries

G14 started after G13 Stage Close PASS. The private `canonical` and `origin/canonical` heads were fetched and checked against the required base `748e3156e85832fcb95334d51bc2be03ea7f9581`; the worktree was clean. Work proceeded on `feature/g14-contribution-refund-reversal`, and a bilingual private G14 Issue was opened before implementation. This authorization covers local code, real PostgreSQL tests, bilingual public-document candidates, security review, and a Public Mirror candidate list. It explicitly excludes Commit, Push, PR, Merge, canonical changes, Issue closure, Public Mirror sync, X publication, and G15.

### User decisions carried into engineering

- Refund and reversal of an eligible G13 system spend must compensate its Contribution in the same database transaction. Full and multiple partial refunds are required; cumulative refund cannot exceed the original spend.
- Already-used Contribution cannot be ignored. Available balance stays nonnegative; any unrecovered amount becomes separate Recovery Debt. Future eligible credits settle debt before becoming available. Spending during debt or manual review fails closed.
- Original rule version, original spend and player ownership are authoritative. A client cannot name a balance, compensation, or debt result. Non-eligible spends, trade, and transfer remain at zero Contribution effect.
- The G11 FB Ledger and G13 Contribution Ledger are reused. Original earning entries remain immutable. Refunds append immutable linked compensation, and reconciliation reports drift for manual review rather than guessing repairs.
- No live eligible spend or Contribution spend producer is introduced. Build in Public records are English-primary with complete Chinese versions; X remains a draft for manual publication.

### Test-led revisions and acceptance evidence

Tests were written before the refund implementation and initially failed because the refund request, debt state, and spend guard did not exist. A real PostgreSQL migration test then exposed the actual historical G13 check-constraint name; the additive migration was corrected. G13 regression caught an unintended entry ordering change, so the original ordering was restored. A concurrency review identified that multi-query reconciliation needed one consistent snapshot; it now uses a read-only `REPEATABLE READ` transaction and passed concurrent reads during writes. Database triggers were strengthened to reject a G13-linked FB refund even if a caller bypasses the shared Ledger helper.

G14 focused unit spend-guard checks passed 6/6; real PostgreSQL checks passed 21/21, including two 200-iteration property sequences, 100 duplicate requests, 100 distinct partial refunds, 100 concurrent credit/refund operations, 20 concurrent reconciliation reads, rollback injection, deferred FB-ledger commit failure, uncommitted-connection-loss restart recovery, restart replay, orphan rejection, and G13-schema migration upgrade. Full Go passed 447/447 on fresh runtime fixtures; full Go Race passed with separate fresh fixtures; three-platform builds, Vet, Typecheck, and Web build passed. Node remained 95/96 because of the existing restricted G2 Atlas archive dependency. An initial full Go attempt against a reused G9 runtime test database failed its equipment-restore check; the fresh-database rerun passed.

### Publication and review state

G14 passed technical and user manual acceptance; Stage Close is authorized. The candidate manifest lists only project-owned source, migration, tests, and bilingual records. The reviewed allowlist is authorized for Public Mirror export during Stage Close; no social post is authorized. The test-only synthetic Consumption relation does not expose a production spend API. Future product producers need separate design and authorization.

---

## 中文 — 完整对应版本

状态：**技术验收 PASS · 人工验收 PASS · 已授权 Stage Close**

这是整理后的公开工程决策记录，不是私人对话全文。

### 授权与边界

G14 在 G13 Stage Close PASS 后开始。先 Fetch 私有 `canonical` 与 `origin/canonical`，确认两者都等于要求的基线 `748e3156e85832fcb95334d51bc2be03ea7f9581`，且 Working Tree 干净。工作在 `feature/g14-contribution-refund-reversal` 进行，双语私有 G14 Issue 在实现前建立。本轮授权涵盖本地代码、真实 PostgreSQL 测试、双语公开文档候选、安全检查和 Public Mirror 候选清单；明确排除 Commit、Push、PR、Merge、改动 canonical、关闭 Issue、Public Mirror Sync、发布 X 及 G15。

### 用户决策如何落地

- 合格 G13 System Spend 的 Refund/Reversal 必须在同一个数据库 Transaction 中补偿 Contribution。支持完整和多次部分退款，累计金额不得超过原始消费。
- 已使用的 Contribution 不能被忽略。可用余额保持非负；尚未回收的部分成为独立 Recovery Debt。未来合格 Credit 先偿债，再进入可用余额。存在债务或需要人工复核时，消费默认拒绝。
- 原始规则版本、原始消费及玩家归属是权威依据。客户端不能指定 Balance、Compensation 或 Debt 结果。非合格消费、Trade 与 Transfer 的 Contribution 作用仍为零。
- 复用 G11 FB Ledger 与 G13 Contribution Ledger。原收益 Entry 保持不可变。退款追加不可变且可追溯的补偿记录；Reconciliation 报告漂移并交由人工复核，不猜测修复。
- 不引入真实合格消费或 Contribution 消费 Producer。Build in Public 记录以英文为主并附完整中文版本；X 只准备供用户手动发布的草稿。

### 测试驱动修订与验收证据

先写退款测试，初始失败的原因是 Refund Request、Debt State 和 Spend Guard 尚不存在。真实 PostgreSQL Migration Test 随后发现 G13 历史 Check Constraint 的实际名称，已修正增量 Migration。G13 回归发现 Entry 排序被意外改变，随后恢复原有排序。并发检查发现多查询 Reconciliation 需要一致快照，现使用只读 `REPEATABLE READ` Transaction，并在写入并发期间通过读测。数据库 Trigger 也加强为：即使调用方绕过共享 Ledger Helper，G13 关联 FB 退款仍不能缺少对应补偿。

G14 专项消费 Guard 单元检查 6/6 通过；真实 PostgreSQL 检查 21/21 通过，覆盖两组各 200 轮 Property、100 个重复请求、100 个不同部分退款、100 次并发 Credit/Refund、20 次并发 Reconciliation Read、故障回滚、FB Ledger 延迟提交失败、未提交事务断线后的重启恢复、Restart Replay、孤立记录拒绝及 G13 Schema 升级。使用全新 Runtime Fixture 的 Go 完整回归 447/447 通过；完整 Go Race 使用另外两套全新 Fixture 并通过；三平台 Build、Vet、Typecheck 和 Web Build 通过。Node 仍为 95/96，原因是既有 G2 Atlas Test 依赖受限制 Archive。首次 Go 全量运行复用了 G9 Runtime 测试数据库，装备恢复检查因此失败；换新数据库重跑后通过。

### 公开与审核状态

G14 已通过技术与用户人工验收，Stage Close 已获授权。候选 Manifest 只列项目自有 Source、Migration、Test 和双语记录。本轮已授权按审查通过的清单执行 Public Mirror 导出；未授权发布社交内容。仅供测试的 Synthetic Consumption 关系不暴露生产 Spend API。未来产品 Producer 需要另外设计和授权。
