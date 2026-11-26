package conf

import "context"

// 热更语义取值。
const (
	HotReloadNone    = "none"
	HotReloadHot     = "hot"
	HotReloadRestart = "restart"
)

// Meta 是配置项注册元数据。
type Meta struct {
	Group     string
	Order     int
	Secret    bool
	HotReload string
}

// ConfigItem 配置项元数据（供 WebUI 配置中心输出）。
type ConfigItem struct {
	Group      string `json:"group"`                // 配置分组
	Order      int    `json:"order"`                // 组内排序
	Key        string `json:"key"`                  // 注册名（即环境变量名）
	Title      string `json:"title"`                // 中文说明
	Type       string `json:"type"`                 // 注册类型（bool/int/string）
	Defval     string `json:"default"`              // 默认值（字符串形态）
	Current    string `json:"value"`                // 当前值；敏感项恒为空
	Source     string `json:"source"`               // default/file/environment/command_line
	Secret     bool   `json:"secret"`               // 是否敏感项
	HotReload  string `json:"hot_reload"`           // none/hot/restart
	Writable   bool   `json:"writable"`             // P03 恒为 false
	Configured bool   `json:"configured,omitempty"` // 敏感项是否已配置
	Example    string `json:"example"`              // 使用说明/示例（可能为空）
}

// Manager 配置管理器的公开接口（默认实现见 impl.go 的 confManager）
type Manager interface {
	// Current 返回当前只读配置（原子读取，无锁）
	Current() *Config
	// Watch 订阅配置变更；回调在独立 goroutine 执行
	Watch(fn func(*Config))
	// StartWatcher 启动配置文件 mtime 轮询
	StartWatcher() error
	// Shutdown 停止热更轮询，阻塞直到后台 goroutine 退出
	Shutdown(ctx context.Context) error
	// Register 挂件配置项注册（name 即环境变量名）
	Register(pval any, name, defval, title string, usage ...string) error
	// RegisterGroup 按分组注册配置项；group 为空时归入 general。
	RegisterGroup(group string, order int, pval any, name, defval, title string, usage ...string) error
	// RegisterWith 按显式元数据注册配置项；敏感级别与热更语义必填。
	RegisterWith(pval any, name, defval, title string, meta Meta, usage ...string) error
	// SyncDefaultFile 将全部已注册配置项的默认值快照同步到工作目录 default.env（开发规范下即 bin/default.env，全量覆盖）。
	SyncDefaultFile() error
	// Set 运行期按注册名全名设值并广播
	Set(name, value string) error
	// List 列出全部已注册配置项元数据（含底座与各挂件），供管理接口输出。
	List() []ConfigItem
}

// Load 从命令行/环境变量/工作目录 .env 文件加载配置
// 用法：mgr, err := conf.Load(os.Args[1:])
func Load(args []string) (Manager, error) { return defaultLoader(args) }
