# EasyAdmin WebUI

原生 ES Module、零构建前端。生产由 `embed.go` 内嵌进二进制；`-tags dev` 时 `embed_dev.go` 直接读磁盘。

## 目录

- `assets/core/`：请求、路由、布局、菜单、提示、主题、状态、格式化、DOM 安全。
- `assets/components/`：表格、表单、筛选、弹窗、详情、时间范围、页签、标签、空态、日志、DSN、tooltip。
- `assets/pages/`：页面入口，约定导出 `mount(ctx)` / `unmount()`；当前为空，P3 起填充。
- `assets/styles/`：tokens / base / layout / components。
- `_test/`：零依赖浏览器测试面，仅开发构建可访问。

## 浏览器测试

在 `webui/` 上一级启动任意静态服务器（示例）：

```bash
python -m http.server 8080 --directory webui
```

然后打开 `http://127.0.0.1:8080/_test/index.html`。测试完成后：

- `window.__TEST_RESULT__` 给出 total / passed / failed / cases；
- `<html data-test-status="done" data-test-result="pass|fail">` 供自动化脚本判红绿。

测试面不需要后端；主页面 `index.html` 需要 `/api/meta` 等后端接口。

## 约定

- 页面不得裸 `fetch`，统一经 `core/api.js`。
- 提示统一经 `core/notify.js`。
- 页面只操作 `ctx.el`，不得碰顶栏/侧栏等全局 DOM。
- 组件不引用 `api` / `notify`，数据与事件由参数和回调传入。
