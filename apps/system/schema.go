package system

import (
	"context"

	"github.com/iotames/easyadmin/infra/db"
	sqlassets "github.com/iotames/easyadmin/sql"
)

// coreTables 是系统核心结构清单（core scope）。
var coreTables = []db.TableSpec{
	{Table: "sys_user", CreateScript: "sqlite/core/sys_user.sql", IndexScript: "sqlite/core/sys_user_idx.sql"},
	{Table: "sys_role", CreateScript: "sqlite/core/sys_role.sql"},
	{Table: "sys_permission", CreateScript: "sqlite/core/sys_permission.sql", IndexScript: "sqlite/core/sys_permission_idx.sql"},
	{Table: "sys_role_permission", CreateScript: "sqlite/core/sys_role_permission.sql"},
	{Table: "sys_user_role", CreateScript: "sqlite/core/sys_user_role.sql"},
	{Table: "sys_session", CreateScript: "sqlite/core/sys_session.sql", IndexScript: "sqlite/core/sys_session_idx.sql"},
	{Table: "sys_audit_log", CreateScript: "sqlite/core/sys_audit_log.sql", IndexScript: "sqlite/core/sys_audit_log_idx.sql"},
}

// CoreSource 返回系统核心结构脚本源。
func CoreSource() db.SQLSource { return sqlassets.CoreSource() }

// EnsureSchema 幂等补齐核心表与索引（只做加法，不删改列）。
func EnsureSchema(d *db.DB) error { return db.EnsureTables(d, coreTables) }

// DiffSchema 只读预览结构差异，不执行任何变更。
func DiffSchema(ctx context.Context, d *db.DB) ([]db.DiffItem, error) {
	return db.DiffSchema(ctx, d, coreTables)
}
