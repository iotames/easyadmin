// Package scheduler 提供按分钟间隔触发的通用调度器。
//
// 调度器只负责到期判断和提交任务，不执行业务逻辑；任务执行统一交给 infra/tasks。
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/iotames/easyadmin/infra/tasks"
)

// MaxEveryMin 触发间隔上限（一年），只做防呆。
const MaxEveryMin = 525600

var namePattern = regexp.MustCompile(`^[a-z][a-z_]{0,63}$`)

// Plan 是调度计划：每 N 分钟触发一次。
type Plan struct {
	EveryMin int
}

func (p Plan) validate() error {
	if p.EveryMin <= 0 {
		return fmt.Errorf("触发间隔必须是正整数分钟，实际 %d", p.EveryMin)
	}
	if p.EveryMin > MaxEveryMin {
		return fmt.Errorf("触发间隔 %d 分钟超过上限 %d", p.EveryMin, MaxEveryMin)
	}
	return nil
}

// Describe 返回人类可读的计划描述。
func (p Plan) Describe() string {
	if p.EveryMin <= 0 {
		return "计划未设置"
	}
	switch {
	case p.EveryMin%1440 == 0:
		return fmt.Sprintf("每 %d 天一次", p.EveryMin/1440)
	case p.EveryMin%60 == 0:
		return fmt.Sprintf("每 %d 小时一次", p.EveryMin/60)
	default:
		return fmt.Sprintf("每 %d 分钟一次", p.EveryMin)
	}
}

// NextRun 返回严格晚于 now 的下一次触发时刻。
func (p Plan) NextRun(now time.Time) (time.Time, error) {
	if err := p.validate(); err != nil {
		return time.Time{}, err
	}
	return now.Add(time.Duration(p.EveryMin) * time.Minute), nil
}

// ConfigLookup 按配置键取当前值。
type ConfigLookup func(key string) (string, bool)

// Def 是一条调度定义。
type Def struct {
	Name       string
	Title      string
	EnabledKey string
	PlanKey    string
	Plan       Plan
	Remark     string
}

// Entry 是调度登记行。
type Entry struct {
	Name          string
	Title         string
	ConfigKey     string
	PlanKey       string
	Plan          Plan
	PlanDesc      string
	Remark        string
	Enabled       bool
	LastStatus    string
	LastMessage   string
	LastTaskID    string
	PendingTaskID string
	LastRunAt     *time.Time
	NextRunAt     *time.Time
}

// Runner 是调度任务执行体。
type Runner func(ctx context.Context, setProgress tasks.SetProgressFn) error

// Store 是调度登记存储。
type Store interface {
	Sync(ctx context.Context, entries []Entry) error
	List(ctx context.Context) ([]Entry, error)
	MarkRunning(ctx context.Context, name, taskID string, at time.Time) error
	MarkFinished(ctx context.Context, name, status, message string, at time.Time) error
	SetNext(ctx context.Context, name string, at time.Time) error
	AdoptPending(ctx context.Context) ([]Entry, error)
}

// Deps 是调度器依赖。
type Deps struct {
	Store     Store
	Tasks     *tasks.Center
	Config    ConfigLookup
	Location  *time.Location
	Now       func() time.Time
	ScanEvery func() time.Duration
}

