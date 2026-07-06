package model

import (
	"encoding/json"
	"time"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message 消息主模型 - MongoDB文档结构
type Message struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MsgID       string             `bson:"msg_id" json:"msg_id"` // 业务消息ID(雪花ID/UUID)
	ChatID      string             `bson:"chat_id" json:"chat_id"`
	ChatType    types.ChatType     `bson:"chat_type" json:"chat_type"`
	Seq         int64              `bson:"seq" json:"sequence"`
	ClientMsgID string             `bson:"client_message_id,omitempty" json:"client_message_id,omitempty"`

	// 发送者信息
	SenderID   string `bson:"sender_id" json:"sender_id"`
	SenderName string `bson:"sender_name,omitempty" json:"sender_name,omitempty"`

	// 消息类型和内容
	MsgType        types.MessageType `bson:"msg_type" json:"msg_type"`
	Content        interface{}       `bson:"content" json:"content"` // 多态内容
	ContentPreview string            `bson:"content_preview" json:"content_preview"`

	// 引用消息
	QuoteMsgID string              `bson:"quote_msg_id,omitempty" json:"quote_msg_id,omitempty"`
	QuoteMsg   *types.QuoteMessage `bson:"quote_msg,omitempty" json:"quote_msg,omitempty"`

	// 消息状态
	Status types.MessageStatus `bson:"status" json:"status"`

	// 扩展属性
	Ext        map[string]interface{} `bson:"ext,omitempty" json:"ext,omitempty"`
	MentionAll bool                   `bson:"mention_all" json:"mention_all"`

	// 已读信息(群聊场景)
	ReadCount   int `bson:"read_count,omitempty" json:"read_count,omitempty"`
	UnReadCount int `bson:"unread_count,omitempty" json:"unread_count,omitempty"`

	// 撤回信息
	IsRecalled bool       `bson:"is_recalled" json:"is_recalled"`
	RecallTime *time.Time `bson:"recall_time,omitempty" json:"recall_time,omitempty"`

	// 时间戳
	ClientTime time.Time  `bson:"client_time" json:"client_time"`
	ServerTime time.Time  `bson:"server_time" json:"server_time"`
	CreatedAt  time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt  *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// NormalizeContent converts Mongo-decoded content into the concrete backend
// content type for this message. Read paths should call this once at the
// repository boundary so service and gateway code do not depend on BSON shapes.
func (m *Message) NormalizeContent() {
	if m == nil {
		return
	}
	m.Content = NormalizeMessageContent(m.MsgType, m.Content)
}

func NormalizeMessageContent(msgType types.MessageType, content interface{}) interface{} {
	if content == nil {
		return map[string]interface{}{}
	}

	if typed, ok := content.(types.ContentType); ok {
		return typed
	}

	switch msgType {
	case types.MessageTypeText:
		return decodeMessageContent[types.TextContent](content)
	case types.MessageTypeImage:
		return decodeMessageContent[types.ImageContent](content)
	case types.MessageTypeVideo:
		return decodeMessageContent[types.VideoContent](content)
	case types.MessageTypeVoice:
		return decodeMessageContent[types.VoiceContent](content)
	case types.MessageTypeCard:
		return decodeMessageContent[types.CardContent](content)
	case types.MessageTypeLink:
		return decodeMessageContent[types.LinkContent](content)
	case types.MessageTypeTemplate:
		return decodeMessageContent[types.TemplateContent](content)
	case types.MessageTypeFile:
		return decodeMessageContent[types.FileContent](content)
	case types.MessageTypeLocation:
		return decodeMessageContent[types.LocationContent](content)
	default:
		return ContentMap(content)
	}
}

func ContentMap(content interface{}) map[string]interface{} {
	if content == nil {
		return map[string]interface{}{}
	}
	if value, ok := content.(bson.M); ok {
		return map[string]interface{}(value)
	}
	if value, ok := content.(bson.D); ok {
		return value.Map()
	}
	if value, ok := content.(map[string]interface{}); ok {
		return value
	}

	data, err := json.Marshal(content)
	if err != nil {
		return map[string]interface{}{}
	}

	var normalized map[string]interface{}
	if err := json.Unmarshal(data, &normalized); err != nil {
		return map[string]interface{}{}
	}
	if normalized == nil {
		return map[string]interface{}{}
	}
	return normalized
}

func decodeMessageContent[T types.ContentType](content interface{}) T {
	if value, ok := content.(T); ok {
		return value
	}

	var typed T
	if data, err := bson.Marshal(content); err == nil {
		if err := bson.Unmarshal(data, &typed); err == nil {
			return typed
		}
	}

	data, err := json.Marshal(ContentMap(content))
	if err != nil {
		return typed
	}
	_ = json.Unmarshal(data, &typed)
	return typed
}

// MessageIndex 消息索引模型(分表策略)
type MessageIndex struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	MsgID      string             `bson:"msg_id"`
	ChatID     string             `bson:"chat_id"`
	ChatType   types.ChatType     `bson:"chat_type"`
	Seq        int64              `bson:"seq"`
	SenderID   string             `bson:"sender_id"`
	MsgType    types.MessageType  `bson:"msg_type"`
	ClientTime time.Time          `bson:"client_time"`
	CreatedAt  time.Time          `bson:"created_at"`
}
