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
| `internal/schema` | 声明式建表、补列、差异检查 | `infra/db/schema` | P2 先做幂等建表，差异检查只做只读预览 |
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

状态口径：本节表格描述目标公开职责；当前源码已存在不等于对应持久化、管理 API 或 WebUI 已交付。当前代码目录/文件以 `PROJECT_STRUCTURE.md` 的现状清单为准。

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
| 结构同步 | Ensure 补齐缺失表/索引、DiffSchema 只读预览 | `db/schema` | 启动时隐式删改列 |
| 时间 | UTC 存储、接口 RFC3339 | `db` + 模块约定 | 各模块自定义时区口径 |

### 3.5 infra/httpapi

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 路由 | 方法 + 路径 + 权限 + handler | `httpapi.Router` | 散落 `http.HandleFunc` |
| 中间件 | Recover/RequestID/AccessLog/CORS/Auth… | `httpapi.Use` | 模块自造全局中间件 |
| 响应 | 成功/失败/分页统一结构 | `httpapi.Reply` | 手写 JSON |
| 参数 | 查询/JSON/表单解析与绑定 | `httpapi.Bind` | 各处重复解析 |
| 错误 | 错误码到状态码映射 | `apperror` + `httpapi` | 裸 `err` 返回 |
| 静态 | WebUI 内嵌与 SPA 回退 | `httpapi.Static` | 业务模块自带静态目录 |

请求校验的职责边界：

- `httpapi.Bind` 负责 HTTP 输入的解析、类型转换和绑定，不承载具体业务规则。
- 字段级校验负责必填、长度、格式、范围和枚举等可复用规则；失败映射为 HTTP 422，并按 `API.md` 返回字段级 `details.fields`。
- 唯一性、资源状态、权限和跨实体约束由 Service/Repository 与数据库约束共同保证；不能仅靠通用 Validator。
- 是否独立建设 `infra/validate`，留待 P1 使用真实请求模型评估；P0 不为对标其他框架而先引入校验依赖。

### 3.6 infra/auth

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 会话 | 登录、登出、当前用户 | `auth.Session` | 业务自建 Cookie |
| 密码 | bcrypt 哈希与校验 | `auth.Password` | 明文/自造加密 |
| 权限 | 权限码校验 | `auth.Can` | 前端隐藏代替后端校验 |
| 登录限流 | 按来源限制失败次数 | `auth.LoginLimiter` | 无限制登录 |
| 初始化 | 首次管理员创建 | `auth.Bootstrap` | 默认弱口令 |

### 3.7 infra/credential

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 凭据加解密 | AES-256-GCM 密文封装与解封装 | `credential.Encrypt/Decrypt` | 明文落库、日志或 API 回显 |
| SSH 连接 | 认证参数校验、连接测试、known_hosts 校验 | `credential.Test` | 业务模块自行拼接 SSH 鉴权 |
| SSH 配置 | `~/.ssh/config` 别名解析 | `credential.LookupAlias` | 未验证别名解析结果直接执行 |
| 隧道 | TCP 端口转发 | `credential.Tunnel` | 让 infra 依赖 apps 页面/权限 |

当前 `infra/credential` 已有源码实现；服务器记录、凭据仓储、页面和授权仍属于上层应用及后续阶段，不等于已交付。

### 3.8 infra/tasks

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 提交 | 创建任务并后台执行 | `tasks.Center.Submit` | handler 里直接起 goroutine |
| 进度 | 节流写库 | `Run.Progress` | 每个循环都写库 |
| 日志 | 任务日志追加 | `Run.Log` | 混入应用日志 |
| 取消 | 上下文取消 | `Center.Cancel` | 无法取消的长任务 |
| 重试 | 按 kind 注册重试逻辑 | `Center.Retry` | 通用盲目重试 |
| 接管 | 目标能力：重启后恢复/收口 | 持久化仓储适配层（待建设） | 将当前内存态误称为已持久化 |

### 3.9 infra/scheduler

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 登记 | 调度项与 runner 绑定 | `scheduler.Register` | 模块自建 ticker |
| 触发 | 到期扫描、跳过记账 | `Scheduler.Start` | 堆积排队 |

当前 `infra/scheduler` 已有 `Store` 接口与 `MemoryStore`；数据库 Store 尚未实现。
| 配置 | 开关、周期热更 | 配置中心 | 硬编码周期 |
| 执行 | 统一下发任务中心 | `tasks.Submit` | 调度器内直接跑长任务 |

### 3.10 infra/storage

