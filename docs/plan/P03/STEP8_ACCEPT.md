# STEP8：浏览器实测、验收回写、提交推送

状态：已实施

## 目标
- 真实浏览器跑通初始化、登录与六个页面全流程。
- 验证权限边界、按需加载、敏感配置遮蔽与控制台/网络健康。
- 回写验收结论与已知边界，提交并推送。

## 改动文件清单
- `webui/assets/core/router.js`（新增 `ctx.bindActions`）
- `webui/assets/core/boot.js`（注入完整 `api` 对象）
- `webui/assets/pages/system/users/index.js`、`roles/index.js`（改用 `ctx.bindActions`）
- `cmd/easyadmin/main.go`（版本标识 `0.1.0-p03`）
- `docs/FRONTEND.md`（ctx 契约）
- `docs/acceptance/p03/*.png`（12 张验收截图）
- `docs/plan/P03_WEBUI_DESIGN_PLAN.md`（状态与验收结论）

## 实施步骤
- [x] 初始化 → 登录 → 仪表盘
- [x] 用户创建、编辑、禁用/启用、重置密码
- [x] 角色创建、编辑权限、删除临时角色
- [x] 审计、配置只读、个人资料改密
- [x] 改密后登出并用新密码登录
- [x] 普通用户菜单与 API 越权边界
- [x] 按需加载与控制台/网络检查
- [x] 截图、验收回写、提交推送

## 验证
- 接手核实命令：`python tmp/p03-accept/accept_all.py`（需 dev 服务与全新数据目录）
- 本步完整验证：
  - [x] `gofmt -l` 无输出；`go vet ./...`、`go test ./...` 通过
  - [x] 浏览器测试面 51/51
  - [x] 生产构建 `/_test/index.html` 404
  - [x] 全流程脚本输出 `ALL OK`

## 完成标准
- 设计文档 §13 记录验收证据、修复项与已知边界；总纲与 ROADMAP 状态同步。

## 实施回填区
### 产物锚点清单
- `docs/acceptance/p03/01-setup.png` … `12-operator-view.png`
- `webui/assets/core/router.js`：`ctx.bindActions`
- `docs/plan/P03_WEBUI_DESIGN_PLAN.md` §13
### 偏差与现场记录
- 验收发现页头动作按钮无响应：`setActions` 渲染在 `#page-head`，页面 `bindActions(ctx.el)` 只覆盖 `#page-body`。新增 `ctx.bindActions` 同时绑定两处，用户/角色页切换调用。
- 验收发现 `ctx.api` 缺 `put/patch/del`：boot 原只注入 `{get, post}`，改为注入完整 `api` 对象。
- SQLite 在多个写请求真正并发时会返回 500；验收脚本的等待条件误判“禁用/启用”按钮文案，导致三个写请求并发复现一次。修正脚本串行节奏后稳定，已记入设计文档已知边界。
- 归档（迁 `docs/done/`、清理执行期文档）未执行，待用户确认。