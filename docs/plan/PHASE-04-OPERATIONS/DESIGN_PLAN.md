# P04 设计计划：运营能力

> 状态：PROPOSED（待用户评审；未批准前不得进入本期实施）。

## 1. OGSM 与目标

**Objective**：在现有内存任务中心和调度核心之上，提供可审计、可恢复、可运维的单实例后台任务与运营能力。

**Goals**：任务数据可持久化查询；进程重启后任务状态不丢失且有确定收口语义；调度配置与任务记录一致；文件访问安全；运维页面有明确权限和数据保留策略。

**Strategies**：复用现有 tasks.Center 执行/并发机制与 scheduler.Scheduler 核心；新增持久化 Store/Repository；通过注册 kind 的可重放执行器做安全恢复，不尝试持久化 Go 闭包；把 Scheduler 限定为触发器。

**Measures**：任务状态转换有状态机测试；重复触发和 pending_task_id 一致性可验证；重启后的 running 记录全部收口；文件路径/内容/权限测试通过；所有查询有分页和时间范围上限。

## 2. 现状结论（源码证据）

| 当前资产 | 证据 | 结论 |
|---|---|---|
| 任务执行与并发 | infra/tasks/taskcenter.go | 当前纯内存；不能持久化历史、进度或跨进程执行闭包 |
| 调度核心 | infra/scheduler/scheduler.go | 已有 Def/Entry/Runner、Store 接口、MemoryStore；SQL Store 未实现 |
| 调度状态 | infra/scheduler/scheduler.go 的 Entry/Store | 有 LastTaskID 与 PendingTaskID，数据库结构必须分别保留 |
| 凭据能力 | infra/credential/credential.go | AES-GCM、SSH 测试/别名/隧道已存在；服务器管理存储/UI 尚未交付 |
| 数据库基础 | infra/db/ | 可复用连接、DDL/Catalog/Diff 等能力；本期要核验而非重写 DB 管理器 |
| 存储 | 当前仓库尚无 infra/storage | 本期新建本地文件驱动与受控对象元数据 |

## 3. 本期范围

- Task Center：持久化任务、进度、日志、取消、重试、任务清理与重启收口。
- Scheduler：复用现有扫描/派发核心，实现 SQL Store；调度器只提交任务。
- 本地 Storage：原子写入、哈希去重、对象元数据、鉴权下载、软删除和清理。
- 运维 API/UI：任务、调度、文件、应用日志/访问日志/审计日志查询与系统诊断。
- 使用 infra/credential 支持的 SSH 凭据能力建设服务器记录管理；密钥密文持久化，页面不得读取明文私钥。
- 所有能力以单实例部署为基线，不承诺多实例协调和集群抢占。

## 4. 明确不做

- 分布式调度、集群 leader、跨实例租约/抢占。
- S3/OSS 等厂商强依赖；本期只交付 local storage。
- 完整日志分析平台、外部监控后端和复杂告警编排。
- 任意进程内任务闭包的崩溃后续跑；不能将 Go 函数/闭包序列化。
- sys_datasource 运行时 SQL 数据源目录；当前 DataSourceManager 是 JSON 文件仓储，该能力另行设计。
- 多租户、分布式任务队列与跨节点执行。

## 5. 架构与契约决策

- 任务执行仍由现有 Center 管理；持久化层保存状态，不复制第二个 Task Center。
- 任务以稳定 kind + payload + 注册 Handler 描述可重试/可重放操作；不能只存闭包。
- 任务提交/执行/完成状态必须由单一状态机维护；存储失败不能静默丢失任务。
- 当前 tasks.Center 主要状态与目标持久化状态需显式映射：现有 done 映射为对外 success；重启遗留 running 记录收口为 interrupted；pending、skipped 等状态只有在执行队列/调度语义实现后才可产生。
- 重启恢复不等于从函数中间位置续跑：未完成任务先标为 interrupted；仅注册为幂等且允许重试的 kind 可以重新创建执行尝试，生成新的 task_no，attempt_no 递增，retry_of_no 指向前一次尝试。
- Scheduler 使用唯一 sys_schedule 数据模型；config_key 是启停唯一真源，不额外加 enabled 列。
- Scheduler 的 pending_task_id 必须引用真实可查询的任务记录；Task Store 和 Scheduler Store 的更新顺序、事务边界及孤儿清理策略必须在实现设计中明确。
- 文件元数据与文件内容分离；对象 key 由服务端产生；所有下载经过权限检查。
- 日志语义区分 Application、Access、Audit、Task；没有真实 reader/index 的日志类型不得展示“可查询”页面。

