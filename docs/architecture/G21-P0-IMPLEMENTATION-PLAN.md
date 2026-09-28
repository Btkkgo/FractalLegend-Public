# G21-P0 Implementation Plan / 实施计划

Accepted private canonical source. The six prerequisite steps below are complete and received final human acceptance PASS. This remains an implementation record for G21-P0 only; full G21 settlement requires the next human gate.

来源为已验收私有 canonical 版本。下列六项前置步骤均已完成并获得最终人工验收 PASS。本记录仅限 G21-P0；完整 G21 结算仍需下一轮人工门槛授权。

1. Inspect canonical G17–G20 schema and tests, verify clean isolated checkout, and allocate migration `0014` without editing `0001–0013`. / 检查当前 G17–G20 schema 与测试，确认独立工作树及干净基线；新增 `0014`，不修改 `0001–0013`。
2. Add G18 OPEN binding in the same reservation transaction; prove source/receipt/instance uniqueness and historical unbound fail closed. / 在 G18 OPEN 原事务追加绑定；验证来源、receipt、实例唯一性及历史未绑定关闭失败。
3. Add G20 acceptance row protocol and database Activity guard; capture actual CharacterID at acceptance; add READ COMMITTED Seal and immutable snapshot. Test acceptance/Seal/replay overlap, finalization gate, restart, 500 and 501. / 增加 G20 接受状态行协议与数据库 Activity 守卫；在接受时取得真实 CharacterID；增加 READ COMMITTED Seal 与不可变快照。测试接受／Seal／重放交错、终态门禁、重启、500 与 501。
4. Remove the zero Distributed assumption, extend existing recovery records and four-component reconstruction, add isolated TEST transfer and positive-debt read gate. Verify refunds and debt repayment with G14/G17/G18 authority. / 移除 Distributed 恒零假设，扩展现有恢复记录和四分量重建，增加隔离 TEST 转移及正债务只读门禁。用 G14/G17/G18 权威路径验证退款与偿债。
5. Add independent `MINING_REWARD` lot and projection, reuse the reviewed G19 definition, retain original G19 migration commitments, and verify stale revision rejection and corruption reporting. / 增加独立的 `MINING_REWARD` lot 与 projection，复用已审核 G19 definition，保留原 G19 migration commitments，并验证旧修订拒绝与损坏报告。
6. Fix cross-platform canonical vectors, run actual child-process kill plus independent verifier cases, run focused P0 matrix and complete normal/race regression, scan for secrets and routes, then write bilingual evidence. / 固定跨平台规范向量，运行真实子进程 kill 与独立验证进程，执行 P0 专项矩阵和完整 normal／race 回归，扫描 secret 与入口，并形成双语证据。

All P0 gates were observed: 38/38 matrix PASS, native Windows/Linux/macOS identical digest `c50197762badfb95166d1ca81ff56baa1b69d689179055dc47f76fd06d09a436`, and 14/14 branch CI jobs PASS. Cross-compilation was not used as native evidence. PR CI and merge were separately checked before this public export. / 全部 P0 门槛已取得证据：矩阵 38/38 PASS，Windows／Linux／macOS 原生摘要一致，分支 CI 14/14 PASS。未以交叉编译代替原生证据。PR CI 与合并已在公开导出前另行核对。
