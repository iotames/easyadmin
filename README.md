# EasyAdmin

> 以自研 `easyconf`、`easydb`、`easyserver` 为地基的 Go 中后台框架。把 Web 业务之外的基础设施收敛为一套透明、可组合、可独立演进的公共底座，让业务项目主要关注领域模型、业务规则和业务页面。

当前状态：**P00–P03 已实施、验收并归档（2026-11-26）；P04–P06 未开工**。仓库已有 `infra/conf`、`infra/db`、`infra/tasks`、`infra/scheduler`、`infra/credential` 等代码，其中任务中心和调度 Store 当前为内存实现；`cmd/easyadmin` 组合根、`kernel` App 生命周期、`infra/logx`、`infra/apperror`、`infra/httpapi`、`infra/auth`、`infra/audit`、`sql/` 与 `apps/system` 已建立。`webui/` 已落地前端内核、通用组件、样式令牌、浏览器测试面和六个核心页面。P00–P03 均通过 `go build ./...`、`go vet ./...`、`go test ./...`，P02/P03 另通过关键包 race 与浏览器测试面；验收证据见 `docs/done/` 对应设计计划。

已确认：**API 统一使用 `/api/` 前缀，不设路径版本号，接口从首个公开版本起向下兼容，只增不改。**

## 1. 定位

EasyAdmin 的目标是成为 Go 生态中具有 Yii 3 / Symfony / Laravel 工程完成度的中后台框架，但不复制 PHP 框架的隐式魔法。借鉴成熟框架的组件化、约定、认证授权、结构同步、任务、后台 UI、脚手架和开发体验；坚持 Go 的显式依赖、静态类型、可读调用链、可检查错误和标准库兼容。

核心原则：**简易之道，以简驭繁；基础功能收敛封装，约束统一入口。**

## 2. 使用方式

### 2.1 运行框架自身

框架仓库自带可运行的管理后台：

```bash
go run ./cmd/easyadmin
# 生产构建
go build -o bin/easyadmin ./cmd/easyadmin
./bin/easyadmin
```

浏览器打开 http://127.0.0.1:19527。首次打开进入初始化页，创建第一个管理员后登录；默认使用 SQLite `data/easyadmin.db`，核心表在启动时幂等补齐。

常用启动配置（可由环境变量或同名命令行参数覆盖）：

| 变量 | 默认值 | 说明 |
|---|---|---|
| `EASYADMIN_LISTEN` | `127.0.0.1:19527` | HTTP 监听地址 |
| `EASYADMIN_DB_DRIVER` | `sqlite` | 默认数据源驱动 |
| `EASYADMIN_DB_DSN` | `data/easyadmin.db` | 默认数据源 DSN |
| `EASYADMIN_DATA_DIR` | `./data` | 运行数据目录 |
| `EASYADMIN_ADMIN_PASSWORD` | 空 | 非回环地址首次初始化必填，并作为确认口令 |
| `EASYADMIN_LOG_LEVEL` | `info` | 应用日志级别 |

非回环监听时首次初始化必须设置 `EASYADMIN_ADMIN_PASSWORD`；不要把真实密钥写入仓库。

### 2.2 作为业务项目依赖（P05 目标用法）

> 以下为 P05 交付后的目标用法，当前 CLI、模板和业务页面注入尚未实现；接口以届时设计计划为准。

```bash
easyadmin new myshop
cd myshop
easyadmin module add orders
go run ./cmd/myshop
```

生成的项目只包含业务薄壳：`cmd/`、`apps/`、`webui/`、`config/`、`go.mod`。业务项目通过 `go.mod` 依赖版本化 EasyAdmin，不复制 `kernel`、`infra` 和内置 `apps`。

业务模块实现 `kernel/module.Module`：

```go
type Module interface {
    Manifest() module.Manifest
    Declare(module.ConfigRegistrar) error
    Register(module.Registrar) error
}
```

在 Manifest 声明 `Pages`/`Permissions`；在业务项目 `webui/pages.js` 登记同名 Page ID：

