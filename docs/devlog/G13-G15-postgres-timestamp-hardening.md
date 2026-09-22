# G13–G15 PostgreSQL Timestamp Replay Hardening

Status: **Technical and manual acceptance PASS · merged into canonical · public-safe reliability record**

## English — Primary

### Why this follow-up was needed

G16's first executable GitHub CI run exposed a real mismatch between a newly returned recycle receipt and its PostgreSQL replay. The G16 receipt was repaired at its persistence boundary. A separate audit then checked the independent G13 Contribution Ledger, G14 refund/reversal, and G15 system-spend paths for the same class of risk. This work is reliability hardening before G17 planning; it does not implement G17.

### Observed root cause

Real PostgreSQL regression tests reproduced exact-equality failures in all three paths. The first Go response held a UTC timestamp, while a persisted row reloaded through PostgreSQL was decoded in the local timezone. The two values described the same instant, but their `time.Time` representations differed. In one G13 sample, both Unix seconds and nanoseconds were identical and the precision difference was **0 ns**; **TIMEZONE** was the actual observed cause. Non-canonical nanosecond timestamps could additionally exceed PostgreSQL's microsecond precision on hosts with finer clock resolution.

The affected persisted records include G13 Contribution accounts, entries, audit events, and linked FB ledger transactions; G14 refund and reversal compensations and their linked ledger/audit records; and G15 eligible and non-eligible system-spend records. Rule version, IDs, amounts, status, links, metadata, and all other current immutable response fields remain in the exact comparisons.

### Repair

New timestamps are normalized to UTC and truncated to microseconds before insertion. First responses read back the PostgreSQL-persisted record, while replay and snapshot readers normalize decoded timestamps to UTC. Regression tests use complete-record exact equality across first response, same-process replay, reopened-store replay, and snapshot/reload. No tolerance, ignored timestamp, skipped test, or economic-rule adjustment was introduced.

### Verification and safety

Focused G13, G14 refund/reversal, G15 eligible/non-eligible, and G16 regressions passed. The complete local Go suite and Race suite each passed **500/500** with **0 skipped tests**; Race used independent clean databases. Formal GitHub Actions pull-request CI passed all **6/6 jobs**: normal Go **500/500**, Race **500/500**, **0 skipped**, Go vet, and Linux, Windows, and macOS builds. A changed-content secret scan passed.

The first local Race attempt had reused a previously populated G9 runtime database and failed with `ITEM_ALREADY_EQUIPPED`. Repeating the full Race run on isolated fresh databases passed 500/500. Database isolation and deterministic reset remain a separate technical-debt task to finish before expanding PostgreSQL integration or economy CI.

No economic behavior or amount changed. There was no database schema change, historical ledger modification, or migration. No Mining, Black Iron issuance, G17 branch, or G17 implementation was created. This hardening is not a game release or an X publication.

## 中文 — 完整审核版

### 为什么需要本次后续修复

G16 首次真正执行的 GitHub CI 发现新返回的回收 Receipt 与 PostgreSQL 重放结果不一致。G16 已在自己的持久化边界修复。随后专项审计 G13 Contribution Ledger、G14 Refund/Reversal 和 G15 System Spend 各自独立的路径，检查是否存在同类风险。本次工作属于 G17 规划前的可靠性硬化，不实现 G17。

### 真实观察到的根因

真实 PostgreSQL 回归测试在三条路径都复现了精确相等失败。首次 Go 响应的时间采用 UTC，PostgreSQL 持久化行重新读取后却被解码为本地时区。两个值代表同一时刻，但 `time.Time` 的内部表示不同。在一个 G13 样本中，Unix 秒与纳秒完全相同，精度差为 **0 ns**；实际观察到的原因是 **TIMEZONE**。在时钟分辨率更高的主机上，未规范化的纳秒值还可能超过 PostgreSQL 的微秒精度。

受影响的持久化记录包括 G13 Contribution Account、Entry、Audit Event 与关联 FB Ledger Transaction；G14 Refund/Reversal Compensation 及其关联 Ledger/Audit Record；以及 G15 合格和非合格 System Spend Record。规则版本、ID、数量、状态、关联 ID、Metadata 和其他当前不可变响应字段继续参与完整记录的精确比较。

### 修复

新时间在 INSERT 前统一为 UTC 并截取到微秒。首次响应重新读取 PostgreSQL 已保存的记录；回放与快照读取把解码时间统一为 UTC。回归测试对首次响应、同进程重放、重开 Store 后重放和 Snapshot/Reload 进行完整记录精确相等比较。没有引入容差、忽略时间、跳过测试或调整经济规则。

### 验证与安全边界

G13、G14 退款/撤销、G15 合格/非合格消费及 G16 专项回归均通过。本地完整 Go 测试与 Race 测试各为 **500/500 PASS**、**跳过 0**；Race 使用独立干净数据库。正式 GitHub Actions Pull Request CI **6/6 Jobs PASS**：普通 Go **500/500**、Race **500/500**、**跳过 0**、Go vet，以及 Linux、Windows、macOS Build。变更内容 Secret Scan 通过。

首轮本地 Race 复用了此前已有数据的 G9 runtime 数据库，出现 `ITEM_ALREADY_EQUIPPED`。使用全新隔离数据库重新执行完整 Race 后，500/500 通过。数据库隔离与确定性重置作为独立技术债，在扩大 PostgreSQL integration 或 economy CI 之前解决。

经济行为与任何数量均未改变。数据库 Schema、历史账本数据未修改，无需 Migration。没有创建 Mining、黑铁矿石发放、G17 Branch 或 G17 实现。本次硬化不是游戏发布，也没有发布 X。
