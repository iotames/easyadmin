# P01 设计计划：运行时内核

> 状态：`ACCEPTED`（2026-11-26 实施并验收完成）。

## 1. 背景与目标

**Objective**：建立透明、可测试的 App/Module 生命周期和 HTTP 主链路，使模块启停、配置和 HTTP 入口具备明确的依赖及故障边界。

**Goals**：形成可验证的 kernel.App / Module / Manifest 契约；模块依赖错误在启动前失败；生命周期失败可逆序清理；easy 库行为有实测证据。

**Strategies**：显式组合根、两阶段注册、net/http 作为业务边界、适配器隔离底层库；在 P00 已交付的内核骨架上扩展，不重写已验证能力。

**Measures**：模块依赖图正反例测试通过；Declare/Register/Start/Stop 顺序可验证；启动部分失败资源可清理；easyconf/easyserver 适配测试覆盖配置/关闭语义；重复路由、重复页面 ID、重复权限码在监听前失败。

## 2. 现状基线与设计前提（源码证据）

P00 已交付并验收（`docs/done/P00_FOUNDATION_DESIGN_PLAN.md`，ACCEPTED），P01 必须在其上增量扩展：

| 已存在资产 | 证据 | P01 处理方式 |
|---|---|---|
| 模块契约 | `kernel/module/module.go`：`Manifest{ID,Version,Description,DependsOn}`、`Module`、`ConfigRegistrar`、`Registrar`、`Lifecycle` | 扩展 Manifest 元信息与 Registrar 登记方法，不重写包结构 |
| App 生命周期 | `kernel/app/app.go`：重复 ID/缺依赖/环校验、稳定拓扑序、启动失败逆序回滚、`stopReverse` 聚合错误 | 复用；仅增加"禁用模块"过滤，不改拓扑与回滚算法 |
| 组合根 | `cmd/easyadmin/main.go`：`run()`、已装配 `conf.Load`→`Declare`→`Register`→`Start` | 保留骨架，替换 P00 临时 HTTP 面为 infra/httpapi |
| 临时 HTTP 面 | `cmd/easyadmin/http.go`：`/api/meta`、`/healthz`、`/readyz`、`requestID`、`writeOK/writeError`、`spaHandler` | 整体下沉到 `infra/httpapi` 并删除该文件 |
| 前端内核 | `webui/` 原生 ESM；`boot.js` 已按 `features.auth` 与 `initialized` 分支 | 不改前端契约；P01 只保证 `/api/meta` 字段稳定 |
| 架构守卫 | `internal/archtest/arch_test.go`：分层 import 与页面裸 fetch 正反例 | 继续作为红线守卫，P01 新增包必须满足其规则 |
| 配置与数据库 | `infra/conf`（Load/Register/Current/Watch/Shutdown）、`infra/db`（Open/EnsureTables/DiffSchema） | 直接复用，不造第二套配置/数据库入口 |

尚不存在、P01 需新建：`infra/logx`、`infra/apperror`、`infra/httpapi`；`apps/` 仍为空，P01 通过一个最小内置模块验证寄存器。

## 3. 本期范围

- 扩展 App、Module、Manifest、Registrar、依赖图与生命周期（禁用模块过滤）。
- Declare/Register 两阶段配置注册。
- `infra/logx` 日志门面、`infra/apperror` 稳定错误、`infra/httpapi` 路由/中间件/响应/绑定。
- easyconf 适配（复用 `infra/conf`）、配置声明时序、只读快照。
- 统一中间件链固定顺序：RequestID → Recover → AccessLog。
- `/api/meta`、`/healthz`、`/readyz` 与优雅停机。
- 最小内置模块（仅登记页面/权限元信息与一条示例路由，不含业务表）。

## 4. 明确不做

- 业务数据库表、账号系统和 RBAC 的持久化与判定（P02）。
- 完整 WebUI 管理台、后台任务中心和调度器。
- 未经验证的底层 easy API 假设；不新建配置/数据库第二入口。

## 5. 设计与契约（设计对象级）

### 5.1 Manifest 扩展（菜单/页面/权限权威来源，D05）

`kernel/module` 在既有字段上增加声明式元信息，保持 `ID/Version/Description/DependsOn` 不变：

