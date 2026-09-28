# Public command boundary / 公开命令入口边界

## English — Primary

This sanitized mirror intentionally contains no executable command entry points. The private runtime assembly and deployment wiring are excluded by [Public Mirror Policy](../../../PUBLIC-MIRROR-POLICY.md) and [Code Provenance](../../../PUBLIC-CODE-PROVENANCE.md).

The directory is retained as a documented audit boundary for the unchanged G21 production-call-site test. That test examines this command directory and the exported internal packages. It can prove only that the public tree has no settlement production call sites; accepted canonical CI separately audits the complete private runtime. This file adds no Go executable, provider, production flag, or settlement route.

## 中文 — 完整审核版

本脱敏镜像按[公开镜像政策](../../../PUBLIC-MIRROR-POLICY.md)与[代码来源记录](../../../PUBLIC-CODE-PROVENANCE.md)，有意不包含可执行命令入口；私有完整 Runtime 与部署装配不导出。

保留此目录作为未修改的 G21 生产调用点测试的明确审计边界。测试扫描此命令目录与已导出的内部包，只能证明公开文件树不存在结算生产调用点；完整私有 Runtime 由已验收 canonical CI 单独审计。本文件不增加 Go 可执行程序、Provider、生产开关或结算入口。
