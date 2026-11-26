# P03 设计计划：WebUI 管理台

> 状态：`ACCEPTED`（2026-11-26 实施、测试与浏览器验收完成）。

## 1. 背景与目标

交付统一布局、菜单、页面懒加载和可复用后台组件，并把「业务页面不改框架源码」的前端接缝一次固化。

## 2. 现状结论（源码证据）

| 当前资产 | 证据 | 本期处理 |
|---|---|---|
| 前端内核与组件已落地 | `webui/assets/core/`（api、boot、data、dom、format、layout、menu、notify、perm、router、store、theme）、`components/` 12 个组件、`styles/` 4 份 CSS；`2026-11-26` 复验 `go build ./webui`、`go vet ./webui` 通过 | 只做接线与契约校正，不重写 |
| 双模式资源已通 | `webui/embed.go`（生产 embed）、`webui/embed_dev.go`（开发读盘）、`webui/index.html` | 直接复用 |
| 路由登记入口存在但未接线 | `webui/assets/core/router.js:13` 导出 `registerEntries`；`boot.js` 只调用 `createRouter`/`start`，全仓库无任何 `registerEntries` 调用，当前无页面可渲染 | 建立 `core/pages.js` 统一注册并接线 |
| 路由来源只有菜单树 | `router.js` 的 `render()` 以 `/api/meta` 的 `menus` 解析路径与 Page ID（`findByPath` → `entries[key]`），不在菜单树中的页面不可达 | 明确「菜单即路由来源」，不建第二份路由表 |
| 菜单排序声明了但未生效 | `kernel/module.PageMeta.Order` → `httpapi.Page.Order`（`catalog.go:15`）全链存在；`buildMenus`（`catalog.go:119`）从未读取 `Order`，且顶层叶子固定排在所有分组之后 | 服务端按 `Order` 统一排序 |
| 页面元信息现状 | `apps/system/module.go:56` 声明 `system.users/roles/audit`；`apps/console/module.go:25` 声明 `console.home` | 校正为冻结页面集（见 5.3） |
| 权限码声明重复 | 权限实际由组合根 `cmd/easyadmin/main.go:113-126` 遍历各模块 Manifest 同步入库并标记 orphaned；同时 `apps/system/module.go:18` 的 `permissionDefs` 又维护一份与自身 Manifest 完全相同的清单，`system.Bootstrap` 再同步一次 | 收敛为 Manifest 单一声明；同步下沉到模块内统一执行，组合根不再重复遍历 |
| 配置中心缺口 | `infra/conf` 的 `Manager.List()` 只提供 key/title/type/defval/current/example，无来源、敏感级别、热更语义；`docs/API.md` §9 已定义 GET/PUT 契约 | 本期补元数据并实现只读 GET |
| 业务资源无挂载点 | `infra/httpapi/handler.go:44` 仅 `Assets fs.FS` 单一资源根；`cmd/easyadmin/main.go:133` 只传 `webui.FS()` | 新增 `AppAssets` 接缝 |
| 业务页面入口无加载点 | 无 `/assets/app/pages.js` 加载逻辑，无 `app.css` 注入 | 本期建立 |
| 脚手架与示例不存在 | 无 `templates/`、`examples/`（P05 范围） | 不在本期 |

结论：内核与组件已具备，缺的是「页面清单 → 路由 → 菜单」的完整链路；且页面 ID、菜单排序、权限码声明、配置元数据四处契约存在缺失、未生效或重复，必须先校正再建页面，否则页面建得再多也接不上菜单与授权。

## 3. 本期范围

- 原生 ESM 前端内核收口：boot、router 页面注册、layout、menu、api、notify、theme、store、perm、format、dom。
- 冻结页面集与页面契约：`dashboard`、`profile`、`system.users`、`system.roles`、`system.audit`、`system.config`（只读）。
- 菜单排序由服务端按 `Order` 统一裁决，前端按返回顺序渲染。
- 权限码单一真源：Manifest 是唯一声明，系统权限表由统一入口同步，角色编辑器可授予；业务项目组合根不再复制同步逻辑。
- 配置中心只读接口与配置项元数据补齐（来源、敏感级别、热更语义）。
- 登录/初始化流程收口；Dashboard、用户、角色、审计、配置、个人资料页面。
- 业务页面接缝：`registerEntries`、`/assets/app/pages.js`、`HandlerOptions.AppAssets`、`/assets/app/styles/app.css`。
- 浏览器内测试面、架构机检和真实浏览器验收。

## 4. 明确不做

