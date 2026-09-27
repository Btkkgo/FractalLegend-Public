# G19 — Public Export Allowlist

## English — Primary

Independent public history; project-owned Go/SQL and curated bilingual Markdown only. No private Git objects, raw seller source, protected fixtures, audit CSV, private tools, generated provenance, database dumps, production data, credentials or new binary assets.

Code allowlist (byte-identical to accepted canonical):

- `apps/game-server/internal/blackiron/model.go`
- `apps/game-server/internal/blackiron/model_test.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration_read.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration_test.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration_crash_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0011_black_iron_inventory_migration.sql`
- `apps/game-server/internal/persistence/postgres/store.go`
- `apps/game-server/internal/persistence/postgres/mining_block_test.go`
- `apps/game-server/internal/persistence/postgres/recycle_test.go`

Curated documents:

- `docs/adr/0018-g19-bun-black-iron-migration.md`
- `docs/devlog/G19-bun-black-iron-migration.md`
- `docs/interactions/G19-bun-black-iron-migration.md`
- `docs/reports/G19-BUN-AUDIT.md`
- `docs/reports/G19-MIGRATION-TEST-REPORT.md`
- `docs/public-sync/G19-PUBLIC-SYNC-CANDIDATE.md`
- `docs/social/G19-x-draft.md`
- `docs/social/G19-status-card-brief.md`

README, PUBLIC-CODE-PROVENANCE and current public status/architecture/roadmap/history/economy/overview pages are updated for G19. Historical milestone records remain historical. No dependency or licensing change; third-party source is not vendored. Existing approved PNG assets are unchanged and hash-checked against the prior public commit. Verified public results appear below.

## 中文 — 完整审核版

公开历史独立；仅项目自有 Go／SQL 和整理的双语 Markdown。不复制私有 Git 对象、卖家原始源码、保护 fixture、审计 CSV、私有工具、生成来源、数据库 dump、生产数据、凭据或新增二进制素材。

代码允许清单与已验收 canonical 逐字节相同，上方列明全部路径；整理文档允许清单也逐项列出。README、PUBLIC-CODE-PROVENANCE 和当前公开状态／架构／路线／历史／经济／概览更新为 G19，历史阶段记录仍保留历史含义。没有依赖或许可变化，不 vendor 第三方源码。既有批准 PNG 与此前公开提交逐一校验哈希且不改变，已验证的公开结果见下方。


## Public verification / 公开验证

Fresh sequential PostgreSQL test databases: Go Test **383/383 PASS**, Go Race **383/383 PASS**, failed **0**, skipped **0**; Go Vet **PASS**. All **173** tracked/nonignored files passed Secret, Personal Information, Legacy Source, Restricted Third-party Asset and local-link checks. Ten code paths are byte-identical to accepted canonical; nine approved baseline PNG files retain exact SHA-256 hashes. No new binaries, archives, oversized files, dependency or license changes. Pattern scans are complemented by explicit allowlist, source/diff and provenance review. Publication uses the account noreply identity and independent public Git history.

顺序使用各自全新 PostgreSQL 测试库：Go Test **383/383 PASS**、Go Race **383/383 PASS**、失败 **0**、跳过 **0**，Go Vet **PASS**。全部 **173** 个受追踪／非忽略文件通过秘密、个人信息、旧源码、受限素材及本地链接检查。十个代码路径与已验收 canonical 逐字节相同，九个既有批准 PNG 保持精确 SHA-256 哈希。没有新二进制、归档、超大文件、依赖或许可变化；模式扫描配合明确允许清单、源码／Diff 和来源审查。发布使用账号 noreply 身份，保留独立公开 Git 历史。
