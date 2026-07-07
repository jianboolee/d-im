Conversation Service 是 IM 系统中连接"底层消息存储"和"上层用户界面"的关键桥梁。让我系统梳理它的定位和职责。

## 一、Conversation Service 的服务定位

### 1.1 一句话定义

**Conversation Service 是用户维度的会话视图管理中心，负责将底层的 Chat（物理会话）和 Message（消息）转化为每个用户个性化的会话列表和展示状态。**

### 1.2 核心职责

```go
type ConversationServiceResponsibilities struct {
    // ====== 核心职责 ======
    
    // 1. 用户会话列表维护
    SessionList string // 每个用户的会话列表（置顶、排序、归档等）
    
    // 2. 未读计数管理
    UnreadCount string // 每个用户在每个会话的未读消息数
    
    // 3. 最后一条消息摘要
    LastMessage string // 会话列表展示的最后一条消息预览
    
    // 4. 用户个性化设置
    PersonalSettings string // 置顶、免打扰、自定义名称、归档
    
    // 5. 会话状态同步
    SyncService string // 多端同步会话状态（已读、删除等）
    
    // ====== 明确不做的事情 ======
    
    // ❌ 不存储消息内容（Message Service 负责）
    // ❌ 不管理群成员（Group Service 负责）
    // ❌ 不推送消息（Push Service 负责）
}
```

## 二、Conversation 与 Chat 的本质区别

### 2.1 概念对比

```
Chat（物理会话）：
  · 独立存在的聊天实体
  · 所有人看到的是同一个
  · 例如：一个群 "技术交流群"
  
Conversation（用户会话视图）：
  · 用户维度的投影
  · 每个用户有独立的个性化设置
  · 例如：用户A把群置顶了，用户B把群静音了

关系：
  1个 Chat → N个 Conversation
```

### 2.2 具体示例

```
Chat: "group_1234567890"（技术交流群）
  ├── 用户A的 Conversation:
  │   · 置顶: true
  │   · 免打扰: false
  │   · 未读: 99+
  │   · 自定义名称: "摸鱼群"
  │
  ├── 用户B的 Conversation:
  │   · 置顶: false
  │   · 免打扰: true
  │   · 未读: 0
  │   · 自定义名称: ""
  │
  └── 用户C的 Conversation:
      · 置顶: false
      · 免打扰: false
      · 未读: 5
      · 自定义名称: ""
      · 已归档: true
```

## 三、数据模型

### 3.1 Conversation 文档

```go
// pkg/model/conversation.go

// Conversation 用户会话视图
type Conversation struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    
    // 用户和会话关联
    UID       string            `bson:"uid" json:"uid"`           // 用户ID
    ChatID    string            `bson:"chat_id" json:"chat_id"`   // 会话ID（Chat的ID）
    ChatType  ChatType          `bson:"chat_type" json:"chat_type"` // single/group/channel
    
    // ====== 展示信息（从Chat冗余） ======
    // 单聊：对端用户的信息
    // 群聊：群信息
    ShowName    string `bson:"show_name" json:"show_name"`       // 展示名称
    ShowAvatar  string `bson:"show_avatar" json:"show_avatar"`   // 展示头像
    
    // ====== 用户个性化设置 ======
    IsTop       bool   `bson:"is_top" json:"is_top"`             // 是否置顶
    IsMuted     bool   `bson:"is_muted" json:"is_muted"`         // 免打扰
    IsArchived  bool   `bson:"is_archived" json:"is_archived"`   // 是否归档
    CustomName  string `bson:"custom_name" json:"custom_name"`   // 用户自定义名称
    
    // ====== 消息状态 ======
    UnreadCount  int       `bson:"unread_count" json:"unread_count"`   // 未读消息数
    LastReadSeq  int64     `bson:"last_read_seq" json:"last_read_seq"` // 最后已读序列号
    LastReadTime time.Time `bson:"last_read_time" json:"last_read_time"` // 最后阅读时间
    
    // ====== 最后一条消息摘要 ======
    LastMsg *LastMessage `bson:"last_msg,omitempty" json:"last_msg,omitempty"`
    
    // ====== 用户与会话的关系 ======
    JoinedAt  time.Time  `bson:"joined_at" json:"joined_at"`     // 加入时间
    LeftAt    *time.Time `bson:"left_at,omitempty" json:"left_at,omitempty"` // 退出时间
    
    // ====== 状态 ======
    Status    ConvStatus `bson:"status" json:"status"`           // active/archived/deleted
    
    // ====== 时间戳 ======
    CreatedAt time.Time `bson:"created_at" json:"created_at"`
    UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// LastMessage 最后一条消息摘要
type LastMessage struct {
    MsgID      string    `bson:"msg_id" json:"msg_id"`
    SenderID    string    `bson:"sender_id" json:"sender_id"`
    SenderName   string    `bson:"sender_name" json:"sender_name"`     // 发送者名称
    MsgType    string    `bson:"msg_type" json:"msg_type"`       // 消息类型
    Content    string    `bson:"content" json:"content"`         // 内容摘要（截取前50字）
    ClientTime time.Time `bson:"client_time" json:"client_time"` // 消息时间
}

// ConvStatus 会话状态
type ConvStatus string
const (
    ConvStatusActive   ConvStatus = "active"   // 正常
    ConvStatusArchived ConvStatus = "archived" // 已归档
    ConvStatusDeleted  ConvStatus = "deleted"  // 已删除
)
```

