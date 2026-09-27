# G20 — Public Export Allowlist

Independent public history; accepted private canonical source 5c1026b7c196412d8e17caeac5e5e443cdfb6aae. No private repository links, Git objects, raw conversations, local paths/caches, production data or credentials. G18.1 is an explicit owned dependency.

独立公开历史；源为已验收私有 canonical 5c1026b7c196412d8e17caeac5e5e443cdfb6aae。不含私有链接／Git 对象／原始对话／本地路径或缓存／生产数据／凭据；G18.1 为明确的自有前置依赖。

## Byte-identical code allowlist / 逐字节相同代码清单

- `apps/game-server/internal/miningblock/model.go`
- `apps/game-server/internal/miningpower/calculator.go`
- `apps/game-server/internal/miningpower/calculator_test.go`
- `apps/game-server/internal/miningpower/canonical_time.go`
- `apps/game-server/internal/miningpower/canonical_time_test.go`
- `apps/game-server/internal/miningpower/identity_test.go`
- `apps/game-server/internal/miningpower/intent.go`
- `apps/game-server/internal/miningpower/intent_test.go`
- `apps/game-server/internal/miningpower/model.go`
- `apps/game-server/internal/miningpower/reconcile.go`
- `apps/game-server/internal/miningpower/reconcile_test.go`
- `apps/game-server/internal/miningpower/report_test.go`
- `apps/game-server/internal/miningpower/review_test.go`
- `apps/game-server/internal/miningpower/service.go`
- `apps/game-server/internal/miningpower/service_test.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0012_mining_block_instance_identity.sql`
- `apps/game-server/internal/persistence/postgres/migrations/0013_mining_power_foundation.sql`
- `apps/game-server/internal/persistence/postgres/mining_block.go`
- `apps/game-server/internal/persistence/postgres/mining_block_identity_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power.go`
- `apps/game-server/internal/persistence/postgres/mining_power_behavior_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_block.go`
- `apps/game-server/internal/persistence/postgres/mining_power_canonical_time_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_crash_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_fixture_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_isolation_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_order_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_parallel_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_profiles.go`
- `apps/game-server/internal/persistence/postgres/mining_power_read.go`
- `apps/game-server/internal/persistence/postgres/mining_power_review_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`

## Curated documents / 整理文档

- `docs/devlog/G20-mining-power-foundation.md`
- `docs/interactions/G20-mining-power-foundation.md`
- `docs/reports/G20-MINING-POWER-TEST-REPORT.md`
- `docs/reports/G20-T50-NATIVE-CI-EVIDENCE.md`
- `docs/adr/0019-g20-mining-power-foundation.md`
- `docs/architecture/G20-MINING-POWER-FOUNDATION-SPEC.md`
- `docs/reports/G20-scenario-evidence.csv`
- `docs/social/G20-x-draft.md`
- `docs/social/G20-status-card-brief.md`
- `docs/social/G20-status-card.svg`
- `docs/social/G20-status-card-data.csv`
- `README.md`
- `PUBLIC-CODE-PROVENANCE.md`
- `docs/public/CURRENT-STATUS.md`
- `docs/public/ARCHITECTURE.md`
- `docs/public/ROADMAP.md`
- `docs/public/DEVELOPMENT-HISTORY.md`
- `docs/public/ECONOMY.md`
- `docs/public/PROJECT-OVERVIEW.md`

Existing approved PNG hashes and dependency manifests are preserved. Public full Normal/Race, Vet, platform builds and every scan must pass before push. Verification receipts are appended after execution.

既有已批准 PNG 哈希和依赖清单保留；公开完整 Normal／Race、Vet、三平台构建与全部扫描必须通过后才能推送。实际验证回执在运行后补充。

## Independent public verification / 公开独立验证

Fresh isolated PostgreSQL databases were used sequentially: public full Normal **646/646 PASS**, full Race **646/646 PASS**, **0 FAIL / 0 SKIP / 0 data race**. All 90 G20 scenario assertions mapped to passing events in both runs; the three-native T50 receipts remain exact-source-equivalent to the exported calculator/tests. Go Vet and Linux/Windows/macOS all-package builds PASS. These public checks were run locally under the existing mirror workflow; private PR/canonical CI results are identified separately.

All **215** tracked/nonignored public files passed Secret, Privacy/Personal Path, Legacy Source, Restricted Asset, active/external SVG and relative-link audits with **0 findings**. The **34** code paths remain byte-identical to accepted canonical. All **9** approved baseline PNG hashes are unchanged. No new binary/archive/oversized file, dependency or licensing change, production data, credential, private Git object or raw conversation is exported. Public commits use the existing GitHub noreply identity; Git history remains independent.

顺序使用各自全新隔离 PostgreSQL 测试库：公开完整 Normal **646/646**、Race **646/646**，失败／跳过／数据竞争均为 **0**。两个运行均将 90 个 G20 场景断言对应到通过事件；三平台原生 T50 回执与导出的计算器／测试源完全相同。Go Vet 和 Linux／Windows／macOS 全包构建通过。这些公开检查按既有镜像流程本地运行，私有 PR／canonical CI 结果单独标明。

全部 **215** 个受追踪／非忽略公开文件通过秘密、隐私／个人路径、旧源码、受限资产、活动／外部 SVG 与相对链接检查，**零发现**。**34** 个代码路径保持与已验收 canonical 逐字节相同；**9** 个既有批准 PNG 哈希不变。没有新二进制／归档／超大文件、依赖／许可变更、生产数据、凭据、私有 Git 对象或原始对话。公开提交使用既有 GitHub noreply 身份，保持独立历史。
