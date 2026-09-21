# Fractal Legend / 分形传奇

![Fractal Legend official launch poster](docs/public/assets/marketing/fractal-legend-launch-poster.png)

> **Official promotional artwork. Actual implementation status is documented below.**
>
> **官方宣传画。实际实现状态以本文下方的工程记录为准。**

## English — Primary

**Building toward the first Legend-style MMORPG experience for the Bitcoin / Fractal ecosystem.**

Fractal Legend is a browser-native Legend-style MMORPG being built for the Fractal / Bitcoin ecosystem. Its direction combines classic combat, player-driven economies, digital ownership, Ordinals assets, and a modular social game world.

Development is milestone-driven. The accepted server-authoritative G1–G13 foundation covers world movement, browser rendering, living entities, combat, monster AI, one Warrior skill, loot, inventory, equipment, character stats, PostgreSQL persistence, secure item trading, an internal FB ledger, atomic item + FB player Trade settlement, and an internal non-transferable Contribution Ledger. This is an engineering foundation, not a public game release.

### Repository role

This repository is the **sanitized public mirror**. It began after G10 with a new Git history and contains only allowlisted, audited material. The private canonical repository remains the authority for complete internal development history, migration references, restricted evidence, and acceptance records.

The public mirror does not copy or reconstruct the private repository history. Earlier milestone SHAs are retained only as plain private archive references in the development history; they are not links and are not commits in this repository.

### Why Fractal / Bitcoin

The long-term design treats Fractal and Bitcoin as more than branding. Ownership, identity, economic rules, and eligible digital assets are intended to connect through explicit verification and game-balance rules. Mainnet wallets, deposits, withdrawals, and Ordinals activation are **not live**.

### Current status

| Capability | Status | Accepted boundary |
|---|---|---|
| G1–G8 gameplay foundations | **FOUNDATION COMPLETE** | World, rendering, entities, combat, AI, one Warrior skill, loot, inventory, equipment, and runtime stats |
| G9 Persistence Foundation | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate, reconnect, and full Game Server restart restore |
| G10 Trade Foundation | **FOUNDATION COMPLETE** | Secure item-trade domain, extended in G12; no Trade UI |
| G11 FB Ledger Foundation | **FOUNDATION COMPLETE** | Internal server-authoritative ledger; no blockchain, deposit, withdrawal, or wallet |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | Atomic PostgreSQL settlement, 0% fee, no Marketplace or blockchain settlement |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | Internal non-transferable 1:1 eligible-spend ledger; no live spend producer |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted technical slices do not form a public release |
| Wallet / blockchain deposit and withdrawal | **PLANNED** | Not live |
| Ordinals activation | **PLANNED** | No production activation exists |
| Mining, guild, siege, social and media systems | **PLANNED / RESEARCH** | Product direction only |

### Latest accepted milestone

**G13 — Contribution Ledger Foundation: technical and manual acceptance PASS**

- G13 targeted checks: **41/41 PASS**, including **19/19** real PostgreSQL integration checks
- Property sequence: **200/200 PASS**; **100** concurrent distinct-source postings and same-source replays passed
- Private canonical full Go regression: **420/420 PASS**, 0 failures, 0 skips
- Go Race, Vet, and Windows/Linux/macOS builds: **PASS**
- Node: **95/96 — KNOWN PRE-EXISTING G2 LIMITATION** caused only by the unavailable restricted Legacy Archive
- Private canonical G13 commit reference: `3986d02947b63afe2ed0822bd3859fea431d4dda`; merge reference: `748e3156e85832fcb95334d51bc2be03ea7f9581`

`CONTRIBUTION_RULE_V1` grants one point per eligible FB system-spend unit. Only the internal `SYSTEM_SERVICE` foundation category is eligible; unknown sources fail closed and G12 player Trade earns zero points. FB debit and Contribution credit share one PostgreSQL transaction. **POST-G13 HARD GATE:** Blocking refunds/reversals of linked spends is temporary safety behavior, not the final production design. Before any real eligible spend producer connects, atomic FB refund/reversal plus Contribution reversal/compensation must be implemented, including an explicit recovery path if points were already spent.

### Public source snapshot

The mirror publishes audited project-owned Go domains for AI, Character Stats, Combat Rules, Navigation, registries, PostgreSQL Persistence, G10/G12 Trade, the G11 FB Ledger, and the G13 Contribution Ledger Foundation. It includes unit tests and versioned schema migrations. It intentionally excludes Legacy seller source, private fixtures, Canonical exports, import tools, protected assets, local-environment integrations, and the complete private runtime assembly.

Run the public Go checks:

```bash
cd apps/game-server
go test ./...
```

See [Public Code Provenance](PUBLIC-CODE-PROVENANCE.md) for the exact publication boundary.

### Public documentation

