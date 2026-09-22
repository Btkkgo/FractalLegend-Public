# Fractal Legend / 分形传奇

![Fractal Legend official launch poster](docs/public/assets/marketing/fractal-legend-launch-poster.png)

> **Official promotional artwork. Actual implementation status is documented below.**
>
> **官方宣传画。实际实现状态以本文下方的工程记录为准。**

## English — Primary

**Building toward the first Legend-style MMORPG experience for the Bitcoin / Fractal ecosystem.**

Fractal Legend is a browser-native Legend-style MMORPG being built for the Fractal / Bitcoin ecosystem. Its direction combines classic combat, player-driven economies, digital ownership, Ordinals assets, and a modular social game world.

Development is milestone-driven. The accepted server-authoritative G1–G16 foundation covers world and combat systems, inventory and equipment, PostgreSQL persistence, direct item + FB trade settlement, internal FB and Contribution ledgers, refund recovery, eligible system spend orchestration, and an internal recycle migration foundation. This is an engineering foundation, not a public game release.

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
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | Atomic FB + Contribution compensation, recovery debt, hold; no live spend producer |
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | Internal server-owned orchestration; no live gameplay producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | Internal item consumption into allowed materials and Reputation; no gameplay recycle entry or production rule |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted technical slices do not form a public release |
| Wallet / blockchain deposit and withdrawal | **PLANNED** | Not live |
| Ordinals activation | **PLANNED** | No production activation exists |
| Mining, guild, siege, social and media systems | **PLANNED / RESEARCH** | Product direction only |

### Latest accepted milestone

**G16 — Recycle Migration Foundation: technical and manual acceptance PASS**

- Private canonical normal Go and Race: **492/492 PASS each**, **0 skipped**; Vet and Linux/Windows/macOS builds PASS
- GitHub Actions: **6/6 jobs PASS** after CI identified and the team fixed an exact PostgreSQL receipt timestamp replay mismatch
- Same-item concurrency: one settlement from 100 requests; 100 distinct valid items settled
- Receipt: first response, same-process replay, restart replay, and reload use the same persisted canonical value
- G2 Node remains **13/14** due to the existing restricted Asset Atlas limitation; Windows runtime was **NOT_RUN**

G16 atomically consumes one owned item instance and records permitted material and non-transferable Reputation output with an immutable receipt and audit trail. It creates **no FB, Contribution, or Black Iron Ore**. Production recycle rules and gameplay entry remain disabled. The first Linux CI run exposed Go nanosecond versus PostgreSQL microsecond timestamp behavior; canonical UTC handling and persisted receipt replay fixed it. See the bilingual [G16 Devlog](docs/devlog/G16-recycle-migration-foundation.md) for the full failure and repair record. This is not a public game release.

### Public source snapshot

The mirror publishes audited project-owned Go domains for AI, Character Stats, Combat Rules, Navigation, registries, PostgreSQL Persistence, G10/G12 Trade, the G11 FB Ledger, G13–G14 Contribution and refund recovery, G15 system spend, and G16 recycle. It includes unit tests and versioned schema migrations. It intentionally excludes Legacy seller source, private fixtures, Canonical exports, import tools, protected assets, local-environment integrations, and the complete private runtime assembly.

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
- [G9–G16 Devlogs](docs/devlog/)
- [Curated Interaction Records](docs/interactions/)
- [G10 Trade ADR](docs/adr/0009-g10-trade-foundation.md)
- [G11 FB Ledger ADR](docs/adr/0010-g11-fb-ledger-foundation.md)
- [G12 Atomic Settlement ADR](docs/adr/0011-g12-atomic-item-fb-trade-settlement.md)
- [G12 Public Sync Allowlist](docs/public-sync/G12-PUBLIC-SYNC-CANDIDATE.md)
- [G13 Contribution ADR](docs/adr/0012-g13-contribution-ledger-foundation.md)
- [G13 Devlog](docs/devlog/G13-contribution-ledger-foundation.md)
- [G13 Interaction Record](docs/interactions/G13-contribution-ledger-foundation.md)
- [G13 Public Sync Allowlist](docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md)
- [G14 Refund ADR](docs/adr/0013-g14-contribution-refund-reversal.md)
- [G14 Devlog](docs/devlog/G14-contribution-refund-reversal.md)
- [G14 Interaction Record](docs/interactions/G14-contribution-refund-reversal.md)
- [G14 Public Sync Allowlist](docs/public-sync/G14-PUBLIC-SYNC-CANDIDATE.md)
- [G15 Devlog](docs/devlog/G15-eligible-system-spend.md)
- [G16 Recycle ADR](docs/adr/0015-g16-recycle-migration-foundation.md)
- [G16 Devlog](docs/devlog/G16-recycle-migration-foundation.md)
- [G16 Interaction Record](docs/interactions/G16-recycle-migration-foundation.md)
- [G16 Public Sync Allowlist](docs/public-sync/G16-PUBLIC-SYNC-CANDIDATE.md)
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

