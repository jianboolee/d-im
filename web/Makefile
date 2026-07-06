IMAGE_REGISTRY ?= registry.cn-beijing.aliyuncs.com
IMAGE_NAMESPACE ?= jianboo
IMAGE_NAME ?= dim-frontend
IMAGE_TAG ?= $(shell git rev-parse --short HEAD 2>/dev/null || date +%Y%m%d%H%M%S)
DOCKER_PLATFORM ?= linux/amd64
DOCKER_BUILDER ?= docker buildx
VITE_IM_API_BASE ?= /
VITE_IM_WS_BASE ?=

IMAGE := $(IMAGE_REGISTRY)/$(IMAGE_NAMESPACE)/$(IMAGE_NAME):$(IMAGE_TAG)
LATEST_IMAGE := $(IMAGE_REGISTRY)/$(IMAGE_NAMESPACE)/$(IMAGE_NAME):latest

.PHONY: help install dev build preview type-check format clean docker-build docker-push docker-release

help:
	@echo "MP 项目常用命令"
	@echo ""
	@echo "  make install     - 安装依赖"
	@echo "  make dev         - 启动开发服务器"
	@echo "  make build       - 构建生产包"
	@echo "  make preview     - 预览生产构建"
	@echo "  make type-check  - 运行 Vue/TypeScript 类型检查"
	@echo "  make format      - 格式化 src 目录"
	@echo "  make clean       - 清理构建和临时文件"
	@echo "  make docker-build   - 构建 linux/amd64 Docker 镜像"
	@echo "  make docker-push    - 推送 Docker 镜像"
	@echo "  make docker-release - 构建并推送 Docker 镜像"

install:
	@echo "========================================="
	@echo "安装项目依赖"
	@echo "========================================="
	@npm install

dev:
	@echo "========================================="
	@echo "启动开发服务器"
	@echo "========================================="
	@npm run dev

build:
	@echo "========================================="
	@echo "构建生产包"
	@echo "========================================="
	@npm run build

preview:
	@echo "========================================="
	@echo "预览生产构建"
	@echo "========================================="
	@npm run preview

type-check:
	@echo "========================================="
	@echo "运行类型检查"
	@echo "========================================="
	@npm run type-check

format:
	@echo "========================================="
	@echo "格式化 src 目录"
	@echo "========================================="
	@npm run format

clean:
	@echo "========================================="
	@echo "清理构建和临时文件"
	@echo "========================================="
	@rm -rf dist node_modules/.tmp

docker-build:
	$(DOCKER_BUILDER) build \
		--platform $(DOCKER_PLATFORM) \
		--build-arg VITE_IM_API_BASE=$(VITE_IM_API_BASE) \
		--build-arg VITE_IM_WS_BASE=$(VITE_IM_WS_BASE) \
		-t $(IMAGE) \
		-t $(LATEST_IMAGE) \
		--load \
		.

docker-push:
	docker push $(IMAGE)
	docker push $(LATEST_IMAGE)

docker-release: docker-build docker-push
