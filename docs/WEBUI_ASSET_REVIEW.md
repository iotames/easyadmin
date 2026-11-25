# WebUI 资产审查与移植记录

> 日期：2026-11-25。来源：`D:\projects\emailplugin\webui`。状态：**基础资产已落地，页面与 P3 验收未完成**。
> 本文只记录“从参考实现取什么、放到哪、改了什么、证据是什么”，不宣称 P3 已完成。

## 1. 为什么移植

emailplugin/webui 是已在生产使用的原生 ESM 中后台前端，具备 EasyAdmin 需要的工程完成度：零构建、统一请求/提示、可复用表格与筛选、焦点恢复、浏览器测试和架构机检。EasyAdmin 的目标是成为 Go 生态中具有 Yii 3 / Symfony / Laravel 工程完成度的中后台框架，因此这些通用资产应作为 P3 的实现基线，而不是重新发明。

## 2. 已落地文件

```text
webui/
├── index.html
├── embed.go                 # //go:embed index.html + assets
├── embed_dev.go             # -tags dev：os.DirFS
├── assets/
│   ├── core/                # 12 个模块
│   │   ├── api.js  boot.js  data.js  dom.js  format.js  layout.js
│   │   ├── menu.js notify.js perm.js router.js store.js theme.js
│   ├── components/          # 14 个组件
│   │   ├── page-header.js data-table.js form.js filter-bar.js modal.js
│   │   ├── detail-modal.js date-range.js tabs.js badge.js empty.js
│   │   ├── action-panel.js log-view.js dsn-form.js tooltip.js
│   ├── pages/               # 已建目录，暂无页面
│   └── styles/              # tokens/base/layout/components
└── _test/                   # runner + core/component/architecture 用例
```

## 3. 能力映射

| 能力 | 来源 | 落地实现 | 说明 |
|---|---|---|---|
| 统一请求 | `assets/js/api.js` | `core/api.js` | 增加 CSRF、`{ok,data,error,meta}`、分页归一化；401 交回 boot |
| 提示 | `assets/js/ui.js` | `core/notify.js` | error/warning 常驻；页面切换清理 |
| 主题 | `assets/js/theme.js` | `core/theme.js` | light/dark/green；键名 `easyadmin.theme` |
| 数据缓存 | `assets/js/data.js` | `core/data.js` | SWR + 单飞 + TTL + 失败兜底 |
| 菜单/路由 | `assets/js/nav.js` + `main.js` | `core/menu.js` + `router.js` + `boot.js` | 菜单改服务端；hash 改路径；页面 mount/unmount |
| 布局 | `index.html` 静态骨架 | `core/layout.js` | 顶栏/侧栏/页头/连接状态唯一 owner |
| 格式化 | `assets/js/util.js` | `core/dom.js` + `format.js` | 保留焦点/光标恢复 |
| 表格 | `components/dataTable.js` | `components/data-table.js` | 分页、多选、行详情、空态对象 |
| 筛选 | `components/filterBar.js` | `components/filter-bar.js` | 中文输入法安全、焦点恢复、searchSelect |
| 表单 | `components/form.js` | `components/form.js` | 增加 422 字段错误映射 `applyFieldErrors` |
| 详情/弹窗 | `detailModal.js` + `ui.js` | `detail-modal.js` + `modal.js` | 复制反馈本地化，不引用 notify |
| 时间范围 | `components/dateRange.js` | `components/date-range.js` | 严格拒绝 2026-02-31 |
| 日志 | `components/logView.js` | `components/log-view.js` | 级别过滤、转义 |
| DSN | `components/dsnForm.js` | `components/dsn-form.js` | 命名数据源表单 |
| 测试 | `_test/*` | `_test/*` | runner + core/component/architecture |

## 4. 与参考实现的关键偏差

1. **ESM 化**：取消 `window.Rock` 与 IIFE，全部命名导出；不保留全局可变 Service Locator。
2. **API 契约**：`msg` → `error.code` / `error.message`；`X-CSRF-Token` 由 `/api/meta` 注入内存；`request_id` 放在 `meta`。
3. **路由**：路径路由 + `popstate`；Page ID → loader 通过 `registerEntries` 显式登记。
4. **菜单**：只能来自 `/api/meta`；前端只保留图标映射，不保留页面清单。
5. **组件解耦**：组件不 import `api`/`notify`；错误与提示由页面处理。
6. **生命周期**：页面 `mount(ctx)/unmount()`；`ctx.onCleanup` 统一释放监听器、计时器和请求。
7. **样式**：单文件 style.css 拆为 tokens/base/layout/components；业务页面样式未带入。

## 5. 未移植

- 业务页面与业务状态：邮箱、邮件、同步、归档、ERP、凭据业务字段；
- 调度页专用样式（原 style.css 第 22 段）；
- 独立 `login.html`：登录纳入 SPA，按 `initialized` / `user` 分流；
- drawer：参考实现没有，等真实需求再设计。

## 6. 验证证据

| 项目 | 命令 / 操作 | 结果 |
|---|---|---|
| ESM 语法 | `node --experimental-default-type=module --check`（30 个 JS） | 30/30 通过 |
| 生产内嵌 | `go build ./webui` | 通过 |
| 开发磁盘 | `go build -tags dev ./webui` | 通过 |
| 静态检查 | `go vet ./webui` | 通过 |
| 浏览器用例 | 静态服务 + 无头 Chrome 打开 `/_test/index.html` | 47/47 通过 |
| 布局实测 | mock `/api/meta` 后打开 `/`，1366×900 截图 | 顶栏/菜单/主题/用户区/空页面提示正常 |

## 7. 后续事项

1. P3 页面：Dashboard、用户、角色、配置、个人资料；补齐 `pages.js` 与 Manifest 一致性测试。
2. 登录联调：`/api/meta` 未登录分支、429 限流文案、CSRF 失效提示。
3. `infra/httpapi` 静态服务：SPA 回退、`/_test` 仅 dev 可访问、缓存头。
4. 补充 `cases_module.js`：页面 mount/unmount、请求桩、卸载清理。
5. 业务项目扩展：`/assets/app/pages.js` 合并与 Page ID 冲突启动失败。

