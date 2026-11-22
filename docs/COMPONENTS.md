# 组件清单与提取来源

本文回答三个问题：从参考项目里提取了什么、EasyAdmin 把它放在哪、边界怎么划。

**证据说明**：当前工作区包含 RockSys、EmailPlugin、KitBus 三个项目的源码。本文的能力归纳已按实际代码结构与公开 API 核对：RockSys 的 `internal/conf`、`internal/db`、`internal/adminapi`、`internal/taskcenter`、`webui`，EmailPlugin 的 `internal/httpapi`、`internal/taskcenter`、`internal/schedule`、`internal/schema`、`internal/storage`，KitBus 的 `infra/` 与 `webui/app` 均已检查。源码存在与接口形态可证，但并发、故障恢复和边界行为仍需在锁定版本后用测试认证。

## 1. 提取原则

只提取**与业务无关、被至少两个项目重复实现、且有明确统一入口**的能力。

不提取：

- 只服务单一业务的能力（代理转发、容器遥控、邮件同步）；
- 实现很重但收益不明确的抽象；
- 与特定部署环境强绑定的逻辑；
- 参考项目里本身还在演进的实验性代码。

提取后的三种归宿：

| 归宿 | 含义 | 例子 |
|---|---|---|
| `infra/` | 基础设施，业务可直接依赖 | 配置、日志、数据、HTTP、认证 |
| `apps/` | 内置后台应用，可按需装配 | 用户、角色、配置、任务、日志 |
| 不进入框架 | 留在业务项目 | 代理规则、容器操作、邮件归档 |

## 2. 从参考项目提取的能力

### 2.1 RockSys

| 来源 | 能力 | EasyAdmin 落地 | 处理方式 |
|---|---|---|---|
| `internal/conf` | 注册式配置、默认值快照、热更、密钥识别 | `infra/conf` | 保留模型，去掉代理专属字段，增加分组与类型化 API |
| `internal/adminapi` | 管理 API 骨架、鉴权、统一响应 | `infra/httpapi` + `apps/system/*` | 拆分：通用 HTTP 能力进 infra，用户/配置等进 app |
| `internal/adminapi/auth.go` | 会话/JWT、静态令牌、登录限流 | `infra/auth` | 默认改为服务端会话，保留限流思路 |
| `internal/db` | SQL 脚本、方言、结构检查 | `infra/db` | 保留 SQL-first；schema diff 作为 P6 增强 |
| `internal/taskcenter` | 长任务统一注册与观测 | `infra/tasks` | 从纯内存升级为持久化，吸收 EmailPlugin 的任务模型 |
| `internal/hotswap/script.go` | 外挂脚本优先、内嵌兜底 | `infra/db` 的 SQL 加载 | 保留“外挂优先”语义，限定在 SQL/静态资源 |
| `catalog` | 组件/服务元信息 | `kernel` 模块清单 | 合并进模块 Manifest，不单独造概念 |
| `webui` 分层与组件 | 原生 JS、布局、表格、提示、主题 | `webui/core` + `webui/components` | 保留交互经验，模块化方式升级为 ESM |
| `-tags dev` | 前端免编译热重载 | `webui` 构建模式 | 直接继承 |
| WebUI 测试反馈环 | 浏览器内用例与架构机检 | `webui/_test` | 直接继承并扩展 |
| 文档红线与计划宪法 | 工作流约束 | `AGENTS.md` + `docs/` | 提炼为更精简的框架版 |

不进入框架：代理引擎、转发链、WAF、限流、GeoIP、Lua 脚本、MQ、对象存储插件。

### 2.2 EmailPlugin

