package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/iotames/easyadmin/infra/logx"
)

func testAssets() fstest.MapFS {
	return fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>index</html>")},
		"app.js":     &fstest.MapFile{Data: []byte("console.log('ok')")},
	}
}

func newTestHandler(t *testing.T, rt *Router, cat *Catalog, ready bool, meta MetaSource, features map[string]bool) http.Handler {
	t.Helper()
	return NewHandler(HandlerOptions{
		Info:      AppInfo{Name: "demo", Version: "test"},
		Readiness: fakeReady(ready),
		Assets:    testAssets(),
		Logger:    logx.New("error"),
		Router:    rt,
		Catalog:   cat,
		Features:  features,
		Meta:      meta,
	})
}

type fakeReady bool

func (f fakeReady) Ready() bool { return bool(f) }

func decode(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("解码响应失败: %v", err)
	}
	return m
}

func TestMetaWithCatalog(t *testing.T) {
	cat := NewCatalog()
	if err := cat.AddPage(Page{ID: "system.users", Title: "用户", Path: "/system/users", Menu: "系统"}); err != nil {
		t.Fatal(err)
	}
	if err := cat.AddPermission(Permission{Code: "system.user.read", Name: "查看用户"}); err != nil {
		t.Fatal(err)
	}
	h := newTestHandler(t, NewRouter(), cat, true, nil, map[string]bool{"auth": false})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/meta", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 = %d", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("缺少 X-Request-ID 响应头")
	}
	env := decode(t, rec.Body)
	meta := env["meta"].(map[string]any)
	if meta["request_id"] == "" || meta["request_id"] == nil {
		t.Fatal("meta.request_id 为空")
	}
	data := env["data"].(map[string]any)
	if data["initialized"] != false {
		t.Fatalf("initialized = %v", data["initialized"])
	}
	perms := data["permissions"].([]any)
	if len(perms) != 1 || perms[0] != "system.user.read" {
		t.Fatalf("permissions = %v", perms)
	}
	menus := data["menus"].([]any)
	if len(menus) != 1 {
		t.Fatalf("menus = %v", menus)
	}
	group := menus[0].(map[string]any)
	if group["key"] != "系统" {
		t.Fatalf("菜单分组 = %v", group["key"])
	}
	features := data["features"].(map[string]any)
	if features["auth"] != false {
		t.Fatalf("features = %v", features)
	}
}

func TestMetaSourceOverrides(t *testing.T) {
	meta := fakeMeta{MetaValues{Initialized: true, Permissions: []string{"x.y"}, Features: map[string]bool{"auth": true}}}
	h := newTestHandler(t, NewRouter(), NewCatalog(), true, meta, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/meta", nil))
	data := decode(t, rec.Body)["data"].(map[string]any)
	if data["initialized"] != true {
		t.Fatal("MetaSource 未生效")
	}
	if data["features"].(map[string]any)["auth"] != true {
		t.Fatal("features 未取 MetaSource")
	}
}

type fakeMeta struct{ v MetaValues }

func (f fakeMeta) Meta(context.Context, *http.Request) (MetaValues, error) { return f.v, nil }

func TestProbes(t *testing.T) {
	h := newTestHandler(t, NewRouter(), NewCatalog(), true, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz = %d", rec.Code)
	}

	h2 := newTestHandler(t, NewRouter(), NewCatalog(), false, nil, nil)
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz 未就绪 = %d", rec2.Code)
	}
}

func TestPanicRecoveredWithRequestID(t *testing.T) {
	rt := NewRouter()
	if err := rt.Handle("/api/boom", http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("kaboom") })); err != nil {
		t.Fatal(err)
	}
	h := newTestHandler(t, rt, NewCatalog(), true, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("panic 状态码 = %d", rec.Code)
	}
	env := decode(t, rec.Body)
	body := env["error"].(map[string]any)
	if body["code"] != "COMMON_INTERNAL" {
		t.Fatalf("panic 错误码 = %v", body["code"])
	}
	if env["meta"].(map[string]any)["request_id"] == nil {
		t.Fatal("panic 响应缺少 request_id")
	}
}

func TestUnknownAPIReturns404(t *testing.T) {
	h := newTestHandler(t, NewRouter(), NewCatalog(), true, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("未知 API = %d", rec.Code)
	}
}

func TestSPAFallback(t *testing.T) {
	h := newTestHandler(t, NewRouter(), NewCatalog(), true, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "index") {
		t.Fatalf("SPA 回退失败: %d %q", rec.Code, rec.Body.String())
	}
	miss := httptest.NewRecorder()
	h.ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/missing.js", nil))
	if miss.Code != http.StatusNotFound {
		t.Fatalf("缺失静态资源 = %d", miss.Code)
	}
}

