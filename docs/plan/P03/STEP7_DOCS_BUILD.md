# STEP7：文档同步与双模式构建

状态：已实施

## 目标
- 同步 `README.md`、`AGENTS.md`、`docs/FRONTEND.md`、`docs/API.md`、`docs/COMPONENTS.md`、`docs/CONVENTIONS.md`、`docs/ROADMAP.md`、`docs/PROJECT_STRUCTURE.md`。
- dev 构建 `go build -tags dev` 与生产构建均通过。
- 确认生产 embed 不含 `_test/`。

## 改动文件清单
- `README.md`、`AGENTS.md`
- `docs/FRONTEND.md`、`docs/API.md`、`docs/COMPONENTS.md`、`docs/ROADMAP.md`、`docs/PROJECT_STRUCTURE.md`
- `docs/plan/TODO.md`

## 实施步骤
- [x] 更新阶段状态与页面清单
- [x] 更新配置/API 契约
- [x] 更新前端页面与测试面说明
- [x] 双模式构建验证

## 验证
- 接手核实命令：`go build ./...`
- 本步完整验证：
  - [x] `go build -tags dev -o bin/easyadmin-dev.exe ./cmd/easyadmin`
  - [x] `go build -o bin/easyadmin.exe ./cmd/easyadmin`
  - [x] `go vet ./...`
  - [x] 生产版 `/_test/index.html` 返回 404，`/` 与 `/assets/core/pages.js` 返回 200

## 完成标准
- 文档无死链、无计划引用；契约与代码一致。

## 实施回填区
### 产物锚点清单
- 状态：`README.md` 首段、`AGENTS.md` 项目状态、`docs/ROADMAP.md` 首段与 P03 行、`docs/PROJECT_STRUCTURE.md` P03 段
- 页面：`docs/FRONTEND.md` 页面状态表（六个 P03 已交付）、`docs/COMPONENTS.md` 页面清单
- 契约：`docs/API.md` 第 9 节配置项对象改为实际字段（group/order/key/title/type/default/value/source/secret/hot_reload/writable/example）
### 偏差与现场记录
- API 文档原配置示例使用 `APP_LISTEN_ADDR`/`persisted`/`restart_required` 等未实现字段，已按 P03 实际响应校正；PUT 仍标注为未承诺。
- `README.md` 与 `AGENTS.md` 保留 P00–P02 归档表述，P03 标注“已验收待归档”，归档动作待用户确认。