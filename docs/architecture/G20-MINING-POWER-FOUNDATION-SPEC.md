# G20 — Accepted Mining Power Specification

**G20 Mining Power Foundation: final human acceptance PASS; private canonical closure complete.**

The accepted chain is **G17 Global Mining Pool → G18 Mining Block + Recovery → G18.1 Stable Block Identity → G20 Server-authoritative Mining Power**. G17 capacity/reservation is not player ore. G19 identity migration preserves existing quantities without issuance.

G20 records immutable ValidatedMiningActivity as authority and derives Participant aggregates. Stable BlockInstanceID spans sessions/events/history. G18 uses a read-only ordinary SELECT without upstream row locks or cross-domain foreign keys, saving an immutable validation snapshot. PostgreSQL SERIALIZABLE commits activity and aggregate atomically, with global SourceEvent uniqueness and at most five attempts including the first. Synthetic Tool/Map inputs are server-resolved, final client power is rejected, integer scale is 1,000,000 and one final floor division determines power. Canonical replay timestamps use `t.UTC().Round(0)`; strict receipt equality is retained. Reconciliation is read-only and never repairs unknown history.

G20 has **90/90 scenarios PASS**. Private PR and post-merge CI each have **10/10 jobs PASS**, full Normal and Race each **882/882 PASS**, zero failures/skips/data race. Native Windows/Linux/macOS T50 has 24 identical logical results with digest `f5f2e54d342d02544c54565d488aa1e096fd9208c4ea4d23b291fa2f5ceb3d60`. The public subset is tested independently; its results appear in the public export manifest rather than borrowing private counts.

This is a TEST foundation, not a public game release. Production Service remains fail-closed. **NOT IMPLEMENTED:** reward distribution, block reward settlement, ore claim, mining pool deduction, production reward/duration/tool/map rules and G21. G20 mutates no G18, FB, Contribution, Player Ore or Mining Pool state. Social material remains an unpublished draft.

**G20 算力基础：最终人工验收 PASS，私有 canonical 收尾完成。**

已验收链条为 **G17 全服矿池 → G18 挖矿区块与恢复 → G18.1 稳定区块身份 → G20 服务器权威算力**。G17 容量／预留不等于玩家矿石；G19 身份迁移保持既有数量，不发行矿石。

G20 以不可变 ValidatedMiningActivity 为权威、Participant 为派生汇总。稳定 BlockInstanceID 贯穿会话／事件／历史。G18 普通只读 SELECT 无上游行锁或跨域外键，保存不可变验证快照。PostgreSQL SERIALIZABLE 原子提交事实与汇总；SourceEvent 全局唯一，含首次最多五次。工具／地图输入由服务器解析合成数据，拒绝客户端最终算力；整数定点比例 1,000,000，只作一次最终除法向下截断。重放时间采用 `t.UTC().Round(0)`，保留回执严格相等。对账只读，不修复未知历史。

G20 **90/90 场景通过**；私有 PR 及合并后 CI 各 **10/10 作业通过**，完整 Normal／Race 各 **882/882**，零失败、跳过、数据竞争。Windows／Linux／macOS 原生 T50 的 24 个逻辑结果完全一致，摘要如上。公开子集独立运行验证，结果在公开导出清单中记录，不借用私有数量。

这是 TEST 基础，不是公开游戏发布；生产 Service 失败关闭。**未实现：**奖励分发、区块奖励结算、矿石申领、矿池扣减、正式奖励／时长／工具／地图规则及 G21。G20 不修改 G18、FB、Contribution、玩家矿石或 Mining Pool。社交素材保持未发布草稿。

## English primary

### Scope and authoritative inputs

G20 records server-validated mining activity and a derived participant power aggregate. Mining Power is not FB, Contribution, Reputation, Player Ore, emission capacity, a reservation or a reward. G20 has no economy write or reward-settlement interface. Production reward, duration, equipment, map and durability rules remain NOT FINALIZED. No client route or production producer is wired.

Only `DEV_G20_MINING_POWER_V1`, `INTEGER_PRODUCT_FLOOR_ONCE_V1`, `TEST_NON_PRODUCTION` profiles and `SYNTHETIC_TEST` evidence are accepted. Synthetic catalogs, sessions and events are seeded exclusively by tests. The compiled manifest must match exactly; unknown or altered rules fail closed. A configured `Service` in production rejects before invoking its repository, including a repository containing valid TEST profiles.

