# Fractal Legend / 分形传奇

![Fractal Legend official launch poster](docs/public/assets/marketing/fractal-legend-launch-poster.png)

> **Official promotional artwork. Actual implementation status is documented below.**
>
> **官方宣传画。实际实现状态以本文下方的工程记录为准。**

## English — Primary

**Building toward the first Legend-style MMORPG experience for the Bitcoin / Fractal ecosystem.**

Fractal Legend is a browser-native Legend-style MMORPG being built for the Fractal / Bitcoin ecosystem. Its direction combines classic combat, player-driven economies, digital ownership, Ordinals assets, and a modular social game world.

Development is milestone-driven. The accepted server-authoritative G1–G21-P0 foundation covers world and combat systems, inventory and equipment, PostgreSQL persistence, direct item + FB trade settlement, internal FB and Contribution ledgers, refund recovery, eligible system spend orchestration, internal recycle migration, global emission capacity, and Mining Block reward reservation. This is an engineering foundation, not a public game release.

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
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | Global capacity only; no player ore, production ratio, mining block, or reward distribution |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | Server-owned blocks and atomic reserved capacity; no reward distribution or player ore |
| G19 Black Iron Ore Migration | **FOUNDATION COMPLETE** | Reviewed identity compatibility, 1:1 inventory, immutable provenance; no issuance |
| G18.1 Stable Block Identity | **FOUNDATION COMPLETE** | Immutable BlockInstanceID |
| G20 Mining Power | **FOUNDATION COMPLETE** | SERVER authority, synthetic TEST profiles; no rewards |
| G21-P0 Settlement Prerequisites | **FOUNDATION COMPLETE** | Sealed input and immutable bindings; TEST-only accounting/issuance prerequisites, no real player ore or full settlement |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted technical slices do not form a public release |
| Wallet / blockchain deposit and withdrawal | **PLANNED** | Not live |
| Ordinals activation | **PLANNED** | No production activation exists |
| Mining, guild, siege, social and media systems | **PLANNED / RESEARCH** | Product direction only |

### Latest accepted milestone

**G21-P0 Settlement Prerequisite Foundation: final human acceptance PASS.** It closes the Seal, reservation/beneficiary binding, recovery, reward provenance and 500 participant prerequisite boundaries. It adds TEST-only writers; no production route, full reward allocation or settlement, real player ore issuance, claim or G22. Full G21 remains at the next human gate. [Public allowlist and verification](docs/public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md).

**Earlier G20 Mining Power Foundation: final human acceptance PASS; private canonical closure complete.**

The accepted chain is **G17 Global Mining Pool → G18 Mining Block + Recovery → G18.1 Stable Block Identity → G20 Server-authoritative Mining Power**. G17 capacity/reservation is not player ore. G19 identity migration preserves existing quantities without issuance.

G20 records immutable ValidatedMiningActivity as authority and derives Participant aggregates. Stable BlockInstanceID spans sessions/events/history. G18 uses a read-only ordinary SELECT without upstream row locks or cross-domain foreign keys, saving an immutable validation snapshot. PostgreSQL SERIALIZABLE commits activity and aggregate atomically, with global SourceEvent uniqueness and at most five attempts including the first. Synthetic Tool/Map inputs are server-resolved, final client power is rejected, integer scale is 1,000,000 and one final floor division determines power. Canonical replay timestamps use `t.UTC().Round(0)`; strict receipt equality is retained. Reconciliation is read-only and never repairs unknown history.

G20 has **90/90 scenarios PASS**. Private PR and post-merge CI each have **10/10 jobs PASS**, full Normal and Race each **882/882 PASS**, zero failures/skips/data race. Native Windows/Linux/macOS T50 has 24 identical logical results with digest `f5f2e54d342d02544c54565d488aa1e096fd9208c4ea4d23b291fa2f5ceb3d60`. The public subset is tested independently; its results appear in the public export manifest rather than borrowing private counts.

This is a TEST foundation, not a public game release. Production Service remains fail-closed. **NOT IMPLEMENTED:** reward distribution, block reward settlement, ore claim, mining pool deduction, production reward/duration/tool/map rules and full G21. G20 mutates no G18, FB, Contribution, Player Ore or Mining Pool state. Social material remains an unpublished draft.

### Public source snapshot

