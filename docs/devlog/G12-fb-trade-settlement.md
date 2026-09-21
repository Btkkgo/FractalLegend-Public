# G12 — FB-backed Player Trade Settlement

Status: **Technical acceptance PASS · Stage Close authorized**

Tracking: private canonical Issue #8 (archive reference only) / 私有 Canonical Issue #8（仅档案引用）

Base commit: `0ebaa5d49b32d115ec9dd77998aafa4711f2abb6`

## English — Primary

### Goal and delivered behavior

G12 connects the accepted G10 item Trade state machine to the accepted G11 FB Ledger through one atomic settlement path. A Trade may now contain item offers, FB-only offers, or both sides' item and FB offers. FB amounts use `int64`; negative values are rejected, receiver overflow fails closed, and the player-to-player settlement fee is exactly 0%.

Changing an FB offer advances the Trade revision and invalidates both confirmations, just like an item-offer change. Confirmation validates current ownership, persistent item locks, and current FB balance. Finalize repeats every authority check against locked PostgreSQL rows.

### Atomic transaction and accounting

The Trade Service coordinates one serializable transaction across Trade state, both Character Aggregates, persistent item locks, Ledger accounts, Ledger transactions/entries, materialized balances, audit events, and the durable receipt. The Ledger posting implementation is shared with G11 and can run inside the Trade transaction; G12 did not create a second Ledger SQL implementation in the Trade Repository.

G12 selected no FB Hold for direct Trade. Finalize locks balances and revalidates them. It selected Gross Settlement for audit clarity: `A_TO_B` and `B_TO_A` use separate stable `PLAYER_TRADE` references. Repeated Finalize calls return the stored settlement and cannot repeat item or FB movement.

### Known risks

- **Option A:** Confirmation does not reserve FB. Other legitimate operations may reduce the balance before Finalize. Finalize locks account rows, revalidates the authoritative balance, and rolls back the entire transaction if funds are insufficient; no partial settlement is permitted. This semantic was explicitly accepted.
- **Finite retry exhaustion:** PostgreSQL serialization/deadlock retries are limited to three attempts. Under extreme contention, that budget may be exhausted, causing Finalize to fail without partial settlement. This remains an accepted, documented risk.

### Persistence and failure recovery

Migration `0004_fb_trade_settlement.sql` adds both FB offers and linked Ledger transaction IDs to the durable Trade schema. Real PostgreSQL tests prove in-progress and completed state reload, full restart replay, stable references, concurrent double-spend protection, deterministic opposite-direction settlement, and complete rollback.

Failure injection covers:

- after item ownership update;
- during the first Ledger balance/entry application;
- after Ledger write;
- before the completed-state transition;
- after Trade persistence and immediately before database commit.

No injected failure produced a partial item transfer, FB movement, posted settlement, receipt, or completed Trade.

### Fresh verification

- G12 targeted checks: **45/45 PASS**, 0 skip.
- Real PostgreSQL G12 checks inside that target set: **12/12 PASS**.
- Randomized property sequence: **200/200 iterations PASS**.
- High concurrency: **100 simultaneous settlements PASS** with FB conservation.
- Full Go regression with real PostgreSQL and complete G9/G11 restart fixtures: **379/379 PASS**, 0 failures, 0 skips.
- Final `go test -race ./... -count=1`: **PASS** across all 14 Go packages after resetting the dedicated repeat-run G9 fixture database.
- The first Race rerun reused a previously mutated G9 database and failed with `ITEM_ALREADY_EQUIPPED`; the isolated schema was reset and the complete final Race run passed. No test was weakened or skipped.
- `go vet ./...`: **PASS**.
- Main Node suite: **24/24 PASS**.
- G1: **9/9 PASS**; G2: **13/14** with the existing restricted Legacy archive `Atlas Determinism Test` environment failure; G3: **21/21 PASS**; G4: **7/7 PASS**; G5: **3/3 PASS**; G6: **6/6 PASS**; G7: **6/6 PASS**; G8: **6/6 PASS**. Combined Node result: **95/96**, with no new failure.
- TypeScript typecheck: **PASS**.
- Web production build: **PASS** with the existing Vite large-chunk warning.
- Windows amd64, Linux amd64, and macOS arm64 Game Server builds: **PASS**.

### Security and limits

The browser/client cannot set FB balances, Ledger entries, Trade completion, fee, receipt, or settlement outcome. Production code contains no Trade funding shortcut. Test balances remain isolated in `_test.go` fixtures with conserving SYSTEM liability entries.

G12 does not implement Marketplace, Auction, Escrow, Trade UI, Blockchain RPC, on-chain settlement, Wallet, Deposit, Withdrawal, Contribution, Mining, automated Trade reversal, player-owned city/continent code, or G13. The Stage Close commit, integration, Issue update, and Public Mirror evidence are tracked in Issue #8. The X draft is not published.

---

## 中文 — 完整对应版本

### 目标与已交付行为

