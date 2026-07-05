下面是一份可直接交给“重构 AI”的前端交互设计说明，基于当前项目 `im-frontend` 的现有实现整理。

**重构目标**

重构时不要只复刻页面外观，要保留 IM 的核心交互契约：登录态恢复、单标签页 WebSocket 主控、会话列表 cursor 分页、Pinia 全局会话状态、消息乐观发送、历史消息向上加载、输入框 IME 保护、上传预览与失败重试。

**技术栈与结构**

前端是 Vue 3 + Vite + Pinia + Vue Router。

关键文件：

- 会话列表状态：[conversationList.ts](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/stores/conversationList.ts)
- IM WebSocket/SDK 状态：[im.ts](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/stores/im.ts)
- 用户登录状态：[user.ts](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/stores/user.ts)
- 单标签页主控：[imTab.ts](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/stores/imTab.ts)
- 聊天主页面：[chat.vue](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/views/im/chat.vue)
- 会话列表组件：[ConversationList.vue](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/components/im/ConversationList.vue)
- 多行输入框：[MultilineInput.vue](/Users/jianboo/Sites/donktech/d-im/im-frontend/src/components/im/MultilineInput.vue)

**登录与会话入口**

IM 不提供独立账号密码登录。`/im/login` 只提示“请从业务系统进入聊天”。

真实入口是 `/im/enter`，通过业务系统传入 entry token，调用 `establishSession()` 换取 access token 和 refresh token。

登录态要求：

- access token 存在 Pinia `userStore.token`
- refresh token 存 localStorage，key 为 `d-im-refresh-token`
- 页面初始化时自动尝试 refresh token 恢复登录
- token 快过期时提前刷新
- 页面重新 visible 时触发 token 校验
- 多标签页之间通过 BroadcastChannel `d-im-auth` 同步 token 更新和 logout
- refresh token 有 localStorage 锁，避免多标签同时刷新
- auth 失败时展示 session expired 状态，确认后 logout

退出登录时必须：

- 清空 token、refresh token、userInfo
- 关闭 WebSocket
- reset 单标签页主控
- reset 会话列表状态
- 可选调用后端注销 session
- 跳转 `/im/login`

**单标签页 IM 主控**

项目设计为同一用户只允许一个标签页持有 WebSocket 连接。

UI 要求：

- suspended 时主聊天区显示“已在其他标签页打开”
- 提供“在此标签页使用”按钮
- 点击后 `claimActive()`，当前页接管 WebSocket，并重新初始化聊天
- suspended 时关闭当前 tab 的 IM connection，避免多连接推送重复

**IM SDK 与连接状态**

`imStore` 持有：

- `imSDK`
- `isConnected`
- message handlers
- connection handlers
- reconnect timer
- manualDisconnect
- sdkToken

连接设计：

- 只有 primary tab 可以建立 WebSocket
- init SDK 前必须有 token
- WebSocket disconnected 且非手动断开时自动重连
- 最大重连 5 次
- 重连延迟指数增长，基础 3 秒
- connected 后启动 heartbeat
- token 刷新后要 `imSDK.updateToken(token)`
- 顶部导航在断连时显示 loading/reconnect 图标

收到非 ping/pong、非自己发送、非 muted 的消息时，需要触发页面标题通知。

**路由设计**

路由关系：

- `/` 重定向 `/im/home`
- `/im/home` 移动端/首页式会话列表
- `/im/chat` 聊天布局但未选中会话
- `/im/chat/:conversationId` 聊天布局并选中会话
- `/im/video-player` 视频消息播放页
- `/im/enter` 业务系统进入页
- `/im/login` 无独立登录提示页

鉴权规则：

- `/im/**` 需要 token
- 未登录跳 `/im/login?redirect=当前路径`
- 已登录访问 `/im/login` 时跳 redirect 或 `/im/home`

**整体聊天布局**

桌面端 `chat.vue` 是三段结构：

- 左侧固定 300px 会话侧栏
- 右侧聊天主区
- 底部用户菜单

左侧侧栏：

- 顶部标题“消息”
- 搜索按钮打开会话搜索弹窗
- 中间嵌入 `ConversationList`
- 底部展示当前用户头像、昵称、菜单按钮
- 菜单里展示用户头像昵称和“退出登录”
- 点击页面其他区域关闭菜单

未选中会话：

- 右侧显示空状态图标

选中会话：

- 顶部 nav 显示会话标题
- 移动端显示返回按钮
- 右侧更多按钮打开会话信息面板
- 连接中显示 loading 图标
- 中间消息列表
- 底部输入区

系统用户会话：

