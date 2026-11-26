# STEP6：测试补齐

状态：待实施

## 目标
- Go：菜单顺序、Manifest↔entries 双向一致（缺失/多余）、AppAssets、配置来源与敏感遮蔽、权限集回归。
- 浏览器：内核页面注册、业务 pages.js 合并/重复 ID、页面 mount/unmount。
- 架构：页面无裸 fetch、无全局 DOM、组件不依赖 api/notify。

## 改动文件清单
- `internal/archtest/arch_test.go`（或新增 `pages_test.go`）
- `infra/httpapi/httpapi_test.go`
- `infra/conf/conf_test.go`
- `apps/system/system_test.go`
- `webui/_test/cases_core.js`、`webui/_test/cases_arch.js`

## 实施步骤
- [ ] Manifest 与 core/pages.js 双向一致测试
- [ ] 菜单顺序测试
- [ ] AppAssets 测试
- [ ] 配置来源/敏感遮蔽测试
- [ ] 浏览器页面注册与合并测试
- [ ] 页面生命周期测试

## 验证
- 接手核实命令：`go test ./...`
- 本步完整验证：
  - [ ] `gofmt -l` 无输出
  - [ ] `go test ./...`
  - [ ] `go vet ./...`
  - [ ] 浏览器测试面全绿

## 完成标准
- 新增测试真实覆盖验收标准，不以“本地可运行”替代。

## 实施回填区
### 产物锚点清单
- 新增测试：待回填
### 偏差与现场记录
- 待回填
