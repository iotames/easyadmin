package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/iotames/easyadmin/apps/system"
	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
)

var pageEntryPattern = regexp.MustCompile(`(?m)^\s*'([^']+)'\s*:\s*\(\)\s*=>\s*import\(`)

func repoRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD 失败: %v", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" {
		t.Fatal("未找到当前模块的 go.mod")
	}
	return filepath.Dir(mod)
}

// manifestPageIDs 复用组合根的模块清单，避免测试另写一份模块列表。
func manifestPageIDs(t *testing.T) []string {
	t.Helper()
	database, err := db.Open("sqlite", filepath.Join(t.TempDir(), "pages.db"), system.CoreSource())
	if err != nil {
		t.Fatalf("打开测试数据库失败: %v", err)
	}
	defer database.Close()

	seen := map[string]bool{}
	var ids []string
	for _, m := range builtinModules(system.New(database, httpapi.NewCatalog())) {
		for _, page := range m.Manifest().Pages {
			if page.ID == "" {
				continue
			}
			if seen[page.ID] {
				t.Fatalf("Manifest Page ID 重复: %s", page.ID)
			}
			seen[page.ID] = true
			ids = append(ids, page.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

// coreEntryIDs 从 core/pages.js 静态提取注册的 Page ID。
func coreEntryIDs(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "webui", "assets", "core", "pages.js"))
	if err != nil {
		t.Fatalf("读取 core/pages.js 失败: %v", err)
	}
	seen := map[string]bool{}
	var ids []string
	for _, match := range pageEntryPattern.FindAllStringSubmatch(string(src), -1) {
		id := match[1]
		if seen[id] {
			t.Fatalf("core/pages.js Page ID 重复: %s", id)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// diffPageIDs 返回双向差异：Manifest 缺失于 entries、entries 多出。
func diffPageIDs(manifest, entries []string) []string {
	have := map[string]bool{}
	for _, id := range entries {
		have[id] = true
	}
	want := map[string]bool{}
	var problems []string
	for _, id := range manifest {
		want[id] = true
		if !have[id] {
			problems = append(problems, "前端缺少入口: "+id)
		}
	}
	for _, id := range entries {
		if !want[id] {
			problems = append(problems, "Manifest 未声明页面: "+id)
		}
	}
	sort.Strings(problems)
	return problems
}

func TestManifestAndCorePagesMatch(t *testing.T) {
	if problems := diffPageIDs(manifestPageIDs(t), coreEntryIDs(t)); len(problems) > 0 {
		t.Fatalf("页面 ID 双向一致性失败:\n%s", strings.Join(problems, "\n"))
	}
}

func TestPageSetDiffDetectsMissingAndExtra(t *testing.T) {
	joined := strings.Join(diffPageIDs([]string{"a", "b"}, []string{"b", "c"}), "\n")
	if !strings.Contains(joined, "前端缺少入口: a") || !strings.Contains(joined, "Manifest 未声明页面: c") {
		t.Fatalf("差异检测不完整: %s", joined)
	}
}
