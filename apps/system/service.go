package system

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/iotames/easyadmin/infra/apperror"
	"github.com/iotames/easyadmin/infra/audit"
	"github.com/iotames/easyadmin/infra/auth"
	"github.com/iotames/easyadmin/infra/conf"
	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
)

// SuperAdminCode 是内置超级管理员角色码。
const SuperAdminCode = "super_admin"

const sessionTTL = 12 * time.Hour

// System 组合仓储、认证原语与审计，实现身份与授权用例。
type System struct {
	store         *Store
	catalog       *httpapi.Catalog
	csrf          *auth.CSRF
	limiter       *auth.RateLimiter
	recorder      audit.Recorder
	adminPassword func() string
	config        ConfigLister
	setupMu       sync.Mutex
}

// NewSystem 创建系统服务。
func NewSystem(d *db.DB, cat *httpapi.Catalog, adminPassword func() string) *System {
	return &System{
		store:         NewStore(d),
		catalog:       cat,
		csrf:          auth.NewCSRF(),
		limiter:       auth.NewRateLimiter(5, 5*time.Minute),
		recorder:      audit.NewSQLRecorder(d),
		adminPassword: adminPassword,
	}
}

func isSuperActor(actor *Principal) bool {
	return actor != nil && auth.Has(actor.Perms, auth.Wildcard)
}

// resolveRoles 校验角色码并返回角色 ID；超级管理员角色只能由超级管理员授予。
func (s *System) resolveRoles(ctx context.Context, actor *Principal, codes []string) ([]int64, []string, error) {
	ids := make([]int64, 0, len(codes))
	normalized := make([]string, 0, len(codes))
	seen := make(map[string]bool, len(codes))
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" || seen[code] {
			continue
		}
		r, err := s.store.GetRoleByCode(ctx, code)
		if errors.Is(err, ErrNotFound) {
			return nil, nil, apperror.Validation(map[string][]string{"roles": {"角色不存在: " + code}})
		}
		if err != nil {
			return nil, nil, apperror.Internal("读取角色失败").WithCause(err)
		}
		if code == SuperAdminCode && !isSuperActor(actor) {
			return nil, nil, apperror.Forbidden("SYSTEM_ROLE_FORBIDDEN", "只有超级管理员可以授予超级管理员角色")
		}
		seen[code] = true
		ids = append(ids, r.ID)
		normalized = append(normalized, code)
	}
	return ids, normalized, nil
}

