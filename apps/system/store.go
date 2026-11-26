package system

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/iotames/easyadmin/infra/db"
)

// ErrNotFound 表示记录不存在。
var ErrNotFound = errors.New("system: 记录不存在")

// DBTX 抽象 easydb 与 *sql.Tx 的共同方法，便于事务内外复用同一套仓储。
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Store 是系统表的 SQL 仓储。
type Store struct {
	db *db.DB
}

// NewStore 创建仓储。
func NewStore(d *db.DB) *Store { return &Store{db: d} }

func (s *Store) q() DBTX { return s.db.EasyDB() }

func nowText() string                       { return time.Now().UTC().Format(time.RFC3339) }
func fmtTime(t time.Time) string            { return t.UTC().Format(time.RFC3339) }
func parseTime(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }

// CountUsers 返回用户总数。
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.q().QueryRow(`SELECT COUNT(*) FROM sys_user`).Scan(&n)
	return n, err
}

func scanUser(row interface{ Scan(...any) error }) (*User, error) {
	var u User
	var lastLogin sql.NullString
	var created, updated string
	if err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.PasswordHash, &u.Status, &lastLogin, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if lastLogin.Valid && lastLogin.String != "" {
		if t, err := parseTime(lastLogin.String); err == nil {
			u.LastLoginAt = &t
		}
	}
	if t, err := parseTime(created); err == nil {
		u.CreatedAt = t
	}
	if t, err := parseTime(updated); err == nil {
		u.UpdatedAt = t
	}
	return &u, nil
}

const userColumns = `id, username, display_name, password_hash, status, last_login_at, created_at, updated_at`

// GetUserByUsername 按用户名查用户（含密码哈希）。
func (s *Store) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	return scanUser(s.q().QueryRow(`SELECT `+userColumns+` FROM sys_user WHERE username = ?`, username))
}

// GetUserByID 按 ID 查用户。
func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	return scanUser(s.q().QueryRow(`SELECT `+userColumns+` FROM sys_user WHERE id = ?`, id))
}