## 6. 数据结构（本期独立设计契约）

完整字段清单与通用类型见 DATA_DICT.md。本章锁定本期数据关系、核心字段、约束、SQL 逻辑和生命周期；正式评审前须保证与数据字典一致。

### 6.1 sys_task

| 字段 | 类型/约束 | 含义 |
|---|---|---|
| id | BIGINT PK | 内部主键 |
| task_no | VARCHAR(64) UNIQUE NOT NULL | 稳定任务号 |
| parent_no | VARCHAR(64) NULL | 业务父任务号，最多两级 |
| attempt_no | INTEGER NOT NULL DEFAULT 1 | 执行尝试序号 |
| retry_of_no | VARCHAR(64) NULL | 前一次尝试 task_no |
| kind | VARCHAR(64) NOT NULL | 注册任务类型 |
| title | VARCHAR(255) NOT NULL | 展示标题 |
| status | VARCHAR(16) NOT NULL | pending/running/success/failed/cancelled/skipped/interrupted |
| progress_current/total | BIGINT NOT NULL DEFAULT 0 | 当前/总量，total=0 表示未知 |
| result_text | TEXT NOT NULL DEFAULT '' | 人类可读结论 |
| error_code/error_text | VARCHAR(96)/TEXT | 失败信息 |
| data_bytes | BIGINT NOT NULL DEFAULT 0 | 处理数据量 |
| created_by | VARCHAR(64) NOT NULL | 用户、scheduler 或 system |
| started_at/finished_at | TIMESTAMP NULL | 执行区间 |
| created_at/updated_at | TIMESTAMP NOT NULL | 记录时间 |

约束：task_no 唯一；父子关系用 parent_no 关联；按 kind/status、parent_no、created_at 建索引。不得清理 pending/running；终态按保留策略清理。任务记录只保存可审计 payload 的引用/脱敏摘要，不保存秘密。

### 6.2 sys_task_log

| 字段 | 类型/约束 | 含义 |
|---|---|---|
| id | BIGINT PK | 日志主键 |
| task_no | VARCHAR(64) NOT NULL | 所属任务号 |
| seq | BIGINT NOT NULL | 任务内递增序号 |
| level | VARCHAR(8) NOT NULL | debug/info/warn/error |
| message | TEXT NOT NULL | 日志消息 |
| attrs_json | JSON NULL | 脱敏结构化字段 |
| created_at | TIMESTAMP NOT NULL | 时间 |

约束：UNIQUE(task_no, seq)，FK 到任务号；按 task_no/seq 和 created_at 索引。任务日志不得记录密码、令牌、DSN 或凭据明文。

### 6.3 sys_schedule

| 字段 | 类型/约束 | 含义 |
|---|---|---|
| name | VARCHAR(64) PK | 调度项稳定名 |
| title | VARCHAR(128) NOT NULL | 展示名 |
| config_key | VARCHAR(128) NOT NULL | 启停配置键，唯一真源 |
| plan_key | VARCHAR(128) NOT NULL DEFAULT '' | 周期配置键；空则用注册 Plan |
| plan_every_min | INTEGER NOT NULL | 当前生效周期 |
| remark | VARCHAR(500) NOT NULL DEFAULT '' | 说明 |
| last_status/last_message | VARCHAR(16)/VARCHAR(255) | 最近结果 |
| last_task_id | VARCHAR(64) NOT NULL DEFAULT '' | 最近派发任务 |
| pending_task_id | VARCHAR(64) NOT NULL DEFAULT '' | 当前跟踪任务 |
| last_run_at/next_run_at | TIMESTAMP NULL | 时间状态 |
| updated_at | TIMESTAMP NOT NULL | 更新时间 |

约束：不创建 id 或 enabled 列；config_key 是启停唯一来源；last_task_id 与 pending_task_id 语义不同。SQL 唯一口径以 DATA_DICT.md §4.3 为准。

### 6.4 sys_file

| 字段 | 类型/约束 | 含义 |
|---|---|---|
| id | BIGINT PK | 文件记录 ID |
| object_key | VARCHAR(512) UNIQUE NOT NULL | 服务端生成的对象 key |
| original_name | VARCHAR(255) NOT NULL | 展示名 |
| mime_type | VARCHAR(128) NOT NULL DEFAULT '' | MIME |
| size | BIGINT NOT NULL | 字节数 |
| sha256 | VARCHAR(64) NOT NULL | 内容哈希 |
| driver | VARCHAR(32) NOT NULL DEFAULT 'local' | 存储驱动 |
| created_by | BIGINT NULL | 上传者 |
| created_at | TIMESTAMP NOT NULL | 上传时间 |
| deleted_at | TIMESTAMP NULL | 软删除时间 |

