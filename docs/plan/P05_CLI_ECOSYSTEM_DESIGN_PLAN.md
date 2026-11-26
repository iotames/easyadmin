# P05 设计计划：CLI 与模块生态

> 状态：`PROPOSED`（待用户评审；未批准前不得进入本期实施）。

## 1. 背景与目标

把快速起项目、生成模块和 Provider 接入变成可复现流程。

## 2. 本期范围

本期不新增框架核心业务表。模板元数据与版本检查属于生成物/文件清单；若实现要求持久化生成器运行状态，必须另行设计而不在此阶段临时加表。

- CLI new/module add/dev/build/doctor/version。
- `templates/app` 脚手架与唯一的 `examples/biz` 业务示例；不保留 `examples/minimal`。
- 模块骨架生成、Manifest/页面入口/SQL/测试同步校验。
- 内置模块页面禁用/替换的最小选项契约，避免业务为单个页面替换整模块。
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
- 业务项目只依赖版本化 EasyAdmin，不复制 `kernel`、`infra` 和内置 `apps`；核心与业务结构脚本、配置、前端资源分域。

模板文件树、`easyadmin new` 流程和独立升级机制见 `docs/SCAFFOLD_AND_UPGRADE.md`。

### 4.1 缺口与公共装配接口计划（本期冻结）

P05 必须导出业务项目可用的公共装配面，业务 `cmd` 不得复制框架私有适配器：

| 接口 | 归属/形式 | 说明 |
|---|---|---|
| `easyadmin.New()` | 仓库根包 | 高层门面，返回 App；业务项目通过它完成装配 |
| `App.Use(modules ...module.Module)` | 根包 | 显式登记业务模块和内置模块 |
| `App.WithWebAssets(fs.FS)` | 根包 | 挂载业务 `/assets/app/`，与 core `/assets/` 合并 |
| `App.Run(ctx) error` | 根包 | 统一监听、生命周期与优雅停机 |
| `httpapi.NewRegistrar(*Router, *Catalog) module.Registrar` | `infra/httpapi` | 低层逃生口；导出后自定义 `cmd` 不再复制 Registrar 桥接器 |
| `webui.Assets` | 业务项目 | 业务 `webui/embed.go` 导出，只含 `pages.js` 与业务 assets |

约束：

- 根门面只做装配，不承载业务规则；kernel 不反向依赖根包。
- 业务扩展一律走 `Use`、`WithWebAssets` 或追加注册；不支持 `.so`、运行期目录扫描、覆盖 core 文件。
- `module add` 生成的 `webui/pages.js` 必须与 Manifest Page ID 一致；重复 Page ID 启动失败。
- 内置 app 若提供页面禁用或替换能力，必须通过构造函数选项或独立窄接口暴露，禁止业务侧直接修改内置模块源码。

完整缺口清单与阶段归属见 `docs/SCAFFOLD_AND_UPGRADE.md` §8：本期负责 G3-G6；G1、G2、G7 由 P03 建立接缝，G8 由 P06 验证。

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md`、`docs/CONVENTIONS.md` 和 `docs/SCAFFOLD_AND_UPGRADE.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

## 5. 交付与影响面

- `cmd/easyadmin` CLI 子命令。
- `templates/embed.go`、`templates/app/`、`examples/biz/`。
- `module` 开发工具、manifest lint 和模板测试。
- Provider 示例包和兼容性测试。
- 仓库根包 `easyadmin` 装配门面与 `httpapi.NewRegistrar` 低层适配器。

## 6. 验收标准

- clean dir 执行 new 后可构建、启动、登录。
- module add 生成后可通过静态检查、测试和构建。
- 模块 manifest 与 API/菜单/权限/ESM/SQL 目录一致。
- 生成项目含一个业务页面时，`/assets/app/pages.js` 可加载，core 资源未被复制或改写。
- 业务 `cmd` 只使用公开门面和公开包；删除框架内私有 Registrar 桥接器后仍可构建。
- 内置模块页面禁用或替换可通过公开选项完成，有测试且不改 core。
- Provider 缺失/重复/不兼容时明确失败。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 7. 数据与持久化边界

P05 不新增系统业务表。CLI 配置、模板和生成结果以文件为载体；不在数据库中维护生成器内部状态。

## 8. 风险与缓解

- 生成物必须直接可读可编辑。
- 不使用隐式目录扫描注册模块。
- 不运行时下载任意 Go 代码。
- CLI 失败不得留下半成品项目；支持临时目录生成后原子替换。

## 9. 前置条件

- 前置阶段为 P04；只有 P04 完成终验并标记 ACCEPTED 后，本期才可开工。
- 本期 DESIGN_PLAN 先由用户明确确认；IMPL_PLAN 在设计确认后按当时仓库代码校准，不是设计评审前置条件。
- 用户另行明确授权本期实施后，才建立 TODO/STEP 并进入实施。
- 外部依赖和易库版本以本期实施前实测为准；未知语义必须有验证步骤。

## 10. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
