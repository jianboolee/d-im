# Push 独立服务架构文档

## 一、服务定位

### 1.1 服务职责

Push Service 是 IM 系统中专门负责**离线消息推送**的独立微服务，通过 APNs/FCM/极光推送等第三方通道，将消息送达至离线用户的设备。

### 1.2 核心职责

| 职责 | 说明 |
|------|------|
| **离线推送消费** | 订阅并消费 NATS 离线推送事件 |
| **第三方推送调用** | 对接 APNs、FCM、极光推送等推送通道 |
| **推送策略管理** | 去重、限流、优先级控制 |
| **设备管理** | 推送 Token 管理、设备注册/注销 |
| **推送质量保障** | 重试机制、失败处理、监控告警 |

### 1.3 服务边界

```
┌─────────────────────────────────────────────────────────────┐
│                        Push Service                         │
│                                                             │
│  ┌─────────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │ 离线事件消费      │  │  推送策略     │  │  设备管理      │  │
│  │                 │  │              │  │               │  │
│  │ · 订阅NATS事件   │  │ · 去重检查    │  │ · Token存储    │  │
│  │ · 事件解析       │  │ · 限流控制    │  │ · 设备注册     │  │
│  │ · 重试处理       │  │ · 优先级排序   │  │ · Token失效    │  │
│  └────────┬────────┘  └──────┬───────┘  │   处理         │  │
│           │                  │           └───────────────┘  │
│           └──────────────────┤                              │
│                              ▼                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    推送提供商层                        │  │
│  │                                                      │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │  │
│  │  │ Mock     │  │  APNs    │  │  FCM     │  ...      │  │
│  │  │ Provider │  │ Provider │  │ Provider │           │  │
│  │  └──────────┘  └──────────┘  └──────────┘           │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘

✅ 应该做的：
  · 消费离线推送事件
  · 调用第三方推送服务
  · 推送策略管理
  · 设备Token管理

❌ 不应该做的：
  · WebSocket连接管理（属于Connector）
  · 消息存储（属于Message Service）
  · 会话管理（属于Conversation Service）
```

## 二、系统架构

### 2.1 整体架构图

```
┌──────────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│  Client  │     │Connector │     │   NATS   │     │   Push   │
│  (离线)   │     │ Service  │     │  Server  │     │  Service │
└────┬─────┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │                  │
     │  断开连接       │                │                  │
     │───────────────>│                │                  │
     │                │                │                  │
     │                │ 检测离线        │                  │
     │                │ Publish        │                  │
     │                │───────────────>│                  │
     │                │ im.push.offline│                  │
     │                │ .{user_id}     │                  │
     │                │                │   Subscribe      │
     │                │                │─────────────────>│
     │                │                │                  │
     │                │                │          去重检查 │
     │                │                │          限流检查 │
     │                │                │          获取设备 │
     │                │                │                  │
     │                │                │           ┌──────┴──────┐
     │                │                │           │  推送提供商   │
     │                │                │           │  Mock/APNs  │
     │                │                │           │  /FCM/极光   │
     │                │                │           └──────┬──────┘
     │                │                │                  │
     │  APNs/FCM推送   │                │                  │
     │<───────────────────────────────────────────────────┘
```

### 2.2 服务内部模块

```
Push Service
│
├── Consumer Layer (消费层)
│   ├── OfflinePushConsumer      # 离线推送事件消费者
│   ├── DeadLetterConsumer       # 死信队列消费者（可选）
│   └── RetryConsumer            # 重试队列消费者（可选）
│
├── Business Layer (业务层)
│   ├── PushService              # 推送核心业务逻辑
│   ├── DedupService             # 去重服务
│   ├── ThrottleService          # 限流服务
│   └── PriorityService          # 优先级服务
│
├── Provider Layer (提供商层)
│   ├── ProviderFactory          # 提供商工厂
│   ├── MockProvider             # Mock实现（开发测试）
│   ├── APNsProvider             # APNs实现
│   ├── FCMProvider              # FCM实现
│   └── JPushProvider            # 极光推送实现
│
├── Repository Layer (数据层)
│   ├── DeviceRepository         # 设备数据访问
│   └── PushRecordRepository     # 推送记录数据访问
│
└── API Layer (接口层)
    ├── PushController           # HTTP管理接口
    └── HealthController         # 健康检查接口
```

## 三、数据流转

### 3.1 主流程

