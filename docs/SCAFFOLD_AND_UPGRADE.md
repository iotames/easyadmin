# 脚手架与独立升级设计

> 目标：业务项目只写业务，EasyAdmin 的基础设施和通用组件只维护一份，升级后所有项目共同受益。

## 1. 边界

- `templates/app/`：`easyadmin new` 的脚手架源，参数化、最小化，不直接编译。
- `examples/biz/`：唯一业务示例，由模板生成后扩展，直接构建运行，用于公开 API 和端到端回归。
- 不保留 `examples/minimal/`。模板的 golden 测试已经覆盖“最小项目可构建、可启动、可登录”，再放一份最小示例只会重复维护。
- 业务项目生成在用户目录，不把 `kernel`、`infra`、内置 `apps` 复制进业务仓库。

## 2. templates/app 完整文件树

```text
templates/
├── embed.go                       # package templates; //go:embed app
└── app/
    ├── go.mod.tmpl
    ├── .gitignore.tmpl
    ├── README.md.tmpl
    ├── Makefile.tmpl
    ├── cmd/
    │   └── {{.AppName}}/
    │       └── main.go.tmpl
    ├── apps/
    │   └── README.md.tmpl         # 业务域放这里，初始为空
    ├── webui/
    │   ├── embed.go.tmpl          # //go:embed pages.js assets
    │   ├── pages.js.tmpl          # 业务页面入口表，初始为空
    │   └── assets/
    │       └── styles/
    │           └── app.css.tmpl   # 业务样式覆盖入口
    └── config/
        └── app.env.example.tmpl
```

模板变量只保留必要项：

| 变量 | 用途 |
|---|---|
| `{{.AppName}}` | 应用名、二进制名、默认数据库文件名 |
| `{{.ModulePath}}` | 生成项目的 Go module path |
| `{{.EasyAdminModule}}` | EasyAdmin 的 Go module path |
| `{{.EasyAdminVersion}}` | 生成时绑定的 EasyAdmin 版本 |
| `{{.GoVersion}}` | 生成项目的 `go` 指令版本 |

模板中不出现业务表、业务页面和示例 CRUD。业务内容由 `easyadmin module add <name>` 生成。

## 3. easyadmin new 的机制

```text
解析参数
  → 校验目标目录
  → 在临时目录渲染模板
  → 替换模板变量和文件名
  → 生成 go.mod 并绑定 EasyAdmin 版本
  → go mod tidy / go build ./...
  → 原子移动到目标目录
  → 输出下一步命令
```

关键规则：

1. 模板由 `templates` 包通过 `go:embed` 编入 CLI，`easyadmin` 单二进制即可生成项目。
2. 目标目录已存在且非空时拒绝生成，不覆盖已有业务文件。
3. 全部写入临时目录，构建成功后再原子替换；失败删除临时目录，不留半成品。
4. 模板版本与 CLI 版本一致，`go.mod` 写入同一个 EasyAdmin 版本。
5. 网络不可用时可用 `--no-build` 生成，再由开发者手动 `go mod tidy`；默认仍然执行构建校验。
6. `new` 只创建项目，不修改已有项目；升级现有项目走 Go module 版本更新，不重新生成业务文件。

## 4. 生成后的项目结构

```text
myshop/
├── go.mod
├── go.sum
├── cmd/myshop/main.go            # 唯一组合根
├── apps/README.md                # 业务域目录，初始为空
├── webui/
│   ├── embed.go                  # 业务页面和样式
│   ├── pages.js                  # Page ID → 业务页面入口
│   └── assets/styles/app.css
├── config/app.env.example
├── Makefile
├── README.md
├── .gitignore
└── data/                         # 运行时生成，已忽略
```

`easyadmin module add orders` 后增加：

```text
apps/orders/
├── app.go
├── config.go
├── model.go
├── service.go
├── repository.go
├── http.go
├── sql/{sqlite,mysql,postgres}/001_init.sql
└── app_test.go

webui/assets/pages/orders/index.js
```

`webui/pages.js` 同步增加入口：

```js
export default {
  'orders': () => import('./assets/pages/orders/index.js'),
};
```

app 内部默认扁平，不预建 `domain/`、`application/`、`adapters/` 空壳；只有业务复杂到需要拆包时才拆。

组合根只做装配，示意如下（具体 API 在 P1/P5 冻结）：