G12 通过一条原子结算路径，把已验收的 G10 Item Trade State Machine 与已验收的 G11 FB Ledger 连接起来。Trade 现在可以包含 Item Offer、FB-only Offer，或双方同时提供 Item 与 FB。FB Amount 使用 `int64`；负数会被拒绝，接收方 Overflow 会 Fail Closed，玩家自由交易 Settlement Fee 严格为 0%。

修改 FB Offer 会增加 Trade Revision，并像修改 Item Offer 一样使双方 Confirmation 失效。Confirmation 会验证当前 Ownership、持久 Item Lock 和当前 FB Balance；Finalize 会在锁定的 PostgreSQL Row 上重新执行全部 Authority Check。

### 原子事务与记账

Trade Service 通过一个 Serializable Transaction 协调 Trade State、双方 Character Aggregate、持久 Item Lock、Ledger Account、Ledger Transaction/Entry、Materialized Balance、Audit Event 和持久 Receipt。Ledger Posting Implementation 与 G11 共用，并可在 Trade Transaction 内执行；G12 没有在 Trade Repository 中复制第二套 Ledger SQL。

G12 为 Direct Trade 选择不建立 FB Hold，Finalize 通过锁定并重新验证 Balance 保证安全。G12 选择 Gross Settlement 以提高审计清晰度：`A_TO_B` 和 `B_TO_A` 使用独立、稳定的 `PLAYER_TRADE` Reference。重复 Finalize 会返回已保存的 Settlement，不会重复移动 Item 或 FB。

### 已知风险

- **Option A：**Confirmation 不预留 FB。其他合法操作可能在 Finalize 前减少余额。Finalize 会锁定 Account Row、重新验证权威余额；余额不足时整个 Transaction 回滚，不允许部分结算。这一语义已被明确接受。
- **有限 Retry 耗尽：**PostgreSQL Serialization/Deadlock 最多重试三次。在极端竞争下，次数可能耗尽，使 Finalize 失败，但不会产生部分结算。这是已接受并持续记录的风险。

### 持久化与故障恢复

Migration `0004_fb_trade_settlement.sql` 为持久 Trade Schema 增加双方 FB Offer 和关联 Ledger Transaction ID。真实 PostgreSQL Test 已验证进行中与已完成状态 Reload、完整 Restart Replay、稳定 Reference、并发 Double-spend Protection、确定性相反方向 Settlement 和完整 Rollback。

Failure Injection 覆盖：

- Item Ownership Update 之后；
- 第一条 Ledger Balance/Entry 应用过程中；
- Ledger Write 之后；
- Completed-state Transition 之前；
- Trade 持久化完成且 Database Commit 前一刻。

所有注入失败均未产生部分 Item Transfer、FB Movement、已入账 Settlement、Receipt 或 Completed Trade。

### 本轮最新验证

- G12 专项检查：**45/45 PASS**，0 Skip。
- 上述专项中的真实 PostgreSQL G12 检查：**12/12 PASS**。
- 随机 Property Sequence：**200/200 Iteration PASS**。
- High Concurrency：**100 个同时 Settlement PASS**，FB 保持守恒。
- 使用真实 PostgreSQL 与完整 G9/G11 Restart Fixture 的 Go 完整回归：**379/379 PASS**，0 Fail，0 Skip。
- 最终 `go test -race ./... -count=1`：重置专用 Repeat-run G9 Fixture Database 后，全部 14 个 Go Package **PASS**。
- 第一次 Race 重跑复用了此前已变更的 G9 Database，并以 `ITEM_ALREADY_EQUIPPED` 失败；隔离 Schema 重置后，最终完整 Race 运行通过。没有弱化或跳过测试。
- `go vet ./...`：**PASS**。
- 主 Node Suite：**24/24 PASS**。
- G1：**9/9 PASS**；G2：**13/14**，唯一失败仍是受限制 Legacy Archive 导致的 `Atlas Determinism Test` 环境失败；G3：**21/21 PASS**；G4：**7/7 PASS**；G5：**3/3 PASS**；G6：**6/6 PASS**；G7：**6/6 PASS**；G8：**6/6 PASS**。Node 合计：**95/96**，没有新增失败。
- TypeScript Typecheck：**PASS**。
- Web Production Build：**PASS**，保留既有 Vite Large-chunk Warning。
- Windows amd64、Linux amd64 和 macOS arm64 Game Server Build：**PASS**。

### Security 与限制

Browser/Client 不能设置 FB Balance、Ledger Entry、Trade Completion、Fee、Receipt 或 Settlement Outcome。生产代码不存在 Trade Funding Shortcut。Test Balance 仍只存在于 `_test.go` Fixture，并使用守恒的 SYSTEM Liability Entry。

G12 不实现 Marketplace、Auction、Escrow、Trade UI、Blockchain RPC、On-chain Settlement、Wallet、Deposit、Withdrawal、Contribution、Mining、自动 Trade Reversal、玩家持有城池/大陆的代码或 G13。Stage Close Commit、Integration、Issue Update 与 Public Mirror 证据记录在 Issue #8。X Draft 未发布。
