# STEP1：SQLite 驱动 + 核心 SQL + 结构同步

状态：待实施

## 目标
引入纯 Go SQLite 驱动，提供核心表 SQL 与 db.SQLSource，复用 infra/db 的 EnsureTables/DiffSchema 完成幂等结构同步与只读差异预览。

## 改动文件清单
- go.mod（modernc.org/sqlite）
- sql/sqlite/core/*.sql、sql/source.go（go:embed）
- apps/system/schema.go（TableSpec 清单与 Ensure）
- 测试：sqlite 集成

## 实施步骤
- [ ] 引入 modernc.org/sqlite 并注册驱动
- [ ] 编写 7 张核心表 SQL 脚本与 SQLSource
- [ ] TableSpec 清单 + EnsureTables + DiffSchema 预览
- [ ] 新建 DB/重复执行/缺表补齐测试

## 验证
- 接手核实命令：go test ./sql/... ./apps/system/...
- 本步完整验证：
  - [ ] 删除临时 db 后重复 Ensure 幂等
  - [ ] DiffSchema 只读

## 完成标准
- 空库可初始化，重复 Ensure 无冲突，差异只预览不执行。
