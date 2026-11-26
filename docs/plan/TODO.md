# 执行总纲

> 执行依据：docs/plan/README.md（计划目录工作宪法）。

项目状态：进行中

## §0 断点续传（接手者从这里开始）
1. 读项目状态行与 §1 总表；
2. 定位第一个非「已实施」的步骤；
3. 「实施中」步骤先跑其验证节命令，仍有未勾选项则从首个未勾选项继续；
4. 若锚点检索已有产物命中，先核实产物再续做，禁止从零重写；
5. 每次状态迁移后跑一致性自检；
6. 严禁凭会话记忆做实施判断。

## §1 状态总表

| # | STEP | 内容 | 依赖 | 状态 |
|---|------|------|------|------|
| 1 | P03/STEP1_BACKEND_META.md | 页面 ID、菜单排序、权限单源 | — | 已实施 |
| 2 | P03/STEP2_CONFIG_READONLY.md | 配置元数据与只读接口 | 1 | 已实施 |
| 3 | P03/STEP3_APP_ASSETS.md | AppAssets 与业务资源接缝 | 1 | 已实施 |
| 4 | P03/STEP4_CORE_WIRING.md | 前端内核页面注册与启动接线 | — | 已实施 |
| 5 | P03/STEP5_PAGES.md | 六个核心页面 | 2,3,4 | 已实施 |
| 6 | P03/STEP6_TESTS.md | Go 与浏览器测试补齐 | 5 | 已实施 |
| 7 | P03/STEP7_DOCS_BUILD.md | 文档同步与双模式构建 | 6 | 已实施 |
| 8 | P03/STEP8_ACCEPT.md | 浏览器实测、验收回写、提交推送 | 7 | 待实施 |

## §2 执行期红线
- 页面不得裸 fetch、不得操作全局 DOM、不得未转义 innerHTML；页面必须可 unmount。
- 所有 API 经 `webui/assets/core/api.js`，提示经 `notify.js`。
- SQL 参数化；权限只增不删；敏感配置不回显。
- git 提交：本次任务适用破例授权；提交前必须 `gofmt`、`go test ./...`、`go vet ./...` 通过；提交即推送 origin。

## §3 进度日志

| 日期 | 执行者 | STEP | 结果 |
|------|--------|------|------|
| 2026-11-26 | 智能体 | 准备 | P03 DESIGN_PLAN 标记 APPROVED，建立总纲与步骤 |
| 2026-11-26 | 智能体 | STEP1 | 已实施：菜单 Order 排序、dashboard、权限单源；go test/go vet 通过 |
| 2026-11-26 | 智能体 | STEP2 | 已实施：配置元数据、来源判定、GET /api/system/config、敏感遮蔽；全量测试通过 |
| 2026-11-26 | 智能体 | STEP3 | 已实施：AppAssets 挂载 /assets/app/，未配置保持 404；全量测试通过 |
| 2026-11-26 | 智能体 | STEP4-6 | 已实施：内核页面注册、业务 entries 合并、六个页面、Go 与浏览器测试；浏览器 51/51 全绿 |
| 2026-11-26 | 智能体 | STEP7 | 已实施：README/AGENTS/ROADMAP/FRONTEND/API/COMPONENTS/PROJECT_STRUCTURE 同步；dev 与生产构建通过，生产 _test 为 404 |
