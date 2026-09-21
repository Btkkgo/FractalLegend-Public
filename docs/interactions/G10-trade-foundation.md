# G10 Trade Foundation — Public Engineering Interaction Record

Status: Technical Acceptance: PASS; Build in Public: PASS; G10 Status: CLOSED. This is a curated engineering record, not a private chat transcript.

## English — Primary

### Requirements recorded

- Build a server-authoritative player-to-player item trade foundation.
- Reuse the accepted G7–G9 Canonical Item, Inventory, Equipment, ownership, and Persistence chain.
- Bind both confirmations to the current Offer Revision.
- Persist locks, reject concurrent use of one item, and block supported runtime mutation of locked items.
- Settle both ownership directions atomically and make Finalize idempotent.
- Recover negotiating, ready, and completed trades after restart.
- Support Cancel, Expire, safe Disconnect behavior, quantity, capacity, and audit events.
- Keep future player trade fee at 0% as a product constraint without implementing a fee or FB module.
- Produce English-primary public records with a complete Chinese version.
- Stop before commit, push, Issue closure, X publication, or G11.

### Architecture decisions

The Trade Domain was placed in `internal/trade`, independent of transport. Its repository transaction exposes the accepted G9 `CharacterAggregate` instead of inventing a trade-specific inventory. PostgreSQL stores the Trade state and applies both character changes in one transaction. A memory adapter uses the same aggregate types for deterministic domain and concurrency tests.

The state machine persists `NEGOTIATING`, `READY_TO_SETTLE`, `COMPLETED`, `CANCELLED`, and `EXPIRED`. No `SETTLING` row is exposed because the database transaction makes that intermediate state unnecessary. Offer changes return a ready session to negotiation and invalidate both confirmations.

Persistent item locks use `item_instance_id` as their unique key. Full-stack transfer preserves the instance ID; a partial transfer creates a server-generated destination instance and reduces the source instance. G8 Equip and Unequip consult the same lock checker.

### Technical issues resolved

The G9 PostgreSQL adapter rewrites one Character Aggregate at a time. That method cannot atomically exchange globally unique item IDs between two owners. G10 therefore stages both aggregates, deletes both old item collections, then inserts both final collections inside one transaction. A test-only internal failure injector proves rollback after the first Character revision write.

Adding migration `0002` exposed an assumption in the migration rollback test, which had also used version `0002` for a deliberately broken migration. The test fixture was moved to unused version `9999`; the production migration sequence remains `0001` then `0002`.

The initial macOS run did not contain the protected C0/G2 Legacy archives used by historical absolute-path Node tests and could not execute the Windows-only process fixtures. It recorded **199/221 PASS** and 22 environment/platform failures. No protected resource was copied, no historical test was weakened, and no failing case was converted to Skip.

The same scope was then run in the private Windows acceptance environment. The required 221-test baseline passed; complete discovery produced **224/224 PASS**, 0 failures, and 0 skips because three additional subtests could register once protected fixture initialization succeeded. Both Windows-native process fixtures passed independently and in the full suite. This confirms that the previous 22 failures were environment/platform-specific.

### Final execution result

The implementation contains the Trade Service, state transitions, revision-bound confirmation, persistent locks, partial-stack handling, capacity validation, atomic two-character settlement, unique settlement result, cancellation, expiration, disconnect cancellation, restart recovery, audit events, PostgreSQL migration, real transaction failure injection, and G8 mutation gate.

The Windows acceptance run passed G10 Trade **29/29**, Go full regression **290/290** with 0 skips, and the Windows amd64 build. Real PostgreSQL restart, duplicate Finalize, unique lock, and rollback scenarios passed. Existing tests removed: 0. No test was skipped to obtain a pass. No production code or test logic changed during the Windows closure run.

### Closure decision

The user approved final G10 commit, push, Build in Public closure, and Issue #2 closure after reviewing the technical acceptance result. X remains unpublished. G11 has not started.

---

## 中文 — 完整对应版本

状态：Technical Acceptance: PASS；Build in Public: PASS；G10 Status: CLOSED。这是整理后的公开工程记录，不是私人聊天全文。

### 已记录的需求