开发按里程碑推进。已验收的 Server-authoritative G1–G16 Foundation 覆盖 World 与 Combat 系统、Inventory 与 Equipment、PostgreSQL Persistence、直接 Item + FB Trade Settlement、内部 FB 与 Contribution Ledger、退款恢复、合格 System Spend 编排，以及内部回收迁移基础。这是一套工程基础，并不代表游戏已经公开上线。

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
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | 原子 FB + Contribution 补偿、Recovery Debt 与 Hold；尚无真实消费 Producer |
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | 内部由服务器控制的编排；尚无真实玩法 Producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | 内部物品消费、允许材料与 Reputation 产出；没有真实玩法回收入口或生产规则 |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Technical Slice 尚未组成公开 Release |
| Wallet / Blockchain Deposit 与 Withdrawal | **PLANNED** | 尚未上线 |
| Ordinals Activation | **PLANNED** | 不存在 Production Activation |
| Mining、Guild、Siege、Social 与 Media System | **PLANNED / RESEARCH** | 仅为产品方向 |

### 最新已验收里程碑

**G16 — Recycle Migration Foundation：技术与人工验收 PASS**

- Private Canonical 普通 Go 与 Race：各 **492/492 PASS**、**0 跳过**；Vet 及 Linux/Windows/macOS 构建通过
- GitHub Actions：发现并修复 PostgreSQL Receipt 时间精确重放差异后，**6/6 Jobs 通过**
- 同一物品 100 请求只结算一次；100 件不同合法物品均完成结算
- 首次 Receipt、同进程重放、重启重放与重新读取使用相同的持久化规范值
- G2 Node 仍因既有受限 Asset Atlas 限制为 **13/14**；Windows Runtime **NOT_RUN**

G16 原子消费一件归属玩家的物品实例，记录允许材料及不可转让的 Reputation 产出，并保存不可变 Receipt 与审计证据。它**不创建 FB、Contribution 或黑铁矿石**。生产回收规则与真实玩法入口仍禁用。首次 Linux CI 发现 Go 纳秒与 PostgreSQL 微秒时间的差异；UTC 规范化及返回持久化 Receipt 修复了重放。完整故障与修复过程见双语 [G16 Devlog](docs/devlog/G16-recycle-migration-foundation.md)。这不是公开游戏 Release。

### 公开源码快照

本镜像公开经过审计、属于项目自有的 Go Domain，包括 AI、Character Stats、Combat Rules、Navigation、Registry、PostgreSQL Persistence、G10/G12 Trade、G11 FB Ledger、G13–G14 Contribution 与 Refund Recovery、G15 System Spend，以及 G16 Recycle，并包含 Unit Test 与 Versioned Schema Migration。镜像明确排除 Legacy Seller Source、Private Fixture、Canonical Export、Import Tool、受保护 Asset、本地环境 Integration 和完整 Private Runtime Assembly。

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
- [G9–G16 Devlog](docs/devlog/)
- [整理后的 Interaction Record](docs/interactions/)
- [G10 Trade ADR](docs/adr/0009-g10-trade-foundation.md)
- [G11 FB Ledger ADR](docs/adr/0010-g11-fb-ledger-foundation.md)
- [G12 Atomic Settlement ADR](docs/adr/0011-g12-atomic-item-fb-trade-settlement.md)
- [G12 Public Sync Allowlist](docs/public-sync/G12-PUBLIC-SYNC-CANDIDATE.md)
- [G13 Contribution ADR](docs/adr/0012-g13-contribution-ledger-foundation.md)
- [G13 Devlog](docs/devlog/G13-contribution-ledger-foundation.md)
- [G13 Interaction Record](docs/interactions/G13-contribution-ledger-foundation.md)
- [G13 Public Sync Allowlist](docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md)
- [G14 Refund ADR](docs/adr/0013-g14-contribution-refund-reversal.md)
- [G14 Devlog](docs/devlog/G14-contribution-refund-reversal.md)
- [G14 Interaction Record](docs/interactions/G14-contribution-refund-reversal.md)
- [G14 Public Sync Allowlist](docs/public-sync/G14-PUBLIC-SYNC-CANDIDATE.md)
- [G15 Devlog](docs/devlog/G15-eligible-system-spend.md)
- [G16 Recycle ADR](docs/adr/0015-g16-recycle-migration-foundation.md)
- [G16 Devlog](docs/devlog/G16-recycle-migration-foundation.md)
- [G16 Interaction Record](docs/interactions/G16-recycle-migration-foundation.md)
- [G16 Public Sync Allowlist](docs/public-sync/G16-PUBLIC-SYNC-CANDIDATE.md)
- [Public Mirror Policy](PUBLIC-MIRROR-POLICY.md)
- [Security Policy](SECURITY.md)

### License

项目尚未选择 License。Source 可以公开访问，不代表除适用法律外自动授予额外再使用权利。

### 诚实边界

Fractal Legend 尚未公布公开上线日期。它目前不是 Production Service、Mainnet Integration、完成版 Browser MMORPG 或投资产品。公开宣传画表达产品方向，不能作为 Gameplay 实现证据。
