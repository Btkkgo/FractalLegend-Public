# Public Mirror Development History / 公开镜像开发历史

## English — Primary

The sanitized public mirror began **after G10** with a new Git history. G1–G10 were developed and accepted in the private canonical development repository before this mirror was created.

The table below is a public-safe retrospective. It preserves accepted results and recorded test counts without pretending that this repository's commit history represents the original chronology.

| Gate | Name | Result | Recorded tests | Public-safe summary |
|---|---|---|---|---|
| G1 | First Playable World | PASS | Acceptance and three-platform standalone PASS; no separate count recorded | Server-authoritative movement, position, bounds, speed, and collision validation |
| G2 | Rendering Pipeline | PASS | Acceptance PASS; no separate count recorded | Browser-ready 2D rendering and deterministic Fractal override atlas pipeline |
| G3 | Living World | PASS | Acceptance and three-platform standalone PASS; no separate count recorded | Authoritative NPC and six-monster world snapshot and browser synchronization |
| G4 | Combat Sandbox | PASS | Acceptance and cross-platform builds PASS; no separate count recorded | Server-owned target selection, normal attack, damage, HP, and death |
| G5 | Basic Monster AI | PASS | Acceptance PASS; no separate count recorded | Deterministic aggro, chase, attack, and return state machine |
| G6 | Skills and Character Combat | PASS | Acceptance PASS; no separate count recorded | One Warrior active-damage skill with server-owned MP, cooldown, damage, and retry protection |
| G7 | Drop, Pickup, and Inventory | PASS | G1–G6 regression, Race, standalone, and cross-platform PASS; no separate count recorded | Death drop, atomic pickup, ownership, reconnect state, and 20-slot inventory |
| G8 | Equipment Runtime and Character Stats | PASS | Combined 456/456; Node 222; Go 234; Race and three-platform builds PASS | Item-instance equipment, runtime stat modifiers, combat integration, reconnect, and drift checks |
| G9 | Persistence Foundation | PASS | Persistence 25/25; Node 222/222; Go 261/261; Combined 483/483; Race, PostgreSQL integration, restart, and three-platform builds PASS | Transactional PostgreSQL Character Aggregate with optimistic revision and restart restore |
| G10 | Trade Foundation | CLOSED / PASS | Historical Node 224/224; Trade 29/29; Go 290/290; Windows fixtures 2/2; cross-platform PASS | Item-only, server-authoritative trade with persistent locks and atomic settlement |

Private canonical archive references retained for internal verification:

- G9 documentation acceptance: `06aecadb0026cf538ef589d2ddfdaf9aea5aab67`
- G10 acceptance: `b47f7db006bf121f194bfb82808c3864c1c5f90e`
- Public overview prepared before mirror creation: `a4dae045f695904be8f564be14d42ccd33dc2bdf`
- Private overview merge: `97215c7e87c317fc28b94e1ecaf73d334748f291`

These are plain references to the private canonical archive. They are not public links and are not commits in this mirror.

After mirror creation, G11 FB Ledger and G12 atomic item + FB Trade settlement were accepted and exported as separate sanitized public commits. G11 recorded 44/44 targeted checks, 11/11 real PostgreSQL checks, and 334/334 full Go regression. G12 recorded 45/45 targeted checks, 12/12 real PostgreSQL checks, 379/379 full Go regression, Race, and three-platform builds. See their bilingual [Devlogs](../devlog/) and ADRs for scope and limits. These public commits are sanitized export events, not the original private development chronology.

---

## 中文 — 完整对应版本

Sanitized Public Mirror 在 **G10 之后**以全新 Git History 建立。G1–G10 在本镜像创建前，已经在 Private Canonical Development Repository 中完成开发与验收。

下表是 Public-safe Retrospective。它保留已验收结果与记录的 Test Count，但不会把本仓库 Commit History 伪装成原始开发时间线。

| Gate | 名称 | 结果 | 已记录测试 | Public-safe 摘要 |
|---|---|---|---|---|
| G1 | First Playable World | PASS | Acceptance 与三平台 Standalone PASS；未单独记录计数 | Server-authoritative Movement、Position、Bounds、Speed 与 Collision Validation |
| G2 | Rendering Pipeline | PASS | Acceptance PASS；未单独记录计数 | Browser-ready 2D Rendering 与确定性的 Fractal Override Atlas Pipeline |
| G3 | Living World | PASS | Acceptance 与三平台 Standalone PASS；未单独记录计数 | 权威 NPC、六只怪物 World Snapshot 与 Browser Synchronization |
| G4 | Combat Sandbox | PASS | Acceptance 与 Cross-platform Build PASS；未单独记录计数 | Server-owned Target Selection、Normal Attack、Damage、HP 与 Death |
| G5 | Basic Monster AI | PASS | Acceptance PASS；未单独记录计数 | 确定性的 Aggro、Chase、Attack 与 Return State Machine |
| G6 | Skills and Character Combat | PASS | Acceptance PASS；未单独记录计数 | 一项 Warrior Active-damage Skill，MP、Cooldown、Damage 与 Retry Protection 由服务器拥有 |
| G7 | Drop, Pickup, and Inventory | PASS | G1–G6 Regression、Race、Standalone 与 Cross-platform PASS；未单独记录计数 | Death Drop、Atomic Pickup、Ownership、Reconnect State 与 20-slot Inventory |
| G8 | Equipment Runtime and Character Stats | PASS | Combined 456/456；Node 222；Go 234；Race 与三平台 Build PASS | Item-instance Equipment、Runtime Stat Modifier、Combat Integration、Reconnect 与 Drift Check |
| G9 | Persistence Foundation | PASS | Persistence 25/25；Node 222/222；Go 261/261；Combined 483/483；Race、PostgreSQL Integration、Restart 与三平台 Build PASS | Transactional PostgreSQL Character Aggregate、Optimistic Revision 与 Restart Restore |
| G10 | Trade Foundation | CLOSED / PASS | Historical Node 224/224；Trade 29/29；Go 290/290；Windows Fixture 2/2；Cross-platform PASS | 仅 Item、Server-authoritative Trade，包含 Persistent Lock 与 Atomic Settlement |

为内部验证保留的 Private Canonical Archive Reference：

- G9 Documentation Acceptance：`06aecadb0026cf538ef589d2ddfdaf9aea5aab67`
- G10 Acceptance：`b47f7db006bf121f194bfb82808c3864c1c5f90e`
- Public Mirror 建立前准备的 Public Overview：`a4dae045f695904be8f564be14d42ccd33dc2bdf`
- Private Overview Merge：`97215c7e87c317fc28b94e1ecaf73d334748f291`

这些只是 Private Canonical Archive 的纯文本 Reference，不是 Public Link，也不是本镜像中的 Commit。

镜像建立后，G11 FB Ledger 与 G12 原子 Item + FB Trade Settlement 分别通过验收，并以独立的脱敏公开 Commit 导出。G11 记录专项检查 44/44、真实 PostgreSQL 检查 11/11、Go 完整回归 334/334。G12 记录专项检查 45/45、真实 PostgreSQL 检查 12/12、Go 完整回归 379/379、Race 与三平台 Build。具体范围与限制详见双语 [Devlog](../devlog/) 和 ADR。这些 Public Commit 是脱敏导出事件，不是原始 Private Development Chronology。
