// Package system 提供用户、角色、权限、会话与审计的内置实现，供 EasyAdmin 自身与业务项目复用。
package system

import "time"

// User 是系统用户（不含会话）。
type User struct {
	ID           int64
	Username     string
	DisplayName  string
	PasswordHash string
	Status       string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Roles        []Role
}

// Active 报告用户是否启用。
func (u *User) Active() bool { return u != nil && u.Status == "active" }

// Role 是角色。
type Role struct {
	ID          int64
	Code        string
	Name        string
	Description string
	IsSystem    bool
	Permissions []string
}

// Permission 是权限码。
type Permission struct {
	ID     int64
	Code   string
	Name   string
	Module string
	Status string
}

// Session 是服务端会话（ID 为 token 的 SHA-256）。
type Session struct {
	ID        string
	UserID    int64
	ExpiresAt time.Time
}