// Scheduler 是通用调度器。
type Scheduler struct {
	deps    Deps
	mu      sync.RWMutex
	defs    map[string]Def
	runners map[string]Runner
	pending map[string]string
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New 创建调度器。
func New(deps Deps) (*Scheduler, error) {
	if deps.Store == nil {
		return nil, errors.New("调度器缺少 Store")
	}
	if deps.Tasks == nil {
		return nil, errors.New("调度器缺少任务中心")
	}
	if deps.Location == nil {
		return nil, errors.New("调度器缺少时区")
	}
	if deps.Now == nil {
		deps.Now = time.Now
	}
	if deps.ScanEvery == nil {
		deps.ScanEvery = func() time.Duration { return 5 * time.Second }
	}
	return &Scheduler{
		deps:    deps,
		defs:    map[string]Def{},
		runners: map[string]Runner{},
		pending: map[string]string{},
	}, nil
}

// Register 登记调度定义并绑定执行体。
func (s *Scheduler) Register(def Def, runner Runner) error {
	if def.Name == "" || !namePattern.MatchString(def.Name) {
		return fmt.Errorf("调度器名 %q 不合法：只允许小写字母与下划线", def.Name)
	}
	if strings.TrimSpace(def.Title) == "" {
		return fmt.Errorf("调度器 %s 缺少标题", def.Name)
	}
	if strings.TrimSpace(def.EnabledKey) == "" {
		return fmt.Errorf("调度器 %s 缺少启用配置键", def.Name)
	}
	if def.PlanKey == "" {
		if err := def.Plan.validate(); err != nil {
			return fmt.Errorf("调度器 %s 的计划不可用: %w", def.Name, err)
		}
	}
	if runner == nil {
		return fmt.Errorf("调度器 %s 缺少执行体", def.Name)
	}
	s.mu.Lock()
	if _, ok := s.defs[def.Name]; ok {
		s.mu.Unlock()
		return fmt.Errorf("调度器 %s 重复注册", def.Name)
	}
	s.defs[def.Name] = def
	s.runners[def.Name] = runner
	s.mu.Unlock()
	s.deps.Tasks.RegisterCreators("scheduler:" + def.Name)
	return nil
}

// Start 启动扫描循环。
func (s *Scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	scanCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.started = true
	s.mu.Unlock()

	s.adoptPending(scanCtx)
	if _, err := s.ScanOnce(scanCtx); err != nil {
		cancel()
		return err
	}
	s.wg.Add(1)
	go s.loop(scanCtx)
	return nil
}

// Stop 停止扫描并等待循环退出。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	s.started = false
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

func (s *Scheduler) loop(ctx context.Context) {
	defer s.wg.Done()
	timer := time.NewTimer(s.deps.ScanEvery())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			_, _ = s.ScanOnce(ctx)
			timer.Reset(s.deps.ScanEvery())
		}
	}
}

// Sync 把已注册定义同步到 Store，并按配置回填计划和启用状态。
func (s *Scheduler) Sync(ctx context.Context) (int, error) {
	s.mu.RLock()
	defs := make([]Def, 0, len(s.defs))
	for _, def := range s.defs {
		defs = append(defs, def)
	}
	s.mu.RUnlock()

	now := s.deps.Now().In(s.deps.Location)
	entries := make([]Entry, 0, len(defs))
	for _, def := range defs {
		plan := def.Plan
		if def.PlanKey != "" && s.deps.Config != nil {
			raw, ok := s.deps.Config(def.PlanKey)
			if !ok {
				return 0, fmt.Errorf("调度器 %s 的计划键 %s 未注册", def.Name, def.PlanKey)
			}
			var minutes int
			if _, err := fmt.Sscanf(strings.TrimSpace(raw), "%d", &minutes); err != nil {
				return 0, fmt.Errorf("调度器 %s 的计划键 %s 值 %q 不是整数", def.Name, def.PlanKey, raw)
			}
			plan = Plan{EveryMin: minutes}
		}
		if err := plan.validate(); err != nil {
			return 0, fmt.Errorf("调度器 %s 的计划不可用: %w", def.Name, err)
		}
		enabled := true
		if s.deps.Config != nil {
			enabled = configBool(s.deps.Config, def.EnabledKey)
		}
		next, _ := plan.NextRun(now)
		entries = append(entries, Entry{
			Name:      def.Name,
			Title:     def.Title,
			ConfigKey: def.EnabledKey,
			PlanKey:   def.PlanKey,
			Plan:      plan,
			PlanDesc:  plan.Describe(),
			Remark:    def.Remark,
			Enabled:   enabled,
			NextRunAt: &next,
		})
	}
	if err := s.deps.Store.Sync(ctx, entries); err != nil {
		return 0, err
	}
	return len(entries), nil
}

