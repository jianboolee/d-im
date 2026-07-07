# 群聊模块落地结构设计

本文档定义群聊模块的目标目录结构、每个文件的职责边界，以及后续落地顺序。后续实现应先对照本文档确认职责归属，再修改代码。

## 设计目标

- `chat` 只表示消息会话实体，不承载群资料、成员、角色、公告和权限。
- `group` 是独立领域聚合，拥有自己的资料、设置、成员、角色和事件。
- `model` 包只定义实体、常量和纯函数，不直接操作 MongoDB 或其他外部存储。
- 数据访问统一放在 repository 中，领域用例统一由 service 编排。
- `handler` 只做 HTTP 适配，不拼系统事件文案，不做业务副作用。
- `service` 承载用例编排和业务规则；仓储只做数据访问。
- 重要写路径需要明确事务边界，不能留下半成品数据。

## 目标目录结构

```text
backend/internal/group/
  service/
    group_service.go
    member_service.go
    permission_service.go
    event_publisher.go
    avatar_service.go
    ports.go
    errors.go
    types.go

  repository/
    group_repo.go
    member_repo.go

  avatar/
    generator.go
    generator_test.go
```

```text
backend/internal/chat/
  repository/
    chat_repo.go
```

```text
backend/internal/gateway/handler/
  group_handler.go
  group_request.go
  group_response.go
```

```text
backend/pkg/model/
  chat.go
  group.go
  group_member.go
```

```text
backend/cmd/migrate/
  main.go
```

```text
backend/cmd/group/
  main.go
```

## 文件职责

### `backend/pkg/model/chat.go`

职责：

- 定义消息会话实体 `Chat`。
- 定义 `ChatType`、`GenerateChatID` 等消息会话相关常量和纯函数。
- 维护 `chat_id`、`chat_type`、单聊成员、单聊幂等键、消息序号等字段定义。

不做：

- 不直接访问 MongoDB。
- 不提供 `CreateGroupChat`、`NextChatMessageSeq`、`FindChatByID` 这类持久化函数。
- 不保存群名称、群头像、群公告、群主、管理员、群设置。
- 不保存群成员列表。
- 不判断群权限。

群聊约束：

- 群聊场景下 `Chat.Members` 和 `Chat.MemberCount` 不作为成员事实源。
- 群成员统一以 `group_members` 集合为准。
- `Chat.Members` 仅服务单聊，群聊创建时不维护该字段。

迁移要求：

- 当前 `pkg/model/chat.go` 中已有的 MongoDB 操作应迁移到 `internal/chat/repository/chat_repo.go`。

### `backend/pkg/model/group.go`

职责：

- 定义群聚合根 `Group`。
- 定义 `GroupSettings`、`JoinMethod`、`GroupStatus`。
- 提供新群默认配置，例如公开、可邀请、默认成员上限。
- 群默认权限为公开且可邀请。

不做：

- 不定义成员角色的持久化细节。
- 不包含消息序号、会话列表设置等非群资料字段。

字段策略：

- `Group` 使用 `ChatID` 作为群的稳定主键。
- 不再长期保留与 `ChatID` 永远相等的 `GroupID` 字段。
- 如果迁移期需要兼容旧数据，`GroupID` 只能作为过渡字段，并需要在迁移和清理计划中明确删除。
- `Group.Admins` 是兼容字段，角色以 `GroupMember.Role` 为准。迁移期只读不写，最终在清理阶段删除。
- 禁言状态不放在 `GroupSettings.MutedMembers` 中维护。

### `backend/pkg/model/group_member.go`

职责：

- 定义 `GroupMember`。
- 定义 `MemberRole`。
- 承载成员维度状态：角色、群昵称、禁言、入群时间、邀请人。
- 禁言状态以 `GroupMember.IsMuted` 和 `GroupMember.MutedUntil` 为唯一事实源。

不做：

- 不承载群名称、群公告、群设置。
- 不承载用户全局资料，用户资料只可做必要冗余或查询时组装。

禁言策略：

- 读取禁言状态时只判断 `GroupMember.IsMuted` 和 `GroupMember.MutedUntil`。
- `MutedUntil` 到期后视为未禁言；后台清理可以异步修正 `IsMuted`。
- `GroupSettings.MutedMembers` 应废弃；迁移期只能作为旧字段读取，不再新增写入。

### `backend/internal/chat/repository/chat_repo.go`

职责：

- 封装 `chats` 集合读写。
- 创建单聊/群聊消息会话实体。
- 查询 `Chat`。
- 使用原子更新分配消息序号。
- 管理单聊幂等键和必要索引依赖。

