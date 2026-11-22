// Package db 提供 SQL-first 数据访问、建表检查和结构差异分析。
//
// 本包只保留通用能力：连接管理、脚本读取、幂等建表、表/列/索引结构比对与安全 SQL 生成。
// 具体业务表名、业务字段和业务 SQL 由调用方通过 TableSpec/SQLSource 传入。
package db

import (
	"database/sql"
	"fmt"

	"github.com/iotames/easydb"
)

// SQLSource 按脚本文件名返回 SQL 文本，方言目录和嵌入/外挂策略由实现方决定。
type SQLSource interface {
	SQL(name string) (string, error)
}

// MapSource 是内存 SQL 脚本源，适合测试和少量内嵌脚本。
type MapSource map[string]string

// SQL 返回指定脚本。
func (m MapSource) SQL(name string) (string, error) {
	txt, ok := m[name]
	if !ok {
		return "", fmt.Errorf("db: SQL 脚本 %q 不存在", name)
	}
	return txt, nil
}

// DB 是通用数据访问对象，持有 easydb 和脚本源。
type DB struct {
	edb    *easydb.EasyDb
	source SQLSource
	driver string
}

// Open 打开数据库并绑定 SQL 脚本源。
func Open(driver, dsn string, source SQLSource) (*DB, error) {
	if driver == "" {
		return nil, fmt.Errorf("db: 驱动名不能为空")
	}
	sqldb, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: 打开驱动 %s 失败: %w", driver, err)
	}
	return &DB{
		edb:    easydb.NewEasyDbBySqlDB(sqldb),
		source: source,
		driver: driver,
	}, nil
}

// EasyDB 返回底层 easydb 对象。
func (d *DB) EasyDB() *easydb.EasyDb {
	return d.edb
}

// Driver 返回当前驱动名。
func (d *DB) Driver() string {
	return d.driver
}

// SQL 返回指定脚本。
func (d *DB) SQL(name string) (string, error) {
	if d.source == nil {
		return "", fmt.Errorf("db: 未配置 SQLSource，无法读取 %q", name)
	}
	return d.source.SQL(name)
}

// Close 关闭数据库连接。
func (d *DB) Close() error {
	if d.edb == nil || d.edb.GetSqlDB() == nil {
		return nil
	}
	return d.edb.GetSqlDB().Close()
}
