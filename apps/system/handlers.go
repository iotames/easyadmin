package system

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/iotames/easyadmin/infra/apperror"
	"github.com/iotames/easyadmin/infra/auth"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/kernel/module"
)

type route struct {
	pattern string
	handler http.Handler
}

func (s *System) registerRoutes(reg module.Registrar) error {
	routes := []route{
		{"POST /api/auth/setup", http.HandlerFunc(s.handleSetup)},
		{"POST /api/auth/login", http.HandlerFunc(s.handleLogin)},
		{"POST /api/auth/logout", s.protected("", s.handleLogout)},
		{"GET /api/auth/me", s.protected("", s.handleMe)},
		{"PUT /api/auth/password", s.protected("", s.handleChangePassword)},
		{"GET /api/system/users", s.protected("system.user.read", s.handleListUsers)},
		{"POST /api/system/users", s.protected("system.user.write", s.handleCreateUser)},
		{"GET /api/system/users/{id}", s.protected("system.user.read", s.handleGetUser)},
		{"PUT /api/system/users/{id}", s.protected("system.user.write", s.handleUpdateUser)},
		{"PUT /api/system/users/{id}/status", s.protected("system.user.write", s.handleUserStatus)},
		{"PUT /api/system/users/{id}/password", s.protected("system.user.write", s.handleResetPassword)},
		{"GET /api/system/roles", s.protected("system.role.read", s.handleListRoles)},
		{"POST /api/system/roles", s.protected("system.role.write", s.handleCreateRole)},
		{"GET /api/system/roles/{id}", s.protected("system.role.read", s.handleGetRole)},
		{"PUT /api/system/roles/{id}", s.protected("system.role.write", s.handleUpdateRole)},
		{"DELETE /api/system/roles/{id}", s.protected("system.role.write", s.handleDeleteRole)},
		{"GET /api/system/permissions", s.protected("system.role.read", s.handleListPermissions)},
		{"GET /api/system/audit", s.protected("system.audit.read", s.handleListAudit)},
	}
	for _, rt := range routes {
		if err := reg.Handle(rt.pattern, rt.handler); err != nil {
			return err
		}
	}
	return nil
}

type authedHandler func(http.ResponseWriter, *http.Request, *Principal)

func unsafeMethod(m string) bool {
	return m == http.MethodPost || m == http.MethodPut || m == http.MethodPatch || m == http.MethodDelete
}

// protected 实现认证 → CSRF → 权限的顺序守卫。
func (s *System) protected(perm string, h authedHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := s.authenticateReq(r)
		if err != nil {
			httpapi.WriteAppError(w, r, err)
			return
		}
		if unsafeMethod(r.Method) {
			if !s.csrf.Validate(p.Session.ID, r.Header.Get("X-CSRF-Token")) {
				httpapi.WriteError(w, r, http.StatusForbidden, "AUTH_CSRF_INVALID", "CSRF token 缺失或错误，请刷新页面后重试")
				return
			}
		}
		if !auth.Has(p.Perms, perm) {
			httpapi.WriteError(w, r, http.StatusForbidden, "AUTH_FORBIDDEN", "没有权限执行该操作")
			return
		}
		h(w, r, p)
	})
}

func (s *System) authenticateReq(r *http.Request) (*Principal, error) {
	raw := ""
	if c, err := r.Cookie(auth.CookieName); err == nil {
		raw = c.Value
	}
	return s.Authenticate(r.Context(), raw)
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func isHTTPS(r *http.Request) bool { return r.TLS != nil }

func setSessionCookie(w http.ResponseWriter, r *http.Request, raw string) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.CookieName,
		Value:    raw,
		Path:     "/",
		HttpOnly: true,
		Secure:   isHTTPS(r),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.CookieName, Value: "", Path: "/", HttpOnly: true, MaxAge: -1, SameSite: http.SameSiteLaxMode})
}

func parseID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.BadRequest(apperror.CodeInvalidArgument, "ID 参数非法")
	}
	return id, nil
}

