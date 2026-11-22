# P05 设计计划：CLI 与模块生态

> 状态：`PROPOSED`（待用户评审；未批准前不得进入本期实施）。

## 1. 背景与目标

把快速起项目、生成模块和 Provider 接入变成可复现流程。

## 2. 本期范围

- CLI new/module add/dev/build/doctor/version。
- `templates/app` 脚手架与唯一的 `examples/biz` 业务示例；不保留 `examples/minimal`。
- 模块骨架生成、Manifest/页面入口/SQL/测试同步校验。
- Provider/Adapter 最小契约与兼容测试。
- 模块开发指南、独立升级边界和发行资产。

## 3. 明确不做

- 远程代码下载和在线模块市场。
- Go `.so` 插件加载。
- 低代码建表、拖拽页面生成器。
- 未经真实需求驱动的万能 Provider 系统。

## 4. 架构与契约决策

- CLI 自身不复制业务规则；生成文件显式可编辑。
- 模块注册仍由组合根显式完成。
- Provider 由消费方定义最小接口、实现方独立包提供。
- 模板与框架版本绑定并有兼容说明。
- 业务项目只依赖版本化 EasyAdmin，不复制 `kernel`、`infra` 和内置 `apps`；核心与业务迁移、配置、前端资源分域。

模板文件树、`easyadmin new` 流程和独立升级机制见 `docs/SCAFFOLD_AND_UPGRADE.md`。

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md`、`docs/CONVENTIONS.md` 和 `docs/SCAFFOLD_AND_UPGRADE.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

## 5. 交付与影响面

- `cmd/easyadmin` CLI 子命令。
- `templates/embed.go`、`templates/app/`、`examples/biz/`。
- `module` 开发工具、manifest lint 和模板测试。
- Provider 示例包和兼容性测试。

## 6. 验收标准

- clean dir 执行 new 后可构建、启动、登录。
- module add 生成后可通过静态检查、测试和构建。
- 模块 manifest 与 API/菜单/权限/ESM/SQL 目录一致。
- Provider 缺失/重复/不兼容时明确失败。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 7. 风险与缓解

- 生成物必须直接可读可编辑。
- 不使用隐式目录扫描注册模块。
- 不运行时下载任意 Go 代码。
- CLI 失败不得留下半成品项目；支持临时目录生成后原子替换。

## 8. 前置条件

- 前置阶段已验收，记录可查。
- 本期 DESIGN_PLAN 已由用户明确确认。
- IMPL_PLAN 已根据当前仓库代码校准文件路径和接口。
- 外部依赖和易库版本已固定；未知语义已列出验证步骤。

## 9. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
