# 核心数据字典（草案）

本文是 EasyAdmin 核心表的逻辑口径。实现时必须同步三方言建表脚本与 Go 模型，字段名、含义、枚举不得各自为政。

状态：P2 已按本口径落地 SQLite 核心结构（`sql/sqlite/core/`）；其后的字段变更必须走结构差异预览、显式确认与文档同步。

## 1. 通用约定

### 1.1 类型映射

| 逻辑类型 | SQLite | MySQL | PostgreSQL |
|---|---|---|---|
| 自增主键 | `INTEGER PRIMARY KEY AUTOINCREMENT` | `BIGINT AUTO_INCREMENT PRIMARY KEY` | `BIGSERIAL PRIMARY KEY` |
| 短字符串 | `TEXT` | `VARCHAR(n)` | `VARCHAR(n)` |
| 长文本 | `TEXT` | `TEXT` | `TEXT` |
| 布尔 | `INTEGER`（0/1） | `TINYINT(1)` | `BOOLEAN` |
| 时间 | `DATETIME` | `DATETIME(3)` | `TIMESTAMPTZ` |
| JSON | `TEXT` | `JSON` | `JSONB` |

### 1.2 时间

- 全部时间字段存 UTC；
- Go 侧使用 `time.Time`；
- API 输出 RFC3339；
- 字段命名统一 `created_at`、`updated_at`、`deleted_at`、`last_login_at` 等。

### 1.3 表名

- 系统表统一 `sys_` 前缀；
- 表名通过 `{table}` 占位符注入，不硬编码；
- 不设数据库前缀，命名空间由数据库/Schema 负责。

### 1.4 删除策略

- 用户不做物理删除：用 `status` 禁用，保留审计链；
- 角色仅在未被用户引用时可删除；系统角色不可删除；
- 文件使用 `deleted_at` 软删除，物理清理由任务执行；
- 任务与日志按保留期物理清理；
- 审计日志只增不改，按保留期清理。

## 2. 身份与权限

### 2.1 `sys_user`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 用户 ID |
| `username` | 短字符串(64) | NOT NULL, UNIQUE | 登录名，创建后不可改 |
| `display_name` | 短字符串(64) | NOT NULL | 展示名 |
| `password_hash` | 短字符串(255) | NOT NULL | bcrypt 哈希，永不回显 |
| `status` | 短字符串(16) | NOT NULL DEFAULT 'active' | `active` / `disabled` |
| `last_login_at` | 时间 | NULL | 最近登录成功时间 |
| `created_at` | 时间 | NOT NULL | 创建时间 |
| `updated_at` | 时间 | NOT NULL | 更新时间 |

索引：

- `UNIQUE(username)`
- `INDEX(status)`

规则：

- 至少保留一个 `active` 的超级管理员；
- 禁用用户立即失效其会话；
- 用户名不允许复用已存在记录（含禁用用户），避免审计歧义。

### 2.2 `sys_role`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 角色 ID |
| `code` | 短字符串(64) | NOT NULL, UNIQUE | 稳定编码，如 `super_admin` |
| `name` | 短字符串(64) | NOT NULL | 展示名 |
| `description` | 短字符串(255) | NOT NULL DEFAULT '' | 说明 |
| `is_system` | 布尔 | NOT NULL DEFAULT 0 | 系统内置角色，不可删除 |
| `created_at` | 时间 | NOT NULL | 创建时间 |
| `updated_at` | 时间 | NOT NULL | 更新时间 |

规则：

- `super_admin` 为内置角色，拥有全部权限；
- `is_system=1` 的角色不可删除、不可改 `code`；
- 角色被用户引用时不可删除。

