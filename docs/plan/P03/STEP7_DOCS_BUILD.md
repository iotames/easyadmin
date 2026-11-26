# STEP7：文档同步与双模式构建

状态：待实施

## 目标
- 同步 `README.md`、`AGENTS.md`、`docs/FRONTEND.md`、`docs/API.md`、`docs/COMPONENTS.md`、`docs/CONVENTIONS.md`、`docs/ROADMAP.md`、`docs/PROJECT_STRUCTURE.md`。
- dev 构建 `go build -tags dev` 与生产构建均通过。
- 确认生产 embed 不含 `_test/`。

## 改动文件清单
- 上述文档
- 必要时 `webui/README.md`

## 实施步骤
- [ ] 更新阶段状态与页面清单
- [ ] 更新配置/API 契约
- [ ] 更新前端页面与测试面说明
- [ ] 双模式构建验证

## 验证
- 接手核实命令：`go build ./...`
- 本步完整验证：
  - [ ] `go build -tags dev -o bin/easyadmin-dev.exe ./cmd/easyadmin`
  - [ ] `go build -o bin/easyadmin.exe ./cmd/easyadmin`
  - [ ] `go vet ./...`

## 完成标准
- 文档无死链、无计划引用；契约与代码一致。

## 实施回填区
### 产物锚点清单
- 待回填
### 偏差与现场记录
- 待回填
