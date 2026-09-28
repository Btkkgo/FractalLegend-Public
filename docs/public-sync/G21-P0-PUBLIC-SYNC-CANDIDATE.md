# G21-P0 — Public Mirror Export Allowlist / 公开镜像导出清单

## English — Primary

The source is the accepted private canonical G21-P0 merge. This independent public history receives only the 27 project-owned Go/SQL paths below, seven curated bilingual records, and nine public repository records. Go/SQL contents are byte-identical to accepted canonical. The private CI workflow, evidence scripts, Git objects, issue/PR links, local caches, database data, credentials, private runtime assembly, Legacy seller source, restricted third-party assets and raw conversations are excluded.

### Byte-identical code, tests and migration

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/miningpower/canonical_evidence.go`
- `apps/game-server/internal/miningpower/canonical_evidence_test.go`
- `apps/game-server/internal/miningpower/seal.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration_read.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery.go`
- `apps/game-server/internal/persistence/postgres/migrations/0014_settlement_prerequisite_binding.sql`
- `apps/game-server/internal/persistence/postgres/mining_block.go`
- `apps/game-server/internal/persistence/postgres/mining_power.go`
- `apps/game-server/internal/persistence/postgres/mining_power_beneficiary.go`
- `apps/game-server/internal/persistence/postgres/mining_power_fixture_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_isolation_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_seal.go`
- `apps/game-server/internal/persistence/postgres/mining_power_seal_boundary_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_seal_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_test.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_beneficiary_test.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_binding_test.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_crash_test.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_distribution.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_distribution_test.go`
- `apps/game-server/internal/persistence/postgres/mining_prerequisite_security_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reservation_binding.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_issuance.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_issuance_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`

### Curated bilingual records

- `docs/architecture/G21-DESIGN-GATE-STATUS.md`
- `docs/architecture/G21-P0-IMPLEMENTATION-PLAN.md`
- `docs/architecture/G21-P0-SETTLEMENT-PREREQUISITES-SPEC.md`
- `docs/decisions/ADR-G21P0-001-settlement-prerequisite-authority.md`
- `docs/devlog/G21-P0-settlement-prerequisite-foundation.md`
- `docs/reports/G21-P0-SETTLEMENT-PREREQUISITE-TEST-MATRIX.md`
- `docs/reports/G21-P0-SETTLEMENT-PREREQUISITE-TEST-REPORT.md`

### Public repository records

- `README.md`
- `PUBLIC-CODE-PROVENANCE.md`
- `docs/public/CURRENT-STATUS.md`
- `docs/public/ARCHITECTURE.md`
- `docs/public/ROADMAP.md`
- `docs/public/DEVELOPMENT-HISTORY.md`
- `docs/public/ECONOMY.md`
- `docs/public/PROJECT-OVERVIEW.md`
- `docs/public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md`

### Safety and verification

The public source is independently tested. No production route, full G21 allocation or settlement, real player ore, claim or G22 is exported. Final receipts are recorded below after the gates pass.

## 中文 — 完整审核版

来源为已验收的私有 canonical G21-P0 合并版本。此独立公开历史仅接收上方逐项列出的 27 个项目自有 Go／SQL 路径、7 份整理后的双语记录和 9 份公开仓库记录。Go／SQL 内容与已验收 canonical 逐字节一致。排除私有 CI 工作流、证据脚本、Git Object、Issue／PR 链接、本地缓存、数据库数据、凭据、私有完整 Runtime、旧卖家源码、受限第三方素材及原始对话。

公开源码须独立测试。未导出生产入口、完整 G21 奖励分配或结算、真实玩家矿石、领取或 G22。全部门槛通过后在下方记录最终回执。

## Final verification / 最终验证

Fresh isolated PostgreSQL databases were used sequentially. Public full Normal **692/692 PASS** and full Race **692/692 PASS**, with **0 FAIL / 0 SKIP / 0 data race**. Go Vet and all-package macOS arm64, Linux amd64 and Windows amd64 builds PASS. The 27 Go/SQL paths are byte-identical to accepted canonical; 9 previously approved PNG hashes are unchanged. All 240 tracked/nonignored public files passed Secret, Privacy/Personal Path, Private Link, Legacy Source, Restricted Asset, active/external SVG, binary/archive/oversize and relative-link audits with **0 findings**. No dependency or license change, production data, credential or private Git object is exported. This independent public verification does not claim native Windows/Linux runtime tests; the accepted private PR recorded those separately.

依次使用两座全新隔离 PostgreSQL 测试库：公开完整普通测试 **692/692 PASS**、完整竞态测试 **692/692 PASS**，**0 FAIL／0 SKIP／0 数据竞争**。Go Vet 与 macOS arm64、Linux amd64、Windows amd64 全包构建通过。27 个 Go／SQL 路径与已验收 canonical 逐字节相同；9 张既有批准 PNG 的哈希未变。全部 240 个受追踪／非忽略公开文件通过 Secret、Privacy／个人路径、私有链接、旧源码、受限素材、活动／外部 SVG、二进制／归档／超大文件及相对链接审计，**零发现**。未导出依赖或许可变更、生产数据、凭据或私有 Git Object。此公开仓库的独立验证不声称已在其自身运行 Windows／Linux 原生测试；已验收私有 PR 的原生结果另行记录。
