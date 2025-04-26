# 定义变量
PROJECT_ROOT := .
BIN_DIR := bin
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)

# ANSI 颜色定义
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
RESET := \033[0m

# 默认目标：显示帮助手册
.PHONY: default
default: help

# 帮助目标
help:
	@echo "${MAGENTA}Makefile Commands${RESET}"
	@echo "======================="
	@echo "${GREEN}all${RESET}              - Build all projects (default)"
	@echo "${GREEN}build${RESET}          - Build all projects"
	@echo "${GREEN}build-linux${RESET}    - Build all projects for Linux"
	@echo "${GREEN}build-windows${RESET}  - Build all projects for Windows"
	@echo "${GREEN}build-darwin${RESET}   - Build all projects for macOS (Darwin)"
	@echo "${YELLOW}run${RESET}            - Run all projects (ginserver and wsclient)"
	@echo "${YELLOW}run-gin${RESET}        - Run the ginserver application"
	@echo "${YELLOW}run-ws${RESET}         - Run the wsclient application"
	@echo "${RED}clean${RESET}          - Remove all build artifacts"
	@echo "${MAGENTA}help${RESET}           - Show this help message"

# 自动获取 cmd/ 目录下的所有子文件夹
CMD_DIRS := $(wildcard cmd/*)
PROJECT_NAMES := $(notdir $(CMD_DIRS))

# 创建必要的目录
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# 定义编译规则
define build_rule
$(BIN_DIR)/$(1)_$(OS)_$(ARCH): $$(wildcard $(PROJECT_ROOT)/cmd/$(1)/*.go) internal/**/* | $(BIN_DIR)
	go build -o $$@ $(PROJECT_ROOT)/cmd/$(1)
endef

# 为每个项目生成规则
$(foreach project,$(PROJECT_NAMES),\
	$(eval $(call build_rule,$(project)))\
)

# 构建命令
build: $(addsuffix _$(OS)_$(ARCH),$(addprefix $(BIN_DIR)/,$(PROJECT_NAMES)))
	@echo "${GREEN}Build completed for all projects: $(PROJECT_NAMES)${RESET}"

# 运行指定项目
run: run-gin run-ws

# 运行 ginserver
run-gin: $(BIN_DIR)/ginserver_$(OS)_$(ARCH)
	@echo "${YELLOW}Starting ginserver...${RESET}"
	./$(BIN_DIR)/ginserver_$(OS)_$(ARCH)

# 运行 wsclient
run-ws: $(BIN_DIR)/wsclient_$(OS)_$(ARCH)
	@echo "${YELLOW}Starting wsclient...${RESET}"
	./$(BIN_DIR)/wsclient_$(OS)_$(ARCH)

# 清理生成的文件
clean:
	@echo "${RED}Cleaning up...${RESET}"
	rm -rf $(BIN_DIR)/*

# 跨平台构建 (可选目标)
build-linux: OS := linux
build-linux: clean $(addsuffix _linux_$(ARCH),$(addprefix $(BIN_DIR)/,$(PROJECT_NAMES)))
	@echo "${BLUE}Linux binaries built successfully.${RESET}"

build-windows: OS := windows
build-windows: clean $(addsuffix _windows_$(ARCH).exe,$(addprefix $(BIN_DIR)/,$(PROJECT_NAMES)))
	@echo "${BLUE}Windows binaries built successfully.${RESET}"

build-darwin: OS := darwin
build-darwin: clean $(addsuffix _darwin_$(ARCH),$(addprefix $(BIN_DIR)/,$(PROJECT_NAMES)))
	@echo "${BLUE}Darwin binaries built successfully.${RESET}"

# 设置默认目标为 help
.PHONY: all
all: help

# 默认目标
.DEFAULT_GOAL := help