Connector 是 IM 系统中连接"服务端消息"和"客户端设备"的关键桥梁。让我系统梳理它的定位和边界。

## 一、Connector Service 的服务定位

### 1.1 一句话定义

**Connector Service 是长连接管理中心，负责维护与客户端的长连接（WebSocket/TCP），并作为服务端消息推送到客户端的唯一出口。**

### 1.2 核心职责

```go
type ConnectorServiceResponsibilities struct {
    // ====== 核心职责（不可替代） ======
    
    // 1. 长连接管理
    ConnectionManagement string // 接受、维护、断开 WebSocket/TCP 连接
    
    // 2. 连接认证
    ConnectionAuth string // 验证连接请求的合法性（JWT 校验）
    
    // 3. 心跳检测
    HeartbeatDetection string // 检测客户端存活状态，清理死连接
    
    // 4. 消息投递
    MessageDelivery string // 将服务端消息投递到客户端（WebSocket Push）
    
    // 5. 连接状态管理
    ConnectionState string // 管理用户在线/离线状态，多端连接管理
    
    // 6. 连接路由
    ConnectionRouting string // 知道哪个用户连接在哪个节点
    
    // ====== 明确不做的事情 ======
    
    // ❌ 不存储消息（Message Service 负责）
    // ❌ 不决策推送方式（Push Service 负责）
    // ❌ 不管理会话（Conversation Service 负责）
    // ❌ 不处理业务逻辑（由各业务服务负责）
}
```

## 二、Connector 的服务边界

### 2.1 拥有什么

```
Connector 拥有：
  ✅ WebSocket/TCP 连接实例
  ✅ 连接池（所有在线客户端的连接）
  ✅ 用户→连接的映射关系（uid → connection）
  ✅ 设备→连接的映射关系（device_id → connection）
  ✅ 连接状态（在线、心跳时间、IP等）
  ✅ 连接统计（连接数、消息数、流量等）
```

### 2.2 不拥有什么

```
Connector 不拥有：
  ❌ 消息内容（不知道消息是什么）
  ❌ 业务数据（用户信息、群组信息等）
  ❌ 推送决策（不知道谁该推、什么时候推）
  ❌ 离线数据（用户离线后的处理）
  ❌ 会话状态（未读计数等）
```

## 三、Connector 的极简设计

### 3.1 核心定位：透明的消息通道

```
Connector = 管道

类比：就像电话线
  · 不知道通话内容是什么
  · 不知道谁该给谁打电话
  · 只负责建立和维护连接
  · 只负责传输信号

输入：服务端要推送的消息
输出：客户端的 WebSocket 收到消息

中间不做任何业务判断。
```

### 3.2 两个核心接口

```go
// Connector 只提供两个核心能力

// 1. 被动推送（接收服务端的推送请求）
//    RPC: im.connector.push
//    输入：{user_id, payload}
//    输出：{success, reason}
//    逻辑：找连接 → 推送 → 返回结果

// 2. 主动上报（客户端消息上传）
//    WebSocket Message
//    输入：客户端发送的消息
//    输出：发布到 NATS（im.client.message）
//    逻辑：接收 → 验证 → 发布到 NATS
```

## 四、完整的架构设计

### 4.1 服务架构图

```
                          ┌─────────────────────────┐
                          │     NATS Server         │
                          └─────────┬───────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    │               │               │
            ┌───────▼──────┐ ┌──────▼──────┐ ┌──────▼──────┐
            │  Connector-1 │ │ Connector-2 │ │ Connector-3 │
            │  (节点1)     │ │  (节点2)    │ │  (节点3)    │
            └───────┬──────┘ └──────┬──────┘ └──────┬──────┘
                    │               │               │
            ┌───────┼───────┐       │               │
            │       │       │       │               │
        ┌───▼──┐ ┌─▼──┐ ┌─▼──┐   ┌▼──┐          ┌▼──┐
        │客户端A│ │客户端B│ │客户端C│   │客户端D│          │客户端E│
        │(手机) │ │(平板)│ │(电脑)│   │(手机)│          │(电脑)│
        └──────┘ └─────┘ └─────┘   └────┘          └─────┘
```

### 4.2 内部模块

