# G11 — FB Ledger Foundation Interaction Record

Status: **CLOSED / PASS · Manual acceptance confirmed**

Tracking: private canonical archive Issue #3 (not publicly accessible).

This is a curated engineering-decision record, not a copy of the private conversation.

## English — Primary

### Accepted baseline and sequence

G10 Trade Foundation remained closed and unchanged at base commit `b47f7db006bf121f194bfb82808c3864c1c5f90e`. G11 proceeded on `feature/fb-ledger-foundation` in the Private Canonical Repository. The Public Mirror remained untouched during development and received only the accepted sanitized allowlist export after stage close.

The chosen sequence is:

`G10 Trade Foundation → G11 Internal FB Ledger → later separately reviewed adapters`

### Decisions

- The Game Server is the only authority for FB intent and outcome.
- Core amounts use integer FB atomic units (`int64` / `BIGINT`).
- Every production Posting conserves value and is checked in Go and by a deferred PostgreSQL constraint trigger.
- Immutable Entries are audit history; materialized balances are a read model updated in the same transaction.
- New Accounts start at zero and currency is fixed to `FB`.
- Production has no `MintFB`, `GrantFB`, `SetBalance`, `AddBalance`, administrator override, or external-credit service.
- Synthetic genesis funding exists only in `_test.go` and uses an equal isolated SYSTEM liability Posting.
- Stable row-lock order, Reference advisory locks, and database uniqueness protect concurrency and idempotency.
- Reversal is a compensating Transaction. It never edits the original and fails when return funds are insufficient.
- Reconciliation reports mismatches and never repairs silently.
- Runtime lifecycle events cover Created, Completed, Failed, Reversed, and Reconciliation Mismatch.
- G10 Trade-to-FB settlement and all blockchain adapters remain future work.

### Execution evidence

The latest tests ran against real local PostgreSQL, including Repository reload and complete Game Server restart. G11 passed 44/44 top-level tests; the full Go suite passed 334/334 with no failures or skips; full Race and vet passed; and three target builds passed. Web/typecheck and G1/G3–G8 Node checks passed. G2 preserved one honest environment failure because the restricted Legacy archive path is unavailable on this host.

Security review found no client balance command, production direct-funding method, blockchain integration, real Deposit/Withdrawal, wallet integration, Contribution, Mining, Marketplace, or G10 Trade logic change.

### Stage-close state

Manual acceptance confirmed G11 as PASS. Final verification, Private Canonical commit/PR/merge, Issue #3 final update and closure, and the separately scanned sanitized allowlist export to the independent Public Mirror completed in that order. The G11 X post remains an unpublished draft, and G12 has not started.

---

## 中文 — 完整对应版本

这是整理后的工程决策记录，不是私人聊天全文。

### 已验收基线与阶段顺序

G10 Trade Foundation 保持 CLOSED 且未被修改，基础提交为 `b47f7db006bf121f194bfb82808c3864c1c5f90e`。G11 在 Private Canonical Repository 的 `feature/fb-ledger-foundation` 上执行。开发期间 Public Mirror 保持不变，只在 Stage Close 后接收已验收的脱敏 Allowlist Export。

采用的阶段顺序为：

`G10 Trade Foundation → G11 Internal FB Ledger → 后续独立审核的 Adapter`

### 决策

- Game Server 是 FB Intent 与 Outcome 的唯一权威。
- 核心金额使用整数 FB Atomic Unit（`int64` / `BIGINT`）。
- 每笔生产 Posting 都保持价值守恒，并由 Go 与 Deferred PostgreSQL Constraint Trigger 双重检查。
- Immutable Entry 是 Audit History；Materialized Balance 是在同一事务中更新的 Read Model。
- 新 Account 默认余额为零，Currency 固定为 `FB`。
- 生产代码不存在 `MintFB`、`GrantFB`、`SetBalance`、`AddBalance`、管理员覆盖或 External-credit Service。
- Synthetic Genesis Funding 只存在于 `_test.go`，并使用等额的隔离 SYSTEM Liability Posting。
- 稳定 Row-lock Order、Reference Advisory Lock 与数据库唯一约束共同保护 Concurrency 和 Idempotency。
- Reversal 是 Compensating Transaction。它不修改原 Transaction，并在 Return Funds 不足时失败。
- Reconciliation 只报告 Mismatch，绝不静默修复。
- Runtime Lifecycle Event 覆盖 Created、Completed、Failed、Reversed 和 Reconciliation Mismatch。
- G10 Trade-to-FB Settlement 与所有 Blockchain Adapter 都属于未来工作。

### 执行证据

最新测试使用真实本地 PostgreSQL，覆盖 Repository Reload 与完整 Game Server Restart。G11 顶层测试 44/44 通过；Go 完整套件 334/334 通过、0 Fail、0 Skip；完整 Race 与 Vet 通过；三个目标平台 Build 通过。Web/Typecheck 和 G1/G3–G8 Node 检查通过。由于本机无法访问受限制 Legacy Archive 路径，G2 如实保留 1 项环境失败。

Security Review 确认不存在 Client Balance Command、生产 Direct-funding Method、Blockchain Integration、真实 Deposit/Withdrawal、Wallet Integration、Contribution、Mining、Marketplace 或 G10 Trade Logic 变更。

### Stage Close 状态

人工验收已经确认 G11 为 PASS。最终验证、Private Canonical Commit/PR/Merge、Issue #3 Final Update 与关闭，以及向独立 Public Mirror 执行的独立扫描脱敏 Allowlist Export 已按此顺序完成。G11 X 内容仍是未发布草稿，G12 尚未开始。