// normalizePermissions 校验权限码存在且去重排序。
func (s *System) normalizePermissions(ctx context.Context, codes []string) ([]string, error) {
	all, err := s.store.ListPermissions(ctx)
	if err != nil {
		return nil, apperror.Internal("读取权限失败").WithCause(err)
	}
	valid := make(map[string]bool, len(all))
	for _, p := range all {
		if p.Status == "active" {
			valid[p.Code] = true
		}
	}
	out := make([]string, 0, len(codes))
	seen := make(map[string]bool, len(codes))
	var bad []string
	for _, raw := range codes {
		code := strings.TrimSpace(raw)
		if code == "" || seen[code] {
			continue
		}
		if !valid[code] {
			bad = append(bad, code)
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	if len(bad) > 0 {
		return nil, apperror.Validation(map[string][]string{"permissions": {"未知权限: " + strings.Join(bad, ", ")}})
	}
	sort.Strings(out)
	return out, nil
}

// Bootstrap 幂等准备系统角色与 Manifest 权限码。
func (s *System) Bootstrap(ctx context.Context) error {
	if _, err := s.store.EnsureRole(ctx, s.store.q(), SuperAdminCode, "超级管理员", "系统内置角色", true); err != nil {
		return err
	}
	for _, p := range permissionDefs() {
		if err := s.store.UpsertPermission(ctx, p.Code, p.Name, moduleOf(p.Code)); err != nil {
			return err
		}
	}
	return nil
}

// Initialized 报告系统是否已创建管理员。
func (s *System) Initialized(ctx context.Context) (bool, error) {
	n, err := s.store.CountUsers(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// Setup 创建首个超级管理员（单实例串行，重复调用返回 409）。
func (s *System) Setup(ctx context.Context, username, display, password, ip, ua, reqID string, loopback bool) (*User, error) {
	s.setupMu.Lock()
	defer s.setupMu.Unlock()

	if n, err := s.store.CountUsers(ctx); err != nil {
		return nil, err
	} else if n > 0 {
		return nil, apperror.Conflict("SYSTEM_ALREADY_INITIALIZED", "系统已初始化，不可重复创建管理员")
	}
	fields := map[string][]string{}
	if strings.TrimSpace(username) == "" {
		fields["username"] = []string{"用户名不能为空"}
	}
	if strings.TrimSpace(display) == "" {
		fields["display_name"] = []string{"显示名称不能为空"}
	}
	if err := auth.ValidatePasswordPolicy(password); err != nil {
		fields["password"] = []string{err.Error()}
	}
	if len(fields) > 0 {
		return nil, apperror.Validation(fields)
	}
	if !loopback {
		want := ""
		if s.adminPassword != nil {
			want = s.adminPassword()
		}
		if want == "" || password != want {
			return nil, apperror.Forbidden("AUTH_SETUP_FORBIDDEN", "非本机初始化需要服务端配置的初始口令")
		}
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, apperror.Internal("密码哈希失败").WithCause(err)
	}
	u := &User{Username: strings.TrimSpace(username), DisplayName: strings.TrimSpace(display), PasswordHash: hash, Status: "active"}
	var uid int64
	// 事务：建用户 + 建超级管理员角色 + 关联
	if err := s.txCreateSuperAdmin(ctx, u, &uid); err != nil {
		return nil, apperror.Internal("创建管理员失败").WithCause(err)
	}
	u.ID = uid
	s.record(ctx, &u.ID, u.Username, "auth.setup", audit.OutcomeSuccess, "user", u.Username, "创建初始管理员", ip, ua, reqID)
	return u, nil
}

func (s *System) txCreateSuperAdmin(ctx context.Context, u *User, uid *int64) error {
	return s.store.db.WithTx(ctx, func(tx *sql.Tx) error {
		id, err := s.store.CreateUser(ctx, tx, u)
		if err != nil {
			return err
		}
		roleID, err := s.store.EnsureRole(ctx, tx, SuperAdminCode, "超级管理员", "系统内置角色", true)
		if err != nil {
			return err
		}
		if err := s.store.SetUserRole(ctx, tx, id, roleID); err != nil {
			return err
		}
		*uid = id
		return nil
	})
}

// Login 校验账号密码并创建会话。
func (s *System) Login(ctx context.Context, username, password, ip, ua, reqID string) (string, *User, []string, error) {
	username = strings.TrimSpace(username)
	key := username + "|" + ip
	if ok, retry := s.limiter.Allow(key); !ok {
		return "", nil, nil, apperror.RateLimited("AUTH_RATE_LIMITED", "登录失败次数过多，请稍后重试").
			WithDetails(map[string]any{"retry_after_seconds": retry})
	}
	u, err := s.store.GetUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) || (err == nil && !auth.VerifyPassword(u.PasswordHash, password)) {
		s.record(ctx, nil, username, "auth.login", audit.OutcomeFailure, "user", username, "登录失败", ip, ua, reqID)
		return "", nil, nil, apperror.Unauthorized("AUTH_INVALID_CREDENTIALS", "用户名或密码错误")
	}
	if err != nil {
		return "", nil, nil, apperror.Internal("查询用户失败").WithCause(err)
	}
	if !u.Active() {
		s.record(ctx, &u.ID, u.Username, "auth.login", audit.OutcomeDenied, "user", u.Username, "账号已禁用", ip, ua, reqID)
		return "", nil, nil, apperror.Unauthorized("AUTH_INVALID_CREDENTIALS", "用户名或密码错误")
	}
	s.limiter.Reset(key)
	raw, hash, err := auth.NewSessionToken()
	if err != nil {
		return "", nil, nil, apperror.Internal("生成会话失败").WithCause(err)
	}
	if err := s.store.CreateSession(ctx, Session{ID: hash, UserID: u.ID, ExpiresAt: time.Now().Add(sessionTTL)}, ip, ua); err != nil {
		return "", nil, nil, apperror.Internal("创建会话失败").WithCause(err)
	}
	_ = s.store.UpdateLastLogin(ctx, u.ID)
	perms, err := s.store.GetUserPermissions(ctx, u.ID)
	if err != nil {
		return "", nil, nil, apperror.Internal("读取权限失败").WithCause(err)
	}
	s.record(ctx, &u.ID, u.Username, "auth.login", audit.OutcomeSuccess, "user", u.Username, "登录成功", ip, ua, reqID)
	return raw, u, perms, nil
}

// Principal 是已认证主体。
type Principal struct {
	Session *Session
	User    *User
	Perms   []string
}

// Authenticate 从 Cookie 解析会话并加载用户与权限。
func (s *System) Authenticate(ctx context.Context, cookieRaw string) (*Principal, error) {
	if cookieRaw == "" {
		return nil, apperror.Unauthorized("AUTH_REQUIRED", "请先登录")
	}
	sid := auth.HashToken(cookieRaw)
	sess, err := s.store.GetSession(ctx, sid)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.Unauthorized("AUTH_REQUIRED", "会话已失效，请重新登录")
	}
	if err != nil {
		return nil, apperror.Internal("读取会话失败").WithCause(err)
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = s.store.DeleteSession(ctx, sid)
		return nil, apperror.Unauthorized("AUTH_REQUIRED", "会话已过期，请重新登录")
	}
	u, err := s.store.GetUserByID(ctx, sess.UserID)
	if errors.Is(err, ErrNotFound) || (err == nil && !u.Active()) {
		_ = s.store.DeleteSessionsByUser(ctx, sess.UserID, "")
		return nil, apperror.Unauthorized("AUTH_REQUIRED", "会话已失效，请重新登录")
	}
	if err != nil {
		return nil, apperror.Internal("读取用户失败").WithCause(err)
	}
	perms, err := s.store.GetUserPermissions(ctx, u.ID)
	if err != nil {
		return nil, apperror.Internal("读取权限失败").WithCause(err)
	}
	u.Roles, _ = s.store.GetUserRoles(ctx, u.ID)
	return &Principal{Session: sess, User: u, Perms: perms}, nil
}

// CSRF 返回 CSRF 服务。
func (s *System) CSRF() *auth.CSRF { return s.csrf }

// SetConfig 注入配置元数据来源。
func (s *System) SetConfig(l ConfigLister) { s.config = l }

// ConfigItems 返回只读配置元数据；未注入来源时返回空列表。
func (s *System) ConfigItems() []conf.ConfigItem {
	if s.config == nil {
		return []conf.ConfigItem{}
	}
	return s.config.List()
}

// Logout 删除当前会话。
func (s *System) Logout(ctx context.Context, p *Principal, ip, ua, reqID string) error {
	if p == nil || p.Session == nil {
		return nil
	}
	_ = s.store.DeleteSession(ctx, p.Session.ID)
	s.record(ctx, &p.User.ID, p.User.Username, "auth.logout", audit.OutcomeSuccess, "session", p.User.Username, "退出登录", ip, ua, reqID)
	return nil
}

// ChangePassword 校验旧密码后更新，并撤销其他会话。
func (s *System) ChangePassword(ctx context.Context, p *Principal, oldPw, newPw, ip, ua, reqID string) error {
	if !auth.VerifyPassword(p.User.PasswordHash, oldPw) {
		s.record(ctx, &p.User.ID, p.User.Username, "auth.password", audit.OutcomeFailure, "user", p.User.Username, "旧密码错误", ip, ua, reqID)
		return apperror.Unauthorized("AUTH_INVALID_CREDENTIALS", "原密码错误")
	}
	if err := auth.ValidatePasswordPolicy(newPw); err != nil {
		return apperror.Validation(map[string][]string{"new_password": {err.Error()}})
	}
	hash, err := auth.HashPassword(newPw)
	if err != nil {
		return apperror.Internal("密码哈希失败").WithCause(err)
	}
	if err := s.store.UpdateUserPassword(ctx, p.User.ID, hash); err != nil {
		return apperror.Internal("更新密码失败").WithCause(err)
	}
	_ = s.store.DeleteSessionsByUser(ctx, p.User.ID, p.Session.ID)
	s.record(ctx, &p.User.ID, p.User.Username, "auth.password", audit.OutcomeSuccess, "user", p.User.Username, "修改密码", ip, ua, reqID)
	return nil
}

// CreateUser 创建用户并关联角色。
func (s *System) CreateUser(ctx context.Context, actor *Principal, username, display, password string, roleCodes []string, ip, ua, reqID string) (*User, error) {
	fields := map[string][]string{}
	if strings.TrimSpace(username) == "" {
		fields["username"] = []string{"用户名不能为空"}
	}
	if strings.TrimSpace(display) == "" {
		fields["display_name"] = []string{"显示名称不能为空"}
	}
	if err := auth.ValidatePasswordPolicy(password); err != nil {
		fields["password"] = []string{err.Error()}
	}
	if len(fields) > 0 {
		return nil, apperror.Validation(fields)
	}
	roleIDs, _, err := s.resolveRoles(ctx, actor, roleCodes)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, apperror.Internal("密码哈希失败").WithCause(err)
	}
	u := &User{Username: strings.TrimSpace(username), DisplayName: strings.TrimSpace(display), PasswordHash: hash, Status: "active"}
	var uid int64
	err = s.store.db.WithTx(ctx, func(tx *sql.Tx) error {
		id, err := s.store.CreateUser(ctx, tx, u)
		if err != nil {
			return err
		}
		uid = id
		for _, roleID := range roleIDs {
			if err := s.store.SetUserRole(ctx, tx, uid, roleID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperror.Conflict("SYSTEM_USER_DUPLICATE", "用户名已存在")
		}
		return nil, apperror.Internal("创建用户失败").WithCause(err)
	}
	u.ID = uid
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.user.create", audit.OutcomeSuccess, "user", u.Username, "创建用户", ip, ua, reqID)
	return u, nil
}

// UpdateUserStatus 启用/禁用用户，保护最后一个活跃超级管理员。
func (s *System) UpdateUserStatus(ctx context.Context, actor *Principal, targetID int64, status, ip, ua, reqID string) error {
	if status != "active" && status != "disabled" {
		return apperror.Validation(map[string][]string{"status": {"状态只能为 active 或 disabled"}})
	}
	target, err := s.store.GetUserByID(ctx, targetID)
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("SYSTEM_USER_NOT_FOUND", "用户不存在")
	}
	if err != nil {
		return apperror.Internal("读取用户失败").WithCause(err)
	}
	if isAdmin, err := s.store.IsSuperAdmin(ctx, targetID); err != nil {
		return apperror.Internal("读取用户角色失败").WithCause(err)
	} else if isAdmin && !isSuperActor(actor) {
		return apperror.Forbidden("SYSTEM_USER_FORBIDDEN", "只有超级管理员可以管理超级管理员账号")
	}
	if status == "disabled" {
		if isAdmin, _ := s.store.IsSuperAdmin(ctx, targetID); isAdmin {
			n, err := s.store.CountActiveSuperAdmins(ctx, targetID)
			if err != nil {
				return apperror.Internal("统计超级管理员失败").WithCause(err)
			}
			if n == 0 {
				return apperror.Conflict("SYSTEM_LAST_SUPER_ADMIN", "不能禁用最后一个活跃超级管理员")
			}
		}
	}
	if err := s.store.UpdateUserStatus(ctx, targetID, status); err != nil {
		return apperror.Internal("更新状态失败").WithCause(err)
	}
	if status == "disabled" {
		_ = s.store.DeleteSessionsByUser(ctx, targetID, "")
	}
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.user.status", audit.OutcomeSuccess, "user", target.Username, "更新用户状态为 "+status, ip, ua, reqID)
	return nil
}

// ResetPassword 管理员重置用户密码并撤销其会话。
func (s *System) ResetPassword(ctx context.Context, actor *Principal, targetID int64, newPw, ip, ua, reqID string) error {
	if err := auth.ValidatePasswordPolicy(newPw); err != nil {
		return apperror.Validation(map[string][]string{"new_password": {err.Error()}})
	}
	target, err := s.store.GetUserByID(ctx, targetID)
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("SYSTEM_USER_NOT_FOUND", "用户不存在")
	}
	if err != nil {
		return apperror.Internal("读取用户失败").WithCause(err)
	}
	if isAdmin, err := s.store.IsSuperAdmin(ctx, targetID); err != nil {
		return apperror.Internal("读取用户角色失败").WithCause(err)
	} else if isAdmin && !isSuperActor(actor) {
		return apperror.Forbidden("SYSTEM_USER_FORBIDDEN", "只有超级管理员可以管理超级管理员账号")
	}
	hash, err := auth.HashPassword(newPw)
	if err != nil {
		return apperror.Internal("密码哈希失败").WithCause(err)
	}
	if err := s.store.UpdateUserPassword(ctx, targetID, hash); err != nil {
		return apperror.Internal("更新密码失败").WithCause(err)
	}
	_ = s.store.DeleteSessionsByUser(ctx, targetID, "")
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.user.password", audit.OutcomeSuccess, "user", target.Username, "重置用户密码", ip, ua, reqID)
	return nil
}

// ListUsers 返回用户及其角色。
func (s *System) ListUsers(ctx context.Context, keyword, status string) ([]*User, error) {
	if status != "" && status != "active" && status != "disabled" {
		return nil, apperror.Validation(map[string][]string{"status": {"状态只能为 active 或 disabled"}})
	}
	users, err := s.store.ListUsers(ctx, keyword, status)
	if err != nil {
		return nil, apperror.Internal("读取用户失败").WithCause(err)
	}
	for _, u := range users {
		u.Roles, _ = s.store.GetUserRoles(ctx, u.ID)
	}
	return users, nil
}

// GetUser 返回单个用户。
func (s *System) GetUser(ctx context.Context, id int64) (*User, error) {
	u, err := s.store.GetUserByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.NotFound("SYSTEM_USER_NOT_FOUND", "用户不存在")
	}
	if err != nil {
		return nil, apperror.Internal("读取用户失败").WithCause(err)
	}
	u.Roles, _ = s.store.GetUserRoles(ctx, u.ID)
	return u, nil
}