- Node/npm/Vite/Webpack 作为默认生产运行依赖；Vue/React 运行时、复杂图表和代码编辑器首屏依赖。
- 配置写入（`PUT /api/system/config/{key}`）与 Runtime Settings：v0.x 只做启动配置，安全写回（原子写入、来源核验、审计）须单独契约评审。
- 浏览器本地持久化用户偏好（`localStorage`）。
- P04 页面：系统日志、任务、调度、文件、系统信息；本期不登记其 entries，也不做空壳页面。
- 内置页面禁用/替换的模块选项（G6，P05 关闭）。

## 5. 架构与契约决策

### 5.1 现行决策（本期冻结）

| 决策点 | 结论 | 出处 |
|---|---|---|
| 前端技术基线 | 原生 ESM + 动态 import，零构建 | 仓库法 §6、`ARCHITECTURE.md` |
| 页面元信息权威 | Manifest `Pages` 是菜单、页面、权限的唯一来源，前端 entries 与其一一对应 | `DECISIONS.md` D05 |
| 前端请求与提示 | 只经 `core/api.js`、`core/notify.js` | 仓库法 §7 |
| 全局布局所有权 | 顶栏、侧栏、菜单、用户区只由 `core/layout.js`/`menu.js` 管理 | 仓库法 §6/§7 |
| 页面 ID 集 | `dashboard`、`profile`、`system.users`、`system.roles`、`system.audit`、`system.config` | 本期冻结（5.3） |
| 菜单排序 | 服务端 `Order` 裁决：顶层叶子与分组统一按 `Order` 升序，分组取组内最小 `Order`，组内按 `Order` 升序、同序按 Page ID 兜底 | 本期冻结（5.3） |
| 个人资料形态 | 路由页面（`Menu` 为空即顶层菜单项），不使用弹层 | 本期冻结（5.3） |
| 配置中心 | 本期只读 GET；写入延后到有安全写回契约的阶段 | `DECISIONS.md` D10、`API.md` §9 |
| 权限码入库 | Manifest 为唯一声明；同步统一在 system 模块执行，删除 `permissionDefs` 重复清单，组合根不再逐个模块遍历 | 统一入口红线 1 |
| 业务页面接缝 | `registerEntries` + `/assets/app/pages.js` + `AppAssets` | `SCAFFOLD_AND_UPGRADE.md` §8 G1/G2 |

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md`、`docs/CONVENTIONS.md` 和 `docs/SCAFFOLD_AND_UPGRADE.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

### 5.2 缺口与扩展接口计划（本期冻结）

本期必须先把“业务页面不改框架源码”的接缝固化，P05 只做生成器和脚手架，不反向改造 core：

| 接口/行为 | 归属 | 本期要求 |
|---|---|---|
| `router.registerEntries(map)` | `webui/assets/core/router.js` | core 页面与业务页面都经此注册；core 页面清单独立模块持有，`boot` 在 `router.start` 前先注册 core 再合并业务；重复 Page ID 启动期失败 |
| `/assets/app/pages.js` | 业务项目 `webui/` | boot 在挂载路由前动态导入；文件不存在或 404 视为无业务页面；加载失败或重复 Page ID 启动失败 |
| `httpapi.HandlerOptions.AppAssets fs.FS` | `infra/httpapi` | 新增字段（只增不改）：`/assets/` 固定来自 core，`/assets/app/` 来自业务；未配置时行为与现状一致 |
| `/assets/app/styles/app.css` | 业务项目 | 存在时按需注入；业务只覆盖 CSS 变量与业务样式，不覆盖 core 文件 |
| Page 禁用/替换 | Manifest/Module | 不支持静默覆盖 core Page ID；需要替换时使用新 Page ID + Manifest 声明；内置页面禁用粒度由 P05 的模块选项定义 |

架构测试至少覆盖：core 不 import 业务页面、`pages.js` 缺失/合并/重复 ID、`AppAssets` 未配置与配置两条路径、页面卸载清理。

完整缺口清单与阶段归属见 `docs/SCAFFOLD_AND_UPGRADE.md` §8：本期负责 G1、G2、G7，G3-G6 由 P05 关闭，G8 由 P06 验证。

### 5.3 页面与数据契约

| Page ID | 路径 | 菜单 | 权限 | 数据来源 |
|---|---|---|---|---|
| `dashboard` | `/` | 顶层，Order 0 | `console.read` | `/api/meta`、`/api/console/ping` |
| `profile` | `/profile` | 顶层，Order 99 | 空（登录即可） | `GET /api/auth/me`、`PUT /api/auth/password` |
| `system.users` | `/system/users` | 系统，Order 10 | `system.user.read` | `/api/system/users` 系列 |
| `system.roles` | `/system/roles` | 系统，Order 20 | `system.role.read` | `/api/system/roles` 系列、`/api/system/permissions` |
| `system.audit` | `/system/audit` | 系统，Order 30 | `system.audit.read` | `/api/system/audit` |
| `system.config` | `/system/config` | 系统，Order 40 | `system.config.read` | `GET /api/system/config`（只读） |

