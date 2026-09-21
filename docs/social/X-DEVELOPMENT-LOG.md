# Fractal Legend G1–G13 — X Development Log Drafts

Status: milestone posts below are drafts only and have not been published by Codex.

Publication record: the first Fractal Legend project-introduction post was **PUBLISHED_MANUALLY** by the user on **2026-09-21**. URL: **not recorded**. Its approved text is not rewritten here.

状态：以下阶段内容均为草稿，Codex 没有发布任何内容。

发布记录：首条 Fractal Legend 项目介绍已由用户在 **2026-09-21** **PUBLISHED_MANUALLY**。URL：**not recorded**。此处不重写其已批准正文。

## G1 — First playable world

> Fractal Legend Web reached its first server-authoritative playable world. Browser movement is intent-only; the Go Game Server validates position, speed, bounds, and collision against a frozen content slice. The first map uses a reviewed Fractal visual/collision projection. #gamedev #golang #webgame

Screenshot suggestion: the full playfield with the movement controls and authoritative X/Y, collision revision, and world instance visible.

## G2 — Rendering pipeline

> G2 moved the playable slice into a browser-ready 2D rendering pipeline: a verified map parser, deterministic atlas generation, and packaged Web assets. The current visuals are clearly labeled Fractal Override; original Legacy art decoding is not claimed. #gamedev #phaser #webgl

Screenshot suggestion: the generated tile atlas beside the rendered map, with `FRACTAL_OVERRIDE` visible in the caption.

## G3 — Living world

> 天罚厅 is alive in G3: one NPC, six server-owned monster entities, authoritative snapshots, and synchronized browser rendering. NPC/monster placement comes from preserved content; player spawn and temporary sprites remain explicit overrides. #gamedev #multiplayer #golang

Screenshot suggestion: the full room with the player, NPC, and several colored monster silhouettes plus the entity counters.

## G4 — Combat sandbox

> G4 added server-authoritative targeting, normal attacks, damage, and death. The browser sends intent; the Game Server owns range checks, cooldowns, RNG, defense, HP, and the final combat event. #gamedev #serverauthoritative #golang

Screenshot suggestion: a selected monster immediately after a server damage event, with target HP and last damage visible.

## G5 — Monster AI

> G5 added basic monster AI and counterattacks with a deterministic server tick: IDLE, AGGRO, CHASE, ATTACK, RETURN, and DEAD. Movement respects the collision layer and leash rules. This is Fractal AI v1, not a claim of recovered Legacy behavior. #gamedev #gameai

Screenshot suggestion: a monster in CHASE or ATTACK with AI state, aggro target, distance, and leash distance visible.

## G6 — Skills

> G6 brought 烈火剑法 into the runtime with its real Legacy identity and a versioned server-authoritative skill rule. MP cost, cooldown, range, damage, death, retries, and reconnect state are all owned by the Game Server. #gamedev #rpg #golang

Screenshot suggestion: the skill bar and a `skill.cast.result` state with MP, cooldown, target HP, and skill source visible.

## G7 — Loot and inventory

> G7 completed the first ownership loop: monster death → ground drop → manual pickup → authoritative inventory. 觉醒石 keeps its real Legacy identity, while pickup remains atomic under concurrency. Auto pickup, trade, and warehouse are still off. #gamedev #inventory #multiplayer

Screenshot suggestion: a ground item before pickup and a second frame showing the same named item in the inventory.

## G8 — Equipment and stats

> G8 is accepted: 玛法屠龙 now moves from inventory to a real WEAPON slot, and the server recalculates Base Stats + Equipment Modifiers = Runtime Stats. Normal attacks and 烈火剑法 use the new attack values. Reconnect, 100-way equip contention, 1,000 equip cycles, race checks, cross-platform builds, and 456/456 tests passed. Trade, warehouse, and durability remain off. #gamedev #rpg #golang

Screenshot suggestion: the G8 equipment panel after equip, showing 玛法屠龙, ATK +10/+10, Base Attack 8–12, Runtime Attack 18–22, and `Game Server READY`.

## G9 — Persistence foundation

### A. English Primary Post

> Fractal Legend now has its first real persistence foundation. Characters, inventory, equipment, learned skills, and safe world position can survive disconnects and full Game Server restarts through PostgreSQL-backed transactional persistence. Stable item instances and optimistic revisions protect ownership and concurrent saves, while RuntimeStats are recalculated from canonical rules after load. Persistence checks passed 25/25, the complete suite passed 483/483, and Game Server builds passed on Windows, Linux, and macOS. Trade, warehouse, durability, and economy remain outside this phase. #gamedev #golang #postgresql