不做：

- 不保存群资料。
- 不查询或维护群成员。
- 不判断群权限。

### `backend/internal/group/repository/group_repo.go`

职责：

- 封装 `groups` 集合读写。
- 提供按 `chat_id` 查询、创建、更新资料、更新设置、解散、更新成员数、头像为空时回写等原子操作。
- 只接收明确字段，不做 HTTP 请求解析，不拼业务文案。

不做：

- 不创建会话视图。
- 不写 `group_members`。
- 不发布事件。

### `backend/internal/group/repository/member_repo.go`

职责：

- 封装 `group_members` 集合读写。
- 提供成员添加、删除、查询、分页、批量 UID 查询、按用户查询群列表、角色变更、禁言状态更新。
- 维护成员唯一约束：`chat_id + uid`。

不做：

- 不更新 `groups.member_count`。
- 不创建会话视图。
- 不发布事件。

### `backend/internal/group/service/group_service.go`

职责：

- 群生命周期和群资料用例：
  - 创建群
  - 获取群详情
  - 当前用户群列表
  - 更新群资料
  - 更新群设置
  - 设置公告
  - 解散群
- 编排 `Chat` 创建、`Group` 创建、初始成员创建、初始会话视图创建。
- 在用例成功后调用事件发布器和头像服务。
- 通过 `ChatRepository` 或 `ChatPort` 创建群聊消息会话，不直接调用 model 层 DB 函数。

不做：

- 不直接实现成员邀请、踢人、角色变更等成员用例。
- 不拼系统消息文案。
- 不访问 HTTP request/response。

### `backend/internal/group/service/member_service.go`

职责：

- 群成员、角色、成员权限用例：
  - 加入群
  - 邀请成员
  - 退出群
  - 踢出成员
  - 成员列表
  - 批量成员 UID 查询
  - 设置成员角色
  - 转让群主
  - 成员禁言
- 负责维护 `groups.member_count` 与 `group_members` 的一致性。
- 负责成员变更后创建或标记会话视图。
- 自由加入群属于成员用例，但群加入策略检查委托给 `permission_service.go`。

不做：

- 不更新群名称、头像、公告等群资料。
- 不拼 HTTP 响应 DTO。

### `backend/internal/group/service/permission_service.go`

职责：

- 集中实现群权限判断。
- 判断操作是否允许：
  - 发消息
  - 邀请成员
  - 自由加入
  - 踢人
  - 更新群资料
  - 更新设置
  - 转让群主
  - 设置角色
- 输出稳定的拒绝原因，供 Gateway/Message Service 映射错误。

不做：

- 不修改数据库。
- 不发布事件。

输出结构：

```go
type PermissionResult struct {
    Allowed bool
    Reason  string
}
```

`Reason` 使用稳定枚举值，例如：

- `not_group_member`
- `group_muted_all`
- `member_muted`
- `owner_required`
- `admin_required`
- `join_not_allowed`
- `group_full`

权限动作类型：

```go
type PermissionAction string

const (
    ActionSendMessage     PermissionAction = "send_message"
    ActionInviteMember    PermissionAction = "invite_member"
    ActionJoinGroup       PermissionAction = "join_group"
    ActionKickMember      PermissionAction = "kick_member"
    ActionUpdateGroupInfo PermissionAction = "update_group_info"
    ActionUpdateSettings  PermissionAction = "update_settings"
    ActionDismissGroup    PermissionAction = "dismiss_group"
    ActionTransferOwner   PermissionAction = "transfer_owner"
    ActionSetMemberRole   PermissionAction = "set_member_role"
)
```

### `backend/internal/group/service/event_publisher.go`

职责：

- 定义群领域事件类型。
- 将群领域事件发布到系统事件通道。
- 当前阶段通过 `SystemEventPort` 发送 `system_event` 消息；后续可替换为 NATS 领域事件。
- 统一维护系统事件文案。
- 明确依赖方向为 `event_publisher -> SystemEventPort -> message service`，不能直接依赖 Message Service 具体类型。
- 接收结构化事件数据（非 `map[string]any`），内部根据事件类型生成文案。

事件数据结构：

```go
type GroupSystemEvent struct {
    EventType   string   // GroupCreated / MembersInvited / ...
    OperatorUID string
    TargetUIDs  []string
    GroupID     string
    GroupName   string
    BeforeValue string
    AfterValue  string
}
```

典型事件类型：

