# P03 设计计划：WebUI 管理台

> 状态：`PROPOSED`（待用户评审；未批准前不得进入本期实施）。

## 1. 背景与目标

交付统一布局、菜单、页面懒加载和可复用后台组件。

## 2. 本期范围

- 原生 ESM 前端内核、统一 Layout/Menu/Router/API/Notify/Theme/Store。
- CSS tokens、响应式、面包屑、页面插槽、登录和初始化页。
- 表格、表单、筛选、分页、弹窗、抽屉、空态和反馈组件。
- Dashboard、用户、角色、配置、个人资料页面。
- 浏览器内测试面、架构机检和实际浏览器验收。

## 3. 明确不做

- Node/npm/Vite/Webpack 作为默认生产运行依赖。
- Vue/React 运行时、复杂图表和代码编辑器首屏依赖。
- 任务/调度/文件等 P4 页面。

## 4. 架构与契约决策

- 保持原生 ESM + 动态 import。
- Manifest Page ID 与 router.entries 的一致性契约。
- Layout 全局区域唯一所有者。
- 组件通过参数和回调，不知道业务 API。
- API Client 和 Notify 是唯一请求/提示入口。

设计必须服从 `docs/ARCHITECTURE.md`、`docs/DECISIONS.md` 和 `docs/CONVENTIONS.md`。若本期需要改变其中任一正式边界，必须先提交 ADR/决策修订，不可在实施计划中暗改。

## 5. 交付与影响面

- `webui/assets/core/`、`components/`、`pages/`、`styles/`、`_test/`。
- `webui/embed*.go` 和 `apps/*/Manifest` 页面元数据。
- `docs/FRONTEND.md`、`API.md`、组件清单和验收截图。

## 6. 验收标准

- 初始化→登录→Dashboard→用户/角色/配置管理完整可用。
- 页面切换网络请求证明按需加载。
- 页面卸载无遗留事件、计时器和请求。
- 菜单权限隐藏与后端授权均正确。
- 浏览器测试全绿，关键流程人工实测。

每项验收都必须对应可执行测试、命令、浏览器操作或源码核验记录。截图不能替代功能测试。

## 7. 风险与缓解

- 页面不得碰顶栏/侧栏等全局 DOM。
- 不允许裸 fetch、alert、自造 toast、未转义 innerHTML。
- 页面入口必须具备卸载能力。
- 前端零构建策略若要改变，必须另做架构评审。

## 8. 前置条件

- 前置阶段已验收，记录可查。
- 本期 DESIGN_PLAN 已由用户明确确认。
- IMPL_PLAN 已根据当前仓库代码校准文件路径和接口。
- 外部依赖和易库版本已固定；未知语义已列出验证步骤。

## 9. 设计确认记录

| 日期 | 评审人 | 结论 | 需调整事项 |
|---|---|---|---|
| 待评审 | 待确认 | `PROPOSED` | 无 |