// CreateRole 创建普通角色。
func (s *System) CreateRole(ctx context.Context, actor *Principal, code, name, desc string, perms []string, ip, ua, reqID string) (*Role, error) {
	code = strings.TrimSpace(code)
	fields := map[string][]string{}
	if code == "" {
		fields["code"] = []string{"角色编码不能为空"}
	}
	if strings.TrimSpace(name) == "" {
		fields["name"] = []string{"角色名称不能为空"}
	}
	if len(fields) > 0 {
		return nil, apperror.Validation(fields)
	}
	if _, err := s.store.GetRoleByCode(ctx, code); err == nil {
		return nil, apperror.Conflict("SYSTEM_ROLE_DUPLICATE", "角色编码已存在")
	} else if !errors.Is(err, ErrNotFound) {
		return nil, apperror.Internal("读取角色失败").WithCause(err)
	}
	permCodes, err := s.normalizePermissions(ctx, perms)
	if err != nil {
		return nil, err
	}
	var id int64
	err = s.store.db.WithTx(ctx, func(tx *sql.Tx) error {
		rid, err := s.store.CreateRole(ctx, tx, code, name, desc)
		if err != nil {
			return err
		}
		id = rid
		return s.store.ReplaceRolePermissions(ctx, tx, rid, permCodes)
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, apperror.Conflict("SYSTEM_ROLE_DUPLICATE", "角色编码已存在")
		}
		return nil, apperror.Internal("创建角色失败").WithCause(err)
	}
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.role.create", audit.OutcomeSuccess, "role", code, "创建角色", ip, ua, reqID)
	return &Role{ID: id, Code: code, Name: name, Description: desc, Permissions: permCodes}, nil
}

