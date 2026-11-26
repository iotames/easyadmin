# 调度、凭据、服务器与数据源方案

> 目标：把 emailplugin 的调度器、凭据加密、SSH 隧道和命名数据源吸收到 `infra`，同时保留 rocksys 的简单运维体验。

## 1. 取舍

### 1.1 调度器：以 emailplugin 为主

emailplugin 的调度器比 rocksys 的“登记表 + 展示”更完整：

- 结构化 `Plan`（按分钟间隔）；
- `Def`/`Entry`/`Runner` 分离；
- 到期扫描、漏跑补一次、上轮未结束跳过；
- 任务提交到任务中心，不自己执行业务；
- 持久化 Store 与任务 Store 均实现后，可按 pending_task_id 对账；当前代码只有 MemoryStore，不具备进程重启后的持久接管；
- 调度名与任务 kind 同名，查询关系可逆。

rocksys 只保留两点：登记表用于统一展示，配置键作为启停唯一真源。

### 1.2 凭据与服务器：以 emailplugin 为主

emailplugin 已经具备服务器记录、AES-256-GCM 加密、私钥/密码认证、`~/.ssh/config` 别名、SSH 隧道和连接测试。rocksys 的数据源主要停留在 dsn.json 层面，缺少凭据加密和隧道。

结论：

- `infra/credential` 提供纯能力：加密、SSH 测试、别名解析、隧道；
- `apps/system/servers` 负责 `sys_server` 表、页面和权限；
- `infra/db` 负责数据源连接，不直接依赖 `apps`；
- 凭据密文与 DSN 密文都通过 `infra/credential` 加密。

### 1.3 数据源：融合两者

- Bootstrap 数据源：启动前就要用，放 `infra/conf`（`EASYADMIN_DB_DRIVER`、`EASYADMIN_DB_DSN`）。
- 运行期命名数据源可作为后续能力候选，支持只读/读写角色、SSH 隧道和脚本方言；当前尚未批准其 SQL 表模型。
- 当前 `infra/db/datasource.go` 使用 JSON 文件仓储；不得将其描述成 `sys_datasource` 已落库。密文存储、激活切换和迁移时机需另行完成设计评审。

## 2. infra 包边界

```text
infra/
├── credential/        # 已迁移：AES-GCM、SSH 测试、ssh config、隧道
├── conf/              # 已迁移：注册制配置、分组、热更
├── db/                # 已迁移：连接、Schema、Diff、DataSourceManager
├── scheduler/         # 已迁移核心 + Store 接口；目前仅 MemoryStore
├── tasks/             # 已迁移：任务中心、互斥规则
├── logx/              # 待迁移：日志 reader/query 接口
└── systeminfo/        # 待迁移：进程/CPU/内存采样
```

HTTP handler、页面、菜单、权限都放 `apps/system`，不进 `infra`。

## 3. 调度器设计

### 3.1 类型

```go
type Plan struct {
    EveryMin int
}

type Def struct {
    Name       string   // 小写字母+下划线；同时是 task kind
    Title      string
    EnabledKey string   // 配置键，启停唯一真源
    PlanKey    string   // 计划来源配置键；空则用 Plan
    Plan       Plan
    Remark     string
}

type Entry struct {
    Name, Title, ConfigKey, PlanKey, PlanDesc, Remark string
    Plan          Plan
    Enabled       bool
    LastStatus    string
    LastMessage   string
    LastTaskID    string
    LastRunAt     *time.Time
    NextRunAt     *time.Time
}

type Runner func(ctx context.Context, progress tasks.SetProgressFn) error

type ConfigLookup func(key string) (string, bool)

type Store interface {
    Sync(ctx context.Context, entries []Entry) error
    List(ctx context.Context) ([]Entry, error)
    MarkRunning(ctx context.Context, name, taskID string, at time.Time) error
    MarkFinished(ctx context.Context, name, status, message string, at time.Time) error
    AdoptPending(ctx context.Context) ([]Entry, error)
}

type Deps struct {
    Store     Store
    Tasks     *tasks.Center
    Config    ConfigLookup
    Location  *time.Location
    Now       func() time.Time
    ScanEvery func() time.Duration
}
```

### 3.2 行为和约束

- `Register(def, runner)`：名称唯一，`def.Name == runner.Kind`。
- `Start(ctx)`：立即 `ScanOnce`，之后按 `ScanEvery`（默认 5s）扫描。
- 到期判断：`NextRun` 严格晚于当前时刻；停机漏跑只补一次。
- 上轮任务仍在运行：本轮记 `skipped`，不堆积。
- `Submit` 到 `tasks.Center`，`CreatedBy = "scheduler:<调度名>"`，同源任务因此互斥。
- 重启接管目标：SQL Store 读取 `pending_task_id` 后核对任务仓储状态；当前任务中心也是内存态，进程重启后的任务记录需要由 P04 统一设计恢复语义，不能仅靠 schedule 表指针宣称可接管。
- 关闭：`Stop` 取消扫描并等待跟踪协程退出，不等待任务执行完毕（任务由任务中心收口或随进程退出）；不删除登记行。

### 3.3 表结构与当前实现边界

sys_schedule 的唯一字段口径见 DATA_DICT.md §4.3；本文不再复制第二份 DDL。关键约束：name 是主键；启停状态不落独立 enabled 列，而由 config_key 指向的配置项提供唯一真值；plan_key 为空时使用注册的 Plan；last_task_id 和 pending_task_id 不能混为一个字段。

当前源码只提供 Store 接口和 MemoryStore，尚无 SQL Store；数据库表属于 P04 待实施持久化契约，不代表现有调度器已具备重启后的持久恢复。

