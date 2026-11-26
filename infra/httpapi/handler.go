package httpapi

import (
	"context"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/iotames/easyadmin/infra/apperror"
	"github.com/iotames/easyadmin/infra/logx"
)

// AppInfo 是 /api/meta 的应用静态信息。
type AppInfo struct {
	Name    string
	Version string
}

// Readiness 报告依赖是否就绪。
type Readiness interface {
	Ready() bool
}

// MetaValues 是动态元信息；由业务侧（P02 认证）提供，未提供时用注册目录兜底。
type MetaValues struct {
	Initialized bool
	User        any
	Permissions []string
	Menus       []MenuNode
	Features    map[string]bool
	CSRFToken   string
}

// MetaSource 提供动态 meta；P02 据此返回登录态、权限与 CSRF。
type MetaSource interface {
	Meta(ctx context.Context, r *http.Request) (MetaValues, error)
}

// HandlerOptions 是 HTTP 面装配参数。
type HandlerOptions struct {
	Info      AppInfo
	Readiness Readiness
	Assets    fs.FS
	AppAssets fs.FS
	Logger    *logx.Logger
	Router    *Router
	Catalog   *Catalog
	Features  map[string]bool
	Meta      MetaSource
}

type metaData struct {
	App         appInfo         `json:"app"`
	Initialized bool            `json:"initialized"`
	User        any             `json:"user"`
	Permissions []string        `json:"permissions"`
	Menus       []MenuNode      `json:"menus"`
	Features    map[string]bool `json:"features"`
	CSRFToken   string          `json:"csrf_token"`
}

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// NewHandler 组装探针、meta、模块路由、/api 兜底与静态资源，并套上固定中间件链。
func NewHandler(opts HandlerOptions) http.Handler {
	rt := opts.Router
	if rt == nil {
		rt = NewRouter()
	}
	cat := opts.Catalog
	if cat == nil {
		cat = NewCatalog()
	}
	mux := rt.Mux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, http.StatusMethodNotAllowed, apperror.CodeMethodNotAllowed, "请求方法不支持")
			return
		}
		WriteProbe(w, http.StatusOK, map[string]any{"ok": true, "status": "alive"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, http.StatusMethodNotAllowed, apperror.CodeMethodNotAllowed, "请求方法不支持")
			return
		}
		if opts.Readiness != nil && !opts.Readiness.Ready() {
			WriteProbe(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "status": "not_ready", "checks": map[string]string{"app": "starting"},
			})
			return
		}
		WriteProbe(w, http.StatusOK, map[string]any{
			"ok": true, "status": "ready", "checks": map[string]string{"app": "ok"},
		})
	})
	mux.HandleFunc("/api/meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			WriteError(w, r, http.StatusMethodNotAllowed, apperror.CodeMethodNotAllowed, "请求方法不支持")
			return
		}
		mv := MetaValues{Features: opts.Features}
		if opts.Meta != nil {
			got, err := opts.Meta.Meta(r.Context(), r)
			if err != nil {
				if opts.Logger != nil {
					opts.Logger.WithRequestID(RequestIDFrom(r)).Error("组装 meta 失败", "err", err)
				}
				WriteAppError(w, r, err)
				return
			}
			mv = got
		} else {
			mv.Permissions = cat.PermissionCodes()
			mv.Menus = cat.Menus()
		}
		data := metaData{
			App:         appInfo{Name: opts.Info.Name, Version: opts.Info.Version},
			Initialized: mv.Initialized,
			User:        mv.User,
			Permissions: nonNilStrings(mv.Permissions),
			Menus:       nonNilMenus(mv.Menus),
			Features:    nonNilFeatures(mv.Features),
			CSRFToken:   mv.CSRFToken,
		}
		WriteOK(w, r, data)
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, apperror.CodeNotFound, "接口不存在")
	})
	if opts.AppAssets != nil {
		// 业务资源固定挂载在 /assets/app/，不会覆盖 core 的 /assets/ 资源。
		mux.Handle("/assets/app/", http.StripPrefix("/assets/app/", http.FileServer(http.FS(opts.AppAssets))))
	}
	mux.Handle("/", spaHandler{fsys: opts.Assets, files: http.FileServer(http.FS(opts.Assets))})

	return Chain(mux, RequestID, Recover(opts.Logger), AccessLog(opts.Logger))
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func nonNilMenus(in []MenuNode) []MenuNode {
	if in == nil {
		return []MenuNode{}
	}
	return in
}

func nonNilFeatures(in map[string]bool) map[string]bool {
	if in == nil {
		return map[string]bool{}
	}
	return in
}

// spaHandler 提供静态资源与前端路由回退：
// 存在的文件直接交付；无扩展名的未知路径回退 index.html；带扩展名的缺失资源返回 404。
type spaHandler struct {
	fsys  fs.FS
	files http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		WriteError(w, r, http.StatusMethodNotAllowed, apperror.CodeMethodNotAllowed, "请求方法不支持")
		return
	}
	name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if name == "" {
		name = "index.html"
	}
	if fs.ValidPath(name) {
		if f, err := h.fsys.Open(name); err == nil {
			_ = f.Close()
			h.files.ServeHTTP(w, r)
			return
		}
	}
	if path.Ext(name) == "" {
		clone := r.Clone(r.Context())
		clone.URL.Path = "/"
		h.files.ServeHTTP(w, clone)
		return
	}
	WriteError(w, r, http.StatusNotFound, apperror.CodeNotFound, "资源不存在")
}
