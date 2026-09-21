# ADR 0011: G12 Atomic Item + FB Trade Settlement

Status: **Accepted decision · G12 technical acceptance PASS**

Tracking: private canonical Issue #8 (archive reference only) / 私有 Canonical Issue #8（仅档案引用）

## English — Primary

### Context

G10 can atomically exchange persistent `ItemInstance` ownership, and G11 provides an immutable, conserving FB Ledger. Calling those systems in separate transactions would permit partial settlement: items could move while FB does not, or FB could post while item ownership remains unchanged. G12 needs one server-authoritative settlement boundary without adding a Marketplace, escrow product, blockchain adapter, or player-controlled balance endpoint.

### Decision

The Trade `Service` is the Settlement Coordinator. The Trade Domain still owns session state, offers, revision-bound confirmation, item locks, and completion. The Ledger Domain still validates and writes balanced immutable postings. The PostgreSQL Store exposes the existing Ledger posting logic to an already-open database transaction, so the Trade adapter does not contain a copied Ledger implementation.

One serializable PostgreSQL transaction locks and validates the Trade, affected FB accounts, characters, and item locks; transfers item ownership; posts balanced FB Ledger transactions and entries; updates materialized balances; stores the settlement receipt; marks the Trade completed; and commits. Any error rolls back every mutation.

### FB reservation decision

G12 adopts **Option A: no FB Hold or Reservation**. Confirmation validates the current balance, and Finalize locks the affected FB account rows and validates the authoritative balance again. This prevents double spending while keeping cancel, expiry, disconnect, restart, and failure recovery free of hidden frozen funds. A Hold system would add lifecycle and recovery complexity mainly needed by longer-lived Marketplace orders, which are outside G12.

### Gross settlement decision

G12 uses **Gross Settlement**. `A_TO_B` and `B_TO_A` are separate `PLAYER_TRANSFER` Ledger transactions when both directions are nonzero. Each uses a stable `PLAYER_TRADE` reference derived from the Trade ID and direction. Gross postings preserve the exact accepted offers for audit, receipt reconstruction, and any future compensating recovery workflow. The fee is exactly **0%**: each debit has an equal counterparty credit and no fee entry.

### Lock order and retry

The Finalize lock order is:

1. Trade row.
2. FB account rows ordered by account ID.
3. Character rows ordered by character ID.
4. Item lock rows ordered by item instance ID.
5. Ledger reference and journal rows.

PostgreSQL `40001` serialization failures and `40P01` deadlocks are retryable for at most three transaction attempts. Other errors fail immediately. Stable Ledger references and completed-state replay protect retries from duplicate item or FB movement.

### Known risks

- **Option A balance change after confirmation:** No FB Hold is created. Another legitimate operation may spend the confirmed balance before Finalize. Finalize locks the affected account rows and rechecks authoritative balances; insufficient funds fail the entire transaction without partial item or FB settlement. This is an accepted direct-Trade semantic.
- **Retry exhaustion:** Under extreme database contention, the finite three-attempt retry budget can be exhausted. Finalize then fails without a partial settlement; callers may retry the operation. This remains a known risk and is not a G12 acceptance blocker.

### Revision, idempotency, and receipt

Each FB offer is a nonnegative Go `int64` / PostgreSQL `BIGINT`. Changing either an item offer or FB offer advances the Trade revision and clears both confirmations. Finalize requires both confirmations to match the current revision. A completed Trade returns its stored `SettlementResult` on repeated Finalize calls, including after Repository reload or Game Server restart.

The durable receipt reconstructs participants, item offers, FB offers, final revision, completion status/time, Settlement ID, and linked Ledger transaction IDs. It is evidence of server settlement and is not a client authorization token.

### Failure rollback

Tests inject failures after item ownership update, during Ledger balance entry application, after Ledger write, before completed-state transition, and after Trade persistence immediately before commit. In each case, item ownership, FB balances, Ledger history, receipt, and Trade state roll back together.

### Security and scope

The production API exposes no client method to set a balance, force completion, author a Ledger posting, or inject a settlement result. G12 does not implement Marketplace, Auction, Trade UI, escrow, fees, blockchain settlement, Wallet, Deposit, Withdrawal, Contribution, Mining, automated reversal, or G13. A future authorized recovery must use new compensating Ledger transactions and explicit reverse item transfer; it must never edit posted history.

---

## 中文 — 完整对应版本

### 背景

G10 可以原子交换持久化 `ItemInstance` 所有权，G11 提供不可变且守恒的 FB Ledger。如果两个系统分别提交 Transaction，就可能产生部分结算：Item 已转移但 FB 未入账，或 FB 已入账但 Item 所有权未变化。G12 需要一个 Server-authoritative Settlement Boundary，同时不新增 Marketplace、Escrow 产品、Blockchain Adapter 或玩家可控制的 Balance Endpoint。