规则：

- **一致性是双向的**：Manifest 的每个 Page ID 必须在 entries 中注册，entries 也不得出现 Manifest 未声明的 Page ID；任一方向缺失在测试期失败。
- **页面 ID 校正**：`console.home` 更名为 `dashboard`，路径由 `/console` 改为 `/`；P03 是首个消费页面元信息的阶段，无外部依赖，校正成本最低。
- **菜单即路由来源**：router 只解析 `/api/meta` 菜单树中的路径；因此需要可访问的页面必须进 Manifest。本期不引入第二份前端路由表，避免菜单与路由各说一套。
- **排序单源**：顺序由服务端 `buildMenus` 裁决，前端不二次排序；`Order` 必须真正参与排序，否则“控制台”会被排到“系统”分组之后。
- **权限码单源**：页面与权限一律来自 Manifest；系统权限表由统一入口同步，模块不得再维护第二份权限清单，业务项目也不必复制组合根同步逻辑。
- **配置中心只读**：`GET /api/system/config` 按 `docs/API.md` §9 字段输出分组元数据，含 `source`（default/file/environment/command_line）、`secret`、`hot_reload`；本期 `writable` 恒为 `false`，不实现 PUT。敏感项只返回是否已配置，不回显值；`EASYADMIN_ADMIN_PASSWORD` 必须显式声明为敏感项并有测试断言其值不出现在响应中。
- **配置元数据必须显式声明**：配置注册需声明敏感级别与热更语义（红线 8）；来源类别由加载层判定，不靠键名猜测。

## 6. 交付与影响面

- `webui/assets/core/`、`components/`、`pages/`、`styles/`、`_test/`；新增 core 页面注册表。
- `infra/httpapi`：菜单排序、权限清单读取、`AppAssets` 挂载。
- `infra/conf`：配置项元数据（来源、敏感级别、热更语义）。
- `apps/system`：页面与权限声明、权限表同步、`GET /api/system/config`。
- `apps/console`：页面 ID 与路径校正。
- `docs/FRONTEND.md`、`API.md`、组件清单、`CONVENTIONS.md` 页面契约和验收截图。

## 7. 验收标准

- 初始化 → 登录 → Dashboard → 用户/角色/审计/配置查看完整可用，个人资料可打开并改密。
- 页面切换网络请求证明按需加载，切换前不加载目标页面 ESM。
- 页面卸载无遗留事件、计时器和未完成请求。
- 菜单权限隐藏与后端授权均正确；非授权直接访问 API 仍被拒绝。
- `/api/meta` 菜单顺序为：控制台 → 系统（用户、角色、审计、配置）→ 个人资料，由测试固定。
- Manifest 声明的全部权限码出现在 `/api/system/permissions`，`console.read` 可被角色授予并在页面生效；删除 `permissionDefs` 后权限集不变（回归测试）。
- 页面 ID 双向一致：Manifest ⊆ entries 且 entries ⊆ Manifest，测试覆盖缺失、多余、重复三种失败。
- 配置页：`GET /api/system/config` 返回来源/敏感/热更字段；敏感项值不回显；无 PUT 路由。
- `AppAssets` 未配置时行为与 P02 一致；配置后 `/assets/app/pages.js` 可加载，且不覆盖 `/assets/` core 资源。
- 浏览器测试全绿，关键流程人工实测。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 8. 数据与持久化边界

P03 不新增业务持久化表。唯一持久化影响是 `sys_permission` 的同步执行位置变化，同步内容（Manifest 权限）不变，表结构不变，只增不删（历史孤儿权限行不自动清理）。配置页只读：不写配置文件、不写数据库。页面临时状态按页面生命周期管理；跨会话偏好不在本期写入浏览器存储。

## 9. 质量自评

| 维度 | 自评结论 | 评审重点 |
|---|---|---|
| 工程化最佳实践 | 页面、菜单、权限、配置各自单源，契约有双向一致性测试 | 菜单顺序与权限同步是否真由服务端裁决 |
| 用户体验 | 菜单顺序稳定、深链可用、敏感配置不外泄、改密有明确反馈 | 配置页只读是否被误当“管理”能力 |
| 改动量与复杂度 | 只校正页面、菜单、权限、配置四处契约，不引入新路由框架、不建第二份配置读取 | 配置元数据扩展是否波及过多注册点 |
| 架构统一性 | core 只管布局与内核，页面按模块目录聚合，业务接缝与 P05 对齐 | 是否把业务能力塞进 core |

