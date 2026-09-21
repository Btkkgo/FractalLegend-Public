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
| `apps/game-server/internal/trade/` | Fractal Legend project-owned G10 implementation | Server-authoritative item Trade Domain, repository contract, audit model, and synthetic tests | Standard library plus project persistence package |
| `apps/game-server/internal/ledger/` | Fractal Legend project-owned G11 implementation | Server-authoritative FB Ledger Domain, repository contract, conservation rules, and synthetic tests | Standard library only |
| `apps/game-server/internal/persistence/postgres/ledger*.go` and `migrations/0003_fb_ledger.sql` | Fractal Legend project-owned G11 implementation | PostgreSQL Ledger adapter, integration/schema tests, and DDL; no database content | Uses audited Go modules listed below |
| `docs/public/` | Fractal Legend public documentation | Sanitized bilingual project overview, status, roadmap, architecture, history, and approved media | No runtime dependency |
| `docs/adr/0009-g10-trade-foundation.md`, `docs/adr/0010-g11-fb-ledger-foundation.md` | Fractal Legend project-owned ADRs | Bilingual accepted decision records with private environment generalized | No runtime dependency |
| `docs/devlog/G9-persistence-foundation.md` through `docs/devlog/G11-fb-ledger-foundation.md` | Fractal Legend project-owned Devlogs | Bilingual technical results; private links and paths removed or generalized | No runtime dependency |
| `docs/interactions/` | Fractal Legend curated engineering records | Bilingual decision summaries, not raw private conversations | No runtime dependency |
| `docs/social/screenshots/` | Fractal Legend technical screenshots | Approved, sanitized validation screenshots; no account, credential, or local path visible | Image assets only |
| `docs/public/assets/marketing/` | Owner-approved Fractal Legend marketing asset | Approved launch poster with recorded SHA-256 | Image asset only |

The three `.sql` files under `internal/persistence/postgres/migrations/` are project-owned schema migrations. They contain DDL only and are not database dumps or player data.

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
| `apps/game-server/internal/trade/` | Fractal Legend 项目自有 G10 实现 | Server-authoritative Item Trade Domain、Repository Contract、Audit Model 与 Synthetic Test | Standard Library 加项目 Persistence Package |
| `apps/game-server/internal/ledger/` | Fractal Legend 项目自有 G11 实现 | Server-authoritative FB Ledger Domain、Repository Contract、Conservation Rule 与 Synthetic Test | 仅 Standard Library |
| `apps/game-server/internal/persistence/postgres/ledger*.go` 与 `migrations/0003_fb_ledger.sql` | Fractal Legend 项目自有 G11 实现 | PostgreSQL Ledger Adapter、Integration/Schema Test 与 DDL；不含 Database Content | 使用下方已审计 Go Module |
| `docs/public/` | Fractal Legend 公开文档 | 已脱敏的双语 Project Overview、Status、Roadmap、Architecture、History 与已批准 Media | 无 Runtime Dependency |
| `docs/adr/0009-g10-trade-foundation.md`、`docs/adr/0010-g11-fb-ledger-foundation.md` | Fractal Legend 项目自有 ADR | 双语已验收 Decision Record，Private Environment 已泛化 | 无 Runtime Dependency |
| `docs/devlog/G9-persistence-foundation.md` 至 `docs/devlog/G11-fb-ledger-foundation.md` | Fractal Legend 项目自有 Devlog | 双语技术结果；已删除或泛化 Private Link 与 Path | 无 Runtime Dependency |
| `docs/interactions/` | Fractal Legend 整理后的 Engineering Record | 双语 Decision Summary，不是完整私人对话 | 无 Runtime Dependency |
| `docs/social/screenshots/` | Fractal Legend Technical Screenshot | 已批准并脱敏；不显示 Account、Credential 或 Local Path | 仅 Image Asset |
| `docs/public/assets/marketing/` | 所有者批准的 Fractal Legend Marketing Asset | 已批准 Launch Poster，并记录 SHA-256 | 仅 Image Asset |

`internal/persistence/postgres/migrations/` 下的三个 `.sql` 文件是项目自有 Schema Migration，只包含 DDL，不是 Database Dump 或 Player Data。

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
