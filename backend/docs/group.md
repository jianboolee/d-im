群组服务是 IM 系统中相对独立且功能丰富的模块。让我系统梳理它的职责和边界。

## 一、Group Service 的服务定位

### 1.1 一句话定义

**Group Service 是群组的生命周期管理中心，负责群组的创建、成员管理、权限控制和群组信息维护。**

### 1.2 核心职责

```go
// Group Service 的核心职责
type GroupServiceResponsibilities struct {
    // ====== 核心职责 ======
    
    // 1. 群组生命周期管理
    LifecycleManagement string // 创建、解散、激活、归档
    
    // 2. 成员管理
    MemberManagement string // 加入、退出、踢出、邀请
    
    // 3. 角色和权限管理
    RoleManagement string // 群主、管理员、普通成员
    
    // 4. 群组信息管理
    InfoManagement string // 群名称、头像、公告、群设置
    
    // 5. 群组查询
    QueryService string // 群信息查询、成员列表查询
    
    // ====== 明确不做的事情 ======
    
    // ❌ 不负责群消息（Message Service 负责）
    // ❌ 不负责消息推送（Push Service 负责）
    // ❌ 不负责会话视图（Conversation Service 负责）
}
```

## 二、服务边界定义

### 2.1 与 Message Service 的边界

```
Group Service：
  ✅ 管理群成员列表
  ✅ 管理群权限（谁可以发言）
  ✅ 提供群成员查询

Message Service：
  ✅ 查询群成员列表（从 Group Service 获取或缓存）
  ✅ 分发消息到群成员信箱
  ✅ 不关心群权限（权限由 Gateway 检查）

交互方式：
  Message Service → RPC 调用 Group Service → 获取群成员列表
  或者：Message Service 缓存群成员列表（性能优化）
```

### 2.2 与 Conversation Service 的边界

```
Group Service：
  ✅ 管理群组实体（Chat 表中的群聊记录）
  ✅ 群信息变更时发布事件

Conversation Service：
  ✅ 消费群信息变更事件
  ✅ 更新用户的会话视图
  ✅ 管理用户维度的群会话设置（置顶、免打扰等）

交互方式：
  Group Service 发布事件 → Conversation Service 消费
```

### 2.3 与 Push Service 的边界

```
Group Service：
  ✅ 群操作通知（被邀请入群、被踢出群等）

Push Service：
  ✅ 消费群操作通知
  ✅ 推送给目标用户

交互方式：
  Group Service 发布事件 → Push Service 消费
```

## 三、完整的功能模块

### 3.1 模块划分

```go
// Group Service 内部模块
type GroupServiceModules struct {
    
    // ====== 群组管理 ======
    GroupManagement struct {
        CreateGroup()       // 创建群组
        DismissGroup()      // 解散群组（仅群主）
        UpdateGroupInfo()   // 更新群信息
        GetGroupInfo()      // 获取群信息
        GetGroupList()      // 获取群列表
    }
    
    // ====== 成员管理 ======
    MemberManagement struct {
        JoinGroup()         // 加入群组
        LeaveGroup()        // 退出群组
        KickMember()        // 踢出成员
        InviteMembers()     // 邀请成员
        GetMemberList()     // 获取成员列表
        GetMemberInfo()     // 获取成员信息
        BatchGetMembers()   // 批量获取成员（Message Service 用）
    }
    
    // ====== 角色和权限 ======
    RoleManagement struct {
        SetRole()           // 设置角色（群主/管理员/普通）
        TransferOwner()     // 转让群主
        CheckPermission()   // 检查权限
    }
    
    // ====== 群设置 ======
    GroupSettings struct {
        GetSettings()       // 获取群设置
        UpdateSettings()    // 更新群设置
        // - 入群验证方式
        // - 全员禁言
        // - 群成员上限
        // - 是否公开群
    }
    
    // ====== 群公告 ======
    Announcement struct {
        SetAnnouncement()   // 设置群公告
        GetAnnouncement()   // 获取群公告
        DeleteAnnouncement()// 删除群公告
    }
    
    // ====== 事件发布 ======
    EventPublisher struct {
        PublishGroupCreated()   // 群创建
        PublishGroupDismissed()  // 群解散
        PublishMemberJoined()   // 成员加入
        PublishMemberLeft()     // 成员退出
        PublishMemberKicked()   // 成员被踢
        PublishGroupInfoUpdated() // 群信息更新
        PublishRoleChanged()    // 角色变更
    }
}
```

## 四、数据模型

### 4.1 群组实体