G21-P0 public subset verification: Normal and Race each **692/692 PASS**, zero failures/skips/data race; Vet, three-platform builds and a 240-file safety/link audit PASS. [Exact G21-P0 allowlist and receipts](docs/public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md).

Earlier G20 public subset verification: Normal **646/646**, Race **646/646**, zero failures/skips/data race; Vet, three platform builds and 215-file public safety/link audits PASS. [Full allowlist and receipts](docs/public-sync/G20-PUBLIC-SYNC-CANDIDATE.md).

The mirror publishes audited project-owned Go domains for AI, Character Stats, Combat Rules, Navigation, registries, PostgreSQL Persistence, G10/G12 Trade, the G11 FB Ledger, G13–G14 Contribution and refund recovery, G15 system spend, G16 recycle, G17 emission capacity, and G18 block reservation. It includes unit tests and versioned schema migrations. It intentionally excludes Legacy seller source, private fixtures, Canonical exports, import tools, protected assets, local-environment integrations, and the complete private runtime assembly.

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
- [G9–G21-P0 Devlogs](docs/devlog/)
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
- [G17 Emission Pool ADR](docs/adr/0016-g17-black-iron-emission-pool.md)
- [G17 Devlog](docs/devlog/G17-black-iron-emission-pool.md)
- [G17 Interaction Record](docs/interactions/G17-black-iron-emission-pool.md)
- [G17 Public Sync Allowlist](docs/public-sync/G17-PUBLIC-SYNC-CANDIDATE.md)
- [G18 Block Reservation ADR](docs/adr/0017-g18-mining-block-reward-reservation.md)
- [G18 Devlog](docs/devlog/G18-mining-block-reward-reservation.md)
- [G18 Interaction Record](docs/interactions/G18-mining-block-reward-reservation.md)
- [G18 Public Sync Allowlist](docs/public-sync/G18-PUBLIC-SYNC-CANDIDATE.md)
- [G13–G15 Timestamp Hardening Devlog](docs/devlog/G13-G15-postgres-timestamp-hardening.md)
- [G13–G15 Timestamp Hardening Interaction Record](docs/interactions/G13-G15-postgres-timestamp-hardening.md)
- [G13–G15 Timestamp Hardening Public Sync Allowlist](docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md)
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

开发按里程碑推进。已验收的 Server-authoritative G1–G20 Foundation 覆盖 World 与 Combat 系统、Inventory 与 Equipment、PostgreSQL Persistence、直接 Item + FB Trade Settlement、内部 FB 与 Contribution Ledger、退款恢复、合格 System Spend 编排、内部回收迁移、全服发行额度及 Mining Block 奖励预留。这是一套工程基础，并不代表游戏已经公开上线。

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
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | 仅建立全服发行额度；不发放玩家矿石，未确定生产比例，也没有 Mining Block 或奖励分配 |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | 服务器权威区块与原子容量预留；没有奖励分配或玩家矿石 |
| G19 黑铁矿石迁移 | **FOUNDATION COMPLETE** | 明确身份兼容、1:1 库存及不可变来源；不发行矿石 |
| G18.1 稳定区块身份 | **FOUNDATION COMPLETE** | 不可变 BlockInstanceID |
| G20 挖矿算力 | **FOUNDATION COMPLETE** | 服务器权威、合成 TEST 输入，不结算奖励 |
| G21-P0 结算前置契约 | **FOUNDATION COMPLETE** | 封存输入与不可变绑定；仅 TEST 的记账／发行前置，不发真实玩家矿石、不实现完整结算 |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Technical Slice 尚未组成公开 Release |
| Wallet / Blockchain Deposit 与 Withdrawal | **PLANNED** | 尚未上线 |
| Ordinals Activation | **PLANNED** | 不存在 Production Activation |
| Mining、Guild、Siege、Social 与 Media System | **PLANNED / RESEARCH** | 仅为产品方向 |

### 最新已验收里程碑

**G21-P0 结算前置契约：最终人工验收 PASS。** 本阶段闭合 Seal、预留／受益人绑定、恢复、奖励来源和 500 人前置边界，仅增加 TEST 写入器；没有生产入口、完整奖励分配或结算、真实玩家矿石发放、领取或 G22。完整 G21 仍待下一轮人工门槛。[公开清单与验证](docs/public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md)。

**此前 G20 算力基础：最终人工验收 PASS，私有 canonical 收尾完成。**

