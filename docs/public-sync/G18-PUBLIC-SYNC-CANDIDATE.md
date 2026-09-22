# G18 Public Mirror Sync Allowlist

Status: **G18 accepted and merged privately; sanitized public export reviewed**

## English — Primary

The public repository retains its independent history. The G18 export copies only the following project-owned contents from the accepted canonical merge, with curated bilingual documents. It contains no private Git history, private issue links, local paths, database contents, credentials, legacy seller source, or restricted third-party assets.

### Canonical code allowlist

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/miningblock/model.go`
- `apps/game-server/internal/persistence/postgres/emission.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0010_mining_block_reservation.sql`
- `apps/game-server/internal/persistence/postgres/mining_block.go`
- `apps/game-server/internal/persistence/postgres/mining_block_crash_test.go`
- `apps/game-server/internal/persistence/postgres/mining_block_read.go`
- `apps/game-server/internal/persistence/postgres/mining_block_test.go`
- `apps/game-server/internal/persistence/postgres/recycle_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`

The curated documentation allowlist is `docs/adr/0017-g18-mining-block-reward-reservation.md`, `docs/devlog/G18-mining-block-reward-reservation.md`, and `docs/interactions/G18-mining-block-reward-reservation.md`. This manifest, `README.md`, `PUBLIC-CODE-PROVENANCE.md`, and selected bilingual `docs/public/` pages describe the accepted scope. No product code beyond the thirteen listed paths is exported.

The boundary is **G17 global pool → G18 atomic block reservation → immutable history and receipt → finalization retaining the reservation or cancellation releasing it**. Upstream refunds may create Recovery Debt; future emission and cancellations pay that debt first. `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`, with `Distributed = 0` in G18. Production block reward and duration are not finalized; no miners, mining power, tool, map, reward allocation, player ore, or Bun migration exists.

## 中文 — 完整审核版

公开仓库保持独立历史。G18 只从已验收的 canonical 合并中复制上面列明的项目自有内容，并整理双语文档。导出不含私有 Git 历史、私有 Issue 链接、本地路径、数据库内容、凭据、旧卖家源码或受限第三方资产。

### Canonical 代码允许清单

- `apps/game-server/internal/emission/model.go`
- `apps/game-server/internal/miningblock/model.go`
- `apps/game-server/internal/persistence/postgres/emission.go`
- `apps/game-server/internal/persistence/postgres/emission_read.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery.go`
- `apps/game-server/internal/persistence/postgres/emission_recovery_test.go`
- `apps/game-server/internal/persistence/postgres/migrations/0010_mining_block_reservation.sql`
- `apps/game-server/internal/persistence/postgres/mining_block.go`
- `apps/game-server/internal/persistence/postgres/mining_block_crash_test.go`
- `apps/game-server/internal/persistence/postgres/mining_block_read.go`
- `apps/game-server/internal/persistence/postgres/mining_block_test.go`
- `apps/game-server/internal/persistence/postgres/recycle_test.go`
- `apps/game-server/internal/persistence/postgres/store.go`

整理后的文档允许清单为 `docs/adr/0017-g18-mining-block-reward-reservation.md`、`docs/devlog/G18-mining-block-reward-reservation.md` 与 `docs/interactions/G18-mining-block-reward-reservation.md`。本清单、`README.md`、`PUBLIC-CODE-PROVENANCE.md` 和选定的双语 `docs/public/` 页面说明已验收范围。不会导出以上十三条路径之外的产品代码。

边界为 **G17 全服池 → G18 原子 Block 预留 → 不可变历史和回执 → 保留预留的完成或释放预留的取消**。上游退款可形成恢复债务，未来发行和取消优先偿债。守恒式为 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`，G18 的已分发始终为零。正式区块奖励和时长尚未确定；不存在矿工、挖矿算力、工具、地图、玩家奖励分配、玩家矿石或馒头迁移。
