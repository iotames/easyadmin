# P02 设计计划：数据与身份安全

> 状态：`ACCEPTED`（2026-11-26 实施并验收完成）。

## 1. OGSM 与目标边界

**Objective**：让 EasyAdmin 从最小运行内核进入可安全承载业务数据的状态，建立声明式结构同步通道、身份会话和服务端授权。

**Goals**：默认 SQLite 可初始化；核心表结构可由 `TableSpec` 清单幂等同步；首次管理员创建安全且幂等；会话可撤销；RBAC 由服务端判定；审计与应用日志分离。

**Strategies**：复用已有 infra/db 与 easydb；SQL-first；以事务和数据库约束维护数据不变量；把认证/授权放在服务端；先完成 SQLite，再以测试证据认证其他方言。

**Measures**：结构同步重复执行安全；列级变更必须经差异预览并显式确认；401/403/409/422 契约有测试；最后一个活跃超级管理员保护有效；敏感数据不回显。

## 2. 现状结论（源码证据）

| 当前资产 | 证据 | 本期处理 |
|---|---|---|
| 通用 DB/schema 能力 | infra/db/（Open、EnsureTables、DiffSchema、Catalog、GenerateSQL） | 复用 infra/db 的 EnsureTables/DiffSchema，不另造迁移引擎 |
| 命名数据源管理 | infra/db/datasource.go | 目前 JSON 文件仓储；不能宣称已经有 sys_datasource 表 |
| 配置启动入口 | infra/conf/ | P01 提供启动配置装配；D10 约束 v0.x 不做 Runtime Settings |
| 身份与授权 | 当前树中尚无 apps/system、infra/auth、infra/audit | 按本期设计新建，不能引用目标树冒充已有 |
| 系统表与结构脚本 | 当前树中尚无 sql/ | 本期建立 SQLite 核心结构脚本与 `TableSpec` 清单 |
| HTTP 主链路 | P01 交付 infra/httpapi（RequestID→Recover→AccessLog、Registrar、Bind） | 本期在其后追加 Session→CSRF→Permission，不另建链路 |

## 3. 本期范围

- 以现有 infra/db 为地基核验数据库连接、事务、SQL 执行和方言能力。
- 默认 SQLite 核心 schema、幂等结构同步、结构差异预览、并发初始化保护与失败诊断。
- 用户、角色、权限、用户角色/角色权限、服务端会话、审计。
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
| 服务端会话 | 默认 DB 会话 + HttpOnly Cookie + CSRF，不默认 JWT | DECISIONS.md D06 |
| 结构同步 | `TableSpec` 清单幂等 Ensure；`DiffSchema` 只读预览 | DECISIONS.md D20 |

以下安全、结构同步和数据设计为经用户确认的本期约束。

- **统一入口**：密码哈希、Session、CSRF、登录限流和授权判定一律经 `infra/auth`；审计一律经 `infra/audit`；数据库访问一律经 `infra/db`。业务 handler 不得自建 Cookie、自写密码哈希或直接 `sql.Open`。
- **复用 P01 中间件链**：infra/httpapi 的 RequestID→Recover→AccessLog 保持不变；会话认证→CSRF→权限由 system 模块的 `protected` 守卫按固定顺序统一执行，公开路由（登录/初始化/探针）显式不经该守卫，不另建第二条链路。
- SQL-first，不引入 ORM；SQL 脚本按方言组织。
- 用户密码只存成熟算法（bcrypt）生成的哈希；会话只存 token 的 SHA-256，不存 Cookie 原文。
- 服务端是 RBAC 最终权威；前端隐藏菜单不构成授权。
- 首次管理员创建必须是原子流程；初始化完成后关闭重复初始化入口。
- 禁止删除/禁用最后一个活跃超级管理员。
- 审计记录只追加，不记录密码、session token、DSN 或凭据明文。
- 核心结构脚本与业务结构脚本分 scope；启动时幂等 Ensure，`DiffSchema` 差异只读预览，变更须显式确认。

## 6. 数据结构（本期独立设计契约）

