// Package module 定义运行期模块契约与 Manifest。
//
// 模块只声明自身能力，不持有全局状态；装配、依赖校验和生命周期由 kernel/app 负责。
package module

import (
	"context"
	"net/http"
)

// Manifest 是模块的静态元信息。ID 必须稳定且全局唯一，依赖使用模块 ID。
// Pages 与 Permissions 是菜单/页面/权限元信息的权威来源（D05），由 httpapi 汇总进 /api/meta。
type Manifest struct {
	ID          string
	Version     string
	Description string
	DependsOn   []string
	Pages       []PageMeta
	Permissions []PermissionMeta
}

// PageMeta 描述一个前端页面入口。ID 必须全局唯一并与前端 entry 一一对应。
// Menu 为空表示不进入菜单；Perm 为空表示登录即可访问。
type PageMeta struct {
	ID    string
	Title string
	Path  string
	Menu  string
	Order int
	Perm  string
}

// PermissionMeta 描述一个权限码。Code 形如 system.user.read，必须全局唯一。
type PermissionMeta struct {
	Code string
	Name string
}

// ConfigRegistrar 接收配置元数据声明，由组合根提供实现。
type ConfigRegistrar interface {
	Register(pval any, name, defval, title string, usage ...string) error
}

// Registrar 接收运行资源登记，由组合根提供实现。
// 实现负责路径前缀、中间件、重复路由/页面/权限冲突校验；模块不直接操作底层 Server。
type Registrar interface {
	// Handle 登记路由；重复 pattern 必须返回错误。
	Handle(pattern string, h http.Handler) error
	// RegisterPage 登记页面元信息；重复 Page ID 必须返回错误。
	RegisterPage(PageMeta) error
	// RegisterPermission 登记权限码；重复 Code 必须返回错误。
	RegisterPermission(PermissionMeta) error
}

// Module 是可被显式装配的模块。
type Module interface {
	Manifest() Manifest
	// Declare 在配置加载前登记配置元数据，不得做 IO 或启动 goroutine。
	Declare(ConfigRegistrar) error
	// Register 在配置就绪后登记路由等服务资源，不得执行 DDL。
	Register(Registrar) error
}

// Lifecycle 由需要后台资源的模块实现；启动和停止必须响应传入的 context。
type Lifecycle interface {
	Start(context.Context) error
	Stop(context.Context) error
}
