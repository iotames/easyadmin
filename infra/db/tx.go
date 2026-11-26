package db

import (
	"context"
	"database/sql"
	"fmt"
)

// WithTx 在事务中执行 fn：fn 返回错误则回滚，否则提交。事务内禁止外部网络调用。
func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	if d == nil || d.edb == nil || d.edb.GetSqlDB() == nil {
		return fmt.Errorf("db: 未打开数据库")
	}
	tx, err := d.edb.GetSqlDB().BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db: 开启事务失败: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: 提交事务失败: %w", err)
	}
	return nil
}