// GetRole 返回角色详情与权限码。
func (s *System) GetRole(ctx context.Context, id int64) (*Role, error) {
	r, err := s.store.GetRoleByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.NotFound("SYSTEM_ROLE_NOT_FOUND", "角色不存在")
	}
	if err != nil {
		return nil, apperror.Internal("读取角色失败").WithCause(err)
	}
	r.Permissions, err = s.store.GetRolePermissions(ctx, id)
	if err != nil {
		return nil, apperror.Internal("读取角色权限失败").WithCause(err)
	}
	return r, nil
}

// ListRoles 返回角色列表。
func (s *System) ListRoles(ctx context.Context) ([]Role, error) {
	rs, err := s.store.ListRoles(ctx)
	if err != nil {
		return nil, apperror.Internal("读取角色失败").WithCause(err)
	}
	return rs, nil
}

// ListPermissions 返回权限列表。
func (s *System) ListPermissions(ctx context.Context) ([]Permission, error) {
	ps, err := s.store.ListPermissions(ctx)
	if err != nil {
		return nil, apperror.Internal("读取权限失败").WithCause(err)
	}
	return ps, nil
}

// ListAudit 返回审计记录。
func (s *System) ListAudit(ctx context.Context, limit, offset int) ([]AuditRow, error) {
	rows, err := s.store.ListAudit(ctx, limit, offset)
	if err != nil {
		return nil, apperror.Internal("读取审计失败").WithCause(err)
	}
	return rows, nil
}