The server supplies an authenticated Principal `{AccountID, PlayerID}` and verifies character ownership. Client ActionIntent contains exactly five case-sensitive strings: `activityId`, `sourceEventId`, `activitySessionId`, `blockId`, `blockInstanceId`. It supplies no power, coefficient, profile, rule, player or timestamp. Decoder rejects duplicate/unknown keys, nested/non-string values, invalid UTF-8, trailing JSON, malformed or oversized (>4096 bytes) payloads and all documented final-power aliases, including case/underscore variants and NaN/Infinity forms. IDs use bounded ASCII letters, digits, underscore, dot, colon and hyphen; source ≤96 bytes, other references ≤128 bytes. Canonical ActivityID is `mpa:` plus SourceEventID.

### Stable identity and historical authority

`BlockInstanceID` is the concrete G18.1 identity, shaped `mining-block-instance-` plus 32 nonzero hexadecimal digits in persistence. `BlockID` is display/business metadata. Session, SourceEvent, Activity, Participant, totals, rebuild and reconciliation all carry and scope by BlockInstanceID. Participant key is `(PlayerID, BlockInstanceID, ActivitySessionID, RuleVersion)`; Tool/Map are immutable session bindings. Multiple sessions cannot silently combine profile identities.

There is **no G20→G18 FK**, including to BlockInstanceID. `ReadBlock` performs one ordinary SELECT of the exact instance before the G20 transaction. It reads instance/display, height, CreateCommandID, state, G18 rule and window plus database server validation time. It does not lock or mutate G18 rows or invoke lifecycle/reservation/pool/debt methods. A new event requires the matching instance/display and OPEN state. Missing, FINALIZED or CANCELLED instances reject with zero credit.

Each accepted Activity owns an immutable `MiningBlockValidationSnapshot`: BlockInstanceID, BlockID, BlockHeight, CreateCommandID, Status, StartedAt, ScheduledEndAt, G18RuleVersion, G20RuleVersion, ValidatedAt and SourceEvidenceVersion. `G18_SCHEMA_0012` names the adapter's schema/evidence contract; it is not an invented upstream revision number. History never refreshes this snapshot from a live G18 row. Privileged test recreation of identical BlockID/height/command/window with a fresh instance must leave old replay/rebuild/reconciliation intact and create separate new-instance power rows.

Eligibility is a **pre-transaction snapshot policy**, not commit-time G18 fencing. A G18 close after the adapter read does not retroactively invalidate that observed OPEN snapshot. AcceptedAt must still satisfy the recorded half-open window. G20 holds FOR SHARE only on its own session to order ACTIVE→CLOSED against acceptance. No G18 row lock or mutation is allowed.

### Deterministic calculation

Scale S=1,000,000. B=0..MaxInt64; E, M=1..1,000,000,000; valid A=1..1,000,000,000. Compute the entire bounded arbitrary-precision integer product B×E×A×M and divide once by S³=10¹⁸, flooring once. Output, participant sum, count and block total must fit signed int64; overflow returns an explicit error, never wraps, saturates or leaves partial writes. No floats, intermediate division, minimum-one or fractional carry.

| Vector | B | E | A | M | Exact P |
|---|---:|---:|---:|---:|---:|
| Basic |100|1000000|1000000|1000000|100|
| Advanced |250|1000000|1000000|1000000|250|
| Efficiency |100|1250000|1000000|1000000|125|
| Activity |100|1000000|500000|1000000|50|
| Map |100|1000000|1000000|1500000|150|
| Combined |100|1250000|500000|1500000|93|
| No intermediate floor |3|1500000|1500000|1000000|6|
| Fractional positive |1|500000|500000|1000000|0|
| Max fitting output |9223372036854775807|1000000|1000000|1000000|9223372036854775807|

Zero B or a positive product below one is a valid audited Activity, counted exactly once with P=0. Zero A is invalid evidence. Synthetic Advanced power may exceed Basic power without changing capacity, reserved/distributed reward, emission or debt.

### Additive persistence contract

Migration `0013_mining_power_foundation.sql` follows accepted G18.1 migration0012. Prior migration bytes and G18 schema/guards are unchanged. It creates seven G20 tables and zero seeds. A character `(id, account_id)` unique index supports explicit ownership FK; it changes no economy row.

| Table | Required contract |
|---|---|
| mining_power_rules | Exact compiled TEST manifest; immutable |
| mining_power_tool_profiles | TEST kind, local rule FK, B/E bounds, immutable |
| mining_power_map_profiles | TEST kind, local rule FK, M bounds, immutable |
| mining_power_sessions | Stable instance, character/account ownership, local rule/tool/map FKs, fixed bindings and ordered nonnull timestamps; ACTIVE→CLOSED only |
| mining_power_source_events | Global source PK and canonical ActivityID UNIQUE, complete session/instance/player/profile/rule binding, synthetic evidence, bounded weight, ordered nonnull times; immutable |
| mining_power_activities | Global Activity PK and SourceEvent UNIQUE independent of player/block/rule; local composite Source FK; complete nonnull validation snapshot; integer input/result/window checks; immutable |
| mining_power_participants | Composite participant PK and complete local session FK; nonnegative sum, positive count, min/max AcceptedAt |

