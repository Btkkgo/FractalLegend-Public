# G13 Public Sync Candidate Manifest

Status: **STAGE CLOSE ALLOWLIST · Manual acceptance PASS · Export authorized subject to fresh safety review**

## English — Primary

This manifest identifies G13-owned files for the separately authorized Stage Close Public Mirror sync. Manual acceptance is PASS. Each file still requires a fresh provenance, privacy, restricted-asset, and secret review before copying. The Public Mirror retains independent Git history; X publication and G14 remain unauthorized.

### Candidate allowlist

- `apps/game-server/internal/contribution/model.go`
- `apps/game-server/internal/contribution/policy.go`
- `apps/game-server/internal/contribution/policy_test.go`
- `apps/game-server/internal/contribution/service.go`
- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go` (G13-linked refund/reversal guard only)
- `apps/game-server/internal/persistence/postgres/store.go` (failure-injection test hook only)
- `apps/game-server/internal/persistence/postgres/migrations/0005_contribution_ledger.sql`
- `docs/adr/0012-g13-contribution-ledger-foundation.md`
- `docs/devlog/G13-contribution-ledger-foundation.md`
- `docs/interactions/G13-contribution-ledger-foundation.md`
- `docs/social/X-DEVELOPMENT-LOG.md` (G13 draft and screenshot record)
- `docs/social/screenshots/g13-contribution-ledger-foundation.png`
- `docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md`

### Exclusions and future gate

Exclude private Git history, database files and rows, local fixtures, credentials, tokens, DSNs, personal accounts and email, local absolute paths, Legacy source, restricted binary/assets, terminal captures, unapproved third-party content, and all unrelated dirty files. Before export, verify project ownership and provenance, review each candidate diff and binary, rerun secret/privacy/asset scans, and compare the destination. The screenshot summarizes actual G13 checks and says `MANUAL ACCEPTANCE PENDING`, which is its historical capture state; manual acceptance later passed. X publication remains manual.

---

## 中文 — 完整对应版本

状态：**Stage Close Allowlist · 人工验收 PASS · 已授权导出但须重新安全审查**

本清单标识独立 Stage Close 指令授权同步到 Public Mirror 的 G13 项目自有文件。人工验收已 PASS。复制任何文件前，仍须重新审查来源、隐私、受限制素材与秘密。Public Mirror 保持独立 Git History；X 发布和 G14 仍未获授权。

### 候选允许清单

- `apps/game-server/internal/contribution/model.go`
- `apps/game-server/internal/contribution/policy.go`
- `apps/game-server/internal/contribution/policy_test.go`
- `apps/game-server/internal/contribution/service.go`
- `apps/game-server/internal/persistence/postgres/contribution.go`
- `apps/game-server/internal/persistence/postgres/contribution_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go`（仅 G13 关联 Refund/Reversal Guard）
- `apps/game-server/internal/persistence/postgres/store.go`（仅故障注入 Test Hook）
- `apps/game-server/internal/persistence/postgres/migrations/0005_contribution_ledger.sql`
- `docs/adr/0012-g13-contribution-ledger-foundation.md`
- `docs/devlog/G13-contribution-ledger-foundation.md`
- `docs/interactions/G13-contribution-ledger-foundation.md`
- `docs/social/X-DEVELOPMENT-LOG.md`（G13 草稿与截图记录）
- `docs/social/screenshots/g13-contribution-ledger-foundation.png`
- `docs/public-sync/G13-PUBLIC-SYNC-CANDIDATE.md`

### 排除内容与未来 Gate

Private Git History、Database File 和 Row、本地 Fixture、Credential、Token、DSN、个人账号与 Email、本地绝对路径、Legacy Source、受限制 Binary/Asset、Terminal Capture、未批准第三方内容及所有无关 Dirty File 均排除。导出前必须核对项目所有权与来源、逐项审核 Candidate Diff 与 Binary、重跑 Secret/Privacy/Asset Scan，并比较目标仓库。截图汇总真实 G13 检查；其中的 `MANUAL ACCEPTANCE PENDING` 是拍摄时的历史状态，人工验收此后已经 PASS。X 发布仍由用户手动执行。
