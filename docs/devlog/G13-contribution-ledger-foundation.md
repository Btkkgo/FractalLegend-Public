# G13 — Contribution Ledger Foundation

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

Tracking: private canonical Issue #10 (archive reference only) / 私有 Canonical Issue #10（仅档案引用）

Base commit: `cfe925edb24878e4483671e3445c21b1363991f2`

## English — Primary

### Goal and delivered behavior

G13 adds an internal, server-authoritative, non-transferable Contribution Ledger. It is a progression/eligibility value, not an FB currency. `CONTRIBUTION_RULE_V1` issues 1 point for 1 unit of an explicitly eligible FB system spend. The sole V1 eligible category is internal `SYSTEM_SERVICE`, exercised through synthetic tests; no live game spending producer or client command was added. The centralized `EligibilityPolicy` gives zero points for known non-eligible sources and rejects unknown sources and unsupported rule versions.

A Contribution account records player ID, balance, revision, and timestamps. An immutable entry records source/source ID, eligible FB spend, credited amount, before/after balance, rule version, linked FB transaction, and timestamps. Source + source ID + rule version is unique. Identical replay returns the stored posting; altered intent fails. Account revision, row locks, and immutable entries protect against lost updates, double credit, and drift. Audit queries and reconciliation expose history and detect balance mismatches.

### FB coordination and security

`contribution.Service` validates the source and delegates to the PostgreSQL coordinator. A single `READ COMMITTED` transaction takes a source advisory lock, uses the existing G11 FB posting helper to debit the player's FB account and credit a system FB account, locks the Contribution account, inserts the point entry, updates balance/revision, writes audit, and commits. No second FB balance implementation exists. A database trigger verifies the linked FB transaction is a posted eligible `SYSTEM_SPEND` with the same stable source reference, player debit, system credit, and 1:1 amount; a forged reference to an unrelated FB transaction is rejected.

G12's zero-fee Player Trade stays on its existing settlement path. A real PostgreSQL regression trades 100 FB, verifies the recipient receives 100 FB, and verifies both participants' Contribution balances remain zero. Deposit, Marketplace, Transfer, Red Packet, Tip, Guild Salary, Withdrawal, Refund, Recycle, Reward, Mining, Siege, and Admin Adjustment do not issue Contribution. Unknown source and rule-version tampering fail closed. The Contribution API has no player Transfer, Send, Trade, Deposit, Withdraw, or direct FB exchange method.

The V1 model only credits points. Refunding or reversing a G13-linked FB spend without an atomic point compensation would leave unearned Contribution, so the shared FB posting helper rejects that compensation until a separately authorized design exists. Refund itself credits zero points.

**POST-G13 HARD GATE:** This fail-closed block is temporary safety behavior, not the final production refund design. Before connecting any real NPC, enhancement, manufacturing, asset activation, mining-tool, or other eligible system-spend producer, implement atomic FB refund/reversal plus Contribution reversal/compensation. A successful FB refund must not leave the original points permanently credited. If points have already been spent, define a compensation, recovery, or review process. G13 records this gate without implementing it.

### Fault recovery and concurrency

Injected failures before/after FB debit, before/after Contribution entry, after account update, before final state, and before commit leave no partial FB or Contribution state. A repeated request after rollback succeeds once. Restart/repository reopen replay returns the same entry and FB transaction. Serialization/deadlock retry is bounded and source identity prevents duplicate effects. Initial `Serializable` contention testing exhausted retries under 100 simultaneous unique sources; switching to the existing FB helper's `READ COMMITTED` row-lock discipline produced repeated successful high-contention runs without lost updates. The failed exploratory result is retained as design evidence, not represented as a passing run.

### Verification

