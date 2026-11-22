package conf

// Config 是 EasyAdmin 启动配置的只读快照。
// 业务模块配置通过 Register/RegisterGroup 注册后由各自持有，不堆进本结构。
type Config struct {
	ListenAddr string // 管理后台监听地址
	AdminAddr  string // 独立管理接口监听地址，空=与 ListenAddr 同端口
	ConfigFile string // 配置文件路径；空=仅环境变量与命令行
	DataDir    string // 运行数据目录：SQLite、日志、上传、缓存
	DBDriver   string // 默认数据源驱动：sqlite/mysql/postgres
	DBDSN      string // 默认数据源 DSN
	LogLevel   string // debug/info/warn/error
	LogToFile  bool   // 是否写文件
	LogFile    string // 日志文件路径
	LogMaxSize int64  // 单文件上限（MB，0=不限制）
}

const (
	defaultListenAddr = "127.0.0.1:19527"
	defaultAdminAddr  = ""
	defaultConfigFile = ""
	defaultDataDir    = "./data"
	defaultDBDriver   = "sqlite"
	defaultDBDSN      = "data/easyadmin.db"
	defaultLogLevel   = "info"
	defaultLogFile    = "data/logs/easyadmin.log"
	defaultLogMaxSize = 50
)

// DefaultDataDir 返回默认运行数据目录。
func DefaultDataDir() string { return defaultDataDir }
