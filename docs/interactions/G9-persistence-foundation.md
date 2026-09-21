# G9 Persistence Foundation — Public Engineering Decision Record

Status: G9 manually accepted — **PASS**

Build in Public tracking: private canonical archive Issue #1 (retrospective; not publicly accessible).

## English — Primary

## Why persistence was the G9 priority

G1–G8 established an authoritative playable runtime: world state, rendering, living entities, combat, AI, skills, loot, inventory, equipment, and calculated character stats. The next architectural risk was durability. Without a persistence boundary, a Game Server process exit discarded the accepted character state. G9 therefore focused on preserving that state before adding another gameplay system.

## Why PostgreSQL

The Character Aggregate needs transactions, referential constraints, concurrent writer protection, and predictable recovery after process restart. PostgreSQL provides those guarantees directly and works across the Windows, Linux, and macOS server targets. The verified G9 environment used PostgreSQL 16.13 with dedicated synthetic test databases.

## Why pgx/v5

The Go Game Server needs a direct, maintained PostgreSQL driver with pooling, context support, transactions, and PostgreSQL-native error handling. `pgx/v5` supplies that boundary without introducing an application framework or changing the runtime domain model.

## Why no large ORM

G9 has one deliberate aggregate and one versioned schema. Direct SQL keeps transaction order, revision checks, constraints, and migration behavior visible in code and tests. A large ORM would add mapping and lifecycle behavior that this phase does not need.

## Why one Character Aggregate

Character identity, Inventory, Equipment, learned Skills, and World State must describe one internally consistent player. Saving them independently could produce half-saved states such as a moved item with an old equipment slot or a new character revision with missing inventory rows. The repository therefore loads and saves the complete aggregate in one transaction.

## Why Derived RuntimeStats are recalculated

RuntimeStats are the result of base character state, equipped items, and accepted game rules. Persisting the final calculated values would create drift when equipment or stat rules change. G9 stores the canonical inputs and runs the G8 `CharacterStatsService` after load. Normal attacks and 烈火剑法 continue to consume the recalculated runtime values.

## Why the Item Ownership Invariant is required

Every independent item has a stable `ItemInstanceID`. That instance can be in Inventory or Equipment, never both. Repository validation and database constraints enforce the owner, location, equipment slot, and item-instance relationship. This prevents duplication and loss during Equip, Unequip, save failure, reconnect, and restart.

## Why Optimistic Revision is required

Two sessions may load the same character revision. If both could save unconditionally, the older session could overwrite a newer committed state. Every successful save advances a revision; a writer presenting a stale revision receives a conflict instead of silently winning.

## Why real PostgreSQL integration is required

An in-memory repository can verify domain contracts but cannot prove SQL constraints, transaction rollback, migration behavior, connection failure handling, or actual restart durability. G9 therefore uses real PostgreSQL integration tests as a release gate. Memory-backed tests remain useful for fast unit coverage but are not the final acceptance evidence.

## Why a full Game Server restart is the G9 gate

Reconnect inside one process can reuse memory and does not prove durability. The acceptance path starts a server, writes legal state, stops the process, starts a new server, and loads the state from PostgreSQL. A third server start verifies the controlled shutdown position flush. This is the proof that the process is no longer the sole owner of durable character state.

## Build in Public workflow adoption

Fractal Legend adopted the Build in Public stage-tracking workflow while G9 was already being completed. Private canonical archive Issue #1 was therefore created retrospectively after implementation and manual acceptance. It is explicitly a historical record and must not be presented as a pre-development issue. G1–G8 historical indexing is outside this closeout.

## Public boundary

The public G9 material contains source code, schema, migrations, synthetic tests, engineering documentation, and a sanitized verification screenshot. It contains no credentials, database address, real account or player data, database dump, private machine path, protected Legacy asset, original map, or private Canonical export.

## Scope boundary

Trade, Warehouse, Durability, Economy, Wallet, Ordinals, Mining, and G10 are not implemented or started by this record.

---

# G9 Persistence Foundation — 公开工程决策记录（完整中文版本）

状态：G9 已通过人工验收 — **PASS**

Build in Public 跟踪：Private Canonical Archive Issue #1（Retrospective，不可公开访问）。

## 为什么 G9 优先实施 Persistence

G1–G8 已建立一套 Server-authoritative 可运行 Runtime，包括 World State、Rendering、Living Entities、Combat、AI、Skills、Loot、Inventory、Equipment 和计算后的 Character Stats。下一个 Architecture 风险是 Durability。没有 Persistence Boundary 时，Game Server 进程退出会丢失已验收的 Character State。因此，G9 在增加下一项 Gameplay System 之前，优先保存这些状态。

