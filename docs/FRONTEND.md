# EasyAdmin 前端设计

## 1. 设计立场

前端采用**原生 ES Module + 零构建链**：

- 不依赖 Node、npm、Vite、Webpack；
- 不引入 Vue/React 等运行时框架；
- 浏览器原生 `import()` 实现按页面按需加载；
- 生产环境静态资源经 `go:embed` 内嵌进二进制；
- 开发环境 `-tags dev` 直接读磁盘，改文件刷新即生效。

为什么不用现代前端框架：

1. EasyAdmin 的界面以中后台表格、表单、配置为主，交互复杂度可控；
2. 框架目标是“一条 Go 命令跑起来”，构建链会把起步成本重新拉高；
3. 原生 ESM 已足够提供模块边界与懒加载，比传统 IIFE + 全局命名空间更清晰；
4. 少一层工具链，就少一类“版本不兼容、构建失败、产物不一致”的问题。

代价与边界：

- 不使用 JSX、单文件组件、TypeScript；
- 组件用普通函数/类返回 DOM，显式挂载与卸载；
- 复杂可视化（图表、编辑器）作为可选重组件动态加载，不进入首屏；
- 如果未来确实需要构建工具，只作为**可选开发增强**，不能成为运行前提。

## 2. 目录结构

```text
webui/
├── index.html                  # 唯一 HTML 入口，只加载 core/boot.js
├── embed.go                    # go:embed index.html + assets
├── embed_dev.go                # -tags dev 读磁盘
├── assets/
│   ├── core/
│   │   ├── boot.js             # 启动流程
│   │   ├── router.js           # 路由与动态 import
│   │   ├── layout.js           # 布局与插槽（全局区域唯一 owner）
│   │   ├── menu.js             # 菜单渲染
│   │   ├── api.js              # 唯一请求入口
│   │   ├── notify.js           # 唯一提示入口
│   │   ├── theme.js            # 主题切换
│   │   ├── store.js            # 显式状态容器
│   │   ├── perm.js             # 权限判断（UI 辅助）
│   │   ├── format.js           # 日期/字节/数字格式化
│   │   └── dom.js              # 安全 DOM 构造与转义
│   ├── components/
│   │   ├── page-header.js
│   │   ├── data-table.js
│   │   ├── form.js
│   │   ├── filter-bar.js
│   │   ├── modal.js
│   │   ├── drawer.js
│   │   ├── tabs.js
│   │   ├── empty.js
│   │   └── pagination.js
│   ├── pages/
│   │   ├── dashboard/
│   │   │   └── index.js
│   │   ├── system/
│   │   │   ├── users/index.js
│   │   │   ├── roles/index.js
│   │   │   ├── config/index.js
│   │   │   ├── logs/index.js
│   │   │   ├── tasks/index.js
│   │   │   └── ...
│   │   └── profile/
│   │       └── index.js
│   └── styles/
│       ├── tokens.css          # 颜色/间距/字号/圆角变量
│       ├── base.css            # 重置与基础排版
│       ├── layout.css          # 顶栏/侧栏/内容区
│       └── components.css      # 通用组件样式
└── _test/
    ├── index.html
    ├── runner.js
    ├── cases_core.js
    ├── cases_comp.js
    ├── cases_module.js
    └── cases_arch.js
```

模块 ID 与目录保持一致：

| 模块 ID | 前端入口 |
|---|---|
| `dashboard` | `assets/pages/dashboard/index.js` |
| `system.users` | `assets/pages/system/users/index.js` |
| `system.roles` | `assets/pages/system/roles/index.js` |

## 3. 启动流程

`index.html` 保持极薄：

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>EasyAdmin</title>
  <link rel="stylesheet" href="/assets/styles/tokens.css">
  <link rel="stylesheet" href="/assets/styles/base.css">
  <link rel="stylesheet" href="/assets/styles/layout.css">
  <link rel="stylesheet" href="/assets/styles/components.css">
  <script type="module" src="/assets/core/boot.js"></script>
</head>
<body>
  <div id="app" aria-live="polite"></div>
  <div id="toast-root"></div>
  <div id="modal-root"></div>
</body>
</html>
```

启动顺序：

```text
1. 读取本地主题偏好，尽早设置 data-theme（防闪烁）
2. GET /api/meta
   ├─ 未登录 → 渲染登录页
   ├─ 未初始化 → 渲染首次初始化页
   └─ 已登录 → 继续