Immutable catalog/session/source/activity domain records use JSONB plus generated relational columns, so the identity and bounds constrained by SQL are the same values decoded by Go. Participant uses normal scalar columns for atomic increments. Closed recursive case-sensitive JSON field contracts reject extra aliases, so case-insensitive Go struct decoding cannot override exact SQL-generated fields. Missing and JSON-null timestamps/context are rejected explicitly, because SQL CHECK alone permits unknown. Activity's SQL NUMERIC formula checks exact floor-once output. UPDATE, DELETE and TRUNCATE guards protect Activities and immutable catalogs/sources; sessions prohibit binding change, reopening, delete and truncate. Derived participant corruption is reported, never repaired by reconciliation. Exact comparisons also audit display BlockID. Ambiguous historical identities poison every affected group independent of iteration order; conflicting actual aggregate duplicates expose null actual/delta.

### Acceptance, replay and transaction policy

1. Validate intent/principal and live character ownership, then look up committed Activity by global source/activity identity. Other-player replay returns no original receipt. Altered instance/display/session/version/activity binding returns REPLAYED or invalid canonical identity, with zero applied power.
2. Exact owner replay returns the original immutable fact with DUPLICATE and AppliedPower=0 after fail-closed historical reconciliation. It does not demand current session/block eligibility and remains valid after expiry, restart, close or business-ID recreation. Corrupt history does not permit fresh credit.
3. For a new event, read the ordinary G18 snapshot before opening any G20 transaction. Acquire one **G20-only admission advisory lock** on a dedicated connection before SERIALIZABLE Begin. A prior participant-scoped gate still exhausted real SSI retries across different players because small tables can receive relation-level predicate locks. The one-key nonproduction foundation serializes acceptance writes; it trades throughput for bounded correctness, with no G18 lock. Release only after commit/rollback; failed unlock closes the connection. Validation has a 30-second internal context cap and honors shorter caller cancellation/deadlines.
4. Each fresh SERIALIZABLE transaction rechecks ownership and global history, loads the immutable SourceEvent, takes G20 session FOR SHARE, validates exact bindings/ACTIVE status and loads persisted synthetic profiles/manifest. Server AcceptedAt is one database clock_timestamp normalized to UTC microseconds. Require OpenedAt≤ObservedAt; StartedAt≤ObservedAt≤ValidatedAt≤AcceptedAt; ObservedAt<EndAt; AcceptedAt<session/source/block expiry; source expiry>ObservedAt; session expiry≤block end. Future evidence at the adapter snapshot cannot become valid by waiting. Client time has no authority.
5. Insert Activity and increment Participant atomically in the same transaction. Sum/count bounds fail before a partial aggregate can commit; CreatedAt=min AcceptedAt, UpdatedAt=max AcceptedAt. Commit once. Lost response after commit replays the original and adds zero.
6. Retry only PostgreSQL 40001 or 40P01, at most **five attempts including the first**, each with a new transaction/history lookup and bounded 1/2/4/8ms backoff. Context cancellation, other SQL errors, overflow or invariant errors are not retried. Exhaustion returns ErrRetryExhausted explicitly. No higher cap, weakened assertion or skip is permitted.

### Read-only reporting and recovery

Pure `RebuildParticipants` derives expected rows entirely in memory from authoritative historical Activity inputs, catalogs/source/session and each saved validation snapshot. No mutable G18 lookup determines old identity or eligibility. Historical session close does not revoke a committed fact.

`SnapshotMiningPower` uses PostgreSQL REPEATABLE READ READ ONLY for all G20 reads. It reads deterministic ordered catalogs/session/source/activity/participant rows and reconstructs historical BlockContexts solely from saved snapshots. `ReconcileMiningPower` reports per-identity/field expected, actual, arbitrary-precision decimal delta and status. Matching comparisons include delta=0; discrepancies appear in Findings. Unknown, corrupt or overflowing historical groups are UNVERIFIABLE with expected/delta=null, never zero/current-rule fallback or a partial sum. Duplicate history, missing/extra aggregates, wrong count/power/time/profile and large deltas are visible. Repeated reporting cannot write, repair, distribute or settle rewards.

### Verification and evidence gates

The approved 90 scenarios below are unchanged in scope. Scenario count differs from Go test/subtest/parent event count. Test fixtures use fresh isolated PostgreSQL databases with fractal_g9_test names; G9/G11 restart databases are independently fresh per full stage. Full normal/race assertions and package timeouts stay15m/20m. Historical failures remain failures in the report; focused reruns do not waive a failed full gate.

