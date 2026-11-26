// Package sqlassets 提供内嵌的 SQL 结构脚本，按方言与脚本名映射为 db.SQLSource。
package sqlassets

import (
	"embed"
	"io/fs"
	"strings"

	"github.com/iotames/easyadmin/infra/db"
)

//go:embed sqlite/core/*.sql
var coreFS embed.FS

// CoreSource 返回核心结构脚本源，键为完整脚本名（如 sqlite/core/sys_user.sql）。
func CoreSource() db.SQLSource {
	m := db.MapSource{}
	names, err := fs.Glob(coreFS, "sqlite/core/*.sql")
	if err != nil {
		return m
	}
	for _, name := range names {
		b, err := coreFS.ReadFile(name)
		if err != nil {
			continue
		}
		m[name] = string(b)
		// infra/db 的 catalog 查询按裸文件名读取 schema_query_*.sql，额外汇入基名键。
		base := name[strings.LastIndexByte(name, '/')+1:]
		if strings.HasPrefix(base, "schema_query_") {
			m[base] = string(b)
		}
	}
	return m
}