### 2.3 `sys_permission`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 权限 ID |
| `code` | 短字符串(96) | NOT NULL, UNIQUE | 权限码，如 `system.user.read` |
| `name` | 短字符串(64) | NOT NULL | 展示名 |
| `module` | 短字符串(64) | NOT NULL | 所属模块 ID |
| `status` | 短字符串(16) | NOT NULL DEFAULT 'active' | `active` / `orphaned`；模块下线后标记 `orphaned`，不删除 |
| `description` | 短字符串(255) | NOT NULL DEFAULT '' | 说明 |
| `created_at` | 时间 | NOT NULL | 创建时间 |

索引：

- `UNIQUE(code)`
- `INDEX(module, status)`

规则：

- 权限码由模块 Manifest 声明，启动时同步到表；
- 已删除模块的权限标记为 `orphaned`，不自动删除，避免历史角色权限丢失；权限校验忽略 `orphaned`；
- 权限码一经发布不复用。

### 2.4 `sys_role_permission`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `role_id` | 主键类型 | NOT NULL | 角色 ID |
| `permission_id` | 主键类型 | NOT NULL | 权限 ID |

约束：

- `PRIMARY KEY(role_id, permission_id)`
- 外键级联删除（角色或权限删除时）

### 2.5 `sys_user_role`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `user_id` | 主键类型 | NOT NULL | 用户 ID |
| `role_id` | 主键类型 | NOT NULL | 角色 ID |

约束：

- `PRIMARY KEY(user_id, role_id)`
- 外键级联删除

### 2.6 `sys_session`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 短字符串(64) | PK | 会话 token 的 SHA-256，不存原文 |
| `user_id` | 主键类型 | NOT NULL | 用户 ID |
| `ip` | 短字符串(64) | NOT NULL DEFAULT '' | 登录 IP |
| `user_agent` | 短字符串(255) | NOT NULL DEFAULT '' | 浏览器 UA |
| `expires_at` | 时间 | NOT NULL | 过期时间 |
| `created_at` | 时间 | NOT NULL | 创建时间 |
| `last_seen_at` | 时间 | NOT NULL | 最近活跃时间 |

索引：

- `INDEX(user_id)`
- `INDEX(expires_at)`

规则：

- 登出删除会话；
- 禁用用户删除其全部会话；
- 过期会话由清理任务删除。

## 3. 审计

### 3.1 `sys_audit_log`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 审计 ID |
| `actor_id` | 主键类型 | NULL | 操作者用户 ID；系统任务可为空 |
| `actor_name` | 短字符串(64) | NOT NULL DEFAULT '' | 操作者名称快照 |
| `action` | 短字符串(96) | NOT NULL | 动作码，如 `system.user.update` |
| `outcome` | 短字符串(16) | NOT NULL | `success` / `failure` / `denied` |
| `target_type` | 短字符串(64) | NOT NULL DEFAULT '' | 目标类型 |
| `target_id` | 短字符串(64) | NOT NULL DEFAULT '' | 目标 ID |
| `summary` | 短字符串(255) | NOT NULL | 人读摘要 |
| `detail_json` | JSON | NULL | 结构化详情（已脱敏） |
| `ip` | 短字符串(64) | NOT NULL DEFAULT '' | 来源 IP |
| `user_agent` | 短字符串(255) | NOT NULL DEFAULT '' | UA |
| `request_id` | 短字符串(64) | NOT NULL DEFAULT '' | 请求链路 ID |
| `created_at` | 时间 | NOT NULL | 发生时间 |

索引：

- `INDEX(created_at)`
- `INDEX(actor_id)`
- `INDEX(action)`

规则：

- 只增不改；
- `outcome` 必填，区分成功、失败与授权拒绝；
- `detail_json` 不记录密码、令牌、密钥；
- 保留期由调度任务清理。

## 4. 任务与调度

