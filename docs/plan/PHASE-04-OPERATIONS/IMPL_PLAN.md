# P04 实施计划：运营能力

> 本文件是 DESIGN_PLAN 的实施指导，不单独送审；按设计定稿后的实际代码基线校准，实施仍需独立开工授权。

## 1. 实施前检查

- [ ] P04 DESIGN_PLAN 已获用户明确确认并标记 APPROVED。
- [ ] P03 已 ACCEPTED，遗留问题已评估。
- [ ] 用户已另行明确授权 P04 开工。
- [ ] 当前仓库代码、目录、依赖版本和测试入口已重新检查。
- [ ] 本期 API/配置/数据/权限/菜单/页面契约影响已列明。
- [ ] 任何超出设计边界的发现已先提交决策，不擅自扩大范围。

## 2. 实施步骤

1. **复用并扩展任务中心**：审计现有 Center、状态类型、并发互斥、取消和终态保留；先定义持久化 Store 接口和状态映射，不复制另一个 Center。
2. **任务持久化**：提交/开始/进度/日志/终态的原子边界；增加 kind+payload handler 注册和 attempt/retry 语义。
3. **恢复协议**：启动将遗留 running 标记 interrupted；仅幂等且明确注册的任务允许重新排队，禁止宣称从闭包中断点恢复。
4. **Scheduler SQL Store**：复用现有 Scheduler/Store 接口；按 DATA_DICT 唯一 sys_schedule schema 实现 SQL Store，处理 task 与 schedule 双写一致性。
5. **可控时钟调度回归**：扫描、漏跑、skip 记账、活动任务防重和 Store 故障测试。
6. 实现本地 Storage 原子写入、内容哈希、元数据、软删除、鉴权下载和安全清理。
7. 实现 Server 管理 Repository/API/UI，凭据加密与 known_hosts 安全验证复用 infra/credential。
8. 明确 Application Log reader/索引来源；没有真实数据来源的日志类型不得开放查询页面。
9. 建立 tasks/schedules/logs/audit/files/servers/info 运维模块与页面。
10. 加入保留期清理、失败告警、资源关闭和 graceful shutdown 测试。

## 3. 主要修改文件/目录

- 现有 infra/tasks、infra/scheduler（复用扩展）；新增持久化适配与 infra/storage。
- 新增 apps/system/tasks、schedules、files、servers、logs 等模块。
- sys_task、sys_task_log、sys_schedule、sys_file、sys_server 表与迁移；字段按本期 DESIGN_PLAN 和 DATA_DICT 对齐。
- 系统运维模块 API/UI。
- `LOGGING_DESIGN.md` 和对应数据字典。

实际文件以实施前仓库核对为准；如与此计划不同，先更新计划并记录原因。

## 4. 测试与验收执行

注入假时钟的 scheduler 测试、任务取消/重试/接管、并发上限、重复提交、进程关闭、文件安全和日志分页测试。
回退：运营模块可配置关闭；保留任务/审计数据备份；存储驱动接口先只提供 local。

- [ ] `gofmt`、`go test ./...`、`go vet ./...` 通过。
- [ ] 适用的 race、集成、浏览器或跨平台测试通过。
- [ ] 失败用例和边界情况有证据，不以“本地可运行”代替验收。

## 5. 文档同步清单

- [ ] README / AGENTS（使用方式或长期工程规则变化时）。
- [ ] ARCHITECTURE / DECISIONS / ROADMAP（架构、决策、阶段变更时）。
- [ ] API / 配置 / 数据字典 / 日志专题文档（契约变化时）。
- [ ] COMPONENTS / FRONTEND / PROJECT_STRUCTURE（模块、组件或目录变化时）。
- [ ] 示例、模板和浏览器验收记录（如适用）。

## 6. 风险控制与回退

- 不在 handler 中直接起无人管理 goroutine。
- scheduler 不运行长任务。
- 不宣称当前多实例安全。
- 不序列化 Go 闭包；重启运行中任务默认 interrupted，重试是新 attempt。
- Task Store 与 Scheduler Store 双写失败必须有 reconciliation 和监控，不能静默忽略。
- 日志中心不得查询不存在的索引数据。

每一项涉及持久化数据的步骤必须明确备份、恢复和失败状态；不得把不可逆迁移描述为可自动回滚。

## 7. 完成定义

- [ ] 设计计划中的验收标准均有测试或操作证据。
- [ ] 测试全部通过；未通过项经明确评审接受并记录。
- [ ] 文档、API、数据字典、配置、菜单、权限和测试同步。
- [ ] 已知限制、兼容性影响和后续事项明确。
- [ ] DESIGN_PLAN 记录实际偏差、验收日期和结论，阶段状态更新为 `ACCEPTED`。

## 8. 实施记录

| 日期 | 变更 | 验证证据 | 偏差/遗留 |
|---|---|---|---|
| 待实施 | - | - | - |
