# G16 — Recycle Migration Foundation

Status: **Technical acceptance PASS · manual acceptance PASS · PR merged into canonical**

## English — Primary

### Why

Equipment recycling must migrate away from legacy currency rewards before Mining begins. A recycle operation must not mint FB, Contribution, or Black Iron Ore, and material return must not turn a verifiable input into a profitable loop. Reputation can record participation without becoming transferable currency.

### What

G16 adds an internal `RecycleIntent`, typed versioned rule registry, exact-template matching, a 50% same-material return ceiling, a fixed material allowlist, and policy fields for future Reputation period caps and diminishing returns. There are no enabled production rules. The only active rule is a test fixture, because canonical production crafting inputs are not yet available for verification.

Migration 0008 adds permanent item-instance revision and consumed state to the existing inventory lifecycle, a minimal Reputation account and immutable entry, immutable recycle receipts and material credits, and immutable audit events. A consumed ID cannot reenter inventory. No FB or Contribution ledger was changed or duplicated.

### Architecture and economy invariants

An internal service rejects JSON-decoded client intent, resolves the rule on the server, and passes it to the PostgreSQL coordinator. One transaction locks the owning character and item, validates ownership, revision, inventory/equipment state, template, trade locks and active offers, consumes the instance, credits materials to the existing inventory and Reputation to its account, saves a versioned rule snapshot and receipt, and writes an audit event. Recycle awards FB = 0, Contribution = 0, Black Iron Ore = 0. Unknown and disabled rules fail closed. An operation lock, unique keys, and permanent consumed state prevent duplicate settlement. Exact retries return the original result after restart; changed requests conflict.

### Security

The server calculates every output. No gameplay recycle route, NPC, UI, or legacy adapter invokes this service. Only isolated test players and PostgreSQL databases were used. Reconciliation checks receipt-to-consumed-item, material credit-to-inventory, Reputation entry-to-balance, snapshot, zero forbidden awards, and audit links. Database constraints reject duplicate item receipts and Reputation entries; immutable triggers reject history edits.

### Tests

The final real-PostgreSQL Go run passed **492/492 test events**, with **0 failures and 0 skips**; the final Race run also passed **492/492**, with no race report. G16 contributed **20/20 PostgreSQL events** plus **2/2 rule unit events**. The property test checked 50 generated input values and rejected returns above 50%, unknown material output, and Black Iron Ore. Eight injected failure points rolled back all effects. Concurrency passed for 100 requests against one item (one settlement, 99 safe rejections), 100 exact retries of one operation, 100 distinct items, and 100 distinct players. Repository reopen returned the original receipt. Fresh migration and G15→G16 upgrade passed. The full suite covered G11 persistence, G12 trade settlement, G13 Contribution, G14 refund/reversal, and G15 system spend. Vet and Windows amd64, Linux amd64, and macOS arm64 builds passed.

G2 Node tests passed **13/14**; the sole failure is the existing restricted Asset Atlas limitation. An initial run in the isolated worktree also lacked a generated local G2 bundle; a read-only link to the existing local test bundle removed that environmental error before the final 13/14 result. Windows 11 VM remains suspended, so Windows runtime is **NOT_RUN**.

### CI discovery and canonical receipt repair

The initial local Go and Race runs each passed **492/492**. G16 then introduced reusable GitHub Actions CI. The first workflow attempt exposed a fixture-path configuration error, which was corrected without changing product behavior. The first executable Linux CI run found a real failure in `TestG16PostgresSuccessfulRecycleReplayConflictRestartAndSnapshot`: both normal and Race suites had **491 pass / 1 fail**. The local macOS clock had happened to supply microsecond-aligned timestamps, so the earlier local pass had not exercised the mismatch.

Diagnosis confirmed that the first Go receipt could contain nanoseconds, PostgreSQL persisted microseconds, and a reload could represent the same time in a different timezone. The first response and replay therefore lacked exact canonical equality. The fix normalizes the request timestamp to UTC microseconds at the G16 service and PostgreSQL boundaries, reads the actual persisted receipt inside the settlement transaction, and returns that database-decoded value as the first response. Regression checks compare every immutable receipt field and its serialized JSON across the first response, same-process replay, repository reload, and restart replay. G13–G15 have separate persistence paths with a possible similar precision risk; a dedicated follow-up audit records them without changing those stages here.

The CI test and Race shell steps now enable `pipefail`, so a failing `go test` fails its step directly while the zero-skipped-test check remains in place. The final GitHub Actions run passed **6/6 jobs**: normal **492/492**, Race **492/492**, **0 skipped**, Vet, and Linux, Windows, and macOS builds. This progression preserves the observed failure and repair; G16 did not pass CI on its first attempt.

### Known issues and not implemented

- Real gameplay recycle NPC/UI not enabled.
- Black Iron Mining not implemented.
- Mining Pool not implemented.
- Mining Block not implemented.
- Final Reputation caps not decided.
- Final recycle balance not decided.
- Ordinals protection not implemented; a protected-asset policy boundary remains for a later stage.
- No real player assets changed.
- Production rules remain disabled until canonical item input evidence exists. Future material spending will need its own immutable consumption evidence before reconciliation can stop requiring current inventory presence.

### Next

The G16 PR was merged after human approval and a verified green CI run. Public Mirror export requires the separate allowlist and privacy gates. X publication, Mining, and G17 remain outside this stage close.

## 中文 — 完整版本

### 原因

Mining 开始前必须把装备回收从旧货币奖励迁移出来。回收不得发行 FB、Contribution 或黑铁矿石；材料返还也不得使可核验投入形成盈利循环。Reputation 可以记录参与，但不能成为可转让货币。

