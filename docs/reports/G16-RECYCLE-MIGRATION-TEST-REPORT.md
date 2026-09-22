# G16 Recycle Migration Foundation — Technical Test Report

Status: **Technical and manual acceptance PASS · final GitHub CI 6/6 PASS**

## English — Primary

### Evidence and method

The final regular and Race runs used separate, newly created local PostgreSQL 16 test databases for persistence and G9/G11 restart cases. Go JSON `pass`, `fail`, and `skip` test events, including named subtests, determine the counts; package summaries are excluded. No production database or real player record was opened. The final regular run passed **492/492**, failed **0**, skipped **0**. The final Race run passed **492/492**, failed **0**, skipped **0**, and emitted no race report. The G16 PostgreSQL suite contributed **20/20 passing events**; the rule package contributed **2/2 passing unit events**, including 50 generated return-ratio cases.

### G16 acceptance evidence

| Check | Result |
| --- | --- |
| Server rule, exact template, unknown/disabled rejection, output allowlist, Black Iron rejection | PASS |
| Material return ceiling | PASS; 5 of 10 verified test units, all generated cases ≤ 50% |
| Ownership, item revision, equipped state, trade lock and active-offer checks | PASS |
| Atomic item consume, material inventory credit, Reputation account/entry, receipt, audit | PASS; one PostgreSQL transaction |
| FB, Contribution, Black Iron awards | 0 / 0 / 0; receipt fields and database constraints |
| Same operation replay and conflicting operation | PASS; exact replay returns original receipt, changed intent rejected |
| Same item concurrency | 100 requests: 1 settlement, 99 safe consumed rejections |
| Different valid items | 100/100 settled |
| Different players | 100/100 settled without a global operation lock |
| Restart retry | PASS; reopened repository returned original receipt, even after rule disablement |
| Rollback safety | PASS at 8 injected points, including all 7 required boundaries |
| Reconciliation | PASS; detected balance corruption, missing material inventory, and consumed item without receipt |
| Duplicate and immutable history protection | PASS; duplicate receipt/entry rejected and receipt update rejected |
| Fresh migration; G15→G16 upgrade | PASS; migration 0008 applied and repeated safely |
| Economy graph | PASS for enabled G16 graph; no profitable material loop or FB/Contribution/ore creation loop |
| G11–G15 regression | PASS within the full Go suite |

The historical G15 G14→G15 migration test initially failed because it invoked all current migrations and asserted that the total remained seven. It was scoped to migrations 0001–0007, preserving its original acceptance question; the new G15→G16 test checks version eight. The focused pair and final full suite passed. The first isolated G2 Node run had an additional missing local generated bundle; linking the existing local test bundle read-only removed that environmental error. The final G2 Node result is **13/14**, with only the pre-existing restricted Atlas source limitation. These earlier failures are retained here and are not represented as passing attempts.

### Toolchain and release boundary

`go vet ./...` passed. Windows amd64, Linux amd64, and macOS arm64 Go builds passed. Windows runtime is **NOT_RUN** because the available Windows 11 VM is suspended. Secret, personal-information, private-email, local-path, legacy-source, and restricted-asset scans of the changed-file manifest passed. The status card is based on the 492/492 local runs and observed concurrency outcomes; its “manual review pending” label is the historical capture state. Afterward, GitHub Actions found and helped repair an exact receipt timestamp replay failure. The final GitHub run passed 6/6 jobs, including normal and Race 492/492 each with zero skips. No real gameplay recycle entry, Mining, real player asset mutation, or X publication occurred.

Production recycle remains disabled until canonical crafting inputs can verify real return ratios. Reputation caps and diminishing returns are policy boundaries, not final values. Current material reconciliation expects credited instances to remain in inventory; a later material-consumption writer will require immutable spend evidence and a revised reconciliation rule. These are explicit later integration gates, not claims of live economy readiness.

## 中文 — 完整版本

### 证据与方法

