//go:build dev

// Package webui 承载管理界面资源：开发构建（-tags dev）直接读源码目录，
// 改完页面刷新浏览器即可，不必重新编译二进制。
//
// 与生产构建的差别：dev 下整个包目录（含 _test/）都可访问，因此测试反馈环
// /_test/index.html 只在这里存在；生产 embed 指令不含 _test，线上取不到即 404。
package webui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// FS 返回界面资源根目录：开发构建下指向源码树里的 webui 目录。
// 用调用点所在文件定位，进程在 bin/ 目录运行时也能读到这里。
func FS() fs.FS {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("开发构建无法定位界面源码目录：请在本仓库内构建，或改用内嵌资源的正式构建")
	}
	dir := filepath.Dir(file)
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		panic(fmt.Sprintf("开发构建的界面资源目录 %s 不可读：请确认源码树完整: %v", dir, err))
	}
	return os.DirFS(dir)
}
