# G21 — Public export allowlist / 公开导出清单

## English — Primary

**G21 STAGE CLOSED — PASS.** The owner accepted the completed settlement foundation. Accepted settlement merge: `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`; canonical closeout revision: `37452efe5c9d529eec04c7629c483de4156dd6fb`. Public Git history remains independent.

The complete accepted repository passed **121/121 required G21 checks**, plus **9 P0_COVERED** (130 total); normal and race each **1296 PASS**, real PostgreSQL integration PASS, crash recovery **9/9**, CI **18/18**, and native Windows/Linux/macOS deterministic equality. Receipt digest: `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`.

This public subset was independently checked against fresh, isolated PostgreSQL databases: normal **1060/1060 PASS**, race **1060/1060 PASS**, zero failures or test skips; Go vet and builds for macOS arm64 / Linux amd64 / Windows amd64 PASS. Cross-compilation is a build check, not native execution; native results above belong to the accepted complete repository.

Production settlement stays **disabled**. G18 **R=10** authority and economic parameters are unchanged. G22 has not started; no release or X publication.

Public initial Normal: 1059 PASS + 1 FAIL because the intentionally excluded command directory was absent. A bilingual cmd/README.md now documents the exported audit boundary; no Go/test changes were made. The fresh full rerun results above replace that initial failure.

Only the following exact paths are authorized for this sync. All code/tests/migration are project-owned and byte-identical to the accepted canonical merge. Documentation is curated bilingually; no private repository links, local paths, credentials, private operational configuration, workflow, legacy source, third-party assets, database data, raw conversations or private Git history are exported. Existing nine approved PNG assets remain unchanged. Secret/privacy/restricted asset/size/dependency/license/link audit must pass before push.

## 中文 — 完整审核版

**G21 STAGE CLOSED — PASS。** 用户已验收完整结算基础。已验收实现合并版本为 `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`，canonical 收尾版本为 `37452efe5c9d529eec04c7629c483de4156dd6fb`；公开仓库继续保持独立 Git 历史。

完整已验收仓库 **121/121 必需 G21 项通过**，另有 **9 P0_COVERED**（共 130 项）；普通与竞态各 **1296 PASS**，真实 PostgreSQL 集成通过、崩溃恢复 **9/9**、CI **18/18**，Windows／Linux／macOS 真实原生执行的确定性一致性通过。回执摘要为 `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`。

本公开子集使用新建隔离 PostgreSQL 库独立验证：普通 **1060/1060 PASS**、竞态 **1060/1060 PASS**，零失败、零测试跳过；Go vet 及 macOS arm64／Linux amd64／Windows amd64 构建通过。交叉编译仅作为构建检查，上述真实原生结果属于完整已验收仓库。

生产结算保持**禁用**；G18 **R=10** 权威与经济参数不变。G22 未开始，未发布 Release 或 X。

公开首次普通测试为 1059 PASS + 1 FAIL，原因是按规则排除的命令目录不存在。现以双语 cmd/README.md 明确记录导出审计边界；未修改 Go 或测试。上方全新全量重跑结果覆盖首次失败，原始失败日志保留。

仅同步下列精确路径。代码／测试／迁移均为自有且与验收版本逐字节一致；文档双语整理。排除私有仓库链接、本地路径、凭据、私有运行配置、工作流、旧源码、第三方素材、数据库、原始对话及私有 Git 历史。9 张既有批准 PNG 不变；通过秘密、隐私、受限素材、大小、依赖、许可和链接审计后才推送。

## Code / Tests / Migration — 46 paths

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/miningreward/allocation.go`
- `apps/game-server/internal/miningreward/allocation_test.go`
- `apps/game-server/internal/miningreward/canonical_time.go`
- `apps/game-server/internal/miningreward/canonical_time_test.go`
- `apps/game-server/internal/miningreward/closure_test.go`
- `apps/game-server/internal/persistence/postgres/black_iron_migration.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery.go`
- `apps/game-server/internal/persistence/postgres/migrations/0015_test_mining_reward_settlement.sql`
- `apps/game-server/internal/persistence/postgres/mining_block_identity_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power.go`
- `apps/game-server/internal/persistence/postgres/mining_power_fixture_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_isolation_test.go`
- `apps/game-server/internal/persistence/postgres/mining_power_seal.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_activity_limit_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_canonical.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_canonical_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_allocation_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_authority_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_bounds_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_constraints_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_final_concurrency_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_gate_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_inventory_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_migration_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_payload_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_recovery_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_replay_artifacts_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_reviewfix_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_snapshot_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_closure_timeout_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_crash_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_decode.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_decode_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_issuance.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_native_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_preview.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_preview_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_schema_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_settlement.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_settlement_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_t044_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_t117_test.go`
- `apps/game-server/internal/persistence/postgres/mining_reward_timeout_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`

## Curated documentation / 整理文档

- `apps/game-server/cmd/README.md`
- `docs/devlog/G21-mining-reward-full-settlement.md`
- `docs/devlog/G21-mining-reward-full-settlement.zh-CN.md`
- `docs/interactions/G21-mining-reward-full-settlement.md`
- `docs/interactions/G21-mining-reward-full-settlement.zh-CN.md`
- `docs/reports/G21-FULL-SETTLEMENT-ACCEPTANCE-EVIDENCE.md`
- `docs/reports/G21-FULL-SETTLEMENT-TEST-REPORT.md`
- `docs/reports/G21-NUMBERED-TEST-MATRIX.md`
- `README.md`
- `docs/public/CURRENT-STATUS.md`
- `docs/public/ROADMAP.md`
- `docs/public/DEVELOPMENT-HISTORY.md`
- `docs/public/ARCHITECTURE.md`
- `docs/public/ECONOMY.md`
- `docs/public/PROJECT-OVERVIEW.md`
- `PUBLIC-CODE-PROVENANCE.md`
- `docs/public-sync/G21-PUBLIC-SYNC-CANDIDATE.md`

## Final public gates / 最终公开门槛

PASS: all 284 tracked/nonignored files scanned with zero findings; Secret, Privacy, private-link, restricted-source/asset, active/external SVG, binary/archive/oversize and relative-link audits. All 46 Go/SQL paths are byte-identical; 9 approved PNG hashes unchanged. No dependency, manifest or license changes. Public normal/Race: 1060 PASS each, zero failed/skipped tests; 9/9 real crash barriers each. Vet and three-platform builds PASS. Public commit uses GitHub noreply identity; remote/anonymous verification follows push.

PASS：全部 284 个受追踪／非忽略文件零扫描发现，秘密、隐私、私有链接、受限源码／素材、活动／外部 SVG、二进制／归档／超大文件及相对链接审计通过。46 个 Go／SQL 路径逐字节一致，9 张批准 PNG 哈希不变。无依赖、依赖清单或许可变更。公开普通／竞态各 1060 PASS、零失败／跳过，两套真实崩溃窗口均 9/9；Vet 与三平台构建通过。公开提交采用 GitHub noreply 身份，推送后验证远端与匿名访问。