DB tests cover 100 concurrent same-source requests, distinct same-player and different-player requests, five-attempt retry policy, atomic rollback, OS-killed independent workers and separate verification processes at five crash points, session-close ordering, G18 snapshot overlap/no row lock, immutable guards, migration rollback/idempotence, same BlockID/fresh-instance isolation, read-only corruption reporting and 2/10/50/100/500 miners. Nonempty upstream snapshots include FB/Contribution/Reputation/Ore, inventory/G19 journals, spend/refund/history, all pool/debt and reservation rows; G20 must leave them byte-equivalent by sorted row JSON.

Native Windows/Linux/macOS T50 and all private platform builds passed. Public builds and database suites are verified independently in the export manifest. Full Windows/Linux native DB suites are not claimed.

## Approved shared 90-scenario matrix / 已批准共用90场景矩阵

| ID | Scenario / 场景 | Required behavior / 必须行为 |
|---|---|---|
| T01 | Valid event / 合法活动 | Accepted; Basic vector=100; one fact/participant / 接受 Basic=100，恰好一条事实和汇总 |
| T02 | No tool / 无工具 | NOT_ELIGIBLE, zero, no rows / 无资格，新增零且不写行 |
| T03 | Invalid tool / 无效工具 | Reject unknown/non-TEST tool, zero / 未知或非 TEST 工具拒绝并得零 |
| T04 | No map / 无地图 | Reject, zero / 拒绝并得零 |
| T05 | Invalid map / 无效地图 | Reject wrong/non-TEST map, zero / 错误或非 TEST 地图拒绝并得零 |
| T06 | No activity / 无活动 | Missing source rejects, zero / 缺失来源拒绝并得零 |
| T07 | Invalid evidence / 无效活动 | INVALID source, zero / 无效来源拒绝并得零 |
| T08 | Expired event / 活动过期 | Equal/after expiry reject; client cannot backdate / 恰好到期和到期后拒绝，客户端不能回填时间 |
| T09 | Wrong block / 错误区块 | Session/source/intent mismatch rejects / 会话、来源、意图区块不一致拒绝 |
| T10 | Inactive block / 非活动区块 | FINALIZED and CANCELLED reject new event / 已完成和已取消区块拒绝新事件 |
| T11 | Invalid player / 无效玩家 | Missing character or ownership rejects / 角色不存在或归属错误拒绝 |
| T12 | Invalid session / 无效会话 | Unknown/CLOSED/wrong-player session rejects / 未知、已关闭或他人会话拒绝 |
| T13 | Expired session / 会话过期 | Half-open window boundary rejects / 半开窗口到期边界拒绝 |
| T14 | Invalid identity / 无效身份 | Empty, overlength, non-ASCII/delimiter abuse reject / 空值、超长、非 ASCII 或分隔符滥用拒绝 |
| T15 | Fake client power / 虚假客户端算力 | miningPower=999999999 explicitly rejects / 明确拒绝 miningPower=999999999 |
| T16 | Extreme client power / 极端客户端算力 | Huge decimal final power rejects, no decode overflow credit / 超大最终算力拒绝，解码溢出不能计入 |
| T17 | Negative client power / 负客户端输入 | Explicit reject, no acceptance / 明确拒绝且不接受活动 |
| T18 | NaN/Infinity / 非有限客户端输入 | Strings/raw invalid JSON both reject, no float path / 非有限字符串或无效 JSON 均拒绝，无浮点路径 |
| T19 | Final power aliases / 最终算力字段变体 | All named fields and case/underscore variants reject / 所有列明字段及大小写、下划线变体拒绝 |
| T20 | Forged coefficients / 伪造权威输入 | B/E/A/M/rule/player/time fields reject / 客户端基础值、倍率、规则、身份和时间字段拒绝 |
| T21 | Strict decoder / 严格解码 | Duplicate/unknown/nested/case/trailing/oversize/malformed reject / 重复、未知、嵌套、大小写、尾随、超大和非法格式拒绝 |
| T22 | Forged principal / 伪造归属 | Server ownership verification fails / 服务器归属核验失败 |
| T23 | Another player's event / 他人活动 | Zero and no other-player receipt disclosure / 新增零且不泄露他人回执 |
| T24 | Noncanonical activity key / 非规范活动身份 | ActivityID differs from mpa:source rejects / 活动身份不等于 mpa:来源时拒绝 |
| T25 | Source reuse / 换身份重用来源 | Different activity ID cannot re-credit event / 换活动 ID 不能重用来源计入 |
| T26 | Altered replay binding / 篡改重放绑定 | Changed player/block/session/version rejects without credit / 改玩家、区块、会话或版本拒绝且不增加 |
| T27 | Network/server retries / 网络和服务重试 | Original fact replay; no added count or power / 重放原事实，不增加数量或算力 |
| T28 | Restart/version replay / 重启或换版本重放 | Global identity independent of rule; no re-credit / 身份去重与规则无关，重启不再计入 |
| T29 | Exact vectors / 精确向量 | §8 exact B/E/A/M outputs / 第 8 节整数向量结果精确相等 |
| T30 | Tool efficiency / 工具效率 | E=1,250,000 at B=100,A=M=S →125 / 其余单位倍率时效率 1.25 得 125 |
| T31 | Activity modifier / 活动权重 | A=500,000 at B=100,E=M=S →50 / 其余单位倍率时活动权重 0.5 得 50 |
| T32 | Map modifier / 地图倍率 | M=1,500,000 at B=100,E=A=S →150 / 其余单位倍率时地图 1.5 得 150 |
| T33 | Advanced versus Basic / 高级和基础工具 | 250>100; server-derived only / 高级工具 250 大于基础工具 100，结果只来自服务器 |
| T34 | Single floor / 单次截断 | 93.75→93 and 6.75→6; no intermediate division / 93.75 得 93、6.75 得 6，不做中间除法 |
| T35 | Zero activity weight / 零活动权重 | Reject no valid activity, zero / 无有效活动拒绝，新增零 |
| T36 | Zero synthetic base / 零合成基础值 | Valid audited zero, counted once / 合法零结果保留审计且只计数一次 |
| T37 | Fractional under-one / 小于一的正数 | Floor to zero, no minimum-one or carry / 向零截断，无最少一或小数结转 |
| T38 | Negative trusted inputs / 负权威输入 | B/E/A/M rejects before arithmetic / 负基础值或倍率在运算前拒绝 |
| T39 | Invalid multipliers / 无效倍率 | E/M zero or >max; A invalid reject / 效率、地图零值或超限及非法权重拒绝 |
| T40 | Unknown rule / 未知规则 | New event fails closed / 新活动遇未知规则失败关闭 |
| T41 | Input max boundaries / 输入上限 | Values at max handled; max+1 rejected / 上限合法处理，上限加一拒绝 |
| T42 | Output overflow / 输出溢出 | MaxInt64×1.000001 rejects / MaxInt64 乘 1.000001 拒绝 |
| T43 | Wide intermediate / 宽整数中间值 | >64-bit numerator and fitting output succeeds / 分子超过 64 位但结果可容纳时成功 |
| T44 | MaxInt64 output / 最大结果 | Exact MaxInt64 with unity multipliers / 单位倍率下精确得到 MaxInt64 |
| T45 | Participant/count overflow / 汇总溢出 | No activity leak or partial row update / 无事实泄漏或部分汇总更新 |
| T46 | Block-total overflow / 区块总和溢出 | Explicit error; no wrap or writes / 明确报错，无回绕或写入 |
| T47 | Deterministic inputs / 确定性输入 | Same snapshot inputs produce identical P / 相同权威快照输入产生相同算力 |
| T48 | Shuffled acceptance / 打乱提交顺序 | Fixed accepted set sums/counts invariant / 固定已接受集合的总和和数量不受顺序影响 |
| T49 | Timezone independence / 时区无关 | UTC server windows; varied local TZ cannot change P / 服务器 UTC 窗口，本地时区不改变算力 |
| T50 | Native cross-platform vectors / 原生跨平台向量 | Windows/Linux/macOS vector tests execute and agree / 三个原生平台执行相同向量且结果一致 |
| T51 | Serial duplicate / 顺序重复 | ACCEPTED once, then DUPLICATE and AppliedPower=0 / 首次接受，之后重复且 AppliedPower 为零 |
| T52 | Concurrent duplicate / 并发重复 | 100 simultaneous same-source submissions yield one credit / 一百个同来源并发请求只有一次计入 |
| T53 | Same-player distinct events / 同玩家不同活动 | Atomic exact sum/count; no lost update / 精确原子总和和数量，无丢更新 |
| T54 | Parallel different players / 多玩家并发 | Independent exact per-player rows/totals / 各玩家独立精确行和总和 |
| T55 | Out-of-order evidence / 乱序活动 | Eligible observations accumulate with explicit min/max times / 合法乱序观察累加，时间最小最大规则明确 |
| T56 | Transaction retry / 事务重试 | 40001/40P01 bounded; fresh lookup; no double count / 重试有界、重新查历史、不重复计入 |
| T57 | Rollback matrix / 回滚矩阵 | Fail after activity/aggregate/before commit leaves neither partial write / 事实后、汇总后、提交前失败均无部分写 |
| T58 | Kill before writes / 写入前崩溃 | New process sees no partial fact/aggregate / 新进程看到无部分事实或汇总 |
| T59 | Kill after fact / 事实写入后崩溃 | Uncommitted fact rolled back / 未提交事实随事务回滚 |
| T60 | Kill after aggregate / 汇总后崩溃 | Both uncommitted writes rolled back / 两个未提交写入一起回滚 |
| T61 | Kill before commit / 提交前崩溃 | Restart revalidates; at most one acceptance / 重启重新验证，最多接受一次 |
| T62 | Kill after commit/lost ACK / 提交后丢响应 | New process sees one fact, replay adds zero / 新进程看到一条事实，重放新增零 |
| T63 | Reopen/replay after expiry / 重开过期重放 | Existing authorized fact duplicates without new eligibility credit / 过期后已授权历史只重复返回，不再获新资格 |
| T64 | Session close race / 会话关闭竞争 | G20 share/update lock ordering; no stale new acceptance / G20 共享与更新锁排序，不能用过时状态接受新事件 |
| T65 | G18 close overlap / G18 关闭重叠 | Snapshot eligibility policy explicit; no G18 lock/write / 验证快照策略明确，不锁或写 G18 |
| T66 | Cross-block isolation / 跨区块隔离 | Source cannot move blocks; totals scoped correctly / 来源不能换区块，总和正确隔离 |
| T67 | Audit record / 审计记录 | All bindings/version/input/result/server times; no sensitive fields / 身份、版本、输入、结果和服务器时间齐全，无敏感字段 |
| T68 | Immutable guards / 不可变保护 | UPDATE/DELETE/TRUNCATE rejected; session binding/profile immutable / 更新、删除、清空禁止，会话绑定和配置不可变 |
| T69 | Exact reconciliation / 精确对账 | PASS with exact expected/actual/delta=0 / 期望等于实际且差额为零时通过 |
| T70 | Missing activity/source / 缺失活动或来源 | Reports observable gap; unconsumed source not falsely missing / 报告可观察缺失，未消费来源不误报 |
| T71 | Duplicate corrupt snapshot / 重复损坏快照 | Duplicate activity/source reported, not deduplicated silently / 重复活动或来源报告，不静默去重 |
| T72 | Unknown historical rule / 未知历史规则 | UNVERIFIABLE; expected=null, no current-rule fallback / 报告无法核验、期望为 null，不用当前规则回退 |
| T73 | Invalid reference / 无效引用 | Catalog/source/session mismatch reported / 配置、来源、会话不匹配报告 |
| T74 | Per-activity power mismatch / 活动算力不一致 | Historical recomputation differs; delta reported / 历史重算不同则报告差额 |
| T75 | Missing participant / 缺失汇总 | Expected fact sum versus missing actual reported / 事实期望与缺失实际汇总差异报告 |
| T76 | Extra participant / 多余汇总 | Actual row with no facts reported / 无事实支持的实际汇总报告 |
| T77 | Wrong aggregate count/power / 错误汇总 | Exact expected/actual/delta; no repair / 精确期望、实际、差额，不修复 |
| T78 | Wrong aggregate times / 错误汇总时间 | Min/max AcceptedAt differences reported / 最小最大接受时间差异报告 |
| T79 | Large delta / 大差额 | Arbitrary-precision delta cannot overflow / 任意精度差额不溢出 |
| T80 | No reconciliation repair / 对账不修复 | Repeatable read-only report; before/after state identical / 只读重复对账，前后状态完全相等 |
| T81 | Pure rebuild / 纯内存重建 | Rebuild equals expected participants; no DB mutation / 内存重建等于期望汇总，不改数据库 |
| T82 | Zero total / 零总算力 | Empty and accepted-zero cases return exactly zero / 空集合和合法零活动总值均为零 |
| T83 | Multiple miners / 多矿工 | Two distinct miners; exact totals/no duplicate participants / 两名矿工，总值正确、无重复汇总 |
| T84 | 10 miners / 十矿工 | N rows, correct sum, all P≥0, no assets/pool changes / 十行、总和正确、非负且不改资产或矿池 |
| T85 | 50 miners / 五十矿工 | Same invariants and deterministic repeated fixture / 相同不变量，重复夹具计算确定 |
| T86 | 100 miners / 百矿工 | Same invariants, concurrent and replayed attempts / 相同不变量，并发和重放仍正确 |
| T87 | 500 miners / 五百矿工 | Same invariants, exact checked total / 相同不变量，五百矿工总和精确受检 |
| T88 | Advanced economic isolation / 高级工具经济隔离 | Personal P rises; capacity/reward/distributed/emission/debt unchanged / 个人算力增加，容量、奖励、分发、发行和债务不变 |
| T89 | Nonempty economy/G19 snapshot / 非空经济隔离 | FB/Contribution/Reputation/Ore/G19 and all G17/G18 rows unchanged / FB、贡献、声望、矿石、G19 及全部 G17/G18 状态不变 |
| T90 | Production fail-closed / 生产默认拒绝 | TEST profiles cannot enable production; no runtime route or reward method / 测试配置不能启用生产，无运行路由或奖励方法 |

