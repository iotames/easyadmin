# lib 依赖库管理：同步 lib/ 下的地基库到上游最新代码。
# 同步逻辑见 scripts/sync-lib.sh。
# 用法：
#   make lib                                        # 同步 lib/ 下全部依赖库
#   make lib LIB_GIT_PREFIX=git@github.com:iotames   # 覆盖仓库前缀

LIB_DIR := lib
LIB_GIT_PREFIX ?= github:iotames

.DEFAULT_GOAL := help

.PHONY: help lib

help:
	@echo "用法:"
	@echo "  make lib    同步 $(LIB_DIR)/ 下的依赖库到上游最新代码"
	@echo "  可选变量: LIB_GIT_PREFIX=<前缀>（默认 github:iotames）"

lib:
	@LIB_DIR="$(LIB_DIR)" GIT_PREFIX="$(LIB_GIT_PREFIX)" bash scripts/sync-lib.sh