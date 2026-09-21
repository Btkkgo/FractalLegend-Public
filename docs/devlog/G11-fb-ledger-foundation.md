# G11 — FB Ledger Foundation

Status: **CLOSED / PASS · Technical Acceptance: PASS**

Tracking: private canonical archive Issue #3 (not publicly accessible).

Base commit: `b47f7db006bf121f194bfb82808c3864c1c5f90e`

## English — Primary

### Why

G10 established a persistent player-item Trade Foundation, but Fractal Legend still lacked an authoritative FB monetary history. G11 builds the internal Ledger before any Trade-to-FB settlement, blockchain, wallet, deposit, withdrawal, Marketplace, Contribution, or Mining work.

### What changed

- Added the transport-independent `internal/ledger` Domain and `Service`.
- Added `LedgerAccount`, `LedgerTransaction`, `LedgerEntry`, and unique `LedgerReference` models.
- Added Memory and PostgreSQL Repository implementations.
- Added PostgreSQL migration `0003_fb_ledger.sql`.
- Wired the accepted PostgreSQL Store into the Game Server as `LedgerStore` without adding a browser/client command.
- Added lifecycle events for Created, Completed, Failed, Reversed, and Reconciliation Mismatch outcomes.

### Architecture and amount model

The client cannot author a balance, Posting, settlement result, or Reversal result. The Game Server owns the intent and the Ledger owns persistence. Amounts use Go `int64` and PostgreSQL `BIGINT` FB atomic units. Zero and negative transfer amounts are rejected; checked addition protects `int64` boundaries and overflow.

Each Account starts at zero, uses currency `FB`, carries an optimistic revision, and is unique by owner type and owner ID. PLAYER and SYSTEM are implemented; the model remains open to later ESCROW and GUILD account types without implementing their business flows now.

### Conservation and production funding boundary

Every production Posting is balanced before persistence, and a deferred PostgreSQL constraint trigger verifies at commit that `Σ postings = 0`. Materialized balances and immutable Entries commit together. No production `MintFB`, `GrantFB`, `SetBalance`, `AddBalance`, administrator override, or external-credit service exists.

Tests use a `_test.go` genesis fixture only. Each synthetic player credit is offset by an equal isolated SYSTEM liability debit, so fixture Posting sums and total system supply remain zero. This helper is absent from production builds.

### Persistence, concurrency, and idempotency

PostgreSQL stores accounts, transactions, entries, and audit records. Entry/Transaction UPDATE and DELETE are rejected by immutable-history triggers. Account rows are locked in sorted ID order. A transaction-scoped advisory lock serializes each business Reference, while a unique database constraint supplies final replay protection.

A retry with the same Reference and identical intent returns the stored Transaction. Reusing it for a different amount, account, type, original Transaction, classification, or reason returns `ErrReferenceConflict`. Concurrent spending cannot lose updates, create a negative PLAYER balance, or settle the same Reference twice.

### Transfer, rollback, reversal, and reconciliation

A successful Transfer creates one negative and one equal positive Entry. Missing accounts, self transfer, invalid amount, overflow, or insufficient funds fail without committed state. Failure injection after the first Entry proves that the entire PostgreSQL transaction rolls back.

Reversal creates a new compensating Transaction and preserves the original. Database uniqueness prevents a second Reversal of the same original. If the destination no longer has enough funds to return, Reversal fails, produces no partial Entry, and never creates debt automatically.

Reconciliation recomputes each Account from immutable Entries. It reports mismatches and emits an audit event, but does not repair state. Total-supply reporting compares account and Entry totals.

### Restart and auditability

Repository reload and a complete Game Server stop/start preserve Account balances, Transaction, Entries, and Reference idempotency in real PostgreSQL. Posted lifecycle records remain durable. Runtime events cover created, completed, failed, reversed, and mismatch states without logging credentials or database URLs.

### Fresh verification

Commands were executed on the uncommitted G11 worktree with dedicated local PostgreSQL databases:

- G11 top-level tests: **44/44 PASS**.
- Ledger Domain and production-surface tests: **32/32 PASS**.
- Real PostgreSQL Ledger integration: **11/11 PASS**.
- Deterministic 1,000-operation conservation test: **PASS**.
- Memory and PostgreSQL high-concurrency/double-spend/opposite-direction tests: **PASS**.
- Full Go regression: **334/334 PASS**, 0 failures, 0 skips.
- `go test -race ./... -count=1`: **PASS** across all 14 Go packages.
- `go vet ./...`: **PASS**.
- TypeScript typecheck: **PASS**.
- Web production build: **PASS** with the existing Vite large-chunk warning.
- Main Node suite: **24/24 PASS**.
- G1: **9/9 PASS**; G3: **21/21 PASS**; G4: **7/7 PASS**; G5: **3/3 PASS**; G6: **6/6 PASS**; G7: **6/6 PASS**; G8: **6/6 PASS**.
- Local G2: **13/14 PASS**. `Atlas Determinism Test` could not access the restricted Legacy archive path. It was not skipped, weakened, or rewritten.
- Windows amd64, Linux amd64, and macOS arm64 Game Server builds: **PASS**.

### Security impact

G11 rejects self transfer, zero/negative amount, overflow, insufficient balance, duplicate/conflicting Reference, concurrent overspend, duplicate Reversal, and Reversal with insufficient return funds. Database constraints protect currency, ownership uniqueness, Reference uniqueness, amount direction, foreign keys, duplicate compensation, transaction conservation, and immutable journal history. No Ledger client command or direct balance-mutation endpoint exists.

### Not implemented

Real Fractal Bitcoin Deposit/Withdrawal, Blockchain RPC, Wallet, UniSat, Indexer, TXID verification, Confirmation, Reorg handling, Signer, Hot Wallet, Ordinals, NFT, Contribution, Mining, Guild Treasury, Red Packet, Marketplace, Trade-to-FB settlement, and G12 are not implemented.

### Stage close

Manual G11 acceptance is confirmed. The Private Canonical commit/PR/merge and Issue #3 closure completed, followed by a separately scanned sanitized Public Mirror allowlist export. The X post remains an unpublished draft, and G12 has not started.

---

## 中文 — 完整对应版本

状态：**CLOSED / PASS · Technical Acceptance: PASS**

跟踪：Private Canonical Archive Issue #3（无法公开访问）。

基础提交：`b47f7db006bf121f194bfb82808c3864c1c5f90e`

### 为什么

G10 已建立持久的玩家物品 Trade Foundation，但 Fractal Legend 仍缺少权威 FB 货币历史。G11 在任何 Trade-to-FB Settlement、Blockchain、Wallet、Deposit、Withdrawal、Marketplace、Contribution 或 Mining 工作之前建立内部 Ledger。

### 变更内容

- 新增独立于 Transport 的 `internal/ledger` Domain 与 `Service`。
- 新增 `LedgerAccount`、`LedgerTransaction`、`LedgerEntry` 和唯一 `LedgerReference` Model。
- 新增 Memory 与 PostgreSQL Repository 实现。
- 新增 PostgreSQL Migration `0003_fb_ledger.sql`。
- 把已验收 PostgreSQL Store 作为 `LedgerStore` 接入 Game Server，但没有新增 Browser/Client Command。
- 新增 Created、Completed、Failed、Reversed 和 Reconciliation Mismatch Lifecycle Event。

### Architecture 与金额模型

Client 不能决定 Balance、Posting、Settlement Result 或 Reversal Result。Game Server 掌握 Intent，Ledger 掌握 Persistence。金额使用 Go `int64` 与 PostgreSQL `BIGINT` FB Atomic Unit。零数和负数 Transfer 会被拒绝；Checked Addition 保护 `int64` 边界和 Overflow。

每个 Account 默认余额为零，Currency 为 `FB`，带有 Optimistic Revision，并通过 Owner Type 与 Owner ID 保持唯一。本轮实现 PLAYER 与 SYSTEM；Model 允许未来扩展 ESCROW 和 GUILD，但当前不实现其业务 Flow。

### 守恒与生产注资边界

每笔生产 Posting 在持久化前必须平衡，Deferred PostgreSQL Constraint Trigger 会在提交时验证 `Σ postings = 0`。Materialized Balance 与 Immutable Entry 同时提交。生产代码不存在 `MintFB`、`GrantFB`、`SetBalance`、`AddBalance`、管理员覆盖或 External-credit Service。

测试只使用 `_test.go` Genesis Fixture。每次 Synthetic Player Credit 都对应等额的隔离 SYSTEM Liability Debit，因此 Fixture Posting 合计与整个系统 Total Supply 仍为零。该 Helper 不会进入生产 Build。

### 持久化、并发与幂等