### 3.2 MongoDB 索引设计

```go
func (Conversation) Indexes() []mongo.IndexModel {
    return []mongo.IndexModel{
        {
            // 用户会话列表查询（按更新时间排序）
            Keys: bson.D{
                {Key: "uid", Value: 1},
                {Key: "status", Value: 1},
                {Key: "is_top", Value: -1},
                {Key: "updated_at", Value: -1},
            },
            Options: options.Index().SetName("idx_uid_list"),
        },
        {
            // 唯一索引：一个用户对一个Chat只有一个Conversation
            Keys: bson.D{
                {Key: "uid", Value: 1},
                {Key: "chat_id", Value: 1},
            },
            Options: options.Index().SetUnique(true).SetName("idx_uid_chat_unique"),
        },
        {
            // 按ChatID查询（群信息更新时更新所有成员的会话）
            Keys: bson.D{
                {Key: "chat_id", Value: 1},
            },
            Options: options.Index().SetName("idx_chat_id"),
        },
        {
            // 未读消息查询
            Keys: bson.D{
                {Key: "uid", Value: 1},
                {Key: "unread_count", Value: -1},
            },
            Options: options.Index().SetName("idx_unread"),
        },
    }
}
```

## 四、Conversation 的创建时机

### 4.1 触发场景

```
Conversation 创建时机：

1. 单聊：用户A给用户B发第一条消息
   → Message Service 发布 msg.saved
   → Conversation Service 消费事件
   → 为用户A和用户B各创建 Conversation 记录

2. 群聊：群创建时
   → Group Service 发布 group.created
   → Conversation Service 消费事件
   → 为所有初始成员创建 Conversation 记录

3. 用户被邀请入群
   → Group Service 发布 member.joined
   → Conversation Service 消费事件
   → 为新成员创建 Conversation 记录

4. 用户被移出群
   → Group Service 发布 member.kicked
   → Conversation Service 消费事件
   → 更新用户的 Conversation 状态（标记退出）
```

### 4.2 创建流程

```go
// Conversation Service 消费消息事件
func (s *ConversationService) HandleMessageSaved(ctx context.Context, event *MessageSavedEvent) {
    for _, targetUID := range event.TargetUIDs {
        // 检查是否已有 Conversation
        conv, err := s.convRepo.FindByUserAndChat(ctx, targetUID, event.ChatID)
        
        if err == ErrNotFound {
            // 不存在，创建新的 Conversation
            conv = &Conversation{
                UID:         targetUID,
                ChatID:      event.ChatID,
                ChatType:    event.ChatType,
                ShowName:    s.getShowName(event),    // 获取展示名称
                ShowAvatar:  s.getShowAvatar(event),  // 获取展示头像
                JoinedAt:    time.Now(),
                Status:      ConvStatusActive,
                UnreadCount: 1,  // 新消息，未读+1
                LastMsg:     s.buildLastMsg(event),
                CreatedAt:   time.Now(),
                UpdatedAt:   time.Now(),
            }
            s.convRepo.Create(ctx, conv)
        } else {
            // 已存在，更新最后消息和未读计数
            s.convRepo.Update(ctx, targetUID, event.ChatID, UpdateFields{
                UnreadCount: conv.UnreadCount + 1,
                LastMsg:     s.buildLastMsg(event),
                UpdatedAt:   time.Now(),
            })
        }
    }
}
```

