

项目名称: d-im

包名: d-im


## 一、完整目录结构

```
d-im/backend/
│
├── cmd/                                    # 应用程序入口
│   ├── api-gateway/                        # API网关服务
│   │   └── main.go
│   ├── connector/                          # 长连接服务（WebSocket/TCP）
│   │   └── main.go
│   ├── message/                            # 消息处理服务
│   │   └── main.go
│   ├── user/                               # 用户服务
│   │   └── main.go
│   ├── group/                              # 群组服务
│   │   └── main.go
│   ├── media/                              # 媒体处理服务
│   │   └── main.go
│   ├── push/                               # 推送服务
│   │   └── main.go
│   ├── scheduler/                          # 定时任务服务
│   │   └── main.go
│   └── migrate/                            # 数据库迁移工具
│       └── main.go
│
├── internal/                               # 私有应用和库代码
│   ├── gateway/                            # API网关业务逻辑
│   │   ├── handler/                        # HTTP处理器
│   │   │   ├── message_handler.go
│   │   │   ├── conversation_handler.go
│   │   │   ├── user_handler.go
│   │   │   ├── group_handler.go
│   │   │   └── middleware/
│   │   │       ├── auth.go                # 认证中间件
│   │   │       ├── ratelimit.go           # 限流中间件
│   │   │       ├── logger.go              # 日志中间件
│   │   │       └── recovery.go            # 恢复中间件
│   │   ├── router/                         # 路由定义
│   │   │   └── router.go
│   │   ├── dto/                            # 数据传输对象
│   │   │   ├── request/
│   │   │   │   ├── message_request.go
│   │   │   │   └── conversation_request.go
│   │   │   └── response/
│   │   │       ├── message_response.go
│   │   │       └── common_response.go
│   │   └── server.go                       # HTTP服务器配置
│   │
│   ├── connector/                          # 长连接服务逻辑
│   │   ├── ws/                             # WebSocket实现
│   │   │   ├── server.go
│   │   │   ├── client.go
│   │   │   ├── handler.go
│   │   │   └── heartbeat.go
│   │   ├── tcp/                            # TCP实现（可选）
│   │   │   ├── server.go
│   │   │   └── client.go
│   │   ├── channel/                        # 连接通道管理
│   │   │   └── channel_manager.go
│   │   └── auth.go                         # 连接认证
│   │
│   ├── message/                            # 消息服务核心逻辑
│   │   ├── service/                        # 业务逻辑层
│   │   │   ├── message_service.go
│   │   │   ├── send_service.go
│   │   │   ├── recall_service.go
│   │   │   └── forward_service.go
│   │   ├── repository/                     # 数据访问层
│   │   │   ├── message_repo.go
│   │   │   └── message_query.go
│   │   ├── router/                         # 消息路由
│   │   │   ├── router.go
│   │   │   └── strategy.go
│   │   └── dispatcher/                     # 消息分发
│   │       ├── dispatcher.go
│   │       └── queue.go
│   │
│   ├── conversation/                       # 会话服务
│   │   ├── service/
│   │   │   ├── conversation_service.go
│   │   │   └── sync_service.go            # 会话同步
│   │   └── repository/
│   │       ├── conversation_repo.go
│   │       └── mailbox_repo.go
│   │
│   ├── user/                               # 用户服务
│   │   ├── service/
│   │   │   ├── user_service.go
│   │   │   └── online_service.go          # 在线状态
│   │   └── repository/
│   │       └── user_repo.go
│   │
│   ├── group/                              # 群组服务
│   │   ├── service/
│   │   │   ├── group_service.go
│   │   │   └── member_service.go
│   │   └── repository/
│   │       ├── group_repo.go
│   │       └── member_repo.go
│   │
│   ├── media/                              # 媒体处理服务
│   │   ├── service/
│   │   │   ├── upload_service.go
│   │   │   └── thumbnail_service.go
│   │   ├── storage/                        # 存储适配器
│   │   │   ├── storage.go                 # 存储接口
│   │   │   ├── minio.go                   # MinIO实现
│   │   │   └── s3.go                      # AWS S3实现
│   │   └── processor/                      # 媒体处理器
│   │       ├── image_processor.go
│   │       └── video_processor.go
│   │
│   ├── push/                               # 推送服务
│   │   ├── service/
│   │   │   └── push_service.go
│   │   ├── provider/                       # 推送提供商
│   │   │   ├── provider.go                # 推送接口
│   │   │   ├── apns.go                    # iOS推送
│   │   │   ├── fcm.go                     # Android推送
│   │   │   └── huawei.go                  # 华为推送
│   │   └── template/                       # 推送模板
│   │       └── template.go
│   │
│   └── common/                             # 公共组件
│       ├── metadata/                       # 元数据传递
│       │   └── metadata.go
│       └── metrics/                        # 监控指标
│           └── metrics.go
│
├── pkg/                                    # 可公开使用的库
│   ├── types/                              # 核心类型定义
│   │   ├── message.go                      # MessageType等枚举
│   │   ├── content.go                      # 消息内容接口+各种Content
│   │   ├── template.go                     # 模板相关类型
│   │   ├── conversation.go                 # 会话相关类型
│   │   └── errors.go                       # 错误类型定义
│   │
│   ├── model/                              # 数据模型
│   │   ├── message.go                      # 消息文档模型
│   │   ├── conversation.go                 # 会话文档模型
│   │   ├── mailbox.go                      # 信箱文档模型
│   │   ├── chat.go                         # 聊天实体模型
│   │   ├── user.go                         # 用户模型
│   │   └── group.go                        # 群组模型
│   │
│   ├── mongodb/                            # MongoDB相关
│   │   ├── client.go                       # MongoDB客户端
│   │   ├── collection.go                   # 集合名称常量
│   │   ├── index.go                        # 索引定义
│   │   └── transaction.go                  # 事务辅助
│   │
│   ├── cache/                              # 缓存层
│   │   ├── redis/
│   │   │   ├── client.go
│   │   │   ├── message_cache.go
│   │   │   ├── session_cache.go
│   │   │   └── online_cache.go
│   │   └── local/
│   │       └── lru_cache.go
│   │
│   ├── queue/                              # 消息队列
│   │   ├── nats/
│   │   │   ├── publisher.go
│   │   │   └── subscriber.go
│   │   └── rabbitmq/
│   │       ├── producer.go
│   │       └── consumer.go
│   │
│   ├── protocol/                           # 通信协议
│   │   ├── ws_protocol.go                  # WebSocket协议
│   │   ├── http_protocol.go                # HTTP协议
│   │   └── codec/                          # 编解码
│   │       ├── json_codec.go
│   │       └── protobuf_codec.go
│   │
│   ├── pkg/errcode/                        # 错误码定义
│   │   └── errcode.go
│   │
│   ├── snowflake/                          # 雪花ID生成器
│   │   └── snowflake.go
│   │
│   ├── crypto/                             # 加密工具
│   │   ├── aes.go
│   │   └── hash.go
│   │
│   └── utils/                              # 工具函数
│       ├── time.go
│       ├── string.go
│       ├── slice.go
│       └── validator.go
│
├── api/                                    # API定义
│   ├── proto/                              # Protobuf定义
│   │   ├── message.proto
│   │   ├── conversation.proto
│   │   ├── user.proto
│   │   └── common.proto
│   └── swagger/                            # Swagger文档
│       └── api.yaml
│
├── configs/                                # 配置文件模板
│   ├── config.yaml                         # 主配置
│   ├── config.dev.yaml                     # 开发环境
│   ├── config.test.yaml                    # 测试环境
│   └── config.prod.yaml                    # 生产环境
│
├── deployments/                            # 部署相关
│   ├── docker/
│   │   ├── Dockerfile.api
│   │   ├── Dockerfile.connector
│   │   └── Dockerfile.message
│   ├── kubernetes/
│   │   ├── api-gateway.yaml
│   │   ├── connector.yaml
│   │   └── message.yaml
│   └── docker-compose/
│       ├── docker-compose.yaml
│       └── docker-compose.dev.yaml
│
├── scripts/                                # 脚本工具
│   ├── init_mongo.sh                       # MongoDB初始化
│   ├── init_mongo.js                       # MongoDB索引创建
│   ├── migrate.sh                          # 数据库迁移
│   └── benchmark.sh                        # 性能测试
│
├── test/                                   # 测试相关
│   ├── integration/                        # 集成测试
│   │   ├── message_test.go
│   │   └── conversation_test.go
│   ├── benchmark/                          # 基准测试
│   │   └── message_bench_test.go
│   └── testdata/                           # 测试数据
│       └── fixtures.go
│
├── docs/                                   # 文档
│   ├── architecture.md                     # 架构文档
│   ├── api.md                              # API文档
│   └── development.md                      # 开发指南
│
├── Makefile                                # 构建脚本
├── go.mod                                  # Go模块定义
├── go.sum                                  # 依赖校验
├── .gitignore
└── README.md
```

