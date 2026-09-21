# G10 — Trade Foundation

Status: **Technical Acceptance: PASS · Build in Public: PASS · G10 Status: CLOSED**

Tracking: private canonical archive Issue #2 (not publicly accessible).

## English — Primary

### Why

The accepted G7–G9 runtime can create, own, equip, and persist stable item instances, but it had no safe ownership-exchange boundary. G10 adds that boundary before any Trade UI, currency settlement, Marketplace, or Auction House work.

### Problem

A valid player trade must remain conserved through duplicate requests, stale confirmation, concurrent use of the same item, disconnect, restart, inventory limits, and database failure. Both transfers must happen once or neither may happen. Client claims about ownership, revision, completion, or result are never authoritative.

### Architecture

`internal/trade` contains a transport-independent `Service`, state model, repository contract, memory test adapter, and scenario tests. The PostgreSQL adapter lives beside the accepted G9 adapter and reuses `persistence.CharacterAggregate` and `persistence.ItemInstance`. No second Character, Item, Inventory, or Persistence model was created.

The persistent schema adds `trade_sessions`, `trade_offers`, `trade_item_locks`, `trade_settlements`, and `trade_audit_events`. The existing migration runner applies `0002_trade_foundation.sql` transactionally and repeatably.

### Trade state machine

Allowed transitions are:

`NEGOTIATING → READY_TO_SETTLE → COMPLETED`

`NEGOTIATING | READY_TO_SETTLE → CANCELLED`

`NEGOTIATING | READY_TO_SETTLE → EXPIRED`

An offer change from `READY_TO_SETTLE` returns the session to `NEGOTIATING`. Terminal states never return to negotiation. `SETTLING` is deliberately not durable: the PostgreSQL transaction hides the intermediate boundary and exposes only the old valid state or `COMPLETED`.

### Offer revision and confirmation

Every add, remove, or quantity change advances `Revision` and clears both confirmed revisions. `ConfirmTrade` accepts only the current revision and stores that exact value for the participant. Duplicate confirmation of the same revision is idempotent. A stale revision cannot make the session ready.

### Item lock and ownership

Every offered instance receives one persistent server-owned lock. The lock includes Trade ID, owner Character ID, instance ID, and quantity. A primary key on instance ID rejects a second active trade. Add, Confirm, and Finalize consult authoritative Character Aggregate state. Settlement also verifies that the item is still in `INVENTORY`, the quantity is legal, and the lock belongs to the same Trade.

The G8 Equip and Unequip handlers now consult this lock boundary. A locked item returns `ITEM_TRADE_LOCKED`; a lock-store failure returns `ITEM_LOCK_CHECK_FAILED` and leaves equipment state unchanged.

### Quantity and inventory capacity

The current Canonical Item Instance supports `Quantity`, so G10 implements partial stacks. Trading 3 of 10 leaves 7 on the original instance and creates one server-generated instance with quantity 3 for the receiver. Capacity is checked after both outgoing and incoming offers are applied. If either final aggregate exceeds the existing 20-slot rule, the transaction fails and both inventories remain unchanged.

### Atomic settlement and idempotency

The PostgreSQL repository locks the Trade Session and both Character rows, stages both aggregate changes, advances both Character revisions, rewrites both item/equipment collections, records one settlement, releases locks, records completion, and commits once. A failure at any point rolls back the entire unit.

`trade_settlements.trade_id` is unique. A repeated Finalize on a completed Trade returns the stored Settlement ID and timestamp. It does not run the transfer again.

### Persistence, disconnect, and recovery

Session state, offers, current confirmations, locks, settlement result, and audit events are durable. A new Service instance restores a `NEGOTIATING` or `READY_TO_SETTLE` Trade with its locks intact. A completed Trade restores its result and remains idempotent. A non-terminal disconnect cancels the Trade and releases locks. Cancel and Expire never change ownership.

### Audit events

The implementation emits `trade.created`, `trade.offer_changed`, `trade.confirmed`, `trade.cancelled`, `trade.expired`, `trade.settlement_started`, `trade.completed`, and `trade.settlement_failed`. Events contain Trade ID, participants, revision, previous/new state, outcome, and time. They contain no passwords, tokens, database URLs, or production identities.

