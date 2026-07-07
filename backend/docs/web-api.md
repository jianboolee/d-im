# Web API 接入契约与落地步骤

本文档定义 `web/` 接入重构后后端的目标契约和逐项落地顺序。目标不是一次性补完所有能力，而是按可验证事项逐步推进：先完成基础 auth 和登录，再完成单聊创建、会话、文本消息、历史消息、WebSocket、已读未读，最后逐个补齐其它消息类型和群聊能力。

## 总体原则

### 统一响应结构

所有 HTTP API 都返回统一结构：

```json
{
  "code": 0,
  "data": {},
  "error": ""
}
```

约定：

- `code = 0` 表示成功。
- `code != 0` 表示业务失败。
- `data` 始终承载成功数据。没有数据时使用 `{}` 或 `null`，由具体接口定义。
- `error` 始终承载失败说明。成功时为空字符串。
- HTTP status 仍然表达通用错误分类，例如 `400`、`401`、`403`、`404`、`500`。

示例：

```json
{
  "code": 0,
  "data": {
    "id": "u_001"
  },
  "error": ""
}
```

```json
{
  "code": 401001,
  "data": null,
  "error": "invalid or expired token"
}
```

### 命名边界

HTTP DTO、WebSocket DTO、web SDK 类型尽量使用同一套最终字段，避免长期靠前端做大规模映射。

统一使用：

- `conversation_id`
- `message_id`
- `sender`
- `sender_id`
- `message_type`
- `content`
- `access_token`
- `refresh_token`

避免使用：

- `chat_id` 暴露给 web 作为主 ID
- `msg_id` 暴露给 web 作为主 ID
- 旧的发送者字段命名
- `type` 表示消息类型
- `token` 表示 access token

后端内部存储也统一使用 `sender_id`，gateway 层输出稳定 DTO。

### Chat ID 与单聊唯一性

`chat_id` 是会话实体 ID，不承载成员信息和业务唯一性，不使用 `single_user_a_user_b` 这类语义化拼接格式。

内部约定：

- `chat_id` 统一由 ID 生成器生成，例如 `chat_178...`。
- 单聊幂等性由 `single_key` 表达，规则是两个用户 ID 排序后用 `:` 拼接。
- `chats` 集合对 `{ chat_type, single_key }` 建唯一索引，仅约束单聊文档。
- 群聊也使用同一套 `chat_id` 生成方式，不使用时间戳拼接。
- web 仍只使用 `conversation_id`，不依赖底层 `chat_id` 命名规则。

### 输入输出一致

以下场景必须尽量返回同一种 `MessageDTO`：

- HTTP 发送消息响应
- 历史消息列表项
- Connector WebSocket 消息推送
- 撤回、转发等消息事件中引用的消息主体

这样 web 只需要一套消息渲染逻辑。

### Base URL

- HTTP API: `/api/v1`
- WebSocket: `/ws`

开发环境建议由 Vite 代理：

- `/api/v1` -> API Gateway
- `/ws` -> Connector

### 鉴权

受保护 HTTP API 使用 access token：

```http
Authorization: Bearer <access_token>
```

WebSocket 连接使用 query token：

```text
/ws?access_token=<access_token>
```

### 时间格式

所有时间字段使用 RFC3339 字符串，例如：

```text
2026-07-05T12:34:56Z
```

### Cursor 分页

列表接口统一使用 cursor 机制，不使用 offset。

通用响应结构：

```json
{
  "items": [],
  "next_cursor": "",
  "has_more": false
}
```

请求参数：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `limit` | number | 否 | 默认 20，最大建议 100 |
| `cursor` | string | 否 | 下一页游标 |

### Mailbox 架构

IM 的会话历史读取以 `messages(chat_id, seq)` 为权威入口。`user_mailbox` 保存用户维度的投递记录和增量同步流水，不作为会话历史分页的数据源。

核心职责：

| 模块 | 职责 |
| --- | --- |
| `messages` | 保存消息主体，包括 `message_id`、`conversation_id/chat_id`、`seq`、发送者、消息类型、内容、服务端时间、撤回状态、扩展字段 |
| `user_mailbox` | 保存用户维度消息投递记录，包括 `uid`、`conversation_id/chat_id`、`message_id`、`message_seq`、`seq_id`、投递/已读状态、创建时间 |
| `conversations` | 保存用户维度会话视图，包括置顶、免打扰、最后已读序列、最后消息摘要 |

写入规则：

- 每条消息先从 `chat.last_seq` 原子分配 `message.seq`，再写入 `messages` 主体。
- 再为所有可见用户写入 `user_mailbox`。
- 发送者也必须写入自己的 mailbox，状态为 `sent`。
- 接收者写入 mailbox，状态为 `delivered`。
- 群聊按实际可见成员写入 mailbox；已退群用户是否可见历史由成员关系和 mailbox 记录决定。
- 不允许只写接收者 mailbox，否则发送者自己的多端同步流水和投递状态会缺失。

读取规则：