### 决策

Trade `Service` 作为 Settlement Coordinator。Trade Domain 继续负责 Session State、Offer、Revision-bound Confirmation、Item Lock 和 Completion；Ledger Domain 继续负责验证并写入守恒、不可变的 Posting。PostgreSQL Store 允许现有 Ledger Posting Logic 复用已打开的 Database Transaction，因此 Trade Adapter 不包含复制出来的第二套 Ledger 实现。

一个 Serializable PostgreSQL Transaction 会锁定并验证 Trade、受影响的 FB Account、Character 和 Item Lock；转移 Item Ownership；写入守恒的 FB Ledger Transaction 与 Entry；更新 Materialized Balance；保存 Settlement Receipt；将 Trade 标记为 Completed；最后 Commit。任一步骤发生错误，所有变更都会一起 Rollback。

### FB Reservation 决策

G12 采用 **Option A：不建立 FB Hold 或 Reservation**。Confirmation 会验证当前 Balance；Finalize 会锁定受影响的 FB Account Row，并再次验证权威余额。该设计可以防止 Double Spend，同时不会让 Cancel、Expiry、Disconnect、Restart 或 Failure Recovery 遗留隐藏冻结资金。Hold System 增加的 Lifecycle 和 Recovery 复杂度主要适用于存续时间更长的 Marketplace Order，而 Marketplace 不属于 G12。

### Gross Settlement 决策

G12 使用 **Gross Settlement**。当两个方向的金额都不为零时，`A_TO_B` 与 `B_TO_A` 分别形成独立的 `PLAYER_TRANSFER` Ledger Transaction。每个方向都使用由 Trade ID 和方向派生的稳定 `PLAYER_TRADE` Reference。Gross Posting 会完整保留双方已确认报价，便于 Audit、Receipt Reconstruction，以及未来可能采用补偿方式的 Recovery Workflow。手续费严格为 **0%**：每个 Debit 都有等额 Counterparty Credit，不产生 Fee Entry。

### Lock Order 与 Retry

Finalize 的加锁顺序为：

1. Trade Row。
2. 按 Account ID 排序的 FB Account Row。
3. 按 Character ID 排序的 Character Row。
4. 按 Item Instance ID 排序的 Item Lock Row。
5. Ledger Reference 与 Journal Row。

PostgreSQL `40001` Serialization Failure 与 `40P01` Deadlock 最多重试三次 Transaction；其他错误立即失败。稳定 Ledger Reference 与 Completed-state Replay 共同保护重试过程，防止重复移动 Item 或 FB。

### 已知风险

- **Option A 在确认后余额变化：**不建立 FB Hold。其他合法操作可能在 Confirmation 后、Finalize 前消耗已确认时的余额。Finalize 会锁定受影响的 Account Row，并重新检查权威余额；余额不足时整个 Transaction 失败，Item 与 FB 都不会发生部分结算。这是已接受的 Direct Trade 语义。
- **重试次数耗尽：**在极端数据库竞争下，最多三次的有限 Retry 可能全部耗尽。此时 Finalize 失败且不会产生部分结算；调用方可以重试该操作。这是保留的已知风险，不构成 G12 验收阻塞。

### Revision、Idempotency 与 Receipt

每个 FB Offer 都是非负 Go `int64` / PostgreSQL `BIGINT`。任何一方修改 Item Offer 或 FB Offer，Trade Revision 都会增加，双方 Confirmation 都会失效。Finalize 要求双方 Confirmation 与当前 Revision 一致。已完成 Trade 再次 Finalize 时会返回已保存的 `SettlementResult`，Repository Reload 或完整 Game Server Restart 后也保持该语义。

持久 Receipt 可以重建参与者、Item Offer、FB Offer、最终 Revision、Completion Status/Time、Settlement ID 和关联 Ledger Transaction ID。它是服务器结算证据，不是 Client Authorization Token。

### Failure Rollback

测试分别在 Item Ownership Update 之后、Ledger Balance Entry 应用过程中、Ledger Write 之后、Completed-state Transition 之前，以及 Trade 持久化完成但 Database Commit 之前注入失败。每种情况下，Item Ownership、FB Balance、Ledger History、Receipt 和 Trade State 都会一起回滚。

### Security 与范围

生产 API 不暴露由 Client 设置 Balance、强制 Completion、编写 Ledger Posting 或注入 Settlement Result 的方法。G12 不实现 Marketplace、Auction、Trade UI、Escrow、Fee、Blockchain Settlement、Wallet、Deposit、Withdrawal、Contribution、Mining、自动 Reversal 或 G13。未来经授权的 Recovery 必须使用新的补偿式 Ledger Transaction 和明确的反向 Item Transfer，绝不能修改已入账历史。
