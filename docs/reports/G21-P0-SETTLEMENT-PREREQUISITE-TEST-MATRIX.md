# G21-P0 Numbered Test Matrix / 编号测试矩阵

Final human acceptance: **PASS**. Planned **38**, executed **38**, PASS **38**, FAIL **0**, PARTIAL **0**, NOT RUN **0**. The 38 logical scenarios are 35 prerequisite cases plus three native executions of the same canonical fixture target. Its two Go test functions remain one logical scenario per OS; the child-process helper is not a scenario. Database tests use synthetic isolated PostgreSQL databases. Final branch CI (private acceptance record): 14/14 jobs PASS. / 最终人工验收：**PASS**。计划 **38**、执行 **38**、PASS **38**、FAIL **0**、PARTIAL **0**、NOT RUN **0**。38 个逻辑场景为 35 项前置契约及同一规范样本目标在三平台各一次原生执行。该目标含两个 Go 测试函数，但每个 OS 只计一个逻辑场景；子进程辅助函数不计入。数据库测试使用隔离的合成 PostgreSQL 数据库。最终分支 CI：14/14 作业通过。

| # | Test scenario / 测试场景 | Requirement / 要求 |
|---:|---|---|
| 01 | New reservation bound in G18 OPEN transaction / 新预留同事务绑定 | P2 source, receipt, instance / 来源、receipt、实例 |
| 02 | Historical unbound reservation rejected / 历史未绑定预留拒绝 | P2 fail closed / 关闭失败 |
| 03 | Actual character owner captured / 捕获真实角色 owner | P3 authoritative CharacterID / 权威 CharacterID |
| 04 | Seal, new-source reject and accepted exact replay / 封存、新来源拒绝、旧来源重放 | P1/P6 |
| 05 | Acceptance commit while Seal and replay wait / Seal 与重放等待中的接受提交 | P1 three-party serialization / 三方序列化 |
| 06 | Two sessions merge; two characters under one account separate / 同角色多会话合并、同账号不同角色分离 | P3 |
| 07 | S→D without second Remaining deduction / S→D 不重复扣 Remaining | P4 conservation / 守恒 |
| 08 | Post-distribution refund, debt gate, repayment and rebuild / 分配后退款、债务门禁、偿债及重建 | P4 |
| 09 | Synthetic lot, projection, stale save and both audits / 合成 lot、投影、旧版本保存及双审计 | P5 |
| 10 | Audit missing projection / 审计缺失投影 | P5 |
| 11 | Audit wrong quantity / 审计错误数量 | P5 |
| 12 | Audit wrong CharacterID / 审计错误角色 | P5 |
| 13 | Audit wrong BlockInstanceID / 审计错误实例 | P5 |
| 14 | Audit stale revision / 审计过期修订 | P5 |
| 15 | Audit altered source digest / 审计篡改来源摘要 | P5 |
| 16 | Immutable UPDATE, DELETE and TRUNCATE guards / 不可变写入、删除与截断守卫 | P1/P2/P3/P5 |
| 17 | Seal 2 participants / 2 人封存 | P6 ordering, rebuild, restart / 排序、重建、重启 |
| 18 | Seal 10 participants / 10 人封存 | P6 |
| 19 | Seal 50 participants / 50 人封存 | P6 |
| 20 | Seal 100 participants / 100 人封存 | P6 |
| 21 | Seal 500 participants / 500 人封存 | P6 digest and restart / 摘要与重启 |
| 22 | Reject participant 501 / 拒绝第 501 人 | P6 transaction rollback / 事务回滚 |
| 23 | Kill acceptance before commit / 接受提交前进程终止 | Crash and independent verifier / 崩溃及独立验证 |
| 24 | Kill in-flight acceptance while Seal waits / Seal 等待期间终止接受进程 | Crash/lock boundary / 崩溃及锁边界 |
| 25 | Kill Seal before commit / Seal 提交前终止 | Crash/retry / 崩溃与重试 |
| 26 | Kill Seal after commit before ACK / Seal 提交后、ACK 前终止 | Crash/replay / 崩溃与重放 |
| 27 | Kill binding before commit / 绑定提交前终止 | Atomic creation / 原子创建 |
| 28 | Kill binding after commit before ACK / 绑定提交后、ACK 前终止 | Exact replay / 精确重放 |
| 29 | Kill synthetic distribution before commit / 合成分配提交前终止 | Recovery rollback / 恢复回滚 |
| 30 | Kill synthetic distribution after commit before ACK / 合成分配提交后、ACK 前终止 | Recovery replay / 恢复重放 |
| 31 | Kill synthetic issuance before commit / 合成发放提交前终止 | Lot/projection rollback / 来源与投影回滚 |
| 32 | Kill synthetic issuance after commit before ACK / 合成发放提交后、ACK 前终止 | Lot/projection replay / 来源与投影重放 |
| 33 | Fixed canonical golden vectors / 固定规范黄金向量 | Seal, participant/order, beneficiary, reservation, issuance, UTC, 2/10/50/100/500/501 / 封存、参与者顺序、受益人、预留、发放、UTC、人数边界 |
| 34 | Native macOS golden execution / macOS 原生黄金向量执行 | PASS, macos-latest, digest identical / 通过，摘要一致 |
| 35 | Native Linux golden execution / Linux 原生黄金向量执行 | PASS, ubuntu-latest, digest identical / 通过，摘要一致 |
| 36 | Native Windows golden execution / Windows 原生黄金向量执行 | PASS, windows-latest, digest identical / 通过，摘要一致 |
| 37 | Historical accepted Activity without beneficiary remains ineligible / 历史已接受但缺少受益人绑定的 Activity 不具结算资格 | P3 fail closed, seal rebuild / 关闭失败、封存重建 |
| 38 | Reward preserves existing G19 migration receipt and quantity / 奖励保留原 G19 migration receipt 与数量 | P5 separate provenance / 独立来源 |

The 500 participant fixture measured approximately 1.78 seconds alone and 7.76 seconds while full suites ran concurrently; its estimated 3,992 fixture/acceptance writes are test-side calls and accepted consequences, not a database I/O counter or SLA. / 500 人夹具单独运行约 1.78 秒，与全量套件并行时约 7.76 秒；估算的 3,992 次夹具／接受写入是测试侧调用和接受后果，不是数据库 I/O 计数器，也不是 SLA。