- 建立 Server-authoritative 的玩家面对面物品交易基础。
- 复用已验收的 G7–G9 Canonical Item、Inventory、Equipment、Ownership 和 Persistence 数据链。
- 将双方 Confirmation 与当前 Offer Revision 绑定。
- 持久化 Lock，拒绝同一 Item 的并发使用，并阻止当前 Runtime 对 Locked Item 的受支持修改操作。
- 原子完成双方 Ownership Transfer，并让 Finalize 保持幂等。
- 重启后恢复 Negotiating、Ready 和 Completed Trade。
- 支持 Cancel、Expire、安全的 Disconnect Behavior、Quantity、Capacity 和 Audit Event。
- 把未来玩家交易手续费 0% 保留为产品约束，不实现 Fee 或 FB Module。
- 公开记录使用 English Primary + 完整中文版本。
- 在 Commit、Push、Issue Close、X Publication 或 G11 前停止。

### Architecture 决策

Trade Domain 位于 `internal/trade`，独立于 Transport。其 Repository Transaction 暴露已验收的 G9 `CharacterAggregate`，没有创建 Trade 专用 Inventory。PostgreSQL 保存 Trade State，并在同一个 Transaction 中应用双方 Character Change。Memory Adapter 使用相同的 Aggregate Type，承担可确定复现的 Domain 与 Concurrency Test。

State Machine 持久化 `NEGOTIATING`、`READY_TO_SETTLE`、`COMPLETED`、`CANCELLED` 和 `EXPIRED`。由于 Database Transaction 使中间状态没有必要，因此不暴露 `SETTLING` Row。Offer Change 会让 Ready Session 返回 Negotiation，并使双方 Confirmation 失效。

Persistent Item Lock 使用 `item_instance_id` 作为 Unique Key。整组 Stack Transfer 保留 Instance ID；Partial Transfer 创建服务器生成的目标 Instance，并减少来源 Instance。G8 Equip 与 Unequip 查询同一个 Lock Checker。

### 已解决的技术问题

G9 PostgreSQL Adapter 每次重写一个 Character Aggregate。该方法无法在两个 Owner 之间原子交换全局唯一 Item ID。因此，G10 会暂存双方 Aggregate，删除双方旧 Item Collection，再在同一个 Transaction 中插入双方最终 Collection。内部 Test-only Failure Injector 证明在第一名 Character Revision 写入后仍可完整回滚。

增加 Migration `0002` 后，Migration Rollback Test 中原来把 `0002` 用作故意损坏 Migration 的假设被暴露。该 Test Fixture 已移动到未使用的 Version `9999`；Production Migration Sequence 保持为 `0001` 后接 `0002`。

最初的 macOS Run 不包含 Historical Absolute-path Node Test 所需的受保护 C0/G2 Legacy Archive，也不能执行 Windows-only Process Fixture，记录为 **199/221 PASS** 和 22 个 Environment/Platform Failure。没有复制受保护资源，没有弱化 Historical Test，也没有把失败 Case 改成 Skip。

随后在 Private Windows Acceptance Environment 中执行同一范围。规定的 221 项基线全部通过；由于受保护 Fixture 初始化成功后额外 3 个子测试可以完成注册，完整发现结果为 **224/224 PASS**、0 Fail、0 Skip。两项 Windows-native Process Fixture 均在独立执行和完整 Suite 中通过。这证明此前 22 个失败均为 Environment/Platform-specific。

### 最终执行结果

实现包含 Trade Service、State Transition、Revision-bound Confirmation、Persistent Lock、Partial-stack Handling、Capacity Validation、Atomic Two-character Settlement、Unique Settlement Result、Cancel、Expire、Disconnect Cancellation、Restart Recovery、Audit Event、PostgreSQL Migration、真实 Transaction Failure Injection 和 G8 Mutation Gate。

Windows Acceptance Run 已通过 G10 Trade **29/29**、Go Full Regression **290/290**（0 Skip）和 Windows amd64 Build。真实 PostgreSQL Restart、Duplicate Finalize、Unique Lock 和 Rollback Scenario 均通过。Existing Tests Removed 为 0，没有通过 Skip 测试来获得 PASS。本轮 Windows Closure 没有修改 Production Code 或 Test Logic。

### 阶段关闭决定

用户在 Review Technical Acceptance Result 后，已批准最终 G10 Commit、Push、Build in Public Closure 和 Issue #2 Closure。X 仍未发布，G11 尚未开始。