约束：对象 key 不含用户路径；下载鉴权；删除先软删，物理清理需验证无业务引用。

### 6.5 sys_server（SSH 服务器凭据记录）

| 字段 | 类型/约束 | 含义 |
|---|---|---|
| id | BIGINT PK | 服务器 ID |
| name | VARCHAR(64) UNIQUE NOT NULL | 稳定名 |
| connect_type | VARCHAR(16) NOT NULL | direct/config |
| host/port | VARCHAR(255)/INTEGER | 连接目标，port 默认 22 |
| username | VARCHAR(64) NOT NULL DEFAULT '' | SSH 用户 |
| auth_type | VARCHAR(16) NOT NULL | key/password |
| key_path | VARCHAR(500) NOT NULL DEFAULT '' | 私钥路径引用 |
| password_enc | TEXT NOT NULL DEFAULT '' | AES-GCM 密文 |
| config_alias | VARCHAR(64) NOT NULL DEFAULT '' | SSH config 别名 |
| remark | VARCHAR(500) NOT NULL DEFAULT '' | 说明 |
| last_test_at | TIMESTAMP NULL | 最近连接测试 |
| last_test_ok | BOOLEAN NOT NULL DEFAULT false | 测试结果 |
| last_test_error | VARCHAR(500) NOT NULL DEFAULT '' | 脱敏错误 |
| created_at/updated_at | TIMESTAMP NOT NULL | 记录时间 |

约束：凭据不得明文存储或回显；密钥引用与密文互斥/优先级需在实现中明确；known_hosts 校验失败不得静默绕过；服务器被数据源引用时不得删除。该表是本期候选，最终字段须与 DATA_DICT.md §5.1 一致。

### 6.6 SQL 逻辑与关系

表关系：
- sys_task.parent_no 自关联业务父任务；retry_of_no 自关联上一次执行尝试；sys_task_log.task_no 关联 sys_task.task_no。
- sys_schedule.last_task_id/pending_task_id 指向 sys_task.task_no。
- sys_file.created_by 可关联 sys_user.id（用户删除策略需与审计保留规则一致）。
- sys_server 独立保存 SSH 目标与加密凭据，不存私钥明文。

核心 SQL 逻辑（通用类型；SQLite 方言由 migration 文件调整）：

~~~sql
CREATE TABLE sys_task (
  id BIGINT PRIMARY KEY,
  task_no VARCHAR(64) NOT NULL UNIQUE,
  parent_no VARCHAR(64) NULL,
  attempt_no INTEGER NOT NULL DEFAULT 1,
  retry_of_no VARCHAR(64) NULL,
  kind VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  status VARCHAR(16) NOT NULL,
  progress_current BIGINT NOT NULL DEFAULT 0,
  progress_total BIGINT NOT NULL DEFAULT 0,
  result_text TEXT NOT NULL DEFAULT '',
  error_code VARCHAR(96) NOT NULL DEFAULT '',
  error_text TEXT NOT NULL DEFAULT '',
  data_bytes BIGINT NOT NULL DEFAULT 0,
  created_by VARCHAR(64) NOT NULL,
  started_at TIMESTAMP NULL,
  finished_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  FOREIGN KEY(parent_no) REFERENCES sys_task(task_no),
  FOREIGN KEY(retry_of_no) REFERENCES sys_task(task_no)
);
CREATE INDEX idx_sys_task_kind_status ON sys_task(kind, status);
CREATE INDEX idx_sys_task_parent_no ON sys_task(parent_no);
CREATE INDEX idx_sys_task_created_at ON sys_task(created_at);
CREATE INDEX idx_sys_task_retry_of_no ON sys_task(retry_of_no);

CREATE TABLE sys_task_log (
  id BIGINT PRIMARY KEY,
  task_no VARCHAR(64) NOT NULL,
  seq BIGINT NOT NULL,
  level VARCHAR(8) NOT NULL,
  message TEXT NOT NULL,
  attrs_json TEXT NULL,
  created_at TIMESTAMP NOT NULL,
  UNIQUE(task_no, seq),
  FOREIGN KEY(task_no) REFERENCES sys_task(task_no) ON DELETE CASCADE
);

