package console

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iotames/easyadmin/kernel/module"
)

type reg struct {
	routes map[string]http.Handler
	pages  []string
	perms  []string
}

func (r *reg) Handle(p string, h http.Handler) error {
	if r.routes == nil {
		r.routes = map[string]http.Handler{}
	}
	r.routes[p] = h
	return nil
}
func (r *reg) RegisterPage(p module.PageMeta) error { r.pages = append(r.pages, p.ID); return nil }
func (r *reg) RegisterPermission(p module.PermissionMeta) error {
	r.perms = append(r.perms, p.Code)
	return nil
}

func TestRegisterHandlesPing(t *testing.T) {
	r := &reg{}
	if err := New().Register(r); err != nil {
		t.Fatal(err)
	}
	h, ok := r.routes["/api/console/ping"]
	if !ok {
		t.Fatal("未登记 /api/console/ping")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/console/ping", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", rec.Code)
	}
}
