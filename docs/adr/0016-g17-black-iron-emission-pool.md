# ADR 0016: G17 Global Black Iron Emission Pool Foundation

Status: **G17 CLOSED / PASS; no production ratio or release**

## English — Primary

### Decision and authority

The global pool accounts for the maximum future Black Iron Ore emission capacity implied by already persisted **G15 eligible system FB spends**. It is a server aggregate, not a player balance, FB, Contribution, withdrawable currency, or an ore item. G17 does not award ore. Only an internal server service may apply a persisted G15 `spend_id`; the caller supplies no eligibility or emission amount. The PostgreSQL adapter reloads and validates the G15 spend, its FB ledger transaction, and its Contribution entry before adding capacity. Non-eligible G15 spends produce no entry or capacity. Other activities have no G15 eligible source identity and cannot enter this accounting path.

The only active G17 rule is explicitly named `DEV_G17_1_TO_1`, where one eligible FB creates one development emission unit. It is a **test/development rule, not final production economics**. There is no production rule or live gameplay caller; an unknown rule fails closed. Every immutable entry and receipt records the rule version used. A later Balance Phase must approve a production ratio and activation separately.

### Persistence and transaction boundaries

Additive migration `0009_black_iron_emission_pool.sql` creates one `black_iron_emission_pools` aggregate, immutable `black_iron_emission_entries`, and immutable `black_iron_emission_receipts`. No G11–G16 table, schema column, or historical row is rewritten. Each initial entry has a unique G15 spend source; each refund entry has a unique G14 compensation source and a negative amount. Foreign keys bind both to existing authoritative records. PostgreSQL locks the original FB transaction before application, matching G14's refund order, then locks the pool. Entry, pool update, receipt, and any already persisted refund compensations share one transaction. Every returned receipt is read back from PostgreSQL with UTC microsecond timestamps, so first response, replay, restart, and snapshot values compare exactly.

G14 keeps the original eligible spend immutable. Refunds and reversals create immutable FB and Contribution compensation records. For a G15 spend that already has an emission entry, G17 appends a negative emission compensation **inside that same G14 transaction**; a failure rolls back FB, Contribution, and emission effects together. If a refund predates initial G17 application, that application atomically records the original capacity and all existing compensation entries. Historical G13 refunds without a G15 spend are unchanged. Replays return the original stored receipt and never repeat a pool update.

### Conservation and review

The pool records gross eligible spend observed, refunded spend, net emission capacity, revision, rule version, and timestamps. Currently `reserved = distributed = 0`, so `total capacity = remaining capacity`. The immutable journal sums to those aggregate values; a read-only reconciliation checks source binding, rule amounts, refund sequence, receipt chain, and unobserved eligible G15 spends. Mismatch is reported as an invariant failure, never silently repaired. PostgreSQL constraints prohibit negative capacity and premature reservation or distribution. G17 has no Mining Block, Mining Power, Mining Tool, Mining Map, player ore inventory, ore mint, or Bun migration.

The internal application is explicit because the production emission ratio is not approved. Existing eligible G15 rows in an upgraded database are not silently backfilled with the development ratio; reconciliation reports them until an authorized future rule and controlled application are available. A separate internal test-isolation follow-up remains open; G17 tests use fresh isolated databases without implementing the broader database-isolation framework.

## 中文 — 完整审核版

### 决策与权威来源

全服发行池记录**已持久化的 G15 合格系统 FB 消费**所允许的未来黑铁矿石最大发行额度。它是服务器汇总，不是玩家余额、FB、Contribution、可提现货币或矿石物品。G17 不发放矿石。只有内部服务器服务可应用已保存的 G15 `spend_id`；调用方不能提供合格资格或发行数量。PostgreSQL Adapter 在增加额度前重新读取并验证 G15 消费、对应 FB Ledger Transaction 与 Contribution Entry。非合格 G15 消费不产生流水或额度。其他活动没有 G15 合格来源身份，无法进入此记账路径。

G17 唯一启用的规则明确命名为 `DEV_G17_1_TO_1`，即每 1 合格 FB 对应 1 个开发用发行单位。这是**测试／开发规则，不是最终生产经济参数**。目前没有生产规则，也没有真实玩法调用方；未知规则一律拒绝。每条不可变流水和回执都记录所用规则版本。正式比例及启用须等待后续 Balance Phase 单独批准。

### 持久化与事务边界

增量迁移 `0009_black_iron_emission_pool.sql` 新建单例 `black_iron_emission_pools` 汇总、不可变 `black_iron_emission_entries` 与不可变 `black_iron_emission_receipts`。不改写 G11–G16 的表、Schema Column 或历史记录。每条初始流水唯一绑定 G15 消费来源；每条退款流水唯一绑定 G14 补偿来源，并记录负数量。外键将两者绑定到已有权威记录。PostgreSQL 在应用时先锁原 FB Transaction，再锁发行池，与 G14 退款锁序一致。流水、池汇总更新、回执以及此前已持久化的退款补偿处于同一事务。所有返回回执均从 PostgreSQL 重新读取，采用 UTC 微秒时间，使首次响应、重放、重启和快照值可精确比较。

G14 保留原始合格消费的不可变记录。退款与冲正创建不可变的 FB 和 Contribution 补偿记录。如果 G15 消费已有发行流水，G17 在**同一个 G14 事务内**追加负数发行补偿；任何失败都让 FB、Contribution 和发行结果一起回滚。如果退款早于首次 G17 应用，首次应用在单个事务内同时写入原额度及所有既有补偿流水。没有 G15 Spend 的历史 G13 退款保持原样。重复请求返回原已保存回执，不重复更新池。

### 守恒与审计

池记录已观察的合格消费总额、退款总额、净发行额度、Revision、规则版本与时间。目前 `reserved = distributed = 0`，因此 `total capacity = remaining capacity`。不可变流水合计必须与这些汇总一致；只读对账检查来源绑定、规则数量、退款次序、回执链及尚未观察的 G15 合格消费。差异作为 Invariant Failure 报告，不自动静默修复。PostgreSQL 约束禁止负额度和过早预留或分发。G17 不包含 Mining Block、Mining Power、Mining Tool、Mining Map、玩家矿石库存、矿石铸造或馒头迁移。

由于正式发行比例尚未批准，内部应用必须显式触发。升级数据库中已有的 G15 合格记录不会被开发比例静默回填；正式规则及受控应用得到授权前，对账会报告这些未处理来源。独立的内部测试隔离后续事项保持开放；G17 测试采用全新隔离数据库，不实现更广泛的数据库隔离框架。
