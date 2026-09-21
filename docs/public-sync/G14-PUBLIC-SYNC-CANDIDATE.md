# G14 Public Mirror Sync Candidate

Status: **Manual acceptance PASS · Stage Close export authorized, subject to final safety gates**

## English — Primary

This allowlist identifies G14-owned files approved for Stage Close Public Mirror sync after manual acceptance PASS. Export is authorized only after fresh provenance, privacy, restricted-asset, secret, link, and destination checks. The Public Mirror keeps independent Git history; X publication remains unauthorized.

### Candidate allowlist

- `apps/game-server/internal/contribution/model.go`
- `apps/game-server/internal/contribution/recovery.go`
- `apps/game-server/internal/contribution/recovery_test.go`
- `apps/game-server/internal/contribution/service.go`
- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go`
- `apps/game-server/internal/persistence/postgres/migrations/0006_contribution_refund.sql`
- `docs/adr/0013-g14-contribution-refund-reversal.md`
- `docs/devlog/G14-contribution-refund-reversal.md`
- `docs/interactions/G14-contribution-refund-reversal.md`
- `docs/social/X-DEVELOPMENT-LOG.md` (G14 bilingual draft only)
- `docs/public-sync/G14-PUBLIC-SYNC-CANDIDATE.md`

### Exclusions and next gate

Exclude private Git history, private Issue/PR links, personal account or email, local absolute path, secret, token, password, DSN, local database, raw test log, Legacy seller source, restricted binary/asset, and all unrelated files. There is no G14 screenshot artifact in this candidate. Before export, review exact diffs and provenance again, run fresh scans, and compare the destination. X publication remains manual and unauthorized.

---

## 中文 — 完整对应版本

状态：**人工验收 PASS · 已授权 Stage Close 导出，但须通过最终安全检查**

本允许清单标识 G14 项目自有且在人工验收 PASS 后获准用于 Stage Close Public Mirror Sync 的文件。只有重新通过来源、隐私、受限素材、秘密、链接和目标仓库检查，才可导出。Public Mirror 保持独立 Git History；X 发布仍未获授权。

### 候选允许清单

- `apps/game-server/internal/contribution/model.go`
- `apps/game-server/internal/contribution/recovery.go`
- `apps/game-server/internal/contribution/recovery_test.go`
- `apps/game-server/internal/contribution/service.go`
- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go`
- `apps/game-server/internal/persistence/postgres/migrations/0006_contribution_refund.sql`
- `docs/adr/0013-g14-contribution-refund-reversal.md`
- `docs/devlog/G14-contribution-refund-reversal.md`
- `docs/interactions/G14-contribution-refund-reversal.md`
- `docs/social/X-DEVELOPMENT-LOG.md`（仅 G14 双语草稿）
- `docs/public-sync/G14-PUBLIC-SYNC-CANDIDATE.md`

### 排除内容与下一道 Gate

排除 Private Git History、Private Issue/PR Link、个人账号或 Email、本地绝对路径、Secret、Token、Password、DSN、本地数据库、原始测试日志、Legacy Seller Source、受限制 Binary/Asset，以及所有无关文件。本候选中没有 G14 截图素材。导出前必须重新审查准确 Diff 和来源、执行新一轮扫描，并比较目标仓库。X 发布仍由用户手动执行，本轮未获授权。
