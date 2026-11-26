# P01 实施计划：运行时内核

> 本文件为实施指导，不设置独立审批状态；设计确认后按实际代码校准，实施仍需独立开工授权。

## 1. 实施前检查

- [ ] P01 DESIGN_PLAN 已明确标记 APPROVED。
- [ ] P00 已验收，遗留项已逐条评估对 P01 的影响。
- [ ] 用户已另行明确授权 P01 开工。
- [ ] 已重新检查 infra/conf、infra/db、infra/credential、infra/tasks、infra/scheduler 的当前 API 与测试。
- [ ] 已根据当前代码校准本实施指导；未把尚不存在的包写成已存在能力。

## 2. 实施步骤

1. 检查锁定版本 easy 系列源码与测试，形成 API/并发/关闭核验报告。
2. 定义最小 kernel App、Module、Manifest、ConfigRegistrar、Registrar 契约。
3. 实现模块收集、重复检查、依赖图、稳定拓扑排序和循环检测。
4. 实现 Declare → config load → Register → Start 的阶段流程。
5. 接入 easyconf 适配、配置快照、来源信息、校验和受控 Watch。
6. 接入统一 Logger/apperror/httpapi，固定 RequestID/Recover/AccessLog 顺序。
7. 实现元信息、健康探针、优雅启动/关闭和失败清理。
8. 增加并发、异常退出和底层适配集成测试。

## 3. 主要修改文件/目录

- `kernel/app`、`kernel/module`、注册表和生命周期测试。
- `infra/conf`、`logx`、`apperror`、`httpapi`。
- `infra/internal` 中 easyconf/easyserver/log 适配。
- `cmd/easyadmin` 组合根。
- `docs/ARCHITECTURE.md`、`API.md`、`CONFIGURATION_DESIGN.md`。

实际文件以实施前仓库核对为准；如与此计划不同，先更新计划并记录原因。

## 4. 测试与验收执行

模块注册表表驱动测试、生命周期失败清理测试、中间件顺序测试、request_id/错误响应测试、easy 系列集成测试、go test -race。
回退：每个 adapter 通过独立构造配置开关；启动链路失败时停止监听，已开启资源逆序关闭。

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

- 不做全局 Service Locator；跨模块调用通过构造函数窄接口。
- 不在 Register 中运行 DDL。
- 底层易库缺能力必须记录并提交决策，不得复制整套基础设施。

每一项涉及持久化数据的步骤必须明确备份、恢复和失败状态；不得把不可逆结构变更描述为可自动回滚。

## 7. 完成定义

- [ ] 设计计划中的验收标准均有测试或操作证据。
- [ ] 测试全部通过；未通过项经明确评审接受并记录。
- [ ] 文档、API、数据字典、配置、菜单、权限和测试同步。
- [ ] 已知限制、兼容性影响和后续事项明确。
- [ ] DESIGN_PLAN 记录实际偏差、验收日期和结论，阶段状态更新为 `ACCEPTED`。

## 8. 实施记录

| 日期 | 变更 | 验证证据 | 偏差/遗留 |
|---|---|---|---|
| 待实施 | - | - | - |
