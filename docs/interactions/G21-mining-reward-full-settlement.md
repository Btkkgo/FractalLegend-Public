# G21 — Engineering interaction record

## English — Primary

**G21 STAGE CLOSED — PASS.** The owner accepted the completed settlement foundation. Accepted settlement merge: `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`; canonical closeout revision: `37452efe5c9d529eec04c7629c483de4156dd6fb`. Public Git history remains independent.

The complete accepted repository passed **121/121 required G21 checks**, plus **9 P0_COVERED** (130 total); normal and race each **1296 PASS**, real PostgreSQL integration PASS, crash recovery **9/9**, CI **18/18**, and native Windows/Linux/macOS deterministic equality. Receipt digest: `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`.

This public subset was independently checked against fresh, isolated PostgreSQL databases: normal **1060/1060 PASS**, race **1060/1060 PASS**, zero failures or test skips; Go vet and builds for macOS arm64 / Linux amd64 / Windows amd64 PASS. Cross-compilation is a build check, not native execution; native results above belong to the accepted complete repository.

Production settlement stays **disabled**. G18 **R=10** authority and economic parameters are unchanged. G22 has not started; no release or X publication.

Atomic PostgreSQL settlement consumes sealed G20 inputs and G18 reservation authority. CharacterID remains the beneficiary authority; canonical byte ordering and integer largest-remainder allocation are deterministic. A 500-participant R=10 settlement yields 500 grants: 10 positive and 490 zero, 10 lots/stacks, total 10. RecoveryDebt and reserved/distributed accounting remain conserved.

Immutable MINING_REWARD provenance records grant → lot → stack mappings and completed receipts. Completed exact replay uses immutable historical authority, while current inventory audit is separate. Nine real process-kill windows cover rollback and committed/ACK-loss recovery. Canonical UTC microsecond timestamps are used at G21 economic boundaries. The earlier T117 precision failure was retained and repaired at the implementation boundary without weakening assertions. Migration 0015 is additive; 0001–0014 are unchanged.

## 中文 — 完整审核版

**G21 STAGE CLOSED — PASS。** 用户已验收完整结算基础。已验收实现合并版本为 `fff6243cb2ac41c28fdcd187623b5248d8ff4b99`，canonical 收尾版本为 `37452efe5c9d529eec04c7629c483de4156dd6fb`；公开仓库继续保持独立 Git 历史。

完整已验收仓库 **121/121 必需 G21 项通过**，另有 **9 P0_COVERED**（共 130 项）；普通与竞态各 **1296 PASS**，真实 PostgreSQL 集成通过、崩溃恢复 **9/9**、CI **18/18**，Windows／Linux／macOS 真实原生执行的确定性一致性通过。回执摘要为 `cc1e297b7e61393de4165e9c369c0fa8d89e6a17b04ae43054a35986e45f8a78`。

本公开子集使用新建隔离 PostgreSQL 库独立验证：普通 **1060/1060 PASS**、竞态 **1060/1060 PASS**，零失败、零测试跳过；Go vet 及 macOS arm64／Linux amd64／Windows amd64 构建通过。交叉编译仅作为构建检查，上述真实原生结果属于完整已验收仓库。

生产结算保持**禁用**；G18 **R=10** 权威与经济参数不变。G22 未开始，未发布 Release 或 X。

PostgreSQL 原子结算消费已封存的 G20 输入与 G18 预留权威。CharacterID 保持受益人权威，规范字节排序与整数最大余数分配保证确定性。R=10 的 500 人结算得到 500 份 grant：10 份正数、490 份零分配，10 个 lot／stack，总量 10；恢复债务与预留／已分配记账保持守恒。

不可变 MINING_REWARD 来源记录 grant → lot → stack 映射与已完成回执。精确完成重放依赖不可变历史权威，当前库存审计独立执行。9 个真实进程强杀窗口覆盖回滚及已提交／ACK 丢失恢复。G21 经济时间边界统一 UTC 微秒规范；保留此前 T117 精度失败记录，在实现边界修复，未弱化断言。仅新增迁移 0015，0001–0014 未变。