// ListUsers 返回全部用户（按 ID 升序）。
func (s *Store) ListUsers(ctx context.Context, keyword, status string) ([]*User, error) {
	query := `SELECT ` + userColumns + ` FROM sys_user`
	var where []string
	var args []any
	if kw := strings.TrimSpace(keyword); kw != "" {
		where = append(where, "(username LIKE ? OR display_name LIKE ?)")
		like := "%" + kw + "%"
		args = append(args, like, like)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY id"
	rows, err := s.q().Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateUser 在 q（可从事务传入）上创建用户，返回新 ID。
func (s *Store) CreateUser(ctx context.Context, q DBTX, u *User) (int64, error) {
	now := nowText()
	res, err := q.Exec(`INSERT INTO sys_user (username, display_name, password_hash, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		u.Username, u.DisplayName, u.PasswordHash, u.Status, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateUserStatus 更新用户状态。
func (s *Store) UpdateUserStatus(ctx context.Context, id int64, status string) error {
	_, err := s.q().Exec(`UPDATE sys_user SET status = ?, updated_at = ? WHERE id = ?`, status, nowText(), id)
	return err
}

// UpdateUserPassword 更新密码哈希。
func (s *Store) UpdateUserPassword(ctx context.Context, id int64, hash string) error {
	_, err := s.q().Exec(`UPDATE sys_user SET password_hash = ?, updated_at = ? WHERE id = ?`, hash, nowText(), id)
	return err
}

// UpdateLastLogin 记录最近登录时间。
func (s *Store) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := s.q().Exec(`UPDATE sys_user SET last_login_at = ? WHERE id = ?`, nowText(), id)
	return err
}

// GetRoleByCode 按 code 查角色。
func (s *Store) GetRoleByCode(ctx context.Context, code string) (*Role, error) {
	var r Role
	var isSystem int
	err := s.q().QueryRow(`SELECT id, code, name, description, is_system FROM sys_role WHERE code = ?`, code).
		Scan(&r.ID, &r.Code, &r.Name, &r.Description, &isSystem)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.IsSystem = isSystem == 1
	return &r, nil
}

// EnsureRole 幂等创建角色。
func (s *Store) EnsureRole(ctx context.Context, q DBTX, code, name, desc string, isSystem bool) (int64, error) {
	now := nowText()
	sysVal := 0
	if isSystem {
		sysVal = 1
	}
	_, err := q.Exec(`INSERT INTO sys_role (code, name, description, is_system, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT(code) DO NOTHING`, code, name, desc, sysVal, now, now)
	if err != nil {
		return 0, err
	}
	var id int64
	err = q.QueryRow(`SELECT id FROM sys_role WHERE code = ?`, code).Scan(&id)
	return id, err
}

// CreateRole 新建角色（唯一冲突由调用方映射为 409）。
func (s *Store) CreateRole(ctx context.Context, q DBTX, code, name, desc string) (int64, error) {
	now := nowText()
	res, err := q.Exec(`INSERT INTO sys_role (code, name, description, is_system, created_at, updated_at)
VALUES (?, ?, ?, 0, ?, ?)`, code, name, desc, now, now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListRoles 返回全部角色。
func (s *Store) ListRoles(ctx context.Context) ([]Role, error) {
	rows, err := s.q().Query(`SELECT id, code, name, description, is_system FROM sys_role ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var r Role
		var isSystem int
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.Description, &isSystem); err != nil {
			return nil, err
		}
		r.IsSystem = isSystem == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetUserRole 幂等关联用户与角色。
func (s *Store) SetUserRole(ctx context.Context, q DBTX, userID, roleID int64) error {
	_, err := q.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, userID, roleID)
	return err
}

// GetUserRoles 返回用户角色。
func (s *Store) GetUserRoles(ctx context.Context, userID int64) ([]Role, error) {
	rows, err := s.q().Query(`SELECT r.id, r.code, r.name, r.description, r.is_system
FROM sys_role r JOIN sys_user_role ur ON ur.role_id = r.id WHERE ur.user_id = ? ORDER BY r.id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Role
	for rows.Next() {
		var r Role
		var isSystem int
		if err := rows.Scan(&r.ID, &r.Code, &r.Name, &r.Description, &isSystem); err != nil {
			return nil, err
		}
		r.IsSystem = isSystem == 1
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetUserPermissions 返回用户权限码；超级管理员返回 ["*"]。
func (s *Store) GetUserPermissions(ctx context.Context, userID int64) ([]string, error) {
	roles, err := s.GetUserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, r := range roles {
		if r.Code == "super_admin" {
			return []string{"*"}, nil
		}
	}
	rows, err := s.q().Query(`SELECT DISTINCT p.code FROM sys_permission p
JOIN sys_role_permission rp ON rp.permission_id = p.id
JOIN sys_user_role ur ON ur.role_id = rp.role_id
WHERE ur.user_id = ? AND p.status = 'active' ORDER BY p.code`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

// GetRolePermissions 返回角色已分配的权限码。
func (s *Store) GetRolePermissions(ctx context.Context, roleID int64) ([]string, error) {
	rows, err := s.q().Query(`SELECT p.code FROM sys_permission p
JOIN sys_role_permission rp ON rp.permission_id = p.id
WHERE rp.role_id = ? ORDER BY p.code`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		out = append(out, code)
	}
	return out, rows.Err()
}

// ReplaceRolePermissions 整体替换角色权限（须在事务内调用）。
func (s *Store) ReplaceRolePermissions(ctx context.Context, q DBTX, roleID int64, codes []string) error {
	if _, err := q.Exec(`DELETE FROM sys_role_permission WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, code := range codes {
		if _, err := q.Exec(`INSERT INTO sys_role_permission (role_id, permission_id)
SELECT ?, id FROM sys_permission WHERE code = ?`, roleID, code); err != nil {
			return err
		}
	}
	return nil
}

// UpsertPermission 幂等写入权限码。
func (s *Store) UpsertPermission(ctx context.Context, code, name, module string) error {
	_, err := s.q().Exec(`INSERT INTO sys_permission (code, name, module, status, description, created_at)
VALUES (?, ?, ?, 'active', '', ?) ON CONFLICT(code) DO UPDATE SET name = excluded.name, module = excluded.module, status = 'active'`,
		code, name, module, nowText())
	return err
}

// MarkOrphanedPermissions 把不在当前清单中的活跃权限标记为 orphaned，不删除记录。
func (s *Store) MarkOrphanedPermissions(ctx context.Context, codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	ph := strings.TrimRight(strings.Repeat("?,", len(codes)), ",")
	args := make([]any, 0, len(codes))
	for _, code := range codes {
		args = append(args, code)
	}
	_, err := s.q().Exec(`UPDATE sys_permission SET status = 'orphaned'
WHERE status = 'active' AND code NOT IN (`+ph+`)`, args...)
	return err
}

// ListPermissions 返回全部权限码。
func (s *Store) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := s.q().Query(`SELECT id, code, name, module, status FROM sys_permission ORDER BY module, code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Module, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateSession 写入会话（ID 为 token 哈希）。
func (s *Store) CreateSession(ctx context.Context, sess Session, ip, ua string) error {
	now := nowText()
	_, err := s.q().Exec(`INSERT INTO sys_session (id, user_id, ip, user_agent, expires_at, created_at, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`, sess.ID, sess.UserID, ip, ua, fmtTime(sess.ExpiresAt), now, now)
	return err
}

// GetSession 读取会话。
func (s *Store) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	var exp string
	err := s.q().QueryRow(`SELECT id, user_id, expires_at FROM sys_session WHERE id = ?`, id).Scan(&sess.ID, &sess.UserID, &exp)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	t, err := parseTime(exp)
	if err != nil {
		return nil, err
	}
	sess.ExpiresAt = t
	return &sess, nil
}

// DeleteSession 删除单个会话。
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.q().Exec(`DELETE FROM sys_session WHERE id = ?`, id)
	return err
}

// DeleteSessionsByUser 删除用户全部会话；except 非空时保留该会话。
func (s *Store) DeleteSessionsByUser(ctx context.Context, userID int64, except string) error {
	if except == "" {
		_, err := s.q().Exec(`DELETE FROM sys_session WHERE user_id = ?`, userID)
		return err
	}
	_, err := s.q().Exec(`DELETE FROM sys_session WHERE user_id = ? AND id <> ?`, userID, except)
	return err
}

// CountActiveSuperAdmins 统计除 excludeUser 外仍启用的超级管理员数量。
func (s *Store) CountActiveSuperAdmins(ctx context.Context, excludeUser int64) (int, error) {
	var n int
	err := s.q().QueryRow(`SELECT COUNT(DISTINCT u.id) FROM sys_user u
JOIN sys_user_role ur ON ur.user_id = u.id
JOIN sys_role r ON r.id = ur.role_id
WHERE r.code = 'super_admin' AND u.status = 'active' AND u.id <> ?`, excludeUser).Scan(&n)
	return n, err
}

// IsSuperAdmin 判断用户是否拥有 super_admin 角色。
func (s *Store) IsSuperAdmin(ctx context.Context, userID int64) (bool, error) {
	var n int
	err := s.q().QueryRow(`SELECT COUNT(*) FROM sys_user_role ur JOIN sys_role r ON r.id = ur.role_id
WHERE ur.user_id = ? AND r.code = 'super_admin'`, userID).Scan(&n)
	return n > 0, err
}

// AuditRow 是审计查询行。
type AuditRow struct {
	ID         int64
	ActorID    *int64
	ActorName  string
	Action     string
	Outcome    string
	TargetType string
	TargetID   string
	Summary    string
	IP         string
	RequestID  string
	CreatedAt  time.Time
}

// ListAudit 返回审计记录（倒序）。
func (s *Store) ListAudit(ctx context.Context, limit, offset int) ([]AuditRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.q().Query(`SELECT id, actor_id, actor_name, action, outcome, target_type, target_id, summary, ip, request_id, created_at
FROM sys_audit_log ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditRow
	for rows.Next() {
		var a AuditRow
		var actor sql.NullInt64
		var created string
		if err := rows.Scan(&a.ID, &actor, &a.ActorName, &a.Action, &a.Outcome, &a.TargetType, &a.TargetID, &a.Summary, &a.IP, &a.RequestID, &created); err != nil {
			return nil, err
		}
		if actor.Valid {
			v := actor.Int64
			a.ActorID = &v
		}
		if t, err := parseTime(created); err == nil {
			a.CreatedAt = t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func moduleOf(code string) string {
	if i := strings.IndexByte(code, '.'); i > 0 {
		return code[:i]
	}
	return "system"
}

// UpdateUserDisplayName 更新展示名。
func (s *Store) UpdateUserDisplayName(ctx context.Context, q DBTX, id int64, name string) error {
	_, err := q.Exec(`UPDATE sys_user SET display_name = ?, updated_at = ? WHERE id = ?`, name, nowText(), id)
	return err
}

// ReplaceUserRoles 在事务内整体替换用户角色。
func (s *Store) ReplaceUserRoles(ctx context.Context, q DBTX, userID int64, roleIDs []int64) error {
	if _, err := q.Exec(`DELETE FROM sys_user_role WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, rid := range roleIDs {
		if _, err := q.Exec(`INSERT INTO sys_user_role (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, userID, rid); err != nil {
			return err
		}
	}
	return nil
}

// GetRoleByID 按 ID 查角色。
func (s *Store) GetRoleByID(ctx context.Context, id int64) (*Role, error) {
	var r Role
	var isSystem int
	err := s.q().QueryRow(`SELECT id, code, name, description, is_system FROM sys_role WHERE id = ?`, id).
		Scan(&r.ID, &r.Code, &r.Name, &r.Description, &isSystem)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	r.IsSystem = isSystem == 1
	return &r, nil
}

// UpdateRole 更新角色名称与描述。
func (s *Store) UpdateRole(ctx context.Context, id int64, name, desc string) error {
	_, err := s.q().Exec(`UPDATE sys_role SET name = ?, description = ?, updated_at = ? WHERE id = ?`, name, desc, nowText(), id)
	return err
}

// DeleteRole 删除角色。
func (s *Store) DeleteRole(ctx context.Context, id int64) error {
	_, err := s.q().Exec(`DELETE FROM sys_role WHERE id = ?`, id)
	return err
}

// CountRoleUsers 统计引用某角色的用户数。
func (s *Store) CountRoleUsers(ctx context.Context, roleID int64) (int, error) {
	var n int
	err := s.q().QueryRow(`SELECT COUNT(*) FROM sys_user_role WHERE role_id = ?`, roleID).Scan(&n)
	return n, err
}