| 项 | 职责 | 入口 | 禁止 |
|---|---|---|---|
| 写入 | 原子写、校验 | `storage.Driver.Put` | 直接写目标路径 |
| 读取 | 鉴权后取流 | `storage.Driver.Open` | 暴露存储目录 |
| 删除 | 显式删除 | `storage.Driver.Delete` | 按无引用批量删 |
| 驱动 | 本地/对象存储 | 显式装配 | 业务 import 厂商 SDK |

## 4. 内置页面清单（Page ID 保持稳定）

| Page ID | 页面 | 能力 | 期次 |
|---|---|---|---|
| `dashboard` | 仪表盘 | 概览卡片、快捷入口、系统状态 | P03 已交付 |
| `profile` | 个人资料 | 当前用户信息、改密码 | P03 已交付 |
| `system.users` | 用户管理 | 列表、创建、编辑、禁用、重置密码 | P02 API / P03 页面 |
| `system.roles` | 角色管理 | 角色 CRUD、权限分配 | P02 API / P03 页面 |
| `system.audit` | 审计日志 | 操作审计查询 | P02 API / P03 页面 |
| `system.config` | 配置中心 | 分组查看、类型展示、密钥脱敏；写入延后 | P03 已交付（只读） |
| `system.logs` | 运行日志 | 级别/时间/关键字查询 | P4 |
| `system.tasks` | 任务中心 | 列表、详情、进度、日志、取消、重试 | P4 |
| `system.schedules` | 调度器 | 列表、启停、周期、上次结果 | P4 |
| `system.files` | 文件管理 | 上传、列表、下载、删除 | P4 |
| `system.info` | 系统信息 | 版本、构建、运行时、依赖状态 | P4 |

模块准入标准：

1. 至少两个真实项目会用；
2. 与具体业务无关；
3. 有清晰的权限码与审计动作；
4. 可以整体关闭而不影响内核启动。

## 5. 前端组件清单

状态：**已落地**（2026-11-25，用户授权提前移植 `D:\projects\emailplugin\webui`）。逐文件映射与证据见 [WEBUI_ASSET_REVIEW.md](WEBUI_ASSET_REVIEW.md)。

### 5.1 前端内核

| 模块 | 职责 | 文件 |
|---|---|---|
| `api` | 唯一请求入口：CSRF、超时、401、错误归一化、分页归一化 | `webui/assets/core/api.js` |
| `boot` | 启动流程：meta → 初始化/登录/主布局 | `webui/assets/core/boot.js` |
| `router` | 路径匹配、动态 import、mount/unmount、Page ID 登记 | `webui/assets/core/router.js` |
| `layout` | 顶栏/侧栏/内容区唯一 owner | `webui/assets/core/layout.js` |
| `menu` | 服务端菜单渲染、展开、激活 | `webui/assets/core/menu.js` |
| `notify` | 唯一提示入口 | `webui/assets/core/notify.js` |
| `store` | 显式低频状态容器 | `webui/assets/core/store.js` |
| `perm` | 按钮级权限辅助 | `webui/assets/core/perm.js` |
| `theme` | 三主题切换与持久化 | `webui/assets/core/theme.js` |
| `dom` | 转义、安全 DOM、焦点恢复、动作委托 | `webui/assets/core/dom.js` |
| `format` | 日期/字节/数字/时长格式化 | `webui/assets/core/format.js` |
| `data` | SWR + 单飞 + TTL 数据供应商缓存 | `webui/assets/core/data.js` |

### 5.2 通用组件

| 组件 | 职责 | 依赖 | 状态 |
|---|---|---|---|
| `page-header` | 标题、描述、页面级操作、全站唯一刷新按钮 | dom | 已落地 |
| `data-table` | 列声明、客户端/服务端分页、多选、行详情 | dom/format/empty/detail-modal | 已落地 |
| `form` | 输入/选择/开关/单选/复选/字段错误映射 | dom | 已落地 |
| `filter-bar` | 声明式筛选、live 查询、中文输入法安全、焦点恢复 | dom/date-range | 已落地 |
| `modal` | 弹窗、二次确认、ESC 栈 | dom | 已落地 |
| `detail-modal` | 键值详情、长文本、复制 | dom/modal | 已落地 |
| `date-range` | 四输入 → RFC3339，严格日期校验 | 无 | 已落地 |
| `tabs` / `badge` / `empty` | 页签 / 语义标签 / 空态 | dom | 已落地 |
| `action-panel` | 维护动作卡 | dom | 已落地 |
| `log-view` | 任务日志分级视图 | dom/format | 已落地 |
| `dsn-form` | 命名数据源表单（驱动/直连/SSH/DSN） | form | 已落地 |
| `tooltip` | `[data-tip]` 全局跟随提示 | dom | 已落地 |

### 5.3 明确未落地

