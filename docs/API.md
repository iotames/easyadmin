# HTTP API 契约

本文定义 EasyAdmin 管理端 API 的统一约定与核心端点。实现与文档同步更新；接口变更必须补测试。

## 1. 总则

- 所有服务端 API 统一以 `/api/` 开头，不设置路径版本号；内部管理 UI 与外部调用方消费同一套契约。
- WebUI 页面路由与静态资源不得占用 `/api/`；
- 管理端 API 与内置 WebUI 同源，默认只服务单实例；
- API 从首个公开版本起遵循向下兼容：只增不改、不删字段、不改变既有字段类型与语义；
- 请求与响应均为 UTF-8 JSON，文件下载除外；
- 所有时间输出 RFC3339，例如 `2026-11-22T10:30:00+08:00`。

### 1.1 兼容规则

允许的变更：

- 新增端点；
- 请求中新增可选字段；
- 响应中新增字段；
- 枚举新增取值（前端必须能处理未知值）；
- 新增错误码（旧错误码语义保持不变）。

禁止的变更：

- 删除或重命名既有字段；
- 改变字段类型、单位、时区或业务语义；
- 把可选字段改为必填；
- 改变既有端点的默认行为而不提供显式开关；
- 用同名字段承载不同含义。

弃用流程：

1. 在 API 文档标记 `deprecated`，说明替代方案与计划移除版本；
2. 至少保留一个正式版本的兼容期；
3. 响应中可增加 `Deprecation` 头提示调用方；
4. 兼容期结束后才可移除，且必须经过重大决策评审。

确实无法兼容时，新增独立端点或新资源路径承载新语义；不通过 `/v1`、`/v2` 路径版本来回避兼容责任。若未来必须引入路径版本，作为新的架构决策单独评审，不作为默认方案。

## 2. 认证与 CSRF

### 2.1 会话

- 登录成功后服务端设置 `ea_session` Cookie：`HttpOnly`、`SameSite=Lax`、HTTPS 下 `Secure`；
- 会话失效返回 401；
- 登出删除服务端会话并清 Cookie；
- 禁用用户立即失效其全部会话。

### 2.2 CSRF

- 除登录、初始化、健康探针外，所有 `POST/PUT/PATCH/DELETE` 必须带 `X-CSRF-Token`；
- CSRF token 由 `/api/meta` 返回，前端存入内存，不写 `localStorage`；
- token 与会话绑定，会话变更后重新获取；
- 缺失或错误返回 403，文案提示刷新页面后重试。

### 2.3 公开端点

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/meta` | 应用信息、登录态、菜单、权限、CSRF token |
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/setup` | 首次初始化（仅未初始化时可用） |
| GET | `/healthz` | 存活探针（非 `/api/`） |
| GET | `/readyz` | 就绪探针（非 `/api/`） |

## 3. 响应包

### 3.1 成功

```json
{
  "ok": true,
  "data": {},
  "meta": { "request_id": "req_01J...", "warning": "" }
}
```

`meta.warning` 可选：用于“操作已生效但存在非阻断问题”，如当前支持持久化的配置写入失败。前端必须展示该警告。`request_id` 成功和失败时都只出现在顶层 `meta.request_id`，禁止在 `error` 内重复定义。

### 3.2 失败

```json
{
  "ok": false,
  "error": {
    "code": "SYSTEM_USER_NOT_FOUND",
    "message": "用户不存在或已被删除，请返回列表刷新后重试",
    "details": {}
  },
  "meta": { "request_id": "req_01J..." }
}
```


`error.details` 是可选的结构化扩展对象；没有额外信息时返回空对象 `{}`。字段校验错误的结构由下节统一定义，不得由各模块自行设计。

### 3.3 字段校验失败（HTTP 422）

字段校验失败使用稳定错误码 `COMMON_VALIDATION_FAILED`。当错误能够定位到请求字段时，必须在 `error.details.fields` 中返回字段错误：

```json
{
  "ok": false,
  "error": {
    "code": "COMMON_VALIDATION_FAILED",
    "message": "请求字段校验失败",
    "details": {
      "fields": {
        "username": ["用户名不能为空"],
        "profile.email": ["邮箱格式不正确"],
        "items[0].name": ["名称长度不能超过 64 个字符"]
      }
    }
  },
  "meta": { "request_id": "req_01J..." }
}
```