```go
// Connector 内部模块
type ConnectorModules struct {
    
    // ====== 连接层 ======
    ConnectionLayer struct {
        // WebSocket 服务器
        WSServer struct {
            // 升级 HTTP 到 WebSocket
            Upgrade()
            // 接受新连接
            Accept()
            // 关闭连接
            Close()
        }
        
        // TCP 服务器（可选）
        TCPServer struct {
            Accept()
            Close()
        }
    }
    
    // ====== 会话层 ======
    SessionLayer struct {
        // 客户端连接封装
        Client struct {
            // 连接信息
            ConnID     string
            UID        string
            DeviceID   string
            Platform   string
            
            // 连接状态
            IsOnline   bool
            LastActive time.Time
            
            // 发送消息
            Send(payload []byte) error
            // 关闭连接
            Close()
        }
        
        // 连接管理器
        ChannelManager struct {
            // 用户→客户端映射（一个用户可能有多个设备）
            // uid → []*Client
            userClients map[string][]*Client
            
            // 客户端ID→客户端映射
            // connID → *Client
            clients map[string]*Client
            
            // 获取用户的所有连接
            GetUserClients(uid string) []*Client
            
            // 获取特定连接
            GetClient(uid, deviceID string) *Client
            
            // 注册连接
            Register(client *Client)
            
            // 移除连接
            Unregister(client *Client)
        }
    }
    
    // ====== 协议层 ======
    ProtocolLayer struct {
        // 消息编解码
        Codec struct {
            Encode(msg *Message) ([]byte, error)
            Decode(data []byte) (*Message, error)
        }
        
        // 协议定义
        Protocol struct {
            // 上行消息（客户端→服务端）
            // 下行消息（服务端→客户端）
        }
    }
    
    // ====== 通信层 ======
    CommunicationLayer struct {
        // NATS 客户端
        NATSClient struct {
            // 订阅推送请求（RPC）
            SubscribePush()
            
            // 发布客户端消息
            PublishClientMessage()
            
            // 发布连接状态变更
            PublishConnectionState()
        }
    }
    
    // ====== 管理层 ======
    ManagementLayer struct {
        // 心跳管理
        HeartbeatManager struct {
            // 定时发送 Ping
            SendPing()
            // 检测 Pong 超时
            CheckTimeout()
            // 清理死连接
            CleanDeadConnections()
        }
        
        // 认证管理
        AuthManager struct {
            // 验证 JWT Token
            ValidateToken(token string) (*Claims, error)
            // 检查设备是否被踢
            CheckDeviceStatus(uid, deviceID string) bool
        }
        
        // 监控管理
        MetricsManager struct {
            // 连接数统计
            ConnectionCount()
            // 消息量统计
            MessageCount()
            // 延迟统计
            LatencyStats()
        }
    }
}
```

## 五、核心流程

### 5.1 客户端连接流程

```
客户端连接流程：

1. 客户端发起 WebSocket 连接
   ws://connector:8081/ws?token={jwt_token}

2. Connector 验证 Token
   → 解析 JWT，获取 uid, device_id
   → 验证 Token 有效性
   → 检查设备是否被踢

3. 创建 Client 实例
   → 生成 connID
   → 封装 WebSocket 连接
   → 设置初始状态

4. 注册到 ChannelManager
   → 添加到 userClients（uid → []*Client）
   → 添加到 clients（connID → *Client）

5. 发布连接状态变更
   → NATS: im.connector.online {uid, device_id, node_id}

6. 启动心跳
   → 定时发送 Ping
   → 等待 Pong 响应

7. 开始接收客户端消息
   → 读取 WebSocket 消息
   → 验证消息格式
   → 发布到 NATS: im.client.message
```

### 5.2 消息推送流程

```
消息推送流程（Push Service 调用）：

1. Push Service 发布 RPC 请求
   → NATS RPC: im.connector.push
   → Payload: {user_id: "userA", message: {...}}

2. Connector 接收请求
   → 查找 ChannelManager
   → GetUserClients("userA")
   → 找到该用户的所有设备连接

3. 遍历设备连接
   → 对每个设备：
     → 检查连接是否存活
     → 序列化消息
     → 通过 WebSocket 发送

4. 返回结果
   → {success: true, reason: "online", device_count: 2}
   → 或 {success: false, reason: "offline"}
```

### 5.3 客户端消息上行流程

```
客户端消息上行流程：

1. 客户端通过 WebSocket 发送消息
   → {type: "message", chat_id: "xxx", content: {...}}

2. Connector 接收消息
   → 验证消息格式
   → 补充元数据（uid, device_id, client_time）

3. 发布到 NATS
   → NATS: im.client.message
   → Payload: {uid, device_id, chat_id, content, client_time}

4. 其他服务消费
   → Message Service 消费（存储）
   → 或直接由 Gateway 处理（取决于架构设计）

注意：Connector 不处理消息内容，只负责转发。
```

## 六、关键设计决策

### 6.1 为什么不缓存用户信息？

```
❌ Connector 不缓存用户信息

原因：
  · Connector 是连接层，不是业务层
  · 用户信息由 User Service 管理
  · Connector 只需要知道 uid 和 device_id
  · 保持无状态，方便水平扩展
```

### 6.2 为什么不判断在线状态？

```
❌ Connector 不对外提供在线状态判断

原因：
  · Connector 只知道自己的节点上的连接
  · 用户可能连接在其他节点
  · 全局在线状态由 Redis 或专门的 Presence Service 管理

Connector 只提供：
  ✅ 本节点的连接查询
  ✅ 本节点的推送执行
```

### 6.3 为什么不处理离线消息？

```
❌ Connector 不处理离线消息

原因：
  · 离线消息是业务逻辑
  · Push Service 负责离线推送
  · Connector 只管"在线"时的推送

Connector 只做：
  ✅ 推送成功 → 返回成功
  ✅ 用户不在本节点 → 返回失败
  ✅ 后续由 Push Service 处理
```

