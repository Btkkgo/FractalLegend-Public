# G15 Eligible System Spend — Technical Test Report

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

## English — Primary

### Environment and count method

The Go module was tested against local, real PostgreSQL 16. Each final full run used fresh databases for the shared persistence tests and the G9/G11 server-restart fixtures. Test counts come from Go JSON test events and include named subtests; package summaries were not counted as tests. The final regular run and final Race run each reported **470 passed, 0 failed, 0 skipped**. Temporary database connection values and raw logs are intentionally excluded from public evidence.

### G15 focused checks

The final Go run includes **20/20 G15 real-PostgreSQL test events**, including seven named rollback subtests, and **3/3 G15 unit test events**. The real-database property sequence completed **400/400 iterations** of deterministic spend/replay/refund operations; the separate rule-resolution unit property sequence completed **400/400 iterations**. The real-database sequence checked FB conservation, Contribution entitlement, refund conservation, no double debit or credit, saved rule version, and reconciliation after 400 spends. Recovery debt was checked in its dedicated end-to-end test and in the G14 regression; the property sequence did not itself simulate prior Contribution consumption.

Concurrency checks passed for **100 identical operation requests** producing one spend, one FB debit, and one Contribution economic effect; **100 distinct concurrent eligible spends**; and **100 mixed operations** consisting of 50 new spends and 50 partial refunds against an original spend. The partial-refund end-to-end test confirmed 100 spent, 40 refunded, 60 net FB spend and 60 net Contribution entitlement. The full-refund and recovery-debt test confirmed 80 debt after 80 points of a 100-point earning had been synthetically used, then 30 remaining debt and zero available points after a new eligible credit of 50. Seven injected failures across FB/Contribution posting, SystemSpend insertion, and pre-commit each rolled back and permitted a single clean retry. Lost commit response, repository reopen, duplicate replay, conflicting replay, non-eligible zero-credit spend, G14 refund-policy snapshot, G14-schema upgrade, and rule-mismatch reconciliation also passed.

Client JSON decoding into `Intent` was rejected. Tests rejected unknown and disabled Producers, forged eligibility/Contribution/rule claims in JSON, unsafe references, missing producer reference, zero amount, cross-player and mismatched-account requests, insufficient FB, duplicate reference, changed operation intent, and disallowed G15 refunds. The typed registry and PostgreSQL coordinator also reject invalid rule configuration and non-eligible refund configuration. Existing G11/G12/G13/G14 tests passed in the full regression; G12 player trade remained outside eligible spend.

### Full regression, toolchain, and limits

| Check | Observed result |
| --- | --- |
| Final `go test -json -p 1 ./... -count=1 -timeout=12m` | **470/470 PASS**, zero skipped; real PostgreSQL and G9/G11 restart environments configured |
| Final `go test -race -json -p 1 ./... -count=1 -timeout=15m` | **470/470 PASS**, zero skipped; fresh real PostgreSQL databases |
| `go vet ./...` | **PASS** |
| Windows amd64 `CGO_ENABLED=0 go build ./...` | **PASS**, cross-build on macOS |
| Linux amd64 `CGO_ENABLED=0 go build ./...` | **PASS**, cross-build on macOS |
| macOS arm64 `CGO_ENABLED=0 go build ./...` | **PASS**, local platform build |
| Windows runtime | **NOT RUN**; the available Windows 11 VM was suspended |
| Selected G1–G8 Node tests | **71/72**, one existing G2 Atlas Determinism failure from unavailable restricted Legacy material |
| G2 Node tests alone | **13/14**, same Atlas failure |

The earlier recorded broader Node result was **95/96**. This run did not recreate that exact broader command; it independently confirmed the same sole G2 restricted-asset failure in the selected G1–G8 suite. No restricted asset was imported to increase the count.

### Failed attempts retained in the evidence trail

An initial Race invocation accidentally omitted the G9/G11 runtime environment variable names those tests read; it reported 468 passes and two skips and was not accepted as a full Race result. The corrected invocation exposed a real race in the historical G9 persistence test fixture: direct monster edits overlapped the background AI tick. The fixture now locks its direct world edits and uses the slowest supported tick; its focused Race test passed on a fresh database. Another full Race attempt used a test database name rejected by the repository's safety guard and failed before affected PostgreSQL test bodies ran. The final run used a permitted fresh test database name and completed 470/470 without skips or race reports. These earlier attempts are not reclassified as passes.

### Release boundary

Technical and manual acceptance passed, but this is not a production economy launch. The technical-acceptance test run did not include a live gameplay Producer, Windows runtime result, commit, push, PR, Public Mirror sync, or X publication. Stage Close actions are tracked separately; X publication remains unauthorized.

---

## 中文 — 完整对应版本

### 环境与计数方法

Go Module 使用本地真实 PostgreSQL 16 测试。最终每次完整运行都为共享持久化测试和 G9/G11 Server Restart Fixture 使用全新数据库。测试数量取自 Go JSON Test Event，包含命名子测试；Package Summary 不计作测试。最终普通运行和最终 Race 运行均报告 **470 通过、0 失败、0 跳过**。临时数据库连接值和原始 Log 有意不纳入公开证据。