### B. 中文完整对照稿

> Fractal Legend 现已建立第一套真实持久化基础。角色、背包、装备、已学习技能和安全世界位置可以通过 PostgreSQL 事务持久化，在断线和完整 Game Server 重启后恢复。稳定的物品实例和乐观 revision 保护物品归属及并发保存；角色加载后，RuntimeStats 会根据 Canonical 规则重新计算。持久化验收 25/25 通过，完整测试 483/483 通过，Game Server 在 Windows、Linux 和 macOS 上构建通过。Trade、Warehouse、Durability 和 Economy 仍不属于本阶段。

Screenshot suggestion: the sanitized G9 verification card showing PostgreSQL migration, full server restart recovery, stable item identity, RuntimeStats recalculation, and regression totals.

截图建议：展示经过脱敏的 G9 Verification Card，内容包括 PostgreSQL Migration、完整 Server Restart Recovery、稳定 Item Identity、RuntimeStats Recalculation 和 Regression Total。

## G10 — Trade foundation

### A. English Primary Post

> Fractal Legend’s player trade foundation starts below the UI: persistent item locks, offer revisions, revision-bound confirmation, atomic two-character settlement, idempotent retries, and restart recovery. Windows validation passed the complete Historical Node scope at 224/224, G10 Trade at 29/29, Go regression at 290/290 with 0 skips, both native process fixtures, and the Windows amd64 build. The earlier 22 macOS environment/platform failures were closed without weakening tests. G10 completed technical acceptance and Build in Public closure; it is not released. FB settlement, markets, auctions, wallets, blockchain, fees, and Trade UI remain outside this phase. #gamedev #golang #postgresql

### B. 中文完整对照稿

> Fractal Legend 的玩家交易基础从 UI 下方开始：持久化 Item Lock、Offer Revision、与 Revision 绑定的 Confirmation、双方 Character 的 Atomic Settlement、幂等重试和 Restart Recovery。Windows 验证已通过完整 Historical Node Scope 224/224、G10 Trade 29/29、Go Regression 290/290（0 Skip）、两项 Native Process Fixture 和 Windows amd64 Build。此前 22 个 macOS Environment/Platform Failure 已在不弱化测试的前提下关闭。G10 已完成 Technical Acceptance 和 Build in Public Closure，尚未 Release。FB Settlement、Market、Auction、Wallet、Blockchain、Fees 和 Trade UI 均不属于本阶段。

Screenshot suggestion: the sanitized real Windows closure result showing Historical Node 224/224, G10 Trade 29/29, Go 290/290 with 0 skips, both process fixtures, Windows amd64 build, and `AWAITING MANUAL ACCEPTANCE`. Do not show a local path, database URL, account, token, or protected Legacy content.

截图建议：展示经过脱敏的真实 Windows Closure Result，包括 Historical Node 224/224、G10 Trade 29/29、Go 290/290（0 Skip）、两项 Process Fixture、Windows amd64 Build 和 `AWAITING MANUAL ACCEPTANCE`。不得显示本地路径、Database URL、账号、Token 或受保护 Legacy Content。

## G11 — FB Ledger foundation

### A. English Primary Post

> Fractal Legend’s FB economy now starts with an immutable server-authoritative ledger instead of a mutable balance field. G11 adds integer atomic units, database-verified double-entry conservation, durable idempotency, concurrent double-spend protection, atomic rollback, compensating reversal, reconciliation, and full Game Server restart recovery through PostgreSQL. G11 checks passed 44/44; the full Go suite passed 334/334 with no skips; Race, 11/11 real PostgreSQL Ledger integrations, and Windows/Linux/macOS builds passed. Production exposes no mint or direct balance setter. Blockchain integration is deliberately outside this milestone: real deposits, withdrawals, wallets, Contribution, Mining, and Marketplace are not implemented. #gamedev #golang #postgresql

### B. 中文完整对照稿

> Fractal Legend 的 FB 经济现从不可变、Server-authoritative Ledger 开始，而不是直接修改一个 Balance Field。G11 增加整数 Atomic Unit、数据库验证的 Double-entry Conservation、持久 Idempotency、Concurrent Double-spend Protection、Atomic Rollback、补偿式 Reversal、Reconciliation，以及通过 PostgreSQL 完成的完整 Game Server Restart Recovery。G11 检查 44/44 通过；Go 完整套件 334/334 通过、0 Skip；Race、11/11 真实 PostgreSQL Ledger Integration 和 Windows/Linux/macOS Build 均通过。生产环境不暴露 Mint 或 Direct Balance Setter。Blockchain Integration 被明确排除在本阶段之外：真实 Deposit、Withdrawal、Wallet、Contribution、Mining 和 Marketplace 均未实现。#gamedev #golang #postgresql