- 如果 peer user type 是 `system`
- 底部不显示输入框
- 显示提示：“系统消息，暂不支持回复”

**会话列表 Pinia 管理**

会话列表必须放在 Pinia 中全局管理，不能做成组件局部状态，否则聊天页侧栏、搜索弹窗、WS 推送、未读数会不同步。

`conversationListStore` 状态：

- `conversations`
- `searchResults`
- `loading`
- `loadingMore`
- `searching`
- `searchingMore`
- `error`
- `searchError`
- `hasMore`
- `nextCursor`
- `searchHasMore`
- `searchNextCursor`
- `activeSearchKeyword`
- `pendingScrollRequest`

分页规则：

- 普通会话列表 page size = 20
- 首屏调用 `sdk.getConversationPage({ limit: 20, active_conversation_id })`
- 加载更多调用 `sdk.getConversationPage({ limit: 20, cursor: nextCursor })`
- 后端返回 `items`, `next_cursor`, `has_more`
- `hasMore` 和 `nextCursor` 同时控制是否允许继续加载
- 距离列表底部小于 80px 时触发加载更多
- load 和 loadMore 要用 promise guard 防重复请求
- 新页数据要按 id 去重 merge
- merge 后重新排序

搜索分页：

- 搜索结果和普通列表完全分离
- 搜索 keyword trim 后为空时清空 searchResults
- 搜索 debounce 300ms
- 搜索调用 `sdk.getConversationPage({ limit: 20, q: keyword })`
- 搜索加载更多调用 `cursor + q`
- 用 requestId 防止旧搜索请求覆盖新搜索结果
- 搜索中、搜索错误、搜索加载更多状态独立于普通列表

排序规则：

- 置顶会话永远在前
- 其他按 `last_activity || last_message.created_at || updated_at` 倒序

会话注入：

- 如果收到 WS 消息但会话不在列表中，需要先 `getConversation(conversationId)` 或 `activateConversation(conversationId)` 拉取并 upsert
- 进入某个 conversationId 时，如果列表没有该会话，调用 `ensureConversationInList(conversationId, { activateIfMissing: true })`
- 获取失败时回到 `/im/chat`

**会话列表展示**

每个会话项展示：

- 圆形头像，48x48
- 昵称，单行省略
- 右侧时间
- 最后一条消息预览，单行省略
- 免打扰 icon
- 未读角标
- 可选 preview image，48x48 圆角

未读规则：

- unread count <= 0 不展示
- 普通会话展示红色数字 badge
- 超过 99 显示 `99+`
- muted 会话有未读时只展示红点，不展示数字
- 进入会话时本地清除该会话未读
- 当前聊天中收到对方消息后调用 `markConversationRead()`，并清本地未读

时间规则：

- 今天显示 `HH:mm`
- 昨天显示 `昨天`
- 今年显示 `MM-DD`
- 跨年显示 `YYYY-MM-DD`
- 空时间或 Go zero time 不显示

最后一条消息预览：

- 优先使用后端 `preview_text`
- 没有则用 `content`
- 群聊中，如果最后消息来自别人且不是系统事件，前缀为 `发送者昵称: 内容`

点击行为：

- 点击非当前会话：清未读，跳转 `/im/chat/:conversationId`
- 侧栏内使用 router.replace
- 首页列表使用 router.push
- 点击当前会话：取消选中，跳 `/im/chat`
- 搜索弹窗里点击当前会话：只清未读并滚动侧栏定位，不取消选中

**消息列表加载**

消息列表是局部状态，属于当前 `chat.vue`：

- `messages`
- `loading`
- `hasMore`
- `firstLoad`
- `initialized`

首屏：

- 进入会话后先确保 SDK 和 WebSocket connected
- 拉取会话信息、用户信息
- 调用 `getConversationMessages(conversationId, { limit: 20 })`
- 消息按 created_at 正序排列
- 首屏加载完成后滚到底部
- 调用 `markConversationRead()`

历史消息：

- 向上滚动到 `scrollTop <= 50` 时加载更早消息
- 用当前最旧的非 temp 消息 id 作为 `before_id`
- 请求 `getConversationMessages(conversationId, { limit: 20, before_id })`
- 新旧消息 merge 去重后排序
- 加载前记录 `oldScrollTop` 和 `oldScrollHeight`
- 插入历史消息后恢复滚动位置：`newScrollHeight - oldScrollHeight + oldScrollTop`
- 如果返回条数小于 page size，则 `hasMore = false`
- 顶部显示“没有更多消息了”，但仅在非首屏且消息数大于 pageSize 时显示

断线重连后：