### G15 专项检查

最终 Go 运行包含 **20/20 个 G15 真实 PostgreSQL Test Event**（包括七个命名回滚子测试）和 **3/3 个 G15 Unit Test Event**。真实数据库 Property Sequence 完成 **400/400 轮**确定性的消费、重放与退款操作；独立规则解析 Unit Property Sequence 完成 **400/400 轮**。真实数据库序列核查 FB 守恒、Contribution 权益、退款守恒、无重复扣款或发放、保存的规则版本，以及 400 笔消费后的 Reconciliation。Recovery Debt 由独立端到端测试和 G14 回归覆盖；这组 Property Sequence 本身没有模拟此前使用 Contribution 的情形。

并发检查通过：**100 个相同 Operation 请求**最终只产生一笔 Spend、一次 FB Debit 和一次 Contribution 经济效果；**100 笔不同的并发合格消费**；以及 **100 个混合操作**，包括 50 笔新消费和针对原消费的 50 笔部分退款。部分退款端到端测试确认消费 100、退款 40，最终净 FB 消费与净 Contribution 权益均为 60。完整退款及 Recovery Debt 测试确认：100 点收益中有 80 点经 Synthetic Fixture 模拟使用后，形成 80 点债务；随后新增 50 点合格收益，债务降为 30、可用点数仍为零。跨 FB/Contribution 入账、SystemSpend 插入和 Commit 前的七个故障注入点均回滚，并允许一次干净重试。Commit 响应丢失、Repository 重开、重复重放、冲突重放、非合格零收益消费、G14 退款策略快照、G14 Schema Upgrade 和规则不匹配对账也都通过。

`Intent` 拒绝客户端 JSON 解码。测试拒绝未知与 Disabled Producer、JSON 中伪造的资格/Contribution/规则声明、不安全 Reference、缺少 Producer Reference、零金额、跨玩家与账户不匹配请求、FB 不足、重复 Reference、同一 Operation Intent 改变，以及 G15 不允许的退款。Typed Registry 和 PostgreSQL Coordinator 也拒绝无效规则配置和非合格退款配置。G11/G12/G13/G14 既有测试在全量回归中通过；G12 玩家交易仍不进入合格消费流程。

### 完整回归、工具链与限制

| 检查 | 实测结果 |
| --- | --- |
| 最终 `go test -json -p 1 ./... -count=1 -timeout=12m` | **470/470 PASS**，零跳过；配置真实 PostgreSQL 和 G9/G11 Restart 环境 |
| 最终 `go test -race -json -p 1 ./... -count=1 -timeout=15m` | **470/470 PASS**，零跳过；使用全新真实 PostgreSQL 数据库 |
| `go vet ./...` | **PASS** |
| Windows amd64 `CGO_ENABLED=0 go build ./...` | **PASS**，在 macOS 上 Cross Build |
| Linux amd64 `CGO_ENABLED=0 go build ./...` | **PASS**，在 macOS 上 Cross Build |
| macOS arm64 `CGO_ENABLED=0 go build ./...` | **PASS**，本机平台 Build |
| Windows Runtime | **NOT RUN**；可用的 Windows 11 VM 处于 Suspended 状态 |
| 选定的 G1–G8 Node 测试 | **71/72**；唯一失败是缺少受限 Legacy 素材导致的既有 G2 Atlas Determinism Test |
| 单独 G2 Node 测试 | **13/14**；同一 Atlas 失败 |

此前记录的更大范围 Node 结果为 **95/96**。本轮没有复现那条完全相同的更大范围命令；本轮通过选定 G1–G8 测试独立确认仍只有同一个 G2 受限素材失败。没有为提高数量而导入受限素材。

### 保留在证据链中的失败尝试

第一次 Race 命令误用了 G9/G11 Runtime 测试实际读取的环境变量名，结果为 468 通过、2 跳过，因此没有被当作完整 Race 结果。修正变量名后，发现历史 G9 Persistence Test Fixture 的真实数据竞争：直接修改 Monster 时后台 AI Tick 也在读写。现已为 Fixture 的直接 World 修改加锁，并使用支持范围内最慢 Tick；其单项 Race 测试在新数据库上通过。另一次完整 Race 采用了不符合仓库安全命名条件的测试数据库名，受影响的 PostgreSQL Test Body 尚未执行便被拒绝。最终运行使用符合要求的全新数据库名，470/470 通过，没有跳过或 Race 报告。这些早期尝试不会被改写成通过。

### 发布边界

技术验收与人工验收均已通过，但不代表生产经济上线。技术验收测试时没有启用真实 Gameplay Producer，也没有 Windows Runtime 结果、Commit、Push、PR、Public Mirror 同步或 X 发布。Stage Close 操作另行记录；X 发布仍未获授权。
