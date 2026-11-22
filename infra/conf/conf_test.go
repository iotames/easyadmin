package conf

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestRegisterGroup(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	mgr, err := defaultLoader([]string{})
	if err != nil {
		t.Fatal(err)
	}
	var demo string
	if err := mgr.RegisterGroup("database", 20, &demo, "EASYADMIN_TEST_GROUP", "x", "测试项"); err != nil {
		t.Fatal(err)
	}
	for _, item := range mgr.List() {
		if item.Key == "EASYADMIN_TEST_GROUP" {
			if item.Group != "database" || item.Order != 20 {
				t.Fatalf("unexpected meta: %+v", item)
			}
			return
		}
	}
	t.Fatal("registered item not found")
}

// TestBareBoolFlagSurvivesReload 覆盖两个回归点：
//  1. 裸开关 --EASYADMIN_LOG_TO_FILE 启动后为 true；
//  2. 热更重放命令行参数时保持 true（曾因按空串重放被翻成 false）；
//  3. StartWatcher 返回后立刻发生的配置修改必须被监听到（初始 mtime 曾在 goroutine 内抓取而漏掉）。
func TestBareBoolFlagSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(old) }()

	mgr, err := defaultLoader([]string{"--EASYADMIN_LOG_TO_FILE"})
	if err != nil {
		t.Fatal(err)
	}
	if !mgr.Current().LogToFile {
		t.Fatal("裸开关启动后应为 true")
	}
	if err := mgr.StartWatcher(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = mgr.Shutdown(context.Background()) }()

	// StartWatcher 返回后立即修改 .env：这次修改必须触发一次重载。
	f, err := os.OpenFile(envFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("\nEASYADMIN_LOG_LEVEL = \"debug\"\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if mgr.Current().LogLevel == "debug" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got := mgr.Current().LogLevel; got != "debug" {
		t.Fatalf("热更未生效，LogLevel=%s", got)
	}
	if !mgr.Current().LogToFile {
		t.Fatal("热更重放命令行参数后裸开关被翻成 false")
	}
}