- 如果当前有选中会话，连接从 false 变 true 后调用 `syncLatestMessages()`
- 使用最新一条非 temp 消息 id 作为 `after_id`
- 循环分页拉取直到返回少于 pageSize
- merge 后，如果用户之前接近底部，则滚到底部
- 同步已读状态

滚动到底部规则：

- `isNearBottom`: 距离底部小于 120px
- 收到自己发的消息强制滚到底
- 收到别人消息，仅当当前接近底部时滚动
- 输入框 focus 时强制滚到底

**消息合并与去重**

消息合并必须支持乐观消息和服务端确认合并。

匹配优先级：

- 如果有 `client_message_id`，用它匹配
- 否则用 `id` 匹配

合并规则：

- 新消息覆盖旧消息字段
- 自己发送的服务端消息状态置为 `sent`
- 去重后按 `created_at` 正序
- 如果时间相同，按 id 字符串排序

临时消息：

- id 格式：`temp-${clientMessageId}`
- clientMessageId 格式：`cmid_${currentUserId}_${randomUUID}`
- 发送前立即插入列表，status = `sending`
- 服务端返回后用 `client_message_id` 合并为正式消息
- 失败时 status = `failed`

**消息时间轴**

消息列表不只是消息数组，还要插入时间分割线。

规则：

- 第一条消息前显示时间
- 与上一条消息间隔 >= 15 分钟时显示时间
- 今天显示 `H:mm`
- 今年非今天显示 `M-D H:mm`
- 跨年显示 `YYYY-M-D H:mm`
- 时间分割线居中、弱化样式

**消息项布局与样式**

普通消息：

- 对方消息左侧，自己消息右侧
- 对方消息显示头像
- 自己消息使用当前用户头像
- 群聊里对方消息显示发送者昵称
- 私聊不显示发送者昵称
- 系统事件消息不显示头像，不走左右气泡，居中展示

文本消息：

- 最大宽度 `min(80%, 768px)`
- 对方灰底气泡
- 自己蓝色渐变气泡
- 支持换行，`white-space: pre-line`
- 自动识别 http/https 链接，转成可点击 `<a target="_blank" rel="noopener noreferrer">`
- 文本必须 HTML escape 后再替换链接

图片消息：

- 本地选择后立即显示 blob 预览
- 上传中显示半透明遮罩和“上传中”
- 图片尺寸根据 meta 或实际 natural size 计算
- 最大约 220x280，最小 96x96
- 点击图片打开全屏预览
- 预览支持当前会话所有图片列表和索引
- 上传成功后释放 blob URL
- 上传失败显示重试按钮

视频消息：

- 卡片式展示
- 有 poster 显示 poster，否则显示视频占位图标
- 中央播放按钮
- 有 duration 时右下角显示 `m:ss`
- 根据 meta width/height 决定 16:9 或 9:16
- 点击新窗口打开 `/im/video-player?url=&poster=&type=`

卡片消息：

- 最大宽度 320px，宽度 80%
- 图片 4:3
- 标题最多 2 行
- 描述最多 2 行
- 可显示 price_text
- 点击时先弹确认框：“打开链接”，确认后 `window.open`

链接消息：

- 横向小卡片
- 左侧标题描述，右侧 80x80 缩略图
- 无图或加载失败显示 Placeholder
- 点击直接新窗口打开

系统事件：

- 居中
- max-width `min(76%, 520px)`
- 小字号灰色
- 文案由 `formatSystemEventMessage()` 生成

失败重试：

- 只有 status = `failed` 显示重试按钮
- 重试按钮是气泡左侧小圆按钮
- 文本/卡片/视频重试重新调用 sendMessage
- 图片失败如果还有 localFile，则重新上传再发送
- localFile 丢失时提示“图片文件已失效，请重新选择”

**发送输入框**

输入框组件必须保留这些交互：

- textarea 自适应高度
- 桌面 minRows = 2，maxRows = 15
- 移动端 minRows = 1，maxRows = 10
- 行高 24px，垂直 padding 16px
- 内容为空时高度回到 min height
- 超过 max height 后 textarea 内部滚动
- 禁用 autocomplete/autocorrect/autocapitalize/spellcheck
- 有隐藏 honeypot input，防止 Chrome 自动填充进聊天框

键盘规则：

- IME 组合输入中，Enter 不发送
- `e.isComposing`、内部 composing 状态、`keyCode === 229` 都视为 IME
- Enter：发送
- Shift+Enter / Ctrl+Enter / Command+Enter：插入换行
- 插入换行后保持光标位置

发送按钮：