## 10. 风险与缓解

- 排序变更会改变 `/api/meta` 输出顺序：以固定期望顺序的测试兜底，避免“看起来对了”。
- 权限同步位置变更若漏做，权限表会停在旧集合：以启动期集成测试断言权限集完整，历史孤儿行只增不删、不自动清理。
- 配置注册元数据扩展会触及所有注册点：编译期强制，漏改即构建失败，不留运行期隐式默认。
- 页面不得碰顶栏/侧栏等全局 DOM；不允许裸 fetch、alert、自造 toast、未转义 innerHTML。
- 页面入口必须具备卸载能力；前端零构建策略若要改变，必须另做架构评审。

## 11. 前置条件

- 前置阶段为 P02；只有 P02 完成终验并标记 ACCEPTED 后，本期才可开工。
- 本期 DESIGN_PLAN 先由用户明确确认；IMPL_PLAN 在设计确认后按当时仓库代码校准，不是设计评审前置条件。
- 用户另行明确授权本期实施后，才建立 TODO/STEP 并进入实施。
- 外部依赖和易库版本以本期实施前实测为准；未知语义必须有验证步骤。

## 12. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
| 2026-11-25 | 用户 | 授权提前落地 WebUI 基础资产（core/components/styles/_test），不改变本期整体状态 | P02 前置、登录/页面联调与本期验收仍未完成；本文件仍为 `PROPOSED` |
| 2026-11-26 | 智能体自审 | 契约校正：页面 ID 集、菜单排序、权限码单源、配置只读边界；文档定稿待用户确认 | 实施仍冻结，需用户确认后方可开工 |
| 2026-11-26 | 用户 | APPROVED（含开工、测试、验收、提交并推送授权） | 契约维持自审定稿；按本文件验收标准执行 |
| 2026-11-26 | 智能体 | ACCEPTED（本地验收全绿） | 无；归档待用户确认 |

## 13. 验收结论与已知边界

验收日期：2026-11-26。结论：`ACCEPTED`。

### 13.1 验收证据

- `gofmt -l` 无输出；`go vet ./...`、`go test ./...` 通过；`go build -tags dev` 与生产构建均通过。
- 浏览器测试面 `_test/index.html` 51/51 通过；console 仅两条 HEAD 资源探测的预期 404，network errors 为空。
- 生产构建 `/_test/index.html` 返回 404，`/` 与 `/assets/core/pages.js` 返回 200，证明生产 embed 不含测试面。
- 真实 Edge 浏览器全流程：初始化 admin/管理员/password123 → 登录 → 仪表盘 → 创建角色 → 创建用户并分配角色 → 编辑用户 → 禁用/启用 → 重置密码 → 编辑角色权限 → 删除临时角色 → 审计 → 配置（敏感项仅显示“已配置”）→ 个人资料改密 → 登出 → 新密码登录 → 普通用户登录。
- 权限边界：普通用户菜单仅含 `/system/users`、`/system/audit`、`/profile`；直接请求 `GET /api/system/config` 返回 403。
- 按需加载：登录后仅加载 `dashboard/index.js`；进入角色、用户页后才分别加载对应 ESM。
- 验收截图：`docs/acceptance/p03/01-setup.png` 至 `12-operator-view.png`。

### 13.2 验收中发现并修复的缺陷

1. 页头动作按钮无响应：`ctx.setActions` 渲染在 `#page-head`，页面却用 `bindActions(ctx.el, ...)` 只监听 `#page-body`。修复为 `ctx.bindActions(map)` 同时绑定页头与页面主体，用户/角色页改用该入口。
2. 前端 `ctx.api` 只注入 `{get, post}`，页面调用 `put` 时报 `ctx.api.put is not a function`。修复为 boot 注入完整 `api` 对象。

### 13.3 已知边界

- 配置在线写入未实现，`PUT /api/system/config/{key}` 当前返回 404；P03 只读。
- P04 页面（logs、tasks、schedules）未登记 entry，未交付前不可达。
- 业务页面接缝已交付（`httpapi.HandlerOptions.AppAssets` + `/assets/app/pages.js` 合并）；脚手架与自动升级属 P05。
- 单实例部署。SQLite 在多个写请求真正并发时可能因锁返回 500；本轮验收脚本快速连点曾复现一次，串行操作稳定。生产建议启用 `busy_timeout`/WAL，属基础设施演进项，不在 P03 范围。
- 审计页只读查询，不提供导出。
- 验收截图为 headless Edge、1280×720。
- 归档动作（DESIGN_PLAN 迁 `docs/done/`、清理执行期文档）待用户确认后执行。
