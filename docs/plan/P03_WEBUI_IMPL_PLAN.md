# P03 实施计划：WebUI 管理台

> 本文件是 DESIGN_PLAN 的实施指导，不单独送审；按设计定稿后的实际代码基线校准，实施仍需独立开工授权。

## 1. 实施前检查

- [ ] 本期 DESIGN_PLAN 已由用户明确确认并标记 APPROVED。
- [ ] 前置阶段 P02 已 ACCEPTED，遗留问题已核验。
- [ ] 用户已另行明确授权本期开工。
- [ ] 当前仓库代码、目录、依赖版本和测试入口已重新检查。
- [ ] 本期 API/配置/数据/权限/菜单/页面契约影响已列明。
- [ ] 任何超出设计边界的发现已先提交决策，不擅自扩大范围。

## 2. 实施步骤

1. 校正页面契约：Manifest 页面集改为 `dashboard`、`profile`、`system.users`、`system.roles`、`system.audit`、`system.config`；`console.home` 更名 `dashboard` 并调整路径。
2. 让 `Order` 真正生效：`httpapi.Catalog.buildMenus` 按顶层 `Order`、组内 `Order` 稳定排序，并补固定期望顺序的测试。
3. 权限码单一真源：Manifest 保留唯一声明，删除重复的 `permissionDefs`；同步下沉到 system 模块统一执行，组合根删除逐个模块遍历的重复逻辑，并补权限集不变的回归测试。
4. 配置项元数据：注册时声明敏感级别与热更语义，`infra/conf` 判定来源类别；补对应单测。
5. 配置中心只读接口：`GET /api/system/config`（`system.config.read`）按 `API.md` §9 输出，敏感项掩码，不实现 PUT。
6. 建立 index.html、ESM boot 和全局错误处理；boot 顺序为 core 页面注册 → 业务 `pages.js` → 路由启动。
7. 实现 DOM 安全工具、API Client、Notify、Store、Theme、Format（已落地部分只接线不重写）。
8. 实现 Router、Page Context、动态 import 和 Manifest-entry 双向一致性检查。
9. 实现 Layout 插槽、导航、菜单、面包屑和响应式，菜单按服务端返回顺序渲染。
10. 实现基础通用组件并定义 JSDoc 契约、dispose 语义和可访问性（已落地组件只补契约与测试）。
11. 实现登录/初始化收口、Dashboard、用户、角色、审计、配置（只读）、个人资料页面。
12. 预留 `AppAssets`、`/assets/app/pages.js` 和 `/assets/app/styles/app.css` 接缝，并用测试 fixture 覆盖缺失、合并、重复 ID。
13. 建立 core/components/pages/architecture 浏览器测试用例。
14. 验证生产 embed 与开发磁盘模式，进行网络面板和浏览器操作验收。

## 3. 主要修改文件/目录

- `webui/assets/core/`、`components/`、`pages/`、`styles/`、`_test/`；新增 core 页面注册表。
- `infra/httpapi`：菜单排序、权限清单、`AppAssets` 挂载。
- `infra/conf`：配置项元数据（来源、敏感级别、热更语义）。
- `apps/system`、`apps/console`：页面与权限声明、权限表同步、配置只读接口。
- `docs/FRONTEND.md`、`API.md`、组件清单和验收截图。

实际文件以实施前仓库核对为准；如与此计划不同，先更新计划并记录原因。

## 4. 测试与验收执行

浏览器内单测、页面集成测试、ESM loader 懒加载验证、键盘/焦点检查、权限 UI 与后端联测。
回退：页面和组件按模块目录隔离；可按 Manifest 关闭新增页面，核心 Layout/API 保留兼容入口。

- [ ] 菜单顺序固定期望通过（控制台 → 系统分组 → 个人资料）。
- [ ] Manifest 与 entries 双向一致性：缺失、多余、重复 ID 三类失败用例通过。
- [ ] `console.read` 可从 `/api/system/permissions` 授予并在页面生效。
- [ ] `GET /api/system/config` 返回来源/敏感/热更字段，敏感值不回显，无 PUT 路由。
- [ ] `AppAssets` 缺失/配置与 `/assets/app/pages.js` 合并、重复 ID fixture 测试通过。
- [ ] `gofmt`、`go test ./...`、`go vet ./...` 通过。
- [ ] 适用的 race、集成、浏览器或跨平台测试通过。
- [ ] 失败用例和边界情况有证据，不以“本地可运行”代替验收。

## 5. 文档同步清单

- [ ] README / AGENTS（使用方式或长期工程规则变化时）。
- [ ] ARCHITECTURE / DECISIONS / ROADMAP（架构、决策、阶段变更时）。
- [ ] API / 配置 / 数据字典 / 日志专题文档（契约变化时）。
- [ ] COMPONENTS / FRONTEND / PROJECT_STRUCTURE（模块、组件或目录变化时）。
- [ ] 示例、模板和浏览器验收记录（如适用）。

## 6. 风险控制与回退

- 页面不得碰顶栏/侧栏等全局 DOM。
- 不允许裸 fetch、alert、自造 toast、未转义 innerHTML。
- 页面入口必须具备卸载能力。
- 前端零构建策略若要改变，必须另做架构评审。

每一项涉及持久化数据的步骤必须明确备份、恢复和失败状态；不得把不可逆结构变更描述为可自动回滚。

## 7. 完成定义

- [ ] 设计计划中的验收标准均有测试或操作证据。
- [ ] 测试全部通过；未通过项经明确评审接受并记录。
- [ ] 文档、API、数据字典、配置、菜单、权限和测试同步。
- [ ] 已知限制、兼容性影响和后续事项明确。
- [ ] DESIGN_PLAN 记录实际偏差、验收日期和结论，阶段状态更新为 `ACCEPTED`。