### Security model

The server rejects stale revision, duplicate item use, invalid owner, illegal quantity, equipped items, capacity overflow, expired replay, terminal-state mutation, unsupported assets, and locked-item equipment mutation. The database transaction prevents partial transfer; persistent uniqueness prevents duplicate lock and settlement. A real PostgreSQL failure injected after the first Character revision write rolled back both Character Aggregates and left the Trade ready for retry.

### Verification prepared in this worktree

- New G10 Go tests: **29/29 PASS**.
- Trade Domain and memory transaction scenarios: PASS.
- Real PostgreSQL migration, restart/idempotency, unique lock, and injected transaction rollback: PASS.
- Existing Go regression with real PostgreSQL integration: **290/290 PASS**, with no skipped test.
- Go race with both PostgreSQL integration databases enabled: PASS.
- Windows amd64, Linux amd64, and macOS arm64 Game Server builds: PASS.
- TypeScript typecheck, Web production build, G8 browser regression, and a real standalone Game Server health check with PostgreSQL: PASS.
- G8 locked Equip/Unequip mutation gate: PASS.
- The original macOS Historical Node run produced **199/221 PASS** and 22 environment/platform failures because protected C0/G2 Legacy archives and Windows-only process fixtures were unavailable. No test was weakened, removed, or converted to Skip.
- The same scope was rerun on Windows 11 with Node.js `v22.22.0` in the private Windows acceptance environment with the protected fixtures available. The required 221-test baseline passed, and complete discovery produced **224/224 PASS**, 0 failures, and 0 skips in approximately 35 seconds of runner wall time. The additional three subtests could register only after the protected fixture initialization succeeded.
- The two Windows-native process fixtures passed independently and inside the full suite. Process start, expected behavior, teardown, and assertions all completed successfully.
- G10 Trade on Windows: **29/29 PASS**. Windows Go full regression: **290/290 PASS**, 0 skips. Windows amd64 build: PASS.
- The previous 22 failures were environment/platform-specific and all passed in the private Windows environment.

The sanitized Windows closure screenshot is `docs/social/screenshots/g10-trade-foundation.png`, SHA-256 `c0672d3cd1c998e9a5ce07c8542a118ae93fdfd0b9ffdff7a1a84496624d95b2`. It replaces the earlier review screenshot with SHA-256 `e2dbe70f41236eba07836c5d91fd91d25c14d0b64ab84a2c517710f13896582c`.

### What is not implemented

G10 does not implement FB settlement, Currency, Contribution, Marketplace, Auction House, Trade UI, fees, Warehouse, Wallet, Blockchain, Ordinals, NFT, Recharge, Withdrawal, Mining, Crafting, Guild Treasury, production transport commands, or any G11 feature.

Future player-to-player trade settlement fee remains a product constraint of **0%**. No fee module exists in G10.

### Known limitations

G10 exposes a server/domain foundation and repository contract, not a production player-facing Trade UI or HTTP/WebSocket command surface. Disconnect currently uses immediate safe cancellation for non-terminal trades. Production timeout duration is intentionally not selected in this phase.

### Closure

G10 passed technical acceptance and its Build in Public closure. The feature branch contains the reviewed code, tests, migration, bilingual public records, X draft, and sanitized screenshot. X remains unpublished. G11 has not started.

---

# G10 — 交易基础（完整中文版本）

状态：**Technical Acceptance: PASS · Build in Public: PASS · G10 Status: CLOSED**

跟踪：Private Canonical Archive Issue #2（不可公开访问）。

## 为什么实施 G10

已验收的 G7–G9 Runtime 可以创建、持有、装备和持久化稳定的 Item Instance，但还没有安全的所有权交换边界。G10 在任何 Trade UI、Currency Settlement、Marketplace 或 Auction House 工作之前补上这一边界。

## 问题

合法的玩家交易必须在重复请求、旧 Confirmation、同一物品并发使用、掉线、重启、Inventory Limit 和 Database Failure 下保持守恒。双方转移必须只发生一次，或者双方都不发生。客户端对 Ownership、Revision、Completion 或 Result 的声明都不是权威。

