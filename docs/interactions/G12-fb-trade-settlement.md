# G12 — FB-backed Player Trade Settlement Interaction Record

Status: **Technical acceptance PASS · Stage Close authorized**

This is a curated public engineering record. It is not a private chat transcript.

## English — Primary

### Authorized scope and gates

G12 started from Private Canonical commit `0ebaa5d49b32d115ec9dd77998aafa4711f2abb6` on `feature/g12-fb-trade-settlement` after confirming `canonical == origin/canonical`. GitHub Issue #8 was created before implementation. The initial authorization covered local development, migration, tests, bilingual documentation, a sanitized screenshot, X drafts, and a Public Sync Candidate Manifest; it excluded Commit, Push, Merge, Issue closure, Public Mirror sync, X publication, and G13. After G12 technical acceptance was confirmed as PASS, a separate Stage Close instruction authorized Commit, Push, canonical integration, Issue #8 final update/closure, and a safety-gated Public Mirror sync. X publication and G13 remain excluded.

### Engineering decisions

- The Trade Service coordinates settlement while Trade and Ledger keep their existing domain responsibilities.
- G12 uses no FB Hold. Confirmation checks current funds; Finalize locks the relevant accounts and checks again.
- Both-direction offers use Gross Settlement with stable `PLAYER_TRADE` directional references and 0% fee.
- FB offers share the Trade revision and invalidate both confirmations when changed.
- PostgreSQL uses a deterministic Trade → account → character → item-lock → Ledger-row order.
- Serialization/deadlock retries are bounded at three attempts.
- A durable receipt links the completed Trade to its Ledger transaction IDs.
- Accepted known risks: Option A creates no FB Hold, so a balance may change after confirmation; Finalize locks and revalidates accounts and rolls back entirely on insufficient funds. The three-attempt retry budget may be exhausted under extreme contention, causing a safe failure without partial settlement.

### TDD and verification record

The first G12 test run failed at compile time because `SetFBOffer`, FB offer fields, Ledger-aware transaction methods, receipt support, and Ledger transaction IDs did not exist. Implementation followed those failing contracts. Memory tests then passed before the same behavior was connected to real PostgreSQL.

The final targeted set passed 45/45 with no skips. Real PostgreSQL tests covered mixed bilateral settlement, restart replay, concurrency, opposite-direction locking, and failure rollback. The full Go run passed 379/379 with no skips, final Race passed, and all three target builds passed. Node remained 95/96 only because the already-known G2 restricted Legacy archive fixture is unavailable.

One initial Race rerun reused a mutated G9 fixture database and reported `ITEM_ALREADY_EQUIPPED`. The dedicated schema was reset and the complete Race suite passed. This result is retained as environment/test-isolation history; no production behavior or test assertion was weakened.

### Publication boundary

The G12 X text is a draft only. The screenshot is a generated summary of commands actually run and contains no terminal prompt, local path, account, credential, DSN, wallet information, database rows, or restricted Legacy asset. Public Mirror export requires an explicit file allowlist and a fresh safety/provenance review; the private Git history is excluded.

---

## 中文 — 完整对应版本

状态：**技术验收 PASS · 已授权 Stage Close**

这是整理后的公开工程记录，不是私人聊天全文。

### 授权范围与 Gate

确认 `canonical == origin/canonical` 后，G12 从 Private Canonical Commit `0ebaa5d49b32d115ec9dd77998aafa4711f2abb6` 创建 `feature/g12-fb-trade-settlement`。GitHub Issue #8 在实现前建立。最初授权范围包括本地开发、Migration、Test、双语 Documentation、脱敏 Screenshot、X Draft 和 Public Sync Candidate Manifest；当时排除 Commit、Push、Merge、Issue Close、Public Mirror Sync、X Publication 和 G13。G12 技术验收确认为 PASS 后，单独的 Stage Close 指令授权 Commit、Push、Canonical Integration、Issue #8 Final Update/Close，以及通过安全审核后的 Public Mirror Sync。X Publication 和 G13 仍被排除。

### 工程决策

- Trade Service 负责协调 Settlement，Trade 与 Ledger 继续保持现有 Domain Responsibility。
- G12 不建立 FB Hold。Confirmation 检查当前资金；Finalize 锁定相关 Account 后再次检查。
- 双向 Offer 使用 Gross Settlement、稳定的 `PLAYER_TRADE` Directional Reference 和 0% Fee。
- FB Offer 使用同一 Trade Revision，修改后会使双方 Confirmation 失效。
- PostgreSQL 使用确定性的 Trade → Account → Character → Item Lock → Ledger Row 顺序。
- Serialization/Deadlock Retry 最多三次。
- 持久 Receipt 把 Completed Trade 与 Ledger Transaction ID 关联起来。
- 已接受的风险：Option A 不建立 FB Hold，所以 Confirmation 后余额可能变化；Finalize 会锁定并重新验证 Account，余额不足时整个 Transaction 回滚。极端竞争下三次 Retry 可能耗尽，导致安全失败而不产生部分结算。

### TDD 与验证记录

第一次 G12 Test Run 在编译阶段失败，因为 `SetFBOffer`、FB Offer Field、Ledger-aware Transaction Method、Receipt Support 和 Ledger Transaction ID 都尚不存在。实现按这些失败契约推进。Memory Test 通过后，再把相同行为连接到真实 PostgreSQL。

最终专项测试 45/45 通过、0 Skip。真实 PostgreSQL Test 覆盖混合双向结算、Restart Replay、Concurrency、相反方向锁顺序和失败回滚。Go 完整运行 379/379 通过、0 Skip；最终 Race 通过；三个目标平台 Build 全部通过。Node 保持 95/96，唯一原因仍是已知 G2 受限制 Legacy Archive Fixture 不可用。

第一次 Race 重跑复用了已变更的 G9 Fixture Database，并报告 `ITEM_ALREADY_EQUIPPED`。重置专用 Schema 后，完整 Race Suite 通过。该结果作为 Environment/Test-isolation 历史保留；没有弱化任何 Production Behavior 或 Test Assertion。

### 发布边界

G12 X 文本只保存为 Draft。Screenshot 是根据实际执行命令生成的 Summary，不包含 Terminal Prompt、本地路径、账号、Credential、DSN、Wallet Information、Database Row 或受限制 Legacy Asset。Public Mirror Export 必须使用明确的文件 Allowlist 并重新通过 Safety/Provenance Review；Private Git History 被排除。