- `Pages []PageMeta`：`PageMeta{ID, Title, Path, Menu, Order, Perm}`。`ID` 全局唯一且与前端 entry 一一对应；`Menu` 为空表示不进菜单；`Perm` 为空表示登录即可访问。
- `Permissions []PermissionMeta`：`PermissionMeta{Code, Name}`，`Code` 形如 `system.user.read`，全局唯一。

菜单树由"带 Menu 的 Pages"聚合生成，不单独维护第三份清单。

### 5.2 Registrar 扩展与冲突检测

`Registrar` 由组合根注入，P01 明确其方法与失败语义：

- `Handle(pattern string, h http.Handler) error`：登记路由；重复 pattern 返回错误（监听前失败）。
- `RegisterPage(PageMeta) error`：登记页面；重复 Page ID 返回错误。
- `RegisterPermission(PermissionMeta) error`：登记权限码；重复 Code 返回错误。

模块不直接操作底层 Server；`infra/httpapi` 提供 Registrar 实现。

### 5.3 App 生命周期与禁用模块

- `Options` 增加 `Disabled []string`。被禁用模块不参与 Declare/Register/Start；若某启用模块依赖被禁用模块，`New` 直接失败并指出依赖链。
- 保持 P00 已验证的稳定拓扑序与启动失败逆序回滚；禁用只是从执行集合中剔除，不改变排序算法。

### 5.4 HTTP 主链路（infra/httpapi 唯一入口）

- 响应外壳：成功 `{ok:true,data,meta}`、失败 `{ok:false,error,meta}`，`request_id` 仅在 `meta.request_id`。
- 中间件固定顺序：RequestID → Recover → AccessLog；panic 统一 500 且日志带 request_id。
- 请求绑定 `Bind`：JSON 格式错误、未知字段、类型转换失败返回 400；绑定成功后字段/跨字段校验失败返回 422，并按 `API.md` 返回 `error.details.fields`。二者职责分离，不在`Bind` 内做领域规则。
- 探针：`/healthz` 进程存活即 200；`/readyz` 依赖就绪才 200，否则 503；P01 的 checks 先接入 `app`，数据库检查随 P02。
- `/api/meta`：返回 `app{name,version}`、`initialized`、`user`、`permissions`、`menus`、`features`、`csrf_token`；P01 由已登记 Manifest 组装 permissions/menus，登录态与 CSRF 值由 P02 补齐（P01 期间 `features.auth=false`）。
- 静态资源：SPA 回退规则从 P00 `spaHandler` 原样迁入；生产 `go:embed`、开发 `-tags dev` 磁盘读取保持不变。

### 5.5 配置、日志与错误门面

- 配置复用 `infra/conf`：`Load` 后执行模块 `Declare`（登记模块配置元数据并触发重载），再 `Register`；模块读取配置一律经 Manager，不读环境变量。
- `infra/logx`：结构化日志门面，基于标准 `log/slog` 语义，底层可对接 `easyserver/log`；统一输出 request_id 等字段。
- `infra/apperror`：稳定 `Code` + 用户可读 `Message` + 可选 `Details`，并提供到 HTTP 状态码的映射（401/403/404/409/422/429/500）。

## 6. 交付与影响面

- `kernel/module`、`kernel/app`（增量扩展 + 测试）。
- 新增 `infra/logx`、`infra/apperror`、`infra/httpapi`。
- `cmd/easyadmin` 组合根改造（删除临时 `http.go`）。
- `docs/ARCHITECTURE.md`、`API.md`、`CONFIGURATION_DESIGN.md` 同步（契约变化时）。

## 7. 验收标准

- 重复 ID、缺依赖、依赖环、路由/页面/权限冲突均启动前失败并准确定位。
- Declare 早于 Register；禁用模块不登记运行资源；启用模块依赖禁用模块时启动前失败。
- 中间件顺序、request_id 透传、panic 统一 500 有集成测试。
- 字段绑定错误 400、校验失败 422 且字段结构符合 API.md。
- 启动部分失败能清理已启动资源；Stop 逆序且有超时。
- 复用 infra/conf、infra/db 的适配测试通过，无第二套入口。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 8. 现行有效决策表

| 决策点 | 当前有效结论 | 出处 |
|---|---|---|
| API 兼容 | API 不设路径版本，既有端点向下兼容 | DECISIONS.md D13，用户已确认 |
| 菜单/权限权威 | 后端 Manifest 是菜单/页面/权限元信息权威来源，前端按 Page ID 映射入口 | DECISIONS.md D05 |
| 显式装配 | 模块/路由显式登记，禁止 init/扫描 | DECISIONS.md D01 |

