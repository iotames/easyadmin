package system

import (
	"context"
	"net/http"

	"github.com/iotames/easyadmin/infra/conf"
	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/kernel/module"
)

// permissionDefs 是内置权限定义，也是 Manifest 的唯一声明源。
func permissionDefs() []module.PermissionMeta {
	return []module.PermissionMeta{
		{Code: "system.user.read", Name: "查看用户"},
		{Code: "system.user.write", Name: "管理用户"},
		{Code: "system.role.read", Name: "查看角色"},
		{Code: "system.role.write", Name: "管理角色"},
		{Code: "system.audit.read", Name: "查看审计"},
		{Code: "system.config.read", Name: "查看配置"},
	}
}

// ManifestSource 汇总全部启用模块的 Manifest，由组合根在 Start 前注入。
// 权限同步统一由 system 模块执行，组合根不再逐个模块遍历。
type ManifestSource interface {
	Manifests() []module.Manifest
}

// Module 是内置系统模块：用户、角色、权限、会话与审计。
type Module struct {
	sys            *System
	adminPassword  *string
	manifestSource ManifestSource
	config         ConfigLister
}

// ConfigLister 提供只读配置元数据（通常由 infra/conf.Manager 实现）。
type ConfigLister interface {
	List() []conf.ConfigItem
}

// Option 调整系统模块的可选依赖。
type Option func(*Module)

// WithConfig 注入配置元数据来源；未注入时配置页返回空列表。
func WithConfig(l ConfigLister) Option {
	return func(m *Module) { m.config = l }
}

// New 创建系统模块。catalog 用于按权限过滤菜单。
func New(d *db.DB, catalog *httpapi.Catalog, opts ...Option) *Module {
	m := &Module{}
	for _, opt := range opts {
		opt(m)
	}
	m.sys = NewSystem(d, catalog, func() string {
		if m.adminPassword == nil {
			return ""
		}
		return *m.adminPassword
	})
	m.sys.SetConfig(m.config)
	return m
}

// SetManifestSource 注入全部模块 Manifest 来源；须在 Start 前调用。
func (m *Module) SetManifestSource(src ManifestSource) { m.manifestSource = src }

// Manifest 声明系统页面与权限。
func (Module) Manifest() module.Manifest {
	return module.Manifest{
		ID:          "system",
		Version:     "0.1.0",
		Description: "内置用户、角色、权限与审计",
		Pages: []module.PageMeta{
			{ID: "system.users", Title: "用户管理", Path: "/system/users", Menu: "系统", Order: 10, Perm: "system.user.read"},
			{ID: "system.roles", Title: "角色管理", Path: "/system/roles", Menu: "系统", Order: 20, Perm: "system.role.read"},
			{ID: "system.audit", Title: "审计日志", Path: "/system/audit", Menu: "系统", Order: 30, Perm: "system.audit.read"},
			{ID: "system.config", Title: "配置中心", Path: "/system/config", Menu: "系统", Order: 40, Perm: "system.config.read"},
			{ID: "profile", Title: "个人资料", Path: "/profile", Order: 99},
		},
		Permissions: permissionDefs(),
	}
}

// Declare 登记初始管理员口令配置。
func (m *Module) Declare(reg module.ConfigRegistrar) error {
	m.adminPassword = new(string)
	return reg.Register(m.adminPassword, "EASYADMIN_ADMIN_PASSWORD", "", "初始管理员口令",
		module.ConfigMeta{Group: "system", Order: 10, Secret: true, HotReload: module.HotReloadRestart},
		"非回环地址首次初始化时必填；同时作为确认口令")
}

// Register 登记认证与系统 API 路由。
func (m *Module) Register(reg module.Registrar) error { return m.sys.registerRoutes(reg) }

// Start 幂等准备系统角色，并统一同步全部模块 Manifest 权限，不执行 DDL。
func (m *Module) Start(ctx context.Context) error {
	if err := m.sys.Bootstrap(ctx); err != nil {
		return err
	}
	return m.syncManifestPermissions(ctx)
}

// manifestPermissions 返回全部启用模块声明的权限；未注入来源时退回本模块 Manifest。
func (m *Module) manifestPermissions() []module.PermissionMeta {
	if m.manifestSource == nil {
		return m.Manifest().Permissions
	}
	seen := map[string]bool{}
	out := make([]module.PermissionMeta, 0)
	for _, mf := range m.manifestSource.Manifests() {
		for _, p := range mf.Permissions {
			if p.Code == "" || seen[p.Code] {
				continue
			}
			seen[p.Code] = true
			out = append(out, p)
		}
	}
	return out
}

// syncManifestPermissions 幂等写入全部权限码，并把不再声明的权限标记为 orphaned。
func (m *Module) syncManifestPermissions(ctx context.Context) error {
	perms := m.manifestPermissions()
	codes := make([]string, 0, len(perms))
	for _, p := range perms {
		codes = append(codes, p.Code)
		if err := m.sys.RegisterPermission(ctx, p.Code, p.Name); err != nil {
			return err
		}
	}
	return m.sys.MarkOrphanedPermissions(ctx, codes)
}

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

// RegisterPermission 幂等同步单个权限码。
func (m *Module) RegisterPermission(ctx context.Context, code, name string) error {
	return m.sys.RegisterPermission(ctx, code, name)
}
