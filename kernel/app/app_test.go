package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/iotames/easyadmin/kernel/module"
)

type fakeModule struct {
	id          string
	deps        []string
	pages       []module.PageMeta
	perms       []module.PermissionMeta
	events      *[]string
	startErr    error
	stopErr     error
	declareErr  error
	registerErr error
}

func (f *fakeModule) Manifest() module.Manifest {
	return module.Manifest{ID: f.id, DependsOn: f.deps, Pages: f.pages, Permissions: f.perms}
}

func (f *fakeModule) record(name string) {
	*f.events = append(*f.events, name)
}

func (f *fakeModule) Declare(module.ConfigRegistrar) error {
	f.record("declare:" + f.id)
	return f.declareErr
}

func (f *fakeModule) Register(module.Registrar) error {
	f.record("register:" + f.id)
	return f.registerErr
}

func (f *fakeModule) Start(context.Context) error {
	f.record("start:" + f.id)
	return f.startErr
}

func (f *fakeModule) Stop(context.Context) error {
	f.record("stop:" + f.id)
	return f.stopErr
}

func newFake(events *[]string, id string, deps ...string) *fakeModule {
	return &fakeModule{id: id, deps: deps, events: events}
}

func TestNewReturnsTopologicalOrder(t *testing.T) {
	events := new([]string)
	b := newFake(events, "b", "a")
	a := newFake(events, "a")
	c := newFake(events, "c", "b")

	got, err := New(Options{Name: "demo", Modules: []module.Module{b, a, c}})
	if err != nil {
		t.Fatal(err)
	}
	if ids := strings.Join(got.ModuleIDs(), ","); ids != "a,b,c" {
		t.Fatalf("拓扑序错误: %s", ids)
	}
}

func TestNewRejectsInvalidGraph(t *testing.T) {
	tests := []struct {
		name    string
		modules []module.Module
		want    string
	}{
		{"空 ID", []module.Module{&fakeModule{}}, "ID 为空"},
		{"重复 ID", []module.Module{&fakeModule{id: "a"}, &fakeModule{id: "a"}}, "重复"},
		{"缺失依赖", []module.Module{&fakeModule{id: "a", deps: []string{"missing"}}}, "未装配"},
		{"依赖自身", []module.Module{&fakeModule{id: "a", deps: []string{"a"}}}, "依赖自身"},
		{"依赖环", []module.Module{&fakeModule{id: "a", deps: []string{"b"}}, &fakeModule{id: "b", deps: []string{"a"}}}, "环"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(Options{Name: "demo", Modules: tt.modules})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("期望包含 %q 的错误，实际: %v", tt.want, err)
			}
		})
	}
	if _, err := New(Options{}); err == nil {
		t.Fatal("空名称应失败")
	}
}

