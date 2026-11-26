# STEP5：六个核心页面

状态：已实施

## 目标
- `dashboard`、`profile`、`system.users`、`system.roles`、`system.audit`、`system.config`。
- 页面导出 `mount(ctx)/unmount()`；使用 ctx.api、ctx.notify、ctx.perm、通用组件。
- 列表/表单/弹窗具备加载、空态、错误反馈；卸载清理监听器与未完成请求。
- 配置页只读，敏感项显示“已配置”。

## 改动文件清单
- `webui/assets/pages/_shared.js`（页面共用小工具）
- `webui/assets/pages/dashboard/index.js`
- `webui/assets/pages/profile/index.js`
- `webui/assets/pages/system/users/index.js`
- `webui/assets/pages/system/roles/index.js`
- `webui/assets/pages/system/audit/index.js`
- `webui/assets/pages/system/config/index.js`

## 实施步骤
- [x] Dashboard
- [x] Profile（含改密）
- [x] 用户管理（列表/创建/编辑/状态/重置密码）
- [x] 角色管理（列表/创建/编辑权限/删除）
- [x] 审计日志（查看/分页）
- [x] 配置中心（只读、分组、敏感遮蔽）
- [x] 每页 unmount 清理

## 验证
- 接手核实命令：`rg -n "export async function mount|export function mount|unmount" webui/assets/pages`
- 本步完整验证：
  - [x] `rg` 检查无裸 fetch、无 `#topbar/#sidebar` 操作
  - [x] `go test ./internal/archtest/...`
  - [x] 浏览器 `_test/index.html` 全绿（51/51）

## 完成标准
- 六个入口文件存在并被 pages.js 注册。

## 实施回填区
### 产物锚点清单
- 新增页面：`webui/assets/pages/dashboard/index.js`、`webui/assets/pages/profile/index.js`
- 新增页面：`webui/assets/pages/system/users/index.js`、`roles/index.js`、`audit/index.js`、`config/index.js`
- 新增共用模块：`webui/assets/pages/_shared.js`
### 偏差与现场记录
- 页面内不重复实现权限展示/状态徽标/角色选项等小逻辑，统一收敛到 `_shared.js`；仅保留确有两个以上页面复用的部分。
- 配置页只读；PUT 返回 404 由 Go 测试固定，前端无写入入口。