契约规则：

- `fields` 是对象，键为稳定字段路径，值为**非空字符串数组**；即使某字段只有一条错误，也必须使用数组，以支持同一字段返回多条错误。
- 字段路径使用 API 对外字段名（JSON 使用的 `snake_case` 名称），不得使用 Go 结构体字段名、数据库列名或界面标签。
- 顶层字段直接使用字段名，例如 `username`；嵌套对象使用点号，例如 `profile.email`；数组元素使用从 0 开始的下标，例如 `items[0].name`。Query 参数使用其公开参数名。
- 每条消息必须面向用户、可安全展示。不得在 `fields` 或其他 `details` 中回显密码、令牌、未经处理的用户输入、SQL、堆栈或内部 Cause。
- 只有能够定位到请求字段的错误才放入 `fields`；非字段级错误不应伪造字段路径，前端应展示顶层 `error.message`。

### 3.4 分页

```json
{
  "ok": true,
  "data": { "items": [] },
  "meta": {
    "request_id": "req_01J...",
    "page": 1,
    "page_size": 20,
    "total": 0
  }
}
```

## 4. 通用参数

| 参数 | 位置 | 说明 |
|---|---|---|
| `page` | query | 页码，从 1 开始，默认 1 |
| `page_size` | query | 每页条数，默认 20，最大 200 |
| `keyword` | query | 关键字，按资源定义匹配字段 |
| `sort` | query | 排序字段，必须在资源白名单内 |
| `order` | query | `asc` / `desc`，默认 `desc` |

未知字段、非法排序字段、JSON 格式错误和参数类型转换失败返回 400；请求成功绑定后，字段或跨字段输入约束不满足返回 422；因已存在数据或资源当前状态产生的冲突返回 409；上传内容超过配置大小限制返回 413。

## 5. 元信息

### `GET /api/meta`

> 实现状态：P00 返回 `app{name,version}`、`initialized=false`、`user=null`、空的 `permissions/menus` 与空 `csrf_token`；P01 起 `permissions/menus` 由已登记 Manifest 组装。P01 期间返回 `features.auth=false`（前端显示"认证未启用"占位页）；P02 完成后置 `features.auth=true`，并补齐初始化/登录态与 CSRF。

返回：

```json
{
  "ok": true,
  "data": {
    "app": {
      "name": "EasyAdmin",
      "version": "0.1.0",
      "env": "prod"
    },
    "initialized": true,
    "user": {
      "id": 1,
      "username": "admin",
      "display_name": "管理员",
      "is_super_admin": true
    },
    "permissions": ["system.user.read", "system.user.write"],
    "menus": [],
    "features": { "tasks": true, "schedules": true, "files": true },
    "csrf_token": "..."
  }
}
```

未登录时 `user` 为 `null`，`menus` 为空，`initialized` 仍返回。

## 6. 认证端点

### `POST /api/auth/login`

请求：

```json
{ "username": "admin", "password": "..." }
```

成功：设置 Cookie，返回用户信息。

失败：

- 401：用户名或密码错误（不区分具体原因）；
- 429：登录失败次数过多，返回 `retry_after_seconds`。

### `POST /api/auth/logout`

删除当前会话，返回 `{ "ok": true }`。

### `POST /api/auth/setup`

仅在 `initialized=false` 时可用。

请求：

```json
{ "username": "admin", "display_name": "管理员", "password": "..." }
```

规则：

- 回环地址可直接设置；
- 非回环地址要求 `EASYADMIN_ADMIN_PASSWORD` 已配置，页面只确认与改密；
- 初始化完成后不可重复调用。

### `GET /api/auth/me`

返回当前用户、角色、权限。未登录 401。

### `PUT /api/auth/password`

请求：

```json
{ "old_password": "...", "new_password": "..." }
```

成功后可保留当前会话，其他会话失效。

## 7. 用户