最终普通测试与 Race 测试分别使用新建的本地 PostgreSQL 16 测试数据库，隔离持久化测试及 G9/G11 重启场景。计数取自 Go JSON 的 `pass`、`fail`、`skip` 测试事件，包含命名子测试，不把 Package Summary 算作测试。没有打开生产数据库或真实玩家记录。最终普通测试 **492/492 通过**、**0 失败、0 跳过**；最终 Race **492/492 通过**、**0 失败、0 跳过**，没有数据竞争报告。G16 PostgreSQL 套件贡献 **20/20 个通过事件**；规则包贡献 **2/2 个单元测试事件**，其中包含 50 个生成的返还比例输入案例。

### G16 验收证据

| 检查 | 结果 |
| --- | --- |
| 服务器规则、精确模板、未知/禁用拒绝、输出允许清单、黑铁拒绝 | PASS |
| 材料返还上限 | PASS；测试投入十单位返还五单位，全部生成案例 ≤ 50% |
| 归属、物品版本、装备状态、交易锁及活跃报价检查 | PASS |
| 原子物品消费、材料库存入账、Reputation 账户/Entry、Receipt、审计 | PASS；单一 PostgreSQL 事务 |
| FB、Contribution、黑铁矿石发放 | 0 / 0 / 0；Receipt 字段及数据库约束 |
| 同一操作重放与冲突操作 | PASS；完全相同重放返回原 Receipt，变更 Intent 被拒绝 |
| 同一物品并发 | 100 请求：一次结算、99 次安全的已消费拒绝 |
| 不同合法物品 | 100/100 结算 |
| 不同玩家 | 100/100 结算；没有全局操作锁 |
| 重启重试 | PASS；重开 Repository 后返回原 Receipt，规则随后禁用也一样 |
| 回滚安全 | 8 个故障注入点 PASS，包含要求的全部 7 个边界 |
| 对账 | PASS；发现余额破坏、材料库存缺失及已消费物品缺少 Receipt |
| 重复与不可变历史保护 | PASS；拒绝重复 Receipt/Entry 和修改 Receipt |
| 新库迁移；G15→G16 升级 | PASS；迁移 0008 正确应用且可重复执行 |
| 经济图 | 已启用 G16 图 PASS；无盈利材料闭环或 FB/Contribution/矿石创造闭环 |
| G11–G15 回归 | 完整 Go 套件内 PASS |

历史 G15 的 G14→G15 迁移测试首轮失败，原因是它调用了所有当前迁移，却仍断言总数为七。该测试已限定只运行迁移 0001–0007，保持原来的验收问题；新增 G15→G16 测试验证版本八。两个专项测试及最终全套测试通过。独立 Worktree 首轮 G2 Node 测试还缺少本地生成的 Bundle；只读链接现有本地测试 Bundle 后排除此环境错误。最终 G2 Node 结果为 **13/14**，唯一失败仍是既有的受限 Atlas 源限制。这些早期失败保留在记录中，没有被改写为通过。

### 工具链与发布边界

`go vet ./...` 通过。Windows amd64、Linux amd64、macOS arm64 Go Build 通过。可用 Windows 11 VM 处于 suspended，因此 Windows Runtime 为 **NOT_RUN**。变更文件清单的密钥、个人信息、私人邮箱、本机路径、旧源码及受限资源扫描均通过。状态卡依据本地 492/492 测试及已观察到的并发结果制作；图上的“manual review pending”是拍摄时的历史状态。之后 GitHub Actions 发现并推动修复了 Receipt 时间精确重放故障。最终 GitHub run 的 6/6 Jobs 通过，包括普通与 Race 各 492/492、零跳过。没有真实玩法回收入口、Mining、真实玩家资产改动或 X 发布。

在规范制造投入可证明真实返还比例之前，生产回收保持禁用。Reputation 上限及递减收益只是策略边界，不是最终数值。当前材料对账要求 Credit 对应实例仍在库存；未来材料消费 Writer 需要不可变消费证据及更新后的对账规则。这些是后续独立集成门槛，不表示真实经济玩法已经就绪。