## 为什么采用 PostgreSQL

Character Aggregate 需要 Transaction、Referential Constraint、Concurrent Writer Protection，以及进程重启后的可预测恢复。PostgreSQL 直接提供这些保证，并适用于 Windows、Linux 和 macOS Server Target。已验证的 G9 环境使用 PostgreSQL 16.13 和只含 Synthetic Test Data 的专用测试数据库。

## 为什么采用 pgx/v5

Go Game Server 需要一个直接、持续维护的 PostgreSQL Driver，并需要 Pooling、Context、Transaction 和 PostgreSQL-native Error Handling。`pgx/v5` 提供了该边界，同时不引入 Application Framework，也不改变 Runtime Domain Model。

## 为什么不采用大型 ORM

G9 只有一个经过明确设计的 Aggregate 和一套 Versioned Schema。Direct SQL 让 Transaction Order、Revision Check、Constraint 和 Migration Behavior 在代码及测试中保持可见。大型 ORM 会增加本阶段不需要的 Mapping 与 Lifecycle Behavior。

## 为什么使用一个 Character Aggregate

Character Identity、Inventory、Equipment、Learned Skills 和 World State 必须共同描述一个内部一致的 Player。分别保存可能产生半保存状态，例如物品已经移动但 Equipment Slot 仍是旧值，或 Character Revision 已更新但 Inventory Row 缺失。因此，Repository 在一个 Transaction 中加载和保存完整 Aggregate。

## 为什么重新计算 Derived RuntimeStats

RuntimeStats 是 Base Character State、Equipped Item 和已验收 Game Rule 的计算结果。直接保存最终计算值会在 Equipment 或 Stat Rule 变化时产生 Drift。G9 保存 Canonical Input，并在加载后运行 G8 `CharacterStatsService`。普通攻击和烈火剑法继续使用重新计算的 Runtime Value。

## 为什么需要 Item Ownership Invariant

每个独立物品都有稳定的 `ItemInstanceID`。该实例可以位于 Inventory 或 Equipment，但不能同时位于两者。Repository Validation 与 Database Constraint 强制执行 Owner、Location、Equipment Slot 和 ItemInstance Relationship。这可防止 Equip、Unequip、Save Failure、Reconnect 和 Restart 期间出现 Duplication 或 Loss。

## 为什么需要 Optimistic Revision

两个 Session 可能加载同一 Character Revision。如果两者都能无条件保存，旧 Session 可能覆盖更新的已提交状态。每次成功保存都会推进 Revision；提交 Stale Revision 的 Writer 会收到 Conflict，而不会静默覆盖。

## 为什么必须使用真实 PostgreSQL Integration

In-memory Repository 可以验证 Domain Contract，但不能证明 SQL Constraint、Transaction Rollback、Migration Behavior、Connection Failure Handling 或真实 Restart Durability。因此，G9 把真实 PostgreSQL Integration Test 作为 Release Gate。Memory-backed Test 仍适合快速 Unit Coverage，但不能作为最终验收证据。

## 为什么完整 Game Server Restart 是 G9 Gate

同一进程内的 Reconnect 可以复用内存，不能证明 Durability。验收路径会启动 Server、写入合法状态、完全停止进程、启动新的 Server，并从 PostgreSQL 加载状态。第三次 Server 启动会验证受控的 Shutdown Position Flush。这证明进程不再是 Durable Character State 的唯一持有者。

## Build in Public 工作流采用过程

Fractal Legend 在 G9 已接近完成时采用 Build in Public 阶段跟踪工作流。因此，Private Canonical Archive Issue #1 是在实现与人工验收之后补建的 retrospective / historical Issue。它明确属于历史记录，不得描述成 Development 前已建立的 Issue。G1–G8 Historical Indexing 不属于本次收尾范围。

## 公开边界

公开 G9 内容包含 Source Code、Schema、Migration、Synthetic Test、Engineering Documentation 和经过脱敏的 Verification Screenshot。其中不包含 Credential、Database Address、真实 Account 或 Player Data、Database Dump、Private Machine Path、受保护 Legacy Asset、Original Map 或 Private Canonical Export。

## 范围边界

本记录没有实现或开始 Trade、Warehouse、Durability、Economy、Wallet、Ordinals、Mining 或 G10。
