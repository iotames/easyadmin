package db

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/iotames/easydb/dsn"
)

// DataSourceManager 管理命名数据源组（dsn.json）。
// 只负责数据源的增删、激活和持久化；连接池由调用方按选中 DSN 建立。
type DataSourceManager struct {
	path string
	conf *dsn.DsnConf
}

// NewDataSourceManager 创建数据源管理器；path 为 dsn.json 路径。
func NewDataSourceManager(path string) *DataSourceManager {
	return &DataSourceManager{path: path, conf: dsn.NewDsnConf(path)}
}

// Load 读取数据源组；文件不存在视为空组。
func (m *DataSourceManager) Load() (dsn.DsnGroup, error) {
	var g dsn.DsnGroup
	if _, err := os.Stat(m.path); errors.Is(err, os.ErrNotExist) {
		return g, nil
	}
	if err := m.conf.GetDsnGroup(&g); err != nil {
		return g, fmt.Errorf("db: 读取数据源配置 %s 失败: %w", m.path, err)
	}
	return g, nil
}

// Save 保存数据源组；目录不存在时自动创建。
func (m *DataSourceManager) Save(g dsn.DsnGroup) error {
	if err := os.MkdirAll(filepath.Dir(m.path), 0o755); err != nil {
		return fmt.Errorf("db: 创建数据源目录失败: %w", err)
	}
	if err := m.conf.SaveDsnGroup(g); err != nil {
		return fmt.Errorf("db: 保存数据源配置 %s 失败: %w", m.path, err)
	}
	return nil
}

// Add 追加数据源；driver 必须已注册，DSN 不允许重复。
func (m *DataSourceManager) Add(name, driver, dsnText string) error {
	g, err := m.Load()
	if err != nil {
		return err
	}
	if err := g.AppendNamedDsn(name, driver, dsnText); err != nil {
		return err
	}
	return m.Save(g)
}

// SetActive 按 Code 激活数据源。
func (m *DataSourceManager) SetActive(code string) error {
	g, err := m.Load()
	if err != nil {
		return err
	}
	if err := g.Active(code); err != nil {
		return err
	}
	return m.Save(g)
}
