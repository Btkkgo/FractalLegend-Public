# G9 — Persistence Foundation

Status: **G9 PASS** — manually accepted.

## English — Primary

## Accepted development result

G9 adds the first durable Character Aggregate to the Go Game Server. A character can be created, saved, loaded after session close, and loaded after a complete Game Server process restart from PostgreSQL. The accepted G1–G8 runtime rules remain the authority for movement, combat, skills, loot, inventory, equipment, and stat calculation.

## Why

Through G8, a process exit discarded the test character, world position, inventory, equipment, and learned skill state. G9 gives those canonical values a transactional storage boundary so the runtime can recover the last successful save without turning transient combat state or derived statistics into database authority.

## Architecture and PostgreSQL boundary

The dependency direction is:

`Game Server runtime → CharacterRepository → PostgreSQL adapter`

The domain-facing repository types contain no SQL or PostgreSQL types. `pgx/v5` implements the adapter with a bounded connection pool, context deadlines, transactions, redacted connection errors, and optimistic revisions. `FRACTAL_DATABASE_URL` is read only at process startup. No credential or database address is stored in the repository.

Persistence is optional at boot so the accepted G8 standalone mode remains available without a database. When a database URL is supplied, connection and migration failures stop startup. A failed load or save returns a bounded persistence error; the client is never told that an unsaved change succeeded.

## Versioned schema and migration

`0001_initial_persistence.sql` creates:

- `accounts`
- `characters`
- `character_world_state`
- `character_inventory_items`
- `character_equipment`
- `character_skills`

The migration runner maintains `schema_migrations`. Each unapplied migration and its version record commit in one transaction. A fresh database, a repeat run, recognition of an applied version, and rollback of a deliberately broken migration were verified against PostgreSQL 16.13.

## Character Aggregate

One repository operation loads or saves the complete consistency boundary:

- Account relationship and character identity
- Name, class, level, EXP, current HP, and current MP
- Map ID and X/Y world position
- Inventory item instances
- Equipment slot to item-instance relationships
- Learned skill and skill level

Inventory, equipment, skills, world state, character fields, and revision commit together. A child-row constraint failure rolls back the character update and revision. A partial aggregate is rejected instead of creating a damaged runtime character.

## Inventory and equipment persistence

The database preserves each `ItemInstanceID`. Equipment references the same item row held by the character aggregate; it does not create a second copy. Validation and schema constraints require one legal location, `INVENTORY` or `EQUIPMENT`, and require equipment location and slot relationships to agree.

Equip and Unequip first apply the existing G8 server rule, then save the full aggregate. A failed save restores the prior in-memory item location, slot relationship, revisions, and RuntimeStats. Pickup uses the same fail-closed pattern, retaining the ground item and prior inventory when persistence fails. Tests cover duplication, loss, Equip, Unequip, injected failure rollback, stale writers, and concurrent save conflict.

## Skill persistence

G9 stores learned skill identity and skill level. Cast state, projectile state, animation state, and cooldown timestamps remain transient because G6 defines no cross-restart cooldown rule. Unknown skill references fail the complete load.

## RuntimeStats recalculation

Derived `RuntimeStats` are absent from the schema. Restore loads base character state and equipment, then runs the accepted G8 `CharacterStatsService`:

`Persistent base state + Persistent equipment + Canonical rules → RuntimeStats`

The real restart test restored 玛法屠龙 and recalculated Attack 18–22 from Base Attack 8–12 plus the accepted weapon modifier. Post-load normal attack and 烈火剑法 both used the recalculated values.

## World safe restore

Stored Map ID and X/Y are checked against the existing map registry and walkability rules. A missing map or invalid coordinate falls back to the existing safe spawn. G9 adds no new map or movement rule. Player movement does not issue a SQL update per packet; position is flushed at controlled boundaries such as logout and server shutdown.

## Reconnect and server restart

The integration path verifies character creation, real G7 觉醒石 pickup identity, G8 玛法屠龙 equipment, learned skill state, world coordinates, logout save, reconnect/load, and full process restart. Item instance IDs remain stable. A separate three-server PostgreSQL test verifies shutdown flush by changing position in the second process and loading it in a third process.

