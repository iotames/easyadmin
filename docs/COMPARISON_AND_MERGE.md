# 两套 EasyAdmin 规划对照与综合结论

## 1. 输入材料

- **用户版本**：本轮上传的 `easyadmin.zip`，包含 README、AGENTS、ARCHITECTURE、DECISIONS、ROADMAP、COMPONENTS、CONVENTIONS、FRONTEND、API、DATA_DICT。
- **上一版**：本轮工作区中先前生成的 `easyadmin-planning.zip`，包含总体架构、RockSys 资产审查、配置/日志/数据/API/WebUI 专题文档，以及 P0–P8 共 9 期的成对计划。

两套材料均为设计稿，不代表已实现代码。

## 2. 总体判断

用户版本在具体产品设计和工程细节上更完整：原生 ESM 零构建、模块 Manifest、Declare/Register 两阶段、数据字典、API 端点和 AGENTS 工作清单更贴近 EasyAdmin 的目标。上一版更强的部分是底层依赖核验边界、适配层的独立性、日志语义拆分、配置来源/持久化风险，以及每期 `DESIGN_PLAN.md` / `IMPL_PLAN.md` 的治理方式。

**综合策略：用户版本作为主干，上一版作为架构审查和工程治理补充。** 不简单合并两个目录，不保留两套互相冲突的架构。

## 3. 核心取舍

| 主题 | 用户版本 | 上一版 | 综合决策 |
|---|---|---|---|
| 前端 | 原生 ESM、零构建 | Vue/TS/Vite | 保留原生 ESM；不用前端框架作为运行前提 |
| 分期 | 7 期 | 9 期 | 保留 7 期，因数据身份与 UI 的依赖顺序更自然；所有阶段补成对计划 |
| 模块契约 | Declare/Register、Manifest | Descriptor/Registrar、生命周期 | 合并两阶段声明和依赖图；Register 不直接执行 DDL |
| 跨模块调用 | Context.Service 字符串注册示意 | 反对 Service Locator | 采用消费方窄接口 + 构造函数注入 + cmd 显式接线 |
| 分层 | kernel/infra 依赖关系不够明确 | adapter boundary 更突出 | kernel 不依赖具体 infra；cmd 负责组合 |
| 配置 | 注册、默认值快照、热更和 `.env` 写回 | Bootstrap/Runtime Settings 分离 | 保留注册制，强化来源追踪；v0.x 只做启动配置，运行设置后置 |
| API | 使用 `/api/`、不设路径版本 | 使用路径版本化写法和统一错误契约 | 采用 `/api/`、不设路径版本；request_id 只在 `meta`；接口向下兼容 |
| 数据 | 具体数据字典、三方言 | 迁移/兼容性风险 | 以用户数据字典为底稿，增加结构同步与驱动认证门槛 |
| 日志 | 基础日志规则 | Application/Access/Audit/Task 分离 | 合并，避免日志中心无数据来源 |
| 参考项目 | RockSys、EmailPlugin、KitBus | RockSys 源码审查更明确 | 当前工作区三个项目源码均可用，按实际代码核验后标注证据 |

## 4. 合并后修正的内部不一致

1. **配置注册时机**：模块配置统一在 `Declare` 注册；配置加载完成后 `Register` 才挂载运行资源。移除“在 Register 阶段注册所有配置”的歧义。
2. **依赖方向**：`kernel` 不 import 具体 `infra` 包；平台服务由组合根传入，避免内核和平台形成互相依赖。
3. **跨模块访问**：字符串 Service Locator 不是默认方式。消费者定义小接口，组合根显式注入。
4. **结构同步**：模块只登记 `TableSpec` 结构清单；统一结构同步器幂等补齐，Register 阶段不直接运行 DDL。
5. **API 响应**：成功和失败响应都使用顶层 `meta.request_id`；不在 error 内重复。
6. **API 路径**：统一为 `/api/`，不设路径版本；健康探针保留 `/healthz`、`/readyz`。接口遵循向下兼容，只增不改。
7. **中间件顺序**：RequestID 最外层，确保 Recovery/AccessLog 能关联请求 ID；顺序由集成测试固定。
8. **前端映射**：Manifest 提供菜单、权限和 Page ID；浏览器 ESM loader 映射仍是静态代码登记点，用架构测试保证一一对应，不宣称运行时自动发现。
9. **配置持久化**：环境变量覆盖 `.env` 时，写 `.env` 不能视为有效值已持久化；必须展示来源和重启行为。
10. **日志中心**：只有定义了明确的文件 reader 或索引存储，才能提供 Application Log 查询页面。
11. **参考资产证据**：当前工作区已具备 RockSys、EmailPlugin、KitBus 源码。RockSys 已核对配置、数据、管理 API、任务与 WebUI 分层；EmailPlugin 已核对持久任务、调度、Schema、存储与 HTTP 骨架；KitBus 已核对 `infra/internal` 分层、SQL 外置与前端菜单边界。具体底层行为仍以锁定版本测试为准。

## 5. 最终七期结构

P0 地基与工程规范；P1 运行时内核；P2 数据与身份安全；P3 WebUI 管理台；P4 运营能力；P5 CLI 与模块生态；P6 稳定与正式发行。

这样保留用户版本的纵向闭环和合理依赖，同时吸收上一版的工程化成对计划。每一期都有目标、范围、排除项、文件影响、测试、验收和回退要求。

## 6. 待评审的架构决策

已确认：API 统一使用 `/api/`，不设路径版本，接口向下兼容、只增不改。

其余优先确认：

1. 原生 ESM 零构建作为正式前端路线；
2. kernel 与 infra 的边界及组合根装配方式；
3. `Declare/Register` 两阶段模块契约；
4. SQL-first 和默认 SQLite；
5. 启动配置/运行设置分离及热更持久化语义；
6. 单实例优先、多实例后置。

除已确认的 API 约束外，其余方案仍是待评审设计，不代表用户已批准，也没有开始实现代码。