Screenshot suggestion: use the sanitized G11 verification card showing 44/44 G11 checks, 334/334 Go regression, full Race, 11/11 real PostgreSQL Ledger integration, the 1,000-operation invariant sequence, high-concurrency protection, and three-platform builds. Do not show a terminal prompt, account, token, DSN, local path, wallet data, or protected Legacy content.

截图建议：使用经过脱敏的 G11 Verification Card，展示 44/44 G11 检查、334/334 Go Regression、完整 Race、11/11 真实 PostgreSQL Ledger Integration、1,000 Operation Invariant Sequence、High-concurrency Protection 和三平台 Build。不得显示 Terminal Prompt、账号、Token、DSN、本地路径、Wallet Data 或受保护 Legacy Content。

## G12 — FB-backed player trade settlement

Status: **Draft only · Not published · G12 technical acceptance PASS**

### A. English Primary Post

> Fractal Legend now has its first atomic settlement path connecting secure item trade with the server-authoritative FB Ledger. G12 settles item-only, FB-only, and mixed item + FB offers in one PostgreSQL transaction with 0% player-trade fee, revision-bound confirmation, deterministic locking, double-spend protection, idempotent restart replay, and full rollback on partial failure. Technical acceptance passed: targeted checks 45/45, full Go 379/379 with no skips, final Race, 100 simultaneous settlements with FB conservation, and Windows/Linux/macOS builds. This is not a release. Marketplace, Trade UI, wallets, deposits, withdrawals, and blockchain settlement are not implemented. #gamedev #golang #postgresql

### B. 中文完整对照稿

> Fractal Legend 现在拥有第一条连接安全 Item Trade 与 Server-authoritative FB Ledger 的原子结算路径。G12 在一个 PostgreSQL Transaction 中结算 Item-only、FB-only 和混合 Item + FB Offer，玩家交易手续费为 0%；同时提供 Revision-bound Confirmation、确定性 Lock、Double-spend Protection、幂等 Restart Replay，以及部分失败时的完整 Rollback。技术验收通过：专项检查 45/45、Go 完整回归 379/379 且 0 Skip、最终 Race、100 个同时 Settlement 的 FB 守恒，以及 Windows/Linux/macOS Build。本阶段不是 Release。Marketplace、Trade UI、Wallet、Deposit、Withdrawal 和 Blockchain Settlement 均未实现。#gamedev #golang #postgresql

Screenshot suggestion: use the sanitized G12 verification card showing 45/45 targeted checks, 12/12 real PostgreSQL checks, 379/379 full Go regression, final Race, 200 property iterations, 100 simultaneous settlements, FB conservation, 0% fee, and three-platform builds. Do not show a terminal prompt, account, credential, DSN, local path, wallet data, database rows, or protected Legacy content.

The card was captured before manual acceptance and therefore still says `READY_FOR_REVIEW · MANUAL ACCEPTANCE PENDING`. If used after Stage Close, label that line as the historical capture state and state G12 technical acceptance PASS in the accompanying caption; the original image is intentionally unchanged.

截图建议：使用经过脱敏的 G12 Verification Card，展示 45/45 专项检查、12/12 真实 PostgreSQL 检查、379/379 Go 完整回归、最终 Race、200 轮 Property Iteration、100 个同时 Settlement、FB 守恒、0% Fee 和三平台 Build。不得显示 Terminal Prompt、账号、Credential、DSN、本地路径、Wallet Data、Database Row 或受限制 Legacy Content。

此卡片摄于人工验收前，因此仍显示 `READY_FOR_REVIEW · MANUAL ACCEPTANCE PENDING`。若在 Stage Close 后使用，配文须说明该行是历史截图状态，并明确 G12 技术验收已 PASS；原图有意保持不变。

## G13 — Contribution Ledger foundation

Status: **Draft only · Not published · Technical acceptance PASS · Manual acceptance PASS**

状态：**仅为草稿 · 未发布 · 技术验收 PASS · 人工验收 PASS**

### A. English Primary Thread

1. G13 adds an internal, non-transferable Contribution Ledger to Fractal Legend. V1 grants 1 point per 1 explicitly eligible FB system-spend unit. Unknown sources fail closed; G12 player trades earn 0. #gamedev #golang
2. FB debit and point credit share one PostgreSQL transaction. G13 checks passed 41/41, including 19/19 real PostgreSQL checks; full Go passed 420/420. Race and Windows/Linux/macOS builds passed. #postgresql
3. G13 passed manual acceptance; this foundation is not a release. There is no live spend producer. Mining, Marketplace, Wallet, Deposit, Withdrawal, Blockchain, and Ordinals features are not active.

