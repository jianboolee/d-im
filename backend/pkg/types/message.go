package types

import "time"

// MessageType 消息类型枚举
type MessageType string

const (
	MessageTypeText        MessageType = "text"
	MessageTypeSystemEvent MessageType = "system_event"
	MessageTypeImage       MessageType = "image"
	MessageTypeVideo       MessageType = "video"
	MessageTypeVoice       MessageType = "voice"
	MessageTypeCard        MessageType = "card"
	MessageTypeLink        MessageType = "link"
	MessageTypeTemplate    MessageType = "template"
	MessageTypeFile        MessageType = "file"
	MessageTypeLocation    MessageType = "location"
)

// MessageStatus 消息状态
type MessageStatus string

const (
	MessageStatusSending   MessageStatus = "sending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
	MessageStatusRecalled  MessageStatus = "recalled"
)

// ChatType 会话类型
type ChatType string

const (
	ChatTypeSingle  ChatType = "single"
	ChatTypeGroup   ChatType = "group"
	ChatTypeSystem  ChatType = "system"
	ChatTypeChannel ChatType = "channel"
)

// ContentType 内容类型接口
type ContentType interface {
	Type() MessageType
	Validate() error
}

// QuoteMessage 引用消息摘要
type QuoteMessage struct {
	MsgID          string      `bson:"msg_id" json:"msg_id"`
	SenderID       string      `bson:"sender_id" json:"sender_id"`
	SenderName     string      `bson:"sender_name" json:"sender_name"`
	MsgType        MessageType `bson:"msg_type" json:"msg_type"`
	ContentPreview string      `bson:"content_preview" json:"content_preview"`
}

// LastMessage 最后一条消息摘要
type LastMessage struct {
	MsgID          string      `bson:"msg_id" json:"msg_id"`
	Seq            int64       `bson:"seq" json:"sequence"`
	SenderID       string      `bson:"sender_id" json:"sender_id"`
	SenderName     string      `bson:"sender_name,omitempty" json:"sender_name,omitempty"`
	MsgType        MessageType `bson:"msg_type" json:"msg_type"`
	ContentPreview string      `bson:"content_preview" json:"content_preview"`
	ClientTime     time.Time   `bson:"client_time" json:"client_time"`
}