- 圆形上箭头按钮
- `messageText.trim()` 为空时 disabled
- 点击或 Enter 发送
- 发送时 trim 内容
- 发送后立即清空输入框
- 每个会话维护 draft
- 切换会话时保存上一会话草稿，恢复新会话草稿

附件按钮：

- 当前只支持图片
- accept jpg/jpeg/png/gif/webp
- 支持一次多选
- 最多 9 张
- 单张最大 10MB
- 格式不符合提示“仅支持 JPG、PNG、GIF、WEBP 图片”
- 超限提示“一次最多选择9张图片”或“图片大小不能超过10MB”
- 每张图片并发上传
- 选择后立即插入本地预览消息
- 上传成功后发送 image message
- 上传失败消息状态 failed，可点击重试

**会话搜索**

会话搜索入口在侧栏顶部搜索按钮。

搜索弹窗应使用同一个 `ConversationList`，但传入：

- `searchMode`
- `searchKeyword`
- `navigateMode="replace"`
- 当前 active conversation id

交互：

- 输入为空时不展示普通会话
- 空状态文案：“搜索联系人或用户 ID”
- 有关键词但无结果：“没有找到相关会话”
- 搜索 loading 只在有关键词时显示
- 搜索结果分页独立
- 点击搜索结果后跳转/选择会话，并同步侧栏滚动到该会话

**会话信息**

顶部更多按钮打开会话信息面板。

现有逻辑：

- 私聊 participants 来自 conversation participants、peer_user_info、targetUser、userMap 合并
- 群聊 participants 暂为空
- 邀请成员目前提示“邀请成员稍后开放”
- 会话详情打开后切换会话要关闭

重构时至少要保留：

- 面板开关状态
- 当前会话信息传入
- 成员信息合并
- 切换会话关闭面板

**用户资料缓存**

`useUserProfiles()` 用于缓存用户信息。

需要在这些地方 merge：

- 会话列表中的 `peer_user_info`
- 最后一条消息的 `sender_profile`
- 消息列表中每条消息的 `sender_profile`
- 当前聊天 peer user
- WS 收到的新消息 sender profile

展示优先级：

- 消息头像：message.sender_profile.avatar > userMap[from_id].avatar > targetUser.avatar
- 会话头像：conversation.display_avatar > group avatar > peer avatar > fallback
- 会话名：conversation.display_name > group name > peer nickname > `用户${peerId.slice(-4)}` > `未知用户`

**移动端要求**

当前代码只做了部分移动适配，重构时要明确保留：

- viewport <= 767px 视为移动端
- 移动端输入框 minRows 1，maxRows 10
- 移动端顶部显示返回按钮
- 返回优先 `router.back()`，无 history back 时跳 `/im/home`
- 使用 `100dvh`，避免移动浏览器地址栏造成高度问题
- 列表和消息区域要 `-webkit-overflow-scrolling: touch`

**错误与空状态**

会话列表：

- loading 显示 spinner
- 普通错误：会话列表加载失败
- 未登录错误：登录状态已失效
- 加载更多失败：加载更多失败
- 搜索失败：搜索失败
- 错误块提供“重试”按钮
- 无普通会话：暂无消息

聊天初始化：

- 无 token 跳 login
- conversation activate 失败跳 `/im/chat`
- 初始化失败 toast：“连接失败，请稍后重试”
- 历史消息失败 toast：“加载消息失败”
- 发送失败 toast：“发送失败”
- 上传失败 toast：“上传失败”或“上传失败，点击重试”

**重构验收清单**

重构完成后至少验证这些场景：

1. 未登录访问 `/im/chat/xxx` 会跳 `/im/login?redirect=...`
2. `/im/enter` 能完成 token exchange 并进入 redirect
3. 多标签只保留一个 WebSocket，另一个显示 suspended
4. 点击“在此标签页使用”能接管连接
5. 会话列表首屏 cursor 加载正常
6. 滚动到底部自动加载更多会话
7. 搜索会话 debounce、分页、空状态正常
8. 收到 WS 消息后会话列表更新最后消息、时间、未读数并重排
9. 进入会话后未读数清零并调用 mark read
10. 聊天首屏消息正序并滚到底部
11. 向上滚动加载历史消息后视口位置不跳
12. 断线重连后能拉取 after_id 的遗漏消息
13. 文本消息乐观发送、成功合并、失败重试正常
14. 中文输入法选词时 Enter 不误发送
15. Shift/Ctrl/Command+Enter 能换行
16. 切换会话时草稿分别保存和恢复
17. 图片选择后立即本地预览，上传中有遮罩
18. 图片上传成功后变正式消息，失败可重试
19. 图片预览能在当前会话图片列表中切换
20. 系统用户会话不显示输入框，只显示不可回复提示