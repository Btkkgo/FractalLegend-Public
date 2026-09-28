# G21 — Complete 130 Matrix / 完整矩阵

Accepted complete repository: 121 PASS + 9 P0_COVERED. / 已验收完整仓库：121 通过 + 9 前置条件覆盖。

| ID | Scenario / 场景 | Result / 结果 |
|---|---|---|
| G21-T001 | Exact ratio / 整除 | PASS |
| G21-T002 | Remainder / 余数 | PASS |
| G21-T003 | Equal remainder / 等余数 | PASS |
| G21-T004 | R smaller than N / 奖励少于人数 | PASS |
| G21-T005 | Single beneficiary / 单受益人 | PASS |
| G21-T006 | Zero weight / 零权重 | PASS |
| G21-T007 | No activities / 无活动 | PASS |
| G21-T008 | All zero / 全零 | PASS |
| G21-T009 | Invalid negative / 负权重 | PASS |
| G21-T010 | Invalid R / 非法奖励 | PASS |
| G21-T011 | Wide product / 宽乘积 | PASS |
| G21-T012 | Total power boundary / 总算力边界 | PASS |
| G21-T013 | Total power overflow / 总算力溢出 | PASS |
| G21-T014 | Inventory max / 库存最大数量 | PASS |
| G21-T015 | Inventory overflow / 库存数量溢出 | PASS |
| G21-T016 | Pool bigint overflow / 矿池字段溢出 | PASS |
| G21-T017 | Input permutation / 顺序乱序 | PASS |
| G21-T018 | Case-sensitive bytes / 大小写字节 | PASS |
| G21-T019 | Numeric-looking ID / 数字样式ID | PASS |
| G21-T020 | Conservation properties / 守恒性质 | PASS |
| G21-T021 | Authority binding / 权威绑定 | PASS |
| G21-T022 | Historical unbound beneficiary / 历史未绑定角色 | PASS |
| G21-T023 | Wrong account / 错账号 | PASS |
| G21-T024 | Forged beneficiary / 伪造受益人 | PASS |
| G21-T025 | Same beneficiary sessions / 同受益多会话 | PASS |
| G21-T026 | Duplicate source / 重复来源 | PASS |
| G21-T027 | Independent characters / 独立角色 | PASS |
| G21-T028 | Aliases to one character / 多玩家别名 | PASS |
| G21-T029 | Inconsistent Player/Character / 玩家角色绑定不一致 | PASS |
| G21-T030 | Owner changed after seal / 封存后归属改变 | PASS |
| G21-T031 | Corrupt summary / 汇总损坏 | PASS |
| G21-T032 | Mixed activity rules / 混合规则 | PASS |
| G21-T033 | Shared first commit / 共享接受先提交 | P0_COVERED |
| G21-T034 | Shared first rollback / 共享接受回滚 | P0_COVERED |
| G21-T035 | Seal first / 封存先获锁 | P0_COVERED |
| G21-T036 | Earlier transaction late gate / 早事务迟获锁 | PASS |
| G21-T037 | Seal rollback / 封存回滚 | P0_COVERED |
| G21-T038 | Parallel accepting processes / 多进程接受 | PASS |
| G21-T039 | Concurrent same-instance seal / 同实例双封存 | PASS |
| G21-T040 | Conflicting seal facts / 封存事实冲突 | PASS |
| G21-T041 | Post-seal exact event / 封存后活动重放 | P0_COVERED |
| G21-T042 | Post-seal changed event / 封存后改绑定 | PASS |
| G21-T043 | Post-seal new old-time source / 旧时间新来源 | PASS |
| G21-T044 | Completed settlement exact replay / 已完成结算精确重放 | PASS |
| G21-T045 | Missing acceptance state / 缺接受状态行 | PASS |
| G21-T046 | Snapshot after wait / 等待后快照 | P0_COVERED |
| G21-T047 | App-role bypass / 运行角色绕协议 | PASS |
| G21-T048 | Read-only upstream / 上游只读 | PASS |
| G21-T049 | Atomic reservation binding / 同事务预留绑定 | PASS |
| G21-T050 | Unknown old reservation / 未绑定旧预留 | P0_COVERED |
| G21-T051 | Display ID history / 展示ID历史 | PASS |
| G21-T052 | Changed R evidence / 篡改预留量 | PASS |
| G21-T053 | Duplicate instance binding / 重复实例绑定 | PASS |
| G21-T054 | Duplicate source binding / 重复来源绑定 | PASS |
| G21-T055 | Not finalized / 未终结区块 | PASS |
| G21-T056 | Released reservation / 已释放预留 | PASS |
| G21-T057 | Upstream identity guards / 上游身份保护 | PASS |
| G21-T058 | Missing source history / 缺预留历史 | PASS |
| G21-T059 | Whole transaction / 完整原子事务 | PASS |
| G21-T060 | Remaining unchanged / 不重复扣Remaining | PASS |
| G21-T061 | Same command sequential / 同命令顺序 | PASS |
| G21-T062 | Same command parallel / 同命令并发 | PASS |
| G21-T063 | New command same instance / 新命令同实例 | PASS |
| G21-T064 | Same command changed instance / 同命令换实例 | PASS |
| G21-T065 | Same command changed rule / 同命令换规则 | PASS |
| G21-T066 | Same command changed resolved set / 同命令改集合 | PASS |
| G21-T067 | Same source across instances / 跨实例同预留 | PASS |
| G21-T068 | Two blocks shared owners / 双区块同库存 | PASS |
| G21-T069 | Concurrent aggregate save / 并发整体保存 | PASS |
| G21-T070 | Concurrent migration / 并发迁移 | PASS |
| G21-T071 | Owner revision overflow / 角色版本溢出 | PASS |
| G21-T072 | Slot overflow / 槽位溢出 | PASS |
| G21-T073 | Zero grant revision / 零奖励版本 | PASS |
| G21-T074 | Failure per write stage / 各写阶段失败 | PASS |
| G21-T075 | Retry classification / 重试分类 | PASS |
| G21-T076 | Timeout and retry exhaustion / 超时和重试耗尽 | PASS |
| G21-T077 | Seal kill before commit / Seal提交前强杀 | P0_COVERED |
| G21-T078 | Seal commit ACK lost / Seal提交丢ACK | P0_COVERED |
| G21-T079 | Settlement kill before inventory / 库存前强杀 | PASS |
| G21-T080 | Settlement kill after inventory / 库存后强杀 | PASS |
| G21-T081 | Settlement kill after pool / 矿池后强杀 | PASS |
| G21-T082 | Settlement kill before receipt / 回执前强杀 | PASS |
| G21-T083 | Settlement kill before commit / 提交前强杀 | PASS |
| G21-T084 | Settlement commit ACK lost / 结算提交丢ACK | PASS |
| G21-T085 | Refund from Remaining / 退款先余量 | PASS |
| G21-T086 | Refund beyond Remaining / 退款超余量 | PASS |
| G21-T087 | Partial Remaining refund / 部分余量退款 | PASS |
| G21-T088 | Debt pauses new settle / 债务暂停新结算 | PASS |
| G21-T089 | Debt preserves completed replay / 债务保留完成重放 | PASS |
| G21-T090 | Debt partial repay / 部分偿债 | PASS |
| G21-T091 | Debt exact repay / 恰好偿债 | PASS |
| G21-T092 | Debt repay surplus / 偿债有余量 | PASS |
| G21-T093 | Settle after debt clear / 偿债后结算 | PASS |
| G21-T094 | Refund races settle / 退款结算竞争 | PASS |
| G21-T095 | Refund replay / 退款重放 | PASS |
| G21-T096 | Recovery full rebuild / 完整恢复重建 | PASS |
| G21-T097 | Recovery missing settlement / 恢复缺结算历史 | PASS |
| G21-T098 | Recovery revision gap / 恢复版本缺口 | PASS |
| G21-T099 | Cancel other reserved block / 取消其他预留 | PASS |
| G21-T100 | Cancel consumed block / 取消已消费区块 | PASS |
| G21-T101 | Reward independent source / 奖励独立来源 | PASS |
| G21-T102 | Reward plus migrated inventory / 迁移库存并存 | PASS |
| G21-T103 | Fake migration source / 伪装迁移来源 | PASS |
| G21-T104 | Duplicate issuance / 重复发行 | PASS |
| G21-T105 | Projection mismatch / 投影损坏 | PASS |
| G21-T106 | Movement and deletion guard / 移动删除保护 | PASS |
| G21-T107 | History immutability / 历史不可变 | PASS |
| G21-T108 | Additive migration / 增量迁移 | PASS |
| G21-T109 | DDL failure rollback / DDL失败回滚 | PASS |
| G21-T110 | Unsafe old writer rollback / 旧写入回退 | PASS |
| G21-T111 | Sybil limitation / 女巫限制反例 | PASS |
| G21-T112 | Low power limitation / 低算力限制反例 | PASS |
| G21-T113 | Production exclusions / 生产排除 | PASS |
| G21-T114 | Windows native / Windows原生 | PASS |
| G21-T115 | Linux native / Linux原生 | PASS |
| G21-T116 | macOS native / macOS原生 | PASS |
| G21-T117 | Timezone and strict equality / 时区精确相等 | PASS |
| G21-T118 | 2/10/50/100 participants / 小中规模 | PASS |
| G21-T119 | 500 participants / 500边界 | PASS |
| G21-T120 | 501 and proposed input/payload caps / 超限边界 | PASS |
| G21-T121 | G17/G18 regression / 容量预留回归 | PASS |
| G21-T122 | G19 regression / 迁移回归 | PASS |
| G21-T123 | Full regression gate / 完整回归 | PASS |
| G21-T124 | Real CI gate / 真实CI | PASS |
| G21-T125 | Forbidden client fields / 客户端禁字段 | PASS |
| G21-T126 | Production service gate / 生产关闭 | PASS |
| G21-T127 | Preview read-only / 只读Preview | PASS |
| G21-T128 | Authentication and privilege / 鉴权权限 | PASS |
| G21-T129 | Receipt corruption / 回执损坏 | PASS |
| G21-T130 | Isolation and document safety / 隔离和文档安全 | PASS |
