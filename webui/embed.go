//go:build !dev

// Package webui 承载管理界面资源：生产构建内嵌进二进制，-tags dev 时读磁盘热重载。
//
// 目录约定：index.html 与 assets/ 与本包平级；_test/ 同样是 assets 的平级目录，
// 且不在 embed 指令内——测试反馈环只在 dev 构建（os.DirFS）下可见，
// 正式产物里取不到，保证 /_test/* 在线上返回 404。
package webui

import (
	"embed"
	"io/fs"
)

//go:embed index.html
//go:embed assets
var content embed.FS

// FS 返回界面资源根目录：内嵌文件即包目录本身，页面与静态文件都相对它定位。
func FS() fs.FS {
	return content
}
