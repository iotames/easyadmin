# P01 设计计划：运行时内核

> 状态：`PROPOSED`（待用户评审；未批准前不得进入本期实施）。

## 1. 背景与目标

**Objective**：建立透明、可测试的 App/Module 生命周期和 HTTP 主链路，使模块启停、配置和 HTTP 入口具备明确的依赖及故障边界。

**Goals**：形成可验证的 kernel.App / Module / Manifest 契约；模块依赖错误在启动前失败；生命周期失败可逆序清理；easy 库行为有实测证据。

**Strategies**：显式组合根、两阶段注册、net/http 作为业务边界、适配器隔离底层库；不重复建设现有 infra 能力。

**Measures**：模块依赖图正反例测试通过；Declare/Register/Start/Stop 顺序可验证；启动部分失败资源可清理；easyconf/easyserver 适配测试覆盖配置/关闭语义。

## 2. 现状基线与设计前提

- 当前仓库已有 infra/conf、infra/db、infra/credential、infra/tasks、infra/scheduler；P01 应做适配与生命周期整合，不重复造配置/数据库/任务中心。
- cmd/、kernel/、apps/、webui/ 当前尚不存在；P01 的核心交付是 kernel.App、Module 契约与组合根。
- infra/conf.Manager 当前公开 Current/Watch/StartWatcher/Shutdown/Register/Set/List；来源元数据、安全写回与 restart-required 管理字段需在本期核验并明确扩展边界。

## 3. 本期范围

- App、Module、Manifest、Registrar、依赖图与生命周期。
- Declare/Register 两阶段配置注册。
- easyconf/easyserver/log 适配及统一配置、日志、错误、HTTP 门面。
- `/api/meta`、`/healthz`、`/readyz` 和优雅停机。

## 4. 明确不做

- 业务数据库表、账号系统和 RBAC。
- 完整 WebUI 管理台、后台任务中心和调度器。
- 未经验证的底层 easy API 假设。

## 5. 架构与契约决策

- kernel 不依赖具体 infra。
- 标准 net/http Handler 为业务 HTTP 边界。
- 模块依赖校验和启用/禁用行为。
- 配置来源、热更和生命周期语义。
- 中间件顺序和 API 响应结构。

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md` 和 `docs/CONVENTIONS.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

## 6. 交付与影响面

- `kernel/app`、`kernel/module`、注册表和生命周期测试。
- `infra/conf`、`logx`、`apperror`、`httpapi`。
- `infra/internal` 中 easyconf/easyserver/log 适配。
- `cmd/easyadmin` 组合根。
- `docs/ARCHITECTURE.md`、`API.md`、`CONFIGURATION_DESIGN.md`。

## 7. 验收标准

- 重复 ID、缺依赖、依赖环、路由/菜单/权限冲突均启动前失败。
- Declare 早于配置加载，Register 只为启用模块挂载运行资源。
- panic 统一 500，日志有 request_id。
- 启动部分失败能清理已启动资源；Stop 逆序且有超时。
- easy 系列真实适配测试通过。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 9. 数据结构与持久化边界

P01 不创建业务持久化表，也不建立数据库 Runtime Settings。App/Module 注册表和依赖图仅保存在进程内存中；配置来源元数据与可写入策略属于 Bootstrap Config 契约，不借此引入 sys_setting。

## 10. 风险与缓解

- 不做全局 Service Locator；跨模块调用通过构造函数窄接口。
- 不在 Register 中运行 DDL。
- 底层易库缺能力必须记录并提交决策，不得复制整套基础设施。

## 11. 前置条件与阶段依赖

- 前置阶段为 P00；只有 P00 完成终验并标记 ACCEPTED 后，P01 才可开工。
- P01 DESIGN_PLAN 先独立确认；IMPL_PLAN 在设计确认后根据届时仓库代码校准。
- 用户另行明确授权 P01 开工后，才建立 TODO/STEP 并进入实施。
- easyconf/easyserver/log 语义须在 P01 实测；文档推断不能替代实测。

## 12. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