```go
// pkg/model/group.go

// Group 群组实体
type Group struct {
    // 基础信息
    ChatID      string    `bson:"_id" json:"chat_id"`           // group_{雪花ID}
    Name        string    `bson:"name" json:"name"`             // 群名称
    Avatar      string    `bson:"avatar" json:"avatar"`         // 群头像
    Description string    `bson:"description" json:"description"` // 群简介
    
    // 群主和管理员
    OwnerUID    string    `bson:"owner_uid" json:"owner_uid"`   // 群主
    Admins      []string  `bson:"admins" json:"admins"`         // 管理员列表
    
    // 成员信息
    MemberCount int       `bson:"member_count" json:"member_count"` // 成员数量
    MaxMembers  int       `bson:"max_members" json:"max_members"`   // 成员上限
    
    // 群设置
    Settings    GroupSettings `bson:"settings" json:"settings"`
    
    // 群公告
    Announcement string   `bson:"announcement" json:"announcement"`
    
    // 状态
    Status      GroupStatus `bson:"status" json:"status"`       // active/dismissed
    
    // 时间戳
    CreatedAt   time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

// GroupSettings 群设置
type GroupSettings struct {
    JoinMethod      JoinMethod `bson:"join_method" json:"join_method"`   // 入群方式
    IsMutedAll      bool       `bson:"is_muted_all" json:"is_muted_all"` // 全员禁言
    IsPublic        bool       `bson:"is_public" json:"is_public"`       // 是否公开群
    MutedMembers    []string   `bson:"muted_members" json:"muted_members"` // 被禁言成员
}

// JoinMethod 入群方式
type JoinMethod string
const (
    JoinMethodFree     JoinMethod = "free"     // 自由加入
    JoinMethodVerify   JoinMethod = "verify"   // 需要验证
    JoinMethodInvite   JoinMethod = "invite"   // 仅邀请
    JoinMethodForbidden JoinMethod = "forbidden" // 禁止加入
)

// GroupStatus 群状态
type GroupStatus string
const (
    GroupStatusActive   GroupStatus = "active"   // 正常
    GroupStatusDismissed GroupStatus = "dismissed" // 已解散
)
```

### 4.2 群成员

```go
// GroupMember 群成员
type GroupMember struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    
    ChatID    string    `bson:"chat_id" json:"chat_id"`     // 群ID
    UID       string    `bson:"uid" json:"uid"`             // 用户ID
    
    // 角色
    Role      MemberRole `bson:"role" json:"role"`           // owner/admin/member
    
    // 成员信息（冗余存储，避免查询用户表）
    Nickname  string    `bson:"nickname" json:"nickname"`    // 群内昵称
    Avatar    string    `bson:"avatar" json:"avatar"`        // 头像
    
    // 成员状态
    IsMuted   bool      `bson:"is_muted" json:"is_muted"`   // 是否被禁言
    MutedUntil *time.Time `bson:"muted_until" json:"muted_until"` // 禁言截止时间
    
    // 时间
    JoinedAt  time.Time `bson:"joined_at" json:"joined_at"`
    UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// MemberRole 成员角色
type MemberRole string
const (
    MemberRoleOwner   MemberRole = "owner"   // 群主
    MemberRoleAdmin   MemberRole = "admin"   // 管理员
    MemberRoleMember  MemberRole = "member"  // 普通成员
)
```

### 4.3 MongoDB 索引设计

```go
// Group 集合索引
func (Group) Indexes() []mongo.IndexModel {
    return []mongo.IndexModel{
        {
            Keys:    bson.D{{Key: "_id", Value: 1}},
            Options: options.Index().SetUnique(true),
        },
        {
            Keys: bson.D{{Key: "owner_uid", Value: 1}},
        },
        {
            Keys: bson.D{
                {Key: "status", Value: 1},
                {Key: "created_at", Value: -1},
            },
        },
    }
}

// GroupMember 集合索引
func (GroupMember) Indexes() []mongo.IndexModel {
    return []mongo.IndexModel{
        {
            // 查询用户的所有群
            Keys: bson.D{
                {Key: "uid", Value: 1},
                {Key: "joined_at", Value: -1},
            },
        },
        {
            // 查询群的所有成员
            Keys: bson.D{
                {Key: "chat_id", Value: 1},
                {Key: "role", Value: 1},
            },
        },
        {
            // 唯一索引：一个用户在一个群里只有一条记录
            Keys: bson.D{
                {Key: "chat_id", Value: 1},
                {Key: "uid", Value: 1},
            },
            Options: options.Index().SetUnique(true),
        },
    }
}
```

## 五、核心流程

### 5.1 创建群组

```
创建群组流程：

Client → Gateway → Group Service
                      │
                      ├── 1. 生成 group_id（雪花ID）
                      ├── 2. 创建 Group 文档
                      ├── 3. 批量创建 GroupMember 文档
                      │      - 创建者为 owner
                      │      - 初始成员为 member
                      ├── 4. 发布 GroupCreated 事件
                      │      → Conversation Service 为成员创建会话
                      │      → Push Service 推送给被邀请的成员
                      └── 5. 返回 group_id 给客户端
```

