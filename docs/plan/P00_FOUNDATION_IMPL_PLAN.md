# P00 实施计划：地基与工程规范

> 本文件是 P00 DESIGN_PLAN 的实施指导；设计获批后按当时仓库代码校准，不单独送审，也不构成开工授权。

## 1. 实施前检查

- [ ] P00 DESIGN_PLAN 已由用户明确确认并标记 APPROVED。
- [ ] 用户另行明确授权 P00 开工；设计确认不代替开工授权。
- [ ] 工作分支从最新 main 创建，工作区干净，当前 main SHA 已记录。
- [ ] 已重新检查 go.mod、go.sum、.gitmodules、lib 子模块提交与当前目录。
- [ ] 未确认的架构边界已登记，不在实施中自行扩张。

## 2. 阶段实施切片

以下是阶段级切片，不替代开工后按计划宪法建立的 TODO 与 STEP。每个 STEP 再按实际源码细化为文件级动作与测试。

### 切片 1：源码和依赖事实核验

**范围**：go.mod、go.sum、.gitmodules、lib/*、infra/*、现有测试文件。  
**动作**：记录 Go 版本、直接依赖、replace、本地子模块提交；核查包目录、导出 API、测试覆盖和持久化状态。  
**验证**：git submodule status、go list -m all、go mod verify、go test ./infra/...、go test -race ./infra/...、go vet ./infra/...。  
**完成标准**：形成可复现依赖基线；所有历史“已通过”记录区分历史证据与本次结果。

### 切片 2：架构与工程文档收口

**范围**：README.md、AGENTS.md、docs/ARCHITECTURE.md、DECISIONS.md、ROADMAP.md、COMPONENTS.md、CONVENTIONS.md、PROJECT_STRUCTURE.md、API/配置/数据/日志文档。  
**动作**：修正“当前实现/目标设计/待建设”混用；将现有 infra 事实与目标树分开；同步依赖方向、配置校验和阶段边界。  
**验证**：脚本检查相对 Markdown 链接；逐项核验目标路径存在性和计划配对。  
**完成标准**：没有把文件仓储描述成 SQL 持久化、把内存任务描述成已持久化或把目标目录描述成当前目录。

### 切片 3：公共组件和 HTTP 输入规范

**范围**：docs/COMPONENTS.md、CONVENTIONS.md、API.md、相关 Go 现有代码。  
**动作**：核验每个组件职责、构造方式、错误、Context、并发、生命周期、观测、测试和适配边界；固定 Bind/字段校验/领域约束分层。  
**验证**：现有代码审查 + 示例/单测；400/422/409/403 响应契约检查。  
**完成标准**：不引入万能接口；未知 validator 依赖不在 P0 固定。

### 切片 4：组合根与最小运行骨架

**范围**：cmd/easyadmin、kernel 最小 App、现有 infra 适配边界。  
**动作**：按确认设计建立唯一组合根、最小 App 生命周期和一个健康/最小 HTTP smoke；只装配已存在且需要的能力。  
**验证**：go test ./...、go test -race ./...、go vet ./...、go build ./cmd/easyadmin。  
**完成标准**：可启动、可关闭、错误能定位；不出现身份/权限业务逻辑。

### 切片 5：WebUI 资源交付骨架

**范围**：webui/embed.go、webui/embed_dev.go、webui/index.html、最小静态资源。  
**现状**：双模式资源入口已落地，本轮只补 HTTP 静态交付与 smoke，不重建资产。
**动作**：核验生产 go:embed 与 dev build tag 磁盘读取；明确静态资源路由与 SPA fallback 边界。  
**验证**：默认 build 与 -tags dev 两种构建/HTTP smoke。  
**完成标准**：两种模式均可取到最小页面；运行不依赖 Node/npm。

### 切片 6：CI 与架构守卫

**范围**：.github/workflows、架构测试目录、必要脚本。  
**动作**：建立 gofmt 检查、build、test、vet、race 和依赖方向守卫；违规样例验证规则确实生效。  
**验证**：在 PR 中执行；对违规 fixture 运行测试应失败，对正常依赖图应通过。  
**完成标准**：CI 命令与本地命令一致，失败原因可读。

### 切片 7：干净环境终验与归档材料

**范围**：全仓代码、文档、构建产物与测试记录。  
**动作**：从干净 checkout 验证子模块获取、依赖解析、默认/dev build、全量测试与文档链接。  
**验证**：go test ./...、go test -race ./...、go vet ./...、go build ./cmd/easyadmin、dev build、文档链接检查。  
**完成标准**：验收表逐条有命令/结果/证据；失败项不打勾，遗留风险明示。

## 3. 文件范围基线

预计触及：go.mod/go.sum（仅在依赖审计证明必要时）、.gitmodules、cmd/、kernel/、webui/、.github/workflows/、架构测试目录、README/AGENTS/docs。  
已有 infra 包原则上只增加适配或测试，不整体重写。任何目录/接口偏离必须先更新本计划并记录原因。

## 4. 风险与回退

- 不执行业务表 DDL，不创建真实管理员，不需要数据备份。
- 依赖 pin 未通过验证时先停止受影响切片，不通过随意升级依赖规避问题。
- embed/dev 任一模式失败时可回退该资源入口，不改变业务数据。
- 架构测试误报先修测试规则，不为通过测试而打破真实依赖边界。

## 5. 测试执行矩阵

| 层次 | 命令/操作 | 通过标准 |
|---|---|---|
| 依赖 | go mod verify、git submodule status | 无缺失/漂移 |
| 单测 | go test ./... | 全部通过 |
| 并发 | go test -race ./... | 无 data race |
| 静态 | go vet ./... | 无未解释告警 |
| 构建 | go build ./cmd/easyadmin | 成功 |
| 前端资源 | 默认模式和 -tags dev | 两种模式均可取资源 |
| 架构 | 专项架构测试 | 正反例结果符合规则 |
| 文档 | 相对链接检查 | 0 个断链 |

## 6. 完成定义

- [ ] 已按 P00 DESIGN_PLAN 逐条提供验收证据。
- [ ] 未确认设计边界没有被实现代码绕过。
- [ ] 全部测试与构建结果真实记录；未执行项明确标为未执行。
- [ ] 文档、目录、命令与实际实现一致。
- [ ] DESIGN_PLAN 记录偏差、验收日期、遗留风险和 ACCEPTED 结论。
