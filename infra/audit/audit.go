// Package audit 是审计记录的唯一定义与写入入口，与应用日志分离。
package audit

import (
	"context"
	"time"
)

// 结果枚举。
const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"
	OutcomeDenied  = "denied"
)

// Event 是一条只追加的审计记录；Detail 必须脱敏，不得含密码/令牌/DSN。
type Event struct {
	ActorID    *int64
	ActorName  string
	Action     string
	Outcome    string
	TargetType string
	TargetID   string
	Summary    string
	DetailJSON string
	IP         string
	UserAgent  string
	RequestID  string
	CreatedAt  time.Time
}

// Recorder 写入审计记录。
type Recorder interface {
	Record(ctx context.Context, e Event) error
}

// Noop 是空记录器，便于测试或未装配审计时兜底。
type Noop struct{}

// Record 实现 Recorder。
func (Noop) Record(context.Context, Event) error { return nil }
