# STEP3：组合根装配 + 中间件追加 + features.auth + P02 验收

状态：待实施

## 目标
cmd 装配数据库、结构同步、system 模块与认证中间件；在 P01 链路后追加 Session→CSRF→Permission；完成时置 features.auth=true，移除占位页；完成 P02 全量验收与归档。

## 改动文件清单
- cmd/easyadmin/main.go（装配）
- infra/httpapi（中间件追加点）
- docs/API.md、docs/DATA_DICT.md、docs/LOGGING_DESIGN.md、docs/COMPONENTS.md

## 实施步骤
- [ ] cmd 装配 DB + 结构同步 + system 模块
- [ ] 中间件追加 Session→CSRF→Permission，公开路由豁免
- [ ] features.auth=true，前端走真实 setup/login
- [ ] 全量 gofmt/test/vet/-race + 浏览器实看初始化→登录

## 验证
- 接手核实命令：go build ./... && go test ./...
- 本步完整验证：
  - [ ] go test ./... -race
  - [ ] 浏览器完成初始化→登录→meta

## 完成标准
- 全量测试通过；P02 DESIGN 回写 ACCEPTED 并归档；features.auth=true。
