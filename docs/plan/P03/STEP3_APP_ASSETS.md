# STEP3：AppAssets 与业务资源接缝

状态：待实施

## 目标
- `httpapi.HandlerOptions.AppAssets fs.FS` 新增字段。
- `/assets/` 固定来自 core；`/assets/app/` 来自 AppAssets；未配置时与 P02 行为一致。
- 业务资源不覆盖 core 资源。

## 改动文件清单
- `infra/httpapi/handler.go`
- `cmd/easyadmin/main.go`
- `infra/httpapi/httpapi_test.go`

## 实施步骤
- [ ] HandlerOptions 增加 AppAssets
- [ ] 挂载 `/assets/app/`，仅 GET/HEAD
- [ ] 未配置时保持 404 行为
- [ ] 补配置/未配置/core 不被覆盖三类测试

## 验证
- 接手核实命令：`go test ./infra/httpapi/...`
- 本步完整验证：
  - [ ] `go test ./infra/httpapi/...`
  - [ ] `go vet ./infra/httpapi/...`

## 完成标准
- `/assets/app/pages.js` 在配置后返回内容，未配置返回 404；`/assets/core` 路径仍来自 core。

## 实施回填区
### 产物锚点清单
- 新增字段：`HandlerOptions.AppAssets`
- 新增测试：`TestAppAssetsMount`
### 偏差与现场记录
- 待回填
