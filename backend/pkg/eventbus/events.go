package eventbus

// 约定:每个模块自己维护一个 events.go,定义"这个模块会往总线上发什么事件"。
// Subject 常量和 Payload 结构体放在一起,谁要订阅这个事件,直接 import 这个包,
// 不用去翻文档猜 subject 字符串或者payload字段。

const (
	// SubjectGroupMemberJoined 有新成员加入群聊时发布
	SubjectGroupMemberJoined = "im.group.member_joined"

	// SubjectGroupMemberLeft 有成员退出/被移出群聊时发布
	SubjectGroupMemberLeft = "im.group.member_left"

	// SubjectGroupMessageSent 群消息发送成功后发布(用于推送、未读计数、审计等下游消费)
	SubjectGroupMessageSent = "im.group.message_sent"
)

// GroupMemberJoined 对应 SubjectGroupMemberJoined
type GroupMemberJoined struct {
	GroupID  string `json:"group_id"`
	UserID   string `json:"user_id"`
	Operator string `json:"operator"` // 谁把他拉进来的,系统自动加入则为空
	JoinedAt int64  `json:"joined_at"`
}

// GroupMemberLeft 对应 SubjectGroupMemberLeft
type GroupMemberLeft struct {
	GroupID string `json:"group_id"`
	UserID  string `json:"user_id"`
	Reason  string `json:"reason"` // "left" 主动退出 / "kicked" 被移除
	LeftAt  int64  `json:"left_at"`
}

// GroupMessageSent 对应 SubjectGroupMessageSent
type GroupMessageSent struct {
	GroupID   string `json:"group_id"`
	MessageID string `json:"message_id"`
	SenderID  string `json:"sender_id"`
	SentAt    int64  `json:"sent_at"`
}
