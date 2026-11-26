# 执行总纲

> 执行依据：docs/plan/README.md（计划目录工作宪法）——记号语义、状态机、关口与裁决以宪法为准。

项目状态：进行中

## §0 断点续传（接手者从这里开始）
1. 读项目状态行与 §1 总表，定位第一个非「已实施」的步骤。
2. 若为「实施中」，先跑该 STEP 验证节的接手核实命令；通过则从首个未勾选项继续，禁止重做。
3. 状态迁移收口时跑 §4.1 附状态一致性自检；严禁凭会话记忆做实施判断。

## §1 状态总表
| # | STEP | 内容 | 依赖 | 状态 |
|---|------|------|------|------|
| 1 | P01_RUNTIME/STEP1_KERNEL.md | 扩展 kernel/module 与 kernel/app（Manifest/Registrar/禁用过滤） | — | 已实施 |
| 2 | P01_RUNTIME/STEP2_INFRA.md | 新建 infra/logx、infra/apperror、infra/httpapi | 1 | 已实施 |
| 3 | P01_RUNTIME/STEP3_WIRE.md | 改造组合根 + 最小内置模块 + 文档同步 + P01 验收 | 2 | 已实施 |
| 4 | P02_DATA_IDENTITY/STEP1_SCHEMA.md | SQLite 驱动 + 核心 SQL + 结构同步 | 3 | 待实施 |
| 5 | P02_DATA_IDENTITY/STEP2_IDENTITY.md | infra/auth + infra/audit + apps/system 用例与 API | 4 | 待实施 |
| 6 | P02_DATA_IDENTITY/STEP3_WIRE.md | 组合根装配 + 中间件追加 + features.auth + P02 验收 | 5 | 待实施 |

## §2 执行期红线（自仓库法摘录，冲突时以仓库法原文为准）
- 统一入口：conf/logx/db/httpapi/apperror/auth/audit 各只有一个受支持入口。
- SQL-first、参数化；不在 Register 中执行 DDL；不引入 ORM。
- WebUI 只经 core/api.js；P02 不改前端契约，只切 features.auth。
- 提交：用户已明确授权实施并提交推送（见本轮指令）。

## §3 进度日志（一行一事）
| 日期 | 执行者 | STEP | 结果 |
|---|---|---|---|
| 2026-11-26 | agent | 文档定稿 | P01/P02 DESIGN+IMPL 按审核结论修订 |
| 2026-11-26 | agent | P01 STEP1 | kernel/module+app 扩展，禁用过滤，测试通过 |
| 2026-11-26 | agent | P01 STEP2 | infra/logx+apperror+httpapi，冲突检测/中间件/绑定测试通过 |
| 2026-11-26 | agent | P01 STEP3 | 组合根改造+console 模块+HTTP/浏览器验收，P01 归档 ACCEPTED |