```js
export default {
  'orders': () => import('./assets/pages/orders/index.js'),
};
```

组合根使用 `easyadmin.New()`、`Use(...)`、`WithWebAssets(...)`、`Run(ctx)` 装配；P05 前请按 2.3 的当前边界手工装配。

### 2.3 当前阶段（P00-P03）接入边界

截至 P03 已可用：框架自身启动、初始化/登录、用户/角色/权限/审计 API、核心表结构同步、WebUI 内核与登录流程、六个核心页面（Dashboard、个人资料、用户、角色、审计、配置只读），以及 `httpapi.HandlerOptions.AppAssets` 的业务前端资源挂载与 `/assets/app/pages.js` 合并接缝。

尚未交付：

- `easyadmin new/module add`、`templates/app`、`examples/biz`；
- 配置在线写入（P03 只交付只读 `GET /api/system/config`）；
- P04 的日志、任务、调度页面。

因此当前若要接入业务：

1. 在业务仓库 `go.mod` require `github.com/iotames/easyadmin`；
2. 自建 `cmd/<app>/main.go`，按顺序装配 `infra/conf`、`infra/db`、`apps/system`、`kernel/app`、`infra/httpapi`；
3. 业务模块放业务仓库 `apps/<domain>/`，不要放进 EasyAdmin 仓库；
4. 业务前端页面可经业务 `webui.Assets` + `httpapi.HandlerOptions.AppAssets` 挂载到 `/assets/app/` 接入；脚手架与自动化升级仍待 P05，框架升级前须自行保证 Page ID 唯一。

完整缺口清单（G1-G8）、影响与关闭阶段见 [docs/SCAFFOLD_AND_UPGRADE.md](docs/SCAFFOLD_AND_UPGRADE.md) 第 8 节。

### 2.4 扩展与升级规则

定制按以下优先级，不改框架源码：

1. 配置项；
2. 构造函数选项；
3. 消费方窄接口 + 组合根注入；
4. 追加注册路由、菜单、页面、权限和任务；
5. 提交上游或作为独立业务 app。

明确不支持：Go `.so` 插件、运行期目录扫描、热插拔、覆盖 core 文件。Page ID 冲突会启动失败，不静默替换。

升级只改依赖版本，不改业务文件：

```bash
go get github.com/iotames/easyadmin@vX.Y.Z
go mod tidy
```

核心结构 `scope=core` 随框架升级；业务结构 `scope=app:<module-id>`、业务页面和业务配置由业务项目自己维护。需要固定框架提交时可用官方仓库 submodule + `replace`，但不得在 submodule 内长期改框架源码。

默认目标仍是只需要 Go 工具链和现代浏览器即可启动；SQLite 为零配置默认数据源；WebUI 资源随 Go 二进制交付。PostgreSQL、MySQL、外部存储和通知渠道按需启用，且必须经过对应测试认证。

## 3. 设计原则

1. **显式装配**：模块、服务、路由和 provider 在装配处明确登记；不依赖 `init()`、目录扫描、注解和反射自动发现。
2. **统一入口**：配置、日志、数据库、HTTP、错误、身份权限、任务、调度、存储、审计、前端请求和提示各有唯一权威入口。
3. **分层单向依赖**：easy 系列独立演进；kernel 不依赖具体 infra 实现；infra 不依赖业务模块；模块不直接 import 其他业务模块。
4. **SQL-first**：SQL 外置、参数绑定、按方言管理，不使用 ORM 隐藏查询。
5. **原生 ESM**：WebUI 默认零构建、无 Node/npm 运行时前提，使用动态 `import()` 实现页面与重组件按需加载。
6. **模块自包含**：模块拥有自己的 API、权限、菜单/页面声明、SQL、配置、测试和 WebUI 入口。
7. **可观测、可回归**：每期都有可运行产物、自动化测试、浏览器验收和文档同步。
8. **反魔法**：构造函数注入、小接口、稳定错误码、显式生命周期；不使用全局可变容器或隐式 Service Locator。
9. **API 向下兼容**：统一 `/api/`，不设路径版本；只增不改，破坏性变更不作为常规手段。

