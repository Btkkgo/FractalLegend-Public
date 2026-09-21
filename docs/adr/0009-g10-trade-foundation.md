# ADR 0009: G10 Trade Foundation

Status: Accepted — Technical Acceptance: PASS; Build in Public: PASS; G10 Status: CLOSED

## English — Primary

### Context

Player-to-player trade changes ownership of durable `ItemInstance` records. A transport-level exchange or an in-memory coordinator cannot prevent duplicate settlement, stale confirmation, lock loss after restart, or a partial transfer after database failure. G10 therefore needs a transport-independent Trade Domain built on the accepted G7–G9 item, inventory, equipment, ownership, and PostgreSQL persistence chain.

### Decision

The server owns a persistent `TradeSession`. Its non-terminal states are `NEGOTIATING` and `READY_TO_SETTLE`; its terminal states are `COMPLETED`, `CANCELLED`, and `EXPIRED`. Settlement is one PostgreSQL transaction, so no durable `SETTLING` state is exposed. A transaction either persists both character transfers and `COMPLETED`, or rolls all of them back and leaves the session ready for a safe retry.

Every offer mutation advances the offer revision and clears both confirmations. A confirmation stores the exact current revision for one participant. `READY_TO_SETTLE` is reached only when both stored confirmation revisions equal the current offer revision.

An offered `ItemInstance` receives a persistent row in `trade_item_locks`. Its globally unique `item_instance_id` prevents the same instance from entering two active trades. Confirmation and settlement revalidate owner, inventory location, quantity, and lock identity from server state. G8 Equip and Unequip use the same lock boundary and fail closed when lock state cannot be verified.

Settlement loads and locks both Character Aggregates, verifies capacity, applies both directions in memory, and writes both aggregates inside the same database transaction. Full-stack transfer preserves the existing `ItemInstanceID`; partial-stack transfer keeps the source instance and creates one server-generated destination instance for the transferred quantity. Zero, negative, and excessive quantities are rejected.

`trade_settlements.trade_id` is unique. A completed session returns its stored settlement result, so sequential or concurrent duplicate Finalize intent cannot transfer ownership twice. Persistent session, offer, confirmation, lock, settlement, and audit rows allow a new Service instance to recover negotiating, ready, and completed trades after restart.

Cancel and expiration are terminal transitions before settlement. They preserve ownership and release every lock owned by the trade. Disconnect invokes the same cancellation rule while the trade is non-terminal. Once atomic settlement has acquired the transaction boundary, disconnect cannot create a partial outcome.

Audit events record trade ID, both participants, revision, state transition, outcome, and timestamp. They never include credentials, service tokens, database URLs, or client-authored ownership claims.

`TradeAsset` is represented by a narrow extension input. G10 accepts `ITEM` only and rejects every other asset type with `ErrUnsupportedAsset`; it does not emulate FB or external assets.

### Product constraint

Future player-to-player trade settlement fee is **0%**. G10 has no currency settlement and introduces no fee calculation module.

### Consequences

- PostgreSQL is the durable authority for active locks and final settlement.
- The HTTP/WebSocket layer can submit intent later without owning trade rules.
- A database transaction failure preserves both original inventories and the pre-settlement trade state.
- Runtime code that mutates an offered item must consult the shared lock boundary.
- FB settlement, Marketplace, Auction House, Trade UI, Wallet, Blockchain, Ordinals, and fees remain unimplemented.

### Validation status

The private Windows acceptance environment passed the complete historical Node scope (**224/224**, 0 failures, 0 skips), both Windows-native process fixtures, G10 Trade (**29/29**), Go regression (**290/290**, 0 skips), and the Windows amd64 build. The actual 224-test discovery satisfies the required 221-test baseline and includes three additional subtests that could not register on the macOS host without protected fixtures. The earlier **199/221** macOS result and 22 environment/platform failures remain documented; no test was weakened, removed, or converted to Skip.

---

## 中文 — 完整对应版本

### 背景

玩家之间的交易会改变持久化 `ItemInstance` 的所有权。仅由 Transport 承担交换，或只使用 In-memory Coordinator，无法防止重复结算、旧 Confirmation、重启后 Lock 丢失，或数据库失败造成的部分转移。因此，G10 需要在已验收的 G7–G9 Item、Inventory、Equipment、Ownership 和 PostgreSQL Persistence 数据链上，建立独立于 Transport 的 Trade Domain。