| 来源 | 能力 | EasyAdmin 落地 | 处理方式 |
|---|---|---|---|
| `internal/httpapi` | 路由、鉴权、统一响应、探针、访问日志 | `infra/httpapi` | 作为 HTTP 内核的重要参考 |
| `internal/conf/registry.go` | 配置注册、校验、热更、必填、密钥 | `infra/conf` | 吸收 `Hot/Require/Validate` 语义 |
| `internal/taskcenter` | 持久任务、父子任务、进度、日志、取消、重试、接管 | `infra/tasks` | 直接作为任务中心蓝本，精简业务字段 |
| `internal/schedule` | 登记、扫描、漏跑补触发、跳过记账 | `infra/scheduler` | 保留“到期扫描 + 不堆积”模型 |
| `internal/schema` | 声明式建表、补列、差异检查 | `infra/db/schema` | P2 先做幂等建表与版本迁移，差异检查后置 |
| `internal/storage` | 存储接口 + 驱动注册表 | `infra/storage` | 保留“业务不 import 厂商 SDK”的边界 |
| `internal/db` | 脚本外置、超时上下文、批量写入 | `infra/db` | 合并 RockSys/KitBus 的数据层经验 |
| WebUI 页面组织 | 视图/组件/提示分层 | `webui` | 吸收命名与交互约束 |

不进入框架：邮件 Graph 客户端、归档编排、ERP 同步、正文渲染、凭据加密。

### 2.3 KitBus

| 来源 | 能力 | EasyAdmin 落地 | 处理方式 |
|---|---|---|---|
| `infra/` 与 `internal/` 分层 | 基础设施与业务解耦 | `infra/` 与 `apps/` | 直接继承，并强化为强制依赖方向 |
| `infra/conf` | easyconf 封装 | `infra/conf` | 合并到统一配置中心 |
| `infra/db` | easydb 封装、SQL 脚本、`{table}` 占位 | `infra/db` | 直接继承 |
| `infra/log` | easyserver/log 薄封装 | `infra/logx` | 扩展请求字段与轮转 |
| `infra/adminapi` | 静态资源 + `/api/` 前缀约束 | `infra/httpapi` | 保留前缀红线与 SPA 回退 |
| `webui/app/config.js` | 菜单树、路由、页面登记 | `webui/core` | 升级为服务端菜单 + 动态 import |
| `webui/app/modules.js` | 全局布局唯一 owner | `webui/core/layout.js` | 保留“页面不碰全局区域”红线 |
| `webui` 测试面 | 四层用例与架构机检 | `webui/_test` | 直接继承 |

不进入框架：容器引擎探测、镜像/实例管理、OCI 客户端、镜像源业务。

## 3. 基础设施组件清单

### 3.1 kernel

| 项 | 职责 | 唯一入口 | 禁止 |
|---|---|---|---|
| 装配上下文 | 持有配置、日志、数据、HTTP 等基础设施能力 | `kernel.App` | 全局单例取用 |
| 模块注册表 | 模块登记、依赖排序、冲突检测 | `App.Use` / `App.Install` | `init()` 自动注册 |
| 生命周期 | Init/Start/Stop、优雅停机 | `App.Run` | 模块自行起不受控 goroutine |
| 服务注册 | 跨模块显式接口 | `Context.Provide` / `Context.Service` | 模块间直接 import |
| 版本信息 | 版本、提交、构建时间 | `kernel/buildinfo` | 散落各处的版本变量 |

### 3.2 infra/conf

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 配置注册 | 类型、默认值、说明、必填、热更 | `conf.Registry` | 业务 `os.Getenv` |
| 取值链 | 命令行 → 环境变量 → `.env` → `default.env` → 默认值 | `conf.Manager` | 自定义读取顺序 |
| 默认值快照 | 自动生成 `default.env` | `Manager.SyncDefaultFile` | 手工编辑 |
| 热更 | 标记项运行期变更与广播 | `Manager.Watch` | 未标记项热改 |
| 密钥 | 识别、脱敏、禁止运行期修改 | `conf.IsSecret` | 日志/接口回显 |

### 3.3 infra/logx

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 应用日志 | 结构化 key/value、级别、文件 | `Logger.Info/Warn/Error`（由构造函数/模块上下文注入） | `fmt.Println` |
| 请求日志 | request_id、路由、状态、耗时 | `httpapi` 中间件 | handler 自行打印访问日志 |
| 审计日志 | 谁做了什么 | `audit.Record` | 混入应用日志 |
| 轮转 | 按大小切分与保留 | `logx` 内部 | 各模块自建文件写入 |

