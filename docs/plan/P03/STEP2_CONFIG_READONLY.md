# STEP2：配置元数据与只读接口

状态：待实施

## 目标
- 配置注册必须显式声明分组、顺序、敏感级别、热更语义（编译期强制）。
- `infra/conf` 判定来源：command_line > environment > file > default。
- `GET /api/system/config`（`system.config.read`）返回分组元数据；`writable=false`；敏感项不回显值；不实现 PUT。

## 改动文件清单
- `kernel/module/module.go`（ConfigMeta、ConfigRegistrar 签名）
- `infra/conf/interface.go`、`infra/conf/impl.go`
- `cmd/easyadmin/main.go`、`cmd/easyadmin/registrar.go`
- `apps/system/module.go`、`apps/system/handlers.go`
- `infra/conf/conf_test.go`、`apps/system/system_test.go`

## 实施步骤
- [ ] 定义配置元数据类型并强制注册声明
- [ ] 扩展 ConfigItem：source/secret/hot_reload/writable/configured
- [ ] 实现来源判定（args/env/file/default）
- [ ] 标记 `EASYADMIN_ADMIN_PASSWORD`、`EASYADMIN_DB_DSN` 为敏感
- [ ] 实现 `GET /api/system/config` 并遮蔽敏感值
- [ ] 补来源判定与敏感遮蔽测试

## 验证
- 接手核实命令：`go test ./infra/conf/... ./apps/system/...`
- 本步完整验证：
  - [ ] `go test ./infra/conf/... ./apps/system/...`
  - [ ] `go vet ./infra/conf/... ./apps/system/...`

## 完成标准
- 配置项含 `source/secret/hot_reload/writable`；来源测试覆盖环境变量与命令行。
- 响应中不出现管理员初始口令与数据库 DSN 原值。
- 路由表不存在 `PUT /api/system/config/{key}`。

## 实施回填区
### 产物锚点清单
- 新增类型：`module.ConfigMeta`、`conf.Meta`
- 新增接口：`Manager.RegisterWith`
- 新增端点：`GET /api/system/config`
- 新增测试：`TestConfigSource`、`TestConfigSecretMasked`
### 偏差与现场记录
- 待回填
