package archtest

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/iotames/easyadmin/apps/console"
	"github.com/iotames/easyadmin/apps/system"
	"github.com/iotames/easyadmin/kernel/module"
)

var pageEntryPattern = regexp.MustCompile(`(?m)^\s*'([^']+)'\s*:\s*\(\)\s*=>\s*import\(`)

// manifestPageIDs 返回内置模块 Manifest 声明的 Page ID（去重后排序）。
func manifestPageIDs(t *testing.T) []string {
	t.Helper()
	manifests := []module.Manifest{console.New().Manifest(), system.Module{}.Manifest()}
	seen := map[string]bool{}
	var ids []string
	for _, mf := range manifests {
		for _, page := range mf.Pages {
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
	root := moduleRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "webui", "assets", "core", "pages.js"))
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
	manifest := manifestPageIDs(t)
	entries := coreEntryIDs(t)
	if problems := diffPageIDs(manifest, entries); len(problems) > 0 {
		t.Fatalf("页面 ID 双向一致性失败:\n%s", strings.Join(problems, "\n"))
	}
}

func TestPageSetDiffDetectsMissingAndExtra(t *testing.T) {
	problems := diffPageIDs([]string{"a", "b"}, []string{"b", "c"})
	joined := strings.Join(problems, "\n")
	if !strings.Contains(joined, "前端缺少入口: a") || !strings.Contains(joined, "Manifest 未声明页面: c") {
		t.Fatalf("差异检测不完整: %v", problems)
	}
}
