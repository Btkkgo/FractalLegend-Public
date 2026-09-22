# Public Code Provenance / 公开代码来源记录

## English — Primary

### Audit result

**Public code provenance: PASS for the files present in this repository.**

This mirror was assembled from an explicit allowlist into a new Git repository. No private `.git` directory, commit object, Legacy seller source, binary, protected fixture, Canonical export, or private database was copied.

### Published paths

| Public path | Ownership / origin | Why safe to publish | Third-party status |
|---|---|---|---|
| `apps/game-server/internal/ai/` | Fractal Legend project-owned implementation | Deterministic server-owned AI state model and synthetic tests; no Legacy implementation or private path | Standard library only |
| `apps/game-server/internal/characterstats/` | Fractal Legend project-owned implementation | Runtime character-stat calculation and synthetic tests | Standard library only |
| `apps/game-server/internal/combat/rules/` | Fractal Legend project-owned implementation | Versioned combat rule helpers and deterministic tests | Standard library only |
| `apps/game-server/internal/navigation/` | Fractal Legend project-owned implementation | Server-side movement rules and synthetic tests | Standard library only |
| `apps/game-server/internal/drops/registry.go` | Fractal Legend project-owned implementation | Generic runtime registry only; Legacy-derived sample data and fixtures are excluded | Standard library only |
| `apps/game-server/internal/equipment/registry.go` | Fractal Legend project-owned implementation | Generic equipment identity and modifier model; path-bearing fixtures are excluded | Standard library only |
| `apps/game-server/internal/items/registry.go` | Fractal Legend project-owned implementation | Generic item identity and instance model; path-bearing fixtures are excluded | Standard library only |
| `apps/game-server/internal/skills/registry.go` | Fractal Legend project-owned implementation | Generic skill registry and runtime override model; path-bearing fixtures are excluded | Standard library only |
| `apps/game-server/internal/persistence/` | Fractal Legend project-owned G9 implementation | Character Aggregate repository, PostgreSQL adapter, tests, and versioned schema migration | Uses audited Go modules listed below |
| `apps/game-server/internal/trade/` | Fractal Legend project-owned G10/G12 implementation | Server-authoritative item and FB Trade Domain, atomic settlement coordinator, repository contract, and synthetic tests | Standard library plus project persistence and Ledger packages |
| `apps/game-server/internal/ledger/` | Fractal Legend project-owned G11 implementation | Server-authoritative FB Ledger Domain, repository contract, conservation rules, and synthetic tests | Standard library only |
| `apps/game-server/internal/contribution/` | Fractal Legend project-owned G13 implementation | Versioned non-transferable eligibility policy, account and entry model, internal coordinator contract, and synthetic tests; no live producer | Standard library plus project Ledger model |
| `apps/game-server/internal/persistence/postgres/ledger*.go` and `migrations/0003_fb_ledger.sql` | Fractal Legend project-owned G11 implementation | PostgreSQL Ledger adapter, integration/schema tests, and DDL; no database content | Uses audited Go modules listed below |
| `apps/game-server/internal/persistence/postgres/trade.go`, `ledger_transaction.go`, `g12_trade_settlement_test.go`, and `migrations/0004_fb_trade_settlement.sql` | Fractal Legend project-owned G12 implementation | Shared-transaction Trade/Ledger adapter, synthetic integration tests, and DDL; no database content | Uses audited Go modules listed below |
| `apps/game-server/internal/persistence/postgres/contribution*.go`, G13 update to `ledger_transaction.go` and `store.go`, and `migrations/0005_contribution_ledger.sql` | Fractal Legend project-owned G13 implementation | Atomic FB/Contribution coordinator, refund fail-closed guard, real PostgreSQL synthetic integration tests, and schema constraints; no database content | Uses audited Go modules listed below |
| `apps/game-server/internal/contribution/recovery*.go`, `internal/persistence/postgres/contribution_refund*.go`, G14 update to `ledger_transaction.go`, and `migrations/0006_contribution_refund.sql` | Fractal Legend project-owned G14 implementation | Atomic refund recovery, synthetic tests, and DDL; no live producer, database content, or restricted asset | Uses audited Go modules listed below |
| `apps/game-server/internal/systemspend/`, `internal/persistence/postgres/system_spend*.go`, and `migrations/0007_system_spend.sql` | Fractal Legend project-owned G15 implementation | Internal eligible-spend orchestration, synthetic tests, and DDL; no live producer or database content | Uses audited Go modules listed below |
| `apps/game-server/internal/recycle/`, `internal/persistence/postgres/recycle*.go`, and `migrations/0008_recycle_foundation.sql` | Fractal Legend project-owned G16 implementation | Internal recycle domain, atomic PostgreSQL settlement, synthetic tests, immutable receipt, and DDL; no live gameplay entry or restricted asset | Uses audited Go modules listed below |
| `apps/game-server/internal/emission/`, `internal/persistence/postgres/emission*.go`, G17 changes to `contribution_refund.go`, `recycle_test.go`, `store.go`, and `migrations/0009_black_iron_emission_pool.sql` | Fractal Legend project-owned G17 implementation | Versioned global capacity, immutable entries and receipts, atomic refund compensation, migration and synthetic tests; no player ore or production ratio | Uses audited Go modules listed below |
| `apps/game-server/internal/persistence/postgres/{contribution.go,contribution_refund.go,ledger.go,system_spend.go,timestamp_replay_test.go}` | Fractal Legend project-owned G13–G15 reliability hardening | Canonical UTC microsecond persistence timestamps, exact PostgreSQL replay tests; no economic or schema change | Uses audited Go modules listed below |
| `docs/public/` | Fractal Legend public documentation | Sanitized bilingual project overview, status, roadmap, architecture, history, and approved media | No runtime dependency |
| `docs/adr/0009-g10-trade-foundation.md` through `docs/adr/0016-g17-black-iron-emission-pool.md` | Fractal Legend project-owned ADRs | Bilingual accepted decision records with private environment generalized | No runtime dependency |
| `docs/devlog/G9-persistence-foundation.md` through `docs/devlog/G17-black-iron-emission-pool.md` | Fractal Legend project-owned Devlogs | Bilingual technical results, including the G16 CI failure and repair; private links and paths removed or generalized | No runtime dependency |
| `docs/interactions/` | Fractal Legend curated engineering records | Bilingual decision summaries, not raw private conversations | No runtime dependency |
| `docs/devlog/G13-G15-postgres-timestamp-hardening.md`, `docs/interactions/G13-G15-postgres-timestamp-hardening.md`, and `docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md` | Fractal Legend project-owned Issue #18 records | Bilingual accepted results, curated interaction summary, and explicit export allowlist | No runtime dependency |
| `docs/social/screenshots/` | Fractal Legend technical screenshots | Approved, sanitized validation screenshots; no account, credential, or local path visible | Image assets only |
| `docs/public/assets/marketing/` | Owner-approved Fractal Legend marketing asset | Approved launch poster with recorded SHA-256 | Image asset only |

