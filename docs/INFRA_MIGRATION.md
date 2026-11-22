# Rocksys / EmailPlugin 通用能力迁移清单

> 原则：直接依赖 easy 系列；rocksys/emailplugin 只迁移业务无关部分，业务表名、网关字段、邮件/容器语义不进入 `infra`。

## 1. 已迁移

| 目标包 | 来源 | 迁移方式 | 说明 |
|---|---|---|---|
| `infra/conf` | `rocksys/internal/conf` | 复制后通用化 | 保留注册制、默认值快照、来源优先级、热更、`List()`；环境变量前缀改为 `EASYADMIN_`；移除 upstream/timeout/WWWRoot/404 等网关字段；新增 `RegisterGroup`，配置项带 `group`/`order`，供前端按前缀分组 |
| `infra/db` | `rocksys/internal/db` | 复制通用文件 | 保留 `TableSpec`、DDL 解析、多语句拆分、Catalog 查询、结构 Diff、安全 SQL 生成、幂等建表；移除 `shield_event`/`access_log`/`geoip_list` 等业务表常量 |
| `infra/db/datasource.go` | `rocksys/internal/adminapi/dsn.go` + `easydb/dsn` | 抽象为仓储 | 保留命名数据源、DSN 去重、激活项和 JSON 持久化；HTTP handler 留给 `apps/system` |
| `infra/tasks` | `rocksys/internal/taskcenter` | 原样复制 | 保留来源白名单、`MutexTaskField`/`MutexList`/`MutexMap` 三件套、取消、终态保留、panic 收口；包名改为 `tasks` |
| `infra/credential` | `emailplugin/internal/credential` | 原样复制 | AES-256-GCM、SSH 测试、`~/.ssh/config` 别名、SSH 隧道；记录存储与页面留给 `apps/system` |
| `infra/scheduler` | `emailplugin/internal/schedule` | 重写核心 | 保留 `Plan/Def/Entry/Runner`、到期扫描、漏跑补一次、上轮未结束跳过、重启接管；依赖改为 `infra/tasks` 和 `Store` 接口；当前提供内存 Store |

## 2. 待迁移

| 目标包 | 来源 | 原因 |
|---|---|---|
| `infra/db/migrate` | `rocksys/internal/adminapi/migrate.go` | 表迁移引擎与 HTTP handler、AdminServer、任务中心耦合，需先拆核心执行器 |
| `infra/db/size` | `rocksys/internal/adminapi/dbsize.go` | 表空间统计可迁移，先抽方言查询和缓存 |
| `infra/scheduler` 持久化 Store | `emailplugin/internal/schedule` + `sys_schedule` | 核心已迁移，`sys_schedule` Store 待接 |
| `infra/logx` 查询 | `rocksys/internal/adminapi/handlers_log.go`、`execlogstore.go` | 日志查询/索引依赖 HTTP 层，需先定义 reader 接口 |
| 系统信息 | `rocksys/internal/adminapi/handlers_system.go` | 进程/CPU/内存采样较通用，可后续迁移 |
| 存储 | `emailplugin/internal/storage` | 接口与本地驱动可迁移，qiniu 驱动不进入核心 |

## 3. 不迁移

- rocksys 的代理、WAF、GeoIP 同步、Lua 热切换、网关限流；
- emailplugin 的邮件 Graph、归档编排、ERP 同步；
- kitbus 的容器引擎探测、镜像/实例管理；
- 各项目的 HTTP handler 和页面：核心能力进 `infra`，后台页面进 `apps/system`。

## 4. 验证

```bash
go test ./infra/...
go vet ./infra/...
```

当前 `infra/conf`、`infra/db`、`infra/tasks`、`infra/credential`、`infra/scheduler` 已通过。