## 完整简体中文审阅版

### 范围、权威输入与身份

人工已批准在 G18.1 CLOSED PASS 后恢复 G20 实现和本地测试，基线为 edcb94b86811851cf37a9029f615c6e4108f7c92，官方分支 feature/g20-mining-power-foundation，Issue#31。本材料是未提交的技术验收候选，不代表人工验收、远程CI或阶段关闭。禁止提交、推送、PR、合并、发布或进入G21；当前实测状态见测试报告。

G20只保存服务器验证活动及派生玩家算力汇总。算力不是FB、贡献、声望、玩家矿石、发行容量、预留或奖励；没有经济写入或奖励结算接口。生产奖励、时长、真实装备、地图和耐久规则仍未定稿，也未连接客户端路由或生产活动生产者。仅允许开发规则DEV_G20_MINING_POWER_V1、整数单次截断算法INTEGER_PRODUCT_FLOOR_ONCE_V1、TEST_NON_PRODUCTION配置和SYNTHETIC_TEST证据；测试夹具独立注册配置、会话和来源，Migration不播种。规则必须与编译清单完全相同，未知或修改清单失败关闭。即使仓库已配置且包含TEST资料，生产Service也在调用仓库前拒绝。

服务器提供并核验AccountID／PlayerID归属。客户端意图仅有activityId、sourceEventId、activitySessionId、blockId、blockInstanceId五个大小写敏感字符串，不能提交算力、系数、配置、规则、玩家或时间。解码拒绝重复／未知字段、嵌套／非字符串、非法UTF-8、尾随JSON、格式错误或超过4096字节的载荷，以及最终算力字段变体、大小写／下划线变体和NaN／Infinity形式。身份只允许ASCII字母数字及下划线、点、冒号、连字符；来源最长96字节，其他引用128字节。ActivityID固定为mpa:加SourceEventID。

