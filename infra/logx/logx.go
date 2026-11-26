// Package logx 是应用日志的统一门面，基于标准库 log/slog 语义。
// 统一携带 request_id 等结构化字段；底层可替换为 easyserver/log 适配器。
package logx

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// Logger 包装 *slog.Logger，提供稳定门面。
type Logger struct {
	l *slog.Logger
}

// New 按级别创建默认 JSON 日志器。level 支持 debug/info/warn/error，未知值回落 info。
func New(level string) *Logger {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lv})
	return &Logger{l: slog.New(h)}
}

// NewWithHandler 用自定义 handler 构造，便于对接外部日志后端或测试。
func NewWithHandler(h slog.Handler) *Logger { return &Logger{l: slog.New(h)} }

// With 返回附加字段的派生 Logger。
func (l *Logger) With(args ...any) *Logger {
	if l == nil || l.l == nil {
		return l
	}
	return &Logger{l: l.l.With(args...)}
}

// WithRequestID 返回带 request_id 的派生 Logger。
func (l *Logger) WithRequestID(id string) *Logger { return l.With("request_id", id) }

// Debug/Info/Warn/Error 统一日志入口。
func (l *Logger) Debug(msg string, args ...any) { l.log(slog.LevelDebug, msg, args...) }
func (l *Logger) Info(msg string, args ...any)  { l.log(slog.LevelInfo, msg, args...) }
func (l *Logger) Warn(msg string, args ...any)  { l.log(slog.LevelWarn, msg, args...) }
func (l *Logger) Error(msg string, args ...any) { l.log(slog.LevelError, msg, args...) }

func (l *Logger) log(lv slog.Level, msg string, args ...any) {
	if l == nil || l.l == nil {
		return
	}
	l.l.Log(context.Background(), lv, msg, args...)
}
