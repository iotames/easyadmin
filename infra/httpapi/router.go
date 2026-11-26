package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
)

// 内置保留路径，模块不得占用。
var reservedPatterns = map[string]bool{
	"/":         true,
	"/healthz":  true,
	"/readyz":   true,
	"/api/":     true,
	"/api/meta": true,
}

// Router 收集模块路由并检测重复登记。
type Router struct {
	mu       sync.Mutex
	mux      *http.ServeMux
	patterns map[string]bool
}

// NewRouter 创建路由表。
func NewRouter() *Router {
	return &Router{mux: http.NewServeMux(), patterns: make(map[string]bool)}
}

// Handle 登记路由；重复或保留路径返回错误。
func (rt *Router) Handle(pattern string, h http.Handler) error {
	if pattern == "" {
		return fmt.Errorf("httpapi: 路由 pattern 为空")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if reservedPatterns[pattern] {
		return fmt.Errorf("httpapi: 路由 %s 为保留路径", pattern)
	}
	if rt.patterns[pattern] {
		return fmt.Errorf("httpapi: 路由重复登记: %s", pattern)
	}
	rt.patterns[pattern] = true
	rt.mux.Handle(pattern, h)
	return nil
}

// Mux 返回底层 ServeMux。
func (rt *Router) Mux() *http.ServeMux { return rt.mux }

// Patterns 返回已登记路由（排序后），供测试与诊断。
func (rt *Router) Patterns() []string {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]string, 0, len(rt.patterns))
	for p := range rt.patterns {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
