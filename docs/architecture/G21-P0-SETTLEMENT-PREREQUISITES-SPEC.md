# G21-P0 Settlement Prerequisite Foundation — Specification / 结算前置契约规格

Status: **ACCEPTED — G21-P0 prerequisites only**. Final human acceptance PASS; 38/38 P0 tests and 14/14 branch CI jobs PASS, including native Windows/Linux/macOS determinism. Source: accepted private canonical revision. Scope: P1–P6 only. No full settlement, allocation, player grant, production route, FB/Contribution/Reputation change, or G22. [Accepted ADR](../decisions/ADR-G21P0-001-settlement-prerequisite-authority.md); [G21 next-gate status](G21-DESIGN-GATE-STATUS.md).

状态：**已批准——仅 G21-P0 前置契约**。最终人工验收 PASS；P0 测试 38/38 与分支 CI 14/14 PASS，含 Windows／Linux／macOS 原生确定性。来源为已验收私有 canonical 版本。范围仅限 P1–P6；不实现完整结算、分配、玩家发放、生产入口、FB／Contribution／Reputation 变更或 G22。参见已批准 ADR 与 G21 下一门槛状态。

## P1. G20 acceptance and seal / G20 接受与封存

`mining_power_acceptance_states` is G20 owned and keyed uniquely by `BlockInstanceID`. An acceptance retains G20's SERIALIZABLE validation transaction, creates its OPEN row if absent, holds `FOR SHARE`, and checks OPEN before writing its first Activity. A database BEFORE INSERT guard applies the same state check to direct application-role Activity inserts. Seal uses a READ COMMITTED transaction, takes `FOR UPDATE`, changes OPEN to SEALED, then issues a new statement to read final immutable Activity facts. A transaction that already holds the shared lock finishes before Seal's exclusive lock is granted. New first sources return `BLOCK_SEALED`; an accepted exact source replay still returns its original Activity with zero new power. The source uniqueness constraints remain the final idempotency boundary. Seal requires the bound G18 block to be FINALIZED and does not alter the G18 state machine.

`mining_power_acceptance_states` 属于 G20，以 `BlockInstanceID` 唯一索引。接受路径保留 G20 原有 SERIALIZABLE 验证事务，在必要时建立 OPEN 行，持有 `FOR SHARE`，确认 OPEN 后才写入首个 Activity。数据库 BEFORE INSERT 守卫也约束直接使用应用数据库角色的 Activity 写入。Seal 使用 READ COMMITTED 事务，取得 `FOR UPDATE`，将 OPEN 改为 SEALED，再用新语句读取最终不可变 Activity。先持共享锁的事务会在 Seal 取得排他锁前结束。新的首次来源返回 `BLOCK_SEALED`；已接受来源的精确重放仍返回原 Activity，新增矿力为零。来源唯一约束保留为最终幂等边界。Seal 仅接受对应 G18 区块已 FINALIZED 的实例，不改动 G18 状态机。

The immutable `SettlementInputSeal` contains the instance, display block, G18 source and evidence digest, G18/G20 rule versions, window, counts, total valid power, sorted CharacterID weights, sorted accepted Activity identities, beneficiary and Activity digests, schema/rule versions, UTC canonical seal time, and a digest over fixed-order JSON bytes. It is the sole future allocation input. Historical Activities without beneficiary evidence are marked `NOT_SETTLEMENT_ELIGIBLE`; no retrospective owner guess or multiplier recalculation is allowed. `RebuildMiningPowerSeal` checks the stored bytes against immutable G20 facts and beneficiary bindings in a read-only snapshot.

不可变 `SettlementInputSeal` 包含实例、展示区块、G18 来源及证据摘要、G18/G20 规则版本、时间窗、数量、有效矿力总量、按 CharacterID 排序的权重、按 Activity ID 排序的已接受身份、受益人与 Activity 摘要、schema／规则版本、UTC 规范时间及固定字段顺序 JSON 字节的摘要。未来分配只能使用该 Seal。缺少受益人证据的历史 Activity 标记为 `NOT_SETTLEMENT_ELIGIBLE`；不得事后猜测 owner，也不得重新计算旧倍率。`RebuildMiningPowerSeal` 在只读快照中以不可变 G20 事实和受益人绑定核对存储字节。

## P2. Reservation instance binding / 预留与实例绑定

Every new G18 OPEN reservation appends `mining_reservation_instance_bindings` in its original transaction. The row binds the OPEN entry ID, receipt ID, BlockInstanceID, display BlockID, rule, amount, version, time, and fixed-field evidence digest. Source, receipt, and instance are independently UNIQUE. The table has no G18 FK and is UPDATE/DELETE/TRUNCATE protected. Existing unbound reservations are never inferred from display BlockID, height, time, or command ID and fail closed for Seal.

每个新的 G18 OPEN 预留都在原事务追加 `mining_reservation_instance_bindings`。记录绑定 OPEN entry ID、receipt ID、BlockInstanceID、展示 BlockID、规则、数量、版本、时间及固定字段证据摘要。来源、receipt 与实例分别有 UNIQUE 约束。该表不建立指向 G18 的 FK，并禁止 UPDATE／DELETE／TRUNCATE。历史未绑定预留绝不从展示 BlockID、高度、时间或命令 ID 推断；Seal 对其关闭失败。

## P3. Beneficiary authority / 受益人权威

