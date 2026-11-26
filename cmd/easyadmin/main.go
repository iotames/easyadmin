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

	"github.com/iotames/easyadmin/infra/conf"
	kernelapp "github.com/iotames/easyadmin/kernel/app"
	"github.com/iotames/easyadmin/webui"
)

// version 在 P00 为开发标识；正式构建信息在发行阶段接入。
const version = "0.0.0-p00"

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

	runtime, err := kernelapp.New(kernelapp.Options{Name: "easyadmin", Version: version})
	if err != nil {
		return fmt.Errorf("装配应用: %w", err)
	}
	// P00 无已装配模块：Declare/Register 仍执行，确保装配链贯通且可测试。
	if err := runtime.Declare(manager); err != nil {
		return fmt.Errorf("声明模块配置: %w", err)
	}
	if err := runtime.Register(noopRegistrar{}); err != nil {
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
		Handler:           newHandler(runtime, webui.FS()),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		log.Printf("EasyAdmin 已启动: http://%s", listener.Addr())
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		log.Print("收到退出信号，开始优雅停机")
	case runErr = <-serveErr:
		log.Printf("HTTP 服务异常退出: %v", runErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return errors.Join(runErr, server.Shutdown(shutdownCtx), runtime.Stop(shutdownCtx))
}

// noopRegistrar 是 P00 无模块时的空 Registrar；模块接入后由组合根提供真实实现。
type noopRegistrar struct{}

func (noopRegistrar) Handle(string, http.Handler) {}