### 5.2 成员加入

```
成员加入流程：

场景A：邀请加入
  Owner/Admin → Gateway → Group Service
                              │
                              ├── 1. 检查权限
                              ├── 2. 检查群成员是否已满
                              ├── 3. 创建 GroupMember 文档
                              ├── 4. 更新 Group.MemberCount
                              ├── 5. 发布 MemberJoined 事件
                              │      → Conversation Service 为新成员创建会话
                              │      → Push Service 推送通知
                              └── 6. 返回结果

场景B：申请加入
  申请人 → Gateway → Group Service
                        │
                        ├── 1. 检查入群方式
                        ├── 2. 自由加入 → 直接加入
                        ├── 3. 需要验证 → 发送验证请求给管理员
                        └── 4. 返回结果
```

### 5.3 成员退出/被踢

```
成员退出/被踢流程：

Client → Gateway → Group Service
                      │
                      ├── 1. 检查权限（踢人需要管理员权限）
                      ├── 2. 删除或标记 GroupMember 文档
                      ├── 3. 更新 Group.MemberCount
                      ├── 4. 发布 MemberLeft/MemberKicked 事件
                      │      → Conversation Service 更新会话状态
                      │      → Push Service 推送通知
                      │      → Message Service 清理 mailbox（可选）
                      └── 5. 返回结果

特殊情况：
  · 群主退出 → 需要转让群主或解散群
  · 最后一人退出 → 自动解散群
```

## 六、API 设计

### 6.1 对外提供的接口

```go
// Group Service 提供的 RPC 接口

service GroupService {
    // ====== 群组管理 ======
    rpc CreateGroup(CreateGroupReq) returns (CreateGroupResp);
    rpc DismissGroup(DismissGroupReq) returns (DismissGroupResp);
    rpc GetGroupInfo(GetGroupInfoReq) returns (GetGroupInfoResp);
    rpc UpdateGroupInfo(UpdateGroupInfoReq) returns (UpdateGroupInfoResp);
    
    // ====== 成员管理 ======
    rpc JoinGroup(JoinGroupReq) returns (JoinGroupResp);
    rpc LeaveGroup(LeaveGroupReq) returns (LeaveGroupResp);
    rpc KickMember(KickMemberReq) returns (KickMemberResp);
    rpc InviteMembers(InviteMembersReq) returns (InviteMembersResp);
    rpc GetMemberList(GetMemberListReq) returns (GetMemberListResp);
    
    // 批量获取群成员（Message Service 调用，高性能）
    rpc BatchGetMembers(BatchGetMembersReq) returns (BatchGetMembersResp);
    
    // ====== 角色管理 ======
    rpc SetMemberRole(SetMemberRoleReq) returns (SetMemberRoleResp);
    rpc TransferOwner(TransferOwnerReq) returns (TransferOwnerResp);
    
    // ====== 权限检查 ======
    rpc CheckPermission(CheckPermissionReq) returns (CheckPermissionResp);
    
    // ====== 群公告 ======
    rpc SetAnnouncement(SetAnnouncementReq) returns (SetAnnouncementResp);
    rpc GetAnnouncement(GetAnnouncementReq) returns (GetAnnouncementResp);
}
```

### 6.2 关键接口设计

```go
// 批量获取群成员（Message Service 高频调用）
type BatchGetMembersReq struct {
    ChatID string `json:"chat_id"`
    // 可选：只返回 UID 列表（不需要完整信息时）
    OnlyUIDs bool   `json:"only_uids"`
}

type BatchGetMembersResp struct {
    ChatID  string   `json:"chat_id"`
    Members []string `json:"members"` // UID 列表
    // 或者返回完整信息
    // Members []MemberInfo `json:"members"`
}

// 权限检查（Gateway 在发送消息前调用）
type CheckPermissionReq struct {
    ChatID    string       `json:"chat_id"`
    UID       string       `json:"uid"`
    Action    string       `json:"action"` // send_message, invite_member, etc.
}

type CheckPermissionResp struct {
    Allowed    bool   `json:"allowed"`
    Reason     string `json:"reason"`     // 拒绝原因
}
```

## 七、事件发布

### 7.1 事件定义

