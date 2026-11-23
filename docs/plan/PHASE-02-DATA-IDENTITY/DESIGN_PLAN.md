# P02 设计计划：数据与身份安全

> 状态：PROPOSED（待用户评审；未批准前不得进入本期实施）。

## 1. OGSM 与目标边界

**Objective**：让 EasyAdmin 从最小运行内核进入可安全承载业务数据的状态，建立唯一迁移通道、身份会话和服务端授权。

**Goals**：默认 SQLite 可初始化；核心表迁移有版本/checksum；首次管理员创建安全且幂等；会话可撤销；RBAC 由服务端判定；审计与应用日志分离。

**Strategies**：复用已有 infra/db 与 easydb；SQL-first；以事务和数据库约束维护数据不变量；把认证/授权放在服务端；先完成 SQLite，再以测试证据认证其他方言。

**Measures**：迁移重复执行安全；非法/破坏性迁移不静默通过；401/403/409/422 契约有测试；最后一个活跃超级管理员保护有效；敏感数据不回显。

## 2. 现状结论（源码证据）

| 当前资产 | 证据 | 本期处理 |
|---|---|---|
| 通用 DB/schema 能力 | infra/db/ | 先盘点 easydb adapter、Catalog、DDL Diff；缺少的 migration runner 独立补齐 |
| 命名数据源管理 | infra/db/datasource.go | 目前 JSON 文件仓储；不能宣称已经有 sys_datasource 表 |
| 配置启动入口 | infra/conf/ | P01 提供启动配置装配；D10 约束 v0.x 不做 Runtime Settings |
| 认证、用户与会话 | 当前树中尚无 apps/system/users、infra/auth | 按本期设计新建，不能引用目标树冒充已有 |
| 系统表与迁移脚本 | 当前树中尚无 sql/ | 本期建立 SQLite 核心 schema 与 migration runner |

## 3. 本期范围

- 以现有 infra/db 为地基核验数据库连接、事务、SQL 执行和方言能力。
- 默认 SQLite 核心 schema、迁移版本、checksum、并发初始化保护与失败诊断。
- 用户、角色、权限、用户角色/角色权限、服务端会话、审计、迁移记录。
- 安全密码哈希、登录/登出、CSRF、登录限流、首次管理员初始化。
- RBAC 路由授权、服务端权限检查与菜单权限元信息。
- 将数据库支持范围限定为完成集成测试的方言；MySQL/PostgreSQL 不因脚本存在即宣布支持。

## 4. 明确不做

- Runtime Settings / sys_setting，不让数据库设置覆盖 Bootstrap Config。
- Task Center 持久化、Scheduler SQL Store、文件存储和运维页面。
- OIDC/LDAP、多租户、分布式会话或外部 API Token。
- 未完成方言集成验证的数据库支持承诺。
- sys_server 的 SSH 管理记录与服务器页面；它依赖 P04 的运维域评审。
- sys_datasource 的 SQL 持久化表；当前 DataSourceManager JSON 仓储不是该表，是否将其迁入系统库须另行设计。P02 只负责可启动所需的 Bootstrap 数据库连接。

## 5. 架构与安全决策

**现行有效决策表：**

| 决策点 | 当前有效结论 | 出处 |
|---|---|---|
| API 兼容 | API 不设路径版本，既有端点向下兼容 | DECISIONS.md D13，用户已确认 |

以下安全、迁移和数据设计均为 P02 待评审提案；只有用户确认本 DESIGN_PLAN 后才成为本期约束。

- SQL-first，不引入 ORM；SQL 脚本按方言组织。
- 用户密码只存成熟算法生成的哈希；会话只存 token 哈希，不存 Cookie 原文。
- 服务端是 RBAC 最终权威；前端隐藏菜单不构成授权。
- 首次管理员创建必须是原子流程；初始化完成后关闭重复初始化入口。
- 禁止删除/禁用最后一个活跃超级管理员。
- 审计记录只追加，不记录密码、session token、DSN 或凭据明文。
- 核心迁移与业务迁移分 scope，checksum 不一致阻止启动或升级。

## 6. 数据结构（本期独立设计契约）

类型映射遵循 DATA_DICT.md §1：主键 int64；时间 UTC；API RFC3339；SQLite/MySQL/PostgreSQL 方言类型分别定义。以下为本期逻辑结构；字段变更必须同步本文件与 DATA_DICT.md。

