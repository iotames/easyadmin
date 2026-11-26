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
	"syscall"
	"time"

	"github.com/iotames/easyadmin/apps/console"
	"github.com/iotames/easyadmin/infra/conf"
	"github.com/iotames/easyadmin/infra/httpapi"
	"github.com/iotames/easyadmin/infra/logx"
	kernelapp "github.com/iotames/easyadmin/kernel/app"
	"github.com/iotames/easyadmin/kernel/module"
	"github.com/iotames/easyadmin/webui"
)

// version 在 P01 为开发标识；正式构建信息在发行阶段接入。
const version = "0.1.0-p01"

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatalf("easyadmin 退出: %v", err)
	}
}

func run(args []string) error {
	manager, err := conf.Load(args)
	if err != nil {
		return fmt.Errorf("加载配置: %w", err)
	}
	cfg := manager.Current()
	logger := logx.New(cfg.LogLevel)

	router := httpapi.NewRouter()
	catalog := httpapi.NewCatalog()

	runtime, err := kernelapp.New(kernelapp.Options{
		Name:    "easyadmin",
		Version: version,
		Modules: []module.Module{console.New()},
	})
	if err != nil {
		return fmt.Errorf("装配应用: %w", err)
	}
	if err := runtime.Declare(manager); err != nil {
		return fmt.Errorf("声明模块配置: %w", err)
	}
	if err := runtime.Register(registrar{router: router, catalog: catalog}); err != nil {
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
			Features:  map[string]bool{"auth": false},
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
