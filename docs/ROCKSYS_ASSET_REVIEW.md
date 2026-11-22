# RockSys 与参考项目资产审查

> 本文把已检查的源码事实、由调用方式推导的设计建议、尚待验证的参考项目内容分开。不得把候选设计写成源码已证实能力。

## 1. 审查范围与事实边界

- 已检查：当前工作区的 `rocksys/`、`emailplugin/`、`kitbus/` 三个项目源码，以及 `rocksys/`、`emailplugin/`、`kitbus/lib/` 下的 easy 系列目录。
- 可检查：RockSys 顶层 Go 代码、模块组织、README/ARCHITECTURE/AGENTS、WebUI 和对 easy 系列的调用位置；EmailPlugin 的任务、调度、Schema、存储和 HTTP 骨架；KitBus 的 `infra/internal` 分层与前端应用骨架。
- 已核对：`easyconf` 的注册式配置与默认值快照、`easydb` 的查询/执行/事务封装、`easyserver` 的 HTTP Server/Context/Middleware/Handler 接口。
- 仍需认证：easy 系列的内部并发、关闭、热更、驱动兼容与故障恢复语义不能只靠调用方和静态阅读断言，P1 必须用锁定版本跑测试。
- 证据边界：本文区分“源码已核对的结构与接口”与“仍需运行验证的行为”，不把候选设计写成已证实能力。

## 2. RockSys easy 系列实际使用位置

RockSys `go.mod` 中声明的版本及调用方模式（以 `lib/` 下最新 clone 核对）：

| 库 | RockSys 声明版本 | 调用侧观察 |
|---|---|---|
| easyserver | v1.8.0 | HTTP Server/Context/Handler 注册、优雅停机、日志能力 |
| easyconf | v1.2.6 | 注册配置、获取当前值、Watch/Set 等模式 |
| easydb | v0.7.0 | EasyDb、database/sql 适配和 DSN 相关调用 |

这张表仅表示 RockSys 的依赖声明及调用模式，不等于这些库的完整能力保证。

## 3. 可迁移资产

| RockSys 来源 | 原项目用途 | EasyAdmin 建议归属 | 处理方式 |
|---|---|---|---|
| `internal/conf` | 配置集中注册、默认值、密钥和热更 | `infra/conf` | 保留注册制和快照思想，剥离网关专属字段，明确配置来源 |
| `internal/db` | 数据源、SQL 脚本、方言和结构处理 | `infra/db` | 保留 SQL-first、外挂优先/内嵌兜底；迁移执行需单独治理 |
| `internal/adminapi` | 管理 API、认证、静态资源、统一响应 | `infra/httpapi` + `infra/auth` + 系统模块 | 拆分基础 HTTP 能力和管理业务 |
| `internal/taskcenter` | 任务统一注册、状态和执行观测 | `infra/tasks` | 评估持久化、重启接管、取消和保留期，不照搬业务字段 |
| `internal/workpool` | 受控并发执行 | `infra/tasks` 内部实现候选 | 先测负载、取消和关闭，不另建第二套任务入口 |
| `internal/hotswap` | 网关脚本热切换 | SQL/静态资源覆盖机制候选 | 只保留受控外挂优先，不把网关热插拔概念扩散到模块运行时 |
| `catalog` | 组件/服务元信息 | `kernel.Manifest` | 合并进模块元信息，不另造平行目录系统 |
| `webui` | 布局、菜单、提示、表格、主题和状态 | `webui/core` + `components` | 保留交互经验，组织为原生 ESM 模块 |
| `-tags dev` / embed | 开发文件读取和生产嵌入 | WebUI/SQL 资源交付 | 继承双模式并补齐一致性测试 |
| 工程文档/Agent 规则 | 阶段、架构和工程约束 | `AGENTS.md` + 成对计划 | 继承约束治理，避免计划文件取代正式契约 |

## 4. 不应进入 EasyAdmin 内核的 RockSys 能力

反向代理、上游转发、WAF/Shield、GeoIP、Lua 策略、网关流量调度、网关限流和插件脚本等属于 RockSys 产品能力，不是通用中后台基础设施。将来若有业务项目需要，应作为独立业务模块或 Provider，而非内核依赖。

## 5. EmailPlugin / KitBus 的源码核对结论

按当前工作区源码，可确认的参考能力包括：

- EmailPlugin：持久任务、父子任务、进度与日志、任务取消/重试/接管、调度扫描、Storage 接口、Schema 迁移。
- KitBus：`infra/` 与 `internal/` 分层、easy 系列适配、SQL 外置、静态资源和 WebUI 布局/菜单边界。

**复用边界**：上述能力可以作为模型蓝本，但不能直接复制业务字段和耦合关系。任务/调度进入 `infra/tasks`、`infra/scheduler` 前，必须按 EasyAdmin 的状态机、并发语义和测试要求重新实现；EmailPlugin 的邮件语义、KitBus 的容器语义不进入框架。

## 6. 从 RockSys 迁移时必须重新设计的部分

1. 网关的热插拔生命周期不等于普通后台模块生命周期。EasyAdmin 首期以进程启动时显式装配为准。
2. 网关配置项可能有代理专属语义；EasyAdmin 需要配置来源、运行设置和权限管理的独立模型。
3. 管理 API 的鉴权模式不能原样照搬；默认使用服务端会话和 CSRF。
4. 日志、审计、任务日志必须分离；普通运行日志不能替代审计。
5. 参考项目中的组件不能仅凭“存在”就进入框架；必须满足可复用、唯一入口、边界稳定、测试可覆盖。

## 7. P1/P2/P4 核验清单

- easyserver：标准 Handler 接入、Context 生命周期、路由冲突、超时、优雅关闭、错误传播。
- easyconf：注册重复行为、取值优先级、默认值快照、Watch 并发、热更失败、文件写回和关闭。
- easydb：驱动注册、事务隔离、扫描结果、批量写入、方言、连接池和关闭。
- easyserver/log：结构化字段、并发安全、文件轮转、flush/close、写失败行为。
- 任务/调度候选来源：持久化状态机、取消、重启接管、并发语义、可控时钟测试。

核验结果必须保存为测试、源码引用和决策记录，不能只写“兼容”。
