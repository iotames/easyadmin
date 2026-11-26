// Package app 提供应用实例的最小生命周期编排：模块校验、登记、拓扑启动和逆序停机。
//
// App 不是全局容器：每个实例独立持有模块图，测试可并发创建多个实例。
package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/iotames/easyadmin/kernel/module"
)

// Options 是创建 App 的静态装配参数。
// Disabled 中的模块不参与 Declare/Register/Start；启用模块不得依赖被禁用模块。
type Options struct {
	Name     string
	Version  string
	Modules  []module.Module
	Disabled []string
}

// App 是单个应用实例的生命周期所有者。
type App struct {
	name    string
	version string
	modules []module.Module

	mu      sync.Mutex
	started bool
}

// New 只做静态校验：名称、模块 ID 唯一性、依赖存在性和依赖环；再剔除禁用模块。
// 不监听端口、不启动 goroutine。
func New(opts Options) (*App, error) {
	if opts.Name == "" {
		return nil, errors.New("app: 名称不能为空")
	}
	order, err := resolve(opts.Modules)
	if err != nil {
		return nil, err
	}
	disabled := make(map[string]bool, len(opts.Disabled))
	known := make(map[string]bool, len(order))
	for _, m := range order {
		known[m.Manifest().ID] = true
	}
	for _, id := range opts.Disabled {
		if !known[id] {
			return nil, fmt.Errorf("app: 禁用了未装配的模块 %s", id)
		}
		disabled[id] = true
	}
	enabled := make([]module.Module, 0, len(order))
	for _, m := range order {
		mf := m.Manifest()
		if disabled[mf.ID] {
			continue
		}
		for _, dep := range mf.DependsOn {
			if disabled[dep] {
				return nil, fmt.Errorf("app: 模块 %s 依赖已禁用的模块 %s", mf.ID, dep)
			}
		}
		enabled = append(enabled, m)
	}
	return &App{name: opts.Name, version: opts.Version, modules: enabled}, nil
}

// resolve 返回稳定拓扑序：同一拓扑层内保持装配输入顺序。
func resolve(mods []module.Module) ([]module.Module, error) {
	seen := make(map[string]bool, len(mods))
	for i, m := range mods {
		if m == nil {
			return nil, fmt.Errorf("app: 第 %d 个模块为 nil", i)
		}
		id := m.Manifest().ID
		if id == "" {
			return nil, fmt.Errorf("app: 第 %d 个模块 ID 为空", i)
		}
		if seen[id] {
			return nil, fmt.Errorf("app: 模块 ID 重复: %s", id)
		}
		seen[id] = true
	}
	for _, m := range mods {
		mf := m.Manifest()
		for _, dep := range mf.DependsOn {
			if dep == mf.ID {
				return nil, fmt.Errorf("app: 模块 %s 不能依赖自身", mf.ID)
			}
			if !seen[dep] {
				return nil, fmt.Errorf("app: 模块 %s 依赖未装配的模块 %s", mf.ID, dep)
			}
		}
	}

	done := make(map[string]bool, len(mods))
	order := make([]module.Module, 0, len(mods))
	for len(order) < len(mods) {
		progressed := false
		for _, m := range mods {
			id := m.Manifest().ID
			if done[id] {
				continue
			}
			ready := true
			for _, dep := range m.Manifest().DependsOn {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				done[id] = true
				order = append(order, m)
				progressed = true
			}
		}
		if !progressed {
			left := make([]string, 0, len(mods)-len(order))
			for _, m := range mods {
				if id := m.Manifest().ID; !done[id] {
					left = append(left, id)
				}
			}
			sort.Strings(left)
			return nil, fmt.Errorf("app: 模块依赖存在环: %v", left)
		}
	}
	return order, nil
}

func (a *App) Name() string    { return a.name }
func (a *App) Version() string { return a.version }

// ModuleIDs 返回启用模块的拓扑序 ID，供诊断和测试使用。
func (a *App) ModuleIDs() []string {
	ids := make([]string, 0, len(a.modules))
	for _, m := range a.modules {
		ids = append(ids, m.Manifest().ID)
	}
	return ids
}

// Declare 按拓扑序对所有启用模块声明配置元数据。
func (a *App) Declare(reg module.ConfigRegistrar) error {
	for _, m := range a.modules {
		if err := m.Declare(reg); err != nil {
			return fmt.Errorf("模块 %s Declare: %w", m.Manifest().ID, err)
		}
	}
	return nil
}

// Register 按拓扑序登记启用模块的元信息与运行资源。
// 先登记 Manifest 声明的页面/权限（D05 权威来源），再调用模块 Register 登记路由等资源。
func (a *App) Register(reg module.Registrar) error {
	for _, m := range a.modules {
		mf := m.Manifest()
		for _, p := range mf.Pages {
			if err := reg.RegisterPage(p); err != nil {
				return fmt.Errorf("模块 %s 登记页面 %s: %w", mf.ID, p.ID, err)
			}
		}
		for _, p := range mf.Permissions {
			if err := reg.RegisterPermission(p); err != nil {
				return fmt.Errorf("模块 %s 登记权限 %s: %w", mf.ID, p.Code, err)
			}
		}
		if err := m.Register(reg); err != nil {
			return fmt.Errorf("模块 %s Register: %w", mf.ID, err)
		}
	}
	return nil
}

// Start 按拓扑序启动实现了 Lifecycle 的启用模块；任一失败则逆序停止已启动模块。
func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.started {
		return errors.New("app: 已启动")
	}
	started := make([]module.Lifecycle, 0, len(a.modules))
	for _, m := range a.modules {
		lc, ok := m.(module.Lifecycle)
		if !ok {
			continue
		}
		if err := lc.Start(ctx); err != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = stopReverse(stopCtx, started)
			return fmt.Errorf("模块 %s 启动失败: %w", m.Manifest().ID, err)
		}
		started = append(started, lc)
	}
	a.started = true
	return nil
}

// Stop 按逆拓扑序停止已启动模块，聚合全部错误后返回。
func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.started {
		return nil
	}
	a.started = false
	started := make([]module.Lifecycle, 0, len(a.modules))
	for _, m := range a.modules {
		if lc, ok := m.(module.Lifecycle); ok {
			started = append(started, lc)
		}
	}
	return stopReverse(ctx, started)
}

// Ready 报告 App 是否处于已启动状态。
func (a *App) Ready() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.started
}

func stopReverse(ctx context.Context, started []module.Lifecycle) error {
	var errs []error
	for i := len(started) - 1; i >= 0; i-- {
		if err := started[i].Stop(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
