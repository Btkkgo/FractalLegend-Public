# G20 — Verification

## English — Primary

**G20 Mining Power Foundation: final human acceptance PASS; private canonical closure complete.**

The accepted chain is **G17 Global Mining Pool → G18 Mining Block + Recovery → G18.1 Stable Block Identity → G20 Server-authoritative Mining Power**. G17 capacity/reservation is not player ore. G19 identity migration preserves existing quantities without issuance.

G20 records immutable ValidatedMiningActivity as authority and derives Participant aggregates. Stable BlockInstanceID spans sessions/events/history. G18 uses a read-only ordinary SELECT without upstream row locks or cross-domain foreign keys, saving an immutable validation snapshot. PostgreSQL SERIALIZABLE commits activity and aggregate atomically, with global SourceEvent uniqueness and at most five attempts including the first. Synthetic Tool/Map inputs are server-resolved, final client power is rejected, integer scale is 1,000,000 and one final floor division determines power. Canonical replay timestamps use `t.UTC().Round(0)`; strict receipt equality is retained. Reconciliation is read-only and never repairs unknown history.

G20 has **90/90 scenarios PASS**. Private PR and post-merge CI each have **10/10 jobs PASS**, full Normal and Race each **882/882 PASS**, zero failures/skips/data race. Native Windows/Linux/macOS T50 has 24 identical logical results with digest `f5f2e54d342d02544c54565d488aa1e096fd9208c4ea4d23b291fa2f5ceb3d60`. The public subset is tested independently; its results appear in the public export manifest rather than borrowing private counts.

This is a TEST foundation, not a public game release. Production Service remains fail-closed. **NOT IMPLEMENTED:** reward distribution, block reward settlement, ore claim, mining pool deduction, production reward/duration/tool/map rules and G21. G20 mutates no G18, FB, Contribution, Player Ore or Mining Pool state. Social material remains an unpublished draft.

## 中文 — 完整对应版

**G20 算力基础：最终人工验收 PASS，私有 canonical 收尾完成。**

已验收链条为 **G17 全服矿池 → G18 挖矿区块与恢复 → G18.1 稳定区块身份 → G20 服务器权威算力**。G17 容量／预留不等于玩家矿石；G19 身份迁移保持既有数量，不发行矿石。

G20 以不可变 ValidatedMiningActivity 为权威、Participant 为派生汇总。稳定 BlockInstanceID 贯穿会话／事件／历史。G18 普通只读 SELECT 无上游行锁或跨域外键，保存不可变验证快照。PostgreSQL SERIALIZABLE 原子提交事实与汇总；SourceEvent 全局唯一，含首次最多五次。工具／地图输入由服务器解析合成数据，拒绝客户端最终算力；整数定点比例 1,000,000，只作一次最终除法向下截断。重放时间采用 `t.UTC().Round(0)`，保留回执严格相等。对账只读，不修复未知历史。

G20 **90/90 场景通过**；私有 PR 及合并后 CI 各 **10/10 作业通过**，完整 Normal／Race 各 **882/882**，零失败、跳过、数据竞争。Windows／Linux／macOS 原生 T50 的 24 个逻辑结果完全一致，摘要如上。公开子集独立运行验证，结果在公开导出清单中记录，不借用私有数量。

这是 TEST 基础，不是公开游戏发布；生产 Service 失败关闭。**未实现：**奖励分发、区块奖励结算、矿石申领、矿池扣减、正式奖励／时长／工具／地图规则及 G21。G20 不修改 G18、FB、Contribution、玩家矿石或 Mining Pool。社交素材保持未发布草稿。


## Independent public verification / 公开独立验证

Fresh isolated PostgreSQL databases were used sequentially: public full Normal **646/646 PASS**, full Race **646/646 PASS**, **0 FAIL / 0 SKIP / 0 data race**. All 90 G20 scenario assertions mapped to passing events in both runs; the three-native T50 receipts remain exact-source-equivalent to the exported calculator/tests. Go Vet and Linux/Windows/macOS all-package builds PASS. These public checks were run locally under the existing mirror workflow; private PR/canonical CI results are identified separately.

All **215** tracked/nonignored public files passed Secret, Privacy/Personal Path, Legacy Source, Restricted Asset, active/external SVG and relative-link audits with **0 findings**. The **34** code paths remain byte-identical to accepted canonical. All **9** approved baseline PNG hashes are unchanged. No new binary/archive/oversized file, dependency or licensing change, production data, credential, private Git object or raw conversation is exported. Public commits use the existing GitHub noreply identity; Git history remains independent.

顺序使用各自全新隔离 PostgreSQL 测试库：公开完整 Normal **646/646**、Race **646/646**，失败／跳过／数据竞争均为 **0**。两个运行均将 90 个 G20 场景断言对应到通过事件；三平台原生 T50 回执与导出的计算器／测试源完全相同。Go Vet 和 Linux／Windows／macOS 全包构建通过。这些公开检查按既有镜像流程本地运行，私有 PR／canonical CI 结果单独标明。

全部 **215** 个受追踪／非忽略公开文件通过秘密、隐私／个人路径、旧源码、受限资产、活动／外部 SVG 与相对链接检查，**零发现**。**34** 个代码路径保持与已验收 canonical 逐字节相同；**9** 个既有批准 PNG 哈希不变。没有新二进制／归档／超大文件、依赖／许可变更、生产数据、凭据、私有 Git 对象或原始对话。公开提交使用既有 GitHub noreply 身份，保持独立历史。
