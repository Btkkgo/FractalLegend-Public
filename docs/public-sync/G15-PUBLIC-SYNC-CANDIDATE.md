# G15 Public Mirror Sync Candidate

Status: **Manual acceptance PASS · Stage Close sanitized export: 16 files**

## English — Primary

This public variant records the 16 project-owned G15 code and bilingual evidence files selected for sanitized Public Mirror export after private canonical integration and fresh safety review. The private candidate also listed the G9 restart integration test; this public snapshot deliberately omits it because it excludes the Game Server assembly that test requires. The Public Mirror retains independent Git history. The G14 social-card stash, private Issue, private repository metadata, and local test databases are outside this export.

### Candidate allowlist

- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/migrations/0007_system_spend.sql`
- `apps/game-server/internal/persistence/postgres/store.go`
- `apps/game-server/internal/persistence/postgres/system_spend.go`
- `apps/game-server/internal/persistence/postgres/system_spend_reconcile.go`
- `apps/game-server/internal/persistence/postgres/system_spend_test.go`
- `apps/game-server/internal/systemspend/model.go`
- `apps/game-server/internal/systemspend/service.go`
- `apps/game-server/internal/systemspend/service_test.go`
- `docs/adr/0014-g15-eligible-system-spend.md`
- `docs/devlog/G15-eligible-system-spend.md`
- `docs/interactions/G15-eligible-system-spend.md`
- `docs/reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md`
- `docs/social/G15-MILESTONE-NOTES.md`
- `docs/social/screenshots/g15-eligible-system-spend.png`
- `docs/public-sync/G15-PUBLIC-SYNC-CANDIDATE.md`

### Exclusions and next gate

Do not export secrets, passwords, tokens, personal names or email, local absolute paths, private GitHub links, private Git history, raw logs, database URLs or files, legacy seller source, restricted binaries/assets, or unrelated files. The G9 restart integration test remains in private canonical; no missing Game Server assembly was copied into this mirror to make it compile. Before commit and push, recheck exact file provenance, content scans, destination diff, and mirror policy. No public mirror file, branch, commit, or remote was changed in G15 technical acceptance. No X post was published.

---

## 中文 — 完整对应版本

状态：**人工验收 PASS · Stage Close 脱敏导出 16 个文件**

本公开版本记录 Private Canonical 集成和新一轮安全审查后，为脱敏 Public Mirror 导出选定的 16 个项目自有 G15 代码与双语证据文件。私有候选清单还列出 G9 重启集成测试；公开 Snapshot 刻意不包含该测试，因为它依赖这里未公开的 Game Server Assembly。Public Mirror 保持独立 Git History。G14 社交素材 Stash、私有 Issue、私有仓库 Metadata 以及本地测试数据库均不在本次导出范围内。

### 候选允许清单

- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/migrations/0007_system_spend.sql`
- `apps/game-server/internal/persistence/postgres/store.go`
- `apps/game-server/internal/persistence/postgres/system_spend.go`
- `apps/game-server/internal/persistence/postgres/system_spend_reconcile.go`
- `apps/game-server/internal/persistence/postgres/system_spend_test.go`
- `apps/game-server/internal/systemspend/model.go`
- `apps/game-server/internal/systemspend/service.go`
- `apps/game-server/internal/systemspend/service_test.go`
- `docs/adr/0014-g15-eligible-system-spend.md`
- `docs/devlog/G15-eligible-system-spend.md`
- `docs/interactions/G15-eligible-system-spend.md`
- `docs/reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md`
- `docs/social/G15-MILESTONE-NOTES.md`
- `docs/social/screenshots/g15-eligible-system-spend.png`
- `docs/public-sync/G15-PUBLIC-SYNC-CANDIDATE.md`

### 排除内容与下一道 Gate

不得导出 Secret、Password、Token、个人姓名或 Email、本地绝对路径、私有 GitHub Link、私有 Git History、原始 Log、数据库 URL 或文件、Legacy Seller Source、受限 Binary/Asset 或无关文件。G9 重启集成测试保留在 Private Canonical；没有为了让该测试编译而复制缺失的 Game Server Assembly。Commit 和 Push 前必须重新核验准确文件来源、内容扫描、目标 Diff 和 Mirror Policy。G15 技术验收阶段没有修改 Public Mirror 的任何文件、分支、Commit 或 Remote，也没有发布 X 推文。