权限：`system.user.read` / `system.user.write`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/users` | 列表，支持 `keyword`、`status` |
| POST | `/api/system/users` | 创建 |
| GET | `/api/system/users/{id}` | 详情 |
| PUT | `/api/system/users/{id}` | 更新展示名、角色、状态 |
| PUT | `/api/system/users/{id}/status` | 启用/禁用 |
| PUT | `/api/system/users/{id}/password` | 管理员重置密码 |

用户对象：

```json
{
  "id": 1,
  "username": "admin",
  "display_name": "管理员",
  "status": "active",
  "roles": [{ "id": 1, "code": "super_admin", "name": "超级管理员" }],
  "last_login_at": "2026-11-22T10:30:00+08:00",
  "created_at": "2026-11-01T09:00:00+08:00"
}
```

规则：

- 不返回 `password_hash`；
- 用户名创建后不可改；
- 不能禁用/删除最后一个活跃超级管理员；
- 禁用用户时同时删除其会话。

## 8. 角色与权限

权限：`system.role.read` / `system.role.write`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/roles` | 角色列表 |
| POST | `/api/system/roles` | 创建角色 |
| GET | `/api/system/roles/{id}` | 角色详情（含权限码） |
| PUT | `/api/system/roles/{id}` | 更新名称、描述、权限 |
| DELETE | `/api/system/roles/{id}` | 删除非系统角色 |
| GET | `/api/system/permissions` | 全部权限，按模块分组 |

规则：

- `is_system=true` 的角色不可删除、不可改 `code`；
- 删除角色前检查是否仍被用户引用；
- 权限变更后相关用户下一次请求即生效。

## 9. 配置中心

权限：system.config.read / system.config.write

v0.x 仅管理 Bootstrap Config（进程启动配置），不提供数据库 Runtime Settings。GET 读取已注册启动配置的安全元数据；PUT 只允许显式声明可写、且当前来源具备受支持持久化路径的配置项。环境变量、命令行、容器注入和 immutable 配置不能通过写 .env 伪装成已持久化。

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | /api/system/config | 分组列出已注册的 Bootstrap Config 元数据 |
| PUT | /api/system/config/{key} | 修改显式允许写入、且持久化来源可控的单个启动配置项 |

配置项对象：

~~~json
{
  "key": "APP_LISTEN_ADDR",
  "title": "HTTP 监听地址",
  "type": "string",
  "value": ":19527",
  "default": ":19527",
  "source": "file",
  "hot_reload": "restart",
  "secret": false,
  "writable": true,
  "persisted": true,
  "restart_required": false,
  "group": "应用"
}
~~~

写入成功响应必须明确返回 applied、persisted、restart_required 和 source，不能仅以 HTTP 200 表示配置已生效。restart 项写入文件后只表示候选值已持久化，不表示当前进程监听地址已改变；必须设置 restart_required=true、applied=false。只有 hot 项且消费者更新成功时才可 applied=true。

规则：

- source 至少区分 default、file、environment、command_line；容器平台注入值按 environment 来源处理。
- writable=true 仅适用于应用明确管理的可写配置文件来源，且必须采用原子写入和并发冲突保护；默认值快照 default.env 不得当作用户配置写回目标。
- 当前值由环境变量或命令行提供时，.env 写回不会覆盖实际生效值；此类项必须拒绝写入并返回 SYSTEM_CONFIG_READONLY（409）。
- 密钥类只返回 configured/masked 状态，不回显原文；密钥值默认不可读、不可写。
- 仅内存 Manager.Set 不等于持久化。实现必须由配置适配层提供安全写入与来源核验；不能将 Set 成功描述为持久化成功。
- 配置修改必须审计；持久化失败不得伪装成功。Runtime Settings、sys_setting 和数据库覆盖优先级不属于 v0.x。

## 10. 任务中心

权限：`system.task.read` / `system.task.write`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/tasks` | 任务列表，支持 `kind`、`status`、时间范围 |
| GET | `/api/system/tasks/{task_no}` | 任务详情 |
| GET | `/api/system/tasks/{task_no}/logs` | 任务日志，支持 `after_seq` |
| POST | `/api/system/tasks/{task_no}/cancel` | 取消 |
| POST | `/api/system/tasks/{task_no}/retry` | 重试（需该 kind 注册重试函数） |
| GET | `/api/system/tasks/{task_no}/children` | 子任务 |

任务状态：`pending`、`running`、`success`、`failed`、`cancelled`、`skipped`、`interrupted`。

进度：

```json
{
  "progress_current": 120,
  "progress_total": 500,
  "percent": 24
}
```

`progress_total=0` 表示总量未知，前端显示不确定进度条。

## 11. 调度器

权限：`system.schedule.read` / `system.schedule.write`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/schedules` | 列表 |
| PUT | `/api/system/schedules/{name}` | 修改开关与周期 |
| POST | `/api/system/schedules/{name}/run` | 手动触发一次 |

