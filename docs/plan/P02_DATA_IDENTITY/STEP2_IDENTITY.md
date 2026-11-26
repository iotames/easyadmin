# STEP2：infra/auth + infra/audit + apps/system 用例与 API

状态：待实施

## 目标
实现密码哈希、Session、CSRF、登录限流、授权判定；审计记录器；用户/角色/权限/审计的仓储、用例与 handler。

## 改动文件清单
- infra/auth/*（password/session/csrf/ratelimit/authz）
- infra/audit/*（Recorder + SQL 实现）
- apps/system/*（store/service/handlers/manifest）

## 实施步骤
- [ ] infra/auth：bcrypt、session token SHA-256、CSRF、限流、授权
- [ ] infra/audit：只追加 Recorder
- [ ] apps/system：用户/角色/权限/会话/审计仓储与用例
- [ ] API：setup/login/logout/me/password、users/roles/permissions/audit
- [ ] 最后超级管理员保护、唯一约束 409、字段校验 422

## 验证
- 接手核实命令：go test ./infra/auth/... ./infra/audit/... ./apps/system/...
- 本步完整验证：
  - [ ] SQLite 集成 + HTTP 集成（401/403/409/422/CSRF/限流）

## 完成标准
- 安全与授权验收项均有可复现测试；敏感字段不回显。