func userBrief(u *User) map[string]any {
	return map[string]any{"id": u.ID, "username": u.Username, "display_name": u.DisplayName, "is_super_admin": containsRole(u.Roles, SuperAdminCode)}
}

func userDetail(u *User) map[string]any {
	m := userBrief(u)
	m["status"] = u.Status
	roles := make([]map[string]any, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, map[string]any{"id": r.ID, "code": r.Code, "name": r.Name})
	}
	m["roles"] = roles
	if u.LastLoginAt != nil {
		m["last_login_at"] = u.LastLoginAt.UTC().Format(time.RFC3339)
	}
	m["created_at"] = u.CreatedAt.UTC().Format(time.RFC3339)
	return m
}

func containsRole(roles []Role, code string) bool {
	for _, r := range roles {
		if r.Code == code {
			return true
		}
	}
	return false
}

type setupReq struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (s *System) handleSetup(w http.ResponseWriter, r *http.Request) {
	var req setupReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	ip, ua, rid := clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)
	u, err := s.Setup(r.Context(), req.Username, req.DisplayName, req.Password, ip, ua, rid, IsLoopback(r.RemoteAddr))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	u.Roles, _ = s.store.GetUserRoles(r.Context(), u.ID)
	raw, _, perms, err := s.Login(r.Context(), req.Username, req.Password, ip, ua, rid)
	if err == nil {
		setSessionCookie(w, r, raw)
	}
	httpapi.WriteOK(w, r, map[string]any{"user": userBrief(u), "permissions": perms})
}

type loginReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *System) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	raw, u, perms, err := s.Login(r.Context(), req.Username, req.Password, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	setSessionCookie(w, r, raw)
	u.Roles, _ = s.store.GetUserRoles(r.Context(), u.ID)
	httpapi.WriteOK(w, r, map[string]any{"user": userBrief(u), "permissions": perms})
}

func (s *System) handleLogout(w http.ResponseWriter, r *http.Request, p *Principal) {
	if err := s.Logout(r.Context(), p, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	clearSessionCookie(w)
	httpapi.WriteOK(w, r, map[string]any{})
}

func (s *System) handleMe(w http.ResponseWriter, r *http.Request, p *Principal) {
	u, err := s.GetUser(r.Context(), p.User.ID)
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, map[string]any{"user": userDetail(u), "permissions": p.Perms})
}

type passwordReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *System) handleChangePassword(w http.ResponseWriter, r *http.Request, p *Principal) {
	var req passwordReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	if err := s.ChangePassword(r.Context(), p, req.OldPassword, req.NewPassword, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, map[string]any{})
}

func (s *System) handleListUsers(w http.ResponseWriter, r *http.Request, _ *Principal) {
	users, err := s.ListUsers(r.Context(), httpapi.Query(r, "keyword"), httpapi.Query(r, "status"))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(users))
	for _, u := range users {
		out = append(out, userDetail(u))
	}
	httpapi.WriteOK(w, r, map[string]any{"items": out})
}

func (s *System) handleGetUser(w http.ResponseWriter, r *http.Request, _ *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	u, err := s.GetUser(r.Context(), id)
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, userDetail(u))
}

type createUserReq struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
}

func (s *System) handleCreateUser(w http.ResponseWriter, r *http.Request, p *Principal) {
	var req createUserReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	u, err := s.CreateUser(r.Context(), p, req.Username, req.DisplayName, req.Password, req.Roles, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	u.Roles, _ = s.store.GetUserRoles(r.Context(), u.ID)
	httpapi.WriteOK(w, r, userDetail(u))
}

type statusReq struct {
	Status string `json:"status"`
}

func (s *System) handleUserStatus(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	var req statusReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	if err := s.UpdateUserStatus(r.Context(), p, id, req.Status, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, map[string]any{})
}

func (s *System) handleResetPassword(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	var req passwordReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	if err := s.ResetPassword(r.Context(), p, id, req.NewPassword, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, map[string]any{})
}

func roleDetail(x *Role) map[string]any {
	perms := x.Permissions
	if perms == nil {
		perms = []string{}
	}
	return map[string]any{"id": x.ID, "code": x.Code, "name": x.Name, "description": x.Description, "is_system": x.IsSystem, "permissions": perms}
}

func (s *System) handleListRoles(w http.ResponseWriter, r *http.Request, _ *Principal) {
	rs, err := s.ListRoles(r.Context())
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rs))
	for _, x := range rs {
		out = append(out, map[string]any{"id": x.ID, "code": x.Code, "name": x.Name, "description": x.Description, "is_system": x.IsSystem})
	}
	httpapi.WriteOK(w, r, map[string]any{"items": out})
}

