# G16 Recycle Economy Graph Audit

Status: **Foundation graph audited; production recipe/shop edges are absent**

## English — Primary

### Graph and method

Potential paths are `Recipe → Item → Recycle → Material → Craft → Item`, `Shop → Item → Recycle`, and `Trade-acquired Item → Recycle`. The current canonical game server has no enabled craft or shop item producer connected to G16 and no enabled production recycle rule. The only executable edge is an internal test fixture: one equipment instance with ten verified `MATERIAL_RECYCLE_SCRAP` input units yields five of the same material and two Reputation. Rule validation requires each output material to be registered and at most half of its matching verified input. FB, Contribution, and Black Iron outputs are structurally zero.

| Path | G16 executable state | Material gain | Reputation gain | FB / Contribution / Black Iron |
| --- | --- | --- | --- | --- |
| Craft → Recycle → Craft | Craft edge absent; test fixture only | No profitable closed path established | Two test Reputation per consumed fixture item | 0 / 0 / 0 |
| Buy → Recycle → Sell | Shop purchase and material resale edges absent | No executable loop | No production award | 0 / 0 / 0 |
| Material → Item → Recycle | Canonical recipe edge absent | Test rule returns 5 of 10 input units | Two test Reputation | 0 / 0 / 0 |
| Trade-acquired Item → Recycle | Ownership is checked after transfer; no production rule | At most the same 50% bound if later enabled | Future anti-farm policy required | 0 / 0 / 0 |

No profitable material loop exists in the enabled G16 graph. A future cheap or repeatable item source could still farm Reputation, so period caps and diminishing returns remain a required later integration gate. This audit does not infer profitability from legacy sale prices, names, or promotional value. A future crafting or shop implementation must be added to this graph and tested before enabling a production rule. Mining emissions remain outside this graph and cannot be sourced from recycling.

## 中文 — 完整版本

### 图与方法

潜在路径包括 `配方 → 物品 → 回收 → 材料 → 制造 → 物品`、`商店 → 物品 → 回收`，以及 `交易取得物品 → 回收`。当前规范游戏服务器没有连接 G16 的已启用制造或商店物品生产者，也没有已启用的生产回收规则。唯一可执行边是内部测试 Fixture：一件装备实例有十单位可核验 `MATERIAL_RECYCLE_SCRAP` 投入，返回五单位同种材料及两点 Reputation。规则验证要求每种输出材料已注册，且不超过对应可核验投入的一半。FB、Contribution、黑铁矿石输出在结构上为零。

| 路径 | G16 可执行状态 | 材料增益 | Reputation 增益 | FB / Contribution / 黑铁矿石 |
| --- | --- | --- | --- | --- |
| 制造 → 回收 → 制造 | 制造边不存在；只有测试 Fixture | 未建立盈利闭环 | 每件测试物品两点 Reputation | 0 / 0 / 0 |
| 购买 → 回收 → 出售 | 商店购买与材料转售边不存在 | 无可执行循环 | 无生产发放 | 0 / 0 / 0 |
| 材料 → 物品 → 回收 | 规范配方边不存在 | 测试规则返还投入十单位中的五单位 | 两点测试 Reputation | 0 / 0 / 0 |
| 交易取得物品 → 回收 | 转让后仍核验归属；生产规则未启用 | 日后启用时仍受相同 50% 上限约束 | 未来必须有防刷策略 | 0 / 0 / 0 |

已启用的 G16 图中不存在盈利材料循环。未来若出现便宜或可重复取得的物品来源，仍可能刷 Reputation，因此周期上限与递减收益是后续集成的必要门槛。本审计不从旧式售价、名称或宣传价值推断盈利性。未来制造或商店实现必须先加入图并测试，才能启用生产规则。Mining 发行不在此图内，也不能由回收取得。