### 决策

服务器拥有可持久化的 `TradeSession`。非终止状态为 `NEGOTIATING` 和 `READY_TO_SETTLE`；终止状态为 `COMPLETED`、`CANCELLED` 和 `EXPIRED`。Settlement 在一个 PostgreSQL Transaction 中完成，因此不暴露持久化的 `SETTLING` 状态。一个 Transaction 要么同时保存双方 Character Transfer 和 `COMPLETED`，要么全部回滚，并让 Session 保持在可以安全重试的 Ready 状态。

每次 Offer Mutation 都推进 Offer Revision，并清除双方 Confirmation。每名参与者的 Confirmation 保存其确认的准确当前 Revision。只有双方保存的 Confirmation Revision 都等于当前 Offer Revision 时，Trade 才能进入 `READY_TO_SETTLE`。

加入 Offer 的 `ItemInstance` 会在 `trade_item_locks` 中获得持久化记录。全局唯一的 `item_instance_id` 防止同一个 Instance 同时进入两笔 Active Trade。Confirmation 与 Settlement 都会根据 Server State 重新验证 Owner、Inventory Location、Quantity 和 Lock Identity。G8 Equip 与 Unequip 使用相同的 Lock Boundary；无法验证 Lock State 时会 Fail Closed。

Settlement 会加载并锁定双方 Character Aggregate，验证 Capacity，在内存中应用双向交换，并在同一个 Database Transaction 中写入双方 Aggregate。整组 Stack Transfer 保留原 `ItemInstanceID`；部分 Stack Transfer 保留来源 Instance，并为被转移数量创建一个由服务器生成的目标 Instance。系统拒绝零、负数和超过持有量的 Quantity。

`trade_settlements.trade_id` 具有唯一约束。Completed Session 会返回其已保存的 Settlement Result，因此顺序或并发的重复 Finalize Intent 都不能再次转移所有权。持久化的 Session、Offer、Confirmation、Lock、Settlement 和 Audit Row 允许新的 Service Instance 在重启后恢复 Negotiating、Ready 和 Completed Trade。

Cancel 与 Expiration 是 Settlement 前的终止 Transition。它们保持物品所有权不变，并释放该 Trade 拥有的全部 Lock。交易处于非终止状态时，Disconnect 使用相同的取消规则。一旦 Atomic Settlement 获得 Transaction Boundary，Disconnect 不能产生部分结果。

Audit Event 记录 Trade ID、双方 Participant、Revision、State Transition、Outcome 和 Timestamp。它们不包含 Credential、Service Token、Database URL 或客户端编造的 Ownership Claim。

`TradeAsset` 通过一个窄的扩展输入表达。G10 只接受 `ITEM`，其他 Asset Type 全部返回 `ErrUnsupportedAsset`；本阶段不模拟 FB 或 External Asset。

### 产品约束

未来玩家自由交易 Settlement Fee 为 **0%**。G10 没有 Currency Settlement，也不引入 Fee Calculation Module。

### 结果

- PostgreSQL 是 Active Lock 与最终 Settlement 的持久化权威。
- 未来 HTTP/WebSocket 层可以提交 Intent，但不拥有 Trade Rule。
- Database Transaction Failure 会保留双方原始 Inventory 和结算前 Trade State。
- 修改 Offered Item 的 Runtime Code 必须查询共享 Lock Boundary。
- FB Settlement、Marketplace、Auction House、Trade UI、Wallet、Blockchain、Ordinals 和 Fees 均未实现。

### 验证状态

Private Windows Acceptance Environment 已通过完整 Historical Node Scope（**224/224**、0 Fail、0 Skip）、两项 Windows-native Process Fixture、G10 Trade（**29/29**）、Go Regression（**290/290**、0 Skip）和 Windows amd64 Build。实际发现的 224 项满足规定的 221 项基线，并包含 3 项在 macOS Host 缺少受保护 Fixture 时无法注册的额外子测试。此前 macOS 上 **199/221** 和 22 个 Environment/Platform Failure 仍被保留记录；没有弱化、删除测试或把失败项改成 Skip。
