# Public Mirror Policy / 公开镜像政策

## English — Primary

### Repository model

Fractal Legend uses two repositories:

- **Private canonical repository:** complete internal development history, migration references, restricted evidence, and private acceptance material.
- **Sanitized public mirror:** public-safe project source, tests, documentation, technical screenshots, and approved marketing assets.

The public mirror has independent Git history. It must never be produced by cloning the private repository and deleting files afterward.

### Sync workflow

Every public sync follows this sequence:

1. Complete and accept work in the private canonical repository.
2. Select candidate files through an explicit allowlist.
3. Verify ownership and provenance for every selected directory.
4. Sanitize private paths and private repository links without changing technical results.
5. Run secret, privacy, restricted-source, binary, archive, large-file, dependency, license, and link audits.
6. Use a GitHub noreply identity for public commits.
7. Push only after every mandatory public gate passes.
8. Verify repository, README, assets, and documents anonymously.

### Allowed content

- Fractal Legend-owned source and tests
- Synthetic fixtures
- Public-safe schemas and configuration examples
- Bilingual architecture records, ADRs, Devlogs, and curated Interaction Records
- Approved technical screenshots and marketing artwork
- Public-safe dependency manifests

### Prohibited content

- Secrets, credentials, wallet keys, mnemonics, or production `.env` files
- Personal email addresses, local user directories, private hosts, or internal deployment paths
- Legacy seller source, proprietary third-party source, binaries, databases, or paid archives
- Protected fixtures, private databases, player data, production configuration, and backup archives
- Full private conversations or unsanitized internal audit material

### History and milestone integrity

The mirror began after G10. Earlier milestones are represented through bilingual retrospective documents and private archive references. The public commit history must never be described as the original G1–G10 chronology.

Future milestones, including G11, may be developed and accepted privately first. Only project-owned code, tests, ADRs, Devlogs, curated Interaction Records, and other material that passes this policy may be exported.

---

## 中文 — 完整对应版本

### 仓库模型

Fractal Legend 使用两个 Repository：

- **Private Canonical Repository：**保存完整内部开发历史、迁移参考、受限制证据和私人验收材料。
- **Sanitized Public Mirror：**保存可安全公开的项目 Source、Test、Documentation、Technical Screenshot 和已批准 Marketing Asset。

Public Mirror 使用独立 Git History。禁止通过 Clone Private Repository 后再删除文件的方式生成公开镜像。

### 同步流程

每次 Public Sync 都必须依次执行：

1. 在 Private Canonical Repository 完成并验收工作。
2. 使用明确 Allowlist 选择候选文件。
3. 验证每个候选目录的 Ownership 与 Provenance。
4. 在不改变技术结果的前提下，清理 Private Path 与 Private Repository Link。
5. 执行 Secret、Privacy、Restricted-source、Binary、Archive、Large-file、Dependency、License 和 Link Audit。
6. Public Commit 只使用 GitHub noreply Identity。
7. 全部强制 Public Gate 通过后才能 Push。
8. Push 后匿名验证 Repository、README、Asset 与 Document。

### 允许内容

- Fractal Legend 自有 Source 与 Test
- Synthetic Fixture
- Public-safe Schema 与 Configuration Example
- 双语 Architecture Record、ADR、Devlog 和整理后的 Interaction Record
- 已批准 Technical Screenshot 与 Marketing Artwork
- Public-safe Dependency Manifest

### 禁止内容

- Secret、Credential、Wallet Key、Mnemonic 或 Production `.env` File
- 个人 Email、本地用户目录、Private Host 或内部 Deployment Path
- Legacy Seller Source、Proprietary Third-party Source、Binary、Database 或 Paid Archive
- Protected Fixture、Private Database、Player Data、Production Configuration 与 Backup Archive
- 完整私人对话或未脱敏 Internal Audit Material

### 历史与里程碑完整性

Public Mirror 在 G10 之后建立。更早 Milestone 通过双语 Retrospective Document 和 Private Archive Reference 表达。不得把 Public Commit History 描述成原始 G1–G10 Chronology。

未来包括 G11 在内的 Milestone，可以先在 Private Canonical Repository 开发和验收。只有通过本 Policy 的项目自有 Code、Test、ADR、Devlog、Curated Interaction Record 与其他材料才允许导出。
