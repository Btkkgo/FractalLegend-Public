# ADR 0015: G16 Recycle Migration Foundation

Status: **Technical and manual acceptance PASS; PR merged into canonical**

## English — Primary

### Context and decision

Legacy equipment recycling into currency would bypass the G11–G15 economy boundaries. G16 replaces that prospective behavior with an internal, server-authoritative item-instance operation yielding only permitted materials and non-transferable Reputation. The client may identify an item, but cannot choose quantity, tier, value, price, FB, Contribution, ore, or rule contents. There is no gameplay route, NPC, UI, or legacy-script adapter in this stage.

The typed registry resolves exact item templates and versioned rules. It rejects unknown and disabled rules, unregistered material types, Black Iron Ore outputs, negative or excessive quantities, and any material return above 50% of verified same-material input. The 50% bound is a safety ceiling, not a final balance target. Names, display prices, and purchase labels are not input evidence. No production rule is enabled because the current game server has no canonical craft bill of materials to prove a live item's input. The only active rule is an isolated test fixture. A future rule version must supply verifiable input provenance before a gameplay gate can enable it.

### Economy and security invariants

Recycle awards FB = 0 and Contribution = 0 because neither is earned by destroying equipment. Black Iron Ore = 0 because its emission budget belongs to a later Mining phase. Reputation records non-currency participation and has no transfer, withdrawal, FB exchange, or Contribution route. Rule fields reserve period-cap and diminishing-return policy boundaries; final periods and weights remain undecided.

The existing `character_inventory_items` table remains the inventory authority. Migration 0008 adds permanent `item_instance_lifecycle` revision and consumed state, with an insert/update trigger that advances revision on aggregate rewrites and rejects resurrection of consumed IDs. The recycle transaction locks the owning character and item, validates owner, revision, template, inventory state, equipment state, trade locks and active offers, then marks the instance consumed and deletes its inventory row. It credits material instances into the same inventory, updates the minimal Reputation account and immutable entry, and writes an immutable receipt, rule snapshot, and audit event before committing. Any failure rolls back every effect. The character revision advances so an older gameplay aggregate cannot restore the item.

An operation-scoped advisory lock and unique receipt operation/item keys enforce idempotency. Exact retries return the original receipt, including after repository reopen and rule disablement; conflicting retries fail closed. Character-row locking serializes only operations against that character, not all players. Reconciliation compares receipts, consumed lifecycle records, material credits and current material inventory, Reputation entries and balances, rule snapshots, zero forbidden awards, and audit events. Future material spending will require its own consumption evidence before this current-inventory comparison can be relaxed.

G16 precedes Mining so recycling cannot become an unbudgeted ore source or profitable material loop before mining emissions are defined. Real gameplay entry remains disabled until a separate integration gate reviews canonical input evidence, economic balance, reputation anti-farm limits, protected external assets, and live-asset safety.

## 中文 — 完整版本

### 背景与决策

旧式装备回收为货币会绕过 G11–G15 的经济边界。G16 将未来该行为改为内部、由服务器权威控制的物品实例操作，只产出允许的材料与不可转让的 Reputation。客户端可指出物品，但不能决定数量、档位、价值、价格、FB、Contribution、矿石或规则内容。本阶段没有真实玩法路由、NPC、UI 或旧脚本适配器。

类型化注册表解析精确物品模板及版本化规则。它拒绝未知和禁用规则、未注册材料类型、黑铁矿石输出、负数或超量数量，以及超过可核验同种材料投入 50% 的任何返还。50% 是安全上限，不是最终平衡目标。物品名称、显示售价和购买标签不构成投入证据。当前游戏服务器没有规范制造配方材料清单可证明真实物品投入，因此未启用任何生产规则；唯一启用的是隔离测试 Fixture。未来启用真实玩法前，新规则版本必须提供可核验的投入来源。

### 经济与安全不变量

回收发放 FB = 0、Contribution = 0，因为销毁装备不产生这两种权益。黑铁矿石 = 0，因为其发行预算属于后续 Mining 阶段。Reputation 记录非货币参与，没有转让、提现、兑换 FB 或产生 Contribution 的通道。规则字段预留周期上限和递减收益策略边界；最终周期与权重尚未决定。

现有 `character_inventory_items` 表继续作为库存权威。迁移 0008 增加永久的 `item_instance_lifecycle` 版本与已消费状态；插入/更新 Trigger 在 Aggregate 重建时推进版本，并拒绝已消费 ID 复活。回收事务锁定归属角色和物品，核验归属、版本、模板、库存状态、装备状态、交易锁及活跃报价，然后标记已消费并删除库存行。它将材料实例写回同一库存，更新最小 Reputation 账户及不可变 Entry，并在提交前写入不可变 Receipt、规则快照和审计事件。任一失败均回滚所有影响。角色版本同步增加，使旧玩法 Aggregate 无法恢复该物品。

按操作 ID 的 Advisory Lock 及 Receipt 的操作/物品唯一键确保幂等。完全相同的重试返回原 Receipt，包括重开 Repository 或规则被禁用以后；冲突重试默认拒绝。角色行锁只串行化同一角色的操作，不锁住全部玩家。对账比较 Receipt、已消费生命周期、材料 Credit 与当前材料库存、Reputation Entry 与余额、规则快照、禁止奖励零值以及审计事件。未来材料消费启用后，必须先有对应消费证据，才能放宽当前的库存存在性比较。

G16 必须先于 Mining，以防回收在矿石发行预算确定前成为预算外矿石来源或盈利材料循环。真实玩法入口保持关闭，直到独立 Integration Gate 审查规范投入证据、经济平衡、Reputation 防刷上限、受保护外部资产与真实资产安全。