3. 初始化 store（用户、权限、菜单、应用信息）
4. 渲染布局骨架：顶栏 / 侧栏 / 面包屑 / 内容区
5. 解析当前 path，匹配页面描述
6. 动态 import 页面入口
7. 调用 page.mount(ctx)
8. 监听 popstate 与菜单点击，切换页面时先 unmount 旧页面
```

约束：

- 只有 `boot.js` 是同步入口，其余模块都按需 `import()`。
- `boot.js` 不包含业务判断；业务判断在页面或模块内。
- 启动失败必须渲染可读的错误页，不能白屏。

## 4. 布局系统

```text
┌──────────────────────────────────────────────────────────────┐
│ Topbar                                                       │
│ [品牌] [环境标识]                  [全局状态] [主题] [用户菜单] │
├───────────────┬──────────────────────────────────────────────┤
│ Sidebar       │ Breadcrumb                                   │
│               ├──────────────────────────────────────────────┤
│ 菜单分组       │ PageHeader                                   │
│  - 仪表盘      │  标题 / 描述 / 页面级操作                     │
│  - 系统管理    ├──────────────────────────────────────────────┤
│    - 用户      │ Page Content                                 │
│    - 角色      │                                              │
│    - 配置      │                                              │
│               │                                              │
├───────────────┴──────────────────────────────────────────────┤
│ Footer（可选：版本、构建信息）                                 │
└──────────────────────────────────────────────────────────────┘
```

### 4.1 布局插槽

| 插槽 | Owner | 页面可否修改 | 用途 |
|---|---|---|---|
| `brand` | `layout.js` | 否 | 产品名/Logo |
| `env` | `layout.js` | 否 | 环境标识（dev/prod） |
| `global-status` | `layout.js` | 否 | 数据库/任务等全局状态 |
| `theme` | `layout.js` | 否 | 主题切换 |
| `user-menu` | `layout.js` | 否 | 当前用户、退出 |
| `sidebar` | `menu.js` | 否 | 全局菜单 |
| `breadcrumb` | `router.js` | 页面通过 ctx 提供数据 | 面包屑 |
| `page-header` | 页面 | 是 | 标题、描述、页面级操作 |
| `content` | 页面 | 是 | 页面主体 |

红线：**页面不得直接操作顶栏、侧栏、用户菜单、全局状态等全局 DOM。** 需要影响全局时，只能通过 `ctx` 提供的接口（如 `ctx.setBreadcrumb`、`ctx.setTitle`）。

为什么：全局区域的定义是“所有页面一致”。一旦页面能按需增删全局元素，全局就不成立，且页面越多越失控。

### 4.2 响应式

- 桌面优先，最小支持宽度 1024px；
- 小于 1280px 时侧栏可折叠为图标栏；
- 小于 900px 时侧栏改为抽屉，内容区全宽；
- 表格在窄屏允许横向滚动，不强行改成卡片导致信息丢失；
- 表单在窄屏改为单列。

## 5. 菜单系统

### 5.1 单一事实源

菜单由后端模块 Manifest 声明，`GET /api/meta` 返回按当前用户权限过滤后的菜单树。前端不得硬编码菜单项。

```json
{
  "menus": [
    {
      "key": "dashboard",
      "title": "仪表盘",
      "path": "/dashboard",
      "icon": "home",
      "children": []
    },
    {
      "key": "system",
      "title": "系统管理",
      "children": [
        { "key": "system.users", "title": "用户管理", "path": "/system/users", "permission": "system.user.read" }
      ]
    }
  ]
}
```

### 5.2 渲染规则

- 一级菜单为分组，二级为页面；
- 当前路径高亮对应项，并展开父分组；
- 菜单项按 Manifest 声明顺序排列，不按字母猜；
- 为空的分组不渲染；
- 菜单点击只做客户端路由，不整页刷新；
- 菜单状态由 `menu.js` 拥有，页面不得直接修改菜单 DOM。

### 5.3 权限

- 后端返回前已完成权限过滤，前端不再显示“无权但灰掉”的菜单（避免泄露结构）；
- 前端 `perm.can()` 只用于按钮级辅助隐藏；
- 真正的安全边界永远在后端路由权限校验。

## 6. 路由与按需加载

### 6.1 路由表

前端维护“模块 ID → 入口文件”的静态映射：core 页面在 `core/router.js` 登记，业务页面在项目自己的 `webui/pages.js` 登记。路径来自后端菜单/页面描述：

```js
// core/router.js
const entries = {
  'dashboard':        () => import('../pages/dashboard/index.js'),
  'system.users':     () => import('../pages/system/users/index.js'),
  'system.roles':     () => import('../pages/system/roles/index.js'),
  'system.config':    () => import('../pages/system/config/index.js'),
  'system.logs':      () => import('../pages/system/logs/index.js'),
  'system.tasks':     () => import('../pages/system/tasks/index.js'),
  'system.schedules': () => import('../pages/system/schedules/index.js'),
  'profile':          () => import('../pages/profile/index.js'),
};
```

新增业务模块必须同时：

1. 在 Go 模块 Manifest 声明 `Pages`；
2. 在项目 `webui/pages.js` 注册入口；
3. 在 `_test/cases_arch.js` 的清单一致性检查中通过。

缺一即构建/测试报红，避免“后端有菜单、前端打不开”。

### 6.4 业务页面扩展

EasyAdmin core 资源由框架内嵌，业务页面由业务项目自己的 `webui/` 提供：

```text
/assets/        EasyAdmin core 资源
/assets/app/    业务 webui 资源
```

业务项目导出 `webui.Assets`，在组合根传给 App；启动时 core 路由表与 `webui/pages.js` 合并：

```js
// webui/pages.js
export default {
  'orders': () => import('./assets/pages/orders/index.js'),
};
```

规则：

- Page ID 全局唯一；冲突在启动期失败，不静默覆盖 core 页面。
- `/assets/app/pages.js` 不存在时直接跳过，不影响 core 页面。
- 业务页面使用稳定的 `ctx`、`api`、`notify`、通用组件和 CSS 变量，不复制 core 组件。
- 框架升级只替换 `/assets/` 下的 core 资源，不改写 `/assets/app/` 下的业务页面。
- 业务需要改全局布局或 core 组件时，先确认是否应提交上游；不得在业务侧直接改 core DOM。

### 6.2 页面生命周期

页面模块默认导出：

```js
export default {
  id: 'system.users',

  async mount(ctx) {
    // ctx.el: 页面内容容器
    // ctx.api: 统一请求入口
    // ctx.notify: 统一提示
    // ctx.perm: 权限判断
    // ctx.route: 当前路由信息
    // ctx.setTitle / ctx.setBreadcrumb / ctx.setActions
  },

  unmount() {
    // 清理定时器、事件监听、未完成请求
  }
};
```

规则：

- 页面只操作 `ctx.el` 内部，不碰全局区域；
- 切换页面先调用旧页面 `unmount()`，再挂载新页面；
- 页面内异步请求必须可取消或在 unmount 后忽略结果；
- 页面不得在模块顶层执行请求或 DOM 操作；
- 页面级状态留在页面闭包内，确需跨页共享才进 `store`。

### 6.3 懒加载验证

验收时打开浏览器网络面板：

- 首次进入仪表盘，不应加载用户/角色/配置页 JS；
- 点击“用户管理”后，才出现 `pages/system/users/index.js` 请求；
- 图表/编辑器等重组件，只在对应页面/操作时加载。

## 7. 页面上下文（ctx）

```js
ctx = {
  el,                  // 页面内容容器（HTMLElement）
  route: { path, query, moduleId },
  api,                 // 统一请求入口
  notify,              // 统一提示入口
  perm,                // 权限判断
  store,               // 只读全局状态
  setTitle(title),
  setBreadcrumb(items),
  setActions(actions), // 页面级操作按钮
  onCleanup(fn),       // 注册卸载清理
}
```

页面级操作按钮由 `ctx.setActions` 提供，渲染在 `PageHeader` 内，属于页面局部，不是全局栏。

## 8. API 客户端

`core/api.js` 是前端唯一请求入口。

```js
api.get(path, { query, timeout })
api.post(path, body, { timeout })
api.put(path, body, { timeout })
api.del(path, { timeout })
```

职责：

1. 统一加 `Content-Type`、`Accept`、`X-Requested-With`；
2. 统一解析响应包 `{ ok, data, meta, error }`；
3. 成功返回 `data`，分页返回 `{ items, page, pageSize, total }`；
4. 失败抛 `ApiError`，包含 `code`、`message`、`status`、`requestId`、`details`；
5. 401 触发全局登录态失效处理；
6. 网络不可达与超时给出不同文案；
7. 写操作自动附加 `/api/meta` 返回的 `X-CSRF-Token`；
8. 不自动弹提示，由调用方决定 `notify.error` 或静默处理。

禁止：

- 视图中裸 `fetch`；
- 视图内拼接 API URL；
- 各页面重复实现错误文案；
- 用 HTTP 200 + `ok:false` 混淆网络错误。

## 9. 提示与反馈

`core/notify.js` 是唯一提示入口。

```js
notify.success('保存成功');
notify.info('任务已提交');
notify.warn('配置已生效，但写回 .env 失败，重启后会回落');
notify.error('保存失败：用户名已存在。请更换用户名后重试');
```

分级：

| 级别 | 自动消失 | 场景 |
|---|---|---|
| success | 是，约 3 秒 | 操作成功 |
| info | 是，约 4 秒 | 一般信息 |
| warn | 否 | 需要用户注意但不阻断 |
| error | 否 | 操作失败，需用户处理 |

规则：

- 服务端返回失败（非 2xx 或 `ok:false`）必须弹 `error`；
- 页面内可同时保留行内灰色提示，但不能只写行内不弹 toast；
- 静默刷新失败可不弹，但行内状态必须更新；
- 切换页面/登出时清理所有 toast；
- 文案必须说清：发生了什么、为什么、下一步怎么办。

## 10. 状态管理

`core/store.js` 是极简显式容器：

```js
store.set('meta', meta);
store.get('meta');
store.subscribe('meta', fn);
```

原则：

- 不做深层响应式代理，不做隐式依赖收集；
- 只存全局低频状态：当前用户、权限、菜单、应用信息、主题；
- 页面数据留在页面内，用完即弃；
- 高频轮询数据不进全局 store；
- 订阅回调必须可取消，页面卸载时清理。

## 11. 主题与样式

### 11.1 设计令牌

`tokens.css` 定义语义变量：

```css
:root {
  --color-bg: #f6f7f9;
  --color-surface: #ffffff;
  --color-text: #1f2329;
  --color-text-muted: #6b7280;
  --color-border: #e5e7eb;
  --color-primary: #2563eb;
  --color-danger: #dc2626;
  --color-success: #16a34a;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --radius-sm: 4px;
  --radius-md: 8px;
  --font-sans: system-ui, -apple-system, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
}

