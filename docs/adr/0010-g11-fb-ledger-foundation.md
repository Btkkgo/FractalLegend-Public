# ADR 0010: G11 FB Ledger Foundation

Status: **Accepted · G11 CLOSED / PASS**

Tracking: private canonical archive Issue #3 (not publicly accessible).

## English — Primary

### Context

Fractal Legend needs one server-authoritative FB monetary history before later transfers, settlement, deposits, or withdrawals can be safe. Legacy Gold, Yuanbao, Diamond, recharge-point, and game-point fields have different semantics and are not authoritative for FB.

### Decision

G11 introduces `internal/ledger` with `LedgerAccount`, `LedgerTransaction`, `LedgerEntry`, `LedgerReference`, `Service`, and `Repository`. FB amounts use signed Go `int64` values and PostgreSQL `BIGINT`; the Ledger Core only understands FB atomic units and does not define display precision or a satoshi conversion.

Every production Posting conserves value: the sum of its signed Entries must be zero. Go validates this before persistence, and migration `0003_fb_ledger.sql` adds a `DEFERRABLE INITIALLY DEFERRED` database constraint trigger that verifies the final Posting sum at transaction commit. A materialized account balance and its journal rows update in the same PostgreSQL transaction.

Accounts start at zero, use currency `FB`, and are unique by `(owner_type, owner_id)`. Player balances cannot become negative. Tests obtain genesis funds only through `_test.go` fixtures that pair the player credit with an equal isolated SYSTEM liability debit. No test-funding path is compiled into production.

Posted Transactions and Entries are append-only. PostgreSQL rejects UPDATE and DELETE through immutable-history triggers. Corrections create a new `REVERSAL` or `REFUND` linked to the original; the original record is never changed.

Each mutation has a unique `(reference_type, reference_id)`. A transaction-scoped advisory lock serializes one Reference, the unique constraint supplies final replay protection, and retries return the existing result only when the complete intent matches. A conflicting reuse fails closed.

Affected account rows are locked in sorted account-ID order. This prevents lost updates and provides one stable lock order for opposite transfers. Insufficient funds, overflow, or an injected mid-transaction failure roll back account balances, Transaction, Entries, and durable audit data together.

Reconciliation recalculates each account from immutable Entries and reports `MATCH` or `MISMATCH`; it never repairs data silently. Runtime events expose Transaction Created, Completed, Failed, Reversed, and Reconciliation Mismatch states. Durable posted audit records remain evidence, not a second ledger.

### Why no direct balance mutation

A production `MintFB`, `GrantFB`, `SetBalance`, `AddBalance`, administrator override, or external-credit adapter would bypass conservation and weaken auditability. G11 therefore ships none of them. Future deposit and withdrawal work requires a separate accepted design for confirmation, replay, reorganization, signing, and external settlement.

### Blockchain and G10 boundaries

G11 does not add Blockchain RPC, Wallet, UniSat, Deposit, Withdrawal, Indexer, TXID verification, Confirmation, Reorg, Ordinals, NFT, Contribution, Mining, Marketplace, or Trade-to-FB settlement. G10 Trade state and settlement logic are unchanged.

### Consequences

- Server and PostgreSQL remain the only FB authority.
- Internal transfers are integer, atomic, double-entry, idempotent, restart-durable, and auditable.
- Reversal is a compensating transaction and fails if the account being debited lacks funds.
- Database constraints protect owner uniqueness, Reference uniqueness, currency, amount direction, foreign keys, duplicate reversal, journal immutability, and transaction conservation.
- G11 remains a foundation; it is not an on-chain or production economy release.

---

## 中文 — 完整对应版本

### 背景

在未来的转账、结算、充值或提现能够安全使用 FB 之前，Fractal Legend 需要一套唯一的服务器权威 FB 货币历史。Legacy Gold、Yuanbao、Diamond、充值点和游戏点具有不同语义，不是 FB 的权威来源。

### 决策

G11 新增 `internal/ledger`，包含 `LedgerAccount`、`LedgerTransaction`、`LedgerEntry`、`LedgerReference`、`Service` 和 `Repository`。FB 金额在 Go 中使用有符号 `int64`，在 PostgreSQL 中使用 `BIGINT`；Ledger Core 只理解 FB Atomic Unit，不定义显示精度或 Satoshi 换算。

每一笔生产 Posting 都必须保持价值守恒：其有符号 Entry 合计必须为零。Go 会在持久化前验证，Migration `0003_fb_ledger.sql` 还提供 `DEFERRABLE INITIALLY DEFERRED` 数据库约束 Trigger，在事务提交时验证最终 Posting 合计。物化 Account Balance 与 Journal Row 在同一个 PostgreSQL Transaction 中更新。

Account 默认余额为零，Currency 固定为 `FB`，并通过 `(owner_type, owner_id)` 保持唯一。Player Balance 不能为负。测试 Genesis Funds 只通过 `_test.go` Fixture 建立：Player Credit 必须对应等额的隔离 SYSTEM Liability Debit。任何测试注资路径都不会被编译进生产代码。

已入账 Transaction 和 Entry 只允许追加。PostgreSQL 通过 Immutable-history Trigger 拒绝 UPDATE 和 DELETE。修正通过关联原始记录的新 `REVERSAL` 或 `REFUND` 完成；原始记录永不修改。

每次写操作使用唯一 `(reference_type, reference_id)`。Transaction-scoped Advisory Lock 会串行化同一 Reference，唯一约束提供最终 Replay Protection；只有完整 Intent 相同的重试才返回已有结果，冲突复用会 Fail Closed。

受影响的 Account Row 按 Account ID 排序后加锁。这可以防止 Lost Update，并为相反方向转账提供稳定锁顺序。余额不足、溢出或事务中途注入失败时，Account Balance、Transaction、Entry 和持久 Audit Data 会一起回滚。

Reconciliation 根据不可变 Entry 重新计算每个 Account，并报告 `MATCH` 或 `MISMATCH`；它绝不静默修复数据。Runtime Event 会暴露 Transaction Created、Completed、Failed、Reversed 和 Reconciliation Mismatch 状态。持久的已入账 Audit Record 只是证据，不是第二套账本。

### 为什么没有直接余额修改

生产 `MintFB`、`GrantFB`、`SetBalance`、`AddBalance`、管理员覆盖或 External-credit Adapter 都会绕过守恒并削弱可审计性，因此 G11 不提供这些能力。未来 Deposit 与 Withdrawal 必须先为 Confirmation、Replay、Reorganization、Signing 和 External Settlement 建立独立并通过验收的设计。

### Blockchain 与 G10 边界

G11 不新增 Blockchain RPC、Wallet、UniSat、Deposit、Withdrawal、Indexer、TXID Verification、Confirmation、Reorg、Ordinals、NFT、Contribution、Mining、Marketplace 或 Trade-to-FB Settlement。G10 Trade State 与 Settlement Logic 保持不变。

### 结果

- Server 与 PostgreSQL 是 FB 的唯一权威。
- Internal Transfer 使用整数、原子、Double-entry、幂等、可跨重启恢复并可审计。
- Reversal 使用补偿式 Transaction；如果被扣回资金的 Account 余额不足，则 Reversal 失败。
- 数据库约束保护 Owner Uniqueness、Reference Uniqueness、Currency、Amount Direction、Foreign Key、Duplicate Reversal、Journal Immutability 和 Transaction Conservation。
- G11 仍是基础阶段，不是 On-chain 或 Production Economy Release。
