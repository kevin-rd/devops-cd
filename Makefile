.PHONY: build-base run-base build-git-scanner test clean docker-build-base help

# 变量
APP_NAME=devops-cd
BASE_SERVICE=devops-cd-base
GIT_SCANNER=git-scanner
CONFIG_FILE?=configs/base.yaml
BUILD_DIR=build
VERSION?=1.0.0

# 默认目标
.DEFAULT_GOAL := help

## help: 显示帮助信息
help:
	@echo "可用命令:"
	@echo "  make build-base              构建 base service"
	@echo "  make run-base                运行 base service (使用默认配置)"
	@echo "  make run-base CONFIG_FILE=path  运行 base service (指定配置文件)"
	@echo "  make build-git-scanner       构建 git-scanner 工具"
	@echo "  make test                    运行测试"
	@echo "  make clean                   清理构建文件"
	@echo "  make docker-build-base       构建 base service Docker 镜像"
	@echo "  make fmt                     格式化代码"
	@echo "  make lint                    代码检查"
	@echo "  make deps                    下载依赖"

## build-base: 构建 base service
build-base:
	@echo "构建 base service..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "-X main.appVersion=$(VERSION)" -o $(BUILD_DIR)/base ./cmd/base
	@echo "构建完成: $(BUILD_DIR)/base"

## build-git-scanner: 构建 git-scanner 工具
build-git-scanner:
	@echo "构建 git-scanner..."
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags "-X main.appVersion=$(VERSION)" -o $(BUILD_DIR)/$(GIT_SCANNER) ./cmd/$(GIT_SCANNER)
	@echo "构建完成: $(BUILD_DIR)/$(GIT_SCANNER)"

## run-base: 运行 base service
run-base:
	@echo "运行 base service..."
	@if [ -n "$(CONFIG_FILE)" ]; then \
		echo "使用配置文件: $(CONFIG_FILE)"; \
		go run ./cmd/base -config=$(CONFIG_FILE); \
	else \
		go run ./cmd/base; \
	fi

## dev-base: 开发模式运行 base service (热重载,需要安装air)
dev-base:
	@echo "开发模式运行 base service..."
	@cd cmd/base && air

## test: 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

## test-cover: 运行测试并生成覆盖率报告
test-cover:
	@echo "运行测试并生成覆盖率报告..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

## clean: 清理构建文件
clean:
	@echo "清理构建文件..."
	@rm -rf $(BUILD_DIR)
	@rm -rf logs
	@rm -f coverage.out coverage.html
	@echo "清理完成"

## docker-build-base: 构建 base service Docker 镜像
docker-build-base:
	@echo "构建 Docker 镜像..."
	@docker build -t $(APP_NAME)-base:$(VERSION) -f Dockerfile --target base .
	@echo "镜像构建完成: $(APP_NAME)-base:$(VERSION)"

## docker-run-base: 运行 base service Docker 容器
docker-run-base:
	@echo "运行 Docker 容器..."
	@docker run -d --name $(APP_NAME)-base \
		-p 8080:8080 \
		-v $(PWD)/configs:/app/configs \
		$(APP_NAME)-base:$(VERSION)
	@echo "容器已启动"

## fmt: 格式化代码
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@goimports -w .
	@echo "格式化完成"

## lint: 代码检查
lint:
	@echo "运行代码检查..."
	@golangci-lint run

## deps: 下载依赖
deps:
	@echo "下载依赖..."
	@go mod download
	@go mod tidy
	@echo "依赖下载完成"


## version: 显示版本信息
version:
	@echo "$(APP_NAME) version $(VERSION)"

# 快捷命令
.PHONY: b r c
b: build-base
r: run-base
c: clean

