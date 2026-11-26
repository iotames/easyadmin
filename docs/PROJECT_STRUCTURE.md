# EasyAdmin 目标目录结构

> 目标结构，不代表当前代码已存在。目录可按实施实际调整，但调整必须同步架构决策和依赖测试。
> 命名依据见 `DECISIONS.md` D18：Go 习惯、明确语义、保留约定短名。

```text
easyadmin/
├── cmd/
│   └── easyadmin/                 # 唯一组合根、CLI 和启动参数
├── internal/
│   └── archtest/                  # 架构守卫：依赖方向与裸 fetch 静态检查
├── kernel/                        # App、Module 契约、注册表、生命周期
│   ├── app/
│   ├── module/
│   └── buildinfo/
├── infra/                         # 对业务公开的统一基础设施入口
│   ├── conf/
│   ├── logx/
│   ├── db/
│   ├── credential/                 # AES-GCM、SSH 测试/别名/隧道
│   ├── httpapi/
│   ├── apperror/
│   ├── auth/
│   ├── audit/
│   ├── tasks/
│   ├── scheduler/
│   ├── storage/
│   └── internal/                  # easy 系列适配与基础设施私有实现
├── apps/                          # 按业务域组织的应用，供业务项目选用
│   ├── dashboard/                 # 工作台域
│   ├── system/                    # 系统管理域
│   │   ├── users/
│   │   ├── roles/
│   │   ├── config/
│   │   ├── logs/
│   │   ├── audit/
│   │   ├── tasks/
│   │   ├── schedules/
│   │   ├── files/
│   │   └── info/
│   └── profile/                   # 个人中心域
├── webui/
│   ├── index.html
│   ├── embed.go                    # 生产：go:embed index.html + assets
│   ├── embed_dev.go                # -tags dev：os.DirFS 读磁盘
│   ├── assets/
│   │   ├── core/                   # boot/router/layout/menu/api/notify/theme/store/perm/format/dom/data
│   │   ├── components/             # 表格/表单/筛选/弹窗/页头/空态/标签/日志/DSN/tooltip
│   │   ├── pages/                  # 页面入口（P3 起填充，当前为空）
│   │   └── styles/                 # tokens/base/layout/components
│   └── _test/                      # 浏览器测试面（仅 dev 构建可访问）
├── sql/
│   ├── sqlite/
│   ├── mysql/
│   └── postgres/
├── templates/
│   ├── embed.go                    # //go:embed app，模板随 CLI 二进制交付
│   └── app/                        # easyadmin new 的脚手架源
├── examples/
│   └── biz/                        # 唯一业务示例，直接构建运行
├── lib/                            # easy 系列最新源码 clone：easyconf/easydb/easyserver
├── docs/
├── go.mod
└── Makefile                        # 可选人类便捷命令，不作为唯一构建入口
```

## 1. apps 的领域组织

`apps/` 是物理目录，按业务域/限界上下文组织；`kernel.Module` 是注册契约，两者不是同一个概念。每个 app 或子域最终由一个 `kernel.Module` 实现登记，但不强制每个 app 都做成独立二进制。

- `kernel/app` 的 `App` 是单个运行时实例；`apps/` 是业务域目录，二者层级和职责不同。
- 一级目录对应业务域，如 `dashboard`、`system`、`profile`；业务项目直接增加 `apps/<domain>/`。
- 域内按子域或用例划分，如 `system/users`、`system/roles`，不按 `handler/`、`service/`、`repository/` 这类技术层建目录。
- app 内部默认保持 Go 式扁平文件；只有真实复杂度出现时才拆子包，不预先造 `domain/`、`application/`、`adapters/` 空壳。
- 跨 app 协作通过消费方定义的窄接口，由 `cmd` 组合根显式注入，不直接 import 其他 app 的内部包。

## 2. app 内部结构

```text
apps/system/users/
├── app.go                          # kernel.Module：ID、Manifest、Declare、Register
├── config.go                       # 配置结构和显式注册
├── model.go                        # 领域对象和 DTO
├── service.go                      # 业务规则和用例
├── repository.go                   # 数据访问边界与 SQL 实现
├── http.go                         # handler 和路由登记
├── sql/
│   ├── sqlite/
│   ├── mysql/
│   └── postgres/
└── app_test.go
```

