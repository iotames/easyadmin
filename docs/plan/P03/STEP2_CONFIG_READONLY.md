# STEP2：配置元数据与只读接口

状态：已实施

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
- [x] 定义配置元数据类型并强制注册声明（`module.ConfigMeta`）
- [x] 扩展 ConfigItem：source/secret/hot_reload/writable/configured ✓（`go test ./infra/conf/...`，通过）
- [x] 实现来源判定（args/env/file/default）✓（`TestConfigSourceByLoadOrder`，通过）
- [x] 标记 `EASYADMIN_ADMIN_PASSWORD`、`EASYADMIN_DB_DSN` 为敏感
- [x] 实现 `GET /api/system/config` 并遮蔽敏感值 ✓（`TestConfigEndpointMasksSecrets`，通过）
- [x] 补来源判定与敏感遮蔽测试

## 验证
- 接手核实命令：`go test ./infra/conf/... ./apps/system/...`
- 本步完整验证：
  - [x] `go test ./infra/conf/... ./apps/system/...`
  - [x] `go vet ./infra/conf/... ./apps/system/...`

## 完成标准
- 配置项含 `source/secret/hot_reload/writable`；来源测试覆盖环境变量与命令行。
- 响应中不出现管理员初始口令与数据库 DSN 原值。
- 路由表不存在 `PUT /api/system/config/{key}`。

## 实施回填区
### 产物锚点清单
- 新增类型：`module.ConfigMeta`、`conf.Meta`、`system.ConfigLister`
- 新增接口：`Manager.RegisterWith`、`module.ConfigRegistrar.Register` 新签名
- 新增端点：`GET /api/system/config`
- 新增测试：`TestConfigSourceByLoadOrder`、`TestConfigSecretMaskedAndMetaValidated`、`TestConfigEndpointMasksSecrets`
### 偏差与现场记录
- easyconf 会自动把默认值写入工作目录 `.env`/`default.env`，故"文件存在该键"不等于用户显式配置；来源判定增加"当前值≠默认值"前置条件，避免全部配置误报为 `file`。
- 配置项 JSON 字段按 `docs/API.md` §9 对齐为 `value` / `default`；P03 `writable` 恒为 false。