## Verification

- PostgreSQL: 16.13 via a dedicated local database containing synthetic test data only.
- Driver: `github.com/jackc/pgx/v5` 5.11.0.
- Persistence requirement matrix: **25/25 PASS**.
- Node regression: **222/222 PASS** on Windows.
- Go regression: **261/261 PASS** with real PostgreSQL integration enabled.
- Combined regression: **483/483 PASS**.
- Go race with real PostgreSQL integration: PASS.
- TypeScript typecheck and Web production build: PASS on macOS and Windows.
- Go server builds: Windows amd64, Linux amd64, and macOS arm64 PASS.
- Standalone: G9 PostgreSQL restart recovery PASS on macOS; existing database-free G8 mode PASS on macOS and Windows.

## Known limitations and not implemented

G9 is the persistence foundation for one accepted test-character slice. It does not add password authentication, OAuth, wallet identity, a database administration UI, continuous per-move saves, event sourcing, or production deployment automation.

Trade, Marketplace, Warehouse, Shared Account Storage, Durability, Repair, Enhancement, Socket, Gem, Affix, Crafting, Currency, Economy, FB, Contribution, Mining, Wallet, Ordinals, Guild Persistence, Quest Persistence, Mail, Auction, Admin, and Social persistence are not implemented.

## Next

G9 stops at the reviewed persistence boundary. Any later phase must begin only after manual acceptance; this work does not start G10.

---

# G9 — 持久化基础（完整中文版本）

状态：**G9 PASS** — 已通过人工验收。

## 已验收的开发结果

G9 为 Go Game Server 增加了第一套可持久保存的 Character Aggregate。角色可以被创建、保存、在 Session 关闭后重新加载，并可以在 Game Server 进程完全重启后从 PostgreSQL 恢复。已验收的 G1–G8 Runtime 规则继续作为移动、战斗、技能、掉落、背包、装备和属性计算的权威规则。

## 为什么实施 G9

截至 G8，进程退出会丢失测试角色、世界位置、背包、装备和已学习技能状态。G9 为这些 Canonical 值建立事务存储边界，使 Runtime 能恢复最后一次成功保存的状态，同时不把瞬时战斗状态或 Derived RuntimeStats 变成数据库权威数据。

## Architecture 与 PostgreSQL 边界

依赖方向为：

`Game Server runtime → CharacterRepository → PostgreSQL adapter`

面向 Domain 的 Repository 类型不包含 SQL 或 PostgreSQL 类型。`pgx/v5` Adapter 使用有界连接池、Context Deadline、Transaction、脱敏连接错误和 Optimistic Revision。`FRACTAL_DATABASE_URL` 仅在进程启动时读取。仓库中不保存凭据或数据库地址。

Persistence 在启动时是可选配置，因此没有数据库时仍可使用已验收的 G8 standalone 模式。提供数据库 URL 后，连接或 Migration 失败会阻止服务器启动。Load 或 Save 失败会返回有界的 Persistence Error；客户端不会收到未保存修改已经成功的错误结果。

## Versioned Schema 与 Migration

`0001_initial_persistence.sql` 创建：

- `accounts`
- `characters`
- `character_world_state`
- `character_inventory_items`
- `character_equipment`
- `character_skills`

Migration Runner 维护 `schema_migrations`。每个尚未应用的 Migration 及其版本记录在同一个 Transaction 中提交。全新数据库、重复执行、已应用版本识别和故意损坏 Migration 的回滚都已在 PostgreSQL 16.13 上验证。

## Character Aggregate

一次 Repository 操作加载或保存完整的一致性边界：

- Account 关系与 Character 身份
- Name、Class、Level、EXP、当前 HP 和当前 MP
- Map ID 与 X/Y 世界位置
- Inventory ItemInstance
- Equipment Slot 与 ItemInstance 的关系
- Learned Skill 与 Skill Level

Inventory、Equipment、Skills、World State、Character 字段和 Revision 一起提交。子表 Constraint 失败会回滚 Character 更新和 Revision。系统会拒绝 Partial Aggregate，而不会据此创建损坏的 Runtime Character。