任务明细不重复存表，统一查 `infra/tasks` 的任务记录；调度表只保留上一轮结果和 pending 指针。

## 4. 凭据、服务器与数据源

本节 sys_server / sys_datasource 是待评审的数据模型候选；字段口径收录在 DATA_DICT.md §5，正式纳入某一期前须在对应 DESIGN_PLAN 中完成数据结构、权限、密钥轮换与 SSH 连接生命周期决策。

### 4.1 凭据

`infra/credential` 已迁移：

- `ParseKey/Encrypt/Decrypt/GenerateKey`：AES-256-GCM，密文带 `v1:` 版本前缀；
- `Validate/Resolve`：服务器输入校验和连接参数解析；
- `Test`：SSH 握手、认证、known_hosts 校验和错误分类；
- `LookupAlias`：`~/.ssh/config` 别名解析；
- `Tunnel`：经跳板机转发 TCP，数据库驱动直接使用 `DialContext`。

配置键：`EASYADMIN_CREDENTIAL_KEY`，必须是 base64 的 32 字节。密钥不写日志、不返回前端；更换密钥必须提供迁移或重新录入流程。

### 4.2 服务器表 `sys_server`

```sql
CREATE TABLE sys_server (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            VARCHAR(64) NOT NULL UNIQUE,
    connect_type    VARCHAR(16) NOT NULL, -- direct/config
    host            VARCHAR(255) NOT NULL DEFAULT '',
    port            INTEGER NOT NULL DEFAULT 22,
    username        VARCHAR(64) NOT NULL DEFAULT '',
    auth_type       VARCHAR(16) NOT NULL, -- key/password
    key_path        VARCHAR(500) NOT NULL DEFAULT '',
    password_enc    TEXT NOT NULL DEFAULT '',
    config_alias    VARCHAR(64) NOT NULL DEFAULT '',
    remark          VARCHAR(500) NOT NULL DEFAULT '',
    last_test_at    TIMESTAMP NULL,
    last_test_ok    BOOLEAN NOT NULL DEFAULT FALSE,
    last_test_error VARCHAR(500) NOT NULL DEFAULT '',
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
```

### 4.3 运行期命名数据源（候选，暂不纳入本期）

- 命名数据源 SQL 表和运行期切换属于候选能力，不属于当前 P02/P04 已确认范围。
- 现有 infra/db/datasource.go 是 JSON 文件仓储；不代表 sys_datasource 已建立。
- 候选字段、关系和安全约束见 DATA_DICT.md §5.2；是否需要 sys_datasource、何时从 JSON 迁移、如何保护 dsn_enc 和 active 唯一性，必须另立设计决策。
- 不得在 P02/P04 实施中未经确认直接创建或迁移该表。

## 5. 其他待迁移项

### 5.1 `infra/db` 结构同步增强

- 开发期不引入独立迁移引擎；复用 `TableSpec` + EnsureTables/DiffSchema 完成幂等建表与只读差异预览（见 `DECISIONS.md` D20）。
- 整体数据搬迁工具（Source→Target 批量 replace/upsert）属于数据迁移能力，须独立设计决策，不并入结构同步。
- HTTP handler、数据源选择、任务提交留在 `apps/system`。

### 5.2 `infra/db/size`

从 `dbsize.go` 抽：

- `TableStat`；
- 按方言查询表清单/行数/数据与索引占用；
- SQLite `dbstat` 懒计算 + 进程内 TTL 缓存；
- 不包含 HTTP handler。

### 5.3 `infra/logx`

先定义接口，不搬 handler：

```go
type Reader interface {
    Tail(ctx context.Context, limit int) ([]Line, error)
    Query(ctx context.Context, q Query) ([]Line, int64, error)
}
```

文件 reader、数据库 audit/task log reader 分别实现；HTTP 查询页放 `apps/system/logs`。

### 5.4 `infra/systeminfo`

从 `handlers_system.go` 抽纯采样：

- 进程运行时长、Goroutines、进程内存；
- Linux `/proc` CPU/内存采样；
- 非 Linux 字段返回 null；
- 不带 HTTP 和页面。

### 5.5 `infra/storage`

从 emailplugin 的 storage 抽：

- `Driver` 接口；
- 本地驱动；
- 原子写入、鉴权下载所需的元数据接口；
- qiniu 等厂商驱动放可选包，不进核心。

## 6. 实施顺序

1. `infra/credential`（已完成）。
2. `infra/scheduler`：Plan/Def/Entry/Runner + Store 接口 + 内存 Store 测试；再接 `sys_schedule`。
3. `apps/system/servers`：`sys_server` 表、CRUD、连接测试页面。
4. 命名数据源目录与 SQL 仓储暂不纳入本期；保留当前 JSON 仓储，待独立设计决策。
5. `infra/db` 结构同步增强（D20）；独立数据搬迁工具另行决策。
6. `infra/db/size`。
7. `infra/logx`、`infra/systeminfo`、`infra/storage`。

## 7. 验收

- 调度：注册、到期触发、漏跑策略、上轮未结束 skipped、重启收口与取消；进程重启后任务恢复遵守 interrupted + 新 attempt 语义。
- 凭据：加密往返、错误密钥、SSH 密码/私钥、known_hosts、别名解析。
- 数据源：直连、只读拒绝写、SSH 隧道、激活切换、DSN 密文不回显。
- Schema：三方言解析、Diff 分级、幂等建表。
- 迁移：表级进度、取消、目标库结构对齐。

```bash
go test ./infra/...
go vet ./infra/...
```