### 6.1 核心表字段

| 表 | 字段与类型/约束 | 关系/不变量 |
|---|---|---|
| sys_user | id BIGINT PK；username VARCHAR(64) UNIQUE NOT NULL；display_name VARCHAR(64) NOT NULL；password_hash VARCHAR(255) NOT NULL；status VARCHAR(16) NOT NULL DEFAULT active；last_login_at nullable；created_at/updated_at timestamp NOT NULL | 禁止用户名复用；至少一个 active 超级管理员 |
| sys_role | id BIGINT PK；code VARCHAR(64) UNIQUE NOT NULL；name VARCHAR(64) NOT NULL；description VARCHAR(255) NOT NULL DEFAULT ''；is_system BOOLEAN NOT NULL DEFAULT false；created_at/updated_at timestamp NOT NULL | 系统角色不可删除；code 发布后稳定 |
| sys_permission | id BIGINT PK；code VARCHAR(96) UNIQUE NOT NULL；name VARCHAR(64) NOT NULL；module VARCHAR(64) NOT NULL；description VARCHAR(255) NOT NULL DEFAULT ''；created_at timestamp NOT NULL | 权限码由 Manifest 登记；不复用已发布 code |
| sys_role_permission | role_id BIGINT NOT NULL；permission_id BIGINT NOT NULL；PRIMARY KEY(role_id, permission_id) | FK 到 role/permission，关联删除按约定处理 |
| sys_user_role | user_id BIGINT NOT NULL；role_id BIGINT NOT NULL；PRIMARY KEY(user_id, role_id) | FK 到 user/role |
| sys_session | id VARCHAR(64) PK（token SHA-256）；user_id BIGINT NOT NULL；ip VARCHAR(64)；user_agent VARCHAR(255)；expires_at/created_at/last_seen_at timestamp NOT NULL | FK 到 user；按 user_id、expires_at 建索引；禁用用户立即撤销会话 |
| sys_audit_log | id BIGINT PK；actor_id BIGINT nullable；actor_name VARCHAR(64)；action VARCHAR(96) NOT NULL；target_type VARCHAR(64)；target_id VARCHAR(64)；summary VARCHAR(255) NOT NULL；detail_json JSON nullable；ip VARCHAR(64)；user_agent VARCHAR(255)；request_id VARCHAR(64)；created_at timestamp NOT NULL | 只追加；detail 必须脱敏 |
| sys_migration | scope VARCHAR(64)；version VARCHAR(32)；name VARCHAR(128)；checksum VARCHAR(64)；applied_at timestamp；PRIMARY KEY(scope, version) | scope=core 或 app:<module-id>；已执行 checksum 不可改 |

### 6.2 SQL 关系定义