```go
package main

import (
    "context"
    "log"

    "github.com/iotames/easyadmin"
    "github.com/iotames/easyadmin/apps/dashboard"
    "github.com/iotames/easyadmin/apps/profile"
    "github.com/iotames/easyadmin/apps/system"
    projectweb "{{.ModulePath}}/webui"
)

func main() {
    app := easyadmin.New()
    app.Use(
        dashboard.Module{},
        system.Users{},
        system.Roles{},
        system.Config{},
        system.Logs{},
        system.Audit{},
        system.Tasks{},
        system.Schedules{},
        system.Files{},
        system.Info{},
        profile.Module{},
    )
    app.WithWebAssets(projectweb.Assets)  // 业务页面与样式
    if err := app.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## 5. 独立演进机制

### 5.1 单一版本化框架

EasyAdmin 是独立 Go module。业务项目只通过 `go.mod` 依赖它：

```text
业务项目
  └── import github.com/iotames/easyadmin/...   // 只 import 公开包
        └── kernel / infra/* / apps 内置模块 / webui 契约
              └── lib/easyconf / lib/easydb / lib/easyserver
```

当前 module path 约定：

| 模块 | module path |
|---|---|
| EasyAdmin | `github.com/iotames/easyadmin` |
| 配置 | `github.com/iotames/easyconf` |
| 数据 | `github.com/iotames/easydb` |
| HTTP 服务 | `github.com/iotames/easyserver` |

本地开发时 `examples/biz` 可用 `replace` 指向仓库内副本；生成到用户目录的项目使用上述正式 module path，不写本地 `replace`。

- 业务仓库不复制 `kernel`、`infra`、内置 `apps`。
- 升级通过 `go get` 或后续 `easyadmin upgrade` 完成，所有项目拿到同一份修复。
- `infra/internal`、框架内部包、各 app 的 `internal` 不是公开 API。
- 框架仓库内的 `examples/biz` 可以用 `go.work` 或 `replace` 指向本地 EasyAdmin；生成到用户目录的项目不加 `replace`，始终依赖正式版本。
- v0.x 阶段业务项目精确锁定版本；v1.0 后按语义化版本升级。P5 先提供 `doctor` 和升级文档，自动 `upgrade` 命令在 P6 有真实需求后再做，避免提前造复杂迁移器。

### 5.2 公开契约

稳定面只保留这些：

- `kernel` 的 App、Module、Manifest、Registrar、生命周期契约；
- `infra/*` 各能力门面：`conf`、`logx`、`db`、`httpapi`、`apperror`、`auth`、`audit`、`tasks`、`scheduler`、`storage`；
- 内置 `apps` 的模块构造函数和显式选项；
- `webui` 的 `ctx`、`api`、`notify`、通用组件契约和主题变量；
- `templates` 的生成契约。

这些契约遵守“只增不改”：新增字段、端点、可选参数和枚举可以；删除、重命名、改类型、改语义不可以。

### 5.3 升级与迁移分域

核心迁移和业务迁移分账管理：

```text
schema_migrations(scope, version, checksum, applied_at)
```

- `scope = core`：EasyAdmin 核心和内置 app 的迁移，随框架版本演进。
- `scope = app:<module-id>`：业务 app 的迁移，归业务项目所有。
- 启动时先应用 `core`，再按 app 依赖顺序应用业务迁移。
- 框架升级只新增/执行 `core` 迁移；业务迁移文件不被改写。

核心表使用 `sys_` 前缀；业务表使用自己的前缀，避免命名冲突。

### 5.4 配置分域

- 核心配置键使用框架命名空间，业务键使用 app 命名空间，如 `orders.*`。
- 新版本增加配置项时提供默认值，不要求业务项目同步改配置。
- 业务覆盖只写自己的键；不复制整份核心默认配置。

### 5.5 前端分域

```text
/assets/        EasyAdmin core 资源，随框架升级
/assets/app/    业务 webui 资源，由业务项目 embed 和提供
```

- `webui/pages.js` 只登记业务 Page ID，启动时与 core 路由表合并。
- boot 启动时尝试加载 `/assets/app/pages.js`；文件不存在时跳过，适用于没有业务页面的最小项目。
- Page ID 冲突在启动期失败，不静默覆盖。
- 业务只使用稳定的 `ctx`、`api`、`notify`、组件和 CSS 变量。
- 通用组件的修复随 EasyAdmin 升级进入所有项目；业务页面文件不被改写。

### 5.6 定制优先，不改框架

业务差异按以下顺序解决：

1. 配置项；
2. 内置 app 的构造函数选项；
3. 消费方定义的窄接口和组合根注入；
4. 追加注册菜单、页面、路由和任务；
5. 提交到 EasyAdmin 上游，或作为独立业务 app 实现。

禁止 fork 内置模块、复制 `infra` 代码或长期维护 vendor 补丁；这会让升级收益无法回流。

### 5.7 本地联调与 replace 边界

三种可用方式，按项目选择：

| 场景 | 做法 | 结论 |
|---|---|---|
| 正式依赖 | `require github.com/iotames/easyadmin vX.Y.Z`，不加 `replace` | 适合直接使用发布版本 |
| 项目固定框架提交 | `git submodule` + `replace => ./lib/easyadmin` | 与 rocksys 一致；提交可追踪，升级需主动前移 |
| 本地临时联调 | `go.work` 的 `use` 指向共享 checkout | 可选，不强制 |

`replace github.com/iotames/easyadmin => ./lib/easyadmin` 推荐与 git submodule 配对使用：

```bash
git submodule add https://github.com/iotames/easyadmin lib/easyadmin
git submodule update --init --recursive
go mod edit -replace github.com/iotames/easyadmin=./lib/easyadmin
go mod tidy
```

父仓库的 index 记录 submodule 的精确提交哈希，构建可复现。升级就是前移 submodule：

```bash
cd lib/easyadmin
git fetch origin
git checkout vX.Y.Z
cd ../..
go mod tidy
git add .gitmodules lib/easyadmin go.mod go.sum
git commit -m "chore: 前移 easyadmin 至 vX.Y.Z"
```

必须明确：

- 它绕过版本选择，构建的是 `./lib/easyadmin` 当前提交，不是 `require` 里的版本号；
- submodule 必须指向官方仓库，不在这里做长期私有修改；通用修复要回流 EasyAdmin；
- 每个项目固定自己的提交，升级需要主动前移，不会自动跟随最新；
- 被 replace 的目录必须有自己的 `go.mod`，且 `module` 声明必须是 `github.com/iotames/easyadmin`；
- 父仓库必须提交 `.gitmodules`，否则新克隆无法执行 `git submodule update --init`。

`go.work` 只是可选的本地联调方式，不是 submodule + replace 的前置条件。

CLI 版本也要对齐：

- 本地开发：在 EasyAdmin checkout 中执行 `go build -o bin/easyadmin ./cmd/easyadmin`；使用 submodule 时在 `lib/easyadmin` 内构建，确保 CLI 与业务实际使用的源码一致；
- 正式使用：`go install github.com/iotames/easyadmin/cmd/easyadmin@vX.Y.Z`；
- 不要用最新开发版 CLI 给锁定旧版库的项目生成代码，否则模板 API 可能不匹配；
- `easyadmin doctor` 应校验 CLI 版本、项目 `go.mod` 版本和模板版本是否兼容。

## 6. 验证与发布

- 模板 golden test：渲染到临时目录，执行 `go build ./...`、启动、登录。
- `examples/biz`：直接构建运行，覆盖一个真实业务域、迁移、页面和权限。
- 兼容性测试：固定旧版 API fixture、JS `ctx`/组件契约、Manifest Page ID。
- 升级测试：用上一版本生成项目，升级到当前版本后构建、启动、执行核心迁移，确认业务迁移未变。
- `easyadmin doctor`：检查 Go 版本、框架版本、模板版本、迁移状态和配置缺失。

发布顺序：先冻结公开契约与迁移 checksum，再发 EasyAdmin 版本，最后更新 `templates/app` 和 `examples/biz` 的绑定版本。

## 7. 日常开发工作流

业务项目日常只改这些地方：

- `apps/<domain>/`：模型、业务规则、SQL、HTTP；
- `webui/`：业务页面和样式；
- `config/`：业务配置；
- `cmd/<app>/main.go`：只在增删内置 app、接入新 Provider 时改。

基础设施改进走 EasyAdmin 仓库：

1. 在 EasyAdmin 中修 `infra`、kernel、内置 `apps` 或 core 前端组件；
2. 补对应单元、集成或浏览器测试；
3. 发布新版本；
4. 业务项目 `go get` 升级并跑 `easyadmin doctor`。

升级检查顺序：读变更说明 → 更新 `go.mod` → 启动应用应用 `core` 迁移 → 跑测试和 `doctor` → 灰度验证业务页面。业务迁移文件、业务页面和业务配置不在升级中自动改写。

这套流程的核心约束只有一句：**业务项目可以依赖 EasyAdmin，但不能拥有一份分叉的 EasyAdmin。**
