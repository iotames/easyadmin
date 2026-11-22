# EasyAdmin 工程规范

## 1. 最高原则：统一入口

统一入口不是把代码集中到一个万能包，而是让同类能力只有一种受支持调用方式。

### 1.1 后端入口

| 能力 | 唯一入口 | 禁止绕行 |
|---|---|---|
| 启动配置 | `infra/conf` | 业务 `os.Getenv`、私读 `.env` |
| 应用日志 | `infra/logx` | 业务自行创建 logger/writer |
| 数据库/SQL/事务 | `infra/db` | 业务 `sql.Open`、私建连接池 |
| HTTP 路由/中间件/响应 | `infra/httpapi` | 模块私自监听、手写 JSON 响应 |
| 错误 | `infra/apperror` | 直接把底层 `err.Error()` 返回用户 |
| 登录/会话/RBAC | `infra/auth` | 自建 Cookie、只在前端鉴权 |
| 审计 | `infra/audit` | 把审计当普通 Info 日志 |
| 长任务 | `infra/tasks` | handler 里直接起无主 goroutine |
| 定时触发 | `infra/scheduler` | 模块自建 ticker 执行长任务 |
| 文件 | `infra/storage` | 直接暴露本地目录或厂商 SDK |

### 1.2 前端入口

| 能力 | 唯一入口 | 禁止绕行 |
|---|---|---|
| 请求 | `core/api.js` | 页面裸 `fetch` |
| 提示 | `core/notify.js` | 页面自造 toast/alert |
| 路由 | `core/router.js` | 页面自行改全局 location 状态 |
| 布局 | `core/layout.js` | 页面操作顶栏/侧栏 DOM |
| 菜单 | `core/menu.js` + 后端 Manifest | 前端硬编码菜单树 |
| 状态 | `core/store.js` | 无约束全局变量 |
| 主题 | `core/theme.js` | 页面散落主题常量 |
| DOM 安全 | `core/dom.js` | 不可信内容写入 `innerHTML` |

## 2. 依赖方向

```text
cmd（组合根）→ kernel / infra / apps
apps → kernel 公开契约 + infra 公开服务
infra → easy 系列适配器 + Go 标准库
kernel → 标准库与框架自身契约
webui → 同源 /api 契约
```

`kernel` 不依赖具体 infra；`infra` 不依赖 apps；`apps` 不互相 import；easy 系列不依赖 EasyAdmin。

## 3. 命名约定

- Go package 使用短小、小写、单一职责名称；避免万能 `common`、`utils`、`manager` 包。
- 目录命名遵循 `DECISIONS.md` D18：顶层用 `cmd`、`kernel`、`infra`、`apps`、`webui`、`sql`、`templates`、`examples`、`lib`、`docs`；能力包用 `conf`、`logx`、`db`、`httpapi`、`apperror`、`auth`、`audit`、`tasks`、`scheduler`、`storage`。
- 模块 ID 使用小写点分：`system.users`、`system.roles`、`system.config`。
- 权限码使用小写点分：`system.user.read`、`system.user.write`。
- 环境配置键使用大写下划线；内部配置 catalog key 可使用模块命名空间。
- API JSON 字段使用 `snake_case`；Go 字段使用惯用 MixedCaps。
- SQL 表名 `sys_` 前缀；字段使用 `snake_case`；时间字段统一 `_at`。

## 4. 错误处理

业务错误通过 `apperror` 建立稳定 Code、HTTP Status、面向用户 Message、可选安全 Details 和内部 Cause。错误文案须包含：发生了什么、原因（可安全披露时）和下一步。错误码不复用；前端不得按中文文案分支。内部 Cause 和堆栈进入日志，不进入 HTTP 响应。

400 用于 JSON 格式、参数结构或类型转换错误；422 用于请求绑定成功后的字段及跨字段输入校验；409 用于已存在数据或资源当前状态造成的冲突；401 未认证；403 无权限/CSRF；404 不存在；429 限流；500 未预期错误；503 依赖不可用。字段级 422 的 `details.fields` 结构遵循 `API.md`。

## 5. Go 编码与并发

- 所有 IO 方法接收 `context.Context`；网络、数据库和关闭操作设置超时。
- 导出符号写清注释；错误用 `%w` 保留错误链；不吞错。
- 优先小接口、构造函数注入、组合；接口由消费方定义。
- goroutine 必须有所有者、退出条件、取消机制和测试；禁止无人管理后台 goroutine。
- 可变共享状态明确锁或原子快照；不把可变对象跨模块共享。
- 关键并发路径跑 `go test -race`。

## 6. 数据与 SQL

