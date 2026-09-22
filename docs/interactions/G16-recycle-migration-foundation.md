# G16 Recycle Migration Foundation — Interaction Record

Status: **Technical and manual acceptance PASS; PR merged into canonical**

## English — Primary

### Request and execution boundary

The user authorized G16 as the only development stage after G15 Stage Close: create the feature branch and private Issue, audit G9–G15, implement an internal recycle foundation, test it on isolated PostgreSQL fixtures, prepare bilingual records and a Public Mirror candidate, then stop for manual acceptance. Commit, push, PR, merge, Public Mirror sync, X publication, Mining, G17, and changes to real player assets are outside this authorization.

The original canonical checkout contained untracked files from the preceding user-requested local skill installation. Those files were preserved. G16 uses a separate clean worktree at the requested canonical base commit. Public Mirror and the G14 stash were inspected without modification. private G16 issue records the bilingual G16 scope.

### Decisions

1. Reuse `character_inventory_items`, `characters`, and `trade_item_locks`; add lifecycle revision/consumed metadata instead of creating a second inventory.
2. Permit no production recycle rules until a canonical, verifiable item bill of materials exists. The G16 internal test rule verifies the 50% ceiling without opening a gameplay entry.
3. Allow only the registered test material `MATERIAL_RECYCLE_SCRAP`; forbid Black Iron Ore and keep FB/Contribution receipt fields fixed at zero.
4. Use a single PostgreSQL transaction for item consumption, material instances, Reputation account/entry, receipt, and audit. Keep permanent item consumption state to reject resurrection.
5. Make Reputation non-transferable; expose fields for future period caps and diminishing returns without choosing final economic values.
6. Test exact operation replay, conflicting replay, 100 same-item requests, 100 distinct-item requests, repository reopen, rollback injection, migration, and reconciliation.
7. Record the historical G15 migration test's version expectation separately from the G16 migration test, so each test checks its intended upgrade boundary.

### Open integration gates

Canonical production recipe evidence, full economic balance, Reputation anti-farm parameters, protected external-asset policy, and a real gameplay adapter require later review. No production player asset is exercised in G16.

### Stage Close evidence and correction record

The original local normal and Race suites each passed 492/492. A new reusable GitHub Actions workflow first needed a fixture-path correction; the next Linux run reached the complete suite and revealed one failing PostgreSQL receipt replay test in both normal and Race runs (491 pass / 1 fail each). Investigation reproduced the discrepancy with a nanosecond-bearing timestamp: Go's first response retained nanoseconds, PostgreSQL stored microseconds, and reload used a different timezone representation. The existing test was strengthened to require exact full-receipt and serialized equality for first response, same-process replay, direct reload, and restart replay. The G16 service canonicalizes request time to UTC microseconds; the settlement adapter returns the database-persisted receipt from the same transaction. No economic award rule changed.

The workflow now uses `pipefail` for both Go test pipelines, so a failing command fails its own step; zero-skipped enforcement remains. Final GitHub Actions run completed 6/6 jobs successfully: Go 492/492, Race 492/492, zero skips, Vet, and Linux/Windows/macOS builds. Human approval then authorized merge-commit integration of private G16 PR into canonical. A separate technical-debt issue records the possible G13–G15 timestamp risk; it is an audit follow-up, not G17 implementation. The G14 stash was preserved. X publication and G17 work were not authorized.

## 中文 — 完整版本

### 请求与执行边界

用户授权 G16 作为 G15 Stage Close 后唯一开发阶段：创建指定 Feature Branch 与 Private Issue，审计 G9–G15，实现内部回收基础，在隔离 PostgreSQL Fixture 上测试，准备双语记录及 Public Mirror Candidate，然后停止等待人工验收。Commit、Push、PR、Merge、Public Mirror 同步、X 发布、Mining、G17 和真实玩家资产改动均不在授权范围内。

原 canonical Checkout 含有上一轮用户要求本地安装技能产生的未跟踪文件，这些文件保持原样。G16 在指定 canonical Base Commit 建立的独立干净 Worktree 中进行。Public Mirror 与 G14 stash 只检查，不修改。private G16 issue 用双语记录 G16 范围。

### 开发决策

1. 复用 `character_inventory_items`、`characters` 和 `trade_item_locks`；增加生命周期版本及已消费元数据，不建立第二套库存。
2. 在出现规范且可核验的物品材料清单前，不启用生产回收规则。G16 内部测试规则验证 50% 上限，但不开放真实玩法入口。
3. 只允许已注册的测试材料 `MATERIAL_RECYCLE_SCRAP`；禁止黑铁矿石，并将 Receipt 的 FB/Contribution 字段固定为零。
4. 在单一 PostgreSQL 事务中完成物品消费、材料实例、Reputation 账户/Entry、Receipt 与审计。永久保留物品消费状态，拒绝已消费 ID 复活。
5. Reputation 不可转让；预留未来周期上限和递减收益字段，不擅自决定最终经济数值。
6. 测试相同操作重放、冲突重放、同一物品 100 请求、不同物品 100 请求、重开 Repository、回滚注入、迁移与对账。
7. 将历史 G15 迁移测试的版本预期与 G16 迁移测试分别限定在各自升级边界，避免新迁移使旧测试误报。

### 后续集成门槛

规范生产配方证据、完整经济平衡、Reputation 防刷参数、受保护外部资产策略以及真实玩法适配器都需要后续审查。G16 不操作生产玩家资产。

### Stage Close 证据与修正记录

最初本地普通 Go 与 Race 测试各为 492/492 通过。新建的可复用 GitHub Actions Workflow 先修正了一处 Fixture 路径配置；之后 Linux Runner 跑到完整测试，在普通与 Race 中均发现一个 PostgreSQL Receipt 重放失败（各 491 通过 / 1 失败）。通过带纳秒的时间重现差异后，确认 Go 首次响应保留纳秒、PostgreSQL 仅保存微秒，重新加载还有不同的时区表示。原测试被强化为首次响应、同进程重放、直接重新读取、重启重放之间完整 Receipt 及序列化内容均精确相等。G16 Service 将请求时间规范到 UTC 微秒；结算适配器在同一事务中返回数据库持久化后的 Receipt。经济奖励规则没有改变。

Workflow 的普通测试与 Race 管道均启用 `pipefail`，失败的测试命令会直接使对应 Step 失败；零跳过检查继续保留。最终 GitHub Actions run 的 6/6 Jobs 全部成功：Go 492/492、Race 492/492、0 跳过、Vet 及 Linux/Windows/macOS 构建。人工批准后，private G16 PR 以 Merge Commit 集成到 canonical。独立技术债 Issue 记录 G13–G15 可能存在的同类时间风险；它是后续审计，不是 G17 实现。G14 stash 保持原样。X 发布和 G17 开发未获授权。
