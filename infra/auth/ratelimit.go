package auth

import (
	"sync"
	"time"
)

type rlEntry struct {
	count int
	first time.Time
}

// RateLimiter 是登录失败限流器（单实例内存实现）。
type RateLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]*rlEntry
	now     func() time.Time
}

// NewRateLimiter 创建限流器：window 内最多 max 次失败，超过则拒绝。
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{max: max, window: window, entries: make(map[string]*rlEntry), now: time.Now}
}

// Allow 记录一次尝试并返回是否放行及建议重试秒数。
func (l *RateLimiter) Allow(key string) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	e, ok := l.entries[key]
	if !ok || now.Sub(e.first) > l.window {
		l.entries[key] = &rlEntry{count: 1, first: now}
		return true, 0
	}
	e.count++
	if e.count > l.max {
		remain := int(l.window - now.Sub(e.first))
		if remain < 1 {
			remain = 1
		}
		return false, remain
	}
	return true, 0
}

// Reset 清除某 key 的失败计数（登录成功后调用）。
func (l *RateLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}
