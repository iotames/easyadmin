# STEP6：测试补齐

状态：已实施

## 目标
- Go：菜单顺序、Manifest 与 entries 双向一致（缺失/多余）、AppAssets、配置来源与敏感遮蔽、权限集回归。
- 浏览器：内核页面注册、业务 pages.js 合并/重复 ID、页面 mount/unmount。
- 架构：页面无裸 fetch、无全局 DOM、组件不依赖 api/notify。

## 改动文件清单
- `internal/archtest/pages_test.go`（新增）
- `infra/httpapi/httpapi_test.go`
- `infra/conf/conf_test.go`
- `apps/system/system_test.go`
- `webui/_test/cases_core.js`、`webui/_test/cases_arch.js`

## 实施步骤
- [x] Manifest 与 core/pages.js 双向一致测试
- [x] 菜单顺序测试
- [x] AppAssets 测试
- [x] 配置来源/敏感遮蔽测试
- [x] 浏览器页面注册与合并测试
- [x] 页面生命周期测试

## 验证
- 接手核实命令：`go test ./...`
- 本步完整验证：
  - [x] `gofmt -l` 无输出
  - [x] `go test ./...`
  - [x] `go vet ./...`
  - [x] 浏览器测试面全绿（51/51，console 仅预期探测 404）

## 完成标准
- 新增测试真实覆盖验收标准，不以“本地可运行”替代。

## 实施回填区
### 产物锚点清单
- `internal/archtest/pages_test.go`：`TestManifestAndCorePagesMatch`、`TestPageSetDiffDetectsMissingAndExtra`
- `infra/httpapi/httpapi_test.go`：`TestMenuOrderByOrder`、`TestAppAssetsMount`
- `infra/conf/conf_test.go`：`TestConfigSourceByLoadOrder`、`TestConfigSecretMaskedAndMetaValidated`
- `apps/system/system_test.go`：`TestManifestPermissionsSyncedByModule`、`TestConfigEndpointMasksSecrets`
- `webui/_test/cases_core.js`：core/pages.js 注册表、registerEntries 合并/重复 ID
- `webui/_test/cases_arch.js`：六个页面 mount/unmount、core/pages.js 与文件一一对应
### 偏差与现场记录
- Manifest 与 entries 一致性用 Go 侧解析 `core/pages.js` 文本比对，避免引入 JS 运行时依赖。
- archtest 双向比对同时覆盖“缺失”与“多余”两个方向，差集函数单独测试。