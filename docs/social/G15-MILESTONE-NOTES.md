# G15 Build in Public Milestone Notes

Status: **Technical and manual acceptance PASS; not an X post**

## English — Primary

G15 adds an internal, server-authoritative Eligible System Spend Orchestrator over the existing G11 FB Ledger, G13 Contribution Ledger, and G14 refund/recovery coordinator. An immutable SystemSpend links the FB debit, original contribution, rule-version snapshot, and refundable policy. Only internal test Producers are active; gameplay and client entry points are disabled. Full and partial refunds, debt recovery, replay, rollback, concurrency, and reconciliation are covered by real PostgreSQL checks. Use the exact counts in [the test report](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md) when preparing a future 3–5-stage or complete-milestone X update.

Suggested safe visual: [G15 engineering status card](screenshots/g15-eligible-system-spend.png), showing verified test/build results and the internal-only boundary. Do not show private repository links, local paths, credentials, database URLs, restricted assets, or personal information. This is a draft milestone note, not a standalone G15 X post. Any later combined post should begin `🧑‍💻 >` and use `#FractalLegend #Fractalbitcoin #bitcoin`; publication remains a manual user action.

## 中文 — 完整对应版本

G15 在既有 G11 FB Ledger、G13 Contribution Ledger 和 G14 退款/债务恢复 Coordinator 之上，新增内部、由服务器权威控制的 Eligible System Spend Orchestrator。不可变 SystemSpend 关联 FB Debit、原始 Contribution、规则版本快照和退款权限策略。仅内部测试 Producer 启用；Gameplay 与客户端入口均禁用。真实 PostgreSQL 检查覆盖完整及部分退款、债务恢复、重放、回滚、并发和对账。将来准备 3–5 个阶段或完整里程碑的 X 更新时，应使用[测试报告](../reports/G15-ELIGIBLE-SYSTEM-SPEND-TEST-REPORT.md)中的准确数量。

建议使用安全视觉素材：[G15 工程状态卡](screenshots/g15-eligible-system-spend.png)，展示已验证的测试/构建结果与仅限内部使用的边界。不得展示私有仓库链接、本地路径、凭据、数据库 URL、受限素材或个人信息。这是里程碑草稿记录，不是 G15 单独 X 推文。未来集合推文应以 `🧑‍💻 >` 开头，并使用 `#FractalLegend #Fractalbitcoin #bitcoin`；实际发布仍由用户手动执行。