- `GroupCreated`
- `GroupDismissed`
- `GroupInfoUpdated`
- `MembersInvited`
- `MemberJoined`
- `MemberLeft`
- `MemberKicked`
- `MemberRoleChanged`
- `OwnerTransferred`
- `AnnouncementUpdated`

不做：

- 不解析 HTTP 请求。
- 不执行群业务规则。

### `backend/internal/group/service/avatar_service.go`

职责：

- 调度群头像异步生成。
- 通过 `MemberRepository` 或只读 `MemberQueryPort` 读取成员 UID。
- 调用 `avatar.Generator` 生成九宫格头像。
- 头像为空时回写 `groups.avatar`。

不做：

- 不参与创建群的主事务。
- 不阻塞主请求。
- 不依赖完整的 `member_service.go`，避免头像生成反向占用成员用例服务。

异步安全：

- `avatar_service` 的异步 goroutine 必须包含 `recover` 保护，防止 panic 导致进程崩溃。
- 长期方案可使用统一 worker pool 或任务队列替代裸 goroutine。

### `backend/internal/group/service/ports.go`

职责：

- 定义 group service 需要依赖的外部能力接口。
- 定义其他模块访问 group 域的只读或权限接口。
- 避免 group、message、conversation 之间直接依赖具体实现。

建议接口：

```go
type ConversationPort interface {
    BatchCreate(ctx context.Context, uidList []string, chat *model.Chat) error
    CreateOrUpdate(ctx context.Context, conv *model.Conversation) error
    MarkLeft(ctx context.Context, uid, chatID string) error
}

type SystemEventPort interface {
    SendGroupSystemEvent(ctx context.Context, event GroupSystemEvent) error
}

type GroupQueryPort interface {
    GetGroup(ctx context.Context, chatID string) (*model.Group, error)
    GetMember(ctx context.Context, chatID, uid string) (*model.GroupMember, error)
    GetMemberUIDs(ctx context.Context, chatID string) ([]string, error)
}

type GroupPermissionPort interface {
    CheckPermission(ctx context.Context, chatID, uid string, action PermissionAction) (PermissionResult, error)
}
```

不做：

- 不放具体业务实现。
- 不引入 gateway handler 类型。

设计权衡：

按照 Go 惯用法，“接口由使用方定义”，`GroupQueryPort` 和 `GroupPermissionPort`
的理想位置是 conversation 域和 message 域各自的包内。当前方案选择集中在 group
域定义所有 port，主要是为了：

- 所有 port 在一处可见，便于理解模块间契约全貌。
- 避免 conversation 和 message 各自重复定义相似的 group 查询接口。
- 实现方（group service）与接口定义在同一包内，减少包引用层级。

如果项目规模增长导致 group 包过于臃肿，可将 `GroupQueryPort` 和
`GroupPermissionPort` 分拆到 `backend/internal/group/port/` 子包，
或由消费方各自定义。

### `backend/internal/group/service/errors.go`

职责：

- 定义群服务错误：
  - `ErrForbidden`
  - `ErrInvalid`
  - `ErrGroupFull`
  - `ErrOwnerRequired`
  - `ErrMemberNotFound`
- 保持 service 层错误稳定，handler 只做错误到 HTTP 的映射。

### `backend/internal/group/service/types.go`

职责：

- 定义 service 层请求/结果结构。
- 避免 service 方法参数越来越长。

示例：

```go
type CreateGroupInput struct {
    Name      string
    OwnerUID  string
    MemberUIDs []string
}

type InviteMembersResult struct {
    Group       *model.Group
    AddedUIDs   []string
    SkippedUIDs []string
}
```

### `backend/internal/gateway/handler/group_handler.go`

职责：

- 注册和实现 HTTP handler 方法。
- 从 request 中读取参数。
- 获取当前用户 ID。
- 调用 group/member service。
- 调用 response mapper 输出 JSON。

不做：

- 不拼系统事件。
- 不判断群角色权限。
- 不直接访问 repository。
- 不直接调用 Message Service 发送系统消息。

### `backend/internal/gateway/handler/group_request.go`

职责：

- 定义群接口请求结构。
- 做基础字段解析和浅层校验。

示例：

- `createGroupRequest`
- `updateGroupRequest`
- `inviteMembersRequest`
- `updateSettingsRequest`
- `setMemberRoleRequest`
- `transferOwnerRequest`

### `backend/internal/gateway/handler/group_response.go`

职责：

- 定义群接口响应 DTO。
- 将 `model.Group`、`model.GroupMember`、`model.User` 组装成前端需要的结构。
- 统一处理 `conversation_id`、`user_info`、时间格式。

不做：