- [Project Overview](docs/public/PROJECT-OVERVIEW.md)
- [Gameplay](docs/public/GAMEPLAY.md)
- [Web3 & Ordinals](docs/public/WEB3-AND-ORDINALS.md)
- [Economy](docs/public/ECONOMY.md)
- [Current Status](docs/public/CURRENT-STATUS.md)
- [G1–G10 Development Timeline](docs/public/DEVELOPMENT-TIMELINE.md)
- [Public Mirror Development History](docs/public/DEVELOPMENT-HISTORY.md)
- [Roadmap](docs/public/ROADMAP.md)
- [Architecture](docs/public/ARCHITECTURE.md)
- [Build in Public](docs/public/BUILD-IN-PUBLIC.md)
- [Media](docs/public/MEDIA.md)
- [Technical Screenshots](docs/public/SCREENSHOTS.md)
- [FAQ](docs/public/FAQ.md)
- [G9–G13 Devlogs](docs/devlog/)
- [Curated Interaction Records](docs/interactions/)
- [G10 Trade ADR](docs/adr/0009-g10-trade-foundation.md)
- [G11 FB Ledger ADR](docs/adr/0010-g11-fb-ledger-foundation.md)
- [G12 Atomic Settlement ADR](docs/adr/0011-g12-atomic-item-fb-trade-settlement.md)
- [G12 Public Sync Allowlist](docs/public-sync/G12-PUBLIC-SYNC-CANDIDATE.md)
- [G13 Contribution ADR](docs/adr/0012-g13-contribution-ledger-foundation.md)
- [G13 Devlog](docs/devlog/G13-contribution-ledger-foundation.md)
- [G13 Interaction Record](docs/interactions/G13-contribution-ledger-foundation.md)
- [G13 Public Sync Allowlist](docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md)
- [Public Mirror Policy](PUBLIC-MIRROR-POLICY.md)
- [Security Policy](SECURITY.md)

### License

A project license has not yet been selected. Public availability of the source does not grant additional reuse rights beyond applicable law.

### Honest limitations

Fractal Legend has no announced public launch date. It is not a production service, mainnet integration, finished browser MMORPG, or investment product. Public artwork communicates product direction; it does not prove gameplay implementation.

---

## 中文 — 完整对应版本

**正在构建面向 Bitcoin / Fractal 生态的首个传奇风格 MMORPG 体验。**

Fractal Legend / 分形传奇是一款正在为 Fractal / Bitcoin 生态构建的浏览器原生传奇风格 MMORPG。项目方向结合经典战斗、玩家驱动经济、数字所有权、Ordinals 资产，以及模块化的社交游戏世界。

开发按里程碑推进。已验收的 Server-authoritative G1–G13 Foundation 覆盖 World Movement、Browser Rendering、Living Entity、Combat、Monster AI、一项 Warrior Skill、Loot、Inventory、Equipment、Character Stats、PostgreSQL Persistence、安全的 Item Trade、内部 FB Ledger、原子 Item + FB 玩家 Trade Settlement，以及内部不可转账的 Contribution Ledger。这是一套工程基础，并不代表游戏已经公开上线。

### 仓库定位

本仓库是 **Sanitized Public Mirror**。它在 G10 之后以全新 Git History 建立，只包含通过 Allowlist 和审计的材料。Private Canonical Repository 继续作为完整内部开发历史、迁移参考、受限制证据和验收记录的权威来源。

Public Mirror 不复制或重建 Private Repository History。更早里程碑 SHA 只会在 Development History 中作为纯文本 Private Archive Reference 保留；它们不是链接，也不是本仓库中的 Commit。

### 为什么选择 Fractal / Bitcoin

长期设计不会把 Fractal 和 Bitcoin 只当作品牌标识。Ownership、Identity、Economic Rule 和符合条件的 Digital Asset，计划通过明确的验证流程与 Game Balance Rule 接入游戏。Mainnet Wallet、Deposit、Withdrawal 和 Ordinals Activation **尚未上线**。

### 当前状态

| 能力 | 状态 | 已验收边界 |
|---|---|---|
| G1–G8 Gameplay Foundation | **FOUNDATION COMPLETE** | World、Rendering、Entity、Combat、AI、一项 Warrior Skill、Loot、Inventory、Equipment 与 Runtime Stats |
| G9 Persistence Foundation | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate、Reconnect 与完整 Game Server Restart Restore |
| G10 Trade Foundation | **FOUNDATION COMPLETE** | 安全 Item Trade Domain，在 G12 得到扩展；不含 Trade UI |
| G11 FB Ledger Foundation | **FOUNDATION COMPLETE** | 内部 Server-authoritative Ledger；不含 Blockchain、Deposit、Withdrawal 或 Wallet |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | PostgreSQL 原子结算、0% 手续费；不含 Marketplace 或 Blockchain Settlement |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | 内部不可转账的 1:1 合格消费账本；尚无真实消费 Producer |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Technical Slice 尚未组成公开 Release |
| Wallet / Blockchain Deposit 与 Withdrawal | **PLANNED** | 尚未上线 |
| Ordinals Activation | **PLANNED** | 不存在 Production Activation |
| Mining、Guild、Siege、Social 与 Media System | **PLANNED / RESEARCH** | 仅为产品方向 |

