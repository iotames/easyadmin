# STEP1：页面 ID、菜单排序与权限单源

状态：已实施

## 目标
- `dashboard` 取代 `console.home`，路径 `/`，顶层 Order 0。
- `buildMenus` 按 Order 统一排序：顶层叶子与分组统一升序，分组取组内最小 Order，组内升序，同序按 Page ID。
- 权限码只由 Manifest 声明；`system` 模块统一同步全部模块 Manifest 权限并标记 orphaned，组合根不再遍历。
- `system.config` 页面与 `system.config.read` 权限进入 Manifest。

## 改动文件清单
- `apps/console/module.go`
- `apps/system/module.go`
- `apps/system/service.go`
- `kernel/app/app.go`
- `infra/httpapi/catalog.go`
- `cmd/easyadmin/main.go`
- `infra/httpapi/httpapi_test.go`、`apps/system/system_test.go`、`internal/archtest/arch_test.go`

## 实施步骤
- [x] 校正 console Manifest：`dashboard` / `/` / Order 0
- [x] system Manifest 增加 `system.config` 页面与 `system.config.read` 权限
- [x] `buildMenus` 按 Order 排序并补测试 ✓（`go test ./infra/httpapi/...`，通过）
- [x] `kernel.App` 暴露 `Manifests()`
- [x] system.Bootstrap 统一同步权限并标记 orphaned ✓（`go test ./apps/system/...`，通过）
- [x] 删除组合根逐模块权限同步逻辑
- [x] 补权限集不变回归测试 ✓（`TestManifestPermissionsSyncedByModule`，通过）

## 验证
- 接手核实命令：`go test ./infra/httpapi/... ./apps/console/... ./apps/system/... ./kernel/...`
- 本步完整验证：
  - [x] `go test ./infra/httpapi/... ./apps/console/... ./apps/system/... ./kernel/...`
  - [x] `go vet ./infra/httpapi/... ./apps/... ./kernel/...`

## 完成标准
- `/api/meta` 菜单顺序固定为 dashboard → system（users/roles/audit/config）→ profile，并有测试断言。
- 删除重复权限清单后权限集不变，有回归测试。

## 实施回填区
### 产物锚点清单
- 修改页面：`console.Module.Manifest` 中 `dashboard`
- 新增页面/权限：`system.config`、`system.config.read`
- 新增方法：`kernel/app.App.Manifests`
- 新增测试：`TestMenuOrderByOrder`、`TestManifestPermissionsSyncedByModule`
### 偏差与现场记录
- `permissionDefs()` 保留为 `Manifest.Permissions` 的唯一内部声明源，不再作为第二份清单；Bootstrap 与 Manifest 共用同一数组。
- 权限同步由 `Module.Start` 调用 `syncManifestPermissions` 完成；`cmd` 只注入 `ManifestSource`，不再遍历模块。
