# STEP5：六个核心页面

状态：待实施

## 目标
- `dashboard`、`profile`、`system.users`、`system.roles`、`system.audit`、`system.config`。
- 页面导出 `mount(ctx)/unmount()`；使用 ctx.api、ctx.notify、ctx.perm、通用组件。
- 列表/表单/弹窗具备加载、空态、错误反馈；卸载清理监听器与未完成请求。
- 配置页只读，敏感项显示“已配置”。

## 改动文件清单
- `webui/assets/pages/dashboard/index.js`
- `webui/assets/pages/profile/index.js`
- `webui/assets/pages/system/users/index.js`
- `webui/assets/pages/system/roles/index.js`
- `webui/assets/pages/system/audit/index.js`
- `webui/assets/pages/system/config/index.js`

## 实施步骤
- [ ] Dashboard
- [ ] Profile（含改密）
- [ ] 用户管理（列表/创建/编辑/状态/重置密码）
- [ ] 角色管理（列表/创建/编辑权限/删除）
- [ ] 审计日志（查看/分页）
- [ ] 配置中心（只读、分组、敏感遮蔽）
- [ ] 每页 unmount 清理

## 验证
- 接手核实命令：`rg -n "export async function mount|export function mount|unmount" webui/assets/pages`
- 本步完整验证：
  - [ ] `rg` 检查无裸 fetch、无 `#topbar/#sidebar` 操作
  - [ ] `go test ./internal/archtest/...`

## 完成标准
- 六个入口文件存在并被 pages.js 注册。

## 实施回填区
### 产物锚点清单
- 新增页面：六个 `index.js`
### 偏差与现场记录
- 待回填
