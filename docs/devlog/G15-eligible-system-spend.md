# G15 — Eligible System Spend Orchestrator

Status: **Technical acceptance PASS · Manual acceptance PASS · Stage Close authorized**

Branch base: `86ba77b9dbe39a2bfc1352c3d7842e92e2f2009b`

## English — Primary

G15 adds the server-authoritative `systemspend.Service` and a typed Producer Registry. The only active Producers are internal test fixtures; gameplay Producers and client routes remain disabled. A caller cannot set eligibility, Contribution amount, or rule version. The existing G13 `EligibilityPolicy` resolves V1 (1 eligible FB = 1 Contribution), and the immutable spend snapshots that version and its refund permissions. Non-eligible test spends exercise the FB-only path with zero Contribution. G12 player trade is unchanged.

Migration `0007_system_spend.sql` adds immutable `system_spends`, unique operation and producer references, ledger foreign keys, classification checks, and an insert trigger that verifies the exact FB debit and matching Contribution credit. The PostgreSQL coordinator reuses G13/G11 writers in a single transaction. It serializes identical operations, returns one result for replay, rejects conflicting intent, and rolls back all effects at injected failure boundaries. Restart and lost-response retries recover the committed record without a second debit or credit.

G14 now enforces the G15 spend's immutable refund permission snapshot. Historical G13 refunds are unaffected. The G15 read path derives refund amount and status from G14 compensation rows. Reconciliation checks spend-to-ledger linkage, original rule, refunds, orphan G15 entries, FB account balances, Contribution entitlement, and recovery debt. Tests cover full and partial refund, 80-point debt after a full refund of an already-used 100-point earning, and settlement of 50 future points against that debt.

The focused real-PostgreSQL tests, full Go regression, Race, Vet, and cross-platform build results are recorded in [the G15 test report](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md). The social screenshot uses only those observed results. Technical and manual acceptance passed. Stage Close authorizes a reviewed, sanitized Public Mirror sync; it does not authorize X publication. No live economy Producer, production Contribution consumption, gameplay UI, or G16 implementation is included.

---

## 中文 — 完整对应版本

G15 新增由服务器权威控制的 `systemspend.Service` 和 Typed Producer Registry。仅内部测试 Fixture Producer 处于 Active 状态；真实 Gameplay Producer 和客户端路由仍禁用。调用方不能设定资格、Contribution 数量或规则版本。现有 G13 `EligibilityPolicy` 解析 V1（每 1 合格 FB = 1 Contribution），不可变 Spend 保存该版本与退款权限快照。非合格测试消费覆盖只扣 FB、Contribution 为零的路径。G12 玩家交易没有改变。

Migration `0007_system_spend.sql` 增加不可变的 `system_spends`、唯一 Operation 与 Producer Reference、Ledger 外键、分类约束，以及核查准确 FB 扣款与对应 Contribution Credit 的插入 Trigger。PostgreSQL Coordinator 在一个 Transaction 中复用 G13/G11 Writer。它串行化同一 Operation，重放时返回同一结果，拒绝冲突 Intent，并在注入的故障边界回滚全部效果。重启及响应丢失后的重试会恢复已提交记录，不会再次扣款或发放收益。

G14 现在执行 G15 Spend 不可变的退款权限快照；历史 G13 退款不受影响。G15 读取路径从 G14 Compensation Row 推导退款金额和状态。Reconciliation 检查 Spend 与 Ledger 关联、原始规则、退款、孤立 G15 Entry、FB 账户余额、Contribution 权益及 Recovery Debt。测试覆盖完整和部分退款、已使用 100 点收益中的 80 点后完整退款形成 80 点债务，以及未来 50 点先抵扣该债务。

专项真实 PostgreSQL 测试、完整 Go 回归、Race、Vet 和跨平台 Build 结果记载于[G15 测试报告](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md)。社交截图仅采用这些已观察到的结果。技术验收与人工验收均已通过。Stage Close 授权经审查、脱敏的 Public Mirror 同步，但不授权 X 发布。本轮不包含真实经济 Producer、生产用 Contribution Consumption、Gameplay UI 或 G16 实现。
