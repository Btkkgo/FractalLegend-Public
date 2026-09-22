# G13–G15 PostgreSQL Timestamp Hardening — Interaction Record

Status: **Technical and manual acceptance PASS · private merge complete · public export subject to safety gates**

## English — Primary

### Request and boundary

After G16 CI exposed a timestamp replay mismatch, the user requested a focused audit of the independent G13–G15 PostgreSQL persistence paths. The first phase authorized local diagnosis and regression tests, with no commit, push, public sync, or G17 work. After local acceptance, a separate stage authorized the fix commit, pull request, six-job CI gate, merge, public-safe documentation, and sanitized public sync. Mining and X publication remained outside the request.

### Evidence and decisions

1. A new real-PostgreSQL regression first failed on complete-record exact equality for G13 posting, G14 refund and reversal, and G15 eligible and non-eligible spend. The first response used UTC; replay returned a local-timezone representation of the same instant. The measured precision difference in the observed local sample was zero, so timezone was the demonstrated mismatch. PostgreSQL's microsecond precision also made submicrosecond Go timestamps a cross-host risk.
2. Keep the existing economic and schema contracts. Canonicalize new timestamps to UTC microseconds, return the stored PostgreSQL value on the first call, and normalize replay/snapshot reads to UTC. Do not use tolerance or remove immutable fields from equality assertions.
3. Verify first response, same-process replay, reopened-store replay, and direct snapshot/reload across complete records. Preserve linked IDs, rule versions, amounts, statuses, metadata, and audit relationships.
4. Run focused G13–G16 regressions, complete Go and Race suites, Vet, and Linux/Windows/macOS builds. The final local and pull-request runs passed 500/500 normal and 500/500 Race with zero skipped tests; six CI jobs succeeded.
5. Record the first local Race failure honestly: a reused G9 runtime database contained state from the normal suite and produced `ITEM_ALREADY_EQUIPPED`. A fresh isolated Race database passed the entire suite. Isolation and repeatability remain separate technical debt before larger PostgreSQL integration/economy CI.

### Outcome

The fix was merged into canonical after human acceptance and a green pull-request CI run. No economic behavior, schema, historical ledger row, or migration changed. The separate database-isolation debt stays open. The sanitized Public Mirror has independent history and receives only reviewed project-owned source, tests, and curated bilingual records. No X post or G17 implementation was made.

This document is a curated engineering decision and evidence record, not a transcript of private conversation.

## 中文 — 完整审核版

### 请求与边界

G16 CI 发现时间重放不一致后，用户要求专项审计 G13–G15 各自独立的 PostgreSQL 持久化路径。第一阶段只授权本地诊断与回归测试，不允许 Commit、Push、Public Sync 或 G17 工作。本地验收后，独立 Stage Close 授权修复 Commit、PR、六项 CI 门槛、合并、可公开文档及脱敏 Public Mirror 同步。Mining 与 X 发布仍不在请求范围内。

### 证据与决策

1. 新增的真实 PostgreSQL 回归测试先在 G13 Posting、G14 Refund/Reversal，以及 G15 合格/非合格消费的完整记录精确比较上失败。首次响应采用 UTC，重放采用同一时刻的本地时区表示。本地观察样本的精度差为零，因此真实复现的差异是时区。PostgreSQL 的微秒精度也使亚微秒 Go 时间在其他主机存在风险。
2. 保留现有经济与数据库 Schema 合同。新时间统一为 UTC 微秒；首次调用返回 PostgreSQL 实际保存的值；重放与快照读取统一转为 UTC。不使用时间容差，不从精确比较中移除不可变字段。
3. 对首次响应、同进程重放、重开 Store 后重放与直接 Snapshot/Reload 检查完整记录。关联 ID、规则版本、数量、状态、Metadata 与审计关联均保留在验证范围内。
4. 执行 G13–G16 专项回归、完整 Go 与 Race、Vet，以及 Linux/Windows/macOS Build。最终本地与 Pull Request CI 中，普通 Go 500/500、Race 500/500、跳过 0；六个 CI Job 均成功。
5. 如实保留首轮本地 Race 失败：复用的 G9 runtime 数据库包含普通测试留下的状态，出现 `ITEM_ALREADY_EQUIPPED`。用全新隔离数据库重跑完整 Race 后通过。数据库隔离和重复执行一致性在扩大 PostgreSQL integration/economy CI 前作为独立技术债解决。

### 结果

在人工验收与 Pull Request CI 全绿后，修复合入 canonical。经济行为、数据库 Schema、历史账本行均未改变，也无需 Migration。独立的数据库隔离技术债保持开放。Sanitized Public Mirror 使用独立 Git History，只接收经过审核的项目自有源码、测试及双语整理记录。没有发布 X，也没有实现 G17。

本文件是经过整理的工程决策与证据记录，不是私人对话的逐字稿。
