# Current Status / 当前状态

## English — Primary

Updated for the sanitized public mirror on 2026-09-22. “Foundation Complete” means an accepted technical milestone, not a released product.

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
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | Internal orchestration; no live gameplay producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | Internal material + Reputation settlement with immutable receipt; production rules and gameplay entry disabled |
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | Versioned immutable global capacity; no player ore or production ratio |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | Atomic block reward reservation, recovery debt, immutable replay; no distribution |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted slices do not form a public release |
| Multi-class gameplay | **PLANNED** | Warrior is the only confirmed Web runtime class |
| Boss / Party / Team Dungeon | **PLANNED** | No accepted implementation |
| Guild / Siege | **PLANNED** | No accepted implementation |
| Wallet / blockchain | **PLANNED** | No live deposit, withdrawal, or chain adapter |
| Ordinals activation | **PLANNED** | No production verification or activation |
| Player mining and reward distribution | **PLANNED** | G18 reserves block capacity only; no miners, power, tools, maps, or player ore |
| Social / voice / video / feed | **PLANNED / RESEARCH** | No accepted implementation |

### Repository topology

The private canonical repository preserves complete internal development and acceptance history. This sanitized public mirror began after G10 with a new Git history and an audited subset of project-owned source, tests, and documentation. The public history is not the original G1–G10 chronology.

### Latest closed milestone

G18 technical and manual acceptance are PASS. The accepted private PR and post-merge canonical CI each passed 6/6 jobs: Go Test 556/556, Go Race 556/556, zero skipped, Vet, Linux/Windows/macOS builds, and real independent-process Crash A–E. G18 reserves block rewards against the G17 authoritative pool. Upstream refund shortfalls create Recovery Debt; future emission and cancellation repay it first. Reconciliation checks `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`. **Distributed is zero; capacity and reservation are not player ore.** The development block rule is not a production reward or duration. No miners, mining power, tools, maps, reward allocation, player ore, or Bun migration exist.

### Release status

There is no public launch date, production deployment, mainnet economy, public Trade UI, or complete game release.

---

## 中文 — 完整对应版本

本页按 2026-09-22 的 Sanitized Public Mirror 状态更新。“Foundation Complete”表示技术里程碑已验收，不表示产品已经发布。

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
| G15 Eligible System Spend | **FOUNDATION COMPLETE** | 内部消费编排；尚无真实玩法 Producer |
| G16 Recycle Migration | **FOUNDATION COMPLETE** | 内部材料与 Reputation 结算及不可变 Receipt；生产规则和玩法入口禁用 |
| G17 Black Iron Emission Pool | **FOUNDATION COMPLETE** | 版本化、不可变的全服额度；不发放玩家矿石，正式比例未定 |
| G18 Mining Block Reservation | **FOUNDATION COMPLETE** | 原子区块奖励预留、恢复债务及不可变重放；没有分发 |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Slice 尚未组成公开 Release |
| Multi-class Gameplay | **PLANNED** | Warrior 是唯一确认的 Web Runtime Class |
| Boss / Party / Team Dungeon | **PLANNED** | 没有已验收实现 |
| Guild / Siege | **PLANNED** | 没有已验收实现 |
| Wallet / Blockchain | **PLANNED** | 没有 Live Deposit、Withdrawal 或 Chain Adapter |
| Ordinals Activation | **PLANNED** | 没有 Production Verification 或 Activation |
| 玩家挖矿与奖励分配 | **PLANNED** | G18 只预留区块容量；没有矿工、算力、工具、地图或玩家矿石 |
| Social / Voice / Video / Feed | **PLANNED / RESEARCH** | 没有已验收实现 |

### Repository Topology

Private Canonical Repository 保留完整内部开发与验收历史。本 Sanitized Public Mirror 在 G10 之后以全新 Git History 建立，仅包含经过审计的项目自有 Source、Test 与 Documentation 子集。Public History 不是原始 G1–G10 Chronology。

### 最新关闭里程碑

G18 技术与人工验收均为 PASS。获批的私有 PR 和合并后的 canonical CI 各有 6/6 作业通过：Go Test 556/556、Go Race 556/556、跳过 0、Vet、Linux/Windows/macOS 构建和真实独立进程 Crash A–E。G18 从 G17 权威池预留区块奖励；上游退款的缺口形成恢复债务，未来发行和取消释放先偿债。对账检查 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`。**已分发为零；额度和预留均不是玩家矿石。**开发区块规则不代表正式奖励或时长。没有矿工、挖矿算力、工具、地图、奖励分配、玩家矿石或馒头迁移。

### Release 状态

项目没有公开上线日期、Production Deployment、Mainnet Economy、公开 Trade UI 或完整 Game Release。