## 4. 架构总览

```text
cmd/                  唯一组合根：创建 App、装配基础设施服务和 app、启动/停机
├── kernel/           App、Module 契约、注册表、生命周期（不依赖具体 infra）
├── infra/           基础设施统一入口：conf/db/credential/tasks/scheduler，以及后续补齐的 logx/httpapi/auth/audit/storage
│   └── internal/     easy 系列适配器与私有实现
├── apps/             按业务域组织的内置应用和业务应用
└── webui/            原生 ESM 前端内核、通用组件、模块页面和测试面

lib/easyconf、lib/easydb、lib/easyserver 是独立地基库；不反向依赖 EasyAdmin。
```

对应 module path：`github.com/iotames/easyconf`、`github.com/iotames/easydb`、`github.com/iotames/easyserver`；EasyAdmin 自身为 `github.com/iotames/easyadmin`。

具体依赖约束见 [ARCHITECTURE.md](docs/ARCHITECTURE.md)。

## 5. 核心公共能力

- 运行内核：App、模块 Manifest、显式注册、依赖校验、生命周期、优雅停机。
- 配置：注册表、类型校验、默认值快照、来源追踪、受控热更；v0.x 只做启动配置，运行设置后置。
- 日志与可观测：结构化应用日志、HTTP Access Log、独立 Audit Log、Task Log、健康与诊断。
- 数据：具名数据源、SQL 脚本、事务、结构同步、方言适配、慢查询。
- HTTP：统一路由、中间件、响应、错误映射、参数校验、静态资源。
- 安全：服务端会话、CSRF、RBAC、登录限流、审计、文件访问控制。
- 运维：持久任务、调度、存储、系统信息、日志和审计查询。
- WebUI：统一 Layout、Menu、Router、主题、API Client、Notify、表格、表单、弹窗和反馈组件。

完整清单见 [COMPONENTS.md](docs/COMPONENTS.md)。

## 6. 七期建设路线

| 阶段 | 主题 | 主要交付 | 版本目标 |
|---|---|---|---|
| P0 | 地基与工程规范 | 架构冻结、组件设计规范、依赖核验、开发/测试反馈环 | v0.0 |
| P1 | 运行时内核 | App、模块系统、配置、日志、HTTP、错误、健康检查 | v0.1-alpha |
| P2 | 数据与身份安全 | 数据源、SQL/结构同步、登录会话、RBAC、审计、初始化 | v0.1-beta |
| P3 | WebUI 管理台 | Layout、Menu、Router、懒加载、基础组件、系统管理页面 | v0.1 |
| P4 | 运维能力 | 任务、调度、文件、日志/审计查询、系统诊断 | v0.2 |
| P5 | CLI 与模块生态 | 项目模板、模块生成、Provider/Adapter、示例 | v0.5 |
| P6 | 稳定与正式发行 | 安全、性能、兼容、升级、发布与真实项目验证 | v1.0 |

每期严格配对：`<项目>_DESIGN_PLAN.md` 负责设计与验收边界，`<项目>_IMPL_PLAN.md` 负责文件级实施步骤、测试、风险和回退；两份文件位于 `docs/plan/` 根部，执行期 `STEPn_*.md` 放同名项目子目录。P00-P03 已完成并归档到 `docs/done/`，计划目录当前 P04-P06 共 3 对 6 份，见 [docs/plan/README.md](docs/plan/README.md)。

## 7. 文档索引