已验收链条为 **G17 全服矿池 → G18 挖矿区块与恢复 → G18.1 稳定区块身份 → G20 服务器权威算力**。G17 容量／预留不等于玩家矿石；G19 身份迁移保持既有数量，不发行矿石。

G20 以不可变 ValidatedMiningActivity 为权威、Participant 为派生汇总。稳定 BlockInstanceID 贯穿会话／事件／历史。G18 普通只读 SELECT 无上游行锁或跨域外键，保存不可变验证快照。PostgreSQL SERIALIZABLE 原子提交事实与汇总；SourceEvent 全局唯一，含首次最多五次。工具／地图输入由服务器解析合成数据，拒绝客户端最终算力；整数定点比例 1,000,000，只作一次最终除法向下截断。重放时间采用 `t.UTC().Round(0)`，保留回执严格相等。对账只读，不修复未知历史。

G20 **90/90 场景通过**；私有 PR 及合并后 CI 各 **10/10 作业通过**，完整 Normal／Race 各 **882/882**，零失败、跳过、数据竞争。Windows／Linux／macOS 原生 T50 的 24 个逻辑结果完全一致，摘要如上。公开子集独立运行验证，结果在公开导出清单中记录，不借用私有数量。

这是 TEST 基础，不是公开游戏发布；生产 Service 失败关闭。**未实现：**奖励分发、区块奖励结算、矿石申领、矿池扣减、正式奖励／时长／工具／地图规则及完整 G21。G20 不修改 G18、FB、Contribution、玩家矿石或 Mining Pool。社交素材保持未发布草稿。

### 公开源码快照

G21-P0 公开子集独立验证：Normal／Race 各 **692/692 PASS**，零失败／跳过／数据竞争；Vet、三平台构建和 240 文件安全／链接审计通过。[G21-P0 完整清单与回执](docs/public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md)。

此前 G20 公开子集独立验证：Normal／Race 各 **646/646**，零失败／跳过／数据竞争；Vet、三平台构建及 215 文件安全／链接审计通过。[完整清单与回执](docs/public-sync/G20-PUBLIC-SYNC-CANDIDATE.md)。

本镜像公开经过审计、属于项目自有的 Go Domain，包括 AI、Character Stats、Combat Rules、Navigation、Registry、PostgreSQL Persistence、G10/G12 Trade、G11 FB Ledger、G13–G14 Contribution 与 Refund Recovery、G15 System Spend、G16 Recycle、G17 发行额度及 G18 Block 预留，并包含 Unit Test 与 Versioned Schema Migration。镜像明确排除 Legacy Seller Source、Private Fixture、Canonical Export、Import Tool、受保护 Asset、本地环境 Integration 和完整 Private Runtime Assembly。

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
- [G9–G20 Devlog](docs/devlog/)
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
- [G17 Emission Pool ADR](docs/adr/0016-g17-black-iron-emission-pool.md)
- [G17 Devlog](docs/devlog/G17-black-iron-emission-pool.md)
- [G17 Interaction Record](docs/interactions/G17-black-iron-emission-pool.md)
- [G17 Public Sync Allowlist](docs/public-sync/G17-PUBLIC-SYNC-CANDIDATE.md)
- [G18 区块预留 ADR](docs/adr/0017-g18-mining-block-reward-reservation.md)
- [G18 Devlog](docs/devlog/G18-mining-block-reward-reservation.md)
- [G18 Interaction Record](docs/interactions/G18-mining-block-reward-reservation.md)
- [G18 Public Sync Allowlist](docs/public-sync/G18-PUBLIC-SYNC-CANDIDATE.md)
- [G13–G15 时间持久化硬化 Devlog](docs/devlog/G13-G15-postgres-timestamp-hardening.md)
- [G13–G15 时间持久化硬化 Interaction Record](docs/interactions/G13-G15-postgres-timestamp-hardening.md)
- [G13–G15 时间持久化硬化公开同步清单](docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md)
- [Public Mirror Policy](PUBLIC-MIRROR-POLICY.md)
- [Security Policy](SECURITY.md)

### License

项目尚未选择 License。Source 可以公开访问，不代表除适用法律外自动授予额外再使用权利。

### 诚实边界

Fractal Legend 尚未公布公开上线日期。它目前不是 Production Service、Mainnet Integration、完成版 Browser MMORPG 或投资产品。公开宣传画表达产品方向，不能作为 Gameplay 实现证据。