### 最新已验收里程碑

**G13 — Contribution Ledger Foundation：技术与人工验收 PASS**

- G13 专项检查：**41/41 PASS**，其中真实 PostgreSQL Integration **19/19**
- Property Sequence：**200/200 PASS**；**100** 并发不同 Source Posting 和同 Source Replay 通过
- Private Canonical Go 完整回归：**420/420 PASS**、0 Fail、0 Skip
- Go Race、Vet 与 Windows/Linux/macOS Build：**PASS**
- Node：**95/96 — KNOWN PRE-EXISTING G2 LIMITATION**，唯一原因是受限制 Legacy Archive 不可用
- Private Canonical G13 Commit Reference：`3986d02947b63afe2ed0822bd3859fea431d4dda`；Merge Reference：`748e3156e85832fcb95334d51bc2be03ea7f9581`

`CONTRIBUTION_RULE_V1` 对每 1 单位合格 FB System Spend 发放 1 Point。仅内部 `SYSTEM_SERVICE` Foundation 类别合格；未知来源默认拒绝，G12 Player Trade 发放零分。FB Debit 与 Contribution Credit 在同一个 PostgreSQL Transaction 中完成。**POST-G13 HARD GATE：** 当前对关联消费的 Refund/Reversal Block 是临时安全行为，不是最终生产设计。接入任何真实合格消费 Producer 前，必须实现原子 FB Refund/Reversal 加 Contribution Reversal/Compensation，包括积分已被消费时的明确 Recovery 路径。

### 公开源码快照

本镜像公开经过审计、属于项目自有的 Go Domain，包括 AI、Character Stats、Combat Rules、Navigation、Registry、PostgreSQL Persistence、G10/G12 Trade、G11 FB Ledger 和 G13 Contribution Ledger Foundation，并包含 Unit Test 与 Versioned Schema Migration。镜像明确排除 Legacy Seller Source、Private Fixture、Canonical Export、Import Tool、受保护 Asset、本地环境 Integration 和完整 Private Runtime Assembly。

运行公开 Go 检查：

```bash
cd apps/game-server
go test ./...
```

准确公开边界详见 [Public Code Provenance](PUBLIC-CODE-PROVENANCE.md)。

### 公开文档

- [项目概览](docs/public/PROJECT-OVERVIEW.md)
- [Gameplay](docs/public/GAMEPLAY.md)
- [Web3 与 Ordinals](docs/public/WEB3-AND-ORDINALS.md)
- [经济](docs/public/ECONOMY.md)
- [当前状态](docs/public/CURRENT-STATUS.md)
- [G1–G10 开发时间线](docs/public/DEVELOPMENT-TIMELINE.md)
- [Public Mirror 开发历史](docs/public/DEVELOPMENT-HISTORY.md)
- [Roadmap](docs/public/ROADMAP.md)
- [Architecture](docs/public/ARCHITECTURE.md)
- [Build in Public](docs/public/BUILD-IN-PUBLIC.md)
- [Media](docs/public/MEDIA.md)
- [Technical Screenshot](docs/public/SCREENSHOTS.md)
- [FAQ](docs/public/FAQ.md)
- [G9–G13 Devlog](docs/devlog/)
- [整理后的 Interaction Record](docs/interactions/)
- [G10 Trade ADR](docs/adr/0009-g10-trade-foundation.md)
- [G11 FB Ledger ADR](docs/adr/0010-g11-fb-ledger-foundation.md)
- [G12 Atomic Settlement ADR](docs/adr/0011-g12-atomic-item-fb-trade-settlement.md)
- [G12 Public Sync Allowlist](docs/public-sync/G12-PUBLIC-SYNC-CANDIDATE.md)
- [G13 Contribution ADR](docs/adr/0012-g13-contribution-ledger-foundation.md)
- [G13 Devlog](docs/devlog/G13-contribution-ledger-foundation.md)
- [G13 Interaction Record](docs/interactions/G13-contribution-ledger-foundation.md)
- [G13 Public Sync Allowlist](docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md)
- [Public Mirror Policy](PUBLIC-MIRROR-POLICY.md)
- [Security Policy](SECURITY.md)

### License

项目尚未选择 License。Source 可以公开访问，不代表除适用法律外自动授予额外再使用权利。

### 诚实边界

Fractal Legend 尚未公布公开上线日期。它目前不是 Production Service、Mainnet Integration、完成版 Browser MMORPG 或投资产品。公开宣传画表达产品方向，不能作为 Gameplay 实现证据。
