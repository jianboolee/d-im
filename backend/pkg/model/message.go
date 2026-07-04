package model

import (
	"time"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message 消息主模型 - MongoDB文档结构
type Message struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MsgID    string             `bson:"msg_id" json:"msg_id"` // 业务消息ID(雪花ID/UUID)
	ChatID   string             `bson:"chat_id" json:"chat_id"`
	ChatType types.ChatType     `bson:"chat_type" json:"chat_type"`

	// 发送者信息
	FromUID  string `bson:"from_uid" json:"from_uid"`
	FromName string `bson:"from_name,omitempty" json:"from_name,omitempty"`

	// 消息类型和内容
	MsgType types.MessageType `bson:"msg_type" json:"msg_type"`
	Content interface{}       `bson:"content" json:"content"` // 多态内容

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

// MessageIndex 消息索引模型(分表策略)
type MessageIndex struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	MsgID      string             `bson:"msg_id"`
	ChatID     string             `bson:"chat_id"`
	ChatType   types.ChatType     `bson:"chat_type"`
	FromUID    string             `bson:"from_uid"`
	MsgType    types.MessageType  `bson:"msg_type"`
	ClientTime time.Time          `bson:"client_time"`
	CreatedAt  time.Time          `bson:"created_at"`
}