- `drawer`：参考实现未提供，等真实页面出现抽屉需求再设计，不预造。
- `code-editor` / `chart`：按 P4 需求动态加载，不在首屏引入。
- 业务页面：邮件、邮箱、同步、ERP 等不得进入框架。

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


## 7. 公共组件设计规范、准入与版本治理

### 7.1 统一设计语言（不建立万能接口）

GoFrame 值得借鉴的是组件之间一致的工程约定，而不是要求所有组件实现同一个接口。每个公共组件在设计和实现时，必须明确以下契约：

| 维度 | 统一要求 | 边界 |
|---|---|---|
| 职责 | 说明解决什么问题、不解决什么问题、由谁消费 | 不把多个无关能力合并成万能 Manager |
| 公开 API | 提供少量、稳定、可读的构造函数和操作方法 | 接口由消费者定义，不为未来假设预建空接口 |
| 依赖 | 依赖通过构造函数显式传入，优先小接口 | 不使用全局可变对象或隐式 Service Locator |
| 配置 | 配置项进入统一注册表，描述类型、默认值、敏感级别和热更语义 | 组件不得自行读取环境变量或建立第二套配置源 |
| 上下文 | IO 操作接收调用方的 `context.Context`，明确超时与取消语义 | 不把请求级 Context 长期保存到组件实例 |
| 生命周期 | 只有拥有连接、后台任务等资源的组件才声明 Start/Stop/Close，并明确关闭顺序 | 纯函数或无资源组件不被迫实现生命周期接口 |
| 错误 | 保留底层错误链；跨 HTTP 边界映射为稳定业务错误码 | 内部 Cause/堆栈不得直接暴露给客户端 |
| 日志与观测 | 使用注入的结构化 Logger，关联 request_id/task_no 等上下文 | 不记录密码、令牌、DSN 等敏感信息；不要求每个组件自建 Trace 系统 |
| 并发 | 声明线程安全、可重入和关闭期间的行为 | 有共享状态或并发路径时必须有 race 测试 |
| 测试 | 覆盖正常、失败、取消、关闭和边界行为；提供必要的测试替身 | 不为测试而扩大生产接口 |
| 适配 | easy 系列及第三方 SDK 的具体类型尽量留在 `infra/**/internal` | 对上暴露 EasyAdmin 自己的稳定契约 |

这些是设计与评审清单，不是要求每个组件机械实现全部能力。组件可以保持很小，但不能让调用方猜测其错误、并发和资源所有权语义。

### 7.2 公共组件准入

公共组件进入 `infra/` 或公共 `webui/components/` 前必须回答：

1. 是否属于框架基础能力，或至少有两个真实业务模块/项目会使用？
2. 是否有清晰的职责、消费者和稳定统一入口？
3. 是否有明确的配置、错误、上下文、并发和资源生命周期语义（不适用项说明原因）？
4. 是否能独立关闭/替换，而不让内核依赖具体业务？
5. 是否比直接使用标准库或现有 easy 系列明显更有价值？
6. 是否有可重复的测试证据，尤其是失败、取消和关闭路径？
7. 是否会强迫不需要该能力的项目引入额外依赖？

禁止仅因为“未来可能需要”而建立空接口、空 Provider 或万能 Manager。接口由实际消费者定义，默认实现先保持最小。

### 7.3 版本与适配治理

- easy 系列和第三方适配器的版本必须由仓库依赖清单固定；不得依赖未锁定的“最新版本”。
- 适配层负责隔离底层库的 API 变化；底层具体类型不得无必要地泄漏到业务模块。
- 公共契约只做经过测试的兼容性演进；变更前检查调用方、测试、示例和文档。
- 可选能力通过显式 Provider/Adapter 装配；未安装或未启用时必须有明确状态，不得静默退化。

## 8. 当前版本与验证边界

- easy 系列 API、并发、关闭、热更新和驱动兼容性必须以被锁定版本的源码和测试为准。
- 独立校验包是否成立，必须先在 P1 用真实请求 DTO 和校验规则验证；不以 GoFrame 的 `gvalid` 作为引入依赖的充分理由。
- 国际化、缓存和完整分布式链路追踪均不是当前基础组件缺口；在真实业务需求与数据模型明确前，不新增对应子系统。
- `tasks`、`scheduler`、`storage` 等可以从参考实现提炼设计模型，但不应在尚未完成 P2/P4 之前宣称具备可靠的持久化/接管/多实例能力。
- `EmailPlugin`、`KitBus` 的源码已在当前工作区核对（见 `ROCKSYS_ASSET_REVIEW.md`）；其任务、调度、Schema、存储等实现仍作为待审查来源，完成锁定版本测试前不能标为已验证资产。
