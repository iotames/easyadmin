package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	kernelapp "github.com/iotames/easyadmin/kernel/app"
)

const (
	codeNotFound         = "COMMON_NOT_FOUND"
	codeMethodNotAllowed = "COMMON_METHOD_NOT_ALLOWED"
)

type requestIDKey struct{}

// apiEnvelope 是 /api/ 成功与失败响应的统一外壳。
type apiEnvelope struct {
	OK    bool         `json:"ok"`
	Data  any          `json:"data,omitempty"`
	Error *apiError    `json:"error,omitempty"`
	Meta  responseMeta `json:"meta"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type responseMeta struct {
	RequestID string `json:"request_id"`
}

type appInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// metaData 是 /api/meta 的 P00 子集：登录态、权限、菜单和 CSRF 由 P02 起补齐。
type metaData struct {
	App         appInfo         `json:"app"`
	Initialized bool            `json:"initialized"`
	User        any             `json:"user"`
	Permissions []string        `json:"permissions"`
	Menus       []any           `json:"menus"`
	Features    map[string]bool `json:"features"`
	CSRFToken   string          `json:"csrf_token"`
}

// newHandler 组装 P00 的最小 HTTP 面：探针、meta、API 统一错误和静态资源。
// 业务路由、权限和完整中间件在 P01 收敛到 infra/httpapi。
func newHandler(app *kernelapp.App, assets fs.FS) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, r)
			return
		}
		writeProbe(w, http.StatusOK, map[string]any{"ok": true, "status": "alive"})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, r)
			return
		}
		if !app.Ready() {
			writeProbe(w, http.StatusServiceUnavailable, map[string]any{
				"ok": false, "status": "not_ready", "checks": map[string]string{"app": "starting"},
			})
			return
		}
		writeProbe(w, http.StatusOK, map[string]any{
			"ok": true, "status": "ready", "checks": map[string]string{"app": "ok"},
		})
	})
	mux.HandleFunc("/api/meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, r)
			return
		}
		writeOK(w, r, metaData{
			App:         appInfo{Name: app.Name(), Version: app.Version()},
			Permissions: []string{},
			Menus:       []any{},
			Features:    map[string]bool{},
		})
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, codeNotFound, "接口不存在")
	})
	mux.Handle("/", spaHandler{fsys: assets, files: http.FileServer(http.FS(assets))})
	return requestID(mux)
}

// requestID 生成或透传请求 ID，并只放在响应头与 meta.request_id。
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func newRequestID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buf)
}

func requestIDFrom(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey{}).(string)
	return id
}

func writeOK(w http.ResponseWriter, r *http.Request, data any) {
	writeJSON(w, http.StatusOK, apiEnvelope{OK: true, Data: data, Meta: responseMeta{RequestID: requestIDFrom(r)}})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, apiEnvelope{
		Error: &apiError{Code: code, Message: message},
		Meta:  responseMeta{RequestID: requestIDFrom(r)},
	})
}

func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD")
	writeError(w, r, http.StatusMethodNotAllowed, codeMethodNotAllowed, "请求方法不支持")
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeProbe 输出探针响应；探针非 /api/ 端点，按 API.md 使用独立结构。
func writeProbe(w http.ResponseWriter, status int, body any) {
	writeJSON(w, status, body)
}

// spaHandler 提供静态资源与前端路由回退：
// 存在的文件直接交付；无扩展名的未知路径回退 index.html；带扩展名的缺失资源返回 404。
type spaHandler struct {
	fsys  fs.FS
	files http.Handler
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w, r)
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
	writeError(w, r, http.StatusNotFound, codeNotFound, "资源不存在")
}