type createRoleReq struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

func (s *System) handleCreateRole(w http.ResponseWriter, r *http.Request, p *Principal) {
	var req createRoleReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	role, err := s.CreateRole(r.Context(), p, req.Code, req.Name, req.Description, req.Permissions, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, roleDetail(role))
}

func (s *System) handleGetRole(w http.ResponseWriter, r *http.Request, _ *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	role, err := s.GetRole(r.Context(), id)
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, roleDetail(role))
}

func (s *System) handleListPermissions(w http.ResponseWriter, r *http.Request, _ *Principal) {
	ps, err := s.ListPermissions(r.Context())
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(ps))
	for _, x := range ps {
		out = append(out, map[string]any{"id": x.ID, "code": x.Code, "name": x.Name, "module": x.Module, "status": x.Status})
	}
	httpapi.WriteOK(w, r, map[string]any{"items": out})
}

func (s *System) handleListAudit(w http.ResponseWriter, r *http.Request, _ *Principal) {
	limit := 50
	offset := 0
	if v := httpapi.Query(r, "limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if v := httpapi.Query(r, "offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	rows, err := s.ListAudit(r.Context(), limit, offset)
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, map[string]any{
			"id": a.ID, "actor_name": a.ActorName, "action": a.Action, "outcome": a.Outcome,
			"target_type": a.TargetType, "target_id": a.TargetID, "summary": a.Summary,
			"ip": a.IP, "request_id": a.RequestID, "created_at": a.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	httpapi.WriteOK(w, r, map[string]any{"items": out})
}

func (s *System) meta(ctx context.Context, r *http.Request) (httpapi.MetaValues, error) {
	mv := httpapi.MetaValues{Features: map[string]bool{"auth": true}}
	if init, err := s.Initialized(ctx); err == nil {
		mv.Initialized = init
	}
	if p, err := s.authenticateReq(r); err == nil {
		mv.User = userBrief(p.User)
		mv.Permissions = p.Perms
		mv.CSRFToken = s.csrf.Token(p.Session.ID)
		mv.Menus = s.Menus(p.Perms)
	}
	return mv, nil
}

type updateUserReq struct {
	DisplayName string   `json:"display_name"`
	Roles       []string `json:"roles"`
}

func (s *System) handleUpdateUser(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	var req updateUserReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	if err := s.UpdateUser(r.Context(), p, id, req.DisplayName, req.Roles, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	u, err := s.GetUser(r.Context(), id)
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, userDetail(u))
}

type updateRoleReq struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions *[]string `json:"permissions"`
}

func (s *System) handleUpdateRole(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	var req updateRoleReq
	if err := httpapi.DecodeJSON(r, &req); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	role, err := s.UpdateRole(r.Context(), p, id, req.Name, req.Description, req.Permissions, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r))
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, roleDetail(role))
}

func (s *System) handleDeleteRole(w http.ResponseWriter, r *http.Request, p *Principal) {
	id, err := parseID(r, "id")
	if err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	if err := s.DeleteRole(r.Context(), p, id, clientIP(r), r.UserAgent(), httpapi.RequestIDFrom(r)); err != nil {
		httpapi.WriteAppError(w, r, err)
		return
	}
	httpapi.WriteOK(w, r, map[string]any{})
}