### 4.1 `sys_task`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 主键 |
| `task_no` | 短字符串(64) | NOT NULL, UNIQUE | 任务编号 |
| `parent_no` | 短字符串(64) | NULL | 业务父任务编号，最多两级 |
| `attempt_no` | INTEGER | NOT NULL DEFAULT 1 | 同一逻辑任务的执行尝试序号 |
| `retry_of_no` | 短字符串(64) | NULL | 本次尝试重试的前一任务编号 |
| `kind` | 短字符串(64) | NOT NULL | 任务类型 |
| `title` | 短字符串(255) | NOT NULL | 任务标题 |
| `status` | 短字符串(16) | NOT NULL | 见枚举 |
| `progress_current` | BIGINT | NOT NULL DEFAULT 0 | 当前进度 |
| `progress_total` | BIGINT | NOT NULL DEFAULT 0 | 总进度；0 表示未知 |
| `result_text` | 长文本 | NOT NULL DEFAULT '' | 人读结论 |
| `error_code` | 短字符串(96) | NOT NULL DEFAULT '' | 错误码 |
| `error_text` | 长文本 | NOT NULL DEFAULT '' | 错误摘要 |
| `data_bytes` | BIGINT | NOT NULL DEFAULT 0 | 处理数据量 |
| `created_by` | 短字符串(64) | NOT NULL | 触发者：用户名 / `scheduler` / `system` |
| `started_at` | 时间 | NULL | 开始时间 |
| `finished_at` | 时间 | NULL | 结束时间 |
| `created_at` | 时间 | NOT NULL | 创建时间 |
| `updated_at` | 时间 | NOT NULL | 更新时间 |

枚举 `status`：

| 值 | 含义 | 终态 |
|---|---|---|
| `pending` | 已提交待执行 | 否 |
| `running` | 执行中 | 否 |
| `success` | 成功 | 是 |
| `failed` | 失败 | 是 |
| `cancelled` | 人工取消 | 是 |
| `skipped` | 撞并发跳过 | 是 |
| `interrupted` | 进程退出后发现未完成并被收口 | 是 |

索引：

- `UNIQUE(task_no)`
- `INDEX(parent_no)`
- `INDEX(kind, status)`
- `INDEX(created_at)`
- `INDEX(retry_of_no)`

不变量：

- `pending` / `running` 任务永不被清理；
- 进程启动发现遗留 `running` 时，标记 `interrupted`，不声称恢复执行现场；
- 当前 `infra/tasks` 的 `done` 映射为持久化 `success`；`pending`、`skipped`、`interrupted` 由持久化执行/恢复适配层定义，不代表现有内存 API 已提供这些状态；
- 每次重试创建新的 `task_no`，`attempt_no` 递增，`retry_of_no` 指向前一次尝试；`parent_no` 只表示业务父子任务；
- 终态保留期由配置决定。

### 4.2 `sys_task_log`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 主键 |
| `task_no` | 短字符串(64) | NOT NULL | 任务编号 |
| `seq` | BIGINT | NOT NULL | 同任务内递增序号 |
| `level` | 短字符串(8) | NOT NULL | `debug` / `info` / `warn` / `error` |
| `message` | 长文本 | NOT NULL | 日志内容 |
| `attrs_json` | JSON | NULL | 结构化字段 |
| `created_at` | 时间 | NOT NULL | 时间 |

索引：

- `UNIQUE(task_no, seq)`
- `INDEX(created_at)`

### 4.3 sys_schedule

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| name | 短字符串(64) | PK | 调度项稳定名称 |
| title | 短字符串(128) | NOT NULL | 展示名 |
| config_key | 短字符串(128) | NOT NULL | 启停配置键；启停唯一真源 |
| plan_key | 短字符串(128) | NOT NULL DEFAULT '' | 周期配置键；为空时采用注册的固定计划 |
| remark | 短字符串(500) | NOT NULL DEFAULT '' | 说明 |
| last_status | 短字符串(16) | NOT NULL DEFAULT '' | 最近一次调度结果 |
| last_message | 短字符串(255) | NOT NULL DEFAULT '' | 最近一次结果摘要 |
| last_task_id | 短字符串(64) | NOT NULL DEFAULT '' | 最近一次派发的任务 ID |
| pending_task_id | 短字符串(64) | NOT NULL DEFAULT '' | 当前待跟踪任务 ID |
| last_run_at | 时间 | NULL | 最近一次任务收口时间 |
| next_run_at | 时间 | NULL | 下次计划触发时间 |
| updated_at | 时间 | NOT NULL | 更新时间 |

