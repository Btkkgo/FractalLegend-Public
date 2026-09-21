# G11 Public Sync Allowlist Record

Status: **SANITIZED ALLOWLIST EXPORT · G11 CLOSED / PASS**

## English — Primary

This record lists the project-owned G11 files selected for the sanitized Public Mirror export after manual acceptance and a fresh safety/provenance audit.

### Exported files

- `apps/game-server/internal/ledger/`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/ledger_schema_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0003_fb_ledger.sql`
- G11 PostgreSQL Store and migration-test changes
- `docs/adr/0010-g11-fb-ledger-foundation.md`
- `docs/devlog/G11-fb-ledger-foundation.md`
- `docs/interactions/G11-fb-ledger-foundation.md`
- `docs/social/X-DEVELOPMENT-LOG.md`
- `docs/social/screenshots/g11-fb-ledger-foundation.png`
- G11 public-safe sections in `README.md` and `docs/public/ROADMAP.md`
- G11 provenance entries in `PUBLIC-CODE-PROVENANCE.md`

### Excluded

Private Git history, Legacy seller source, restricted third-party assets, local evidence, database files, credentials, tokens, DSNs, private email, personal paths, and unapproved third-party code/resources are excluded.

### Gate

The Private Canonical worktree remains the only development location. G11 manual acceptance and explicit stage-close export authorization were confirmed before this file-level copy. The Public Mirror preserves its independent history; no private `.git` data or commit object was copied.

The private `foundation/` integration test and Game Server command/runtime wiring were not exported because the Public Mirror intentionally excludes the complete private runtime assembly. Their accepted results remain documented evidence, while the published Ledger and PostgreSQL packages remain independently buildable and testable.

### Final export verification

The final Public Mirror worktree passed applicable Go tests **98/98** with 0 failures and 0 skips, full Go Race, and Vet. Final scans report: Secret **PASS**, Personal Information **PASS**, Private Email **PASS**, Local Path **PASS**, Legacy Source **PASS**, and Restricted Asset **PASS**. The only new PNG is the project-owned sanitized G11 verification card.

---

## 中文 — 完整对应版本

本记录列出 G11 人工验收与新的 Safety/Provenance Audit 完成后，通过脱敏 Allowlist Export 选择进入 Public Mirror 的项目自有文件。

### 已导出文件

- `apps/game-server/internal/ledger/`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/ledger_schema_test.go`
- `apps/game-server/internal/persistence/postgres/ledger_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0003_fb_ledger.sql`
- G11 PostgreSQL Store 与 Migration Test 变更
- `docs/adr/0010-g11-fb-ledger-foundation.md`
- `docs/devlog/G11-fb-ledger-foundation.md`
- `docs/interactions/G11-fb-ledger-foundation.md`
- `docs/social/X-DEVELOPMENT-LOG.md`
- `docs/social/screenshots/g11-fb-ledger-foundation.png`
- `README.md` 与 `docs/public/ROADMAP.md` 中可公开的 G11 章节
- `PUBLIC-CODE-PROVENANCE.md` 中的 G11 Provenance 条目

### 排除内容

Private Git History、Legacy Seller Source、受限制第三方 Asset、本地 Evidence、数据库文件、Credential、Token、DSN、私人 Email、个人路径，以及未确认授权的第三方代码/资源均不在候选范围。

### Gate

Private Canonical Worktree 继续是唯一开发位置。本次文件级 Copy 开始前，G11 人工验收与 Stage Close 公开导出授权已经确认。Public Mirror 保持独立历史；没有复制 Private `.git` Data 或 Commit Object。

Private `foundation/` Integration Test 与 Game Server Command/Runtime Wiring 没有导出，因为 Public Mirror 明确排除完整 Private Runtime Assembly。其已验收结果继续作为文档证据保留；公开的 Ledger 与 PostgreSQL Package 仍可独立 Build 和 Test。

### 最终导出验证

最终 Public Mirror Worktree 的适用 Go Test **98/98** 通过、0 Fail、0 Skip，完整 Go Race 与 Vet 通过。最终扫描结果为：Secret **PASS**、Personal Information **PASS**、Private Email **PASS**、Local Path **PASS**、Legacy Source **PASS**、Restricted Asset **PASS**。唯一新增 PNG 是项目自有的脱敏 G11 Verification Card。