The nine `.sql` files under `internal/persistence/postgres/migrations/` are project-owned schema migrations. They contain DDL only and are not database dumps or player data.

### Go dependency audit

Dependencies are referenced through `go.mod` and `go.sum`; third-party source is not vendored.

| Module | Version | License found in module source | Source |
|---|---|---|---|
| `github.com/jackc/pgx/v5` | `v5.11.0` | MIT | [jackc/pgx](https://github.com/jackc/pgx) |
| `github.com/jackc/pgpassfile` | `v1.0.0` | MIT | [jackc/pgpassfile](https://github.com/jackc/pgpassfile) |
| `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | MIT | [jackc/pgservicefile](https://github.com/jackc/pgservicefile) |
| `github.com/jackc/puddle/v2` | `v2.2.2` | MIT | [jackc/puddle](https://github.com/jackc/puddle) |
| `golang.org/x/sync` | `v0.17.0` | BSD-3-Clause | [golang/sync](https://github.com/golang/sync) |
| `golang.org/x/text` | `v0.29.0` | BSD-3-Clause | [golang/text](https://github.com/golang/text) |

The audit read the license files distributed with the exact downloaded module versions. A Fractal Legend project license has not yet been selected; see the root README. Dependency licenses do not grant a project-level license for Fractal Legend source.

### Explicit exclusions

The initial mirror excludes the complete private runtime assembly, `foundation/` integration code, command entry points tied to private migration inputs, Gateway and browser pipelines that require further provenance review, importer and Canonical tooling, migrated data slices, protected fixtures, Legacy source and binaries, third-party game assets, private configuration, and local environment records.

---

## 中文 — 完整对应版本

### 审计结论

**本仓库现有文件的 Public Code Provenance：PASS。**

本镜像通过明确 Allowlist 导出到新的 Git Repository。没有复制 Private `.git` Directory、Commit Object、Legacy Seller Source、Binary、Protected Fixture、Canonical Export 或 Private Database。

### 已公开路径

| 公开路径 | Ownership / Origin | 可以安全公开的原因 | Third-party 状态 |
|---|---|---|---|
| `apps/game-server/internal/ai/` | Fractal Legend 项目自有实现 | 确定性的 Server-owned AI State Model 与 Synthetic Test；不含 Legacy Implementation 或 Private Path | 仅 Standard Library |
| `apps/game-server/internal/characterstats/` | Fractal Legend 项目自有实现 | Runtime Character-stat Calculation 与 Synthetic Test | 仅 Standard Library |
| `apps/game-server/internal/combat/rules/` | Fractal Legend 项目自有实现 | Versioned Combat Rule Helper 与 Deterministic Test | 仅 Standard Library |
| `apps/game-server/internal/navigation/` | Fractal Legend 项目自有实现 | Server-side Movement Rule 与 Synthetic Test | 仅 Standard Library |
| `apps/game-server/internal/drops/registry.go` | Fractal Legend 项目自有实现 | 仅 Generic Runtime Registry；排除 Legacy-derived Sample Data 与 Fixture | 仅 Standard Library |
| `apps/game-server/internal/equipment/registry.go` | Fractal Legend 项目自有实现 | Generic Equipment Identity 与 Modifier Model；排除包含路径的 Fixture | 仅 Standard Library |
| `apps/game-server/internal/items/registry.go` | Fractal Legend 项目自有实现 | Generic Item Identity 与 Instance Model；排除包含路径的 Fixture | 仅 Standard Library |
| `apps/game-server/internal/skills/registry.go` | Fractal Legend 项目自有实现 | Generic Skill Registry 与 Runtime Override Model；排除包含路径的 Fixture | 仅 Standard Library |
| `apps/game-server/internal/persistence/` | Fractal Legend 项目自有 G9 实现 | Character Aggregate Repository、PostgreSQL Adapter、Test 与 Versioned Schema Migration | 使用下方已审计 Go Module |
| `apps/game-server/internal/trade/` | Fractal Legend 项目自有 G10/G12 实现 | Server-authoritative Item 与 FB Trade Domain、原子 Settlement Coordinator、Repository Contract 与 Synthetic Test | Standard Library 加项目 Persistence 和 Ledger Package |
| `apps/game-server/internal/ledger/` | Fractal Legend 项目自有 G11 实现 | Server-authoritative FB Ledger Domain、Repository Contract、Conservation Rule 与 Synthetic Test | 仅 Standard Library |
| `apps/game-server/internal/contribution/` | Fractal Legend 项目自有 G13 实现 | 版本化不可转账资格规则、Account 与 Entry Model、内部 Coordinator Contract 和 Synthetic Test；无真实 Producer | Standard Library 加项目 Ledger Model |
| `apps/game-server/internal/persistence/postgres/ledger*.go` 与 `migrations/0003_fb_ledger.sql` | Fractal Legend 项目自有 G11 实现 | PostgreSQL Ledger Adapter、Integration/Schema Test 与 DDL；不含 Database Content | 使用下方已审计 Go Module |
| `apps/game-server/internal/persistence/postgres/trade.go`、`ledger_transaction.go`、`g12_trade_settlement_test.go` 与 `migrations/0004_fb_trade_settlement.sql` | Fractal Legend 项目自有 G12 实现 | 共用 Transaction 的 Trade/Ledger Adapter、Synthetic Integration Test 与 DDL；不含 Database Content | 使用下方已审计 Go Module |
| `apps/game-server/internal/persistence/postgres/contribution*.go`、G13 对 `ledger_transaction.go` 与 `store.go` 的更新，以及 `migrations/0005_contribution_ledger.sql` | Fractal Legend 项目自有 G13 实现 | 原子 FB/Contribution Coordinator、Refund Fail-closed Guard、真实 PostgreSQL Synthetic Integration Test 与 Schema Constraint；不含 Database Content | 使用下方已审计 Go Module |
| `apps/game-server/internal/contribution/recovery*.go`、`internal/persistence/postgres/contribution_refund*.go`、G14 对 `ledger_transaction.go` 的更新与 `migrations/0006_contribution_refund.sql` | Fractal Legend 项目自有 G14 实现 | 原子退款恢复、Synthetic Test 与 DDL；无真实 Producer、Database Content 或受限素材 | 使用下方已审计 Go Module |
| `apps/game-server/internal/systemspend/`、`internal/persistence/postgres/system_spend*.go` 与 `migrations/0007_system_spend.sql` | Fractal Legend 项目自有 G15 实现 | 内部合格消费编排、Synthetic Test 与 DDL；无真实 Producer 或 Database Content | 使用下方已审计 Go Module |
| `apps/game-server/internal/recycle/`、`internal/persistence/postgres/recycle*.go` 与 `migrations/0008_recycle_foundation.sql` | Fractal Legend 项目自有 G16 实现 | 内部回收 Domain、PostgreSQL 原子结算、Synthetic Test、不可变 Receipt 与 DDL；无真实玩法入口或受限素材 | 使用下方已审计 Go Module |
| `apps/game-server/internal/emission/`、`internal/persistence/postgres/emission*.go`、G17 对 `contribution_refund.go`、`recycle_test.go`、`store.go` 的修改及 `migrations/0009_black_iron_emission_pool.sql` | Fractal Legend 项目自有 G17 实现 | 版本化全服额度、不可变流水和回执、原子退款补偿、迁移及合成测试；无玩家矿石或正式比例 | 使用下方已审计 Go Module |
| `apps/game-server/internal/persistence/postgres/{contribution.go,contribution_refund.go,ledger.go,system_spend.go,timestamp_replay_test.go}` | Fractal Legend 项目自有 G13–G15 可靠性硬化 | UTC 微秒持久化时间规范化与 PostgreSQL 精确重放测试；经济及 Schema 不变 | 使用下方已审计 Go Module |
| `docs/public/` | Fractal Legend 公开文档 | 已脱敏的双语 Project Overview、Status、Roadmap、Architecture、History 与已批准 Media | 无 Runtime Dependency |
| `docs/adr/0009-g10-trade-foundation.md` 至 `docs/adr/0016-g17-black-iron-emission-pool.md` | Fractal Legend 项目自有 ADR | 双语已验收 Decision Record，Private Environment 已泛化 | 无 Runtime Dependency |
| `docs/devlog/G9-persistence-foundation.md` 至 `docs/devlog/G17-black-iron-emission-pool.md` | Fractal Legend 项目自有 Devlog | 双语技术结果，包括 G16 CI 故障及修复；已删除或泛化 Private Link 与 Path | 无 Runtime Dependency |
| `docs/interactions/` | Fractal Legend 整理后的 Engineering Record | 双语 Decision Summary，不是完整私人对话 | 无 Runtime Dependency |
| `docs/devlog/G13-G15-postgres-timestamp-hardening.md`、`docs/interactions/G13-G15-postgres-timestamp-hardening.md` 与 `docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md` | Fractal Legend 项目自有 Issue #18 记录 | 双语验收结果、整理后的互动摘要与明确公开清单 | 无 Runtime Dependency |
| `docs/social/screenshots/` | Fractal Legend Technical Screenshot | 已批准并脱敏；不显示 Account、Credential 或 Local Path | 仅 Image Asset |
| `docs/public/assets/marketing/` | 所有者批准的 Fractal Legend Marketing Asset | 已批准 Launch Poster，并记录 SHA-256 | 仅 Image Asset |

`internal/persistence/postgres/migrations/` 下的九个 `.sql` 文件是项目自有 Schema Migration，只包含 DDL，不是 Database Dump 或 Player Data。

### Go Dependency Audit

Dependency 通过 `go.mod` 与 `go.sum` 引用；没有 Vendor 第三方 Source。

| Module | Version | 在 Module Source 中确认的 License | Source |
|---|---|---|---|
| `github.com/jackc/pgx/v5` | `v5.11.0` | MIT | [jackc/pgx](https://github.com/jackc/pgx) |
| `github.com/jackc/pgpassfile` | `v1.0.0` | MIT | [jackc/pgpassfile](https://github.com/jackc/pgpassfile) |
| `github.com/jackc/pgservicefile` | `v0.0.0-20240606120523-5a60cdf6a761` | MIT | [jackc/pgservicefile](https://github.com/jackc/pgservicefile) |
| `github.com/jackc/puddle/v2` | `v2.2.2` | MIT | [jackc/puddle](https://github.com/jackc/puddle) |
| `golang.org/x/sync` | `v0.17.0` | BSD-3-Clause | [golang/sync](https://github.com/golang/sync) |
| `golang.org/x/text` | `v0.29.0` | BSD-3-Clause | [golang/text](https://github.com/golang/text) |

审计读取了这些准确下载版本随附的 License File。Fractal Legend 项目尚未选择 License，详见 Root README。Dependency License 不会自动成为 Fractal Legend Source 的项目级 License。

### 明确排除

初始镜像排除完整 Private Runtime Assembly、`foundation/` Integration Code、绑定 Private Migration Input 的 Command Entry Point、仍需进一步 Provenance Review 的 Gateway 与 Browser Pipeline、Importer 与 Canonical Tool、Migrated Data Slice、Protected Fixture、Legacy Source 与 Binary、Third-party Game Asset、Private Configuration 和 Local Environment Record。