类型映射遵循 DATA_DICT.md §1.1：主键逻辑类型为 int64；时间 UTC；API RFC3339；SQLite/MySQL/PostgreSQL 方言物理类型分别由 `sql/<dialect>/` 映射。以下 §6.1 为本期**逻辑结构**（统一用"自增主键"表示 int64 主键），§6.2 为**SQLite 参考 DDL**（物理类型），两者字段语义一致；字段变更必须同步本文件与 DATA_DICT.md。

### 6.1 核心表字段（逻辑结构）

| 表 | 字段与类型/约束 | 关系/不变量 |
|---|---|---|
| sys_user | id 自增主键；username 短字符串(64) UNIQUE NOT NULL；display_name 短字符串(64) NOT NULL；password_hash 短字符串(255) NOT NULL；status 短字符串(16) NOT NULL DEFAULT active；last_login_at 时间 NULL；created_at/updated_at 时间 NOT NULL | 禁止用户名复用；至少一个 active 超级管理员 |
| sys_role | id 自增主键；code 短字符串(64) UNIQUE NOT NULL；name 短字符串(64) NOT NULL；description 短字符串(255) NOT NULL DEFAULT ''；is_system 布尔 NOT NULL DEFAULT false；created_at/updated_at 时间 NOT NULL | 系统角色不可删除；code 发布后稳定 |
| sys_permission | id 自增主键；code 短字符串(96) UNIQUE NOT NULL；name 短字符串(64) NOT NULL；module 短字符串(64) NOT NULL；status 短字符串(16) NOT NULL DEFAULT active；description 短字符串(255) NOT NULL DEFAULT ''；created_at 时间 NOT NULL | 权限码由 Manifest 登记；不复用已发布 code；模块下线标记 orphaned |
| sys_role_permission | role_id 自增主键引用；permission_id 自增主键引用；PRIMARY KEY(role_id, permission_id) | FK 到 role/permission，关联删除按约定处理 |
| sys_user_role | user_id 自增主键引用；role_id 自增主键引用；PRIMARY KEY(user_id, role_id) | FK 到 user/role |
| sys_session | id 短字符串(64) PK（token SHA-256）；user_id 自增主键引用；ip 短字符串(64)；user_agent 短字符串(255)；expires_at/created_at/last_seen_at 时间 NOT NULL | FK 到 user；按 user_id、expires_at 建索引；禁用用户立即撤销会话 |
| sys_audit_log | id 自增主键；actor_id 自增主键 NULL；actor_name 短字符串(64)；action 短字符串(96) NOT NULL；outcome 短字符串(16) NOT NULL；target_type 短字符串(64)；target_id 短字符串(64)；summary 短字符串(255) NOT NULL；detail_json JSON NULL；ip 短字符串(64)；user_agent 短字符串(255)；request_id 短字符串(64)；created_at 时间 NOT NULL | 只追加；outcome 区分 success/failure/denied；detail 必须脱敏 |

### 6.2 SQL 关系定义（SQLite 参考 DDL）

以下为 SQLite 可执行参考；MySQL/PostgreSQL 由 `sql/<dialect>/` 按 `DATA_DICT.md` §1.1 映射生成，不在设计文档复制多份方言 DDL。