func TestAppAssetsMount(t *testing.T) {
	appAssets := fstest.MapFS{
		"pages.js":       &fstest.MapFile{Data: []byte("export default {}")},
		"styles/app.css": &fstest.MapFile{Data: []byte(":root{}")},
	}
	h := NewHandler(HandlerOptions{
		Info:      AppInfo{Name: "demo", Version: "test"},
		Readiness: fakeReady(true),
		Assets:    testAssets(),
		AppAssets: appAssets,
		Logger:    logx.New("error"),
	})

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app/pages.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "export default") {
		t.Fatalf("业务 pages.js = %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/assets/app/styles/app.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("业务 app.css = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "console.log") {
		t.Fatalf("core 资源被覆盖 = %d %q", rec.Code, rec.Body.String())
	}

	h2 := newTestHandler(t, NewRouter(), NewCatalog(), true, nil, nil)
	miss := httptest.NewRecorder()
	h2.ServeHTTP(miss, httptest.NewRequest(http.MethodGet, "/assets/app/pages.js", nil))
	if miss.Code != http.StatusNotFound {
		t.Fatalf("未配置 AppAssets 应为 404，实际 %d", miss.Code)
	}
}

func TestRouterRejectsDuplicateAndReserved(t *testing.T) {
	rt := NewRouter()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	if err := rt.Handle("/api/x", h); err != nil {
		t.Fatal(err)
	}
	if err := rt.Handle("/api/x", h); err == nil {
		t.Fatal("重复路由应失败")
	}
	if err := rt.Handle("/api/meta", h); err == nil {
		t.Fatal("保留路径应失败")
	}
}

func TestCatalogRejectsDuplicate(t *testing.T) {
	c := NewCatalog()
	if err := c.AddPage(Page{ID: "p1"}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddPage(Page{ID: "p1"}); err == nil {
		t.Fatal("重复页面应失败")
	}
	if err := c.AddPermission(Permission{Code: "c1"}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddPermission(Permission{Code: "c1"}); err == nil {
		t.Fatal("重复权限应失败")
	}
}

func TestMenuOrderByOrder(t *testing.T) {
	cat := NewCatalog()
	pages := []Page{
		{ID: "profile", Title: "个人资料", Path: "/profile", Order: 99},
		{ID: "system.users", Title: "用户管理", Path: "/system/users", Menu: "系统", Order: 10, Perm: "system.user.read"},
		{ID: "system.config", Title: "配置中心", Path: "/system/config", Menu: "系统", Order: 40, Perm: "system.config.read"},
		{ID: "dashboard", Title: "仪表盘", Path: "/", Order: 0, Perm: "console.read"},
		{ID: "system.roles", Title: "角色管理", Path: "/system/roles", Menu: "系统", Order: 20, Perm: "system.role.read"},
		{ID: "system.audit", Title: "审计日志", Path: "/system/audit", Menu: "系统", Order: 30, Perm: "system.audit.read"},
	}
	for _, p := range pages {
		if err := cat.AddPage(p); err != nil {
			t.Fatal(err)
		}
	}
	menus := cat.MenusFor([]string{"*"})
	if len(menus) != 3 {
		t.Fatalf("顶层菜单数量 = %d: %+v", len(menus), menus)
	}
	if menus[0].Key != "dashboard" || menus[1].Key != "系统" || menus[2].Key != "profile" {
		t.Fatalf("顶层顺序错误: %s, %s, %s", menus[0].Key, menus[1].Key, menus[2].Key)
	}
	got := make([]string, 0, len(menus[1].Children))
	for _, child := range menus[1].Children {
		got = append(got, child.Key)
	}
	want := []string{"system.users", "system.roles", "system.audit", "system.config"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("分组内顺序 = %v，期望 %v", got, want)
	}

	filtered := cat.MenusFor([]string{"console.read", "system.user.read"})
	if len(filtered) != 3 || filtered[1].Key != "系统" || len(filtered[1].Children) != 1 {
		t.Fatalf("权限过滤后菜单错误: %+v", filtered)
	}
}

func TestDecodeJSONAndValidate(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	req := httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader(`{"name":"a","extra":1}`))
	var p payload
	if err := DecodeJSON(req, &p); err == nil {
		t.Fatal("未知字段应返回 400")
	}
	req2 := httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader(`{"name":"a"}`))
	if err := DecodeJSON(req2, &p); err != nil {
		t.Fatalf("合法请求失败: %v", err)
	}
	req3 := httptest.NewRequest(http.MethodPost, "/api/x", strings.NewReader("{"))
	if err := DecodeJSON(req3, &p); err == nil {
		t.Fatal("非法 JSON 应返回 400")
	}
	if err := Validate(map[string][]string{"name": {"不能为空"}}); err == nil {
		t.Fatal("字段错误应返回 422 错误")
	}
}
