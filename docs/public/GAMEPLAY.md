# Gameplay / 游戏玩法

## English — Primary

This page separates accepted runtime behavior from product direction.

| Area | Intended experience | Current status |
|---|---|---|
| World exploration | Move through maps, discover locations and content | **FOUNDATION COMPLETE** for two accepted technical slices |
| Classes | Distinct roles, skills, and equipment paths | **Warrior FOUNDATION ONLY**; other classes are not confirmed as Web-playable |
| Combat | Server-authoritative targeting, attacks, damage and death | **FOUNDATION COMPLETE** |
| Skills | Class-specific active and passive abilities | **FOUNDATION COMPLETE** for Warrior 烈火剑法 only; broader system planned |
| Monsters | Server-owned AI, chase, attack, return and death | **FOUNDATION COMPLETE** for the accepted slice |
| Bosses | Group challenges and progression encounters | **PLANNED** |
| Loot | Server-owned drops, pickup and stable item identity | **FOUNDATION COMPLETE** |
| Equipment | Equip/unequip and layered runtime stats | **FOUNDATION COMPLETE** |
| Inventory | Capacity, stable slots and ownership validation | **FOUNDATION COMPLETE** |
| Player trading | Secure ownership and FB exchange | **FOUNDATION COMPLETE** for atomic item + FB settlement; formal UI and markets not implemented |
| Party / team dungeons | Cooperative play | **PLANNED** |
| Guild | Membership, roles, missions and reputation | **PLANNED** |
| Siege | Competitive territory and prestige | **PLANNED** |
| Mining | Validated power sharing a capped global emission pool | **PLANNED** |
| Ordinals activation | Verify ownership and metadata, then map eligible assets into game rules | **PLANNED** |
| Social world | Profiles, follows, feeds, chat, voice, video and social transfers | **PLANNED / RESEARCH** |

### Accepted combat slice

The browser submits target, movement, normal-attack, skill, pickup, equip, unequip, and trade intents. The server owns legality, cooldowns, RNG, HP/MP mutation, AI, item identity, ownership, locks, and atomic settlement. G4–G8 prove this boundary for the accepted Warrior slice.

### Character growth

The current Character Aggregate can preserve identity, world position, HP/MP, inventory item instances, equipment relations, and learned skills. RuntimeStats are recalculated from base state and equipment. A full level/EXP progression product, multiple complete classes, broad skill trees, crafting, durability, enhancement, and endgame loops are not yet implemented.

### Bosses, PvP, guilds and siege

These are intended parts of the MMORPG, but they have no accepted production implementation. The current combat sandbox must not be described as a shipped Boss, PvP, Guild War, or Siege system.

### Trade

G10 provides a transport-independent item Trade state machine with revision-bound confirmation, persistent item locks, ownership validation, atomic two-character settlement, idempotent finalization, concurrent item protection, and restart recovery. G12 extends it with revision-bound FB offers and atomic item + FB settlement in PostgreSQL at 0% player-trade fee. A formal Trade UI, Marketplace, and Auction House are not implemented.

---

## 中文 — 完整对应版本

本页明确区分已验收 Runtime Behavior 与产品方向。

| 领域 | 目标体验 | 当前状态 |
|---|---|---|
| World Exploration | 在地图中移动并发现地点与内容 | 两个已验收技术切片达到 **FOUNDATION COMPLETE** |
| Classes | 不同定位、Skill 与 Equipment 成长路线 | 只有 **Warrior FOUNDATION**；其他职业尚未确认为 Web 可玩 |
| Combat | Server-authoritative Targeting、Attack、Damage 与 Death | **FOUNDATION COMPLETE** |
| Skills | 职业 Active / Passive Ability | 仅 Warrior 烈火剑法达到 **FOUNDATION COMPLETE**；更完整系统仍在规划 |
| Monsters | Server-owned AI、Chase、Attack、Return 与 Death | 已验收切片达到 **FOUNDATION COMPLETE** |
| Bosses | 团队挑战与成长遭遇 | **PLANNED** |
| Loot | Server-owned Drop、Pickup 与稳定 Item Identity | **FOUNDATION COMPLETE** |
| Equipment | Equip / Unequip 与分层 Runtime Stats | **FOUNDATION COMPLETE** |
| Inventory | Capacity、稳定 Slot 与 Ownership Validation | **FOUNDATION COMPLETE** |
| Player Trading | 安全 Ownership 与 FB Exchange | 原子 Item + FB Settlement 达到 **FOUNDATION COMPLETE**；正式 UI 与 Market 未实现 |
| Party / Team Dungeon | 协作玩法 | **PLANNED** |
| Guild | Membership、Role、Mission 与 Reputation | **PLANNED** |
| Siege | Territory 与 Prestige 竞争 | **PLANNED** |
| Mining | 经过验证的 Power 共享有上限的 Global Emission Pool | **PLANNED** |
| Ordinals Activation | 验证 Ownership 与 Metadata，再按 Game Rule 映射符合条件的资产 | **PLANNED** |
| Social World | Profile、Follow、Feed、Chat、Voice、Video 与 Social Transfer | **PLANNED / RESEARCH** |

### 已验收 Combat Slice

Browser 提交 Target、Movement、Normal Attack、Skill、Pickup、Equip、Unequip 和 Trade Intent。Server 负责 Legality、Cooldown、RNG、HP/MP Mutation、AI、Item Identity、Ownership、Lock 与 Atomic Settlement。G4–G8 在已验收 Warrior Slice 中证明了这一边界。

### Character Growth

当前 Character Aggregate 可以保存 Identity、World Position、HP/MP、Inventory ItemInstance、Equipment Relation 和 Learned Skill。RuntimeStats 会根据 Base State 与 Equipment 重新计算。完整 Level/EXP 产品、多套完整职业、广泛 Skill Tree、Crafting、Durability、Enhancement 和 Endgame Loop 尚未实现。

### Boss、PvP、Guild 与 Siege

这些是 MMORPG 的目标组成部分，但还没有已验收的 Production Implementation。当前 Combat Sandbox 不能描述为已上线的 Boss、PvP、Guild War 或 Siege System。

### Trade

G10 提供独立于 Transport 的 Item Trade State Machine，包含 Revision-bound Confirmation、Persistent Item Lock、Ownership Validation、双方 Character Atomic Settlement、Idempotent Finalization、Concurrent Item Protection 与 Restart Recovery。G12 在此基础上新增 Revision-bound FB Offer 与 PostgreSQL 原子 Item + FB Settlement，玩家交易手续费为 0%。正式 Trade UI、Marketplace 和 Auction House 尚未实现。