~~~sql
CREATE TABLE sys_user (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  last_login_at DATETIME NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
CREATE INDEX idx_sys_user_status ON sys_user(status);

CREATE TABLE sys_role (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_system INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE sys_permission (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  code TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  module TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'active',
  description TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL
);
CREATE INDEX idx_sys_permission_module_status ON sys_permission(module, status);

CREATE TABLE sys_role_permission (
  role_id INTEGER NOT NULL REFERENCES sys_role(id) ON DELETE CASCADE,
  permission_id INTEGER NOT NULL REFERENCES sys_permission(id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, permission_id)
);

CREATE TABLE sys_user_role (
  user_id INTEGER NOT NULL REFERENCES sys_user(id) ON DELETE CASCADE,
  role_id INTEGER NOT NULL REFERENCES sys_role(id) ON DELETE CASCADE,
  PRIMARY KEY(user_id, role_id)
);

CREATE TABLE sys_session (
  id TEXT PRIMARY KEY,
  user_id INTEGER NOT NULL REFERENCES sys_user(id) ON DELETE CASCADE,
  ip TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  expires_at DATETIME NOT NULL,
  created_at DATETIME NOT NULL,
  last_seen_at DATETIME NOT NULL
);
CREATE INDEX idx_sys_session_user_id ON sys_session(user_id);
CREATE INDEX idx_sys_session_expires_at ON sys_session(expires_at);

CREATE TABLE sys_audit_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  actor_id INTEGER NULL,
  actor_name TEXT NOT NULL DEFAULT '',
  action TEXT NOT NULL,
  outcome TEXT NOT NULL,
  target_type TEXT NOT NULL DEFAULT '',
  target_id TEXT NOT NULL DEFAULT '',
  summary TEXT NOT NULL,
  detail_json TEXT NULL,
  ip TEXT NOT NULL DEFAULT '',
  user_agent TEXT NOT NULL DEFAULT '',
  request_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL
);
CREATE INDEX idx_sys_audit_log_created_at ON sys_audit_log(created_at);
CREATE INDEX idx_sys_audit_log_actor_id ON sys_audit_log(actor_id);
CREATE INDEX idx_sys_audit_log_action ON sys_audit_log(action);
~~~

关系：一个用户可关联多个角色；一个角色可关联多个权限；会话归属用户；审计操作者允许为空（系统任务）。不得以删除权限记录的方式静默改变既有角色的权限解释；模块下线只把权限标记为 orphaned。

### 6.3 初始化与数据流

1. 加载 Bootstrap Config，按 `EASYADMIN_DB_DRIVER`/`EASYADMIN_DB_DSN` 建立默认 SQLite 数据源。
2. 执行幂等结构 Ensure（core scope）：补建缺失的表和索引；同名表冲突启动失败。
3. 结构差异只做 `DiffSchema` 预览；破坏性变更必须显式确认，不自动删改列。
4. 同步系统角色与 Manifest 权限码；不再出现在 Manifest 中的权限标记 `orphaned`，不删除。
5. 若无管理员，进入一次性 setup 状态；初始化事务写入管理员、超级管理员角色和关联。
6. 登录验证密码哈希，创建随机 session token；数据库只存 SHA-256。
7. 每个受保护请求由会话认证与服务端 RBAC 检查；危险操作写审计（含 outcome）。
8. 用户禁用/密码更改/登出时按策略撤销会话。

## 7. API、配置与权限边界

- 初始化、登录、登出、当前用户、密码更新、用户/角色/权限和审计 API 以 API.md 为唯一字段契约。
- 认证/授权实现一律经 `infra/auth`；审计一律经 `infra/audit`；不新增第二条会话或权限判定路径。
- 配置中心继续遵守 D10：只管理 Bootstrap Config；不创建 sys_setting。
- 所有写 API 先做身份验证、CSRF、权限判断，再执行用例；前端菜单不代替后端鉴权。
- 401/403/409/422 错误按 API.md 固定结构返回。
- 完成本期后 `/api/meta` 返回 `features.auth=true`，前端移除"P02 提供初始化"占位页，进入真实 setup/login 流程。

## 8. 验收标准

- 删除默认 SQLite 文件后，启动能幂等建表；重复启动不重复建表、不产生种子冲突。
- 两个并发首次初始化请求只有一个成功。
- 会话 token 原文不落库；禁用用户后会话立即失效。
- 最后一个活跃超级管理员不能被删除或禁用。
- 未登录 401、无权限 403、CSRF 失败 403、字段校验 422、唯一约束冲突 409。
- 缺表/缺索引能自动补齐；列差异只预览不自动执行；单表结构脚本失败可定位，不伪装成功。
- `/api/meta` 返回 `features.auth=true`，初始化/登录/登出/会话失效全流程可操作。
- SQLite 集成测试真实执行；PostgreSQL/MySQL 只有在 CI 提供数据库并通过方言测试后才列为支持。

## 9. 质量自评

| 维度 | 自评结论 | 评审重点 |
|---|---|---|
| 工程化最佳实践 | 幂等结构同步、事务与服务端 RBAC 构成可验证安全边界 | 并发初始化、结构同步中断和不可逆 DDL 是否有恢复方案 |
| 用户体验 | 初始化、登录、会话失效和权限错误均有明确状态 | setup 失败能否安全重试，权限不足是否可理解 |
| 改动量与复杂度 | 仅交付单实例 SQLite 身份核心，不引入 ORM、多租户或外部 IdP | 结构同步是否复用 infra/db，方言范围是否过宽 |
| 架构统一性 | 认证/授权服务端权威，API/DATA_DICT/SQL 共用字段口径 | Manifest 权限、路由授权和数据模型是否一致 |

以上为设计自评；安全实现和方言支持必须由终验测试证明。

## 10. 风险、阶段依赖与确认

- 前置阶段为 P01；P01 已于 2026-11-26 ACCEPTED。
- 密码/会话/CSRF 属安全敏感实现，必须使用成熟算法/协议，不自造加密。
- sys_datasource 的 SQL 持久化表与 sys_server 关系尚属候选；不纳入本期核心表。
- 审计数据保留期与脱敏策略需与 LOGGING_DESIGN 一致。

## 11. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 2026-11-26 | 用户 | APPROVED（含开工授权） | 已补 infra/auth 唯一入口、复用 P01 中间件链、features.auth=true、统一类型口径；命名数据源 SQL 模型不纳入本期 |


## 12. 验收结论与已知边界

**结论**：PASS（2026-11-26）。

**验收证据（真实执行）**：
- `go test ./...` 全绿；`go vet ./...` 无输出；`CGO_ENABLED=1 go test -race ./apps/system/... ./infra/auth/... ./infra/httpapi/... ./kernel/...` 通过。
- SQLite 集成：重复执行 `EnsureSchema` 幂等、`DiffSchema` 无自动差异；两路并发首次初始化仅一路成功；重复初始化返回 409。
- HTTP 实测：初始化后 `/api/meta` 返回 `initialized=true`、`features.auth=true`、用户与 CSRF；未登录 401、缺 CSRF 403、字段校验 422、重复用户名 409、登出后会话 401。
- 角色权限实测：创建角色携带 `system.audit.read`，详情返回 1 项；更新为 2 项后立即生效；未知权限返回 422。
- 越权防护实测：仅持 `system.user.read/write` 的普通管理员创建超级管理员返回 403，重置超级管理员密码返回 403；`keyword` 过滤返回 1 条。
- 可视化浏览器（Edge，可见窗口）实看：首次初始化页提交 admin/管理员/password123 后进入布局，菜单显示"系统（用户管理/角色管理/审计日志）+ 控制台"，无 console 错误、无网络错误。系统页面正文显示"页面入口未注册"属 P03 前端页面尚未交付的预期结果。

**交付物**：`infra/auth`（bcrypt、会话令牌 SHA-256、CSRF HMAC、登录限流、授权）、`infra/audit`（Recorder + SQL 实现）、`infra/db/tx.go`、`apps/system`（model/store/schema/service/module/handlers）、`sql/sqlite/core/` 7 张核心表与索引脚本、`cmd/easyadmin` 数据库与系统模块装配。

**已知边界（本期不做）**：
- 前端页面入口与交互留 P03；当前菜单可见但页面正文提示"页面入口未注册"。
- 配置中心写接口、任务中心、调度 SQL Store、文件存储与运维页面属后续阶段。
- MySQL/PostgreSQL 仅有脚本位置，未完成方言集成测试，不列为正式支持。
- 审计保留期与脱敏策略随 P04 运维域落地；当前只保证敏感字段不写入。
- 单实例边界：登录限流与初始化串行锁为进程内实现，不承诺多实例一致。

## 13. 偏差记录

| 计划 | 实际 | 原因 |
|---|---|---|
| SQLite 驱动未指定 | 使用纯 Go `modernc.org/sqlite v1.34.5` | 避免 CGO，保证单二进制与 CI 可移植 |
| catalog 查询脚本按完整路径注册 | 同时注册完整路径与裸文件名 | `infra/db` 目录查询按基名读取，避免另造第二套脚本源 |
| CSRF token 可落库或签名 | 采用进程内密钥对会话哈希做 HMAC，不落库 | 与会话绑定且减少敏感数据持久化 |
| API.md 含角色权限与角色详情 | 验收阶段补齐 `GET /roles/{id}`、角色权限读写与越权防护 | 使 RBAC 角色权限真正可配置，避免文档契约空转 |
| `PUT /users/{id}` 描述含状态 | 收敛为展示名与角色，状态走独立端点 | 同一能力只保留一个受支持入口 |
