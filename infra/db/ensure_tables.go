package db

import (
	"context"
	"fmt"
	"strings"
)

// EnsureTables 启动时幂等建表：逐表执行建表/建索引脚本（脚本均为 IF NOT EXISTS，幂等）。
// 目的：新增业务表在既有老库上随启动自动补齐，无需人工到「数据库」页
// 执行表结构对齐——消除升级后"业务表缺失 → 查询报数据库异常"的窗口。
// 只做建表/建索引（加法、幂等），不做列级变更；列漂移仍走「数据库」页表结构对齐人工确认。
// 单表失败不阻断其余表（收集首个错误继续）；返回首个错误（含表名），调用方记录告警。
func EnsureTables(d *DB, specs []TableSpec) error {
	if d == nil {
		return nil
	}
	var firstErr error
	fail := func(err error) {
		if firstErr == nil {
			firstErr = err
		}
	}
	for _, spec := range specs {
		for _, name := range []string{spec.CreateScript, spec.IndexScript} {
			if name == "" {
				continue
			}
			txt, err := d.SQL(name)
			if err != nil {
				fail(fmt.Errorf("db: 读取脚本 %s 失败（表 %s）: %w", name, spec.Table, err))
				continue
			}
			txt = strings.ReplaceAll(txt, "{table}", spec.Table)
			// 拆句逐条执行（SplitScriptStatements 兼容分号结尾与按行分条两种脚本约定）
			for _, stmt := range SplitScriptStatements(txt) {
				if strings.TrimSpace(stripComments(stmt)) == "" {
					continue
				}
				if _, err := d.edb.Exec(stmt); err != nil {
					fail(fmt.Errorf("db: 表 %s 执行建表/索引语句失败: %w", spec.Table, err))
				}
			}
		}
	}
	return firstErr
}

// MissingTables 检测清单中哪些业务表不存在（catalog 实查，零行 = 缺表）。
// 用于启动建表之后确认结果：仍有缺表时由装配处注入管理接口告警，
// 经 /admin/warnings 的 errors 字段通知 WebUI 弹统一 error toast（引导去表同步）。
func MissingTables(d *DB, specs []TableSpec) ([]string, error) {
	if d == nil {
		return nil, nil
	}
	var missing []string
	for _, spec := range specs {
		cols, err := d.CatalogColumns(context.Background(), spec.Table)
		if err != nil {
			return nil, fmt.Errorf("db: 检测表 %s 是否存在失败: %w", spec.Table, err)
		}
		if len(cols) == 0 {
			missing = append(missing, spec.Table)
		}
	}
	return missing, nil
}