## 七、对外接口

### 7.1 入站接口

```go
// ====== 客户端连接 ======
// WebSocket 连接
GET ws://connector:8081/ws?token={jwt_token}

// ====== 服务端推送（RPC） ======
// 推送消息给用户
NATS RPC: im.connector.push
Request: {
    "user_id": "userA",
    "payload": {...}  // 要推送的数据
}
Response: {
    "success": true,
    "reason": "online",      // online/offline/not_found
    "device_count": 2        // 成功推送的设备数
}

// ====== 踢出设备（RPC） ======
NATS RPC: im.connector.kickout
Request: {
    "user_id": "userA",
    "device_id": "device_xxx",  // 可选，不传则踢出所有设备
    "reason": "设备被踢出"
}
```

### 7.2 出站事件

```go
// ====== 连接状态变更 ======
// 用户上线
NATS Publish: im.connector.online
{
    "user_id": "userA",
    "device_id": "device_xxx",
    "platform": "ios",
    "node_id": "connector-1",
    "timestamp": "2026-07-07T10:00:00Z"
}

// 用户下线
NATS Publish: im.connector.offline
{
    "user_id": "userA",
    "device_id": "device_xxx",
    "node_id": "connector-1",
    "timestamp": "2026-07-07T10:05:00Z"
}

// ====== 客户端消息上行 ======
NATS Publish: im.client.message
{
    "user_id": "userA",
    "device_id": "device_xxx",
    "chat_id": "single_userA_userB",
    "msg_type": "text",
    "content": {...},
    "client_time": "2026-07-07T10:00:00Z"
}
```

## 八、部署和扩展

### 8.1 多节点部署

```
┌─────────────────────────────────────────────────────────┐
│                     Load Balancer                       │
│                   (IP Hash/一致性哈希)                    │
└────────┬───────────────┬───────────────┬────────────────┘
         │               │               │
    ┌────▼────┐     ┌────▼────┐     ┌────▼────┐
    │Connector│     │Connector│     │Connector│
    │  Node-1 │     │  Node-2 │     │  Node-3 │
    └────┬────┘     └────┬────┘     └────┬────┘
         │               │               │
         └───────────────┼───────────────┘
                         │
                   ┌─────▼─────┐
                   │   NATS    │
                   └───────────┘

关键设计：
  · 客户端连接到哪个节点由负载均衡决定
  · 用户可能连接在不同节点
  · 所有节点都订阅 NATS 推送请求
  · NATS QueueSubscribe 保证消息只被一个节点处理
```

### 8.2 节点路由

```go
// Push Service 调用 Connector 推送时
// 不需要知道用户连接在哪个节点

// NATS RPC 自动路由
// 所有 Connector 节点都订阅了 im.connector.push
// NATS 会将请求随机发送给一个节点（QueueSubscribe）

// 该节点查找本地连接：
//   - 找到 → 推送成功
//   - 未找到 → 返回失败

// 如果要精确路由到用户所在节点：
// 方案1：Redis 记录用户→节点映射
// 方案2：NATS 的 QueueSubscribe 已经够用（性能足够）
```

## 九、与其他服务的边界

```
┌─────────────────────────────────────────────────────────────┐
│                    Connector Service                        │
│                                                             │
│  拥有：                                                       │
│    ✅ WebSocket/TCP 连接实例                                  │
│    ✅ 用户→连接映射（内存）                                     │
│    ✅ 连接状态（内存）                                         │
│                                                             │
│  不拥有：                                                     │
│    ❌ 消息内容                                                │
│    ❌ 用户信息                                                │
│    ❌ 业务数据                                                │
│                                                             │
│  关系：                                                       │
│    ← Push Service：接收推送请求（RPC）                         │
│    ← Auth Service：验证连接 Token                             │
│    → NATS：发布连接状态变更事件                                │
│    → NATS：上行客户端消息                                     │
│    → 客户端：推送消息                                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## 十、总结

```
┌─────────────────────────────────────────────────────────┐
│              Connector Service 服务定位总结                │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  一句话：只负责"连接"和"推送"，不关心"内容"和"决策"       │
│                                                         │
│  核心原则：                                              │
│    📌 透明管道：只传输数据，不关心内容                    │
│    📌 无状态：不存储业务数据，方便水平扩展                │
│    📌 极简接口：只提供连接管理和消息推送两个能力          │
│    📌 被动推送：不主动决策，只执行推送指令                │
│                                                         │
│  完整流程：                                              │
│    客户端连接 → Connector 管理连接                       │
│    Push Service 推送 → Connector 执行推送               │
│    客户端上行 → Connector 转发到 NATS                   │
│                                                         │
│  一句话：Connector 就是 IM 系统的"电话线"。              │
│          它只管接通和传输，不管通话内容。                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

Connector 的设计哲学就是"极简"。它像一个邮递员，只管送信，不看信的内容，不决定信该送给谁，也不管收信人不在家怎么办（那是 Push Service 的事）。这种极简设计让它性能极高、扩展容易、维护简单。