BlockInstanceID才是G18.1具体实例身份；持久层要求mining-block-instance-加32位非零十六进制，BlockID仅显示／业务元数据。会话、来源、事实、汇总、总值、重建和对账均保存并按实例隔离。Participant键为玩家＋实例＋会话＋规则；Tool／Map固定绑定会话，多会话不会隐式合并配置。

### G18只读边界与历史快照

禁止任何G20→G18外键，包括BlockInstanceID外键。ReadBlock在G20事务开始前按实例普通SELECT，读取实例、业务ID、高度、创建命令、状态、G18规则、窗口和数据库服务器验证时间，不锁G18行，不写入或调用生命周期、预留、矿池、债务方法。新事件必须实例／显示ID匹配且OPEN；不存在、FINALIZED或CANCELLED都零新增拒绝。

每条接受事实不可变地保存MiningBlockValidationSnapshot：实例、显示ID、高度、创建命令、状态、开始／结束、G18／G20规则、ValidatedAt、SourceEvidenceVersion。G18_SCHEMA_0012标记适配器schema／证据契约，不伪称上游修订号。历史不得从当前G18行刷新快照。测试以特权夹具重建同BlockID／高度／命令／窗口但新实例时，旧重放／重建／对账不变，新实例独立产生汇总。

采用事务前快照资格策略，不做提交时G18围栏。适配器读完后G18关闭，不追溯否定已观察的OPEN证据，但接受时间仍必须在快照半开窗口内。只对G20自己的会话FOR SHARE，排序ACTIVE→CLOSED与接受；绝不锁或改G18。

