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
| G10 Trade | **FOUNDATION COMPLETE** | Item-only domain; UI, FB settlement, and markets absent |
| G11 FB Ledger | **NEXT** | No G11 implementation is included or claimed in this public mirror |
| Complete browser MMORPG | **IN DEVELOPMENT** | Accepted slices do not form a public release |
| Multi-class gameplay | **PLANNED** | Warrior is the only confirmed Web runtime class |
| Boss / Party / Team Dungeon | **PLANNED** | No accepted implementation |
| Guild / Siege | **PLANNED** | No accepted implementation |
| Wallet / blockchain | **PLANNED** | No live deposit, withdrawal, or chain adapter |
| Ordinals activation | **PLANNED** | No production verification or activation |
| Mining / Contribution / Reputation | **PLANNED** | Design direction only |
| Social / voice / video / feed | **PLANNED / RESEARCH** | No accepted implementation |

### Repository topology

The private canonical repository preserves complete internal development and acceptance history. This sanitized public mirror began after G10 with a new Git history and an audited subset of project-owned source, tests, and documentation. The public history is not the original G1–G10 chronology.

### Latest closed milestone

G10 is CLOSED / PASS. Historical Node 224/224, Trade 29/29, Go 290/290, and two Windows process fixtures passed in the recorded private acceptance process.

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
| G10 Trade | **FOUNDATION COMPLETE** | 仅 Item Domain；不含 UI、FB Settlement 与 Market |
| G11 FB Ledger | **NEXT** | 本 Public Mirror 不包含或声称任何 G11 Implementation |
| 完整 Browser MMORPG | **IN DEVELOPMENT** | 已验收 Slice 尚未组成公开 Release |
| Multi-class Gameplay | **PLANNED** | Warrior 是唯一确认的 Web Runtime Class |
| Boss / Party / Team Dungeon | **PLANNED** | 没有已验收实现 |
| Guild / Siege | **PLANNED** | 没有已验收实现 |
| Wallet / Blockchain | **PLANNED** | 没有 Live Deposit、Withdrawal 或 Chain Adapter |
| Ordinals Activation | **PLANNED** | 没有 Production Verification 或 Activation |
| Mining / Contribution / Reputation | **PLANNED** | 仅为设计方向 |
| Social / Voice / Video / Feed | **PLANNED / RESEARCH** | 没有已验收实现 |

### Repository Topology

Private Canonical Repository 保留完整内部开发与验收历史。本 Sanitized Public Mirror 在 G10 之后以全新 Git History 建立，仅包含经过审计的项目自有 Source、Test 与 Documentation 子集。Public History 不是原始 G1–G10 Chronology。

### 最新关闭里程碑

G10 状态为 CLOSED / PASS。记录的 Private Acceptance Process 通过 Historical Node 224/224、Trade 29/29、Go 290/290 和两项 Windows Process Fixture。

### Release 状态

项目没有公开上线日期、Production Deployment、Mainnet Economy、公开 Trade UI 或完整 Game Release。