// Menus 按权限过滤菜单。
func (s *System) Menus(perms []string) []httpapi.MenuNode {
	return s.catalog.MenusFor(perms)
}

func (s *System) record(ctx context.Context, actorID *int64, actorName, action, outcome, targetType, targetID, summary, ip, ua, reqID string) {
	if s.recorder == nil {
		return
	}
	_ = s.recorder.Record(ctx, audit.Event{
		ActorID: actorID, ActorName: actorName, Action: action, Outcome: outcome,
		TargetType: targetType, TargetID: targetID, Summary: summary,
		IP: ip, UserAgent: ua, RequestID: reqID, CreatedAt: time.Now(),
	})
}

// IsLoopback 判断远端地址是否为本机回环。
func IsLoopback(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// MarkOrphanedPermissions 标记当前 Manifest 未声明的活跃权限为 orphaned。
func (s *System) MarkOrphanedPermissions(ctx context.Context, codes []string) error {
	if err := s.store.MarkOrphanedPermissions(ctx, codes); err != nil {
		return apperror.Internal("标记下线权限失败").WithCause(err)
	}
	return nil
}

// RegisterPermission 幂等同步单个权限码（供组合根同步全部模块 Manifest）。
func (s *System) RegisterPermission(ctx context.Context, code, name string) error {
	return s.store.UpsertPermission(ctx, code, name, moduleOf(code))
}

// UpdateUser 更新展示名与角色；不允许移除最后一个活跃超级管理员。
func (s *System) UpdateUser(ctx context.Context, actor *Principal, targetID int64, display string, roleCodes []string, ip, ua, reqID string) error {
	if strings.TrimSpace(display) == "" {
		return apperror.Validation(map[string][]string{"display_name": {"显示名称不能为空"}})
	}
	target, err := s.store.GetUserByID(ctx, targetID)
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("SYSTEM_USER_NOT_FOUND", "用户不存在")
	}
	if err != nil {
		return apperror.Internal("读取用户失败").WithCause(err)
	}
	isAdmin, err := s.store.IsSuperAdmin(ctx, targetID)
	if err != nil {
		return apperror.Internal("读取用户角色失败").WithCause(err)
	}
	if isAdmin && !isSuperActor(actor) {
		return apperror.Forbidden("SYSTEM_USER_FORBIDDEN", "只有超级管理员可以管理超级管理员账号")
	}
	roleIDs, roleNames, err := s.resolveRoles(ctx, actor, roleCodes)
	if err != nil {
		return err
	}
	keepSuper := false
	for _, code := range roleNames {
		if code == SuperAdminCode {
			keepSuper = true
		}
	}
	if isAdmin && !keepSuper {
		n, err := s.store.CountActiveSuperAdmins(ctx, targetID)
		if err != nil {
			return apperror.Internal("统计超级管理员失败").WithCause(err)
		}
		if n == 0 {
			return apperror.Conflict("SYSTEM_LAST_SUPER_ADMIN", "不能移除最后一个活跃超级管理员的角色")
		}
	}
	err = s.store.db.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.UpdateUserDisplayName(ctx, tx, targetID, strings.TrimSpace(display)); err != nil {
			return err
		}
		return s.store.ReplaceUserRoles(ctx, tx, targetID, roleIDs)
	})
	if err != nil {
		return apperror.Internal("更新用户失败").WithCause(err)
	}
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.user.update", audit.OutcomeSuccess, "user", target.Username, "更新用户信息", ip, ua, reqID)
	return nil
}