~~~sql
CREATE TABLE sys_user (
  id BIGINT PRIMARY KEY,
  username VARCHAR(64) NOT NULL UNIQUE,
  display_name VARCHAR(64) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'active',
  last_login_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE TABLE sys_role (
  id BIGINT PRIMARY KEY,
  code VARCHAR(64) NOT NULL UNIQUE,
  name VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
CREATE TABLE sys_permission (
  id BIGINT PRIMARY KEY,
  code VARCHAR(96) NOT NULL UNIQUE,
  name VARCHAR(64) NOT NULL,
  module VARCHAR(64) NOT NULL,
  description VARCHAR(255) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE sys_role_permission (
  role_id BIGINT NOT NULL REFERENCES sys_role(id) ON DELETE CASCADE,
  permission_id BIGINT NOT NULL REFERENCES sys_permission(id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, permission_id)
);
CREATE TABLE sys_user_role (
  user_id BIGINT NOT NULL REFERENCES sys_user(id) ON DELETE CASCADE,
  role_id BIGINT NOT NULL REFERENCES sys_role(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, role_id)
);
CREATE TABLE sys_session (
  id VARCHAR(64) PRIMARY KEY,
  user_id BIGINT NOT NULL REFERENCES sys_user(id) ON DELETE CASCADE,
  ip VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(255) NOT NULL DEFAULT '',
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,
  last_seen_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_sys_session_user_id ON sys_session(user_id);
CREATE INDEX idx_sys_session_expires_at ON sys_session(expires_at);
CREATE TABLE sys_audit_log (
  id BIGINT PRIMARY KEY,
  actor_id BIGINT NULL,
  actor_name VARCHAR(64) NOT NULL DEFAULT '',
  action VARCHAR(96) NOT NULL,
  target_type VARCHAR(64) NOT NULL DEFAULT '',
  target_id VARCHAR(64) NOT NULL DEFAULT '',
  summary VARCHAR(255) NOT NULL,
  detail_json TEXT NULL,
  ip VARCHAR(64) NOT NULL DEFAULT '',
  user_agent VARCHAR(255) NOT NULL DEFAULT '',
  request_id VARCHAR(64) NOT NULL DEFAULT '',
  created_at TIMESTAMP NOT NULL
);
CREATE TABLE sys_migration (
  scope VARCHAR(64) NOT NULL,
  version VARCHAR(32) NOT NULL,
  name VARCHAR(128) NOT NULL,
  checksum VARCHAR(64) NOT NULL,
  applied_at TIMESTAMP NOT NULL,
  PRIMARY KEY(scope, version)
);
~~~

关系：一个用户可关联多个角色；一个角色可关联多个权限；会话归属用户；审计操作者允许为空（系统任务）；迁移记录按 scope/version 唯一。不得以删除权限记录的方式静默改变既有角色的权限解释。

### 6.3 初始化与数据流

1. 加载 Bootstrap Config，建立默认 SQLite 数据源。
2. 获取迁移锁，检查 sys_migration。
3. 按 core migration 顺序执行 DDL；仅全部成功后写入版本/checksum。
4. 同步系统角色与 Manifest 权限码。
5. 若无管理员，进入一次性 setup 状态；初始化事务写入管理员、超级管理员角色和关联。
6. 登录验证密码哈希，创建随机 session token；数据库只存 SHA-256。
7. 每个受保护请求由会话认证与服务端 RBAC 检查；危险操作写审计。
8. 用户禁用/密码更改/登出时按策略撤销会话。

## 7. API、配置与权限边界

- 初始化、登录、登出、当前用户、密码更新、用户/角色/权限和审计 API 以 API.md 为唯一字段契约。
- 配置中心继续遵守 D10：只管理 Bootstrap Config；不创建 sys_setting。
- 所有写 API 先做身份验证、CSRF、权限判断，再执行用例；前端菜单不代替后端鉴权。
- 401/403/409/422 错误按 API.md 固定结构返回。

## 8. 验收标准

- 删除默认 SQLite 文件后，启动能执行 core migrations；重复启动不重复建表/种子冲突。
- 两个并发首次初始化请求只有一个成功。
- 会话 token 原文不落库；禁用用户后会话立即失效。
- 最后一个活跃超级管理员不能被删除或禁用。
- 未登录 401、无权限 403、CSRF 失败 403、字段校验 422、唯一约束冲突 409。
- migration checksum 漂移阻止继续；失败迁移可诊断，不伪装成功。
- SQLite 集成测试真实执行；PostgreSQL/MySQL 只有在 CI 提供数据库并通过方言测试后才列为支持。

## 9. 质量自评

| 维度 | 自评结论 | 评审重点 |
|---|---|---|
| 工程化最佳实践 | SQL migration、事务、checksum 与服务端 RBAC 构成可验证安全边界 | 并发初始化、迁移中断和不可逆 DDL 是否有恢复方案 |
| 用户体验 | 初始化、登录、会话失效和权限错误均有明确状态 | setup 失败能否安全重试，权限不足是否可理解 |
| 改动量与复杂度 | 仅交付单实例 SQLite 身份核心，不引入 ORM、多租户或外部 IdP | migration runner 是否与 infra/db 重复，方言范围是否过宽 |
| 架构统一性 | 认证/授权服务端权威，API/DATA_DICT/SQL 共用字段口径 | Manifest 权限、路由授权和数据模型是否一致 |

以上为设计自评；安全实现和方言支持必须由终验测试证明。

## 10. 风险、阶段依赖与确认

- 前置阶段为 P01；P01 必须 ACCEPTED。
- P02 设计先由用户确认；IMPL 在设计确认后校准；开工另需用户明确授权。
- 密码/会话/CSRF 属安全敏感实现，必须使用成熟算法/协议，不自造加密。
- sys_datasource 的 SQL 持久化表与 sys_server 关系尚属候选；不纳入本期核心表。
- 审计数据保留期与脱敏策略需与 LOGGING_DESIGN 一致。

## 11. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | PROPOSED | P02 尚未获确认；命名数据源 SQL 模型不纳入本期 |
