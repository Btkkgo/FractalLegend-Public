# G16 Public Mirror Sync Candidate

Status: **Manual acceptance PASS; sanitized export authorized after final safety gates**

## English — Primary

The approved G16 export uses this exact project-owned file allowlist. Review the image metadata and destination diff before committing. Public Mirror must not include private `.git` history, the G14 stash, legacy seller source, restricted assets, secrets, personal local paths, private email, raw databases, credentials, or real player data.

### G16 export allowlist

- `apps/game-server/internal/recycle/model.go`, `model_test.go`, `service.go`
- `apps/game-server/internal/persistence/postgres/recycle.go`, `recycle_reconcile.go`, `recycle_test.go`, `store.go`, `system_spend_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0008_recycle_foundation.sql`
- `docs/adr/0015-g16-recycle-migration-foundation.md`
- `docs/devlog/G16-recycle-migration-foundation.md`
- `docs/interactions/G16-recycle-migration-foundation.md`
- `docs/reports/G16-RECYCLE-ECONOMY-GRAPH.md`, `G16-RECYCLE-MIGRATION-TEST-REPORT.md`
- `docs/social/G16-MILESTONE-NOTES.md`, `screenshots/g16-recycle-migration-foundation.png`
- `docs/public-sync/G16-PUBLIC-SYNC-CANDIDATE.md`

Public overview/status/roadmap/provenance documents may be edited in the public repository to reflect G16 accurately. The private canonical CI workflow is excluded from this export because its integration fixture requires the complete private runtime and data layout. The private Issue URL and its comments are not Public Mirror artifacts. This allowlist does not authorize copying other private files or Git history.

## 中文 — 完整版本

已批准的 G16 导出使用以下精确的项目自有文件允许清单。Commit 前须审查图片元数据和目标 Diff。Public Mirror 不得包含私有 `.git` 历史、G14 stash、旧卖家源码、受限资源、密钥、个人本地路径、私人邮箱、原始数据库、凭据或真实玩家数据。

### G16 导出允许清单

- `apps/game-server/internal/recycle/model.go`、`model_test.go`、`service.go`
- `apps/game-server/internal/persistence/postgres/recycle.go`、`recycle_reconcile.go`、`recycle_test.go`、`store.go`、`system_spend_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0008_recycle_foundation.sql`
- `docs/adr/0015-g16-recycle-migration-foundation.md`
- `docs/devlog/G16-recycle-migration-foundation.md`
- `docs/interactions/G16-recycle-migration-foundation.md`
- `docs/reports/G16-RECYCLE-ECONOMY-GRAPH.md`、`G16-RECYCLE-MIGRATION-TEST-REPORT.md`
- `docs/social/G16-MILESTONE-NOTES.md`、`screenshots/g16-recycle-migration-foundation.png`
- `docs/public-sync/G16-PUBLIC-SYNC-CANDIDATE.md`

可在 Public Repository 中编辑公开 Overview、Status、Roadmap 与 Provenance 文档，使 G16 状态准确。Private Canonical CI Workflow 依赖完整私有 Runtime 与数据布局中的集成测试 Fixture，因此不在本次导出清单。Private Issue URL 及评论不是 Public Mirror 成果。本清单不授权复制其他私有文件或 Git History。
