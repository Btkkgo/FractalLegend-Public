# Current Status / 当前状态

## English — Primary

Updated for the sanitized public mirror on 2026-09-21. “Foundation Complete” means an accepted technical milestone, not a released product.

| Area | Status | Evidence / boundary |
|---|---|---|
| G1 World | **FOUNDATION COMPLETE** | Server-authoritative movement and collision slice |
| G2 Rendering | **FOUNDATION COMPLETE** | Browser-ready 2D atlas pipeline using Fractal override visuals |
| G3 Living World | **FOUNDATION COMPLETE** | NPC and six monsters synchronized; NPC interaction absent |
| G4 Combat | **FOUNDATION COMPLETE** | Target, normal attack, server damage, and death |
| G5 Monster AI | **FOUNDATION COMPLETE** | Deterministic state machine, chase, attack, and return |
| G6 Skills | **FOUNDATION COMPLETE** | Warrior 烈火剑法 only |
| G7 Loot / Inventory | **FOUNDATION COMPLETE** | Death drop, pickup, ownership, and 20-slot inventory |
| G8 Equipment / Stats | **FOUNDATION COMPLETE** | Equip/unequip, RuntimeStats, reconnect, and drift/concurrency tests |
| G9 Persistence | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate and full Game Server restart restore |
| G10 Trade | **FOUNDATION COMPLETE** | Item-trade domain, extended by G12; UI and markets absent |
| G11 FB Ledger | **FOUNDATION COMPLETE** | Internal server-authoritative Ledger; no wallet or blockchain adapter |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | Atomic PostgreSQL settlement with 0% fee; no Marketplace or Trade UI |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | Internal non-transferable ledger; 1:1 eligible system spend; no live producer |
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | Atomic FB + Contribution recovery, bounded partial refunds, debt and hold; no live producer |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted slices do not form a public release |
| Multi-class gameplay | **PLANNED** | Warrior is the only confirmed Web runtime class |
| Boss / Party / Team Dungeon | **PLANNED** | No accepted implementation |
| Guild / Siege | **PLANNED** | No accepted implementation |
| Wallet / blockchain | **PLANNED** | No live deposit, withdrawal, or chain adapter |
| Ordinals activation | **PLANNED** | No production verification or activation |
| Mining / Reputation | **PLANNED** | Design direction only |
| Social / voice / video / feed | **PLANNED / RESEARCH** | No accepted implementation |

### Repository topology

The private canonical repository preserves complete internal development and acceptance history. This sanitized public mirror began after G10 with a new Git history and an audited subset of project-owned source, tests, and documentation. The public history is not the original G1–G10 chronology.

### Latest closed milestone

G14 technical and manual acceptance are PASS. Private checks passed 27/27 G14 focused tests (21/21 real PostgreSQL), two 200-iteration property sequences, 100-way duplicate and partial refund concurrency, full Go and Race 447/447 each, Vet, and Windows/Linux/macOS builds. Node remains **95/96 — KNOWN PRE-EXISTING G2 LIMITATION** because the G2 Atlas test requires a restricted Legacy archive. G14 closes the internal G13-linked refund/reversal compensation hard gate with atomic FB + Contribution posting, recovery debt, future-credit debt repayment, and review hold. Ordinary Ledger refunds remain fail-closed. There is still no real eligible system-spend producer or production Contribution spending.

### Release status

There is no public launch date, production deployment, mainnet economy, public Trade UI, or complete game release.

---

## 中文 — 完整对应版本

本页按 2026-09-21 建立的 Sanitized Public Mirror 更新。“Foundation Complete”表示技术里程碑已验收，不表示产品已经发布。

| 领域 | 状态 | 证据 / 边界 |
|---|---|---|
| G1 World | **FOUNDATION COMPLETE** | Server-authoritative Movement 与 Collision Slice |
| G2 Rendering | **FOUNDATION COMPLETE** | 使用 Fractal Override Visual 的 Browser-ready 2D Atlas Pipeline |
| G3 Living World | **FOUNDATION COMPLETE** | 同步 NPC 与六只怪物；不含 NPC Interaction |
| G4 Combat | **FOUNDATION COMPLETE** | Target、Normal Attack、Server Damage 与 Death |
| G5 Monster AI | **FOUNDATION COMPLETE** | 确定性 State Machine、Chase、Attack 与 Return |
| G6 Skills | **FOUNDATION COMPLETE** | 仅 Warrior 烈火剑法 |
| G7 Loot / Inventory | **FOUNDATION COMPLETE** | Death Drop、Pickup、Ownership 与 20-slot Inventory |
| G8 Equipment / Stats | **FOUNDATION COMPLETE** | Equip/Unequip、RuntimeStats、Reconnect 与 Drift/Concurrency Test |
| G9 Persistence | **FOUNDATION COMPLETE** | PostgreSQL Character Aggregate 与完整 Game Server Restart Restore |
| G10 Trade | **FOUNDATION COMPLETE** | Item Trade Domain，由 G12 扩展；不含 UI 与 Market |
| G11 FB Ledger | **FOUNDATION COMPLETE** | 内部 Server-authoritative Ledger；不含 Wallet 或 Blockchain Adapter |
| G12 Item + FB Trade Settlement | **FOUNDATION COMPLETE** | PostgreSQL 原子结算、0% 手续费；不含 Marketplace 或 Trade UI |
| G13 Contribution Ledger | **FOUNDATION COMPLETE** | 内部不可转账账本；合格 System Spend 1:1；尚无真实 Producer |
| G14 Refund / Reversal Compensation | **FOUNDATION COMPLETE** | 原子 FB + Contribution 恢复、部分退款上限、债务和 Hold；尚无真实 Producer |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Slice 尚未组成公开 Release |
| Multi-class Gameplay | **PLANNED** | Warrior 是唯一确认的 Web Runtime Class |
| Boss / Party / Team Dungeon | **PLANNED** | 没有已验收实现 |
| Guild / Siege | **PLANNED** | 没有已验收实现 |
| Wallet / Blockchain | **PLANNED** | 没有 Live Deposit、Withdrawal 或 Chain Adapter |
| Ordinals Activation | **PLANNED** | 没有 Production Verification 或 Activation |
| Mining / Reputation | **PLANNED** | 仅为设计方向 |
| Social / Voice / Video / Feed | **PLANNED / RESEARCH** | 没有已验收实现 |

### Repository Topology

Private Canonical Repository 保留完整内部开发与验收历史。本 Sanitized Public Mirror 在 G10 之后以全新 Git History 建立，仅包含经过审计的项目自有 Source、Test 与 Documentation 子集。Public History 不是原始 G1–G10 Chronology。

### 最新关闭里程碑

G14 技术验收与人工验收均已 PASS。Private Check 包括 27/27 G14 专项测试（其中 21/21 真实 PostgreSQL）、两组各 200 轮 Property、100 并发重复与部分退款，以及各 447/447 的 Go 全量与 Race、Vet 和 Windows/Linux/macOS Build。Node 仍为 **95/96 — KNOWN PRE-EXISTING G2 LIMITATION**，因为 G2 Atlas Test 需要受限制 Legacy Archive。G14 通过原子 FB + Contribution 入账、Recovery Debt、未来 Credit 先偿债及 Review Hold，关闭了内部 G13 关联退款／冲正补偿硬门。普通 Ledger 退款仍默认拒绝。真实合格 System Spend Producer 与生产用 Contribution 消费仍未接入。

### Release 状态

项目没有公开上线日期、Production Deployment、Mainnet Economy、公开 Trade UI 或完整 Game Release。