本期其余模块生命周期细节、HTTP adapter 与禁用策略为待评审提案，确认后成为本期约束。

## 9. 数据结构与持久化边界

P01 不创建业务持久化表，也不建立数据库 Runtime Settings。App/Module 注册表、页面/权限目录和依赖图仅保存在进程内存中；配置来源元数据属于 Bootstrap Config 契约，不借此引入 sys_setting。

## 10. 风险与缓解

- 不做全局 Service Locator；跨模块调用通过构造函数窄接口。
- 不在 Register 中运行 DDL。
- 扩展 Manifest/Registrar 时保持 P00 包结构，避免破坏已验证的生命周期测试。
- 底层 easy 库能力不足必须记录并提交决策，不得复制整套基础设施。

## 11. 质量自评

| 维度 | 自评结论 | 评审重点 |
|---|---|---|
| 工程化最佳实践 | 显式生命周期、依赖图、失败逆序清理，冲突前移到监听前 | 是否覆盖部分启动失败、禁用依赖和冲突检测 |
| 用户体验 | 区分健康与就绪，故障信息可诊断 | 状态是否能准确反映依赖故障 |
| 改动量与复杂度 | 复用 P00 内核与现有 infra，只做增量扩展 | 是否重写已验证能力、是否引入多余抽象 |
| 架构统一性 | httpapi/logx/apperror 单一入口，cmd 显式装配 | 包依赖是否有自动化守卫 |

以上为设计自评；easy 系列语义仍须 P01 实测后定稿。

## 12. 前置条件与阶段依赖

- 前置阶段为 P00；P00 已完成终验并标记 ACCEPTED（2026-11-26）。
- P01 DESIGN_PLAN 先独立确认；IMPL_PLAN 在设计确认后按届时仓库代码校准。
- 用户另行明确授权 P01 开工后，才建立 TODO/STEP 并进入实施。
- easyconf/easyserver 语义以 P01 实测为准；文档推断不能替代实测。

## 13. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 2026-11-26 | 用户 | APPROVED（含开工授权） | 已按 P00 现状对齐基线；补齐 Manifest/Registrar/httpapi 契约；删除 IMPL 进度回写 |

## 14. 验收结论与已知边界

**结论**：PASS（2026-11-26）。

**验收证据（真实执行）**：
- `go test ./...` 全绿；`go vet ./...` 无输出；`go test -race ./kernel/... ./infra/httpapi/... ./apps/console/...` 通过。
- `internal/archtest` 分层与裸 fetch 守卫通过。
- HTTP 实测：`GET /api/meta` 返回 `app{name,version}`、`initialized:false`、`permissions:["console.read"]`、`menus` 含控制台页、`features.auth:false`；`GET /healthz` 返回 `{"ok":true,"status":"alive"}`；`GET /api/console/ping` 返回 `{"ok":true,"data":{"pong":true}}`；`meta.request_id` 存在。
- 可视化浏览器（Edge）实看：页面标题 EasyAdmin，渲染"系统尚未初始化 / 当前版本只交付最小运行骨架 / 服务端已正常启动"占位页，无 console 错误、无网络错误。

**交付物**：`kernel/module`（Manifest 扩展 Pages/Permissions、Registrar 冲突检测）、`kernel/app`（Disabled 过滤、Manifest 元信息登记、拓扑/回滚复用）、`infra/logx`、`infra/apperror`、`infra/httpapi`（路由/中间件/响应/绑定/meta/探针/spa）、`apps/console` 最小模块、`cmd/easyadmin` 改用 httpapi 并删除临时 http.go。

**已知边界（本期不做，留给 P02/P03）**：
- `features.auth=false`：P01 不提供认证，前端显示占位页；P02 完成时置 true。
- 无业务表、无登录/RBAC、无数据库连接；`/readyz` 仅检查 app。
- `apps/console` 仅示例元信息与一条 ping 路由，不含业务逻辑。

**偏差记录**：计划中的"模块自行调用 RegisterPage"改为由 App 在 Register 时依据 Manifest 统一登记（D05 单一权威来源），减少重复登记与发散；临时 `cmd/easyadmin/http.go` 已删除。
