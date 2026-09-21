# G13 — Contribution Ledger Foundation Interaction Record

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

This is a curated public engineering decision record, not a private chat transcript.

## English — Primary

### Authorization and sequence

G13 began from clean `canonical` and `origin/canonical` at `cfe925edb24878e4483671e3445c21b1363991f2` on `feature/g13-contribution-ledger-foundation`. Bilingual GitHub Issue #10 was created before implementation. The original authorization covered local implementation, tests, documentation, a sanitized evidence image, and a Public Sync Candidate Manifest; it excluded Commit, Push, Merge, Issue closure, Public Mirror changes, X publication, and G14. After technical and manual acceptance both passed, a separate G13 Stage Close instruction authorized safety-gated Commit, PR integration, Issue closure, and Public Mirror sync. X publication and G14 remain excluded.

### Engineering decisions

- Contribution is a non-transferable progression/eligibility ledger, separate from the G11 FB balance. V1 credits exactly one point for each unit of an explicitly eligible FB system spend.
- A single `EligibilityPolicy` allows only the internal `SYSTEM_SERVICE` category in V1. There is no live producer. Known excluded sources earn zero; unknown sources and unsupported rule versions fail closed. No arbitrary FB debit or G12 Player Trade earns points.
- The internal coordinator writes the G11 FB debit/system credit and Contribution credit in one PostgreSQL transaction. It uses a stable `(source, source_id, rule_version)` key, source advisory lock, deterministic account row locks, revision check, immutable entry, and audit record.
- The database validates the 1:1 rule and matching posted FB transaction, including source reference and player/system double entry. A forged link to an unrelated transaction is rejected.
- V1 is credit-only. Refund/reversal of a G13-linked FB spend is blocked until an atomic Contribution compensation rule is separately designed; the refund event itself earns zero points.
- Reconciliation reports any account-versus-entry discrepancy without silently repairing history.
- **POST-G13 HARD GATE:** The G13-linked FB refund/reversal block is temporary fail-closed safety behavior, not a final production design. Before a real eligible spend producer is connected, an atomic FB refund/reversal plus Contribution reversal/compensation design must be implemented; already-spent points need an explicit compensation, recovery, or review path.

### Test-led revisions and evidence

The initial policy and persistence tests failed before their contracts existed. A real PostgreSQL test then exposed that a forged Contribution entry could link to an unrelated FB transaction; the migration trigger was tightened and the test passed. A refund test exposed a final-spend invariant gap; the shared FB helper was changed to reject G13-linked compensation, then the test passed.

The first `Serializable` implementation exhausted its finite retry budget under 100 concurrent unique sources. Investigation found snapshot conflicts under contention. The design moved to `READ COMMITTED` with explicit row locks, matching G11's posting discipline; the contention group passed five reruns. This initial failure is retained as design evidence, not counted as a passing test.

Final G13 checks passed 41/41, including 19/19 real PostgreSQL integration checks. The 200-iteration property sequence, 100-way unique-source posting, 100-way same-source replay, seven injected rollback gates, restart replay, reconciliation, and a 100 FB G12 Trade with zero Contribution for both players passed. Full Go regression passed 420/420 with zero skips; Race, Vet, and Windows/Linux/macOS builds passed. Node checks totaled 95/96 because the already-known G2 atlas fixture requires a restricted Legacy archive unavailable in this environment; typecheck and Web build passed.

### Publication boundary and accepted risks

The screenshot is a project-owned card composed from observed test results, with no terminal, account, credential, DSN, local path, raw database row, or restricted Legacy asset. Its `MANUAL ACCEPTANCE PENDING` line records the historical capture state; manual acceptance later passed. X copy remains a draft. The Public Sync Manifest is the allowlist starting point for the separately authorized, safety-gated Stage Close export. Extreme contention can exhaust the bounded retry safely. The internal eligible category has no production game producer, and G13-linked FB compensation remains blocked pending future design and acceptance.

---

## 中文 — 完整对应版本

状态：**技术验收 PASS · 人工验收 PASS · 已授权 Stage Close**

这是整理后的公开工程决策记录，不是私人聊天全文。

### 授权范围与顺序