## 五、Conversation 的更新场景

### 5.1 更新时机

```
Conversation 更新场景：

1. 新消息到达
   → 更新 last_msg
   → 递增 unread_count（非当前查看的会话）

2. 用户阅读消息
   → 清零 unread_count
   → 更新 last_read_time
   → 更新 last_read_seq

3. 用户操作
   → 置顶/取消置顶
   → 免打扰/取消免打扰
   → 归档/取消归档
   → 自定义名称
   → 删除会话

4. 群信息变更
   → 群名称变更 → 更新 show_name
   → 群头像变更 → 更新 show_avatar

5. 消息撤回
   → 如果撤回的是最后一条消息，更新 last_msg
   → 可能减少 unread_count

6. 用户退出群
   → 更新 status 为 deleted
   → 设置 left_at
```

### 5.2 已读处理

```go
// 用户打开会话，标记已读
func (s *ConversationService) MarkAsRead(ctx context.Context, uid, chatID string) error {
    // 1. 清零未读计数
    return s.convRepo.Update(ctx, uid, chatID, UpdateFields{
        UnreadCount:  0,
        LastReadTime: time.Now(),
        LastReadSeq:  s.getCurrentSeq(ctx, uid, chatID),
        UpdatedAt:    time.Now(),
    })
    
    // 2. 发布已读事件（用于多端同步）
    s.eventPublisher.PublishRead(ctx, &ReadEvent{
        UID:     uid,
        ChatID:  chatID,
        ReadAt:  time.Now(),
    })
}
```

### 5.3 未读计数

```go
// 未读计数规则
func (s *ConversationService) updateUnreadCount(conv *Conversation, isCurrentChat bool) {
    if isCurrentChat {
        // 用户正在查看这个会话，不增加未读
        // 但需要更新 last_read_seq
        return
    }
    
    if conv.IsMuted {
        // 免打扰的会话，不增加未读计数
        // 但消息仍然会更新 last_msg
        return
    }
    
    // 正常增加未读
    conv.UnreadCount++
}
```

## 六、核心 API

### 6.1 对外接口

```go
service ConversationService {
    // ====== 会话列表 ======
    // 获取用户会话列表（支持分页、排序）
    rpc GetConversationList(GetConvListReq) returns (GetConvListResp);
    
    // ====== 未读计数 ======
    // 获取总未读数（App图标角标）
    rpc GetTotalUnreadCount(GetTotalUnreadReq) returns (GetTotalUnreadResp);
    
    // 标记会话已读
    rpc MarkAsRead(MarkAsReadReq) returns (MarkAsReadResp);
    
    // ====== 会话设置 ======
    // 置顶/取消置顶
    rpc ToggleTop(ToggleTopReq) returns (ToggleTopResp);
    
    // 免打扰/取消免打扰
    rpc ToggleMute(ToggleMuteReq) returns (ToggleMuteResp);
    
    // 归档/取消归档
    rpc ToggleArchive(ToggleArchiveReq) returns (ToggleArchiveResp);
    
    // 设置自定义名称
    rpc SetCustomName(SetCustomNameReq) returns (SetCustomNameResp);
    
    // ====== 会话操作 ======
    // 删除会话（用户维度，不影响Chat和消息）
    rpc DeleteConversation(DeleteConvReq) returns (DeleteConvResp);
    
    // ====== 会话同步 ======
    // 同步会话状态（用于多端同步）
    rpc SyncConversations(SyncConvReq) returns (SyncConvResp);
}
```

### 6.2 关键接口设计