## Architecture

`internal/trade` 包含独立于 Transport 的 `Service`、State Model、Repository Contract、Memory Test Adapter 和场景测试。PostgreSQL Adapter 位于已验收的 G9 Adapter 旁，并复用 `persistence.CharacterAggregate` 和 `persistence.ItemInstance`。本阶段没有创建第二套 Character、Item、Inventory 或 Persistence Model。

持久化 Schema 新增 `trade_sessions`、`trade_offers`、`trade_item_locks`、`trade_settlements` 和 `trade_audit_events`。现有 Migration Runner 以 Transactional、Repeatable 方式应用 `0002_trade_foundation.sql`。

## Trade State Machine

允许的 Transition 为：

`NEGOTIATING → READY_TO_SETTLE → COMPLETED`

`NEGOTIATING | READY_TO_SETTLE → CANCELLED`

`NEGOTIATING | READY_TO_SETTLE → EXPIRED`

`READY_TO_SETTLE` 发生 Offer Change 后会返回 `NEGOTIATING`。Terminal State 永远不能重新进入 Negotiation。系统刻意不持久化 `SETTLING`：PostgreSQL Transaction 隐藏中间边界，只暴露此前的合法状态或 `COMPLETED`。

## Offer Revision 与 Confirmation

每次 Add、Remove 或 Quantity Change 都推进 `Revision`，并清除双方 Confirmed Revision。`ConfirmTrade` 只接受当前 Revision，并为该 Participant 保存这一准确值。对同一 Revision 的重复 Confirmation 是幂等操作。旧 Revision 无法让 Session 进入 Ready。

## Item Lock 与 Ownership

每个 Offered Instance 都获得一个由服务器管理的持久化 Lock。Lock 包含 Trade ID、Owner Character ID、Instance ID 和 Quantity。Instance ID 的 Primary Key 会拒绝第二笔 Active Trade。Add、Confirm 和 Finalize 都查询权威 Character Aggregate State。Settlement 还会验证 Item 仍位于 `INVENTORY`、Quantity 合法，并且 Lock 属于同一 Trade。

G8 Equip 与 Unequip Handler 现在会查询该 Lock Boundary。Locked Item 返回 `ITEM_TRADE_LOCKED`；Lock Store Failure 返回 `ITEM_LOCK_CHECK_FAILED`，并保持 Equipment State 不变。

## Quantity 与 Inventory Capacity

当前 Canonical Item Instance 支持 `Quantity`，因此 G10 实现部分 Stack。交易 10 个中的 3 个时，原 Instance 保留 7 个，并为接收方创建一个由服务器生成、Quantity 为 3 的 Instance。系统在应用双方 Outgoing 和 Incoming Offer 后检查 Capacity。如果任意一方最终 Aggregate 超过现有 20 Slot Rule，Transaction 会失败，双方 Inventory 均保持不变。

## Atomic Settlement 与 Idempotency

PostgreSQL Repository 会锁定 Trade Session 和双方 Character Row，在内存中暂存双方 Aggregate Change，推进双方 Character Revision，重写双方 Item/Equipment Collection，记录唯一 Settlement，释放 Lock，记录 Completion，并只提交一次。任意步骤失败都会回滚整个 Unit。

`trade_settlements.trade_id` 具有唯一约束。对 Completed Trade 重复 Finalize 会返回已保存的 Settlement ID 和 Timestamp，不会再次运行 Transfer。

## Persistence、Disconnect 与 Recovery

Session State、Offer、当前 Confirmation、Lock、Settlement Result 和 Audit Event 都会持久化。新的 Service Instance 可以恢复 `NEGOTIATING` 或 `READY_TO_SETTLE` Trade，并保留其 Lock。Completed Trade 会恢复原结果并保持幂等。非终止 Trade 掉线时会被取消并释放 Lock。Cancel 与 Expire 都不会改变 Ownership。

## Audit Event

实现会产生 `trade.created`、`trade.offer_changed`、`trade.confirmed`、`trade.cancelled`、`trade.expired`、`trade.settlement_started`、`trade.completed` 和 `trade.settlement_failed`。Event 包含 Trade ID、Participants、Revision、Previous/New State、Outcome 和 Time，不包含 Password、Token、Database URL 或 Production Identity。