| 文档 | 作用 |
|---|---|
| [AGENTS.md](AGENTS.md) | 人类开发者与 Agent 的强制仓库守则 |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | 总体分层、模块系统、生命周期和边界 |
| [DECISIONS.md](docs/DECISIONS.md) | 关键架构决策、理由、代价与复审条件 |
| [ROADMAP.md](docs/ROADMAP.md) | 七期路线、依赖关系、质量闸门 |
| [COMPONENTS.md](docs/COMPONENTS.md) | 公共基础设施与组件清单、来源及归属 |
| [CONVENTIONS.md](docs/CONVENTIONS.md) | 统一入口、Go/SQL/API/测试工程规范 |
| [FRONTEND.md](docs/FRONTEND.md) | 原生 ESM、布局、菜单、路由与组件契约 |
| [API.md](docs/API.md) | `/api` 管理 API 契约 |
| [DATA_DICT.md](docs/DATA_DICT.md) | 核心数据表与字段草案 |
| [CONFIGURATION_DESIGN.md](docs/CONFIGURATION_DESIGN.md) | 配置注册、来源、热更和运行设置边界 |
| [LOGGING_DESIGN.md](docs/LOGGING_DESIGN.md) | 应用日志、访问日志、审计与任务日志 |
| [PROJECT_STRUCTURE.md](docs/PROJECT_STRUCTURE.md) | 目标目录结构与包职责 |
| [SCAFFOLD_AND_UPGRADE.md](docs/SCAFFOLD_AND_UPGRADE.md) | 脚手架、生成流程与业务项目独立升级机制 |
| [INFRA_MIGRATION.md](docs/INFRA_MIGRATION.md) | 从 rocksys/emailplugin 迁移通用基础设施的清单与状态 |
| [INFRA_SCHEDULER_AND_CREDENTIALS.md](docs/INFRA_SCHEDULER_AND_CREDENTIALS.md) | 调度器、凭据、SSH 隧道、服务器与数据源设计 |
| [ROCKSYS_ASSET_REVIEW.md](docs/ROCKSYS_ASSET_REVIEW.md) | RockSys 实际源码资产及证据边界 |
| [WEBUI_ASSET_REVIEW.md](docs/WEBUI_ASSET_REVIEW.md) | WebUI 通用资产移植清单与验证证据 |
| [COMPARISON_AND_MERGE.md](docs/COMPARISON_AND_MERGE.md) | 两套规划的对照与最终取舍记录 |

## 8. 当前不承诺

- 不做 ORM、Active Record、低代码建站、微服务治理或 RPC 内核。
- v1.0 不承诺多实例调度/任务抢占；当前明确单实例优先。
- 不把 RockSys 的代理、WAF、Lua、GeoIP、流量调度等网关能力纳入 EasyAdmin。
- 不以 Go `plugin` 动态加载 `.so` 作为首期扩展机制。
- 不为“生态丰富”预先堆积缓存、消息队列、对象存储和第三方身份源；按真实项目需求增加。

---

项目目标不是做一个什么都能做的巨型框架，而是建立一套默认路径很短、内部边界很清楚、可以持续演进的公共工程底座。

## 9. 本地开发

```bash
make build  # 构建生产二进制 bin/easyadmin（WebUI 内嵌）
make test   # go test ./...
make vet    # go vet ./...
make lib    # 同步 lib/ 下的 easy 系列依赖到上游最新代码
```

启动：`./bin/easyadmin --listen 127.0.0.1:19527`；`--listen` 可省略，默认 127.0.0.1:19527。开发模式用 `make build-dev` 构建，WebUI 直接读磁盘且 `/_test/` 可访问。

`.gitmodules` 声明的库走 `git submodule update --init --remote`（缺失自动克隆，已存在则更新，HEAD 处于 detached 属正常状态）；`lib/` 下未声明的普通仓库走 fetch + 快进同步。任一库有未提交改动会跳过并报错，不会静默覆盖。同步逻辑位于 `scripts/sync-lib.sh`，Makefile 仅保留入口。默认前缀为 `github:iotames`（走 SSH 别名 `github`），可用 `make lib LIB_GIT_PREFIX=git@github.com:iotames` 覆盖。 若 `git clone --recurse-submodules` 因 HTTPS 访问 GitHub 失败，可用 SSH 别名补齐子模块：`git -c url."github:iotames/".insteadOf="https://github.com/iotames/" submodule update --init --recursive`。