[data-theme="dark"] {
  --color-bg: #111418;
  --color-surface: #1a1f26;
  --color-text: #e6e8eb;
  --color-text-muted: #9aa4b2;
  --color-border: #2a313a;
}
```

规则：

- 组件样式只引用变量，不硬编码色值；
- 主题切换只改 `data-theme`，不重载页面；
- 主题偏好存 `localStorage`；
- `index.html` 在 `<head>` 内尽早设置主题，防闪烁；
- 数据驱动的宽度/比例可内联，其余布局用语义类。

## 12. 通用组件契约

### 12.1 组件形态

纯渲染组件：

```js
export function pageHeaderHTML({ title, description, actions }) { ... }
```

有状态组件：

```js
export function createDataTable(options) {
  return {
    mount(host) {},
    update(rows, meta) {},
    destroy() {},
  };
}
```

规则：

- 组件不引用 `api`、不硬编码业务路径、不读写业务 store；
- 组件通过参数/回调接收数据与事件；
- 组件导出的 DOM 必须可安全插入，文本一律转义；
- 组件不自己弹 toast，错误交给调用方；
- 组件不假设页面结构，挂载点由调用方提供。

### 12.2 表格

```js
createDataTable({
  columns: [
    { key: 'username', title: '用户名' },
    { key: 'status', title: '状态', render: row => badge(row.status) },
  ],
  rowKey: row => row.id,
  selectable: true,
  empty: { title: '暂无用户', description: '点击右上角“新建用户”开始' },
  onPageChange(page) {},
})
```

必须支持：加载态、空态、错误态、分页、列宽控制、行操作区、批量选择（可选）。

### 12.3 表单

表单统一经 `form` 构建：

```js
const f = createForm({
  fields: [
    { name: 'username', label: '用户名', type: 'text', required: true },
    { name: 'role_ids', label: '角色', type: 'select', multiple: true, options: roles },
  ],
  submitText: '保存',
  onSubmit: async values => { ... },
});
```

字段类型：`text`、`password`、`number`、`textarea`、`select`、`switch`、`checkbox`、`radio`、`date`、`datetime`。

校验分两层：

- 前端即时校验：必填、长度、格式；
- 后端权威校验：唯一性、权限、业务规则；
- 后端 422 返回 `details.fields`，表单按字段展示错误。

## 13. 登录与初始化

### 13.1 登录页

- 独立于主布局，不加载侧栏；
- 用户名、密码、登录按钮；
- 失败提示不区分“用户不存在/密码错误”，避免枚举账号；
- 连续失败显示限流提示与等待时间；
- 登录成功拉取 `/api/meta` 后进入默认页。

### 13.2 首次初始化

- 无管理员时，`/api/meta` 返回 `initialized: false`；
- 回环地址可直接设置管理员；
- 非回环地址要求环境变量提供初始口令，页面只做确认与改密；
- 初始化完成后不可重复进入，除非显式重置。

## 14. 测试与验收

### 14.1 浏览器内测试面

`webui/_test/index.html` 仅在 `-tags dev` 下可访问，生产不注册路由。

```text
window.__TEST_RESULT__ = {
  total, passed, failed, cases: [...]
}
document.documentElement.dataset.testStatus = 'done'
```

用例分层：

| 文件 | 覆盖 |
|---|---|
| `cases_core.js` | 路由、菜单、权限、API 错误归一化、store |
| `cases_comp.js` | 表格、表单、筛选、弹窗、分页 |
| `cases_module.js` | 页面挂载/卸载、请求桩、状态变化 |
| `cases_arch.js` | 分层红线、模块清单一致性、禁止裸 fetch/硬编码菜单 |

### 14.2 浏览器实测

测试全绿不等于界面可用。涉及页面的改动必须：

1. 启动 dev 模式；
2. 浏览器实际打开页面；
3. 完成关键操作路径；
4. 截图留证；
5. 检查控制台无错误、网络无 4xx/5xx（预期除外）。

### 14.3 架构机检

必须自动检查：

- 页面模块不得引用 `layout.js` 的全局 DOM；
- 视图不得裸 `fetch`；
- 视图不得硬编码菜单；
- 通用组件不得 import `api`；
- 所有模块入口都在 `router.entries` 注册；
- Go Manifest 的页面与前端入口一一对应。

## 15. 类型表达与组件治理

原生 ESM 不等于放弃工程约束。公共对象、组件 Props、页面 Context、API DTO 和事件参数使用 JSDoc `@typedef` / `@param` / `@returns` 明确契约；避免随意传递无边界 `Object`。在不引入运行期构建链的前提下，使用浏览器测试、静态检查脚本和架构测试校验模块入口、导入边界和接口约定。任何需要 Node 的工具仅可作为可选开发辅助，不得成为运行、默认打包或新项目启动的必要条件。

组件统一遵循：

- 通过参数和回调接收业务数据，不直接访问 `api`、业务 URL 或业务 store。
- DOM 使用 `createElement`、`textContent`、属性白名单等安全 API；不把不可信内容写入 `innerHTML`。
- 组件必须定义 mount/dispose 或等效生命周期，监听器、Observer、计时器和请求可释放。
- 支持键盘操作、可见焦点、标签和禁用/加载/错误状态。
- 重型组件（图表、编辑器）必须使用动态导入，首屏不加载。

## 16. 禁止事项

1. 禁止在页面里直接操作 `#topbar`、`#sidebar`、`#user-menu` 等全局 DOM。
2. 禁止视图裸 `fetch` 或散写 API URL。
3. 禁止自造 toast/modal 提示组件。
4. 禁止硬编码菜单树。
5. 禁止在模块顶层发请求或操作 DOM。
6. 禁止用 `innerHTML` 拼接未转义的用户数据。
7. 禁止引入 npm 运行时依赖作为默认前提。
8. 禁止把业务数据塞进全局 store 当缓存。
9. 禁止用前端隐藏代替后端权限校验。
10. 禁止“测试全绿”代替浏览器实测。
