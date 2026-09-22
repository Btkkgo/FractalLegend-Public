# Architecture / 架构

## English — Primary

### Runtime direction

Browser → player intent and rendering → Protocol / API boundary → validated commands and authoritative events → Cross-platform Fractal Game Server → repositories, transactions and revisions → PostgreSQL

The browser is untrusted. It renders authoritative state and sends intent; it does not decide movement legality, damage, RNG, death, item ownership, equipment modifiers, balances, trade completion, or persistent state.

The Game Server owns world simulation, map and movement rules, monsters, combat, skills, items, inventory, equipment, character stats, and trade rules. PostgreSQL owns durable records under transactions and optimistic revisions.

### Accepted persistence boundary

The G9 Character Aggregate stores character identity, account ownership, class, level, EXP, HP/MP, safe world position, inventory item instances, equipment relations, and learned skill state. Invalid restore positions fall back to a reviewed safe spawn. RuntimeStats are recalculated rather than trusted from a client.

### Accepted trade boundary

G10 adds a transport-independent Trade Domain. Offer revisions invalidate prior confirmations. Persistent locks prevent the same item from entering two active trades. Both Character Aggregates settle in one transaction, and duplicate Finalize calls return one durable result.

### Future modules

The accepted G17 accounting chain is **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**. G18 extends it with **G17 pool → Mining Block → atomic reward reservation → immutable receipt → finalize retaining reservation or cancel releasing it**. Upstream refund shortfalls become Recovery Debt; future emission and cancellation repay it first. PostgreSQL serializes mutations and stores canonical UTC microsecond receipts. The invariant is `Net Emission Capacity = Reserved + Distributed + Remaining - RecoveryDebt`; Distributed is zero in G18. **Emission capacity and reserved reward are not player ore.** Production reward/duration, miners, power, distribution, and player ore remain unimplemented.

Production economy, wallet, blockchain adapters, asset/Ordinals verification, player mining and allocation, guilds, social, media, administration, and analytics remain separate future ownership domains. They must not bypass the Game Server or database invariants.

### Cross-platform direction

The future Game Server supports Windows, Linux, and macOS build targets. The accepted G8–G10 line verified those targets. The Legacy Windows stack is a historical content and migration source, not a required production server platform.

### Repository topology note

The sanitized public mirror began after G10 with an independent Git history. It publishes an audited subset of project-owned domains and documentation; it does not copy or reconstruct the private canonical repository history.

---

## 中文 — 完整对应版本

### Runtime 方向

Browser → Player Intent 与 Rendering → Protocol / API Boundary → Validated Command 与 Authoritative Event → Cross-platform Fractal Game Server → Repository、Transaction 与 Revision → PostgreSQL

Browser 不受信任。它渲染 Authoritative State 并发送 Intent；它不能决定 Movement Legality、Damage、RNG、Death、Item Ownership、Equipment Modifier、Balance、Trade Completion 或 Persistent State。

Game Server 负责 World Simulation、Map 与 Movement Rule、Monster、Combat、Skill、Item、Inventory、Equipment、Character Stats 和 Trade Rule。PostgreSQL 在 Transaction 与 Optimistic Revision 下保存 Durable Record。

### 已验收 Persistence Boundary

G9 Character Aggregate 保存 Character Identity、Account Ownership、Class、Level、EXP、HP/MP、安全 World Position、Inventory ItemInstance、Equipment Relation 和 Learned Skill State。无效 Restore Position 会回退到已审核的 Safe Spawn。RuntimeStats 会重新计算，而不会信任 Client。

### 已验收 Trade Boundary

G10 增加独立于 Transport 的 Trade Domain。Offer Revision 会让旧 Confirmation 失效。Persistent Lock 阻止同一 Item 进入两笔 Active Trade。双方 Character Aggregate 在一个 Transaction 中结算，重复 Finalize Call 返回同一个持久结果。

### 未来模块

已验收的 G17 记账链为 **Eligible System Spend → Versioned Emission Rule → Immutable Entry → Global Capacity Pool → Receipt → Reconciliation**。G18 增加 **G17 池 → Mining Block → 原子奖励预留 → 不可变回执 → 保留预留的完成或释放预留的取消**。上游退款缺口形成 Recovery Debt，未来发行及取消先偿债。PostgreSQL 串行处理池变化并保存 UTC 微秒规范回执。守恒式为 `净发行额度 = 已预留 + 已分发 + 剩余 - 恢复债务`；G18 的已分发为零。**发行额度和预留奖励不等于玩家矿石。**正式奖励／时长、矿工、算力、分配及玩家矿石仍未实现。

正式 Economy、Wallet、Blockchain Adapter、Asset/Ordinals Verification、玩家挖矿及分配、Guild、Social、Media、Administration 和 Analytics 都属于独立的未来 Ownership Domain。它们不能绕过 Game Server 或 Database Invariant。

### Cross-platform 方向

未来 Game Server 支持 Windows、Linux 与 macOS Build Target。已验收 G8–G10 开发线验证了这些目标。Legacy Windows Stack 只是历史内容与 Migration Source，不是 Production Server 的必需平台。

### Repository Topology 说明

Sanitized Public Mirror 在 G10 之后以独立 Git History 建立。它公开经过审计的项目自有 Domain 与 Documentation 子集，不复制或重建 Private Canonical Repository History。