索引：
- INDEX(next_run_at)
- INDEX(last_status)

唯一事实源与约束：
- name 是主键，不另建自增 id。
- 不存储 enabled 列；启停状态由 config_key 指向的配置项计算，避免配置中心与数据库双真相。
- 周期不落库：plan_key 为空时采用代码注册的 Plan，只读；非空时读取该配置键并验证为 1~525600 的整数分钟，配置值是周期唯一真源。
- last_task_id 是最近派发任务，pending_task_id 是当前正在跟踪的任务指针；任务详情由 Task Center 提供，不在调度表重复存储。
- 本表定义的是目标 SQL Store 持久化契约；现有 infra/scheduler 只有 Store 接口和 MemoryStore，数据库实现待 P04。
- P04 必须明确任务中心重启后状态恢复与调度 pending_task_id 的一致性，不能将仅持久化调度指针视为任务接管完成。

SQL 逻辑定义（SQLite 参考；MySQL/PostgreSQL 按 §1.1 映射生成）：

~~~sql
CREATE TABLE sys_schedule (
    name            VARCHAR(64) PRIMARY KEY,
    title           VARCHAR(128) NOT NULL,
    config_key      VARCHAR(128) NOT NULL,
    plan_key        VARCHAR(128) NOT NULL DEFAULT '',
    remark          VARCHAR(500) NOT NULL DEFAULT '',
    last_status     VARCHAR(16) NOT NULL DEFAULT '',
    last_message    VARCHAR(255) NOT NULL DEFAULT '',
    last_task_id    VARCHAR(64) NOT NULL DEFAULT '',
    pending_task_id VARCHAR(64) NOT NULL DEFAULT '',
    last_run_at     TIMESTAMP NULL,
    next_run_at     TIMESTAMP NULL,
    updated_at      TIMESTAMP NOT NULL
);
CREATE INDEX idx_sys_schedule_next_run_at ON sys_schedule(next_run_at);
CREATE INDEX idx_sys_schedule_last_status ON sys_schedule(last_status);
~~~

## 5. 服务器与命名数据源（候选，待阶段评审）

本节来自 INFRA_SCHEDULER_AND_CREDENTIALS.md 的方案，尚未成为已批准的核心表契约。建议在 P04 作为服务器/SSH 运维能力设计一并评审；若 P02 需要先持久化命名数据源，应在 P02 设计确认前重新划分依赖与交付顺序。

### 5.1 sys_server

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | 自增主键 | PK | 服务器 ID |
| name | 短字符串(64) | NOT NULL, UNIQUE | 稳定名称 |
| connect_type | 短字符串(16) | NOT NULL | direct / config |
| host | 短字符串(255) | NOT NULL DEFAULT '' | 主机地址 |
| port | INTEGER | NOT NULL DEFAULT 22 | SSH 端口 |
| username | 短字符串(64) | NOT NULL DEFAULT '' | SSH 用户 |
| auth_type | 短字符串(16) | NOT NULL | key / password |
| key_path | 短字符串(500) | NOT NULL DEFAULT '' | 私钥路径引用，不复制私钥明文 |
| password_enc | TEXT | NOT NULL DEFAULT '' | AES-256-GCM 加密凭据 |
| config_alias | 短字符串(64) | NOT NULL DEFAULT '' | SSH config 别名 |
| remark | 短字符串(500) | NOT NULL DEFAULT '' | 说明 |
| last_test_at | 时间 | NULL | 最近测试时间 |
| last_test_ok | 布尔 | NOT NULL DEFAULT 0 | 最近测试是否成功 |
| last_test_error | 短字符串(500) | NOT NULL DEFAULT '' | 脱敏错误摘要 |
| created_at | 时间 | NOT NULL | 创建时间 |
| updated_at | 时间 | NOT NULL | 更新时间 |