At G20 acceptance the server checks the real `characters(id,account_id)` row under `FOR SHARE` and appends a G20 owned `MiningBeneficiaryBinding` in the same transaction. The source account, PlayerID and resolved CharacterID remain distinct fields. The client cannot supply CharacterID. Multiple sessions of one CharacterID merge into one seal weight. Different CharacterIDs remain distinct even under one AccountID. Existing unbound accepted facts remain readable but cannot become eligible for settlement.

G20 接受时，服务端以 `FOR SHARE` 校验真实 `characters(id,account_id)` 行，并在同一事务追加 G20 所属的 `MiningBeneficiaryBinding`。来源 AccountID、PlayerID 与已解析 CharacterID 是不同字段。客户端不能提供 CharacterID。同一 CharacterID 的多个会话在 Seal 中合并权重；即使同属一个 AccountID，不同 CharacterID 仍分别计权。旧的未绑定已接受事实可以读取，但不得获得结算资格。

The composite character/account FK follows the existing G20 session ownership pattern and has no CASCADE action. It locks the referenced owner row when the binding is inserted; the explicit `FOR SHARE` lookup occurs after the BlockInstance acceptance lock and before the Activity transaction commits. No G20→G18 FK or G18 row lock is introduced. / 角色／账号复合 FK 沿用 G20 会话归属模式，且没有 CASCADE 动作。插入绑定时会锁住所引用的 owner 行；显式 `FOR SHARE` 查找发生在 BlockInstance 接受锁之后、Activity 事务提交之前。不新增 G20→G18 FK 或 G18 行锁。

## P4. Distributed and recovery / 已分配量与恢复

The authoritative invariant is `C = Reserved + Distributed + Remaining − RecoveryDebt`, with nonnegative components and `RecoveryDebt ≤ Reserved + Distributed`. A TEST-only primitive moves `R` from Reserved to Distributed; it does not subtract Remaining again. It appends a dedicated immutable source row and a recovery journal entry in the same transaction. G14 refunds still consume available Remaining first, then create RecoveryDebt; new lawful emission repays debt before exposing Remaining. A read-only future gate denies new settlement while debt is positive. Historical completed replay remains a separate zero-delta path. The recovery reconciler now reconstructs all four components and validates source/revision continuity.

权威守恒式为 `C = Reserved + Distributed + Remaining − RecoveryDebt`；各分量非负，且 `RecoveryDebt ≤ Reserved + Distributed`。仅 TEST 的原语把 R 从 Reserved 转至 Distributed，不会再次减少 Remaining，并在同一事务追加独立不可变来源与恢复日志。G14 退款仍先消耗可用 Remaining，不足部分形成 RecoveryDebt；新的合法 capacity 先偿债，再形成 Remaining。未来新结算在债务为正时由只读门禁暂停；历史已完成结果的零增量重放是独立路径。恢复审计现重建四个分量并校验来源与修订连续性。

The synthetic transfer takes the emission pool lock first, then the finalized G18 block and reservation locks, matching G18's pool-before-block order. It does not expose a G20 or HTTP call path. / 合成转移先锁排放池，再锁已完成的 G18 区块与预留，顺序与 G18 的“池先于区块”一致；不暴露 G20 或 HTTP 调用路径。

## P5. Mining reward provenance / 采矿奖励来源

`MINING_REWARD` has its own immutable issuance lot and inventory projection tables; it never reuses a G19 migration receipt or migration asset. The synthetic adapter requires a sealed, eligible instance and an actual beneficiary binding, and requires exactly one reviewed G19 alias definition in the fixture. Ambiguous or absent definitions fail closed. A fixed-order digest covers the lot source fields. Lot, projection, inventory insertion and character revision increment commit atomically. The existing optimistic revision check rejects a stale aggregate save. G19's asset guard recognizes either an original migration asset or a separate reward projection, while unmarked canonical ore still fails. Read-only reconciliation reports missing or mismatched source/projection/inventory, owner, instance, quantity, digest, or revision; it never repairs data.

`MINING_REWARD` 使用独立不可变的 issuance lot 与库存投影表，绝不复用 G19 migration receipt 或 migration asset。合成适配器要求实例已封存且可用、有真实受益人绑定，并要求测试夹具中恰有一个已审核的 G19 alias definition；定义缺失或有歧义时关闭失败。固定字段顺序摘要覆盖 lot 来源字段。Lot、projection、库存插入与角色修订号递增原子提交。现有乐观修订校验会拒绝旧版本 aggregate 保存。G19 资产守卫承认原 migration asset 或独立 reward projection；未标记的规范矿石仍被拒绝。只读核对报告来源、投影、库存、owner、实例、数量、摘要或修订错误，不自动修复。

## P6. Participant boundary and operational limits / 人数边界与运行限制

Seal supports 2/10/50/100/500 distinct CharacterIDs in TEST. Participant 501 aborts its transaction and leaves the gate OPEN. At 500, tests cover ordering, beneficiary digest, persistence, read-only rebuild, and reopening a separate store. Timing and approximate write counts are recorded without a production SLA. The new Seal, distributed transfer, and issuance writer have no HTTP or production service route; each writer also checks the isolated test database name. No allocation formula or real player Ore is implemented.

TEST Seal 覆盖 2／10／50／100／500 个不同 CharacterID。第 501 人使事务回滚，门禁保持 OPEN。500 人场景覆盖排序、受益人摘要、持久化、只读重建及重新打开独立 Store。记录耗时与估算写入次数，不设生产 SLA。新增 Seal、Distributed 转移与 issuance 写入没有 HTTP 或生产 Service 入口；各写入器还检查隔离测试库名称。不实现分配公式或真实玩家发矿。