### 精度与数据契约

尺度1000000，B范围0..MaxInt64，E／M范围1..1000000000，有效A范围1..1000000000。使用有界任意精度整数先完整乘B×E×A×M，再一次除以10¹⁸向下截断；结果、汇总、计数和区块总值须容纳int64。溢出明确报错，不回绕、饱和或提交部分数据。禁止浮点、中间除法、最少一和小数结转。英文向量表是同一数学契约：Basic100、Advanced250、效率125、活动50、地图150、组合93、无中间截断6、小正数0、单位倍率最大结果MaxInt64。B为零或正乘积小于一合法留档，只计一次零算力；A为零无效。高级工具提高个人算力不改变容量、预留／分发奖励、发行或债务。

Migration0013紧接已验收0012，旧迁移与G18schema／保护不变；新增七张G20表且零播种。角色id/account_id唯一索引支持归属外键，不改经济行。规则表完全匹配TEST清单；工具／地图表约束TEST类型、本地规则外键和有界输入，全部不可变。会话表约束实例、角色账户归属、本地配置外键、固定绑定和非空有序时间，只可ACTIVE→CLOSED。来源表全局来源PK与规范活动ID唯一、完整会话／实例／玩家／配置／规则绑定、合成证据和有界权重，不可变。Activity全局主键及来源唯一与玩家／区块／规则无关，使用本地复合来源外键、完整非空验证快照及整数／窗口约束，不可变。Participant复合主键及完整本地会话外键，非负总值、正计数、接受时间最小最大。

