# Public Game Server Domains / 公开 Game Server Domain

## English — Primary

This directory is an audited source snapshot, not the complete private Game Server assembly. It publishes project-owned domain code and tests for AI, Character Stats, Combat Rules, Navigation, registries, PostgreSQL Persistence, and secure item Trade.

Run all included tests:

```bash
go test ./...
```

PostgreSQL integration tests require a caller-provided test database configuration and do not contain credentials. Unit tests and in-memory repository tests run without a database.

The snapshot excludes private environment adapters, Legacy source and binaries, Canonical exports, protected fixtures, importer tools, and local absolute paths. See [`../../PUBLIC-CODE-PROVENANCE.md`](../../PUBLIC-CODE-PROVENANCE.md).

---

## 中文 — 完整对应版本

本目录是经过审计的 Source Snapshot，不是完整 Private Game Server Assembly。它公开项目自有的 Domain Code 与 Test，范围包括 AI、Character Stats、Combat Rules、Navigation、Registry、PostgreSQL Persistence 和安全 Item Trade。

运行全部已包含测试：

```bash
go test ./...
```

PostgreSQL Integration Test 需要调用者提供 Test Database Configuration，仓库不包含 Credential。Unit Test 与 In-memory Repository Test 不需要数据库即可运行。

本快照排除 Private Environment Adapter、Legacy Source 与 Binary、Canonical Export、Protected Fixture、Importer Tool 和本地绝对路径。详见 [`../../PUBLIC-CODE-PROVENANCE.md`](../../PUBLIC-CODE-PROVENANCE.md)。