```
NATS Event (im.push.offline.{user_id})
        │
        ▼
┌───────────────────┐
│ 1. 事件消费        │
│   · 解析user_id    │
│   · 解析消息内容    │
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 2. 前置检查        │
│   · 用户是否已上线  │ ← 是 → 跳过推送
│   · 消息是否重复    │ ← 是 → 跳过推送
│   · 频率是否超限    │ ← 是 → 延迟推送
└───────┬───────────┘
        │ 通过
        ▼
┌───────────────────┐
│ 3. 获取设备列表    │
│   · 查询用户所有设备 │
│   · 过滤无效Token  │
│   · 按平台分组      │
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 4. 构建推送内容    │
│   · 根据消息类型    │
│   · 生成标题和摘要  │
│   · 构建扩展数据    │
│   · 设置角标数      │
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 5. 选择推送提供商   │
│   · iOS → APNs    │
│   · Android → FCM  │
│   · 按平台批量推送  │
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 6. 执行推送        │
│   · 调用第三方API   │
│   · 记录推送结果    │
│   · 处理失败Token   │
└───────┬───────────┘
        │
        ▼
┌───────────────────┐
│ 7. 后置处理        │
│   · 设置去重标记    │
│   · 更新推送统计    │
│   · 失败重试/告警   │
└───────────────────┘
```

### 3.2 时序图

```
Connector       NATS         Push Service    Redis       MongoDB      APNs/FCM
    │             │               │            │           │             │
    │─Publish────>│               │            │           │             │
    │ offline     │               │            │           │             │
    │             │─Deliver──────>│            │           │             │
    │             │               │            │           │             │
    │             │               │─检查在线───>│           │             │
    │             │               │<─离线───────│           │             │
    │             │               │            │           │             │
    │             │               │─去重检查───>│           │             │
    │             │               │<─未重复─────│           │             │
    │             │               │            │           │             │
    │             │               │─限流检查───>│           │             │
    │             │               │<─允许───────│           │             │
    │             │               │            │           │             │
    │             │               │─查询设备──────────────>│             │
    │             │               │<─设备列表──────────────│             │
    │             │               │            │           │             │
    │             │               │─构建推送内容            │             │
    │             │               │            │           │             │
    │             │               │─────────────────────────────────────>│
    │             │               │            │           │   APNs推送   │
    │             │               │<─────────────────────────────────────│
    │             │               │            │           │  推送结果    │
    │             │               │            │           │             │
    │             │               │─记录推送结果──────────>│             │
    │             │               │            │           │             │
    │             │               │─设置去重───>│           │             │
    │             │               │<─OK────────│           │             │
```

## 四、核心设计

### 4.1 推送提供商抽象

采用**策略模式**，定义统一接口，支持多种推送通道：

```
                    ┌─────────────────────┐
                    │   PushProvider      │
                    │   (接口)             │
                    │                     │
                    │ + Push()            │
                    │ + BatchPush()       │
                    │ + IsHealthy()       │
                    └──────────┬──────────┘
                               │
          ┌────────────────────┼────────────────────┐
          │                    │                    │
          ▼                    ▼                    ▼
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│  MockProvider   │  │  APNsProvider   │  │  FCMProvider    │
│                 │  │                 │  │                 │
│ · 模拟推送       │  │ · HTTP/2 连接   │  │ · Firebase SDK  │
│ · 可配置失败率   │  │ · JWT 认证      │  │ · 服务器密钥认证  │
│ · 推送记录       │  │ · Token 推送     │  │ · Token/主题推送 │
│ · 延迟模拟       │  │ · 角标管理       │  │ · 数据消息      │
└─────────────────┘  └─────────────────┘  └─────────────────┘

切换方式：修改配置文件中的 provider 字段即可切换
  provider: mock     # 开发环境
  provider: jpush    # 生产环境
```

### 4.2 多级去重策略

```
第一级：NATS 消息去重
  · 同一消息可能被多个Connector节点发布
  · NATS JetStream 的 Deduplication 机制去重

第二级：Redis 业务去重
  · Key: push_dedup:{user_id}:{msg_id}
  · TTL: 10分钟
  · 防止短时间内重复推送同一条消息

第三级：设备级去重
  · 同一设备短时间内不重复推送
  · Key: push_device:{device_id}:{msg_id}
  · TTL: 5分钟
```

### 4.3 限流策略

```
用户级限流：
  · 粒度：每个用户每分钟最多 N 条推送
  · 实现：Redis 滑动窗口 + INCR
  · Key: push_throttle:{user_id}
  · 超限处理：丢弃或延迟到下一窗口

设备级限流：
  · 粒度：每个设备每分钟最多 M 条推送
  · 实现：Redis 令牌桶
  · Key: push_device_throttle:{device_id}

全局限流：
  · 粒度：整个服务每秒最多 P 条推送
  · 实现：本地令牌桶
  · 目的：保护第三方推送API不被限流
```