## Security Model

服务器会拒绝 Stale Revision、Duplicate Item Use、Invalid Owner、Illegal Quantity、Equipped Item、Capacity Overflow、Expired Replay、Terminal-state Mutation、Unsupported Asset 和 Locked-item Equipment Mutation。Database Transaction 防止 Partial Transfer；持久化 Unique Constraint 防止 Duplicate Lock 与 Duplicate Settlement。在第一名 Character Revision 写入后注入的真实 PostgreSQL Failure 已证明：双方 Character Aggregate 全部回滚，Trade 保持 Ready，可安全重试。

## 本工作区准备的验证

- 新增 G10 Go Test：**29/29 PASS**。
- Trade Domain 与 Memory Transaction Scenario：PASS。
- 真实 PostgreSQL Migration、Restart/Idempotency、Unique Lock 和注入 Transaction Rollback：PASS。
- 启用真实 PostgreSQL Integration 的既有 Go Regression：**290/290 PASS**，没有 Skip。
- 同时启用两个 PostgreSQL Integration Database 的 Go Race：PASS。
- Windows amd64、Linux amd64 和 macOS arm64 Game Server Build：PASS。
- TypeScript Typecheck、Web Production Build、G8 Browser Regression，以及使用 PostgreSQL 的真实 Standalone Game Server Health Check：PASS。
- G8 Locked Equip/Unequip Mutation Gate：PASS。
- 最初在 macOS 执行 Historical Node 时，由于缺少受保护 C0/G2 Legacy Archive 且不能运行 Windows-only Process Fixture，结果为 **199/221 PASS** 和 22 个 Environment/Platform Failure。没有弱化、删除测试或把失败项改成 Skip。
- 同一范围随后在具备受保护 Fixture 的 Private Windows Acceptance Environment 中使用 Windows 11 与 Node.js `v22.22.0` 重跑。规定的 221 项基线全部通过；完整发现结果为 **224/224 PASS**、0 Fail、0 Skip，Runner Wall Time 约 35 秒。额外 3 个子测试只有在受保护 Fixture 初始化成功后才能完成注册。
- 两项 Windows-native Process Fixture 已分别独立执行并在完整 Suite 中执行，Process Start、Expected Behavior、Teardown 和 Assertions 均通过。
- Windows 上 G10 Trade：**29/29 PASS**；Windows Go Full Regression：**290/290 PASS**、0 Skip；Windows amd64 Build：PASS。
- 此前 22 个失败均为 Environment/Platform-specific，并已在 Private Windows Environment 中全部通过。

经过脱敏的 Windows Closure Screenshot 为 `docs/social/screenshots/g10-trade-foundation.png`，SHA-256 为 `c0672d3cd1c998e9a5ce07c8542a118ae93fdfd0b9ffdff7a1a84496624d95b2`。它替换了 SHA-256 为 `e2dbe70f41236eba07836c5d91fd91d25c14d0b64ab84a2c517710f13896582c` 的旧 Review Screenshot。

## 未实现内容

G10 不实现 FB Settlement、Currency、Contribution、Marketplace、Auction House、Trade UI、Fees、Warehouse、Wallet、Blockchain、Ordinals、NFT、Recharge、Withdrawal、Mining、Crafting、Guild Treasury、Production Transport Command 或任何 G11 Feature。

未来玩家自由交易 Settlement Fee 的产品约束仍为 **0%**。G10 没有 Fee Module。

## 已知限制

G10 提供 Server/Domain Foundation 和 Repository Contract，不提供面向玩家的正式 Trade UI 或 HTTP/WebSocket Command Surface。Disconnect 当前对非终止 Trade 使用立即安全取消。Production Timeout Duration 刻意留待后续阶段决定。

## 阶段关闭

G10 已通过 Technical Acceptance 并完成 Build in Public Closure。Feature Branch 包含已经 Review 的代码、测试、Migration、双语公开记录、X Draft 和脱敏 Screenshot。X 仍未发布，G11 尚未开始。