### B. 中文完整 Thread 对照

1. G13 为 Fractal Legend 新增内部、不可转账的 Contribution Ledger。V1 对每 1 单位经显式判定合格的 FB System Spend 发放 1 Point。未知来源默认拒绝；G12 Player Trade 发放 0 Point。#gamedev #golang
2. FB Debit 与 Point Credit 在同一个 PostgreSQL Transaction 中完成。G13 专项检查 41/41 通过，其中真实 PostgreSQL 检查 19/19；Go 完整回归 420/420 通过。Race 与 Windows/Linux/macOS Build 通过。#postgresql
3. G13 已通过人工验收；这套 Foundation 不是 Release。尚无真实消费 Producer。Mining、Marketplace、Wallet、Deposit、Withdrawal、Blockchain 和 Ordinals 功能均未上线。

Screenshot suggestion: use `screenshots/g13-contribution-ledger-foundation.png`, a sanitized card derived from the actual 41/41 G13, 19/19 real PostgreSQL, 420/420 full Go, Race, 200-iteration property, 100-way concurrency, and three-platform build results. It records `READY_FOR_REVIEW · MANUAL ACCEPTANCE PENDING` and the known G2 Node 95/96 limitation. The pending line is the historical capture state; G13 manual acceptance later passed. Do not describe the image as release evidence. Do not show accounts, credentials, DSNs, local paths, database rows, or restricted Legacy content.

截图建议：使用 `screenshots/g13-contribution-ledger-foundation.png`，它是根据实际 41/41 G13、19/19 真实 PostgreSQL、420/420 Go 完整回归、Race、200 轮 Property、100 并发及三平台 Build 结果制作的脱敏卡片。卡片记录 `READY_FOR_REVIEW · MANUAL ACCEPTANCE PENDING` 和已知 G2 Node 95/96 限制。Pending 一行是拍摄时的历史状态；G13 人工验收此后已通过。不得把图片描述为 Release 证据；不得展示账号、Credential、DSN、本地路径、Database Row 或受限制 Legacy 内容。

## Public screenshot checklist

Before publishing any frame:

- Crop to the game page or a purpose-built public test summary.
- Show no terminal prompt, home directory, local drive path, account name, email, repository credential, token, password, or environment variable.
- Show no raw database row dump, packet capture, private server configuration, or restricted Legacy binary/resource content.
- Generated Fractal Override art, game UI, public architecture labels, test totals, and runtime state are suitable for publication.
- Keep unfinished systems labeled accurately. Trade UI, Warehouse, Durability, Marketplace, and production deployment are not live.

发布任何画面前：

- 裁切到游戏页面或专门制作的公开测试摘要。
- 不展示 Terminal Prompt、Home Directory、本地磁盘路径、账号、Email、仓库 Credential、Token、Password 或 Environment Variable。
- 不展示原始 Database Row Dump、Packet Capture、Private Server Configuration 或受限制 Legacy Binary/Resource 内容。
- 可以展示生成的 Fractal Override 美术、游戏 UI、公开架构标识、测试总数和 Runtime State。
- 准确标识未完成的系统。Trade UI、Warehouse、Durability、Marketplace 和生产部署均未上线。

## Reviewed screenshot material

