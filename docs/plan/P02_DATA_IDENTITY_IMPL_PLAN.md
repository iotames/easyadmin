# P02 实施计划：数据与身份安全

> 本文件为实施指导，不设置独立审批状态；设计确认后按实际代码校准，实施仍需独立开工授权。

## 1. 实施前检查

- [ ] P02 DESIGN_PLAN 已获用户明确确认并标记 APPROVED。
- [ ] P01 已 ACCEPTED，遗留风险已评估。
- [ ] 用户另行明确授权 P02 开工。
- [ ] 重新核验 infra/db 与 easydb 的连接、事务、扫描、方言和关闭语义。
- [ ] P02 数据字典、结构脚本、API 和认证流程一致。
- [ ] 数据库备份、结构同步失败恢复和并发初始化策略已确认。

## 2. 实施切片

1. **DB 能力核验**：复用 infra/db 已有连接、Catalog、DDL/Diff 能力；输出现有 API 到目标结构同步契约的差距清单，不先重写包。
2. **Schema Sync**：复用 infra/db 的 EnsureTables/DiffSchema；校验幂等建表、缺表补齐、列差异预览与失败诊断；明确 core 与 app scope。
3. **SQLite 核心 schema**：按本期数据结构创建身份、会话、审计表；其他方言按测试矩阵推进。
4. **Repository 与 Service**：实现用户、角色、权限、会话、审计的 SQL 读写；服务规则不放 HTTP handler。
5. **认证安全**：密码哈希、Cookie 会话、CSRF、限流、一次性初始化与会话撤销。
6. **RBAC**：同步 Manifest 权限码，路由授权由服务端执行；防止最后一个超级管理员被移除。
7. **API 与前端契约准备**：输出端点、状态码、敏感字段与权限表；WebUI 完整页面留 P03。
8. **回归与升级测试**：验证新装、重复结构同步、列差异预览、结构脚本失败恢复、认证与授权安全。

## 3. 预计文件范围

- infra/db/：EnsureTables/DiffSchema 接入、方言差异、缺表与列差异诊断。
- infra/auth/：密码、Session、CSRF、登录限流、授权能力。
- apps/system/：用户、角色、权限、审计用例与 handler。
- sql/sqlite/：核心结构脚本；sql/mysql/、sql/postgres/ 仅在认证通过后提供。
- docs/DATA_DICT.md、API.md、CONFIGURATION_DESIGN.md、LOGGING_DESIGN.md 与集成测试。

当前 infra/db 已存在，文件范围按实际代码组织调整；禁止复制出第二套数据库管理器。

## 4. 测试与验收

- 单测：密码哈希、session token 哈希、权限判定、结构同步幂等、状态保护。
- SQLite 集成：新建 DB、重复执行、并发初始化、缺表补齐与结构脚本失败恢复。
- HTTP 集成：401/403/409/422、CSRF、限流、用户禁用即时登出。
- 安全：响应不泄露 password_hash、session token、DSN 或凭据。
- 命令：go test ./...、go test -race ./...、go vet ./...。

## 5. 风险与回退

- 所有结构变更先备份；破坏性变更必须有恢复演练，不将不可逆 DDL 写作可自动回滚。
- 结构同步失败时保持失败关闭，不启动半初始化应用。
- 初始管理员建立必须事务化；无法保证事务的场景设计明确补偿逻辑。
- 未通过集成测试的方言不列为正式支持。

## 6. 完成定义

- [ ] 每个设计验收标准均有可复现证据。
- [ ] 结构脚本、模型、API、权限、审计与方言 SQL 完全一致。
- [ ] 文档同步，遗留限制和未认证方言明示。
- [ ] DESIGN_PLAN 记录验收日期、偏差、风险与 ACCEPTED 结论。