- 历史消息接口用 `conversation_id` 定位会话，并从 `messages(chat_id, seq)` 分页。
- `MessageDTO.sequence` 来自 `messages.seq`，它是 chat 维度的严格递增消息序列。
- `user_mailbox.seq_id` 只用于用户级增量同步流水；`user_mailbox.message_seq` 指向 `messages.seq`。
- 历史消息的用户维度状态可通过后续状态接口或必要时额外关联 mailbox 获取，不作为历史分页游标。
- `MessageDTO.content`、`sender`、`message_type`、`server_time` 等主体字段来自 messages。
- cursor 基于 `messages.seq`，不直接基于消息时间字段。

状态边界：

- 后端不再维护 `unread_count`；历史字段和旧数据不作为业务依据。
- last read 的权威来源是 `conversations.last_read_seq`，它指向 `messages.seq`。
- 单用户删除、清空会话、离线补偿、多端同步都应优先改 mailbox/conversation 视图，不删除 messages 主体。
- 全局撤回、合规删除等影响所有用户的行为才修改 messages 主体状态。

## 通用 DTO

### UserDTO

```json
{
  "id": "u_001",
  "nickname": "Alice",
  "avatar": "https://example.com/a.png",
  "status": "active"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 用户 ID |
| `nickname` | string | 否 | 昵称 |
| `avatar` | string | 否 | 头像 URL |
| `status` | string | 否 | `active` / `disabled` |

### ConversationDTO

会话列表项和会话详情使用同一结构。

```json
{
  "id": "conv_001",
  "conversation_id": "conv_001",
  "chat_id": "chat_001",
  "chat_type": "single",
  "title": "Bob",
  "avatar": "",
  "participants": ["u_001", "u_002"],
  "peer_user": {
    "id": "u_002",
    "nickname": "Bob",
    "avatar": "",
    "status": "active"
  },
  "group": null,
  "last_message": {
    "msg_id": "msg_001",
    "sender_id": "u_002",
    "msg_type": "text",
    "content_preview": "hello",
    "client_time": "2026-07-05T12:34:56Z"
  },
  "last_read_sequence": 0,
  "muted": false,
  "pinned": false,
  "created_at": "2026-07-05T12:00:00Z",
  "updated_at": "2026-07-05T12:34:57Z",
  "last_activity_at": "2026-07-05T12:34:57Z"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `id` | string | 是 | 等同 `conversation_id`，方便 web 列表 key |
| `conversation_id` | string | 是 | web 可见稳定会话 ID |
| `chat_id` | string | 是 | 消息容器 ID，后端内部拉历史和分发使用 |
| `chat_type` | string | 是 | `single` / `group` / `system` / `channel` |
| `title` | string | 否 | 会话展示名称 |
| `avatar` | string | 否 | 会话展示头像 |
| `participants` | string[] | 是 | 参与者 ID |
| `peer_user` | UserDTO/null | 单聊建议 | 单聊对方用户 |
| `group` | object/null | 群聊建议 | 群信息 |
| `last_message` | LastMessage/null | 否 | 最后一条消息摘要，结构使用后端 `types.LastMessage` |
| `last_read_sequence` | number | 是 | 当前登录用户视角的已读指针 |
| `last_read_at` | string | 否 | 当前登录用户最近一次已读时间 |
| `muted` | boolean | 是 | 当前登录用户是否免打扰 |
| `pinned` | boolean | 是 | 当前登录用户是否置顶 |
| `last_activity_at` | string | 是 | 排序用活跃时间 |

### MessageDTO

HTTP 发送响应、历史消息、WebSocket 推送统一使用该结构。

```json
{
  "message_id": "msg_001",
  "conversation_id": "conv_001",
  "chat_id": "chat_001",
  "chat_type": "single",
  "sender_id": "u_001",
  "sender": {
    "id": "u_001",
    "nickname": "Alice",
    "avatar": "",
    "status": "active"
  },
  "message_type": "text",
  "content": {
    "text": "hello"
  },
  "content_preview": "hello",
  "status": "sent",
  "sequence": 1001,
  "client_message_id": "cmid_u_001_xxx",
  "client_time": "2026-07-05T12:34:56Z",
  "server_time": "2026-07-05T12:34:57Z",
  "created_at": "2026-07-05T12:34:57Z",
  "updated_at": "2026-07-05T12:34:57Z",
  "recalled": false,
  "quote": null,
  "ext": {}
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `message_id` | string | 是 | web 可见稳定消息 ID |
| `conversation_id` | string | 是 | 所属会话 ID |
| `chat_id` | string | 是 | 所属消息容器 ID |
| `chat_type` | string | 是 | `single` / `group` / `system` / `channel` |
| `sender_id` | string | 是 | 发送者 ID |
| `sender` | UserDTO/null | 建议 | 发送者快照 |
| `message_type` | string | 是 | 消息内容类型 |
| `content` | object | 是 | 按消息类型定义 |
| `content_preview` | string | 是 | 会话列表和通知展示摘要 |
| `status` | string | 是 | `sending` / `sent` / `delivered` / `read` / `failed` / `recalled` |
| `sequence` | number | 建议 | 会话内递增序列，用于历史和已读 |
| `client_message_id` | string | 建议 | 前端临时消息确认和幂等 |

## 消息内容结构

消息内容结构需要作为独立事项提前整理。每新增一种消息类型，都必须同时明确：

- HTTP 发送请求 content
- HTTP 发送响应 MessageDTO
- 历史消息 MessageDTO
- WebSocket 推送 MessageDTO
- content_preview 生成规则
- web 展示规则

消息语义统一由 `message_type + content` 表达。web 不再使用 `payload` 作为消息语义字段，也不维护前端自定义消息类型别名，例如语音消息统一使用后端定义的 `voice`。

### TextContent

```json
{
  "text": "hello",
  "mentions": [],
  "is_at_all": false
}
```

规则：

- `text` 必填，不能为空。
- `mentions` 为被 @ 的用户 ID 列表。
- `is_at_all` 表示是否 @ 全体。
- `content_preview` 使用文本内容，长度由后端截断。

### ImageContent

```json
{
  "url": "https://example.com/a.png",
  "thumb_url": "https://example.com/a.thumb.png",
  "width": 800,
  "height": 600,
  "size": 123456,
  "format": "png",
  "md5": "",
  "file_name": "a.png"
}
```

`content_preview` 使用 `[图片]`。

### VideoContent

```json
{
  "url": "https://example.com/a.mp4",
  "thumb_url": "https://example.com/a.jpg",
  "duration": 12,
  "width": 1280,
  "height": 720,
  "size": 1234567,
  "format": "mp4",
  "md5": ""
}
```

`content_preview` 使用 `[视频]`。

### VoiceContent

```json
{
  "url": "https://example.com/a.aac",
  "duration": 8,
  "size": 12345,
  "format": "aac",
  "md5": ""
}
```

`content_preview` 使用 `[语音]`。

### FileContent

```json
{
  "url": "https://example.com/a.pdf",
  "file_name": "a.pdf",
  "size": 123456,
  "format": "pdf",
  "md5": ""
}
```

`content_preview` 使用 `[文件] a.pdf`。

### CardContent

```json
{
  "title": "商品标题",
  "description": "商品描述",
  "image_url": "https://example.com/a.png",
  "action_url": "https://example.com/item/1"
}
```

`content_preview` 直接使用卡片标题，例如 `商品标题`。

### LinkContent

```json
{
  "url": "https://example.com",
  "title": "页面标题",
  "description": "页面描述",
  "thumb_url": "",
  "favicon": ""
}
```

`content_preview` 使用 `[链接] 页面标题`。

### TemplateContent

```json
{
  "template_id": "order_paid",
  "title": "订单已支付",
  "items": [
    {
      "label": "订单号",
      "value": "O123",
      "type": "text",
      "color": "",
      "action_url": ""
    }
  ],
  "description": "",
  "action_url": "",
  "action_text": ""
}
```

`content_preview` 直接使用模板标题，例如 `订单已支付`。

### LocationContent

```json
{
  "latitude": 31.2304,
  "longitude": 121.4737,
  "address": "上海市",
  "name": "人民广场"
}
```

`content_preview` 使用 `[位置]`。

## Auth 与登录接口

### 登录页面入口

登录页面使用 URL 参数进入，不在页面里硬编码用户：

```text
/im/login?id=u_001
```

或进入指定会话：

```text
/im/login?id=u_001&redirect=/im/chat/conv_001
```

web 从 URL 参数读取 `id`，用户输入或页面预填超级密码后，调用登录接口换取 `access_token` 和 `refresh_token`。

### 超级密码配置

后端支持 `id + 超级密码` 登录。超级密码默认留空，写在配置文件中，并允许由环境变量覆盖，避免写死在 web 或代码中。

建议配置：

```yaml
auth:
  super_password: ""
```

建议环境变量：

```text
IM_SUPER_PASSWORD=<strong-password>
```

规则：

- web 不保存、不内置超级密码。
- 后端启动时从 config 读取超级密码。
- 环境变量优先级高于配置文件默认值。
- 超级密码为空时，密码登录不可用，任何密码都不能通过。
- 只有显式配置了超级密码，并且请求中的 `password` 与配置值一致时，登录才通过。
- 超级密码仅用于开发、内测或受控环境；生产是否启用由部署配置决定。

### 密码登录

```http
POST /api/v1/auth/login
Content-Type: application/json
```

请求：

```json
{
  "id": "u_001",
  "password": "super-password",
  "device_id": "web_chrome_xxx"
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "access_token": "jwt",
    "refresh_token": "jwt",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "error": ""
}
```

### 签发登录 Ticket

业务系统使用 API Key 为用户签发一次性 ticket。

```http
POST /api/v1/auth/ticket
X-API-Key: <api_key>
Content-Type: application/json
```

请求：

```json
{
  "id": "u_001"
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "ticket": "one_time_ticket",
    "redirect_url": "http://localhost:5173/im/enter?ticket=one_time_ticket"
  },
  "error": ""
}
```

### Ticket 换 Token

使用 `auth/ticket` 接口完成 ticket 换 token。该接口通过请求体是否携带 `ticket` 区分签发 ticket 和兑换 token，或者落地时拆成更明确的子动作；最终文档以实现前确认的路由为准。

建议请求：

```http
POST /api/v1/auth/ticket
Content-Type: application/json
```

```json
{
  "ticket": "one_time_ticket",
  "device_id": "web_chrome_xxx"
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "access_token": "jwt",
    "refresh_token": "jwt",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "error": ""
}
```

注意：全链路统一使用 `access_token` 和 `refresh_token` 字段，不再输出 `token`。

### 刷新 Token

```http
POST /api/v1/auth/refresh
Authorization: Bearer <refresh_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "access_token": "jwt",
    "refresh_token": "jwt",
    "token_type": "Bearer",
    "expires_in": 900
  },
  "error": ""
}
```

### 登出

```http
POST /api/v1/auth/logout
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "status": "ok"
  },
  "error": ""
}
```

## 用户接口

### 当前用户

```http
GET /api/v1/users/me
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "id": "u_001",
    "nickname": "Alice",
    "avatar": "",
    "status": "active"
  },
  "error": ""
}
```

### 用户详情

```http
GET /api/v1/users/{user_id}
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "id": "u_002",
    "nickname": "Bob",
    "avatar": "",
    "status": "active"
  },
  "error": ""
}
```

不提供批量获取用户详情接口。需要用户信息的会话和消息接口应尽量携带必要用户快照，减少 web 额外请求。

## 单聊会话接口

### 创建或获取单聊会话

首次单聊按正确后端架构实现：web 不拼接底层 `chat_id`，只提交对方用户 ID，由后端幂等创建或获取单聊会话，返回 `ConversationDTO`。

```http
POST /api/v1/conversations/single
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "peer_user_id": "u_002"
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "id": "conv_001",
    "conversation_id": "conv_001",
    "chat_id": "chat_001",
    "chat_type": "single",
    "title": "Bob",
    "avatar": "",
    "participants": ["u_001", "u_002"],
    "peer_user": {
      "id": "u_002",
      "nickname": "Bob",
      "avatar": "",
      "status": "active"
    },
    "group": null,
    "last_message": null,
    "last_read_sequence": 0,
    "muted": false,
    "pinned": false,
    "created_at": "2026-07-05T12:00:00Z",
    "updated_at": "2026-07-05T12:00:00Z",
    "last_activity_at": "2026-07-05T12:00:00Z"
  },
  "error": ""
}
```

## 会话接口

### 会话列表

```http
GET /api/v1/conversations?limit=20&cursor=
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": "conv_001",
        "conversation_id": "conv_001",
        "chat_id": "chat_001",
        "chat_type": "single",
        "title": "Bob",
        "avatar": "",
        "participants": ["u_001", "u_002"],
        "peer_user": {
          "id": "u_002",
          "nickname": "Bob",
          "avatar": "",
          "status": "active"
        },
        "group": null,
        "last_message": null,
        "last_read_sequence": 0,
        "muted": false,
        "pinned": false,
        "created_at": "2026-07-05T12:00:00Z",
        "updated_at": "2026-07-05T12:00:00Z",
        "last_activity_at": "2026-07-05T12:00:00Z"
      }
    ],
    "next_cursor": "",
    "has_more": false
  },
  "error": ""
}
```

排序：

1. 置顶会话优先。
2. 同置顶状态按 `last_activity_at` 倒序。
3. cursor 应包含排序所需信息，避免翻页重复或遗漏。

### 会话详情

```http
GET /api/v1/conversations/{conversation_id}
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "id": "conv_001",
    "conversation_id": "conv_001",
    "chat_id": "chat_001",
    "chat_type": "single",
    "title": "Bob",
    "avatar": "",
    "participants": ["u_001", "u_002"],
    "peer_user": {
      "id": "u_002",
      "nickname": "Bob",
      "avatar": "",
      "status": "active"
    },
    "group": null,
    "last_message": null,
    "last_read_sequence": 0,
    "muted": false,
    "pinned": false,
    "created_at": "2026-07-05T12:00:00Z",
    "updated_at": "2026-07-05T12:00:00Z",
    "last_activity_at": "2026-07-05T12:00:00Z"
  },
  "error": ""
}
```

会话详情结构必须与会话列表 item 一致。

### 标记会话已读

已读标记按后端权威状态正确实现，不做前端本地映射兜底。

```http
POST /api/v1/conversations/{conversation_id}/read
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "last_read_sequence": 1001
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "conversation_id": "conv_001",
    "last_read_sequence": 1001,
    "read_at": "2026-07-05T12:35:00Z"
  },
  "error": ""
}
```

规则：

- `last_read_sequence` 为空时，后端可使用当前会话最新 sequence。
- `last_read_sequence` 大于当前 `chat.last_seq` 时，后端按 `chat.last_seq` 截断，并返回实际写入值。
- 后端只在 `last_read_sequence` 前进时更新当前用户在该会话中的 `last_read_seq` 和 `last_read_at`，不回退。
- read 接口必须幂等：重复上报、网络重试、乱序旧请求都不能产生副作用或倒退。
- 后端不维护、不返回权威 `unread_count`。
- 发送者自己的新消息在发送成功后必须推进当前用户会话的 `last_read_sequence` 到该消息 `sequence`，自己发送的消息不能算作自己的未读。
- web 应上报当前已经展示的最大 message `sequence`，并用节流队列合并上报；切换会话、页面隐藏或卸载前需要 flush，保证最后一个最大 sequence 不丢。
- web 未读徽标由 `last_message.sequence > last_read_sequence` 推导；持久已读状态以 `last_read_sequence` 为准。

### 更新会话设置

用于更新当前登录用户视角下的会话增强设置。

```http
PATCH /api/v1/conversations/{conversation_id}/settings
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "pinned": true,
  "muted": false
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "id": "conv_001",
    "conversation_id": "conv_001",
    "chat_id": "chat_001",
    "chat_type": "single",
    "title": "Bob",
    "avatar": "",
    "participants": ["u_001", "u_002"],
    "peer_user": null,
    "group": null,
    "last_message": null,
    "last_read_sequence": 0,
    "muted": false,
    "pinned": true,
    "created_at": "2026-07-05T12:00:00Z",
    "updated_at": "2026-07-05T12:00:00Z",
    "last_activity_at": "2026-07-05T12:00:00Z"
  },
  "error": ""
}
```

规则：

- `pinned` 和 `muted` 至少提供一个。
- 设置只影响当前登录用户自己的 conversation 视图，不影响同一 chat 下其它用户。
- 响应返回完整 ConversationDTO，字段保持平铺，不返回 `state`。

## 消息接口

### 发送消息

发消息接口按新后端架构实现：web 只提交 `conversation_id`、`message_type`、`content`、`client_message_id`，后端根据 conversation 找到参与者并分发。

```http
POST /api/v1/messages
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "conversation_id": "conv_001",
  "message_type": "text",
  "content": {
    "text": "hello",
    "mentions": [],
    "is_at_all": false
  },
  "client_message_id": "cmid_u_001_xxx",
  "client_time": "2026-07-05T12:34:56Z",
  "quote_message_id": ""
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "message_id": "msg_001",
    "conversation_id": "conv_001",
    "chat_type": "single",
    "sender_id": "u_001",
    "sender": {
      "id": "u_001",
      "nickname": "Alice",
      "avatar": "",
      "status": "active"
    },
    "message_type": "text",
    "content": {
      "text": "hello",
      "mentions": [],
      "is_at_all": false
    },
    "content_preview": "hello",
    "status": "sent",
    "sequence": 1001,
    "client_message_id": "cmid_u_001_xxx",
    "client_time": "2026-07-05T12:34:56Z",
    "server_time": "2026-07-05T12:34:57Z",
    "created_at": "2026-07-05T12:34:57Z",
    "updated_at": "2026-07-05T12:34:57Z",
    "recalled": false,
    "quote": null,
    "ext": {}
  },
  "error": ""
}
```

规则：

- 发送者从 access token 中读取，不允许 web 传 sender。
- 后端校验发送者是否属于该 conversation。
- 后端生成 `message_id` 和 `sequence`。
- `client_message_id` 作为发送幂等键，唯一范围为 `conversation_id + sender_id + client_message_id`；重复提交必须返回已存在消息，不重复分发和累加未读。
- 后端写入 messages 主体。
- 后端写入发送者和接收者 mailbox，mailbox 记录 `message_seq` 和用户同步 `seq_id`。
- 后端更新会话 last message。
- 后端不维护 `unread_count`。
- HTTP 响应 MessageDTO 必须与 WebSocket 推送 MessageDTO 一致。

### 历史消息

历史消息接口按 `conversation_id` 定位会话，后端从 `messages(chat_id, seq)` 拉取消息主体。这里的 `conversation_id` 是 web 可见稳定会话 ID，不是底层存储细节。

```http
GET /api/v1/conversations/{conversation_id}/messages?limit=20&cursor=
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "message_id": "msg_001",
        "conversation_id": "conv_001",
        "chat_type": "single",
        "sender_id": "u_001",
        "sender": {
          "id": "u_001",
          "nickname": "Alice",
          "avatar": "",
          "status": "active"
        },
        "message_type": "text",
        "content": {
          "text": "hello"
        },
        "content_preview": "hello",
        "status": "sent",
        "sequence": 1001,
        "client_message_id": "cmid_u_001_xxx",
        "client_time": "2026-07-05T12:34:56Z",
        "server_time": "2026-07-05T12:34:57Z",
        "created_at": "2026-07-05T12:34:57Z",
        "updated_at": "2026-07-05T12:34:57Z",
        "recalled": false,
        "quote": null,
        "ext": {}
      }
    ],
    "next_cursor": "",
    "has_more": false
  },
  "error": ""
}
```

排序：

- 默认返回最新一页，但 `items` 内部按时间正序排列，旧消息在前，新消息在后。
- cursor 用于继续向更早历史翻页。
- cursor 基于 `messages.seq`，即 chat 维度消息序列。
- 发送消息时需要同时写入发送者和接收者 mailbox，保证用户级同步流水完整。
- 如果后续需要向后同步新消息，可单独增加 `after_sequence` 或 `since_sequence`，不要混用含义不清的 cursor。

### 搜索会话内消息

用于在当前登录用户自己的某个会话内搜索历史消息。接口仍按 web 可见的 `conversation_id` 定位会话，后端验证当前用户拥有该 conversation 后，再使用内部 `chat_id` 查询 `messages`。

```http
GET /api/v1/conversations/{conversation_id}/messages/search?q=hello&limit=20&cursor=
Authorization: Bearer <access_token>
```

响应结构与历史消息一致：

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "message_id": "msg_001",
        "conversation_id": "conv_001",
        "chat_id": "chat_001",
        "chat_type": "single",
        "sender_id": "u_001",
        "message_type": "text",
        "content": {
          "text": "hello"
        },
        "content_preview": "hello",
        "status": "sent",
        "sequence": 1001,
        "created_at": "2026-07-05T12:34:57Z"
      }
    ],
    "next_cursor": "",
    "has_more": false
  },
  "error": ""
}
```

