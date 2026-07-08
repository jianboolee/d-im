package events

// ============ 消息事件 ============

const (
	SubjectMessageSend  = "im.message.send"
	SubjectMessagePush  = "im.message.push"
	SubjectMessageEvent = "im.message.event"
)

const (
	SubjectMessageSent      = "im.message.sent"      // 消息发送成功
	SubjectMessageDelivered = "im.message.delivered" // 消息送达
	SubjectMessageRead      = "im.message.read"      // 消息已读
	SubjectMessageRecalled  = "im.message.recalled"  // 消息撤回
	SubjectMessageDeleted   = "im.message.deleted"   // 消息删除
)
