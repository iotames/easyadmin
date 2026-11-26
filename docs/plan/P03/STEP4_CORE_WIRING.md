# STEP4：前端内核页面注册与启动接线

状态：待实施

## 目标
- 新增 `core/pages.js` 注册六个核心页面入口。
- boot 在 `router.start` 前先注册 core 页面，再动态加载 `/assets/app/pages.js`；404 跳过，重复 ID 启动失败。
- 存在 `/assets/app/styles/app.css` 时注入。
- `api.js` 提供存在性探测（HEAD），不新增裸 fetch。

## 改动文件清单
- `webui/assets/core/pages.js`（新增）
- `webui/assets/core/boot.js`
- `webui/assets/core/api.js`
- `webui/_test/cases_core.js`、`webui/_test/cases_arch.js`

## 实施步骤
- [ ] 新增 core/pages.js 注册六个 Page ID
- [ ] boot 启动前加载 core 与业务 entries
- [ ] 业务 pages.js 404 跳过、加载失败/重复 ID 失败
- [ ] app.css 存在时注入
- [ ] api.js 增加 exists(path)
- [ ] 补注册与合并测试

## 验证
- 接手核实命令：`node --check webui/assets/core/pages.js; node --check webui/assets/core/boot.js`
- 本步完整验证：
  - [ ] `node --check` 全部改动 JS
  - [ ] 浏览器 `_test/index.html` 全绿（与 STEP6 一并执行）

## 完成标准
- 页面 ID 注册表与 Manifest 双向一致由 STEP6 测试固定。

## 实施回填区
### 产物锚点清单
- 新增文件：`webui/assets/core/pages.js`
- 新增函数：`api.exists`
- 新增测试：`pages.js 注册表`、`业务 pages 合并/重复`
### 偏差与现场记录
- 待回填
