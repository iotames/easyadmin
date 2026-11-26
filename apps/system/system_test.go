package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	_ "modernc.org/sqlite"

	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/infra/logx"
	"github.com/iotames/easyadmin/kernel/module"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	dsn := t.TempDir() + "/test.db?_pragma=foreign_keys(1)"
	d, err := db.Open("sqlite", dsn, CoreSource())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	if err := EnsureSchema(d); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestEnsureSchemaIdempotent(t *testing.T) {
	d := openTestDB(t)
	if err := EnsureSchema(d); err != nil {
		t.Fatalf("重复 Ensure 失败: %v", err)
	}
	missing, err := db.MissingTables(d, coreTables)
	if err != nil {
		t.Fatal(err)
	}
	if len(missing) != 0 {
		t.Fatalf("仍缺表: %v", missing)
	}
	items, err := DiffSchema(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.Auto {
			t.Fatalf("结构应已对齐，仍有自动差异: %+v", it)
		}
	}
}

func newSystem(t *testing.T) (*System, *db.DB) {
	t.Helper()
	d := openTestDB(t)
	cat := httpapi.NewCatalog()
	sys := NewSystem(d, cat, func() string { return "" })
	if err := sys.Bootstrap(context.Background()); err != nil {
		t.Fatal(err)
	}
	return sys, d
}

func TestSetupAndLogin(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	u, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	if u.ID == 0 {
		t.Fatal("setup 未返回用户 ID")
	}
	if _, err := sys.Setup(ctx, "admin2", "管理员2", "password123", "127.0.0.1", "ua", "rid", true); err == nil {
		t.Fatal("重复初始化应失败")
	}
	raw, loginUser, perms, err := sys.Login(ctx, "admin", "password123", "127.0.0.1", "ua", "rid")
	if err != nil {
		t.Fatal(err)
	}
	if raw == "" || loginUser.ID != u.ID {
		t.Fatal("登录返回异常")
	}
	if len(perms) != 1 || perms[0] != "*" {
		t.Fatalf("超级管理员权限应为通配: %v", perms)
	}
	p, err := sys.Authenticate(ctx, raw)
	if err != nil || p.User.ID != u.ID {
		t.Fatalf("会话认证失败: %v", err)
	}
	if _, _, _, err := sys.Login(ctx, "admin", "wrong", "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("错误密码应失败")
	}
}

func TestConcurrentSetupOnlyOneWins(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	var mu sync.Mutex
	success := 0
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := sys.Setup(ctx, "admin"+string(rune('a'+i)), "管理员", "password123", "127.0.0.1", "ua", "rid", true)
			if err == nil {
				mu.Lock()
				success++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if success != 1 {
		t.Fatalf("并发初始化成功数 = %d, want 1", success)
	}
}

func TestLastSuperAdminProtected(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	u, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	u.Roles, _ = sys.store.GetUserRoles(ctx, u.ID)
	p := &Principal{User: u, Perms: []string{"*"}}
	err = sys.UpdateUserStatus(ctx, p, u.ID, "disabled", "127.0.0.1", "ua", "rid")
	if err == nil {
		t.Fatal("禁用最后一个超级管理员应失败")
	}
}

// ---- HTTP 集成 ----

type testReg struct {
	router  *httpapi.Router
	catalog *httpapi.Catalog
}

func (r testReg) Handle(p string, h http.Handler) error { return r.router.Handle(p, h) }
func (r testReg) RegisterPage(p module.PageMeta) error {
	return r.catalog.AddPage(httpapi.Page{ID: p.ID, Title: p.Title, Path: p.Path, Menu: p.Menu, Order: p.Order, Perm: p.Perm})
}
func (r testReg) RegisterPermission(p module.PermissionMeta) error {
	return r.catalog.AddPermission(httpapi.Permission{Code: p.Code, Name: p.Name})
}

func newHTTP(t *testing.T) (http.Handler, *Module) {
	t.Helper()
	d := openTestDB(t)
	router := httpapi.NewRouter()
	catalog := httpapi.NewCatalog()
	m := New(d, catalog)
	if err := m.Register(testReg{router, catalog}); err != nil {
		t.Fatal(err)
	}
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	h := httpapi.NewHandler(httpapi.HandlerOptions{
		Info:      httpapi.AppInfo{Name: "demo", Version: "test"},
		Readiness: readyTrue{},
		Assets:    fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("index")}},
		Logger:    logx.New("error"),
		Router:    router,
		Catalog:   catalog,
		Features:  map[string]bool{"auth": true},
		Meta:      m,
	})
	return h, m
}