func configBool(lookup ConfigLookup, key string) bool {
	raw, ok := lookup(key)
	if !ok {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}

// ScanOnce 执行一轮到期检查。
func (s *Scheduler) ScanOnce(ctx context.Context) (int, error) {
	entries, err := s.deps.Store.List(ctx)
	if err != nil {
		return 0, err
	}
	now := s.deps.Now().In(s.deps.Location)
	triggered := 0
	for _, e := range entries {
		if !e.Enabled || e.NextRunAt == nil || e.NextRunAt.After(now) {
			continue
		}
		if e.PendingTaskID != "" {
			if task, ok := s.deps.Tasks.Get(e.PendingTaskID); ok && !task.Status.Terminal() {
				_ = s.deps.Store.MarkFinished(ctx, e.Name, "skipped", "上轮任务仍在运行", now)
				_ = s.deps.Store.SetNext(ctx, e.Name, nextAfter(now, e.Plan))
				continue
			}
		}
		runner := s.runner(e.Name)
		if runner == nil {
			continue
		}
		title := e.Title
		if title == "" {
			title = e.Name
		}
		taskID, err := s.deps.Tasks.Submit(tasks.Spec{
			CreatedBy: "scheduler:" + e.Name,
			Title:     "[调度] " + title,
			Run: func(ctx context.Context, setProgress tasks.SetProgressFn) error {
				return runner(ctx, setProgress)
			},
		})
		if err != nil {
			_ = s.deps.Store.MarkFinished(ctx, e.Name, "skipped", err.Error(), now)
			_ = s.deps.Store.SetNext(ctx, e.Name, nextAfter(now, e.Plan))
			continue
		}
		_ = s.deps.Store.MarkRunning(ctx, e.Name, taskID, nextAfter(now, e.Plan))
		triggered++
		s.wg.Add(1)
		go s.watch(ctx, e.Name, taskID)
	}
	return triggered, nil
}

func nextAfter(now time.Time, plan Plan) time.Time {
	next, err := plan.NextRun(now)
	if err != nil {
		return now.Add(time.Minute)
	}
	return next
}

func (s *Scheduler) runner(name string) Runner {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runners[name]
}

// watch 跟踪任务终态并回写调度结果；ctx 取消（Stop）时立即退出，
// 避免长任务把 Stop 的 wg.Wait 永久挂住。任务本身不随调度器停止而取消，
// 由任务中心继续执行或随进程退出。
func (s *Scheduler) watch(ctx context.Context, name, taskID string) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		task, ok := s.deps.Tasks.Get(taskID)
		if !ok {
			_ = s.deps.Store.MarkFinished(context.Background(), name, "failed", "任务记录不存在", s.deps.Now())
			return
		}
		if task.Status.Terminal() {
			status := string(task.Status)
			_ = s.deps.Store.MarkFinished(context.Background(), name, status, task.Result, s.deps.Now())
			return
		}
	}
}

func (s *Scheduler) adoptPending(ctx context.Context) {
	entries, err := s.deps.Store.AdoptPending(ctx)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.PendingTaskID == "" {
			continue
		}
		if task, ok := s.deps.Tasks.Get(e.PendingTaskID); ok && !task.Status.Terminal() {
			s.wg.Add(1)
			go s.watch(ctx, e.Name, e.PendingTaskID)
			continue
		}
		_ = s.deps.Store.MarkFinished(ctx, e.Name, "failed", "重启时任务已不在运行", s.deps.Now())
	}
}

// MemoryStore 是内存 Store，适合测试和单进程最小运行。
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]Entry
}

// NewMemoryStore 创建内存 Store。
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{entries: map[string]Entry{}}
}

func (m *MemoryStore) Sync(_ context.Context, entries []Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range entries {
		if old, ok := m.entries[e.Name]; ok {
			e.LastStatus = old.LastStatus
			e.LastMessage = old.LastMessage
			e.LastTaskID = old.LastTaskID
			e.PendingTaskID = old.PendingTaskID
			e.LastRunAt = old.LastRunAt
			if old.NextRunAt != nil {
				e.NextRunAt = old.NextRunAt
			}
		}
		m.entries[e.Name] = e
	}
	return nil
}

func (m *MemoryStore) List(_ context.Context) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Entry, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, e)
	}
	return out, nil
}

func (m *MemoryStore) MarkRunning(_ context.Context, name, taskID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[name]
	e.PendingTaskID = taskID
	e.LastTaskID = taskID
	e.LastStatus = "running"
	e.NextRunAt = &at
	m.entries[name] = e
	return nil
}

func (m *MemoryStore) MarkFinished(_ context.Context, name, status, message string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[name]
	e.PendingTaskID = ""
	e.LastStatus = status
	e.LastMessage = message
	e.LastRunAt = &at
	m.entries[name] = e
	return nil
}

func (m *MemoryStore) SetNext(_ context.Context, name string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e := m.entries[name]
	e.NextRunAt = &at
	m.entries[name] = e
	return nil
}

func (m *MemoryStore) AdoptPending(_ context.Context) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Entry
	for _, e := range m.entries {
		if e.PendingTaskID != "" {
			out = append(out, e)
		}
	}
	return out, nil
}

// Entries 返回内存 Store 的当前快照，供测试和管理页使用。
func (m *MemoryStore) Entries() []Entry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Entry, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, e)
	}
	return out
}