约束：唯一索引 name；不得记录密码/私钥明文；known_hosts 校验失败不得自动降级跳过；凭据轮换必须有重新录入/迁移路径。

### 5.2 sys_datasource

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| id | 自增主键 | PK | 数据源 ID |
| name | 短字符串(64) | NOT NULL, UNIQUE | 稳定名称 |
| driver | 短字符串(32) | NOT NULL | 数据库驱动 |
| dialect | 短字符串(32) | NOT NULL | SQL 方言 |
| role | 短字符串(16) | NOT NULL | readwrite / readonly |
| dsn_enc | TEXT | NOT NULL DEFAULT '' | AES-256-GCM 加密 DSN |
| server_id | BIGINT | NULL, FK | 可选 SSH 隧道服务器，引用 sys_server.id |
| active | 布尔 | NOT NULL DEFAULT 0 | 当前激活项 |
| remark | 短字符串(500) | NOT NULL DEFAULT '' | 说明 |
| created_at | 时间 | NOT NULL | 创建时间 |
| updated_at | 时间 | NOT NULL | 更新时间 |

约束：唯一索引 name；同一时刻至多一个 active 项（需在事务/数据库约束层保证）；删除被活动连接引用的数据源前必须阻止或显式切换；DSN 密文不得通过 API 回显。

### 5.3 关系与 SQL 逻辑定义

- sys_datasource.server_id 可空引用 sys_server.id；服务器不得在仍被数据源引用时删除。
- password_enc 与 dsn_enc 仅存储由 infra/credential 加密的密文；密钥不入库、不进入日志。
- sys_datasource 当前尚无 SQL 表/仓储实现；infra/db/datasource.go 是 JSON 文件仓储，不是此表的实现。