// UpdateRole 更新角色名称、描述与权限；系统内置角色权限不可改。
func (s *System) UpdateRole(ctx context.Context, actor *Principal, id int64, name, desc string, perms *[]string, ip, ua, reqID string) (*Role, error) {
	if strings.TrimSpace(name) == "" {
		return nil, apperror.Validation(map[string][]string{"name": {"角色名称不能为空"}})
	}
	r, err := s.store.GetRoleByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return nil, apperror.NotFound("SYSTEM_ROLE_NOT_FOUND", "角色不存在")
	}
	if err != nil {
		return nil, apperror.Internal("读取角色失败").WithCause(err)
	}
	var permCodes []string
	if perms != nil {
		if r.IsSystem {
			return nil, apperror.Conflict("SYSTEM_ROLE_IMMUTABLE", "系统内置角色权限不可修改")
		}
		permCodes, err = s.normalizePermissions(ctx, *perms)
		if err != nil {
			return nil, err
		}
	}
	err = s.store.db.WithTx(ctx, func(tx *sql.Tx) error {
		if err := s.store.UpdateRole(ctx, id, strings.TrimSpace(name), desc); err != nil {
			return err
		}
		if perms != nil {
			return s.store.ReplaceRolePermissions(ctx, tx, id, permCodes)
		}
		return nil
	})
	if err != nil {
		return nil, apperror.Internal("更新角色失败").WithCause(err)
	}
	r.Name, r.Description = strings.TrimSpace(name), desc
	r.Permissions, _ = s.store.GetRolePermissions(ctx, id)
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.role.update", audit.OutcomeSuccess, "role", r.Code, "更新角色", ip, ua, reqID)
	return r, nil
}

// DeleteRole 删除非系统角色；仍被用户引用时拒绝。
func (s *System) DeleteRole(ctx context.Context, actor *Principal, id int64, ip, ua, reqID string) error {
	r, err := s.store.GetRoleByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return apperror.NotFound("SYSTEM_ROLE_NOT_FOUND", "角色不存在")
	}
	if err != nil {
		return apperror.Internal("读取角色失败").WithCause(err)
	}
	if r.IsSystem {
		return apperror.Conflict("SYSTEM_ROLE_IN_USE", "系统内置角色不可删除")
	}
	n, err := s.store.CountRoleUsers(ctx, id)
	if err != nil {
		return apperror.Internal("统计角色引用失败").WithCause(err)
	}
	if n > 0 {
		return apperror.Conflict("SYSTEM_ROLE_IN_USE", "角色仍被用户引用，无法删除")
	}
	if err := s.store.DeleteRole(ctx, id); err != nil {
		return apperror.Internal("删除角色失败").WithCause(err)
	}
	s.record(ctx, &actor.User.ID, actor.User.Username, "system.role.delete", audit.OutcomeSuccess, "role", r.Code, "删除角色", ip, ua, reqID)
	return nil
}
