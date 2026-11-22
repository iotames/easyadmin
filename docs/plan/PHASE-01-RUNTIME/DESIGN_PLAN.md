# P01 设计计划：运行时内核

> 状态：`PROPOSED`（待用户评审；未批准前不得进入本期实施）。

## 1. 背景与目标

建立透明、可测试的 App/Module 生命周期和 HTTP 主链路。

## 2. 本期范围

- App、Module、Manifest、Registrar、依赖图与生命周期。
- Declare/Register 两阶段配置注册。
- easyconf/easyserver/log 适配及统一配置、日志、错误、HTTP 门面。
- `/api/meta`、`/healthz`、`/readyz` 和优雅停机。

## 3. 明确不做

- 业务数据库表、账号系统和 RBAC。
- 完整 WebUI 管理台、后台任务中心和调度器。
- 未经验证的底层 easy API 假设。

## 4. 架构与契约决策

- kernel 不依赖具体 infra。
- 标准 net/http Handler 为业务 HTTP 边界。
- 模块依赖校验和启用/禁用行为。
- 配置来源、热更和生命周期语义。
- 中间件顺序和 API 响应结构。

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md` 和 `docs/CONVENTIONS.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

## 5. 交付与影响面

- `kernel/app`、`kernel/module`、注册表和生命周期测试。
- `infra/conf`、`logx`、`apperror`、`httpapi`。
- `infra/internal` 中 easyconf/easyserver/log 适配。
- `cmd/easyadmin` 组合根。
- `docs/ARCHITECTURE.md`、`API.md`、`CONFIGURATION_DESIGN.md`。

## 6. 验收标准

- 重复 ID、缺依赖、依赖环、路由/菜单/权限冲突均启动前失败。
- Declare 早于配置加载，Register 只为启用模块挂载运行资源。
- panic 统一 500，日志有 request_id。
- 启动部分失败能清理已启动资源；Stop 逆序且有超时。
- easy 系列真实适配测试通过。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 7. 风险与缓解

- 不做全局 Service Locator；跨模块调用通过构造函数窄接口。
- 不在 Register 中运行 DDL。
- 底层易库缺能力必须记录并提交决策，不得复制整套基础设施。

## 8. 前置条件

- 前置阶段已验收，记录可查。
- 本期 DESIGN_PLAN 已由用户明确确认。
- IMPL_PLAN 已根据当前仓库代码校准文件路径和接口。
- 外部依赖和易库版本已固定；未知语义已列出验证步骤。

## 9. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