## 二、核心包详细说明

### 1. pkg/types/ - 类型定义（最基础层）

```go
// pkg/types/
types/
├── message.go          # 消息基础类型枚举
├── content.go          # 消息内容接口+各类型Content实现
├── content_test.go     # Content类型测试
├── template.go         # 模板消息相关类型
├── conversation.go     # 会话相关类型定义
├── user.go             # 用户相关类型
├── group.go            # 群组相关类型
└── errors.go           # 自定义错误类型
```

### 2. pkg/model/ - 数据模型层

```go
// pkg/model/
model/
├── message.go          # MongoDB消息文档
├── conversation.go     # 会话文档
├── mailbox.go          # 用户信箱文档
├── chat.go             # 聊天实体
├── user.go             # 用户文档
├── group.go            # 群组文档
├── device.go           # 设备信息
└── sequence.go         # 序列号管理
```

### 3. pkg/mongodb/ - 数据库操作封装

```go
// pkg/mongodb/
mongodb/
├── client.go           # MongoDB客户端初始化
├── collection.go       # 集合名称常量定义
├── index.go            # 索引创建和管理
├── index_test.go
├── transaction.go      # 事务辅助函数
├── repository.go       # 基础Repository接口
└── query_builder.go    # 查询构建器
```

### 4. internal/message/ - 消息服务（最核心）