func TestLifecycleRunsInTopologicalOrder(t *testing.T) {
	events := new([]string)
	a := newFake(events, "a")
	b := newFake(events, "b", "a")
	c := newFake(events, "c", "b")

	app, err := New(Options{Name: "demo", Modules: []module.Module{c, b, a}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Declare(nil); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(nil); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !app.Ready() {
		t.Fatal("启动后 Ready 应为 true")
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "declare:a,declare:b,declare:c,register:a,register:b,register:c,start:a,start:b,start:c,stop:c,stop:b,stop:a"
	if got := strings.Join(*events, ","); got != want {
		t.Fatalf("生命周期顺序错误:\n got %s\nwant %s", got, want)
	}
}

func TestStartFailureRollsBackStartedModules(t *testing.T) {
	events := new([]string)
	a := newFake(events, "a")
	b := newFake(events, "b", "a")
	b.startErr = errors.New("boom")

	app, err := New(Options{Name: "demo", Modules: []module.Module{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	err = app.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "b") {
		t.Fatalf("应返回模块 b 的启动错误: %v", err)
	}
	if got := strings.Join(*events, ","); got != "start:a,start:b,stop:a" {
		t.Fatalf("失败回滚顺序错误: %s", got)
	}
	if app.Ready() {
		t.Fatal("启动失败后不应 Ready")
	}
}

func TestStopAggregatesErrors(t *testing.T) {
	a := newFake(new([]string), "a")
	b := newFake(new([]string), "b")
	a.stopErr = errors.New("stop-a")
	b.stopErr = errors.New("stop-b")

	app, err := New(Options{Name: "demo", Modules: []module.Module{a, b}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	err = app.Stop(context.Background())
	if err == nil || !strings.Contains(err.Error(), "stop-a") || !strings.Contains(err.Error(), "stop-b") {
		t.Fatalf("应聚合两个停止错误: %v", err)
	}
}

func TestDeclareAndRegisterPropagateErrors(t *testing.T) {
	a := newFake(new([]string), "a")
	a.declareErr = errors.New("declare-fail")
	app, err := New(Options{Name: "demo", Modules: []module.Module{a}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Declare(nil); err == nil || !strings.Contains(err.Error(), "declare-fail") {
		t.Fatalf("Declare 错误未透出: %v", err)
	}

	b := newFake(new([]string), "b")
	b.registerErr = errors.New("register-fail")
	app, err = New(Options{Name: "demo", Modules: []module.Module{b}})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.Register(nil); err == nil || !strings.Contains(err.Error(), "register-fail") {
		t.Fatalf("Register 错误未透出: %v", err)
	}
}

func TestDisabledModuleExcludedFromLifecycle(t *testing.T) {
	events := new([]string)
	a := newFake(events, "a")
	b := newFake(events, "b", "a")
	c := newFake(events, "c")

	app, err := New(Options{Name: "demo", Modules: []module.Module{a, b, c}, Disabled: []string{"b"}})
	if err != nil {
		t.Fatal(err)
	}
	if ids := strings.Join(app.ModuleIDs(), ","); ids != "a,c" {
		t.Fatalf("禁用模块过滤错误: %s", ids)
	}
	if err := app.Declare(nil); err != nil {
		t.Fatal(err)
	}
	if err := app.Register(nil); err != nil {
		t.Fatal(err)
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(*events, ","), "b") {
		t.Fatalf("被禁用模块不应参与生命周期: %v", *events)
	}
}

func TestDisabledUnknownModuleRejected(t *testing.T) {
	app, err := New(Options{Name: "demo", Modules: []module.Module{newFake(new([]string), "a")}, Disabled: []string{"x"}})
	if err == nil || app != nil {
		t.Fatalf("禁用未装配模块应失败: app=%v err=%v", app, err)
	}
	if !strings.Contains(err.Error(), "未装配") {
		t.Fatalf("错误信息应说明未装配: %v", err)
	}
}

func TestEnabledDependingOnDisabledRejected(t *testing.T) {
	a := newFake(new([]string), "a", "b")
	b := newFake(new([]string), "b")
	_, err := New(Options{Name: "demo", Modules: []module.Module{a, b}, Disabled: []string{"b"}})
	if err == nil || !strings.Contains(err.Error(), "已禁用") {
		t.Fatalf("启用模块依赖禁用模块应失败: %v", err)
	}
}

type fakeRegistrar struct {
	pages  []string
	perms  []string
	routes []string
}

func (f *fakeRegistrar) Handle(pattern string, _ http.Handler) error {
	f.routes = append(f.routes, pattern)
	return nil
}

func (f *fakeRegistrar) RegisterPage(p module.PageMeta) error {
	f.pages = append(f.pages, p.ID)
	return nil
}

func (f *fakeRegistrar) RegisterPermission(p module.PermissionMeta) error {
	f.perms = append(f.perms, p.Code)
	return nil
}

func TestRegisterRegistersManifestMetadata(t *testing.T) {
	a := newFake(new([]string), "a")
	a.pages = []module.PageMeta{{ID: "p1"}, {ID: "p2"}}
	a.perms = []module.PermissionMeta{{Code: "c1"}}
	app, err := New(Options{Name: "demo", Modules: []module.Module{a}})
	if err != nil {
		t.Fatal(err)
	}
	reg := &fakeRegistrar{}
	if err := app.Register(reg); err != nil {
		t.Fatal(err)
	}
	if strings.Join(reg.pages, ",") != "p1,p2" || strings.Join(reg.perms, ",") != "c1" {
		t.Fatalf("元信息登记错误: pages=%v perms=%v", reg.pages, reg.perms)
	}
}