### 4.4 优先级策略

```
优先级定义：
  ┌────────────┬──────────┬──────────────────────┐
  │ 优先级      │ 适用场景   │ 说明                  │
  ├────────────┼──────────┼──────────────────────┤
  │ CRITICAL   │ 系统通知   │ 立即推送，忽略限流      │
  │ HIGH       │ @消息     │ 优先推送，较短的退避时间  │
  │ NORMAL     │ 普通消息   │ 正常推送               │
  │ LOW        │ 营销消息   │ 可延迟推送             │
  └────────────┴──────────┴──────────────────────┘

优先级队列：
  离线事件 → 优先级分类 → 高优先级队列 → 优先消费
                        → 低优先级队列 → 闲时消费
```

### 4.5 重试机制

```
重试策略：
  1次失败 → 延迟 5秒  → 重试
  2次失败 → 延迟 15秒 → 重试
  3次失败 → 延迟 30秒 → 重试
  4次失败 → 标记失败，进入死信队列

可重试错误：
  · 网络超时
  · 服务暂时不可用
  · 连接被拒绝

不可重试错误：
  · 设备Token无效 → 标记设备Token失效
  · 证书过期    → 告警
  · 消息格式错误  → 记录日志
```

## 五、数据存储

### 5.1 MongoDB 集合设计

```
devices 集合（设备信息）：
  {
    _id: ObjectId,
    device_id: "device_uuid_xxx",      // 设备唯一ID
    user_id: "user_123",              // 用户ID
    platform: "ios",                  // ios/android/web
    push_token: "apns_token_xxx",     // 推送Token
    token_status: "active",           // active/inactive/expired
    device_model: "iPhone 15 Pro",
    os_version: "17.0",
    app_version: "1.0.0",
    last_active_at: ISODate(...),
    created_at: ISODate(...),
    updated_at: ISODate(...)
  }
  索引：
    · {user_id: 1, platform: 1}
    · {push_token: 1}
    · {token_status: 1}

push_records 集合（推送记录）：
  {
    _id: ObjectId,
    user_id: "user_123",
    device_id: "device_uuid_xxx",
    msg_id: "msg_456",
    chat_id: "chat_789",
    platform: "ios",
    provider: "apns",
    status: "success",                // success/failed
    provider_msg_id: "apns_msg_xxx",  // 第三方返回的消息ID
    error_code: "",
    error_msg: "",
    retry_count: 0,
    push_time: ISODate(...),
    created_at: ISODate(...)
  }
  索引：
    · {user_id: 1, push_time: -1}
    · {msg_id: 1}
    · {status: 1, push_time: -1}
    · TTL索引: {push_time: 1, expireAfterSeconds: 30天}

dead_letter_queue 集合（死信队列）：
  {
    _id: ObjectId,
    user_id: "user_123",
    msg_id: "msg_456",
    original_event: {...},            // 原始事件数据
    error_msg: "max retry exceeded",
    retry_count: 3,
    status: "pending",                // pending/retrying/resolved
    created_at: ISODate(...)
  }
```

### 5.2 Redis 缓存设计

```
去重缓存：
  Key: push_dedup:{user_id}:{msg_id}
  Value: 推送时间戳
  TTL: 10分钟

限流缓存：
  Key: push_throttle:{user_id}:{minute_window}
  Value: 计数器
  TTL: 60秒

在线状态缓存：
  Key: user_online:{user_id}
  Value: 最近活跃时间戳
  TTL: 30秒（通过心跳续期）

设备Token缓存：
  Key: device_tokens:{user_id}
  Value: JSON [{"device_id": "...", "token": "...", "platform": "..."}]
  TTL: 30分钟

推送统计缓存：
  Key: push_stats:{date}
  Value: {"total": 1000, "success": 950, "failed": 50}
  TTL: 24小时
```

## 六、配置管理

### 6.1 配置分层

```
第一层：代码常量（不可配置）
  · NATS 离线推送主题前缀
  · 默认的重试次数和延迟
  · 默认的限流阈值

第二层：配置文件（可配置，有默认值）
  · 推送提供商选择
  · 各提供商的详细参数
  · 去重和限流的TTL
  · 重试策略参数

第三层：环境变量（环境差异、敏感信息）
  · NATS 连接地址
  · Redis 连接地址
  · MongoDB 连接地址
  · APNs 证书路径/密码
  · FCM 服务器密钥
  · 极光推送 AppKey/MasterSecret
```

### 6.2 配置示例