~~~sql
CREATE TABLE sys_server (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
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

CREATE TABLE sys_datasource (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(64) NOT NULL UNIQUE,
    driver VARCHAR(32) NOT NULL,
    dialect VARCHAR(32) NOT NULL,
    role VARCHAR(16) NOT NULL,
    dsn_enc TEXT NOT NULL DEFAULT '',
    server_id BIGINT NULL REFERENCES sys_server(id),
    active BOOLEAN NOT NULL DEFAULT FALSE,
    remark VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
~~~

## 6. 文件

### 6.1 `sys_file`

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `id` | 自增主键 | PK | 主键 |
| `object_key` | 短字符串(512) | NOT NULL, UNIQUE | 存储对象 key |
| `original_name` | 短字符串(255) | NOT NULL | 原始文件名 |
| `mime_type` | 短字符串(128) | NOT NULL DEFAULT '' | MIME |
| `size` | BIGINT | NOT NULL | 字节数 |
| `sha256` | 短字符串(64) | NOT NULL | 内容哈希 |
| `driver` | 短字符串(32) | NOT NULL DEFAULT 'local' | 存储驱动 |
| `created_by` | 主键类型 | NULL | 上传者 |
| `created_at` | 时间 | NOT NULL | 上传时间 |
| `deleted_at` | 时间 | NULL | 软删除时间 |

索引：

- `UNIQUE(object_key)`
- `UNIQUE(sha256)`
- `INDEX(created_at)`
- `INDEX(deleted_at)`

规则：

- 下载必须经鉴权路由；
- 删除先软删，物理清理由任务执行；
- 物理删除前确认无业务引用。

## 7. 结构同步与迁移

### 7.1 声明式结构同步（开发期方案）

开发期不维护版本迁移账本，数据库结构以各 `sql/<dialect>/` 脚本与 `TableSpec` 清单为唯一期望结构：

- 每张表登记 `TableSpec{Table, CreateScript, IndexScript}`；脚本使用 `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`，可重复执行。
- 启动时按 `core → app:<module-id>` 的 scope 顺序执行幂等 Ensure，只补齐缺失的表和索引。
- 结构差异用 `DiffSchema` 只读预览，生成的 ALTER SQL 必须显式确认后执行；不自动删列、改列、改类型、重命名。
- 多语句脚本由统一拆分器逐条执行；单表失败记录表名与语句，不伪装成功。
- 开发期数据库允许从空库重建，因此不创建 `sys_migration` / `schema_migrations` 表，避免版本账本与真实结构形成第二真相。

理由与取舍见 `DECISIONS.md` D20。

### 7.2 数据迁移（未启用）

纯建表/加列由结构同步覆盖。只有出现非幂等数据操作（数据回填、拆列、跨表修复）时，才另立决策引入 `sys_data_migration`：

| 字段 | 类型 | 约束 | 说明 |
|---|---|---|---|
| `scope` | 短字符串(64) | PK | `core` 或 `app:<module-id>` |
| `version` | 短字符串(32) | PK | 数据迁移版本 |
| `name` | 短字符串(128) | NOT NULL | 迁移名 |
| `checksum` | 短字符串(64) | NOT NULL | 脚本校验和 |
| `applied_at` | 时间 | NOT NULL | 执行时间 |

该表在真实需求出现并决策前不创建。

## 8. 表清单与期次

| 表 | 期次 | 说明 |
|---|---|---|
| `sys_user` | P2 | 用户 |
| `sys_role` | P2 | 角色 |
| `sys_permission` | P2 | 权限 |
| `sys_role_permission` | P2 | 角色权限 |
| `sys_user_role` | P2 | 用户角色 |
| `sys_session` | P2 | 会话 |
| `sys_audit_log` | P2 | 审计 |
| `sys_task` | P4 | 任务 |
| `sys_task_log` | P4 | 任务日志 |
| `sys_schedule` | P4 | 调度 |
| `sys_file` | P4 | 文件 |
| `sys_server` | P4（候选，待评审） | SSH 服务器与凭据引用 |
| `sys_datasource` | 待独立设计评审 | 加密命名数据源；当前 JSON 文件仓储不等于 SQL 系统表 |

## 9. 变更红线

修改任何表字段、默认值、枚举时，必须同时更新：

1. `sql/{sqlite,mysql,postgres}/` 对应脚本；
2. 本文对应章节；
3. Go 模型与 DTO；
4. 相关 API 文档与前端字段；
5. 结构脚本与回归测试。

只改其中一处视为未完成。


## 10. API 与数据类型一致性

- 核心表自增主键在 Go 中统一使用 `int64`，JSON 中使用整数；禁止部分 API 把同一字段编码为数字、部分编码为字符串。
- `task_no`、对象 key 等业务标识可使用字符串，但不得与数据库自增主键混为一谈。
- PostgreSQL/MySQL 的具体自增类型、时间精度、布尔和 JSON 映射必须由方言结构同步测试验证，不以表格定义代替实测。
- 对外接口不暴露 `sql.Null*`、驱动私有类型或数据库内部错误。

## 11. 结构同步执行红线

- 模块 `Register` 只登记 `TableSpec` 与 SQL 资源，不直接执行 DDL。
- 启动 Ensure 只做加法：建缺失表、建缺失索引；不得自动删列、改列、改类型。
- 列级差异必须经 `DiffSchema` 预览并由操作者显式确认；破坏性变更默认拒绝无提示执行。
- 同名表在 core 与 app 之间冲突时启动失败，不静默覆盖。
- SQLite、PostgreSQL、MySQL 结构脚本分别测试；数据库不支持事务 DDL 时必须记录部分执行和恢复办法。
