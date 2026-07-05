.PHONY: run run-gateway run-message run-connector build dev clean demo

# 自动加载 .env 文件（如果存在）
ifneq (,$(wildcard .env))
    include .env
    export
endif

# 默认配置文件
CONFIG ?= configs/config.dev.yaml

# 启动所有基础设施
dev:
	docker compose -f deployments/docker-compose/docker-compose.dev.yaml up -d
	@echo "MongoDB:  mongodb://localhost:27018"
	@echo "Redis:    redis://localhost:6380"
	@echo "NATS:     nats://localhost:4223 (monitor: http://localhost:8223)"

# 关闭基础设施
dev-down:
	docker compose -f deployments/docker-compose/docker-compose.dev.yaml down

# 编译
build:
	go build -o bin/api-gateway ./cmd/api-gateway
	go build -o bin/message ./cmd/message
	go build -o bin/connector ./cmd/connector
	go build -o bin/user ./cmd/user

# 启动 api-gateway
run-gateway:
	go run ./cmd/api-gateway -config $(CONFIG)

# 启动 message 服务
run-message:
	go run ./cmd/message -config $(CONFIG)

# 启动 connector
run-connector:
	go run ./cmd/connector -config $(CONFIG)

# 启动 user 服务（订阅事件总线）
run-user:
	go run ./cmd/user -config $(CONFIG)

# 启动所有服务
run:
	@echo "Starting all services with config: $(CONFIG)"
	@echo "  make run-gateway  # HTTP + WebSocket API"
	@echo "  make run-message  # Message processing"
	@echo "  make run-connector # WebSocket connections"
	@echo "  make run-user      # User event subscription"

# 清理
clean:
	rm -rf bin/

demo:
	go run ./demo