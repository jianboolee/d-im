# D-IM 架构重构会话总结

更新日期：2026-07-18

## 1. 本次会话目标

本次工作围绕长期可维护的领域边界，对 SDK、用户同步、Gateway 响应、ID、Chat、Conversation、Message 和可靠投影进行系统性调整。

核心目标是明确：

- `Chat` 是承载消息的会话实体。
- `Conversation` 是某个用户对 Chat 的个性化视图。
- Message 直接发送到 Chat，不通过 Conversation 间接定位消息实体。
- Conversation 的衍生状态由可靠投影维护，而不是散落在各业务服务中直接写数据库。
- Group、User、Message、Conversation 各自保持独立领域职责。

本次调整不保留旧接口和旧数据兼容逻辑，按目标架构直接收敛。

## 2. SDK 与模块结构

Go SDK 和 Demo 已从 `backend` 中移出，成为独立模块：

```text
sdk/go
├── go.mod
├── client
└── demo
```

根目录没有增加 `go.work`。Backend 和 Go SDK 可以分别构建和发布，避免 SDK 对 Backend 内部包产生隐式依赖。

## 3. 用户信息同步

用户同步由 NATS 订阅模式调整为业务系统主动 HTTP 调用：

```http
PUT /api/v1/management/users/{userID}
X-API-Key: <api-key>
```

业务系统通过 Go SDK 的 `IMClient.UpsertUser` 提交用户快照。IM 系统不再订阅外部用户变化事件。

主要决策：

- 用户 ID 完全由第三方业务系统提供。
- IM 不生成内部用户 ID，也不增加 namespace。
- 用户快照带版本号，防止旧请求覆盖新数据。
- 不兼容旧 NATS 用户同步接口。

## 4. Gateway 响应规范

Gateway JSON 响应已集中到统一 HTTP API 包，统一处理：

- 成功响应结构
- 错误响应结构
- 错误码
- Content-Type
- JSON 编码

Handler 不再各自维护含义重复的 `writeJSON` 实现。相关代码位于：

```text
backend/internal/gateway/httpapi
```

## 5. ID 与命名规范

实体 ID 已统一使用无前缀 UUID v7，包括 Message、Conversation、Chat 等实体。

主要决策：

- ID 生成从 Service 层移到模型或领域工厂。
- 删除雪花 ID 依赖。
- 不再使用 `msg_`、`chat_`、`conv_` 等前缀。
- ID 只负责全局唯一和时间有序，不承载实体类型语义。
- 数据库唯一索引仍是最终一致性约束。

单聊唯一键已从 Repository 移到 Chat 领域模型。第三方用户 ID 被视为不透明字符串，不能依赖其数值含义或业务排序规则。

单聊键生成规则：

1. 按原始字节进行稳定排序。
2. 使用长度前缀消除拼接歧义。
3. 使用 SHA-256 生成固定长度唯一键。
4. 不进行大小写、空白或 Unicode 归一化。

## 6. Chat 与 Conversation 领域边界

最终边界如下：

### Chat

Chat 是消息流的物理实体，负责：

- Chat 生命周期
- Chat 类型
- 成员定位入口
- 消息序列分配
- 单聊幂等创建
- 群聊消息载体创建

### Conversation

Conversation 是用户级读模型，负责：

- 用户会话列表
- 置顶
- 免打扰
- 已读位置
- 未读数
- 最后一条消息摘要
- 用户是否已离开该 Chat

Conversation 不负责创建 Chat，也不作为消息发送入口。

### Group

Group 是独立业务聚合，负责：

- 群资料
- 群成员
- Owner/Admin 角色
- 加群和退群规则
- 群权限

Group 通过 `chat_id` 关联用于承载消息的 Chat，不把 Group 资料塞入 Chat。

## 7. ChatService 与消息发送

新增 ChatService，Repository 只负责持久化：

- `EnsureSingleChat`
- `CreateGroupChat`
- `GetChat`

消息发送接口只接收 `chat_id`。MessageService 会加载 Chat，并根据 Chat 类型完成权限验证和接收者解析。

当前规则：

- Single：发送者必须属于 `Chat.Members`，接收者从成员中推导。
- Group：通过 GroupService 检查权限，接收者从当前群成员中推导。
- 客户端不能自行提交 `chat_type` 或目标用户列表。
- Message 模型的 `chat_type` 来自服务端加载的 Chat。

旧的 Conversation 发送回退逻辑已经移除。

## 8. 单聊公共 API

单聊创建入口已迁移到 Chat 领域：

```http
POST /api/v1/chats/single
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "peer_user_id": "user-b"
}
```

接口返回 Chat，而不是混合用户视图字段的 Conversation。

旧接口已删除：

```http
POST /api/v1/conversations/single
```

Go SDK 使用：

```go
chat, err := client.EnsureSingleChat(ctx, accessToken, peerUserID)
```

Web SDK 使用 `ensureSingleChat`。如果 UI 需要置顶、未读数、展示名等用户视图字段，再通过 `chat_id` 查询 Conversation。

