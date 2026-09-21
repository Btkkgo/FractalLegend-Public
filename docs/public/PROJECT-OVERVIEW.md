# Project Overview / 项目概览

## English — Primary

### What is Fractal Legend?

Fractal Legend is a browser-native Legend-style MMORPG under active development for the Fractal / Bitcoin ecosystem. Its technical direction is a thin browser client connected through a protocol boundary to a server-authoritative, cross-platform Game Server and durable PostgreSQL state.

### Why does it exist?

The project explores how classic MMORPG combat, meaningful player ownership, auditable economies, and portable digital identity can coexist without making the browser, wallet, or asset metadata authoritative over game rules.

### What do players do?

The intended loop is to explore worlds, fight monsters and bosses, grow a character, learn skills, collect and equip items, trade safely, cooperate in parties and guilds, compete in siege, mine resources, activate eligible digital assets, and build a persistent social identity.

The accepted implementation currently covers a smaller technical slice: movement, rendering, living entities, combat, basic monster AI, one Warrior skill, loot, inventory, equipment, runtime stats, Character Aggregate persistence, and an item-only Trade Foundation.

### What makes it different?

1. **Server authority:** the browser sends intent; the Game Server owns validation, RNG, damage, state, ownership, and settlement.
2. **Migration with provenance:** verified Legacy identities may be preserved while uncertain behavior is labeled as a Fractal override.
3. **Economic invariants first:** ownership, locks, revision checks, atomicity, idempotency, and restart recovery precede public trading interfaces.
4. **Digital ownership with balance:** future Ordinals can contribute provenance, identity, appearance, eligibility, and collectibility without automatically granting unlimited power.
5. **Build in Public:** accepted results, failures, boundaries, tests, and retrospectives are documented.

### Why Fractal / Bitcoin?

The long-term goal is to connect game identity and selected assets to a Bitcoin-aligned ecosystem while retaining explicit game rules and server-side verification. Fractal is intended to provide an environment for higher-throughput game interactions; Bitcoin provides the broader ownership and provenance context.

This is product direction. Mainnet integration, wallets, deposits, withdrawals, confirmations, reorg handling, and production Ordinals activation are not implemented.

### Where is development today?

G1–G10 are accepted foundations from the private canonical development line and predate this mirror. G11 added the internal FB Ledger Foundation; G12 added atomic item + FB direct player Trade settlement and passed technical acceptance. Neither milestone is a public game release. No public launch date has been announced.

See [Current Status](CURRENT-STATUS.md), [Development Timeline](DEVELOPMENT-TIMELINE.md), and [Roadmap](ROADMAP.md).

---

## 中文 — 完整对应版本

### Fractal Legend 是什么？

Fractal Legend / 分形传奇是一款正在为 Fractal / Bitcoin 生态开发的浏览器原生传奇风格 MMORPG。技术方向是：由轻量 Browser Client 通过 Protocol Boundary 连接 Server-authoritative、跨平台 Game Server，并使用 PostgreSQL 保存持久状态。

### 为什么要做这个项目？

项目探索经典 MMORPG 战斗、有意义的玩家所有权、可审计经济和可携带数字身份如何共同存在，同时不让 Browser、Wallet 或 Asset Metadata 越过 Game Rule 的权威边界。

### 玩家要做什么？

目标循环包括探索世界、挑战怪物与 Boss、培养角色、学习技能、收集并装备物品、安全交易、参与 Party 与 Guild 协作、竞争 Siege、开采资源、激活符合条件的 Digital Asset，以及建立持久 Social Identity。

当前已验收实现只覆盖较小的技术切片：Movement、Rendering、Living Entity、Combat、Basic Monster AI、一项 Warrior Skill、Loot、Inventory、Equipment、Runtime Stats、Character Aggregate Persistence，以及仅支持 Item 的 Trade Foundation。

### 项目有什么不同？

1. **Server Authority：**Browser 发送 Intent；Game Server 负责 Validation、RNG、Damage、State、Ownership 和 Settlement。
2. **带 Provenance 的迁移：**可以保留已验证的 Legacy Identity；不确定的行为必须标记为 Fractal Override。
3. **先建立经济不变量：**在公开交易界面之前，先建立 Ownership、Lock、Revision Check、Atomicity、Idempotency 和 Restart Recovery。
4. **兼顾 Balance 的 Digital Ownership：**未来 Ordinals 可以提供 Provenance、Identity、Appearance、Eligibility 和 Collectibility，但不会自动获得无限制战力。
5. **Build in Public：**公开记录已验收结果、失败、边界、测试和 Retrospective。

### 为什么选择 Fractal / Bitcoin？

长期目标是在保留明确 Game Rule 与服务器验证的前提下，把 Game Identity 和部分资产接入 Bitcoin-aligned 生态。Fractal 计划为更高吞吐的游戏交互提供环境；Bitcoin 提供更广泛的 Ownership 与 Provenance 背景。

以上属于产品方向。Mainnet Integration、Wallet、Deposit、Withdrawal、Confirmation、Reorg Handling 和 Production Ordinals Activation 均未实现。

### 当前开发到哪里？

G1–G10 是 Private Canonical Development Line 上的已验收 Foundation，且早于本镜像建立。G11 新增内部 FB Ledger Foundation；G12 新增原子 Item + FB Direct Player Trade Settlement，并通过技术验收。这两个阶段都不代表游戏公开 Release。项目尚未公布公开上线日期。

详见[当前状态](CURRENT-STATUS.md)、[开发时间线](DEVELOPMENT-TIMELINE.md)和[路线图](ROADMAP.md)。