```go
// internal/message/
message/
├── service/
│   ├── message_service.go       # 消息服务接口
│   ├── send_service.go          # 发送消息逻辑
│   ├── recall_service.go        # 撤回消息逻辑
│   ├── forward_service.go       # 转发消息逻辑
│   ├── ack_service.go           # 消息确认逻辑
│   └── service_test.go
│
├── repository/
│   ├── message_repo.go          # 消息数据访问
│   ├── message_repo_test.go
│   ├── message_query.go         # 消息查询构建
│   └── message_archive.go       # 消息归档
│
├── router/
│   ├── router.go                # 消息路由接口
│   ├── single_router.go        # 单聊路由
│   └── group_router.go         # 群聊路由
│
├── dispatcher/
│   ├── dispatcher.go            # 消息分发器
│   ├── worker.go                # 工作协程
│   └── queue.go                 # 分发队列
│
└── handler/
    ├── message_handler.go       # 消息处理器
    └── handler_chain.go         # 处理链（责任链模式）
```

### 5. internal/conversation/ - 会话服务

```go
// internal/conversation/
conversation/
├── service/
│   ├── conversation_service.go  # 会话服务
│   ├── sync_service.go          # 会话同步
│   └── unread_service.go       # 未读管理
│
├── repository/
│   ├── conversation_repo.go     # 会话数据访问
│   ├── mailbox_repo.go          # 信箱数据访问
│   └── sequence_repo.go         # 序列号管理
│
└── event/
    ├── conversation_event.go    # 会话事件
    └── handler.go               # 事件处理器
```

