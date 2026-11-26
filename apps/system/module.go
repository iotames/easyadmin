package system

import (
	"context"
	"net/http"

	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/kernel/module"
)

// permDef 是内置权限定义。
type permDef struct {
	Code string
	Name string
}

func permissionDefs() []permDef {
	return []permDef{
		{"system.user.read", "查看用户"},
		{"system.user.write", "管理用户"},
		{"system.role.read", "查看角色"},
		{"system.role.write", "管理角色"},
		{"system.audit.read", "查看审计"},
	}
}

// Module 是内置系统模块：用户、角色、权限、会话与审计。
type Module struct {
	sys           *System
	adminPassword *string
}

// New 创建系统模块。catalog 用于按权限过滤菜单。
func New(d *db.DB, catalog *httpapi.Catalog) *Module {
	m := &Module{}
	m.sys = NewSystem(d, catalog, func() string {
		if m.adminPassword == nil {
			return ""
		}
		return *m.adminPassword
	})
	return m
}

// Manifest 声明系统页面与权限。
func (Module) Manifest() module.Manifest {
	perms := make([]module.PermissionMeta, 0, len(permissionDefs()))
	for _, p := range permissionDefs() {
		perms = append(perms, module.PermissionMeta{Code: p.Code, Name: p.Name})
	}
	return module.Manifest{
		ID:          "system",
		Version:     "0.1.0",
		Description: "内置用户、角色、权限与审计",
		Pages: []module.PageMeta{
			{ID: "system.users", Title: "用户管理", Path: "/system/users", Menu: "系统", Order: 10, Perm: "system.user.read"},
			{ID: "system.roles", Title: "角色管理", Path: "/system/roles", Menu: "系统", Order: 20, Perm: "system.role.read"},
			{ID: "system.audit", Title: "审计日志", Path: "/system/audit", Menu: "系统", Order: 30, Perm: "system.audit.read"},
		},
		Permissions: perms,
	}
}

// Declare 登记初始管理员口令配置。
func (m *Module) Declare(reg module.ConfigRegistrar) error {
	m.adminPassword = new(string)
	return reg.Register(m.adminPassword, "EASYADMIN_ADMIN_PASSWORD", "", "初始管理员口令",
		"非回环地址首次初始化时必填；同时作为确认口令")
}

// Register 登记认证与系统 API 路由。
func (m *Module) Register(reg module.Registrar) error { return m.sys.registerRoutes(reg) }

// Start 幂等准备系统角色与权限码，不执行 DDL。
func (m *Module) Start(ctx context.Context) error { return m.sys.Bootstrap(ctx) }

// Stop 无后台资源。
func (m *Module) Stop(context.Context) error { return nil }

// Meta 实现 httpapi.MetaSource。
func (m *Module) Meta(ctx context.Context, r *http.Request) (httpapi.MetaValues, error) {
	return m.sys.meta(ctx, r)
}

// MarkOrphanedPermissions 标记当前 Manifest 未声明的权限为 orphaned。
func (m *Module) MarkOrphanedPermissions(ctx context.Context, codes []string) error {
	return m.sys.MarkOrphanedPermissions(ctx, codes)
}

// RegisterPermission 幂等同步权限码（供组合根同步全部模块 Manifest）。
func (m *Module) RegisterPermission(ctx context.Context, code, name string) error {
	return m.sys.RegisterPermission(ctx, code, name)
}
