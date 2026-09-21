# G12 Public Sync Candidate Manifest

Status: **STAGE CLOSE ALLOWLIST · EXPORT AUTHORIZED SUBJECT TO SAFETY REVIEW**

## English — Primary

This manifest identifies project-owned G12 files for the separately authorized Stage Close Public Mirror sync. Manual technical acceptance is PASS. Each selected file still requires a fresh safety/provenance review before copying; the final sync outcome is recorded in Issue #8.

### Candidate files

- `apps/game-server/internal/trade/model.go`
- `apps/game-server/internal/trade/repository.go`
- `apps/game-server/internal/trade/service.go`
- `apps/game-server/internal/trade/service_test.go`
- `apps/game-server/internal/trade/g12_settlement_test.go`
- `apps/game-server/internal/persistence/postgres/trade.go`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go`
- `apps/game-server/internal/persistence/postgres/g12_trade_settlement_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0004_fb_trade_settlement.sql`
- `docs/adr/0011-g12-atomic-item-fb-trade-settlement.md`
- `docs/devlog/G12-fb-trade-settlement.md`
- `docs/interactions/G12-fb-trade-settlement.md`
- `docs/social/X-DEVELOPMENT-LOG.md`
- `docs/social/screenshots/g12-fb-trade-settlement.png`
- G12 status/documentation links in `README.md`

### Excluded

Private Git history, database files, local test databases, environment variables, credentials, tokens, DSNs, personal accounts, private email, local absolute paths, Legacy source, restricted binaries/assets, terminal captures, and unapproved third-party content are excluded.

### Candidate scan and publication gate

The allowlist must pass a final added-line and binary review for Secret, Personal Information, Private Email, Local Path, Legacy Source, and Restricted Asset exposure before export. The G12 screenshot is a project-owned verification card derived from actual test results. Stage Close authorization permits a safety-gated Public Mirror Commit and Push; it does not authorize X publication or G13.
The screenshot captures the historical pre-acceptance review state; its `MANUAL ACCEPTANCE PENDING` line is not the current technical acceptance status. Any later use must explain that G12 technical acceptance is PASS.

---

## 中文 — 完整对应版本

状态：**Stage Close Allowlist · 授权导出但必须先通过安全审核**

本清单列出独立 Stage Close 指令授权同步到 Public Mirror 的项目自有 G12 文件。人工技术验收已确认为 PASS。复制任何选定文件之前，仍必须重新完成 Safety/Provenance Review；最终同步结果记录在 Issue #8。

### 候选文件

- `apps/game-server/internal/trade/model.go`
- `apps/game-server/internal/trade/repository.go`
- `apps/game-server/internal/trade/service.go`
- `apps/game-server/internal/trade/service_test.go`
- `apps/game-server/internal/trade/g12_settlement_test.go`
- `apps/game-server/internal/persistence/postgres/trade.go`
- `apps/game-server/internal/persistence/postgres/ledger.go`
- `apps/game-server/internal/persistence/postgres/ledger_transaction.go`
- `apps/game-server/internal/persistence/postgres/g12_trade_settlement_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0004_fb_trade_settlement.sql`
- `docs/adr/0011-g12-atomic-item-fb-trade-settlement.md`
- `docs/devlog/G12-fb-trade-settlement.md`
- `docs/interactions/G12-fb-trade-settlement.md`
- `docs/social/X-DEVELOPMENT-LOG.md`
- `docs/social/screenshots/g12-fb-trade-settlement.png`
- `README.md` 中的 G12 状态与 Documentation Link

### 排除内容

Private Git History、Database File、本地 Test Database、Environment Variable、Credential、Token、DSN、个人账号、Private Email、本地绝对路径、Legacy Source、受限制 Binary/Asset、Terminal Capture 和未获批准的第三方内容均被排除。

### 候选扫描与发布 Gate

导出前，Allowlist 内容必须重新通过 Added-line 与 Binary Review，检查 Secret、Personal Information、Private Email、Local Path、Legacy Source 和 Restricted Asset。G12 Screenshot 是根据实际测试结果制作的项目自有 Verification Card。Stage Close 授权允许经过安全 Gate 的 Public Mirror Commit 与 Push；该授权不包括 X Publication 或 G13。
该截图记录的是人工验收前的历史 Review 状态，其中的 `MANUAL ACCEPTANCE PENDING` 并非当前技术验收状态。以后使用时必须说明 G12 技术验收已 PASS。