- G13 targeted checks: **41/41 PASS**, 0 failures, 0 skips; real PostgreSQL G13 checks within them: **19/19 PASS**.
- Deterministic randomized property sequence: **200/200 iterations PASS**, covering eligible ratio, non-eligible sources, replay, rollback, entry continuity, FB balance linkage, and reconciliation.
- High concurrency: **100 simultaneous unique-source postings PASS**, **100 simultaneous same-source replays PASS**, multiple players in parallel PASS, and competing eligible/non-eligible FB spends PASS. The unique-source contention groups passed five consecutive reruns after the row-lock correction.
- Full Go regression with real PostgreSQL and G9/G11 runtime fixtures: **420/420 PASS**, 0 failures, 0 skips.
- Final full Go Race: **PASS**. `go vet ./...`: **PASS**.
- Windows amd64, Linux amd64, and macOS arm64 Game Server builds: **PASS**.
- Main Node 24/24, G1 9/9, G2 13/14, G3 21/21, G4 7/7, G5 3/3, G6 6/6, G7 6/6, G8 6/6: combined **95/96**. The sole failure is the existing G2 `Atlas Determinism Test`, which cannot access a restricted Legacy archive in this environment; it is not a G13 regression. Typecheck and Web production build: **PASS**.

### Known limits and risks

The only eligible category is an internal foundation boundary with no live producer. A finite retry budget can still fail safely under extreme contention. G13-linked FB refunds/reversals remain blocked until atomic Contribution compensation is designed. Contribution spending, mining, Black Iron Ore, recycle migration, Marketplace, blockchain, wallet, deposit/withdrawal, Ordinals, Guild Salary, Red Packet, Tip, Siege Reward, and G14 are not implemented. X copy is draft only. Manual acceptance is PASS; Stage Close, PR integration, and Public Mirror sync are tracked in Issue #10.

---

## 中文 — 完整对应版本

### 目标与已交付行为

G13 新增内部、Server-authoritative、不可转账的 Contribution Ledger。它是成长/资格数值，不是 FB Currency。`CONTRIBUTION_RULE_V1` 规定，1 单位经显式判定合格的 FB System Spend 产生 1 Point。V1 唯一合格类别是内部 `SYSTEM_SERVICE`，目前只通过 Synthetic Test 验证；没有新增真实游戏消费 Producer 或 Client Command。集中式 `EligibilityPolicy` 对已知非合格来源给出零积分，对未知来源和不支持的 Rule Version 直接拒绝。

Contribution Account 保存 Player ID、Balance、Revision 和时间戳。不可变 Entry 保存 Source/Source ID、合格 FB Spend、Credit Amount、前后 Balance、Rule Version、关联 FB Transaction 和时间戳。Source + Source ID + Rule Version 具有唯一性。相同请求重放返回已存 Posting；改变 Intent 会失败。Account Revision、Row Lock 和不可变 Entry 防止 Lost Update、Double Credit 与 Drift。Audit Query 和 Reconciliation 提供历史，并检测 Balance 不一致。

### FB 协调与安全

`contribution.Service` 验证 Source 后调用 PostgreSQL Coordinator。一个 `READ COMMITTED` Transaction 获取 Source Advisory Lock，复用 G11 FB Posting Helper 从玩家 FB Account 扣款、向 System FB Account 入账，锁定 Contribution Account，写入 Point Entry，更新 Balance/Revision，写入 Audit，最后 Commit。不存在第二套 FB Balance 实现。数据库 Trigger 核验关联 FB Transaction 为已 Posted、Eligible 的 `SYSTEM_SPEND`，并具有相同稳定 Source Reference、Player Debit、System Credit 与 1:1 Amount；伪造无关 FB Transaction 引用会被拒绝。

G12 的 Zero-fee Player Trade 保持既有 Settlement Path。真实 PostgreSQL Regression 执行 100 FB 交易，验证接收方收到 100 FB，且双方 Contribution Balance 都保持零。Deposit、Marketplace、Transfer、Red Packet、Tip、Guild Salary、Withdrawal、Refund、Recycle、Reward、Mining、Siege 和 Admin Adjustment 均不发放 Contribution。未知来源与 Rule Version 篡改默认拒绝。Contribution API 不提供玩家 Transfer、Send、Trade、Deposit、Withdraw 或直接兑换 FB 的方法。

