# G13–G15 Timestamp Hardening Public Sync Candidate

Status: **Private technical and manual acceptance PASS; sanitized export authorized after fresh public gates**

## English — Primary

This explicit allowlist covers only project-owned timestamp-boundary changes and curated evidence for the G13–G15 reliability follow-up. The private canonical merge completed first. The public mirror retains its independent Git history.

### Export allowlist

- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/system_spend.go`
- `apps/game-server/internal/persistence/postgres/timestamp_replay_test.go`
- `docs/devlog/G13-G15-postgres-timestamp-hardening.md`
- `docs/interactions/G13-G15-postgres-timestamp-hardening.md`
- `docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md`

The public repository's README and code-provenance index may be updated only to describe these reviewed files. No private Git objects, issue comments, pull-request links, local test databases, raw logs, credentials, production configuration, Legacy seller source, protected fixtures, or restricted third-party assets are part of this export. There are no binary or image candidates. Run fresh secret, personal-information, local-path, legacy-source, restricted-asset, dependency, license, link, and destination-diff reviews before push. Verify a clean PostgreSQL normal and Race suite with zero skipped tests in the public subset. X publication and G17 implementation remain outside this sync.

## 中文 — 完整审核版

本明确允许清单仅包含 G13–G15 可靠性后续修复中项目自有的时间持久化边界改动与整理后的证据。Private Canonical 已先完成合并；Public Mirror 继续保持独立 Git History。

### 导出允许清单

- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/system_spend.go`
- `apps/game-server/internal/persistence/postgres/timestamp_replay_test.go`
- `docs/devlog/G13-G15-postgres-timestamp-hardening.md`
- `docs/interactions/G13-G15-postgres-timestamp-hardening.md`
- `docs/public-sync/G13-G15-TIMESTAMP-PUBLIC-SYNC-CANDIDATE.md`

Public Repository 的 README 与 Code Provenance Index 仅可为说明这些已审核文件而更新。不得导出 Private Git Object、Issue Comment、Pull Request Link、本地测试数据库、原始 Log、凭据、生产配置、Legacy Seller Source、受保护 Fixture 或受限制第三方素材。本次没有 Binary 或 Image 候选。Push 前须重新执行 Secret、个人信息、本地路径、Legacy Source、受限素材、依赖、License、Link 与目标 Diff 审核。Public 子集必须使用干净 PostgreSQL 数据库跑普通与 Race 完整测试，跳过数为零。X 发布与 G17 实现不在本次同步范围。