### 3.4 infra/db

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 数据源 | 命名连接、驱动、连接池 | `db.Manager` | 业务 `sql.Open` |
| SQL 脚本 | 嵌入、方言选择、外挂覆盖 | `Source.Exec/One/Many` | 长 SQL 拼在 Go 里 |
| 事务 | 显式事务回调 | `Source.Tx` | 跨层持有事务对象 |
| 建表/迁移 | Ensure、版本迁移、结构检查 | `db/schema` | 启动时隐式删改列 |
| 时间 | UTC 存储、接口 RFC3339 | `db` + 模块约定 | 各模块自定义时区口径 |

### 3.5 infra/httpapi

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 路由 | 方法 + 路径 + 权限 + handler | `httpapi.Router` | 散落 `http.HandleFunc` |
| 中间件 | Recover/RequestID/AccessLog/CORS/Auth… | `httpapi.Use` | 模块自造全局中间件 |
| 响应 | 成功/失败/分页统一结构 | `httpapi.Reply` | 手写 JSON |
| 参数 | 查询/JSON/表单绑定 | `httpapi.Bind` | 各处重复解析 |
| 错误 | 错误码到状态码映射 | `apperror` + `httpapi` | 裸 `err` 返回 |
| 静态 | WebUI 内嵌与 SPA 回退 | `httpapi.Static` | 业务模块自带静态目录 |

### 3.6 infra/auth

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 会话 | 登录、登出、当前用户 | `auth.Session` | 业务自建 Cookie |
| 密码 | bcrypt 哈希与校验 | `auth.Password` | 明文/自造加密 |
| 权限 | 权限码校验 | `auth.Can` | 前端隐藏代替后端校验 |
| 登录限流 | 按来源限制失败次数 | `auth.LoginLimiter` | 无限制登录 |
| 初始化 | 首次管理员创建 | `auth.Bootstrap` | 默认弱口令 |

### 3.7 infra/tasks

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 提交 | 创建任务并后台执行 | `tasks.Center.Submit` | handler 里直接起 goroutine |
| 进度 | 节流写库 | `Run.Progress` | 每个循环都写库 |
| 日志 | 任务日志追加 | `Run.Log` | 混入应用日志 |
| 取消 | 上下文取消 | `Center.Cancel` | 无法取消的长任务 |
| 重试 | 按 kind 注册重试逻辑 | `Center.Retry` | 通用盲目重试 |
| 接管 | 重启后收口 | `Center.Takeover` | 遗留 running 不处理 |

### 3.8 infra/scheduler

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 登记 | 调度项与 runner 绑定 | `scheduler.Register` | 模块自建 ticker |
| 触发 | 到期扫描、跳过记账 | `Scheduler.Start` | 堆积排队 |
| 配置 | 开关、周期热更 | 配置中心 | 硬编码周期 |
| 执行 | 统一下发任务中心 | `tasks.Submit` | 调度器内直接跑长任务 |

### 3.9 infra/storage

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 写入 | 原子写、校验 | `storage.Driver.Put` | 直接写目标路径 |
| 读取 | 鉴权后取流 | `storage.Driver.Open` | 暴露存储目录 |
| 删除 | 显式删除 | `storage.Driver.Delete` | 按无引用批量删 |
| 驱动 | 本地/对象存储 | 显式装配 | 业务 import 厂商 SDK |

## 4. 内置 app 清单（模块 ID 保持稳定）

