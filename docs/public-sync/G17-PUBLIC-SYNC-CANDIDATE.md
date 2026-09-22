# G17 Public Mirror Sync Allowlist

Status: **G17 accepted and merged privately; sanitized public export reviewed**

## English — Primary

The Public Mirror retains independent Git history. This export copies only the following project-owned file contents from the accepted G17 canonical merge, then sanitizes the bilingual records. It does not copy private Git objects, private issue or PR links, configuration, databases, logs, credentials, local paths, Legacy seller source, or restricted third-party assets.

### Canonical content allowlist

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/emission/rule.go`
- `apps/game-server/internal/emission/rule_test.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/emission.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0009_black_iron_emission_pool.sql`
- `apps/game-server/internal/persistence/postgres/recycle_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`
- `docs/adr/0016-g17-black-iron-emission-pool.md`
- `docs/devlog/G17-black-iron-emission-pool.md`
- `docs/interactions/G17-black-iron-emission-pool.md`

The public repository also updates `README.md`, `PUBLIC-CODE-PROVENANCE.md`, this allowlist, and selected bilingual `docs/public/` overview, economy, architecture, status, history, and roadmap pages to describe the accepted scope. Those public-facing edits do not widen the code export.

The G17 boundary is **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**. Emission capacity is not player ore. `DEV_G17_1_TO_1` is for tests and development only; the production ratio is not finalized. No mining block, power, tool, reward distribution, player ore inventory, or Bun migration is exported.

## 中文 — 完整审核版

Public Mirror 保持独立 Git History。本次仅从已验收的 G17 canonical 合并中复制下列项目自有文件内容，然后脱敏双语记录。不复制私有 Git Object、私有 Issue 或 PR Link、配置、数据库、日志、凭据、本地路径、旧卖家源码或受限第三方资产。

### Canonical 内容允许清单

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/emission/rule.go`
- `apps/game-server/internal/emission/rule_test.go`
- `apps/game-server/internal/persistence/postgres/contribution_refund.go`
- `apps/game-server/internal/persistence/postgres/emission.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0009_black_iron_emission_pool.sql`
- `apps/game-server/internal/persistence/postgres/recycle_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`
- `docs/adr/0016-g17-black-iron-emission-pool.md`
- `docs/devlog/G17-black-iron-emission-pool.md`
- `docs/interactions/G17-black-iron-emission-pool.md`

公开仓库另更新 `README.md`、`PUBLIC-CODE-PROVENANCE.md`、本清单，以及 `docs/public/` 中的双语概览、经济、架构、状态、历史和路线图页面，以准确描述已验收范围。这些公开说明不会扩大代码导出范围。

G17 边界为 **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**。发行额度不等于玩家矿石。`DEV_G17_1_TO_1` 只用于测试／开发；正式比例尚未确定。本次不导出 Mining Block、Power、Tool、奖励分配、玩家矿石库存或馒头迁移。