type readyTrue struct{}

func (readyTrue) Ready() bool { return true }

func do(t *testing.T, h http.Handler, method, path, body string, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeBody(t *testing.T, r io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(r).Decode(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestHTTPAuthFlow(t *testing.T) {
	h, _ := newHTTP(t)

	// 未登录访问受保护接口 → 401
	rec := do(t, h, http.MethodGet, "/api/system/users", "", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("未登录应为 401，实际 %d", rec.Code)
	}

	// 初始化 → 自动登录并下发 Cookie
	rec = do(t, h, http.MethodPost, "/api/auth/setup", `{"username":"admin","display_name":"管理员","password":"password123"}`, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("setup 失败: %d %s", rec.Code, rec.Body.String())
	}
	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "ea_session" {
			session = c
		}
	}
	if session == nil {
		t.Fatal("setup 未下发会话 Cookie")
	}

	// meta 应显示已初始化、auth=true、用户与 CSRF
	rec = do(t, h, http.MethodGet, "/api/meta", "", session, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("meta = %d", rec.Code)
	}
	data := decodeBody(t, rec.Body)["data"].(map[string]any)
	if data["initialized"] != true || data["features"].(map[string]any)["auth"] != true {
		t.Fatalf("meta 异常: %v", data)
	}
	if data["user"] == nil {
		t.Fatal("meta.user 为空")
	}
	csrf, _ := data["csrf_token"].(string)
	if csrf == "" {
		t.Fatal("meta.csrf_token 为空")
	}

	// 有会话但无 CSRF 的写操作 → 403
	rec = do(t, h, http.MethodPost, "/api/system/users", `{"username":"u2","display_name":"用户2","password":"password123"}`, session, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("缺 CSRF 应为 403，实际 %d", rec.Code)
	}

	// 带 CSRF 创建用户 → 200
	rec = do(t, h, http.MethodPost, "/api/system/users", `{"username":"u2","display_name":"用户2","password":"password123","roles":[]}`, session, csrf)
	if rec.Code != http.StatusOK {
		t.Fatalf("创建用户失败: %d %s", rec.Code, rec.Body.String())
	}

	// 字段校验失败 → 422
	rec = do(t, h, http.MethodPost, "/api/system/users", `{"username":"","display_name":"","password":"x"}`, session, csrf)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("字段校验应为 422，实际 %d", rec.Code)
	}

	// 重复用户名 → 409
	rec = do(t, h, http.MethodPost, "/api/system/users", `{"username":"u2","display_name":"用户2","password":"password123"}`, session, csrf)
	if rec.Code != http.StatusConflict {
		t.Fatalf("重复用户名应为 409，实际 %d", rec.Code)
	}

	// 未登录登录接口错误 → 401
	rec = do(t, h, http.MethodPost, "/api/auth/login", `{"username":"admin","password":"bad"}`, nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("错误密码应为 401，实际 %d", rec.Code)
	}

	// 登出后会话失效
	rec = do(t, h, http.MethodPost, "/api/auth/logout", `{}`, session, csrf)
	if rec.Code != http.StatusOK {
		t.Fatalf("登出失败: %d", rec.Code)
	}
	rec = do(t, h, http.MethodGet, "/api/auth/me", "", session, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("登出后 me 应为 401，实际 %d", rec.Code)
	}
}

func TestUpdateUserCannotDropLastSuperAdmin(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	u, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	u.Roles, _ = sys.store.GetUserRoles(ctx, u.ID)
	p := &Principal{User: u, Perms: []string{"*"}}
	if err := sys.UpdateUser(ctx, p, u.ID, "管理员", []string{}, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("移除最后一个超级管理员角色应失败")
	}
}

func TestRolePermissionAssignment(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	admin, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	admin.Roles, _ = sys.store.GetUserRoles(ctx, admin.ID)
	actor := &Principal{User: admin, Perms: []string{"*"}}
	role, err := sys.CreateRole(ctx, actor, "user_admin", "用户管理员", "", []string{"system.user.read", "system.user.write"}, "127.0.0.1", "ua", "rid")
	if err != nil {
		t.Fatal(err)
	}
	got, err := sys.GetRole(ctx, role.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Permissions) != 2 {
		t.Fatalf("角色权限数量错误: %v", got.Permissions)
	}
	u, err := sys.CreateUser(ctx, actor, "ops", "运维", "password123", []string{"user_admin"}, "127.0.0.1", "ua", "rid")
	if err != nil {
		t.Fatal(err)
	}
	perms, err := sys.store.GetUserPermissions(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 2 {
		t.Fatalf("用户权限数量错误: %v", perms)
	}
	next := []string{"system.user.read"}
	if _, err := sys.UpdateRole(ctx, actor, role.ID, "用户管理员", "", &next, "127.0.0.1", "ua", "rid"); err != nil {
		t.Fatal(err)
	}
	perms, err = sys.store.GetUserPermissions(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(perms) != 1 || perms[0] != "system.user.read" {
		t.Fatalf("权限变更未生效: %v", perms)
	}
	// 系统角色权限不可修改
	sysRole, err := sys.store.GetRoleByCode(ctx, SuperAdminCode)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sys.UpdateRole(ctx, actor, sysRole.ID, "超级管理员", "", &next, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("系统角色权限应不可修改")
	}
	// 保留编码不可重复创建
	if _, err := sys.CreateRole(ctx, actor, SuperAdminCode, "重复", "", nil, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("保留角色编码应冲突")
	}
}

func TestOrphanedPermissionMarked(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	if err := sys.store.UpsertPermission(ctx, "legacy.perm", "旧权限", "legacy"); err != nil {
		t.Fatal(err)
	}
	var codes []string
	for _, p := range permissionDefs() {
		codes = append(codes, p.Code)
	}
	if err := sys.MarkOrphanedPermissions(ctx, codes); err != nil {
		t.Fatal(err)
	}
	ps, err := sys.store.ListPermissions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range ps {
		if p.Code == "legacy.perm" {
			found = true
			if p.Status != "orphaned" {
				t.Fatalf("旧权限应为 orphaned，实际 %s", p.Status)
			}
		}
	}
	if !found {
		t.Fatal("旧权限记录不应被删除")
	}
}

type fakeManifests []module.Manifest

func (f fakeManifests) Manifests() []module.Manifest { return f }

func TestManifestPermissionsSyncedByModule(t *testing.T) {
	d := openTestDB(t)
	cat := httpapi.NewCatalog()
	m := New(d, cat)
	m.SetManifestSource(fakeManifests{
		{ID: "console", Permissions: []module.PermissionMeta{{Code: "console.read", Name: "查看仪表盘"}}},
		m.Manifest(),
	})
	if err := m.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ps, err := m.sys.store.ListPermissions(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	active := map[string]bool{}
	for _, p := range ps {
		if p.Status == "active" {
			active[p.Code] = true
		}
	}
	for _, code := range []string{"console.read", "system.user.read", "system.config.read", "system.audit.read"} {
		if !active[code] {
			t.Fatalf("权限 %s 未同步，现有 %v", code, active)
		}
	}
	if _, ok := active["system.config.write"]; ok {
		t.Fatal("P03 不应声明 system.config.write")
	}
}

func TestNonSuperCannotEscalate(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	admin, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	superActor := &Principal{User: admin, Perms: []string{"*"}}
	if _, err := sys.CreateRole(ctx, superActor, "user_admin", "用户管理员", "", []string{"system.user.read", "system.user.write"}, "127.0.0.1", "ua", "rid"); err != nil {
		t.Fatal(err)
	}
	ops, err := sys.CreateUser(ctx, superActor, "ops", "运维", "password123", []string{"user_admin"}, "127.0.0.1", "ua", "rid")
	if err != nil {
		t.Fatal(err)
	}
	perms, err := sys.store.GetUserPermissions(ctx, ops.ID)
	if err != nil {
		t.Fatal(err)
	}
	opsActor := &Principal{User: ops, Perms: perms}
	if _, err := sys.CreateUser(ctx, opsActor, "evil", "越权", "password123", []string{SuperAdminCode}, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("普通管理员不应能创建超级管理员")
	}
	if err := sys.UpdateUser(ctx, opsActor, ops.ID, "运维", []string{"user_admin", SuperAdminCode}, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("普通管理员不应能授予超级管理员角色")
	}
	if err := sys.UpdateUser(ctx, opsActor, admin.ID, "管理员", []string{"user_admin"}, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("普通管理员不应能修改超级管理员")
	}
	if err := sys.UpdateUserStatus(ctx, opsActor, admin.ID, "disabled", "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("普通管理员不应能禁用超级管理员")
	}
	if err := sys.ResetPassword(ctx, opsActor, admin.ID, "password456", "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("普通管理员不应能重置超级管理员密码")
	}
}

func TestHTTPRoleDetailAndPermissionUpdate(t *testing.T) {
	h, _ := newHTTP(t)
	rec := do(t, h, http.MethodPost, "/api/auth/setup", `{"username":"admin","display_name":"管理员","password":"password123"}`, nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("setup 失败: %d %s", rec.Code, rec.Body.String())
	}
	var session *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "ea_session" {
			session = c
		}
	}
	rec = do(t, h, http.MethodGet, "/api/meta", "", session, "")
	data := decodeBody(t, rec.Body)["data"].(map[string]any)
	csrf, _ := data["csrf_token"].(string)

	rec = do(t, h, http.MethodPost, "/api/system/roles", `{"code":"auditor","name":"审计员","permissions":["system.audit.read"]}`, session, csrf)
	if rec.Code != http.StatusOK {
		t.Fatalf("创建角色失败: %d %s", rec.Code, rec.Body.String())
	}
	roleID := int64(decodeBody(t, rec.Body)["data"].(map[string]any)["id"].(float64))

	rec = do(t, h, http.MethodGet, fmt.Sprintf("/api/system/roles/%d", roleID), "", session, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("角色详情失败: %d %s", rec.Code, rec.Body.String())
	}
	detail := decodeBody(t, rec.Body)["data"].(map[string]any)
	if perms, _ := detail["permissions"].([]any); len(perms) != 1 || perms[0] != "system.audit.read" {
		t.Fatalf("角色详情权限错误: %v", detail)
	}

	rec = do(t, h, http.MethodPut, fmt.Sprintf("/api/system/roles/%d", roleID), `{"name":"审计员","permissions":["system.user.read","system.audit.read"]}`, session, csrf)
	if rec.Code != http.StatusOK {
		t.Fatalf("更新角色权限失败: %d %s", rec.Code, rec.Body.String())
	}
	detail = decodeBody(t, rec.Body)["data"].(map[string]any)
	if perms, _ := detail["permissions"].([]any); len(perms) != 2 {
		t.Fatalf("更新后权限数量错误: %v", detail)
	}

	// 未知权限 → 422
	rec = do(t, h, http.MethodPost, "/api/system/roles", `{"code":"bad","name":"坏角色","permissions":["no.such.perm"]}`, session, csrf)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("未知权限应为 422，实际 %d", rec.Code)
	}
}

func TestDeleteSystemRoleRejected(t *testing.T) {
	sys, _ := newSystem(t)
	ctx := context.Background()
	u, err := sys.Setup(ctx, "admin", "管理员", "password123", "127.0.0.1", "ua", "rid", true)
	if err != nil {
		t.Fatal(err)
	}
	role, err := sys.store.GetRoleByCode(ctx, SuperAdminCode)
	if err != nil {
		t.Fatal(err)
	}
	p := &Principal{User: u, Perms: []string{"*"}}
	if err := sys.DeleteRole(ctx, p, role.ID, "127.0.0.1", "ua", "rid"); err == nil {
		t.Fatal("系统角色不可删除")
	}
}
