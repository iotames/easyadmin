# EasyAdmin 目标目录结构

> 目标结构，不代表当前代码已存在。目录可按实施实际调整，但调整必须同步架构决策和依赖测试。
> 命名依据见 `DECISIONS.md` D18：Go 习惯、明确语义、保留约定短名。

```text
easyadmin/
├── cmd/
│   └── easyadmin/                 # 唯一组合根、CLI 和启动参数
├── kernel/                        # App、Module 契约、注册表、生命周期
│   ├── app/
│   ├── module/
│   └── buildinfo/
├── infra/                         # 对业务公开的统一基础设施入口
│   ├── conf/
│   ├── logx/
│   ├── db/
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
│   ├── embed.go
│   ├── embed_dev.go
│   ├── assets/
│   │   ├── core/
│   │   ├── components/
│   │   ├── pages/
│   │   └── styles/
│   └── _test/
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
- `internal/` 用于真正私有实现；需要被业务项目导入的包必须明确为公开 API。
- 业务项目通过 `go.mod` 依赖版本化 EasyAdmin，不复制 `kernel`、`infra` 和内置 `apps`；生成与升级机制见 `SCAFFOLD_AND_UPGRADE.md`。

## 4. 运行目录和交付

运行数据目录通过显式选项确定，不依赖程序当前工作目录的偶然值。默认可采用 `./data`，但服务必须支持命令行/配置显式覆盖。日志、SQLite 文件、上传文件和缓存分别有明确子目录。

生产静态资源由 `go:embed` 内嵌；开发模式可从磁盘读取。SQL 脚本同样可内嵌，外挂覆盖必须有路径约束、校验和清晰的优先级。

## 5. 公开 API 与私有实现

业务项目可依赖的稳定包必须经过版本兼容策略管理。底层 easy 库类型原则上不向业务模块泄漏；`infra` 门面提供业务使用方需要的最小能力。具体包拆分由 P0/P1 以真实调用和测试为依据，不提前为每个函数创建包装层。