| File | Intended use | Review result |
|---|---|---|
| `screenshots/g8-equipment-runtime.png` | G8 browser runtime and debug equipment panel | Approved after visual inspection; game page only, no account, credential, token, terminal, or local path visible. |
| `screenshots/g9-persistence-foundation.png` | G9 sanitized PostgreSQL restart and test report / G9 脱敏 PostgreSQL 重启与测试报告 | Approved after visual inspection; no account, credential, DSN, token, terminal prompt, local path, raw database row, or restricted resource content visible. / 已通过目视检查；未显示账号、凭据、DSN、Token、Terminal Prompt、本地路径、原始数据库行或受限制资源内容。 |
| `screenshots/g10-trade-foundation.png` | G10 sanitized Windows closure verification / G10 脱敏 Windows Closure 验证结果 | Approved after visual inspection; shows Historical Node 224/224, G10 Trade 29/29, Go 290/290 with 0 skips, process fixtures, Windows amd64 build, preserved failure history, and manual-acceptance status. No account, credential, DSN, token, terminal prompt, local path, raw database row, or restricted resource content is visible. New SHA-256: `c0672d3cd1c998e9a5ce07c8542a118ae93fdfd0b9ffdff7a1a84496624d95b2`; replaced SHA-256: `e2dbe70f41236eba07836c5d91fd91d25c14d0b64ab84a2c517710f13896582c`. / 已通过目视检查；展示 Historical Node 224/224、G10 Trade 29/29、Go 290/290（0 Skip）、Process Fixture、Windows amd64 Build、保留的失败历史和人工验收状态，未显示账号、凭据、DSN、Token、Terminal Prompt、本地路径、原始数据库行或受限制资源内容。新 SHA-256：`c0672d3cd1c998e9a5ce07c8542a118ae93fdfd0b9ffdff7a1a84496624d95b2`；被替换的 SHA-256：`e2dbe70f41236eba07836c5d91fd91d25c14d0b64ab84a2c517710f13896582c`。 |
| `screenshots/g11-fb-ledger-foundation.png` | G11 sanitized Ledger verification / G11 脱敏 Ledger 验证结果 | Approved after visual inspection; shows 44/44 G11 tests, 334/334 Go regression, 11/11 real PostgreSQL Ledger integration, Race, invariants, concurrency, cross-platform builds, the honest G2 environment note, and review state. No account, credential, DSN, token, terminal prompt, local path, raw database row, wallet data, or restricted resource content is visible. SHA-256: `634ccb1a252e36379adeb051b242f478bfb71c2f5864b1ccaa2222f932b21ec0`. / 已通过目视检查；展示 44/44 G11 Test、334/334 Go Regression、11/11 真实 PostgreSQL Ledger Integration、Race、Invariant、Concurrency、Cross-platform Build、如实记录的 G2 环境说明和 Review 状态；未显示账号、凭据、DSN、Token、Terminal Prompt、本地路径、原始数据库行、Wallet Data 或受限制资源内容。SHA-256：`634ccb1a252e36379adeb051b242f478bfb71c2f5864b1ccaa2222f932b21ec0`。 |
| `screenshots/g12-fb-trade-settlement.png` | G12 sanitized atomic-settlement verification / G12 脱敏原子结算验证结果 | Review candidate generated from the actual final test results. It shows 45/45 targeted checks, 12/12 PostgreSQL checks, 379/379 full Go, final Race, 200 property iterations, 100 simultaneous settlements, zero fee, conservation, three-platform builds, and READY_FOR_REVIEW. It contains no terminal prompt, account, credential, DSN, token, local path, database row, wallet data, or restricted Legacy content. SHA-256: `161e2493863bb1c2cc93de35508529f713dc3ea35af76d3bbed8749b28d94517`. / 根据最终实际测试结果生成的 Review Candidate，展示 45/45 专项检查、12/12 PostgreSQL 检查、379/379 Go 完整回归、最终 Race、200 轮 Property Iteration、100 个同时 Settlement、零手续费、守恒、三平台 Build 和 READY_FOR_REVIEW；不包含 Terminal Prompt、账号、Credential、DSN、Token、本地路径、Database Row、Wallet Data 或受限制 Legacy Content。SHA-256：`161e2493863bb1c2cc93de35508529f713dc3ea35af76d3bbed8749b28d94517`。 |
| `screenshots/g13-contribution-ledger-foundation.png` | G13 sanitized Contribution verification / G13 脱敏 Contribution 验证结果 | Visually reviewed candidate from actual final checks: 41/41 G13, 19/19 real PostgreSQL, 420/420 Go, Race, 200 property iterations, 100-way concurrency, three-platform builds, and honest G2 Node 95/96 limitation. The card explicitly says manual acceptance is pending. No terminal prompt, account, credential, DSN, token, local path, database row, wallet data, or restricted Legacy content is visible. SHA-256: `cae1f1a2985a4c8f7ce954310b013c6529c9370ef30e33664ad78fe3c5ba250c`. / 根据最终真实检查生成并经过目视审查的候选图：展示 41/41 G13、19/19 真实 PostgreSQL、420/420 Go、Race、200 轮 Property、100 并发、三平台 Build，以及如实标注的 G2 Node 95/96 限制。卡片明确标记等待人工验收。未显示 Terminal Prompt、账号、Credential、DSN、Token、本地路径、Database Row、Wallet Data 或受限制 Legacy 内容。SHA-256：`cae1f1a2985a4c8f7ce954310b013c6529c9370ef30e33664ad78fe3c5ba250c`。 |

Earlier-stage screenshot suggestions above are a capture plan. They are not evidence that those screenshots have already been produced or published.