```go
// 会话列表（最核心的接口）
type GetConvListReq struct {
    UID    string `json:"uid"`    // 当前用户ID
    Limit  int    `json:"limit"`  // 分页大小
    Offset int    `json:"offset"` // 分页偏移
}

type GetConvListResp struct {
    Conversations []*Conversation `json:"conversations"`
    Total         int64           `json:"total"`
    HasMore       bool            `json:"has_more"`
}

// 排序规则：
// 1. 置顶的优先（is_top = true）
// 2. 按最后消息时间倒序（updated_at DESC）
// 3. 过滤已删除和已退出的
// 4. 归档的默认不显示（可单独查询）

// 总未读数（App角标）
type GetTotalUnreadResp struct {
    TotalUnread int `json:"total_unread"`
}
// 计算方式：sum of unread_count where is_muted = false
```

## 七、事件驱动架构

### 7.1 消费的事件

```
Conversation Service 消费的事件：

来自 Message Service：
  ✅ msg.saved → 更新 last_msg, unread_count
  ✅ msg.recalled → 可能更新 last_msg, unread_count（如果撤回的是最后一条）
  ✅ msg.read → 多端同步已读状态

来自 Group Service：
  ✅ group.created → 为所有初始成员创建 Conversation
  ✅ group.updated → 更新 show_name, show_avatar
  ✅ member.joined → 为新成员创建 Conversation
  ✅ member.left → 更新 Conversation 状态
  ✅ member.kicked → 更新 Conversation 状态
  ✅ group.dismissed → 更新所有成员 Conversation 状态

来自 User Service：
  ✅ user.updated → 更新单聊会话的 show_name, show_avatar
```

### 7.2 发布的事件

```
Conversation Service 发布的事件：

  ✅ conversation.updated → 会话更新（多端同步）
  ✅ conversation.unread_changed → 未读数变更（App角标更新）
```

## 八、与其他服务的边界

```
┌─────────────────────────────────────────────────────────────────┐
│                     Conversation Service                        │
│                                                                 │
│  拥有：                                                          │
│    ✅ conversations 集合（用户会话视图）                           │
│    ✅ 每个用户的会话列表                                          │
│    ✅ 未读计数                                                    │
│    ✅ 用户个性化设置                                              │
│    ✅ 最后一条消息摘要                                            │
│                                                                 │
│  不拥有：                                                        │
│    ❌ messages 集合（Message Service 拥有）                       │
│    ❌ chats 集合（Chat/Group Service 拥有）                       │
│    ❌ mailbox 集合（Message Service 拥有）                        │
│                                                                 │
│  关系：                                                          │
│    ← Message Service：消费消息事件                               │
│    ← Group Service：消费群组事件                                 │
│    ← User Service：消费用户信息变更事件                          │
│    → Gateway：提供会话列表、未读计数查询                          │
│    → NATS：发布会话更新事件（多端同步）                           │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 九、总结

```
┌─────────────────────────────────────────────────────────────────┐
│                Conversation Service 服务定位总结                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  一句话：将消息和会话转化为用户视角的个性化会话列表                   │
│                                                                 │
│  核心价值：                                                       │
│    ✅ 用户打开 App 看到的会话列表                                  │
│    ✅ 每个会话的未读红点                                          │
│    ✅ 最后一条消息预览                                            │
│    ✅ 置顶、免打扰等个性化设置                                     │
│    ✅ 多端状态同步                                                │
│                                                                 │
│  设计原则：                                                       │
│    📌 每个用户拥有独立的 Conversation 视图                        │
│    📌 以事件驱动方式更新（不主动查询）                             │
│    📌 冗余展示信息，避免每次查询多个服务                          │
│    📌 用户操作即时生效，不需要同步到其他用户                       │
│                                                                 │
│  与 Message Service 的关系（最容易混淆）：                        │
│    Message Service：管理消息的"内容"和"存储"                      │
│    Conversation Service：管理消息的"展示"和"状态"                │
│    就像邮局：                                                     │
│      Message Service = 邮件分拣中心（负责处理和存储邮件）          │
│      Conversation Service = 你的信箱（负责展示和状态）            │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

Conversation Service 的核心价值在于**将底层的数据（Chat + Message）转化为面向用户的、个性化的展示视图**。它不产生数据，而是消费其他服务的事件，为用户提供统一的"会话列表"体验。