## 9. Conversation Repository 与 Projector

原 `pkg/model.ConversationManager` 同时包含领域、MongoDB、游标和投影职责，现已删除。

新的结构为：

```text
internal/conversation
├── repository   # Conversation 持久化
├── projector    # 将业务事实应用为用户视图
├── outbox       # 可靠事件存储与消费
├── repair       # 从权威数据修复视图
└── service      # 用户主动查询、设置和已读操作
```

ConversationService 只保留：

- 列表和详情查询
- 按 Chat 查询用户视图
- 置顶和免打扰
- 已读位置推进

旧 `SyncService` 已删除。

## 10. 可靠异步投影

Conversation 投影采用 MongoDB Transactional Outbox。

业务写与 `conversation_outbox` 事件在同一个 MongoDB 事务中提交，覆盖：

- 单聊 Chat 创建
- 群创建
- 成员加入
- 成员退出或被移除
- Message 创建

Gateway 内的后台 Worker 负责：

- 持久化领取事件
- 连续消费积压
- 失败退避重试
- Worker 崩溃后的超时任务回收
- 20 次失败后转为 `failed`
- 成功事件保留 7 天后由 TTL 清理

消息投影按 `message.seq` 幂等推进：

- 重放不会重复增加消息计数。
- 旧消息不会覆盖更新的最后消息。
- 已经应用的事件可以安全重复执行。

详细说明见 [conversation-projection.md](./conversation-projection.md)。

## 11. 重放与修复工具

### 重放保留期内的 Outbox 事件

```bash
cd backend
go run ./cmd/conversation-outbox-replay \
  -config configs/config.yaml \
  -chat-id <chat-id>
```

省略 `-chat-id` 会重放全部仍被保留的事件。

### 从权威数据修复 Conversation

```bash
cd backend
go run ./cmd/conversation-projection-repair \
  -config configs/config.yaml \
  -chat-id <chat-id>
```

修复来源包括：

- Chat
- 当前有效群成员
- 最新 Message

修复会保留用户置顶、免打扰和已有已读位置，并将已经不属于群的用户视图标记为离开。

## 12. MongoDB 运行要求

可靠 Outbox 要求 MongoDB 支持事务，因此开发环境已经从 standalone 调整为单节点 replica set：

```text
replicaSet=rs0
```

Docker Compose 增加了：

- `mongod --replSet rs0`
- MongoDB 健康检查
- 自动执行 `rs.initiate` 的 `mongo-init` 服务

关键业务写使用 `WithRequiredTransaction`，不会在 standalone 环境静默降级为顺序写。

启动 Gateway 前必须确认 replica set 初始化完成。Gateway 启动时会确保 Outbox 等集合索引存在。

## 13. Channel 决策

`ChatTypeChannel` 当前只作为未来概念占位，不做实现。

明确约束：

- 当前 MessageService 只支持 `single` 和 `group`。
- 不使用 Group 成员模型模拟 Channel。
- 不为 Channel 增加模型、Repository、Service、API 或投影事件。
- 遇到 `chat_type=channel` 继续返回 unsupported chat type。

未来实现 Channel 时，应建立独立的 `Channel` 和 `ChannelSubscription` 聚合，将其定义为发布/订阅消息流，而不是大群聊。

## 14. 已完成验证

本次各阶段执行过以下验证：

```bash
cd backend && go test ./...
cd sdk/go && go test ./...
cd web && npm run build
docker compose -f backend/deployments/docker-compose/docker-compose.dev.yaml config
git diff --check
```

验证覆盖：

- Backend 全量编译和测试
- Chat 领域规则
- MessageService 当前 API 契约
- Outbox 事件分派
- Go SDK HTTP 契约
- Web TypeScript 类型检查
- Web 生产构建
- Docker Compose 配置解析

## 15. 后续建议

建议后续按以下优先级继续：

1. 增加 MongoDB 集成测试，真实验证业务写和 Outbox 的事务原子性。
2. 增加 Outbox 指标和告警：pending 数量、最老事件年龄、failed 数量、重试次数。
3. 将 Outbox Worker 从 Gateway 进程拆成独立 Worker 服务，以支持独立扩缩容和故障隔离。
4. 审视现有内存 Message Dispatcher。当前队列满时会丢任务，不具备 Outbox 同等级别的可靠性。
5. 增加定期 Conversation 一致性巡检，而不仅依赖人工 repair 命令。
6. 在真正启动 Channel 前先确定发布、订阅、评论和 Conversation 生成规则。

## 16. 最终架构结论

当前主链路已经收敛为：

```text
业务操作
  -> Chat / Group / Message 权威数据写入
  -> 同事务写入 Conversation Outbox
  -> Worker 消费业务事实
  -> Conversation Projector 幂等更新用户视图
  -> ConversationService 提供用户查询和个性化操作
```

这条链路明确区分了消息实体、业务聚合和用户读模型，为后续扩展新的 Chat 类型、独立 Worker 和一致性治理保留了清晰边界。