## Inventory 与 Equipment Persistence

数据库保留每个 `ItemInstanceID`。Equipment 引用 Character Aggregate 持有的同一条 Item 记录，不创建第二份副本。验证逻辑和 Schema Constraint 要求物品只能处于 `INVENTORY` 或 `EQUIPMENT` 中的一个合法位置，并要求 Equipment Location 与 Slot 关系一致。

Equip 和 Unequip 先应用现有 G8 Server Rule，再保存完整 Aggregate。保存失败时会恢复此前的内存 Item Location、Slot 关系、Revision 和 RuntimeStats。Pickup 使用相同的 Fail-closed 方式；Persistence 失败时保留 Ground Item 和此前的 Inventory。测试覆盖 Duplication、Loss、Equip、Unequip、注入失败回滚、Stale Writer 和 Concurrent Save Conflict。

## Skill Persistence

G9 保存 Learned Skill Identity 与 Skill Level。由于 G6 没有定义跨重启保留 Cooldown 的规则，Cast State、Projectile State、Animation State 和 Cooldown Timestamp 保持为瞬时状态。未知 Skill Reference 会使完整加载失败。

## RuntimeStats 重新计算

Schema 不保存 Derived `RuntimeStats`。恢复时加载 Character Base State 和 Equipment，然后执行已验收的 G8 `CharacterStatsService`：

`Persistent base state + Persistent equipment + Canonical rules → RuntimeStats`

真实重启测试恢复了玛法屠龙，并通过 Base Attack 8–12 与已验收的 Weapon Modifier 重新计算出 Attack 18–22。加载后的普通攻击和烈火剑法均使用重新计算的数值。

## World 安全恢复

存储的 Map ID 和 X/Y 会依据现有 Map Registry 与 Walkability Rule 进行检查。Map 不存在或坐标无效时，系统会回退到现有 Safe Spawn。G9 没有新增 Map 或 Movement Rule。Player Move 不会对每个坐标包执行 SQL Update；位置会在 Logout、Server Shutdown 等受控边界执行 Flush。

## Reconnect 与 Server Restart

Integration Path 验证了 Character Creation、真实 G7 觉醒石 Pickup Identity、G8 玛法屠龙 Equipment、Learned Skill State、World Coordinate、Logout Save、Reconnect/Load 和完整进程重启。ItemInstance ID 保持稳定。另一项三服务器 PostgreSQL 测试会在第二个进程修改位置，并在第三个进程加载该位置，以验证 Shutdown Flush。

## 验证结果

- PostgreSQL：16.13；使用仅含 Synthetic Test Data 的专用本地数据库。
- Driver：`github.com/jackc/pgx/v5` 5.11.0。
- Persistence Requirement Matrix：**25/25 PASS**。
- Node Regression：Windows 上 **222/222 PASS**。
- Go Regression：启用真实 PostgreSQL Integration 时 **261/261 PASS**。
- Combined Regression：**483/483 PASS**。
- 启用真实 PostgreSQL Integration 的 Go Race：PASS。
- TypeScript Typecheck 与 Web Production Build：macOS 和 Windows 均 PASS。
- Go Server Build：Windows amd64、Linux amd64 和 macOS arm64 均 PASS。
- Standalone：macOS 上 G9 PostgreSQL Restart Recovery PASS；macOS 和 Windows 上既有无数据库 G8 模式 PASS。

## 已知限制与未实现内容

G9 是一个已验收测试角色切片的 Persistence Foundation。它没有增加 Password Authentication、OAuth、Wallet Identity、Database Administration UI、每次移动连续保存、Event Sourcing 或 Production Deployment Automation。

Trade、Marketplace、Warehouse、Shared Account Storage、Durability、Repair、Enhancement、Socket、Gem、Affix、Crafting、Currency、Economy、FB、Contribution、Mining、Wallet、Ordinals、Guild Persistence、Quest Persistence、Mail、Auction、Admin 和 Social Persistence 均未实现。

## 下一步

G9 在已审核的 Persistence Boundary 停止。后续阶段只能在人工验收后开始；本工作没有开始 G10。