PostgreSQL 保存 Account、Transaction、Entry 和 Audit Record。Immutable-history Trigger 会拒绝 Entry/Transaction UPDATE 与 DELETE。Account Row 按 ID 排序后加锁。Transaction-scoped Advisory Lock 会串行化每个 Business Reference，数据库唯一约束提供最终 Replay Protection。

相同 Reference 与完全一致 Intent 的重试会返回已保存 Transaction。若金额、Account、Type、Original Transaction、Classification 或 Reason 不同，则返回 `ErrReferenceConflict`。Concurrent Spend 不会造成 Lost Update、PLAYER Negative Balance，或对同一 Reference 重复结算。

### Transfer、回滚、冲正与对账

成功 Transfer 会创建一条负数 Entry 与一条等额正数 Entry。Account 不存在、Self Transfer、非法金额、Overflow 或余额不足都会在无提交状态的情况下失败。在第一条 Entry 后注入失败的测试证明整个 PostgreSQL Transaction 会回滚。

Reversal 会创建新的 Compensating Transaction 并保留原记录。数据库唯一约束防止同一原 Transaction 被第二次 Reversal。若 Destination 已无足够资金退回，Reversal 会失败，不产生部分 Entry，也不会自动制造 Debt。

Reconciliation 根据 Immutable Entry 重新计算每个 Account。它会报告 Mismatch 并发出 Audit Event，但不会修复状态。Total-supply Reporting 会比较 Account 与 Entry Total。

### 重启与可审计性

真实 PostgreSQL 中的 Repository Reload 和完整 Game Server Stop/Start 会保留 Account Balance、Transaction、Entry 和 Reference Idempotency。已入账 Lifecycle Record 持久保存。Runtime Event 覆盖 Created、Completed、Failed、Reversed 和 Mismatch 状态，且不会记录 Credential 或 Database URL。

### 本轮最新验证

以下命令在未提交 G11 Worktree 上使用专用本地 PostgreSQL Database 实际执行：

- G11 顶层测试：**44/44 PASS**。
- Ledger Domain 与 Production-surface Test：**32/32 PASS**。
- 真实 PostgreSQL Ledger Integration：**11/11 PASS**。
- 确定性 1,000 Operation Conservation Test：**PASS**。
- Memory 与 PostgreSQL High-concurrency/Double-spend/Opposite-direction Test：**PASS**。
- Go 完整回归：**334/334 PASS**、0 Fail、0 Skip。
- `go test -race ./... -count=1`：全部 14 个 Go Package **PASS**。
- `go vet ./...`：**PASS**。
- TypeScript Typecheck：**PASS**。
- Web Production Build：**PASS**，保留既有 Vite Large-chunk Warning。
- Main Node Suite：**24/24 PASS**。
- G1：**9/9 PASS**；G3：**21/21 PASS**；G4：**7/7 PASS**；G5：**3/3 PASS**；G6：**6/6 PASS**；G7：**6/6 PASS**；G8：**6/6 PASS**。
- 本机 G2：**13/14 PASS**。`Atlas Determinism Test` 无法访问受限制 Legacy Archive 路径；该测试没有被 Skip、弱化或重写。
- Windows amd64、Linux amd64 和 macOS arm64 Game Server Build：**PASS**。

### 安全影响

G11 会拒绝 Self Transfer、Zero/Negative Amount、Overflow、Insufficient Balance、Duplicate/Conflicting Reference、Concurrent Overspend、Duplicate Reversal，以及 Return Funds 不足的 Reversal。数据库约束保护 Currency、Owner Uniqueness、Reference Uniqueness、Amount Direction、Foreign Key、Duplicate Compensation、Transaction Conservation 和 Immutable Journal History。项目不存在 Ledger Client Command 或 Direct Balance-mutation Endpoint。

### 未实现内容

真实 Fractal Bitcoin Deposit/Withdrawal、Blockchain RPC、Wallet、UniSat、Indexer、TXID Verification、Confirmation、Reorg Handling、Signer、Hot Wallet、Ordinals、NFT、Contribution、Mining、Guild Treasury、Red Packet、Marketplace、Trade-to-FB Settlement 和 G12 均未实现。

### 阶段封板

G11 人工验收已经确认。Private Canonical Commit/PR/Merge 与 Issue #3 关闭已经完成，随后执行了经过独立扫描的脱敏 Public Mirror Allowlist Export。X 内容仍是未发布草稿，G12 尚未开始。