### 完成内容

G16 增加内部 `RecycleIntent`、类型化且版本化的规则注册表、精确模板匹配、同种材料 50% 返还上限、固定材料允许清单，以及未来 Reputation 周期上限和递减收益策略字段。没有启用生产规则。唯一启用的是测试 Fixture，因为目前尚无可核验的规范生产制造投入。

迁移 0008 为现有库存增加永久物品实例版本与已消费状态，建立最小 Reputation 账户及不可变 Entry、不可变回收 Receipt 与材料 Credit、不可变审计事件。已消费 ID 无法重新进入库存。没有改变或复制 FB、Contribution 账本。

### 架构与经济不变量

内部 Service 拒绝直接从客户端 JSON 解码 Intent，在服务器解析规则，再交给 PostgreSQL Coordinator。单一事务锁定归属角色与物品，核验归属、版本、库存/装备状态、模板、交易锁和活跃报价，消费实例，将材料写入现有库存、Reputation 写入账户，保存版本化规则快照与 Receipt，并写入审计事件。回收发放 FB = 0、Contribution = 0、黑铁矿石 = 0。未知或禁用规则默认拒绝。操作锁、唯一键与永久已消费状态防止重复结算。相同重试在重启后返回原结果；变更请求构成冲突。

### 安全

所有输出都由服务器计算。没有真实玩法回收路由、NPC、UI 或旧适配器调用此服务。只使用隔离的测试玩家与 PostgreSQL 数据库。对账核查 Receipt 与已消费物品、材料 Credit 与库存、Reputation Entry 与余额、规则快照、禁止奖励零值及审计关联。数据库约束拒绝重复物品 Receipt 和 Reputation Entry；不可变 Trigger 拒绝修改历史。

### 测试

最终真实 PostgreSQL Go 测试 **492/492 个事件通过**，**0 失败、0 跳过**；最终 Race 也 **492/492 通过**，没有数据竞争报告。G16 包含 **20/20 个 PostgreSQL 事件**及 **2/2 个规则单元事件**。性质测试检查 50 个生成的投入值，并拒绝超过 50% 的返还、未知材料输出与黑铁矿石。八个故障注入点全部回滚。并发验证包括同一物品 100 个请求（一次结算、99 次安全拒绝）、同一操作 100 次完全相同重试、100 件不同物品及 100 名不同玩家。重开 Repository 后返回原 Receipt。新库迁移和 G15→G16 升级通过。完整套件覆盖 G11 持久化、G12 交易结算、G13 Contribution、G14 退款/冲正与 G15 系统消费。Vet 与 Windows amd64、Linux amd64、macOS arm64 构建通过。

G2 Node 测试 **13/14 通过**；唯一失败是既有的受限 Asset Atlas 限制。独立 Worktree 的首轮还缺少本地生成的 G2 Bundle；通过只读链接现有本地测试 Bundle 排除该环境错误后，最终结果为 13/14。Windows 11 VM 仍为 suspended，因此 Windows Runtime 为 **NOT_RUN**。

### CI 发现问题与 Receipt 规范化修复

最初本地 Go 普通测试与 Race 各为 **492/492 通过**。随后 G16 建立可复用的 GitHub Actions CI。第一次 Workflow 尝试暴露测试 Fixture 路径配置错误，修正过程未改变产品行为。首次真正执行的 Linux CI 在 `TestG16PostgresSuccessfulRecycleReplayConflictRestartAndSnapshot` 中发现真实失败：普通与 Race 均为 **491 通过 / 1 失败**。本地 macOS 时钟恰好提供微秒对齐时间，因此此前的本地通过未覆盖该差异。

诊断确认：首次 Go Receipt 可能包含纳秒，PostgreSQL 只保存到微秒，重新读取还可能以不同的时区表示同一时间，因此首次返回与重放缺少精确的规范表示。修复在 G16 Service 与 PostgreSQL 边界把请求时间统一为 UTC 微秒，在结算事务内重新读取数据库实际保存的 Receipt，并将该值作为首次返回。回归测试逐字段及逐字节核对序列化 JSON，覆盖首次返回、同进程重放、Repository 重新读取和重启重放。G13–G15 使用独立持久化路径，可能有类似精度风险；已建立专项后续审计，本轮不修改那些阶段。

CI 的普通测试与 Race Shell Step 现已启用 `pipefail`，`go test` 失败会直接使对应 Step 失败，同时保留零跳过检查。最终 GitHub Actions **6/6 Jobs 通过**：普通测试 **492/492**、Race **492/492**、**0 跳过**、Vet，以及 Linux、Windows、macOS 构建。这份记录保留了真实失败与修复经过；G16 CI 并非首次尝试就通过。

### 已知问题与未实现

- 未启用真实玩法回收 NPC/UI。
- 未实现黑铁 Mining。
- 未实现 Mining Pool。
- 未实现 Mining Block。
- 尚未决定最终 Reputation 上限。
- 尚未决定最终回收平衡。
- 未实现 Ordinals 保护；受保护资产策略边界留待后续阶段。
- 未改变真实玩家资产。
- 在存在规范物品投入证据前，生产规则保持禁用。未来材料消费必须有独立的不可变消费证据，对账才能不再要求材料仍存在于当前库存。

### 下一步

G16 PR 已在人工批准和 CI 全绿确认后合并。Public Mirror 导出仍须通过独立的允许清单与隐私检查。X 发布、Mining 与 G17 均不在本次收尾范围内。