| 模块 ID | 页面 | 能力 | 期次 |
|---|---|---|---|
| `dashboard` | 仪表盘 | 概览卡片、快捷入口、系统状态 | P3 |
| `system.users` | 用户管理 | 列表、创建、编辑、禁用、重置密码 | P2/P3 |
| `system.roles` | 角色管理 | 角色 CRUD、权限分配 | P2/P3 |
| `system.config` | 配置中心 | 分组、类型化编辑、热更、密钥脱敏 | P2/P3 |
| `system.logs` | 运行日志 | 级别/时间/关键字查询 | P4 |
| `system.audit` | 审计日志 | 操作审计查询 | P4 |
| `system.tasks` | 任务中心 | 列表、详情、进度、日志、取消、重试 | P4 |
| `system.schedules` | 调度器 | 列表、启停、周期、上次结果 | P4 |
| `system.files` | 文件管理 | 上传、列表、下载、删除 | P4 |
| `system.info` | 系统信息 | 版本、构建、运行时、依赖状态 | P4 |
| `profile` | 个人资料 | 当前用户信息、改密码 | P3 |

模块准入标准：

1. 至少两个真实项目会用；
2. 与具体业务无关；
3. 有清晰的权限码与审计动作；
4. 可以整体关闭而不影响内核启动。

## 5. 前端组件清单

| 组件 | 职责 | 依赖 | 期次 |
|---|---|---|---|
| `layout` | 顶栏、侧栏、面包屑、内容区、页脚 | core | P3 |
| `menu` | 服务端菜单渲染、展开、激活态 | layout | P3 |
| `router` | 路径匹配、动态 import、404 | core | P3 |
| `api` | 统一请求、错误归一化、超时 | core | P3 |
| `notify` | success/info/warn/error 统一提示 | core | P3 |
| `modal` / `drawer` | 弹窗与抽屉 | core | P3 |
| `data-table` | 列定义、排序、选择、分页、空态 | core | P3 |
| `form` | 输入、选择、开关、校验、布局 | core | P3 |
| `filter-bar` | 查询条件组合与重置 | form | P3 |
| `tabs` | 页内标签 | core | P3 |
| `empty` | 空态与引导 | core | P3 |
| `page-header` | 标题、描述、页面级操作 | core | P3 |
| `code-editor` | SQL/脚本编辑 | 按需加载 | P4 |
| `chart` | 图表 | 按需加载 | P4 |

组件红线：

- 通用组件不得引用 `api`、不得知道业务 URL、不得读写业务 store；
- 组件通过参数与回调接收数据和事件；
- 页面不得手写裸表单控件，统一经 `form`；
- 所有提示经 `notify`，所有请求经 `api`。

## 6. 不进入框架的参考能力

| 能力 | 来源 | 原因 |
|---|---|---|
| 反向代理引擎 | RockSys | 产品能力，不是中后台基础设施 |
| WAF/限流/GeoIP | RockSys | 网关专属 |
| Lua 策略引擎 | RockSys | 非通用需求，维护成本高 |
| 容器/镜像管理 | KitBus | 运维工具专属 |
| OCI 客户端 | KitBus | 外部协议适配，按需 |
| 邮件归档/Graph | EmailPlugin | 业务专属 |
| ERP 同步 | EmailPlugin | 业务专属 |

这些能力未来可以作为独立模块或适配器存在，但不得进入 `infra/`。


## 7. 公共组件准入与版本治理

公共组件进入 `infra/` 或公共 `webui/components/` 前必须回答：

1. 是否至少有两个业务模块/项目会使用？
2. 是否存在明确且稳定的统一入口？
3. 是否有清晰生命周期、错误语义和测试替身？
4. 是否能独立关闭/替换，而不让内核依赖具体业务？
5. 是否比直接使用标准库或现有 easy 系列明显更有价值？

禁止仅因为“未来可能需要”而建立空接口、空 Provider 或万能 Manager。接口由实际消费者定义，默认实现先保持最小。

## 8. 当前版本与验证边界

- easy 系列 API、并发、关闭、热更新和驱动兼容性必须以被锁定版本的源码和测试为准。
- `tasks`、`scheduler`、`storage` 等可以从参考实现提炼设计模型，但不应在尚未完成 P2/P4 之前宣称具备可靠的持久化/接管/多实例能力。
- `EmailPlugin`、`KitBus` 的源码已在当前工作区核对（见 `ROCKSYS_ASSET_REVIEW.md`）；其任务、调度、Schema、存储等实现仍作为待审查来源，完成锁定版本测试前不能标为已验证资产。
