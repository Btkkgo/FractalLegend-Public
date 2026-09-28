# G21-P0 Development Log / 开发日志

## English — Primary

G21-P0 began from the accepted G20 foundation and kept production settlement closed. Migration `0014` adds immutable reservation-instance and CharacterID beneficiary bindings, a G20 OPEN/SEALED acceptance boundary, and an immutable input Seal. PostgreSQL row locks establish the acceptance/Seal ordering across concurrent processes and restarts; historical records without authoritative bindings remain ineligible.

The existing G17/G18 recovery authority now accounts for nonzero Distributed while preserving `C = Reserved + Distributed + Remaining − RecoveryDebt`. A TEST-only transfer moves Reserved to Distributed without deducting Remaining twice. A separate `MINING_REWARD` lot and inventory projection reuse the reviewed G19 ore definition while keeping G19 migration provenance unchanged. No production route, real player ore, allocation or claim was added.

Development tests exercised missing tables and methods first, then verified the new schema and writers. A PostgreSQL trigger was corrected to read immutable raw JSON before generated columns are available. Existing regression fixtures were adapted for the additive migration without changing migrations `0001–0013`. Crash/restart, concurrent acceptance/Seal/replay, recovery, provenance corruption and the 500/501 participant boundary passed.

Final human acceptance covers **38/38** G21-P0 scenarios. Windows, Linux and macOS native evidence had the same canonical digest `c50197762badfb95166d1ca81ff56baa1b69d689179055dc47f76fd06d09a436`; the private PR's required CI passed **14/14** jobs. This mirror records its own independent [verification](../public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md). Full G21 reward settlement remains for the next human gate.

## 中文 — 完整审核版

G21-P0 从已验收的 G20 基础开始，生产结算保持关闭。迁移 `0014` 增加不可变的预留实例与 CharacterID 受益人绑定、G20 OPEN／SEALED 接受边界和不可变输入 Seal。PostgreSQL 行锁规定并发进程及重启后的接受／Seal 顺序；缺少权威绑定的历史记录仍不具备结算资格。

既有 G17／G18 恢复权威现支持非零 Distributed，同时保持 `C = Reserved + Distributed + Remaining − RecoveryDebt`。仅限 TEST 的转移把 Reserved 移到 Distributed，不会再次扣减 Remaining。独立的 `MINING_REWARD` lot 与库存投影复用已审核的 G19 矿石定义，但不复用 G19 迁移来源。没有增加生产入口、真实玩家矿石、奖励分配或领取。

开发测试先验证缺表与缺方法的失败，再验证新增 schema 与写入器。PostgreSQL trigger 曾在 generated column 可用前读取它，现改为读取不可变原始 JSON。既有回归夹具适配增量迁移，`0001–0013` 未修改。真实进程崩溃／重启、并发接受／Seal／重放、恢复、来源损坏及 500／501 人边界均通过。

最终人工验收覆盖 **38/38** 项 G21-P0 场景。Windows、Linux、macOS 原生证据具有相同规范摘要 `c50197762badfb95166d1ca81ff56baa1b69d689179055dc47f76fd06d09a436`；私有 PR 的必需 CI **14/14** 项通过。此公开镜像另行记录[独立验证](../public-sync/G21-P0-PUBLIC-SYNC-CANDIDATE.md)。完整 G21 奖励结算仍待下一轮人工门槛。