- 不执行业务规则。
- 不访问 repository。

### `backend/internal/group/avatar/generator.go`

职责：

- 纯图片生成逻辑。
- 根据成员头像或 fallback 色块生成九宫格头像。
- 不知道群业务规则。

### `backend/cmd/migrate/main.go`

职责：

- 执行数据库结构迁移和幂等回填。
- 创建索引。
- 将旧 `chats` 中的群字段迁移到 `groups` 和 `group_members`。

不做：

- 不承载在线业务逻辑。
- 不被 API Gateway 运行时调用。

### `backend/cmd/group/main.go`

职责：

- group 服务独立进程入口。
- 组装 group service、repository、事件发布、头像生成等依赖。
- 加载配置并启动 group 相关传输层。

不做：

- 不承载群业务规则。
- 不直接访问 handler 内部实现。

## 服务依赖方向

```text
handler
  -> group service / member service
      -> repository
      -> conversation port
      -> event publisher
      -> avatar service

event publisher
  -> system event port
      -> message service

message service
  -> group permission port / group query port

conversation handler
  -> group query port
```

禁止反向依赖：

- `group service` 不依赖 HTTP handler。
- `repository` 不依赖 service。
- `model` 不依赖 internal 包。
- `event_publisher` 不依赖 Message Service 具体类型，只依赖 `SystemEventPort`。
- `avatar_service` 不依赖完整 `member_service`，只依赖成员只读查询能力。

## 数据事实源

### 群主键

- 目标模型使用 `ChatID` 作为群的稳定主键。
- `GroupID` 与 `ChatID` 如果永远相同，就不作为长期字段保留。
- 迁移期如需读取 `GroupID`，必须保证新写入以 `ChatID` 为准。

### 群成员

- 群成员唯一事实源是 `group_members`。
- 群聊 `Chat.Members` 和 `Chat.MemberCount` 不参与群成员维护。
- `groups.member_count` 是 `group_members` 的派生计数字段，由成员用例在事务内维护。

### 禁言

- 成员禁言唯一事实源是 `GroupMember.IsMuted` 和 `GroupMember.MutedUntil`。
- 全员禁言属于群设置，基于 `GroupSettings.IsMutedAll`。
- 不再使用 `GroupSettings.MutedMembers` 表达成员禁言。

### 管理员

- 管理员角色以 `GroupMember.Role == MemberRoleAdmin` 为准。
- `Group.Admins` 是兼容字段，迁移期只读不写，最终在清理阶段删除。

## 事务边界

需要事务的写路径：

- 创建群：`chats`、`groups`、`group_members`、`conversations`
- 邀请成员：`group_members`、`groups.member_count`、`conversations`
- 退出/踢人：`group_members`、`groups.member_count`、`conversations.left_at`
- 解散群：`groups.status`、所有成员会话状态
- 转让群主：旧 owner role、新 owner role、`groups.owner_uid`

头像生成和系统事件发送不进入主事务。主事务成功后再异步或 best-effort 发布。

## 落地顺序

1. 新建 `internal/chat/repository/chat_repo.go`，把 `pkg/model/chat.go` 中的 MongoDB 操作迁移过去。
2. 把 `GroupMember` 和 `MemberRole` 从 `pkg/model/group.go` 拆分到 `pkg/model/group_member.go`。
3. 拆 service 文件和接口（`group_service.go` / `member_service.go` / `permission_service.go` / `errors.go` / `types.go`），不改业务行为，通过 port/repository 调用。
4. 把系统事件从 handler 移到 `event_publisher.go`。
5. 引入事务封装，覆盖关键写路径。
6. 拆 handler request/response 文件。
7. 补单元测试和 handler/service 集成测试。
8. 最后清理旧 helper、旧字段（`GroupID`、`Admins`、`MutedMembers`）和过渡逻辑。

## 验收标准

- `pkg/model` 只包含实体、常量和纯函数，不包含 MongoDB 操作。
- `group_handler.go` 只保留 HTTP 方法，不包含系统事件文案。
- `member_service.go` 不为空，并拥有成员相关用例。
- `permission_service.go` 返回稳定的 `PermissionResult`。
- Message Service 只通过 group port 查询成员和权限。
- Conversation Handler 只通过 group query port 组装群会话信息。
- 群成员只以 `group_members` 为准，群聊不维护 `Chat.Members`。
- 成员禁言只以 `GroupMember.IsMuted` 和 `GroupMember.MutedUntil` 为准。
- `go test ./...` 通过。
- 旧数据可以通过 `cmd/migrate` 幂等迁移。