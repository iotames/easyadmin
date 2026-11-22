package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iotames/easyadmin/infra/tasks"
)

func TestPlanValidateAndNextRun(t *testing.T) {
	if err := (Plan{}).validate(); err == nil {
		t.Fatal("zero plan should fail")
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	next, err := (Plan{EveryMin: 10}).NextRun(now)
	if err != nil {
		t.Fatal(err)
	}
	if got := next.Sub(now); got != 10*time.Minute {
		t.Fatalf("want 10m, got %s", got)
	}
}

func TestSchedulerTriggersOnce(t *testing.T) {
	store := NewMemoryStore()
	center := tasks.New(1, "scheduler:demo")
	var runs int32
	s, err := New(Deps{
		Store:    store,
		Tasks:    center,
		Location: time.UTC,
		Config: func(key string) (string, bool) {
			if key == "DEMO_ENABLED" {
				return "true", true
			}
			return "1", true
		},
		Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Register(Def{
		Name:       "demo",
		Title:      "演示",
		EnabledKey: "DEMO_ENABLED",
		Plan:       Plan{EveryMin: 1},
	}, func(ctx context.Context, _ tasks.SetProgressFn) error {
		atomic.AddInt32(&runs, 1)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	// 把下次触发时间推到当前时刻，模拟到期。
	past := time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)
	if err := store.SetNext(context.Background(), "demo", past); err != nil {
		t.Fatal(err)
	}
	n, err := s.ScanOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("want 1 triggered, got %d", n)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && atomic.LoadInt32(&runs) == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if atomic.LoadInt32(&runs) != 1 {
		t.Fatalf("runner not executed, runs=%d", runs)
	}
}

// TestStopDoesNotBlockOnRunningTask 回归：调度任务仍在运行时 Stop 必须立即返回。
// 曾经 watch goroutine 只等任务终态、不响应取消，Stop 的 wg.Wait 会被长任务永久挂住。
func TestStopDoesNotBlockOnRunningTask(t *testing.T) {
	store := NewMemoryStore()
	center := tasks.New(1, "scheduler:demo")
	block := make(chan struct{})
	defer close(block)
	s, err := New(Deps{
		Store:    store,
		Tasks:    center,
		Location: time.UTC,
		Config:   func(key string) (string, bool) { return "true", true },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Register(Def{Name: "demo", Title: "演示", EnabledKey: "DEMO_ENABLED", Plan: Plan{EveryMin: 1}},
		func(ctx context.Context, _ tasks.SetProgressFn) error {
			<-block
			return nil
		}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := s.Sync(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.SetNext(ctx, "demo", time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	// 等任务真正进入 running，避免 Stop 早于提交。
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		for _, task := range center.List() {
			if !task.Status.Terminal() {
				goto running
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("调度任务未进入运行态")

running:
	done := make(chan struct{})
	go func() { s.Stop(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("任务仍在运行时 Stop 阻塞超过 3s")
	}
}
