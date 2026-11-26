package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/iotames/easyadmin/infra/db"
)

// SQLRecorder 把审计事件写入 sys_audit_log（只追加）。
type SQLRecorder struct {
	db *db.DB
}

// NewSQLRecorder 创建 SQL 审计记录器。
func NewSQLRecorder(d *db.DB) *SQLRecorder { return &SQLRecorder{db: d} }

// Record 写入一条审计记录；detail 为空时写 NULL。
func (r *SQLRecorder) Record(ctx context.Context, e Event) error {
	if r == nil || r.db == nil {
		return nil
	}
	at := e.CreatedAt
	if at.IsZero() {
		at = time.Now()
	}
	var detail any
	if e.DetailJSON != "" {
		detail = e.DetailJSON
	}
	var actor any
	if e.ActorID != nil {
		actor = *e.ActorID
	}
	_, err := r.db.EasyDB().Exec(
		`INSERT INTO sys_audit_log
(actor_id, actor_name, action, outcome, target_type, target_id, summary, detail_json, ip, user_agent, request_id, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		actor, e.ActorName, e.Action, e.Outcome, e.TargetType, e.TargetID, e.Summary, detail, e.IP, e.UserAgent, e.RequestID, at.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("audit: 写入失败: %w", err)
	}
	return nil
}