V1 模型只发放积分。若不同时原子补偿积分就 Refund 或 Reverse G13 关联的 FB Spend，会留下不再有对应最终消费的 Contribution，因此共享 FB Posting Helper 暂时拒绝这类补偿，直到未来单独授权的设计完成。Refund 本身发放零积分。

**POST-G13 HARD GATE：** 此 Fail-closed Block 是临时安全行为，不是最终生产 Refund 设计。接入任何真实 NPC、Enhancement、Manufacturing、Asset Activation、Mining Tool 或其他合格 System Spend Producer 之前，必须实现原子 FB Refund/Reversal 加 Contribution Reversal/Compensation。FB 退款成功后，原 Point 不得永久留存。如果 Point 已被消费，必须定义 Compensation、Recovery 或 Review 流程。G13 只记录此 Gate，不在本轮实现。

### 故障恢复与并发

在 FB Debit 前后、Contribution Entry 前后、Account Update 后、Final State 前及 Commit 前注入故障，都不会留下部分 FB 或 Contribution State。回滚后重复请求可以成功且只记一次。Restart/Repository Reopen 后的重放返回同一 Entry 与 FB Transaction。Serialization/Deadlock Retry 有上限，稳定 Source Identity 防止重复作用。初始 `Serializable` 竞争测试在 100 个不同来源同时入账时耗尽 Retry；改用与现有 FB Helper 一致的 `READ COMMITTED` Row Lock 方案后，反复高竞争测试通过且没有 Lost Update。这次初始失败被保留为设计证据，不会被误写成通过。

### 验证

- G13 专项检查：**41/41 PASS**、0 Fail、0 Skip；其中真实 PostgreSQL G13 检查：**19/19 PASS**。
- 确定性随机 Property Sequence：**200/200 Iteration PASS**，覆盖合格比例、非合格来源、Replay、Rollback、Entry 连续性、FB Balance 关联及 Reconciliation。
- 高并发：**100 个不同来源同时 Posting PASS**、**100 个同一来源同时 Replay PASS**、多玩家并行 PASS、合格/非合格 FB Spend 竞争 PASS。修正 Row Lock 方案后，不同来源竞争组连续重跑五次通过。
- 使用真实 PostgreSQL 与 G9/G11 Runtime Fixture 的 Go 完整回归：**420/420 PASS**，0 Fail、0 Skip。
- 最终完整 Go Race：**PASS**。`go vet ./...`：**PASS**。
- Windows amd64、Linux amd64、macOS arm64 Game Server Build：**PASS**。
- 主 Node 24/24、G1 9/9、G2 13/14、G3 21/21、G4 7/7、G5 3/3、G6 6/6、G7 6/6、G8 6/6：合计 **95/96**。唯一失败是既有 G2 `Atlas Determinism Test` 无法在当前环境访问受限制 Legacy Archive；这不是 G13 新增 Regression。Typecheck 与 Web Production Build：**PASS**。

### 已知限制与风险

唯一合格类别是尚无真实 Producer 的内部 Foundation 边界。有限 Retry Budget 在极端竞争下仍可能安全失败。G13 关联的 FB Refund/Reversal 在原子 Contribution Compensation 设计完成前保持阻塞。Contribution Spend、Mining、Black Iron Ore、Recycle Migration、Marketplace、Blockchain、Wallet、Deposit/Withdrawal、Ordinals、Guild Salary、Red Packet、Tip、Siege Reward 和 G14 均未实现。X 文案只保存为 Draft。人工验收已 PASS；Stage Close、PR Integration 与 Public Mirror Sync 结果记录在 Issue #10。
