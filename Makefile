# EasyAdmin 常用开发命令。lib 依赖库同步见 make lib。
LIB_DIR := lib
LIB_GIT_PREFIX ?= github:iotames

.DEFAULT_GOAL := help

.PHONY: help lib build build-dev test vet race fmt

help:
	@echo "用法:"
	@echo "  make build      构建生产二进制 bin/easyadmin"
	@echo "  make build-dev  构建 dev 二进制 bin/easyadmin-dev（WebUI 读磁盘）"
	@echo "  make test       运行全部 Go 测试"
	@echo "  make vet        go vet 静态检查"
	@echo "  make race       并发检查（需支持 -race 的工具链）"
	@echo "  make fmt        gofmt 格式化仓库 Go 代码"
	@echo "  make lib        同步 lib/ 下的依赖库到上游最新代码"

lib:
	@LIB_DIR="$(LIB_DIR)" GIT_PREFIX="$(LIB_GIT_PREFIX)" bash scripts/sync-lib.sh

build:
	go build -o bin/easyadmin ./cmd/easyadmin

build-dev:
	go build -tags dev -o bin/easyadmin-dev ./cmd/easyadmin

test:
	go test ./...

vet:
	go vet ./...

race:
	go test -race ./...

fmt:
	gofmt -w cmd kernel infra internal webui