```go
// Group Service 发布的事件

const (
    // 群组生命周期
    SubjectGroupCreated   = "im.group.created"
    SubjectGroupDismissed = "im.group.dismissed"
    SubjectGroupUpdated   = "im.group.updated"
    
    // 成员变更
    SubjectMemberJoined   = "im.group.member.joined"
    SubjectMemberLeft     = "im.group.member.left"
    SubjectMemberKicked   = "im.group.member.kicked"
    
    // 角色变更
    SubjectRoleChanged    = "im.group.role.changed"
    
    // 群公告
    SubjectAnnouncementSet = "im.group.announcement.set"
)

// 群创建事件
type GroupCreatedEvent struct {
    ChatID    string   `json:"chat_id"`
    Name      string   `json:"name"`
    OwnerUID  string   `json:"owner_uid"`
    Members   []string `json:"members"`    // 初始成员 UID 列表
    CreatedAt time.Time `json:"created_at"`
}

// 成员加入事件
type MemberJoinedEvent struct {
    ChatID    string   `json:"chat_id"`
    NewMembers []string `json:"new_members"` // 新加入的成员 UID
    JoinedAt  time.Time `json:"joined_at"`
}
```

### 7.2 事件消费者

```
GroupCreated → Conversation Service
  → 为所有成员创建 Conversation 记录

MemberJoined → Conversation Service
  → 为新成员创建 Conversation 记录

MemberLeft/MemberKicked → Conversation Service
  → 更新退出成员的 Conversation 状态

GroupUpdated → Conversation Service
  → 更新群名称、头像等信息到会话视图

GroupDismissed → Conversation Service
  → 标记所有成员的该会话为已解散

MemberJoined/MemberKicked → Push Service
  → 推送通知给相关用户
```

## 八、性能优化

### 8.1 群成员缓存

```go
// Message Service 高频调用群成员查询
// 需要使用缓存优化

type GroupMemberCache struct {
    redis *redis.Client
}

// 缓存群成员列表
func (c *GroupMemberCache) Set(ctx context.Context, chatID string, members []string) error {
    key := fmt.Sprintf("group_members:%s", chatID)
    data, _ := json.Marshal(members)
    return c.redis.Set(ctx, key, data, 5*time.Minute).Err()
}

// 获取群成员列表
func (c *GroupMemberCache) Get(ctx context.Context, chatID string) ([]string, error) {
    key := fmt.Sprintf("group_members:%s", chatID)
    data, err := c.redis.Get(ctx, key).Bytes()
    if err != nil {
        return nil, err
    }
    var members []string
    json.Unmarshal(data, &members)
    return members, nil
}

// 群成员变更时清除缓存
func (c *GroupMemberCache) Invalidate(ctx context.Context, chatID string) {
    key := fmt.Sprintf("group_members:%s", chatID)
    c.redis.Del(ctx, key)
}
```

### 8.2 分页查询

```go
// 大群成员列表分页
func (s *GroupService) GetMemberList(ctx context.Context, req *GetMemberListReq) (*GetMemberListResp, error) {
    // 按角色排序：群主 → 管理员 → 普通成员 → 按加入时间
    filter := bson.M{"chat_id": req.ChatID}
    
    // 支持游标分页（性能优于 offset）
    if req.Cursor != "" {
        filter["_id"] = bson.M{"$gt": req.Cursor}
    }
    
    opts := options.Find().
        SetSort(bson.D{
            {Key: "role", Value: 1},      // 群主和管理员优先
            {Key: "joined_at", Value: 1}, // 按加入时间
        }).
        SetLimit(req.Limit)
    
    cursor, err := s.memberColl.Find(ctx, filter, opts)
    // ...
}
```

## 九、总结

```
┌─────────────────────────────────────────────────────────┐
│              Group Service 服务定位总结                   │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  核心定位：群组生命周期和成员管理的唯一入口                  │
│                                                         │
│  职责：                                                  │
│    ✅ 群组的创建、解散、信息维护                           │
│    ✅ 成员的加入、退出、踢出、邀请                         │
│    ✅ 角色和权限管理（群主、管理员）                        │
│    ✅ 群设置管理（入群方式、禁言等）                       │
│    ✅ 群公告管理                                         │
│    ✅ 群成员查询（支持批量、分页）                         │
│    ✅ 群事件发布（通知其他服务）                           │
│                                                         │
│  不负责：                                                │
│    ❌ 群消息的存储和分发（Message Service 负责）            │
│    ❌ 消息推送（Push Service 负责）                       │
│    ❌ 用户会话视图（Conversation Service 负责）            │
│    ❌ 用户信息管理（User Service 负责）                    │
│                                                         │
│  与其他服务的关系：                                       │
│    → Message Service：提供群成员列表                     │
│    → Conversation Service：发布群事件，触发会话更新       │
│    → Push Service：发布群事件，触发推送通知              │
│    → User Service：查询用户基础信息（冗余到群成员）       │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

Group Service 就像群聊的"管理员办公室"，负责管理群的存在、成员进出、权限分配，但不负责群里的消息传递（这是 Message Service 的事），也不负责每个成员的会话视图（这是 Conversation Service 的事）。