### 6. internal/connector/ - 长连接服务

```go
// internal/connector/
connector/
├── ws/
│   ├── server.go                # WebSocket服务器
│   ├── client.go                # 客户端连接
│   ├── handler.go               # 消息处理
│   ├── heartbeat.go             # 心跳检测
│   └── upgrader.go             # 连接升级
│
├── channel/
│   ├── channel.go               # 连接通道
│   ├── channel_manager.go       # 通道管理器
│   └── channel_map.go          # 通道映射
│
├── protocol/
│   ├── packet.go                # 数据包定义
│   ├── codec.go                 # 编解码
│   └── protocol.go              # 协议定义
│
└── auth.go                      # 连接认证
```

### 7. 配置文件示例

```yaml
# configs/config.yaml
app:
  name: d-im
  version: 1.0.0
  env: development

# 服务端口配置
server:
  gateway:
    http_port: 8080
    grpc_port: 9080
  connector:
    ws_port: 8081
    tcp_port: 8082

# MongoDB配置
mongodb:
  uri: mongodb://localhost:27017
  database: im_db
  pool_size: 100
  timeout: 10s
  collections:
    messages: messages
    conversations: conversations
    user_mailbox: user_mailbox
    chats: chats
    users: users
    groups: groups

# Redis配置
redis:
  addr: localhost:6379
  password: ""
  db: 0
  pool_size: 50

# 消息队列配置
nats:
  url: nats://localhost:4222
  subjects:
    message_send: im.message.send
    message_push: im.message.push
    message_event: im.message.event

# 雪花ID配置
snowflake:
  worker_id: 1
  datacenter_id: 1

# 媒体存储配置
storage:
  type: minio
  minio:
    endpoint: localhost:9000
    access_key: minioadmin
    secret_key: minioadmin
    bucket: im-media
    use_ssl: false

# 推送配置
push:
  apns:
    bundle_id: com.example.im
    key_path: /path/to/apns.p8
    key_id: ABC123
    team_id: TEAM456
  fcm:
    server_key: your-fcm-server-key

# 日志配置
log:
  level: info
  format: json
  output: stdout
  file:
    path: /var/log/im/
    max_size: 100MB
    max_age: 30
    max_backups: 10
```

### 8. 主程序入口示例

```go
// cmd/message/main.go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "d-im/internal/message/service"
    "d-im/pkg/mongodb"
    "d-im/pkg/cache/redis"
    "d-im/pkg/queue/nats"
    "d-im/pkg/snowflake"
)

func main() {
    // 1. 加载配置
    cfg := loadConfig()
    
    // 2. 初始化基础设施
    mongoClient := mongodb.NewClient(cfg.MongoDB)
    redisClient := redis.NewClient(cfg.Redis)
    natsConn := nats.NewConnection(cfg.NATS)
    idGenerator := snowflake.NewGenerator(cfg.Snowflake)
    
    // 3. 创建依赖注入
    deps := &Dependencies{
        MongoDB:  mongoClient,
        Redis:    redisClient,
        NATS:    natsConn,
        IDGen:    idGenerator,
    }
    
    // 4. 初始化服务
    messageService := service.NewMessageService(deps)
    
    // 5. 启动服务
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go messageService.Start(ctx)
    
    // 6. 等待退出信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    log.Println("Shutting down message service...")
    cancel()
    messageService.Stop()
}
```

这个目录结构遵循Go社区的最佳实践，具有以下特点：

1. **清晰的分层架构**：类型定义→模型→数据访问→业务逻辑→服务
2. **服务独立性**：每个微服务有自己的internal目录
3. **可测试性**：接口定义清晰，便于单元测试和集成测试
4. **可扩展性**：新增消息类型或服务时，结构清晰易扩展
5. **标准化**：遵循Go项目的标准布局（cmd、internal、pkg）
6. **部署友好**：包含完整的Docker和Kubernetes配置