CREATE TABLE sys_file (
  id BIGINT PRIMARY KEY,
  object_key VARCHAR(512) NOT NULL UNIQUE,
  original_name VARCHAR(255) NOT NULL,
  mime_type VARCHAR(128) NOT NULL DEFAULT '',
  size BIGINT NOT NULL,
  sha256 VARCHAR(64) NOT NULL,
  driver VARCHAR(32) NOT NULL DEFAULT 'local',
  created_by BIGINT NULL,
  created_at TIMESTAMP NOT NULL,
  deleted_at TIMESTAMP NULL
);

CREATE TABLE sys_server (
  id BIGINT PRIMARY KEY,
  name VARCHAR(64) NOT NULL UNIQUE,
  connect_type VARCHAR(16) NOT NULL,
  host VARCHAR(255) NOT NULL DEFAULT '',
  port INTEGER NOT NULL DEFAULT 22,
  username VARCHAR(64) NOT NULL DEFAULT '',
  auth_type VARCHAR(16) NOT NULL,
  key_path VARCHAR(500) NOT NULL DEFAULT '',
  password_enc TEXT NOT NULL DEFAULT '',
  config_alias VARCHAR(64) NOT NULL DEFAULT '',
  remark VARCHAR(500) NOT NULL DEFAULT '',
  last_test_at TIMESTAMP NULL,
  last_test_ok BOOLEAN NOT NULL DEFAULT FALSE,
  last_test_error VARCHAR(500) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
~~~

sys_schedule 的 SQL 见 DATA_DICT.md §4.3，避免复制第二套表定义。所有外键、索引和 SQLite/MySQL/PostgreSQL 方言差异必须由对应迁移脚本和集成测试落实。

### 6.7 任务生命周期与恢复数据流

1. 提交前验证 kind 已注册、payload 合法、并发规则可用；持久化 pending 记录后才进入可执行队列。
2. worker 将 pending 原子转换为 running，记录 started_at。
3. 执行 progress/log 通过有界节流写入；终态和 result/error 在一次事务中持久化。
4. 取消由 context 取消请求与状态转换协同完成；终态不可被重新转换为 running。
5. 进程启动扫描遗留 running：默认标记 interrupted 并记录原因。
6. 只有显式注册 Retry handler 且业务幂等的 kind 才允许创建新任务记录：生成新的 task_no、attempt_no 递增、retry_of_no 指向前一次尝试；禁止伪装为从进程中断位置续跑。
7. Scheduler 从 SQL Store 读取到期条目；创建任务成功后记录 pending_task_id。任务收口后更新结果与 next_run_at。
8. task 写入成功而 schedule 更新失败、或反之，需通过幂等关联和启动 reconciliation 修复；不得吞掉 Store 错误。

## 7. API、权限与日志边界

- 任务 API、调度 API、文件 API、日志和审计 API 以 API.md 为唯一端点与响应结构契约。
- 各资源读写权限由 Manifest 注册，服务端路由再次执行授权。
- 文件下载强制权限检查；路径由 object_key 解析，不能接受任意磁盘路径。
- Application/Access/Audit/Task 日志分别明确来源、索引和保留期；不得将普通日志查询伪装成审计日志。
- Server 凭据 API 只返回是否已配置，不返回密码、私钥或 DSN 明文。

## 8. 验收标准

- Task Center 持久化提交、进度、日志、完成、失败、取消；并发限制与互斥规则不回退。
- 进程重启后所有遗留 running 均有明确 interrupted 记录；可重试任务必须以新 attempt 有审计地重新入队。
- Scheduler SQL Store 与任务 Store 状态一致；无悬空 pending_task_id；同一调度项不会重复派发活动任务。
- 假时钟测试覆盖到期、漏跑、时区、上轮运行中和 Store 故障。
- 文件上传/下载/软删/物理清理有路径穿越、内容类型、大小、权限和哈希测试。
- SSH 密文不泄露；known_hosts 校验按安全策略失败关闭。
- 日志查询分页、时间范围、保留策略与权限均有测试。
- SQLite 集成测试真实通过；其他方言只在相应 CI 证据完成后声明支持。

## 9. 风险、前置条件与确认

- 前置阶段为 P03；只有 P03 ACCEPTED 才可进入 P04。
- 任务持久化不能序列化执行闭包；重启恢复必须采用 interrupted + opt-in retry 语义。
- task 与 scheduler 两套 Store 的双写一致性需要事务或 reconciliation，未解决前不可宣称无丢失。
- sys_server 字段须与 DATA_DICT.md §5.1 一致；sys_datasource 不纳入本期。
- 本期 DESIGN_PLAN 需用户明确确认；IMPL_PLAN 后置校准；实施需另获明确开工授权。

## 10. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | PROPOSED | P04 尚未获确认；任务恢复语义须重点评审 |
