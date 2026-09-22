# Roadmap / 路线图

## English — Primary

The roadmap is milestone-driven. It does not promise dates.

### Completed foundations

- G1–G8: world, rendering, living entities, combat, AI, one Warrior skill, loot, inventory, equipment, and runtime stats.
- G9: PostgreSQL Character Aggregate persistence and restart restore.
- G10: secure item Trade Foundation.
- G11: server-authoritative internal FB Ledger Foundation with double-entry conservation, immutable history, idempotency, concurrency protection, reversal, reconciliation, and restart persistence.
- G12: atomic item + FB player Trade settlement with revision-bound offers, Gross postings, 0% fee, rollback, and restart replay.
- G13: internal non-transferable Contribution Ledger with versioned 1:1 eligible-system-spend rule, atomic FB linkage, idempotency, rollback, reconciliation, and no live producer.
- G14: atomic linked FB refund/reversal and Contribution compensation with bounded partial refunds, recovery debt, future-credit debt repayment, and review hold; no live producer.
- G15: internal Eligible System Spend orchestration over the existing ledgers; no live gameplay producer.
- G16: internal Recycle Migration Foundation with atomic item consumption, allowed material and non-transferable Reputation output, immutable canonical receipt, and no production rule or gameplay entry.
- G17: internal global Black Iron emission-capacity pool with versioned immutable entries and receipts, refund compensation, and reconciliation; no player ore or production emission ratio.
- G18: server-authoritative Mining Blocks, atomic reservation against the G17 pool, cancellation release, immutable receipts, crash recovery, and Recovery Debt / Future Offset; no reward distribution or player ore.

### Next

- Plan later economy and blockchain adapters as separately reviewed milestones; no live deposit, withdrawal, wallet, or chain feature exists.
- Continue the browser product experience using accepted server authority.
- The G14 refund/compensation hard gate is closed. A real eligible system-spend producer, production Contribution spending, and manufacturing use remain separate future work.
- G16 completed only the internal recycle foundation. Production recycle rules, Reputation caps, gameplay integration, and Mining require separate review.
- G17 completed capacity accounting; G18 completed Mining Block reservation only. Production reward and duration, power, tools, reward distribution, and player ore require separate review; G19 has not started.

### Future

- Full browser gameplay and broader class/skill progression.
- Bosses, parties, and team dungeons.
- Guild, guild roles, reputation, missions, wars, territory, and siege.
- Player mining using the accepted global capacity and block reservation foundations, pending future power, tool, and distribution rules.
- Wallet and domain identity.
- Ordinals ownership, metadata verification, mapping, and activation.
- Fractal / Bitcoin deposit, withdrawal, confirmation, and reorg handling.
- Marketplace and Auction beyond the accepted direct item + FB Trade settlement.
- Profiles, follows, feeds, guild/party chat, voice, video, tips, and red packets.
- Admin, security hardening, performance, mobile support, and Beta readiness.

Siege is intended to reward equipment, skills, titles, visual prestige, and other non-monetary privileges rather than directly emitting FB or Contribution, reducing permanent compounding advantages. This is a product principle, not an implemented rule.

---

## 中文 — 完整对应版本

Roadmap 按里程碑推进，不承诺日期。

### 已完成 Foundation

- G1–G8：World、Rendering、Living Entity、Combat、AI、一项 Warrior Skill、Loot、Inventory、Equipment 与 Runtime Stats。
- G9：PostgreSQL Character Aggregate Persistence 与 Restart Restore。
- G10：安全 Item Trade Foundation。
- G11：Server-authoritative Internal FB Ledger Foundation，包含 Double-entry Conservation、Immutable History、Idempotency、Concurrency Protection、Reversal、Reconciliation 与 Restart Persistence。
- G12：原子 Item + FB 玩家 Trade Settlement，包含 Revision-bound Offer、Gross Posting、0% Fee、Rollback 与 Restart Replay。
- G13：内部不可转账的 Contribution Ledger，包含版本化 1:1 合格 System Spend Rule、原子 FB 关联、Idempotency、Rollback 与 Reconciliation；尚无真实 Producer。
- G14：原子关联 FB Refund/Reversal 与 Contribution Compensation，支持有上限的部分退款、Recovery Debt、未来 Credit 先偿债与 Review Hold；尚无真实 Producer。
- G15：在现有账本上建立内部 Eligible System Spend 编排；尚无真实玩法 Producer。
- G16：内部 Recycle Migration Foundation，包括原子物品消费、允许材料与不可转让 Reputation 产出、不可变规范 Receipt；没有生产规则或玩法入口。
- G17：内部全服黑铁矿石发行额度池，包含版本化不可变流水与回执、退款补偿和对账；不发放玩家矿石，正式发行比例未定。
- G18：服务器权威 Mining Block、针对 G17 池的原子预留、取消释放、不可变回执、崩溃恢复与 Recovery Debt / Future Offset；不分配奖励或发放玩家矿石。

### 下一步

- 把后续 Economy 与 Blockchain Adapter 作为独立审核阶段规划；目前没有真实 Deposit、Withdrawal、Wallet 或 Chain 功能。
- 继续以已验收 Server Authority 扩展 Browser Product Experience。
- G14 Refund/Compensation 硬门已关闭。真实合格 System Spend Producer、生产用 Contribution 消费与 Manufacturing 用途仍须作为独立未来工作。
- G16 只完成内部回收基础。生产回收规则、Reputation 上限、玩法集成及 Mining 都需要分别审查。
- G17 完成额度记账；G18 只完成 Mining Block 预留。正式奖励与时长、Power、Tool、奖励分配与玩家矿石均需单独审核；尚未开始 G19。

### 未来

- 完整 Browser Gameplay 与更广泛 Class / Skill Progression。
- Boss、Party 与 Team Dungeon。
- Guild、Guild Role、Reputation、Mission、War、Territory 与 Siege。
- 基于已验收的全服额度和区块预留基础规划未来玩家挖矿；Power、Tool 与分发规则仍待规划。
- Wallet 与 Domain Identity。
- Ordinals Ownership、Metadata Verification、Mapping 与 Activation。
- Fractal / Bitcoin Deposit、Withdrawal、Confirmation 与 Reorg Handling。
- 已验收的 Direct Item + FB Trade Settlement 以外的 Marketplace 与 Auction。
- Profile、Follow、Feed、Guild/Party Chat、Voice、Video、Tip 与 Red Packet。
- Admin、Security Hardening、Performance、Mobile Support 与 Beta Readiness。

Siege 计划奖励 Equipment、Skill、Title、Visual Prestige 与其他非货币 Privilege，而不是直接发行 FB 或 Contribution，以减少永久滚雪球优势。这是产品原则，不是已实现规则。
