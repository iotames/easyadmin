// Package console 是内置控制台模块：登记仪表盘页面、权限元信息与示例路由，
// 用于验证 Manifest 元信息进入 /api/meta、禁用模块不登记资源，以及路由冲突检测。
package console

import (
	"net/http"

	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/kernel/module"
)

// Module 是控制台模块。
type Module struct{}

// New 创建控制台模块。
func New() *Module { return &Module{} }

// Manifest 声明页面与权限元信息；ID 全局唯一。
func (Module) Manifest() module.Manifest {
	return module.Manifest{
		ID:          "console",
		Version:     "0.1.0",
		Description: "控制台与运行状态示例模块",
		Pages: []module.PageMeta{
			{ID: "dashboard", Title: "仪表盘", Path: "/", Order: 0, Perm: "console.read"},
		},
		Permissions: []module.PermissionMeta{
			{Code: "console.read", Name: "查看仪表盘"},
		},
	}
}

// Declare 无配置项。
func (Module) Declare(module.ConfigRegistrar) error { return nil }

// Register 登记示例路由；页面/权限元信息由 App 依据 Manifest 统一登记。
func (Module) Register(reg module.Registrar) error {
	return reg.Handle("/api/console/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			httpapi.WriteError(w, r, http.StatusMethodNotAllowed, "COMMON_METHOD_NOT_ALLOWED", "请求方法不支持")
			return
		}
		httpapi.WriteOK(w, r, map[string]any{"pong": true})
	}))
}