前端页面入口统一放 `webui/assets/pages/<module-id>/index.js`，由 Manifest 的 Page ID 关联；不在 app 目录和 webui 各放一份。

## 3. 包组织规则

- `kernel` 只定义编排契约和生命周期，不包含业务模块或具体数据库实现。
- `infra` 各能力独立分包；`infra/internal` 隔离 easy 系列具体 API。
- `apps` 可独立关闭；跨 app 使用窄接口/组合根注入。
- `webui/assets/core` 只放前端框架能力；`components` 不知道业务 API；`pages` 放按路由动态加载的页面。
- 核心系统 SQL 与 app SQL 分开；app 删除时不遗留不可归属脚本。
- `internal/` 用于真正私有实现与不依赖业务清单的架构守卫；需要被业务项目导入的包必须明确为公开 API。跨模块契约测试放唯一能看见全部模块的组合根 `cmd/easyadmin`，并复用生产装配清单，不在测试里另写一份。
- 业务项目通过 `go.mod` 依赖版本化 EasyAdmin，不复制 `kernel`、`infra` 和内置 `apps`；生成与升级机制见 `SCAFFOLD_AND_UPGRADE.md`。

## 4. 运行目录和交付

运行数据目录通过显式选项确定，不依赖程序当前工作目录的偶然值。默认可采用 `./data`，但服务必须支持命令行/配置显式覆盖。日志、SQLite 文件、上传文件和缓存分别有明确子目录。

生产静态资源由 `go:embed` 内嵌；开发模式可从磁盘读取。SQL 脚本同样可内嵌，外挂覆盖必须有路径约束、校验和清晰的优先级。

## 5. 公开 API 与私有实现

业务项目可依赖的稳定包必须经过版本兼容策略管理。底层 easy 库类型原则上不向业务模块泄漏；`infra` 门面提供业务使用方需要的最小能力。具体包拆分由 P0/P1 以真实调用和测试为依据，不提前为每个函数创建包装层。

## 3. 当前仓库事实（与目标树分开）

当前已存在的主要基础设施源码：

- `infra/conf/`
- `infra/db/`（含 `datasource.go` 文件级数据源管理实现）
- `infra/credential/`
- `infra/tasks/`（当前为纯内存任务中心）
- `infra/scheduler/`（当前有 Store 接口和 `MemoryStore`，数据库 Store 未实现）
- `infra/logx/`（结构化日志唯一入口）
- `infra/apperror/`（稳定业务错误与 HTTP 状态映射）
- `infra/httpapi/`（路由/中间件/响应外壳/绑定唯一入口）
- `infra/auth/`（密码/会话令牌/CSRF/限流/授权唯一入口）
- `infra/audit/`（审计事件与 SQL 记录器）
- `sql/`（内嵌 SQL 结构脚本与 db.SQLSource）
- `apps/system/`（内置用户、角色、权限、会话与审计）

`cmd/easyadmin/`（组合根、用 infra/httpapi 组装 HTTP 面）与 `kernel/`（app、module）已建立最小骨架；P01 新增 `infra/logx`、`infra/apperror`、`infra/httpapi` 与最小内置模块 `apps/console`；P02 新增 `infra/auth`、`infra/audit`、`sql/` 与 `apps/system`；P03 新增 `webui/assets/core/pages.js` 核心页面注册、`webui/assets/pages/` 六个核心页面与 `httpapi.HandlerOptions.AppAssets` 业务资源接缝。`internal/archtest/` 提供依赖方向与裸 fetch 守卫；Manifest 与 `core/pages.js` 的双向一致性测试位于组合根 `cmd/easyadmin`，复用 `builtinModules` 的同一模块清单。当前尚不存在的目标目录包括 `templates/`、`examples/`。`webui/` 已落地基础资产（core/components/styles/pages/_test），页面按需动态加载。
