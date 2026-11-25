# Rocksys / EmailPlugin 通用能力迁移清单

> 本文记录源码迁移状态，不等同于产品功能、数据库持久化或 WebUI 已交付。状态以当前仓库源码为准；“此前测试通过”不代表本轮已重新执行。

## 1. 已迁移的源码能力

| 目标包 | 来源 | 当前状态 | 尚未包含 |
|---|---|---|---|
| infra/conf | rocksys/internal/conf | 注册制配置、默认值快照、来源优先级、热更、List() 已迁移 | 配置来源/持久化元数据的完整管理 UI 契约仍需 P01 核验 |
| infra/db | rocksys/internal/db | TableSpec、DDL 解析、多语句拆分、Catalog 查询、结构 Diff、安全 SQL 生成、幂等建表已迁移 | schema migration runner、size 等待迁移 |
| infra/db/datasource.go | rocksys/internal/adminapi/dsn.go + easydb/dsn | 命名数据源、DSN 去重、激活项和 JSON 文件仓储已迁移 | 持久化到 EasyAdmin 系统数据库、应用管理 API |
| infra/tasks | rocksys/internal/taskcenter | 任务注册、提交、进度快照、互斥、取消、终态保留和 panic 收口已迁移 | 当前纯内存；无数据库仓储、持久化任务日志、重启恢复、跨进程接管 |
| infra/credential | emailplugin/internal/credential | AES-256-GCM、SSH 测试、~/.ssh/config 别名、SSH 隧道已迁移 | 服务器记录持久化、管理 API、权限和页面 |
| infra/scheduler | emailplugin/internal/schedule | 调度核心已重写；有 Plan/Def/Entry/Runner、到期扫描、任务派发、Store 接口和 MemoryStore | 数据库 Store、持久化恢复与多实例保护 |

## 2. 待补齐或迁移

| 目标能力 | 来源 | 状态与约束 |
|---|---|---|
| infra/db/migrate | rocksys/internal/adminapi/migrate.go | 表迁移引擎与 HTTP handler、AdminServer、任务中心耦合，先拆独立执行器 |
| infra/db/size | rocksys/internal/adminapi/dbsize.go | 表空间统计可迁移，先抽方言查询与缓存 |
| infra/tasks 持久化适配 | EasyAdmin 目标数据模型 | 以现有 tasks.Center 为执行/并发规则核心，补 Store/Repository 与恢复协议；不得平行重写任务中心 |
| infra/scheduler 持久化 Store | 当前 scheduler.Store 接口 + sys_schedule | 对齐唯一数据字典后实现 SQL Store；配置键仍是启停唯一真源 |
| infra/logx 查询 | rocksys/internal/adminapi/handlers_log.go、execlogstore.go | reader/query 接口待定义 |
| 系统信息 | rocksys/internal/adminapi/handlers_system.go | 进程/CPU/内存采样待迁移 |
| 存储 | emailplugin/internal/storage | 本地驱动和接口待迁移；qiniu 驱动不进入核心 |

## 3. 不迁移

- rocksys 的代理、WAF、GeoIP 同步、Lua 热切换、网关限流；
- emailplugin 的邮件 Graph、归档编排、ERP 同步；
- kitbus 的容器引擎探测、镜像/实例管理；
- 各项目 HTTP handler 和页面：核心能力进 infra，管理页面进 apps/system。

## 4. 历史验证记录与本轮验证边界

此前仓库文档记录以下命令已通过：go test ./infra/...、go vet ./infra/...，并在 README 记录过 go test -race ./infra/...。本轮通过 GitHub 源码接口检查了实现与文档，没有运行 Go 工具链，因此不把历史记录宣称为本轮测试结果。

P0/P1 实施前应在实际工作区重新执行：

~~~bash
go test ./infra/...
go test -race ./infra/...
go vet ./infra/...
~~~

infra/tasks 和 infra/scheduler 当前仍为内存态；持久化能力归 P04 设计/实施，不得把“核心源码已迁移”写作“任务中心已经持久化”。


## 4. WebUI 通用资产（emailplugin/webui）

状态：**已落地**（2026-11-25，用户授权提前移植）。只迁移框架级内核、组件、样式和测试面，业务页面不进入框架。证据见 [WEBUI_ASSET_REVIEW.md](WEBUI_ASSET_REVIEW.md)。

| 目标 | 来源 | 迁移内容 | 未迁移 / 待办 |
|---|---|---|---|
| `webui/assets/core` | emailplugin `assets/js/{api,ui,theme,data,state,nav,main,util}.js` | 请求、提示、主题、SWR 缓存、菜单、路由、布局、状态、格式化、DOM 安全与焦点恢复 | 业务状态映射；hash 路由改路径路由；全局 `Rock` 改 ESM |
| `webui/assets/components` | emailplugin `assets/js/components/*.js` | 表格、表单、筛选、弹窗、详情、时间范围、页签、标签、空态、动作卡、日志、DSN、tooltip | drawer 未实现；组件不得引用 api/notify |
| `webui/assets/styles` | emailplugin `assets/style.css` | 设计令牌、三套主题、布局、组件样式 | 调度页专用样式；业务页面样式 |
| `webui/_test` | emailplugin `_test/*` | 零依赖 runner、core/component/architecture 用例 | module 层用例待页面建立后补 |
| `webui/embed*.go` | emailplugin `embed*.go` | 生产 go:embed 与 dev 磁盘双模式 | 静态服务与鉴权白名单归 `infra/httpapi` |
