# ADR-G21P0-001: Settlement prerequisite authority / 结算前置权威

Status: **ACCEPTED — G21-P0 prerequisite architecture only**. Final human acceptance PASS on 2026-09-28; native Windows/Linux/macOS and 14/14 branch CI PASS. Full G21 reward settlement remains subject to the next human gate. / 状态：**已批准——仅 G21-P0 前置架构**。2026-09-28 最终人工验收 PASS；Windows／Linux／macOS 原生测试与分支 CI 14/14 PASS。完整 G21 奖励结算仍须下一轮人工门槛批准。

## Decision / 决议

Use one G20-owned PostgreSQL acceptance row per BlockInstanceID. First Activity insert holds a compatible row lock; Seal takes an exclusive lock under READ COMMITTED and reads Activity in a new statement. Persist the sealed facts as fixed-order JSON plus SHA-256. Use immutable G18 OPEN source/receipt→instance binding and a server-validated CharacterID beneficiary record. Treat both absent bindings as ineligible. / 每个 BlockInstanceID 使用一条 G20 所属 PostgreSQL 接受状态行。首个 Activity 插入持有兼容行锁；Seal 在 READ COMMITTED 下持排他锁，再由新语句读取 Activity。封存事实以固定顺序 JSON 加 SHA-256 持久化。使用不可变 G18 OPEN 来源／receipt→实例绑定，以及服务端验证的 CharacterID 受益人记录。任一绑定缺失都视为不合格。

Extend G17/G18's one recovery authority to nonzero Distributed with `C=S+D+M−H`; a synthetic TEST transfer moves S to D without touching M. Model `MINING_REWARD` with its own immutable source lot and inventory projection, reusing the G19 definition rather than migration provenance. Both writers remain private to isolated test databases. / 在单一 G17/G18 恢复权威中扩展非零 Distributed，守恒式为 `C=S+D+M−H`；合成 TEST 转移只把 S 移至 D，不改 M。用独立不可变 lot 和库存投影表示 `MINING_REWARD`，复用 G19 definition，但不复用 migration provenance。两种写入器仅限隔离测试库的私有路径。

## Rationale and consequences / 理由与后果

The row lock supplies a database serialization point shared across processes and restarts. READ COMMITTED lets Seal see an acceptance committed while it waited. The G18 binding is created with its source rather than inferred later. CharacterID is an actual inventory owner row, not a PlayerID spelling convention. Four-component recovery replay makes a distribution visible to reconciliation and allows later refunds to create debt without reversing an inventory grant. G19 receipts remain frozen. / 行锁提供跨进程及重启后共同遵守的数据库序列化点。READ COMMITTED 让 Seal 看见等待期间已提交的接受事务。G18 绑定随来源创建，不事后推断。CharacterID 来自真实库存 owner 行，不依赖 PlayerID 字符串约定。四分量恢复重放使分配能够被核对，并允许后续退款形成债务而不撤销库存发放。G19 receipts 保持冻结。

This decision does not implement allocation, claims, scheduler, production reward policy, real player Ore, or a production route. Full G21 must separately review those contracts and receive authorization. / 本决议不实现分配、领取、调度器、生产奖励政策、真实玩家矿石或生产入口。完整 G21 必须另行审查这些契约并取得授权。
