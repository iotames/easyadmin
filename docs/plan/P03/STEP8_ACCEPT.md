# STEP8：浏览器实测、验收回写、提交推送

状态：待实施

## 目标
- dev 模式启动，浏览器完成：初始化 → 登录 → Dashboard → 用户 → 角色 → 审计 → 配置 → 个人资料改密。
- 检查网络按需加载、控制台无错误、页面卸载。
- 截图留证；验收结论写入 DESIGN_PLAN。
- 更新 ROADMAP/AGENTS 状态；gofmt、测试、vet 全绿后提交并推送 origin。

## 改动文件清单
- `docs/plan/P03_WEBUI_DESIGN_PLAN.md`（验收结论与已知边界）
- `docs/plan/TODO.md`、`docs/plan/P03/STEP*.md`
- `docs/ROADMAP.md`、`AGENTS.md`
- `docs/acceptance/p03/*.png`（如可截图）

## 实施步骤
- [ ] 启动 dev 服务并完成关键路径
- [ ] 捕获截图与网络/控制台证据
- [ ] 回写验收结论与已知边界
- [ ] 更新阶段状态
- [ ] 全量测试、提交、推送

## 验证
- 接手核实命令：`go test ./...`
- 本步完整验证：
  - [ ] `go test ./...`
  - [ ] `go vet ./...`
  - [ ] 浏览器关键路径人工实测记录
  - [ ] `git status --short` 与提交内容一致

## 完成标准
- DESIGN_PLAN 状态 ACCEPTED，提交已推送。

## 实施回填区
### 产物锚点清单
- 待回填
### 偏差与现场记录
- 待回填