规则：

- 周期为 1~525600 分钟，由 `plan_key` 指向的配置项提供；`plan_key` 为空时使用代码注册的 Plan，只读；
- 修改周期只写 `plan_key` 对应配置并立即重算 `next_run_at`；`plan_key` 为空时拒绝修改并返回 `SYSTEM_SCHEDULE_PLAN_IMMUTABLE`（409）；
- 上一轮未结束时触发记 `skipped`；
- 手动触发仍走任务中心并发规则。

## 12. 日志与审计

权限：`system.log.read` / `system.audit.read`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/logs` | 运行日志查询 |
| GET | `/api/system/audit` | 审计查询 |

运行日志查询参数：`level`、`from`、`to`、`keyword`、`page`、`page_size`。

审计查询参数：`actor_id`、`action`、`outcome`、`from`、`to`、`page`、`page_size`；`outcome` 取 `success` / `failure` / `denied`，响应逐条返回该字段。

日志与审计均为只读接口，不提供删除 API；清理走调度任务。

## 13. 文件

权限：`system.file.read` / `system.file.write`

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/system/files` | 文件列表 |
| POST | `/api/system/files` | 上传（multipart） |
| GET | `/api/system/files/{id}/download` | 下载 |
| DELETE | `/api/system/files/{id}` | 软删除 |

规则：

- 上传大小上限可配，默认 32MB；
- 文件名只用于展示，对象 key 由服务端生成；
- 相同内容重复上传时复用已有对象记录并恢复软删除标记，不产生重复物理文件；
- 下载必须鉴权，不得直接暴露存储目录；
- 删除前检查业务引用，或明确标记为待清理。

## 14. 健康探针

### `GET /healthz`

进程存活即返回 200：

```json
{ "ok": true, "status": "alive" }
```

### `GET /readyz`

依赖就绪才返回 200，否则 503：

```json
{
  "ok": true,
  "status": "ready",
  "checks": {
    "database": "ok",
    "schema": "ok"
  }
}
```

> 实现状态：P00 的 `checks` 仅包含 `app`；数据库与结构同步检查随 P01/P02 接入。

## 15. 错误码规范

格式：`<MODULE>_<RESOURCE>_<REASON>`，全大写，例如：

| 错误码 | 状态 | 场景 |
|---|---|---|
| `AUTH_REQUIRED` | 401 | 未登录 |
| `AUTH_INVALID_CREDENTIALS` | 401 | 用户名或密码错误 |
| `AUTH_FORBIDDEN` | 403 | 无权限 |
| `AUTH_CSRF_INVALID` | 403 | CSRF token 缺失或错误 |
| `AUTH_RATE_LIMITED` | 429 | 登录限流 |
| `SYSTEM_USER_NOT_FOUND` | 404 | 用户不存在 |
| `SYSTEM_USER_DUPLICATE` | 409 | 用户名已存在 |
| `SYSTEM_ROLE_IN_USE` | 409 | 角色仍被用户引用 |
| `SYSTEM_CONFIG_READONLY` | 409 | 配置项来源不可写、策略为 immutable 或不允许在线修改 |
| `SYSTEM_TASK_NOT_RETRYABLE` | 409 | 任务当前状态或类型不支持重试 |
| `SYSTEM_SCHEDULE_PLAN_IMMUTABLE` | 409 | 调度周期由代码注册 Plan 管理，不可在线修改 |
| `SYSTEM_FILE_TOO_LARGE` | 413 | 上传内容超过大小限制 |
| `COMMON_INVALID_ARGUMENT` | 400 | 参数格式、结构或类型错误 |
| `COMMON_VALIDATION_FAILED` | 422 | 字段或跨字段输入校验失败 |
| `COMMON_NOT_FOUND` | 404 | 资源不存在 |
| `COMMON_METHOD_NOT_ALLOWED` | 405 | 请求方法不支持 |
| `COMMON_INTERNAL` | 500 | 未预期错误 |
| `COMMON_UNAVAILABLE` | 503 | 依赖不可用 |