```yaml
push:
  # 当前提供商：mock / apns / fcm / jpush
  provider: mock
  
  # 去重配置
  dedup:
    enabled: true
    ttl_seconds: 600          # 10分钟
    
  # 限流配置
  throttle:
    enabled: true
    user_max_per_min: 10      # 每用户每分钟10条
    device_max_per_min: 5     # 每设备每分钟5条
    global_max_per_sec: 1000  # 全局每秒1000条
    
  # 重试配置
  retry:
    max_attempts: 3
    delays: [5, 15, 30]       # 重试延迟（秒）
    
  # Mock提供商配置
  mock:
    enabled: true
    log_dir: /var/log/push
    failure_rate: 0.05
    latency_min_ms: 10
    latency_max_ms: 100
    
  # 极光推送配置（生产）
  jpush:
    app_key: ${JPUSH_APP_KEY}
    master_secret: ${JPUSH_MASTER_SECRET}
    apns_production: true
    timeout_seconds: 30
```

## 七、监控告警

### 7.1 关键指标

```
业务指标：
  ┌──────────────────┬─────────────────────────┐
  │ 指标              │ 说明                      │
  ├──────────────────┼─────────────────────────┤
  │ push_total        │ 推送总数                  │
  │ push_success      │ 推送成功数                │
  │ push_failed       │ 推送失败数                │
  │ push_success_rate │ 推送成功率                │
  │ push_latency_p50  │ 推送延迟P50               │
  │ push_latency_p99  │ 推送延迟P99               │
  │ push_dedup_count  │ 去重拦截数                │
  │ push_throttle_cnt │ 限流拦截数                │
  │ push_retry_count  │ 重试次数                  │
  │ dead_letter_count │ 死信队列积压数            │
  │ token_invalid_cnt │ 无效Token数               │
  └──────────────────┴─────────────────────────┘

系统指标：
  · 消费延迟（NATS消息到达至推送完成）
  · 各Provider的健康状态
  · Redis/MongoDB连接状态
  · 内存/CPU使用率
```

### 7.2 告警规则

```
严重告警（立即处理）：
  · 推送成功率 < 95%（持续5分钟）
  · 死信队列积压 > 1000 条
  · Provider 全部不健康
  · 服务崩溃

警告告警（需要关注）：
  · 推送成功率 < 98%（持续10分钟）
  · 推送延迟P99 > 3秒
  · 无效Token率 > 10%
  · 重试率 > 5%

通知告警（记录观察）：
  · 限流拦截数突增
  · 单用户推送量异常
  · Provider 切换事件
```

## 八、部署架构

### 8.1 部署拓扑

```
                    ┌──────────────┐
                    │  NATS Server │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │  Push-1  │ │  Push-2  │ │  Push-N  │
        │ (实例1)   │ │ (实例2)   │ │ (实例N)   │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │            │            │
             └────────────┼────────────┘
                          │
              ┌───────────┼───────────┐
              │           │           │
              ▼           ▼           ▼
        ┌─────────┐ ┌─────────┐ ┌─────────┐
        │  Redis  │ │ MongoDB │ │ APNs    │
        │  Cluster│ │ Cluster │ │ FCM     │
        └─────────┘ └─────────┘ └─────────┘

多实例部署优势：
  · NATS QueueSubscribe 实现负载均衡
  · 故障自动转移
  · 水平扩展提升吞吐量
```

### 8.2 启动流程

```
1. 加载配置（配置文件 + 环境变量）
2. 初始化基础设施连接（NATS, Redis, MongoDB）
3. 注册推送提供商（Mock/APNs/FCM/极光）
4. 设置当前活跃的提供商
5. 启动离线事件消费者
6. 启动死信队列消费者（可选）
7. 启动HTTP管理接口
8. 注册服务发现/健康检查
9. 上报启动指标
```

### 8.3 优雅关闭

```
1. 停止接收新请求（HTTP接口返回503）
2. 停止NATS订阅
3. 等待正在处理的推送完成（最多30秒）
4. 关闭推送提供商连接
5. 关闭数据存储连接
6. 上报关闭指标
7. 退出进程
```

## 九、开发路线图

### 阶段一：MVP（Mock实现）

```
目标：验证架构、跑通流程
  · MockProvider 实现
  · 基础消费逻辑
  · 设备数据存储
  · 推送记录存储
  · HTTP管理接口
```

### 阶段二：生产准备

```
目标：具备生产可用性
  · 极光推送Provider
  · 去重服务
  · 限流服务
  · 重试机制
  · 死信队列
  · 监控告警
```

### 阶段三：优化完善

```
目标：高性能、高可用
  · APNs/FCM原生Provider
  · 优先级队列
  · 批量推送优化
  · 推送A/B测试
  · 推送效果分析
```