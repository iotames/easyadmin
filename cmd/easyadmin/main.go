// EasyAdmin 唯一组合根：加载启动配置、装配 App、启动 HTTP 服务并处理优雅停机。
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/iotames/easyadmin/apps/console"
	"github.com/iotames/easyadmin/apps/system"
	"github.com/iotames/easyadmin/infra/conf"
	"github.com/iotames/easyadmin/infra/db"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/infra/logx"
	kernelapp "github.com/iotames/easyadmin/kernel/app"
	"github.com/iotames/easyadmin/kernel/module"
	"github.com/iotames/easyadmin/webui"
)

// version 在 P02 为开发标识；正式构建信息在发行阶段接入。
const version = "0.1.0-p02"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("easyadmin 退出: %v", err)
	}
}

func sqliteDSN(cfg *conf.Config) (string, error) {
	dsn := cfg.DBDSN
	if dsn == "" {
		return "", errors.New("EASYADMIN_DB_DSN 为空")
	}
	path := dsn
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return "", fmt.Errorf("创建数据目录 %s: %w", dir, err)
		}
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&_pragma=foreign_keys(1)", nil
	}
	return dsn + "?_pragma=foreign_keys(1)", nil
}

func run(args []string) error {
	manager, err := conf.Load(args)
	if err != nil {
		return fmt.Errorf("加载配置: %w", err)
	}
	cfg := manager.Current()
	logger := logx.New(cfg.LogLevel)

	dsn := cfg.DBDSN
	if cfg.DBDriver == "sqlite" {
		dsn, err = sqliteDSN(cfg)
		if err != nil {
			return fmt.Errorf("初始化 SQLite DSN: %w", err)
		}
	}
	database, err := db.Open(cfg.DBDriver, dsn, system.CoreSource())
	if err != nil {
		return fmt.Errorf("打开数据库: %w", err)
	}
	defer database.Close()
	if err := system.EnsureSchema(database); err != nil {
		return fmt.Errorf("同步核心结构: %w", err)
	}

	router := httpapi.NewRouter()
	catalog := httpapi.NewCatalog()
	sysModule := system.New(database, catalog, system.WithConfig(manager))
	modules := []module.Module{console.New(), sysModule}

	runtime, err := kernelapp.New(kernelapp.Options{Name: "easyadmin", Version: version, Modules: modules})
	if err != nil {
		return fmt.Errorf("装配应用: %w", err)
	}
	sysModule.SetManifestSource(runtime)
	if err := runtime.Declare(registrar{conf: manager}); err != nil {
		return fmt.Errorf("声明模块配置: %w", err)
	}
	if err := runtime.Register(registrar{router: router, catalog: catalog, conf: manager}); err != nil {
		return fmt.Errorf("登记模块资源: %w", err)
	}

	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.ListenAddr, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runtime.Start(ctx); err != nil {
		_ = listener.Close()
		return fmt.Errorf("启动应用: %w", err)
	}
	server := &http.Server{
		Handler: httpapi.NewHandler(httpapi.HandlerOptions{
			Info:      httpapi.AppInfo{Name: runtime.Name(), Version: runtime.Version()},
			Readiness: runtime,
			Assets:    webui.FS(),
			Logger:    logger,
			Router:    router,
			Catalog:   catalog,
			Features:  map[string]bool{"auth": true},
			Meta:      sysModule,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		logger.Info("EasyAdmin 已启动", "addr", listener.Addr().String())
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		logger.Info("收到退出信号，开始优雅停机")
	case runErr = <-serveErr:
		logger.Error("HTTP 服务异常退出", "err", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(runErr, server.Shutdown(shutdownCtx), runtime.Stop(shutdownCtx))
}
