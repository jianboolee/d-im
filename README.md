# d-im

即时通讯（IM）后端服务，基于 Go 微服务架构。

## 架构总览

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  api-gateway │     │   message    │     │  connector   │
│  :8080 HTTP  │     │  NATS 订阅    │     │  :8081 WS    │
└──────┬───────┘     └──────┬───────┘     └──────┬───────┘
       │                    │                    │
       └────────────────────┼────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
     ┌────┴────┐      ┌─────┴─────┐     ┌─────┴─────┐
     │ MongoDB │      │   Redis   │     │   NATS    │
     └─────────┘      └───────────┘     └───────────┘
```

| 服务 | 端口 | 说明 |
|------|------|------|
| **api-gateway** | 8080 | HTTP API 网关，JWT 认证，消息接口 |
| **connector** | 8081 | WebSocket 长连接，实时消息推送，多设备支持 |
| **message** | — | 消息处理服务（NATS 订阅消息事件） |

## 技术栈

- **语言**: Go 1.25+
- **数据库**: MongoDB 8.0
- **缓存**: Redis 8
- **消息队列**: NATS 2
- **配置**: Viper + YAML + 环境变量
- **认证**: JWT (HS256)，支持 access/refresh/ticket 三种 token

## 快速开始

### 1. 环境要求

- Go 1.25+
- Docker & Docker Compose

### 2. 启动基础设施

```bash
cd backend

# 复制环境变量模板
cp .env.example .env

# 启动 MongoDB + Redis + NATS
make dev
```

服务端口（已避开常见默认端口）：

| 服务 | 地址 |
|------|------|
| MongoDB | `mongodb://localhost:27018` |
| Redis | `redis://localhost:6380` |
| NATS | `nats://localhost:4223` |
| NATS Monitor | `http://localhost:8223` |

### 3. 启动服务

```bash
# 终端 1：启动 API 网关
make run-gateway

# 终端 2：启动消息服务
make run-message

# 终端 3：启动 WebSocket 长连接服务
make run-connector
```

或指定配置文件：

```bash
make run-gateway CONFIG=configs/config.yaml
```

### 4. 验证

```bash
# 健康检查
curl http://localhost:8080/health

# 获取一次性 ticket（业务系统调用，需 API Key）
curl -X POST http://localhost:8080/api/v1/auth/ticket \
  -H "Content-Type: application/json" \
  -H "X-API-Key: im-api-key-change-me" \
  -d '{"uid": "user_001"}'

# ticket 换取 token（前端调用）
curl -X POST http://localhost:8080/api/v1/auth/token \
  -H "Content-Type: application/json" \
  -d '{"ticket": "<ticket>", "device_id": "web_chrome_v1"}'

# 发送消息（需 access_token）
curl -X POST http://localhost:8080/api/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <access_token>" \
  -d '{
    "chat_id": "single_user_001_user_002",
    "message_type": "text",
    "content": {"text": "你好！"}
  }'
```

## WebSocket 连接

```javascript
// 前端 WebSocket 连接
const token = "<access_token>";
const deviceId = "web_chrome_v1";
const ws = new WebSocket(`ws://localhost:8081/ws?token=${token}&device_id=${deviceId}`);

ws.onmessage = (event) => {
  const packet = JSON.parse(event.data);
  // 处理推送消息
};
```

多设备支持：同一用户的不同设备（Web/iOS/Android）可同时建立独立连接，消息会推送到所有在线设备。

## API 接口

### 认证

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/auth/ticket` | X-API-Key | 业务系统换取一次性 ticket |
| POST | `/api/v1/auth/token` | 无 | 前端 ticket 换取 access + refresh token |
| POST | `/api/v1/auth/refresh` | Bearer refresh_token | 刷新 token |
| POST | `/api/v1/auth/logout` | 无 | 登出 |

### 消息

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| POST | `/api/v1/messages` | Bearer access_token | 发送消息 |
| POST | `/api/v1/message/recall` | Bearer access_token | 撤回消息 |
| POST | `/api/v1/message/forward` | Bearer access_token | 转发消息 |
| GET | `/api/v1/message/list` | Bearer access_token | 查询消息列表 |

### 健康检查

| 方法 | 路径 | 鉴权 |
|------|------|------|
| GET | `/health` | 无 |

## 配置

配置文件优先级：**环境变量 > YAML 配置**

### YAML 配置

```bash
# 开发环境（默认）
make run-gateway CONFIG=configs/config.dev.yaml

# 使用 .env 加载环境变量
make run-gateway
```

### 环境变量

复制 `.env.example` 为 `.env` 后修改：

```bash
cp .env.example .env
```

| 变量 | 说明 |
|------|------|
| `MONGODB_URI` | MongoDB 连接地址 |
| `REDIS_ADDR` | Redis 地址 |
| `NATS_URL` | NATS 地址 |
| `JWT_SECRET` | JWT 签名密钥 |
| `JWT_API_KEY` | 业务系统 API Key |

## 目录结构

```
backend/
├── cmd/            # 服务启动入口
│   ├── api-gateway/
│   ├── connector/
│   └── message/
├── configs/        # YAML 配置文件
├── internal/       # 业务逻辑
│   ├── gateway/    # API 网关
│   ├── connector/  # WebSocket 长连接
│   ├── message/    # 消息服务
│   ├── conversation/ # 会话服务
│   ├── user/       # 用户服务
│   └── group/      # 群组服务
├── pkg/            # 公共库
│   ├── types/      # 核心类型定义
│   ├── model/      # 数据模型
│   ├── mongodb/    # 数据库
│   ├── cache/      # Redis 缓存
│   ├── queue/      # NATS 消息队列
│   ├── crypto/     # JWT + 加密
│   ├── config/     # 配置（Viper）
└── deployments/    # Docker Compose
```

## 常用命令

```bash
make dev          # 启动基础设施
make dev-down     # 关闭基础设施
make build        # 编译所有服务
make clean        # 清理编译产物