- SQL 外置 `.sql`，按方言组织；不引入 ORM。
- 所有查询参数绑定；动态排序列、表标识符来自白名单。
- 事务边界在服务层；事务中不做外部网络调用。
- 列表有默认分页和硬上限；统计查询必须有合理范围。
- SQL 加载顺序、外挂覆盖和 fallback 由 Data 层统一实现并测试。
- 时间 UTC 存储，RFC3339 输出，浏览器本地化展示。
- 迁移前进；执行失败要有恢复说明；不能启动时自动删改列。

## 7. HTTP/API

- API 路径固定 `/api/`；WebUI 路由不占用此命名空间。
- 不设路径版本号；接口向下兼容，只增不改：允许新增端点、可选字段、响应字段和枚举值，禁止删除/重命名/改类型/改语义。
- API 变更必须有兼容性测试；弃用需文档标记、至少一个正式版本兼容期和重大决策评审。
- 所有响应带 `meta.request_id`；响应结构见 `API.md`。
- 路由、权限、菜单和页面标识由模块显式登记。
- 中间件顺序固定并有集成测试；公开路由豁免清单显式维护。
- 认证、授权和 CSRF 在服务端执行；前端权限逻辑只用于显示辅助。
- 幂等要求高的操作支持幂等键或版本冲突检查。

### 7.1 请求绑定与校验分层

- `httpapi.Bind` 负责解析 Query/JSON/Form、类型转换和绑定；JSON 格式错误、未知字段及类型转换失败返回 400，不要与绑定成功后的字段校验混为一谈。
- 字段级校验统一处理必填、长度、格式、范围、枚举及跨字段输入约束；失败返回 422，字段错误按 `API.md` 的 `details.fields` 契约返回。
- 领域规则（唯一性、资源当前状态、跨实体约束和权限）由 Service/Repository 与数据库约束负责；已存在数据或当前资源状态导致的冲突返回 409，权限失败返回 403。通用校验器不得访问数据库或网络。
- WebUI 即时校验用于改善填写体验，服务端校验始终是权威结果；前端必须正确展示 422 的字段错误。
- 自定义规则显式注册、可单测；不得通过 `init()`、目录扫描或隐式反射发现规则。
- 是否需要独立 `infra/validate` 包，在 P1 基于真实请求 DTO、重复规则和调用方数量评估；在证明收益前不新增依赖。

## 8. 模块开发清单

- [ ] 唯一 ID、版本、描述、依赖和默认启用策略。
- [ ] Declare 只登记配置，不做 IO。
- [ ] Register 只登记已启用模块的运行资源。
- [ ] Service/Repository/API 分层清楚，依赖经构造函数传入。
- [ ] SQL 按方言组织并登记给迁移器。
- [ ] Manifest 菜单、页面、权限与前端入口一致。
- [ ] 写操作按风险显式审计。
- [ ] 生命周期资源可关闭；模块禁用时系统按依赖规则仍可运行。
- [ ] 单测、API 测试、浏览器测试和文档同步。

## 9. 配置项规范

配置必须提供 key、类型、默认值、标题、说明、模块归属、来源类别、敏感级别、校验、热更类别和持久化语义。配置注册在 Declare 阶段完成。密钥不得普通回显。在线修改只允许运行期可写且有持久化保证的字段；不能持久化或被环境变量覆盖时必须明确告知。

## 10. 安全与审计

密码哈希使用成熟算法；会话 Cookie 设置 HttpOnly、SameSite，HTTPS 设置 Secure；写操作有 CSRF。输入在服务端校验，输出按 HTML/URL/SQL 上下文处理。文件名不能成为物理路径；存储下载走鉴权 API。危险操作审计并要求二次确认。

审计事件至少包含 actor、action、resource、result、time、request_id；敏感内容脱敏。审计记录普通用户不可修改，保留期与归档行为明确。

## 11. 测试与质量

- Go：标准 `testing`、表驱动；单测不依赖外部服务；集成测试使用独立 DSN 和清理策略。
- HTTP：成功、参数错误、未授权、无权限、依赖异常和 panic 恢复。
- DB：事务提交/回滚、迁移重复执行、方言差异和并发初始化。
- Task/Scheduler：注入可控时钟，不 sleep；取消、重启、重复提交和关闭测试。
- Frontend：浏览器内 core/component/module/architecture 用例；关键交互实看。
- 架构测试能对故意违规 import、裸 fetch、底层 DB 使用、Manifest 漂移失败。

## 12. 文档与评审

配置、API、数据字典、菜单、页面、模块、权限和公共接口变化必须同步专题文档。架构决策写入 `DECISIONS.md`。评审按次序检查：是否绕开入口、是否反转依赖、是否引入隐式状态、错误是否可读、日志/审计是否完整、测试是否覆盖、文档是否同步、是否能以更少抽象实现。