规则：

- `q` 必填，后端会 trim 空白。
- 当前简单搜索先基于 `content_preview` 做会话内匹配，可覆盖文本、文件名、卡片标题、链接标题等已进入预览文本的内容。
- cursor 仍基于 `messages.seq`，用于继续向更早搜索结果翻页。
- `items` 内部按时间正序排列，旧消息在前，新消息在后。

## WebSocket

### 连接

```text
ws://localhost:8081/ws?access_token=<access_token>
```

### 消息包结构

WebSocket 使用 envelope：

```json
{
  "type": "message",
  "data": {},
  "request_id": "",
  "server_time": "2026-07-05T12:34:57Z"
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `type` | string | 是 | `ping` / `pong` / `message` / `conversation_state` / `error` |
| `data` | object | 否 | 事件数据 |
| `request_id` | string | 否 | 客户端请求 ID |
| `server_time` | string | 否 | 服务端时间 |

### Ping/Pong

web 发送：

```json
{
  "type": "ping",
  "seq_id": 1,
  "client_time": 1720123456789
}
```

后端响应：

```json
{
  "type": "pong",
  "seq_id": 1,
  "server_time": 1720123456792
}
```

规则：

- `seq_id` 是单条 WebSocket 连接内的心跳自增序号，不使用 Snowflake。
- `client_time` 和 `server_time` 都是毫秒时间戳。
- web SDK 收到 `pong` 后按 `seq_id` 匹配本地 pending ping，计算 RTT 和客户端时钟偏移。
- 连续 3 次没有收到对应 `pong` 时，web SDK 主动断开并触发重连。
- `pong` 不进入消息列表。

### 消息推送

Connector 推送消息：

```json
{
  "type": "message",
  "data": {
    "message": {
      "message_id": "msg_001",
      "conversation_id": "conv_001",
      "chat_type": "single",
      "sender_id": "u_001",
      "sender": {
        "id": "u_001",
        "nickname": "Alice",
        "avatar": "",
        "status": "active"
      },
      "message_type": "text",
      "content": {
        "text": "hello"
      },
      "content_preview": "hello",
      "status": "sent",
      "sequence": 1001,
      "client_message_id": "cmid_u_001_xxx",
      "client_time": "2026-07-05T12:34:56Z",
      "server_time": "2026-07-05T12:34:57Z",
      "created_at": "2026-07-05T12:34:57Z",
      "updated_at": "2026-07-05T12:34:57Z",
      "recalled": false,
      "quote": null,
      "ext": {}
    },
    "conversation": {
      "conversation_id": "conv_001",
      "last_read_sequence": 900,
      "muted": false,
      "pinned": false
    }
  },
  "server_time": "2026-07-05T12:34:57Z"
}
```

规则：

- `data.message` 与 HTTP 发送响应中的 MessageDTO 保持一致。
- `data.conversation.last_read_sequence` 为接收者视角的持久已读指针。
- 推送不携带权威 `unread_count`。

### 会话状态推送

已读、置顶、免打扰等状态变化可以使用：

```json
{
  "type": "conversation_state",
  "data": {
    "conversation_id": "conv_001",
    "last_read_sequence": 1001,
    "muted": false,
    "pinned": false
  },
  "server_time": "2026-07-05T12:35:00Z"
}
```

## 落地步骤

每一步只做一个可验证事项。确认通过后再进入下一步。

### 1. 整理最终 API 契约

目标：确认本文档中的路由、字段、响应结构、消息内容结构和 WebSocket envelope。

完成标准：

- 文档经过确认。
- 不开始实现未确认的字段映射。
- 明确哪些字段是后端内部字段，哪些是 web DTO 字段。

### 2. 基础 Auth 和登录

目标：web 可以通过 `id + 超级密码` 或 ticket 完成登录，并持有 `access_token` / `refresh_token`。

任务：

- 统一 auth HTTP 返回结构。
- 登录接口接收 `id`、`password`、`device_id`。
- 后端从 config 读取超级密码，并支持通过 `IM_SUPER_PASSWORD` 环境变量覆盖；默认空值时禁用密码登录。
- 登录接口返回 `access_token` / `refresh_token`。
- ticket 签发和 ticket 换 token 使用确认后的 `auth/ticket` 语义。
- refresh 使用 `refresh_token` 换新 token pair。
- web 登录页从 URL 参数读取 `id` 或 `ticket`。

验收：

- `/im/login?id=u_001` 可进入登录流程。
- 未配置超级密码时，密码登录返回统一错误结构。
- 输入正确超级密码后可获得 token pair。
- 输入错误超级密码时返回统一错误结构。
- web 本地保存 `access_token` / `refresh_token`。
- access token 过期前后 refresh 可用。

### 3. 用户详情

目标：登录后 web 可以获取当前用户和单个用户详情。

任务：

- 实现 `GET /api/v1/users/me`。
- 实现 `GET /api/v1/users/{user_id}`。
- 不实现批量用户详情接口。

验收：

- web 侧能显示当前用户昵称和头像。
- 会话或消息缺少用户快照时，可以按单用户接口补齐。

### 4. 单聊会话创建或获取

目标：web 给一个 `peer_user_id`，后端返回稳定单聊 ConversationDTO。

任务：

- 实现 `POST /api/v1/conversations/single`。
- 后端内部幂等生成或获取单聊实体。
- 返回会话详情结构。

验收：

- 同一对用户多次调用返回同一个 `conversation_id`。
- 当前用户和对方用户都拥有对应会话视图。

### 5. 会话详情

目标：web 可以通过 `conversation_id` 获取会话详情。

任务：

- 实现 `GET /api/v1/conversations/{conversation_id}`。
- 详情结构与列表 item 一致。

验收：

- 直接访问 `/im/chat/:conversationId` 时，web 能拉取详情并展示标题、头像、参与者。

### 6. 会话列表 Cursor

目标：web 可以分页获取会话列表。

任务：

- 实现 `GET /api/v1/conversations?limit=&cursor=`。
- cursor 排序基于置顶状态和 `last_activity_at`。
- 返回 last message 和当前用户会话状态。

验收：

- 初次加载会话列表正常。
- 翻页不重复、不遗漏。
- 最后消息显示正确。

### 7. 消息内容结构整理

目标：先把所有消息类型的 content schema 和 preview 规则定稳。

任务：

- 实现或整理后端 content DTO。
- 确认 text/image/video/voice/file/card/link/template/location 的请求和响应结构。
- 先只需要完整落地 text，但其它类型结构要先确认，避免后续返工。

验收：

- TextContent 请求和响应稳定。
- 新增其它消息类型时不需要修改 MessageDTO 外层结构。

### 8. 文本消息发送

目标：web 可发送文本消息，HTTP 响应返回完整 MessageDTO。

任务：

- 实现 `POST /api/v1/messages` 的 text 分支。
- 后端校验 conversation 成员。
- 后端生成 `message_id`、`sequence`。
- 写入 messages 主体。
- 写入发送者和接收者 mailbox。
- 更新会话 last message。
- 不维护 `unread_count`。

验收：

- 发送成功后 web 用响应确认临时消息。
- 会话列表 last message 更新。
- 接收方在线时能收到 WS 推送。

### 9. 历史消息

目标：web 可按 `conversation_id` 拉取历史消息。

任务：

- 实现 `GET /api/v1/conversations/{conversation_id}/messages?limit=&cursor=`。
- 从 `messages(chat_id, seq)` 查询。
- 返回 MessageDTO 列表。
- `sequence` 来自 `messages.seq`。
- items 内部按时间正序。

验收：

- 刷新页面后能恢复当前会话历史消息。
- 向上滚动能继续加载更早消息。

### 10. WebSocket Ping/Pong 和消息推送

目标：在线用户能收到消息推送，推送结构与 HTTP 发送响应一致。

任务：

- WS 连接参数统一为 `access_token`。
- 实现 `ping` / `pong`。
- Connector 推送 envelope。
- `data.message` 使用 MessageDTO。
- `data.conversation` 只携带会话状态指针和设置项，不携带权威 `unread_count`。

验收：

- A 给 B 发消息，B 在线实时收到。
- B 收到的 WS message 与 A 发送 HTTP 响应结构一致。
- B 的本地会话列表能更新 last message。

### 11. 已读链路

目标：进入会话后已读状态由后端权威维护。

任务：

- 实现 `POST /api/v1/conversations/{conversation_id}/read`。
- 更新当前用户 `last_read_seq`。
- 防止倒退，超过 `chat.last_seq` 时截断。
- web 按当前已展示最大 `sequence` 节流上报，并在切换会话/页面隐藏/卸载前 flush。
- 会话列表未读点由 `last_message.sequence > last_read_seq` 推导。
- 返回 last read sequence 和 read at。
- 可选推送 `conversation_state` 给当前用户其它设备。

验收：

- B 进入会话后后端 `last_read_seq` 推进到目标 sequence。
- 乱序或重复 read 请求不会让 `last_read_seq` 倒退。
- 新消息竞态下，前端不会因为空 read 请求误读到尚未展示的最新消息。
- 刷新页面后 `last_read_sequence` 仍为后端持久值。
- 当前端切换会话或页面隐藏时，pending 的最大 read sequence 会被 flush。

### 12. 其它消息类型逐个补齐

目标：按类型小步推进，不一次性混做。

建议顺序：

1. image
2. video
3. voice
4. file
5. card
6. link
7. template
8. location

每个类型验收：

- content schema 已确认。
- HTTP 发送成功。
- 历史消息可还原。
- WS 推送可渲染。
- content_preview 正确。

### 13. 群聊能力

目标：复用现有会话、消息、已读链路，把群聊补成可独立验收的最小闭环。群 ID 使用底层 `chat_id`，对 web 暴露为 `group_id`；群会话仍使用 `conversation_id` 发送消息。

已落地顺序：

1. 创建群：`POST /api/v1/groups`。
2. 群详情：`GET /api/v1/groups/{group_id}`。
3. 群成员列表：`GET /api/v1/groups/{group_id}/members`。
4. 群文本消息：复用 `POST /api/v1/messages`，`conversation_id` 传群会话 ID。
5. 群消息未读和已读：复用会话列表和 `POST /api/v1/conversations/{conversation_id}/read`。
6. 邀请成员：`POST /api/v1/groups/{group_id}/members`。
7. 退群：`POST /api/v1/groups/{group_id}/leave`。
8. 改群名：`PATCH /api/v1/groups/{group_id}`。
9. 群系统事件：群操作自动写入 `system_event` 消息。

#### 创建群

```http
POST /api/v1/groups
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "name": "项目群",
  "member_user_ids": ["u_002", "u_003"]
}
```

响应：

```json
{
  "code": 0,
  "data": {
    "group": {
      "id": "chat_100",
      "conversation_id": "conv_100",
      "name": "项目群",
      "avatar_url": "",
      "owner_id": "u_001",
      "member_count": 3,
      "status": "active",
      "created_at": "2026-07-05T12:00:00Z",
      "updated_at": "2026-07-05T12:00:00Z"
    },
    "conversation": {
      "id": "conv_100",
      "conversation_id": "conv_100",
      "chat_id": "chat_100",
      "chat_type": "group"
    }
  },
  "error": ""
}
```

#### 群详情

```http
GET /api/v1/groups/{group_id}
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "group": {
      "id": "chat_100",
      "conversation_id": "conv_100",
      "name": "项目群",
      "owner_id": "u_001",
      "member_count": 3,
      "status": "active",
      "created_at": "2026-07-05T12:00:00Z",
      "updated_at": "2026-07-05T12:00:00Z"
    },
    "members": [
      {
        "id": "chat_100:u_001",
        "group_id": "chat_100",
        "user_id": "u_001",
        "role": "owner",
        "status": "active",
        "joined_at": "2026-07-05T12:00:00Z",
        "invited_by": "u_001",
        "user_info": {
          "id": "u_001",
          "nickname": "Alice",
          "avatar": "",
          "status": "active"
        }
      }
    ]
  },
  "error": ""
}
```

#### 群成员列表

```http
GET /api/v1/groups/{group_id}/members?limit=20&cursor=
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {
    "items": [],
    "next_cursor": "",
    "has_more": false
  },
  "error": ""
}
```

#### 邀请成员

```http
POST /api/v1/groups/{group_id}/members
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "member_user_ids": ["u_004"]
}
```

响应返回更新后的 `group`。

#### 修改群名

```http
PATCH /api/v1/groups/{group_id}
Authorization: Bearer <access_token>
Content-Type: application/json
```

请求：

```json
{
  "name": "新项目群"
}
```

响应返回更新后的 `group`。

#### 退出群聊

```http
POST /api/v1/groups/{group_id}/leave
Authorization: Bearer <access_token>
```

响应：

```json
{
  "code": 0,
  "data": {},
  "error": ""
}
```

验收：

- 创建群后，创建者和被邀请成员都有 `chat_type=group` 的会话视图。
- 会话列表中的群会话带 `group_id`、`group_info`，`title` 使用群名。
- 群成员可以通过群会话 `conversation_id` 发送文本消息，消息写入所有当前成员 mailbox。
- 群已读使用当前会话已读接口，`last_read_sequence` 不倒退。
- 退群后当前用户会话设置 `left_at`，不再出现在会话列表，后续群消息不再投递给该用户。
- 修改群名后再次读取会话列表或群详情能看到新群名。
- 创建群、邀请成员、退群、修改群名会自动生成 `system_event` 消息；客户端不能通过 `POST /api/v1/messages` 直接发送 `system_event`。

#### 群系统事件

系统事件由群接口自动写入，不提供客户端直接发送入口。事件消息复用普通消息结构：

```json
{
  "message_type": "system_event",
  "content_preview": "Alice邀请Bob加入群聊",
  "content": {
    "event_type": "members_invited",
    "text": "Alice邀请Bob加入群聊",
    "title": "Alice邀请Bob加入群聊",
    "operator_id": "u_001",
    "target_user_ids": ["u_002"],
    "group_id": "chat_100",
    "group_name": "项目群",
    "before_value": "",
    "after_value": ""
  }
}
```

当前事件类型：

1. `group_created`：创建群。
2. `members_invited`：邀请成员。
3. `member_left`：成员退群。
4. `group_name_updated`：修改群名，`before_value` 为旧群名，`after_value` 为新群名。

### 14. 会话增强能力

建议顺序：

1. 置顶：已落地 `PATCH /api/v1/conversations/{conversation_id}/settings`。
2. 免打扰：已落地 `PATCH /api/v1/conversations/{conversation_id}/settings`。
3. 会话搜索。
4. 消息搜索：已落地会话内搜索 `GET /api/v1/conversations/{conversation_id}/messages/search`。
5. 消息撤回。
6. 消息转发。

## 开放问题

以下问题在实现前需要确认：

1. `auth/ticket` 是否复用同一路由同时承担签发和兑换，还是拆为更明确的 action。当前用户要求 ticket 换 token 使用 `auth/ticket`，实现前需要最终定请求形态。
2. `conversation_id` 是否直接等于内部单聊 `chat_id`，还是新建外部 ID 并在内部映射。文档倾向 web 只感知 `conversation_id`，不依赖底层命名。
3. cursor 是否使用 base64 JSON，内容包含 `last_activity_at`、`pinned`、`conversation_id`，实现前需定编码规则。
4. 已决策：`MessageDTO.sequence` 使用 `messages.seq`，它是 chat 维度消息序列；`user_mailbox.seq_id` 仅用于用户级同步流水。