不可变域记录用JSONB加生成关系列，SQL约束与Go读取的是同一身份／数值；Participant用普通标量列支持原子增量。递归且大小写敏感的封闭JSON字段契约拒绝额外别名，防止Go大小写不敏感结构解码覆盖SQL精确字段。显式拒绝缺失及JSON-null时间／上下文，避免CHECK未知值被放过。SQL NUMERIC验证单次截断公式。事实／配置／来源禁止UPDATE、DELETE、TRUNCATE；会话禁止改绑定、重开、删除或清空。派生汇总损坏只报告，不自动修复，显示BlockID也必须精确比较。历史身份歧义使所有受影响组无法核验且无读取顺序依赖；实际重复汇总的实际值／差额为null。

### 接受、去重、事务与重试

先校验意图、身份和当前角色归属，再按全局来源／活动查已提交事实。他人重放不返回原回执；修改实例、显示ID、会话、规则或活动绑定返回REPLAYED或非法规范身份，新增为零。合法原所有者精确重放先对历史失败关闭对账，再返回DUPLICATE及原事实，AppliedPower=0，不重新要求当前会话／区块资格；到期、重启、关闭或业务ID重建均不再计入，历史损坏也不允许新计入。

新事件先普通SELECT G18，再在独立连接上、SERIALIZABLE快照开始前取得一个G20专属advisory接受锁。此前按Participant锁仍因小表可能出现关系级SSI谓词锁，使不同玩家耗尽真实40001重试；本非生产基础采用单键序列化写接受，牺牲吞吐换取有界正确性，不锁上游。锁在提交／回滚后释放，释放失败关闭连接。验证内部最多30秒，并尊重更短调用方截止或取消。

每次新事务重新核验归属与全局历史，读取不可变来源，锁G20会话FOR SHARE，验证完全绑定／ACTIVE及持久合成配置／规则。数据库clock_timestamp一次捕获AcceptedAt，规范为UTC微秒。要求OpenedAt≤ObservedAt，StartedAt≤ObservedAt≤ValidatedAt≤AcceptedAt，ObservedAt<EndAt，AcceptedAt严格早于会话／来源／区块截止，来源截止晚于观察，会话截止不超区块结束；适配器观察时仍在未来的证据不能等待后变合法，客户端时钟无权威。

同事务插入不可变事实并原子增量Participant；总值／计数超界不提交任何部分事实，CreatedAt取接受时间最小值、UpdatedAt最大值。提交后丢响应重放原事实，新增零。仅40001／40P01可重试，包括首次最多五次，每次新事务／历史查找，退避1／2／4／8毫秒且受context约束；取消、其他SQL错误、溢出或不变量错误不重试。耗尽明确ErrRetryExhausted，不提高上限、不弱化断言、不跳过测试。

### 只读对账、恢复与验证

纯内存RebuildParticipants依据历史事实输入、配置、来源、会话和每条验证快照重建；不从可变G18决定旧身份／资格，会话后来关闭不撤销已提交事实。SnapshotMiningPower用REPEATABLE READ READ ONLY读取有序G20数据，历史BlockContext只从事实快照构造。对账报告每身份／字段expected、actual、任意精度十进制delta和status；匹配比较delta=0，差异列入Findings。未知、损坏或溢出组UNVERIFIABLE，expected／delta=null，不回退当前规则、零值或部分总和。重复历史、缺失／多余汇总、错误数量／算力／时间／配置及大差额均可见；重复报告不能写、修复、分发或结算。

上方T01–T90为未改范围的共用双语场景矩阵，场景数与Go事件数不同。数据库测试使用全新隔离fractal_g9_test命名库；完整每阶段G9／G11重启库独立新建。完整Normal／Race保留原断言和15m／20m包超时；历史失败原样记录，聚焦复测不能豁免完整失败。DB覆盖同来源100并发、同／不同玩家不同事件、五次重试、原子回滚、独立进程OS强杀与另一进程验证五个崩溃点、会话关闭排序、G18关闭快照／无行锁、不可变保护、迁移回滚／幂等、同业务ID不同实例、只读损坏报告及2／10／50／100／500矿工。非空经济快照覆盖FB、贡献、声望、矿石、库存／G19历史、支出／退款／历史、全部矿池／债务和预留，G20前后有序行JSON完全一致。

Windows／Linux／macOS 原生 T50 及私有三平台构建已通过。公开构建和数据库套件在导出清单独立验证，不声称 Windows／Linux 完整原生 DB 套件。


