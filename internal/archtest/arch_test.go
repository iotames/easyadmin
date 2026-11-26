// Package archtest 用真实 import 图守护分层依赖方向，并对规则本身做正反例校验。
package archtest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const modulePath = "github.com/iotames/easyadmin"

// layerOf 返回本模块包的顶层层名；非本模块包返回空串。
func layerOf(pkg string) string {
	if !strings.HasPrefix(pkg, modulePath+"/") {
		return ""
	}
	rest := strings.TrimPrefix(pkg, modulePath+"/")
	return strings.SplitN(rest, "/", 2)[0]
}

// violation 返回 from 依赖 to 时的规则违反说明；合法依赖返回空串。
func violation(from, to string) string {
	fl, tl := layerOf(from), layerOf(to)
	if fl == "" || tl == "" {
		return ""
	}
	switch fl {
	case "kernel":
		if tl != "kernel" {
			return fmt.Sprintf("kernel 只允许依赖标准库和 kernel 内部契约，实际依赖 %s", to)
		}
	case "infra":
		switch tl {
		case "kernel", "apps", "cmd", "webui":
			return fmt.Sprintf("infra 不得依赖 %s，实际依赖 %s", tl, to)
		}
	case "apps":
		if tl == "cmd" || tl == "webui" {
			return fmt.Sprintf("apps 不得依赖 %s，实际依赖 %s", tl, to)
		}
	case "webui":
		if tl != "webui" {
			return fmt.Sprintf("webui 只允许依赖自身资源包，实际依赖 %s", to)
		}
	}
	return ""
}

type pkgEdges struct {
	from    string
	imports []string
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatalf("go env GOMOD 失败: %v", err)
	}
	mod := strings.TrimSpace(string(out))
	if mod == "" || mod == os.DevNull {
		t.Fatal("未找到当前模块的 go.mod")
	}
	return filepath.Dir(mod)
}

func listImportGraph(t *testing.T, root string) []pkgEdges {
	t.Helper()
	cmd := exec.Command("go", "list", "-e", "-f", `{{.ImportPath}}|{{join .Imports " "}}|{{join .TestImports " "}}`, "./...")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list 失败: %v", err)
	}
	var graph []pkgEdges
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			t.Fatalf("无法解析 go list 输出: %q", line)
		}
		imports := strings.Fields(parts[1] + " " + parts[2])
		graph = append(graph, pkgEdges{from: parts[0], imports: imports})
	}
	return graph
}

func TestImportBoundaries(t *testing.T) {
	root := moduleRoot(t)
	var failures []string
	for _, pkg := range listImportGraph(t, root) {
		for _, imp := range pkg.imports {
			if msg := violation(pkg.from, imp); msg != "" {
				failures = append(failures, msg)
			}
		}
	}
	if len(failures) > 0 {
		t.Fatalf("依赖方向违规:\n%s", strings.Join(failures, "\n"))
	}
}

func TestRuleCatchesViolations(t *testing.T) {
	if got := violation(modulePath+"/kernel/app", modulePath+"/infra/db"); got == "" {
		t.Fatal("kernel 依赖 infra 应被规则捕获")
	}
	if got := violation(modulePath+"/infra/db", modulePath+"/apps/system/users"); got == "" {
		t.Fatal("infra 依赖 apps 应被规则捕获")
	}
	if got := violation(modulePath+"/webui", modulePath+"/kernel/app"); got == "" {
		t.Fatal("webui 依赖 kernel 应被规则捕获")
	}
	if got := violation(modulePath+"/apps/x", modulePath+"/cmd/easyadmin"); got == "" {
		t.Fatal("apps 依赖 cmd 应被规则捕获")
	}
	if got := violation(modulePath+"/kernel/app", "context"); got != "" {
		t.Fatalf("标准库依赖不应判违规: %s", got)
	}
	if got := violation(modulePath+"/kernel/app", modulePath+"/kernel/module"); got != "" {
		t.Fatalf("kernel 内部依赖不应判违规: %s", got)
	}
}

var fetchCall = regexp.MustCompile(`(^|[^.\w])fetch\s*\(`)

// hasBareFetch 判断源码是否直接调用 fetch（页面必须经 core/api.js）。
func hasBareFetch(src string) bool {
	return fetchCall.MatchString(src)
}

func TestBareFetchRule(t *testing.T) {
	if !hasBareFetch("await fetch('/api/users')") {
		t.Fatal("裸 fetch 应被捕获")
	}
	if hasBareFetch("api.get('/api/users')") {
		t.Fatal("经 api 封装的调用不应判违规")
	}
	if hasBareFetch("const x = window.fetch") {
		t.Fatal("fetch 引用未调用不应判违规")
	}
}

func TestPagesHaveNoBareFetch(t *testing.T) {
	root := moduleRoot(t)
	pages := filepath.Join(root, "webui", "assets", "pages")
	if _, err := os.Stat(pages); err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	var failures []string
	err := filepath.WalkDir(pages, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(p) != ".js" {
			return nil
		}
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if hasBareFetch(string(src)) {
			rel, _ := filepath.Rel(root, p)
			failures = append(failures, rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(failures) > 0 {
		t.Fatalf("页面存在裸 fetch，必须改走 core/api.js:\n%s", strings.Join(failures, "\n"))
	}
}