G13 从 `cfe925edb24878e4483671e3445c21b1363991f2` 的干净 `canonical` 和 `origin/canonical` 开始，在 `feature/g13-contribution-ledger-foundation` 开发。双语 GitHub Issue #10 在实现前创建。最初授权范围是本地实现、测试、文档、脱敏证据图和 Public Sync Candidate Manifest；当时排除 Commit、Push、Merge、Issue 关闭、修改 Public Mirror、发布 X 和 G14。技术验收与人工验收均通过后，独立的 G13 Stage Close 指令授权经过安全审查的 Commit、PR Integration、Issue Close 和 Public Mirror Sync。X 发布与 G14 仍被排除。

### 工程决策

- Contribution 是不可转账的成长/资格账本，与 G11 FB Balance 分离。V1 对每单位经显式判定合格的 FB System Spend 恰好发放一个 Point。
- 单一 `EligibilityPolicy` 在 V1 只允许内部 `SYSTEM_SERVICE` 类别，尚无真实游戏 Producer。已知排除来源产生零分；未知来源和不支持的 Rule Version 默认拒绝。任意 FB Debit 或 G12 Player Trade 不会自动发放积分。
- 内部 Coordinator 在同一个 PostgreSQL Transaction 中写入 G11 FB Debit/System Credit 和 Contribution Credit。它使用稳定的 `(source, source_id, rule_version)` Key、Source Advisory Lock、确定性的 Account Row Lock、Revision Check、不可变 Entry 和 Audit Record。
- 数据库验证 1:1 规则及匹配的已 Posted FB Transaction，包括 Source Reference 与 Player/System Double Entry。伪造关联无关 Transaction 会被拒绝。
- V1 只有 Credit。对 G13 关联 FB Spend 的 Refund/Reversal，在另行设计原子 Contribution Compensation 规则之前保持阻塞；Refund 事件本身产生零分。
- Reconciliation 报告 Account 与 Entry 的差异，不会静默修改历史。
- **POST-G13 HARD GATE：** G13 关联 FB Refund/Reversal 的 Block 是临时 Fail-closed 安全行为，不是最终生产设计。接入任何真实合格消费 Producer 以前，必须实现 FB Refund/Reversal 与 Contribution Reversal/Compensation 的原子设计；已经消费的积分需要明确的 Compensation、Recovery 或 Review 路径。

### 测试驱动的修订与证据

初始 Policy 与 Persistence Test 在对应契约尚不存在时失败。之后真实 PostgreSQL Test 发现，伪造 Contribution Entry 可以指向无关 FB Transaction；收紧 Migration Trigger 后测试通过。Refund Test 又发现最终消费不变量的缺口；共享 FB Helper 被修改为拒绝 G13 关联的补偿，测试随后通过。

最初的 `Serializable` 实现在 100 个不同 Source 并发时耗尽有限 Retry Budget。调查确认竞争下出现 Snapshot Conflict。设计改为 `READ COMMITTED` 加显式 Row Lock，与 G11 Posting 规则一致；竞争测试连续重跑五次通过。初始失败作为设计证据保留，不计入通过结果。

最终 G13 专项检查 41/41 通过，其中真实 PostgreSQL Integration 19/19。200 轮 Property Sequence、100 并发不同 Source Posting、100 并发同 Source Replay、七个故障回滚点、Restart Replay、Reconciliation，以及 100 FB 的 G12 Trade 且双方 Contribution 增量为零，均通过。Go 完整回归 420/420、0 Skip；Race、Vet 和 Windows/Linux/macOS Build 通过。Node 合计 95/96，原因是既有 G2 Atlas Fixture 需要当前环境不可用的受限制 Legacy Archive；Typecheck 和 Web Build 通过。

### 发布边界与已知风险

截图是根据已观察测试结果制作的项目自有卡片，不包含 Terminal、账号、Credential、DSN、本地路径、原始 Database Row 或受限制 Legacy Asset。其中 `MANUAL ACCEPTANCE PENDING` 记录的是拍摄时的历史状态；人工验收此后已通过。X 文案仍只是草稿。Public Sync Manifest 是独立授权、受安全审查约束的 Stage Close 导出清单起点。极端竞争可能安全地耗尽有限 Retry。内部合格类别尚无生产游戏 Producer；G13 关联 FB Compensation 在未来设计与验收前保持阻塞。
