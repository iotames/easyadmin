# 日志与可观测性设计

## 1. 目标与边界

`infra/logx` 是唯一日志入口。负责统一字段、上下文、脱敏、输出策略和资源生命周期；不在 EasyAdmin 内重做完整日志分析平台。底层优先复用 `easyserver/log` 已有能力，并以 `log/slog` 的结构化语义作为对上设计参照；是否能直接桥接需 P1 实测。

## 2. 日志语义分类

| 类别 | 用途 | 主要存储 | 是否审计凭证 |
|---|---|---|---|
| Application Log | 程序流程、错误、依赖故障 | stdout/文件 | 否 |
| Access Log | HTTP 请求、状态、耗时、request_id | stdout/文件或受控索引 | 否 |
| Audit Log | 谁在何时对什么做了什么、结果 | 独立数据库表/追加存储 | 是 |
| Task Log | 任务阶段、进度、重试和错误 | 与 Task 关联的持久化记录 | 视操作风险 |

一条普通 Info 日志不能替代 Audit Event。四类日志不可仅因都叫“日志”而合并成同一种存储和保留策略。

## 3. 公共字段

适用字段按类别选用：`timestamp`（UTC）、`level`、`service`、`version`、`module`、`request_id`、`trace_id`、`user_id`、`task_id`、`message`、`attrs`、`error`。字段名和类型统一，属性大小和数量有限制。

## 4. 调用约定

```go
logger.Info(ctx, "module started", logx.String("module", moduleID))
logger.Error(ctx, "database query failed", logx.Err(err))
```

以上为风格示意，不是已冻结签名。业务模块不得自行创建底层 writer。Logger 接口需可被测试替换。高频路径避免昂贵字段构建；敏感属性在统一入口脱敏。

## 5. 输出与生命周期

- 默认控制台可用；文件输出可选，需轮转、保留和磁盘满策略。
- 容器环境可将结构化日志写 stdout，由外部采集。
- logger 属于 App 资源，停机按 deadline flush/close。
- 输出故障不得递归调用同一 logger 报错；使用受控 fallback 或健康状态。
- 不输出 Cookie、Authorization、密码、DSN、私钥、Token、敏感 SQL 参数。

## 6. Access Log

由统一 HTTP 中间件完成请求级记录：request_id、method、路由模板、status、duration、用户 ID（已认证时）、可信来源 IP、受限长度的 UA。只有配置了可信代理范围才解析 Forwarded/X-Forwarded-For。健康探针和静态资源可降噪，但不能破坏 request_id 关联。

## 7. Audit Log

事件至少包含 actor、action、resource type/id、outcome、summary、request_id、时间和脱敏 detail。必须审计登录/登出、用户/角色/权限变更、配置修改、任务危险操作、文件删除和高风险管理动作。

审计失败策略按操作风险声明：高风险操作可要求审计先写成功再提交；低风险动作可降级但必须报警。审计表只允许受控追加与保留期清理；普通用户不可修改/删除。

## 8. 任务日志与保留

Task Log 以 task ID + 序号关联，进度和日志写入须节流，不能每个循环都高频写库。任务状态记录与应用日志分离。日志清理任务本身要审计，保留策略按数据类别区分。

## 9. 指标与健康

`/healthz` 只表示进程存活；`/readyz` 检查关键依赖就绪。可采集请求量、延迟分布、模块状态、数据源状态、任务积压与失败数。指标标签禁止 request_id、user_id、任意 URL 参数等高基数字段。

## 10. 日志中心的可实现边界

若 Application Log 只写轮转文件，系统管理页必须通过受控 reader 进行有界查询，或只提供下载/尾部查看；不能设计一个查询 API 却没有索引数据来源。Audit Log/Task Log 可以数据库分页查询。Access Log 是否入库需根据真实容量和查询需求评审，不能默认全部写入主数据库。

## 11. easyserver/log 核验

`lib/easyserver` 已提供最新源码，但源码存在不等于日志语义已认证。P1 必须核实结构化字段、并发写、文件轮转、同步/异步行为、Flush/Close 和错误处理。
