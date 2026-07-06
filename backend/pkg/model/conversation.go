package model

import (
	"time"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Conversation 会话模型 - 最新消息摘要（用户维度的会话视图）
type Conversation struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID string             `bson:"conversation_id" json:"conversation_id"`
	UID            string             `bson:"uid" json:"uid"`
	ChatID         string             `bson:"chat_id" json:"chat_id"`
	ChatType       types.ChatType     `bson:"chat_type" json:"chat_type"`

	// 用户个性化设置
	IsTop      bool   `bson:"is_top" json:"is_top"`
	IsMuted    bool   `bson:"is_muted" json:"is_muted"`
	IsArchived bool   `bson:"is_archived" json:"is_archived"`
	CustomName string `bson:"custom_name,omitempty" json:"custom_name,omitempty"`

	// 用户维度的消息状态
	LastReadSeq int64              `bson:"last_read_seq" json:"last_read_seq"`
	LastReadAt  *time.Time         `bson:"last_read_at,omitempty" json:"last_read_at,omitempty"`
	UnreadCount int                `bson:"unread_count" json:"unread_count"`
	LastMsg     *types.LastMessage `bson:"last_msg,omitempty" json:"last_msg,omitempty"`

	// 用户与这个会话的关系
	JoinedAt time.Time  `bson:"joined_at" json:"joined_at"`
	LeftAt   *time.Time `bson:"left_at,omitempty" json:"left_at,omitempty"`

	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// UserMailbox 用户消息信箱 - 用于消息分发和同步
type UserMailbox struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	UID        string              `bson:"uid"`
	ChatID     string              `bson:"chat_id"`
	MsgID      string              `bson:"msg_id"`
	MessageSeq int64               `bson:"message_seq"`
	SeqID      int64               `bson:"seq_id"` // 用户维度同步流水序号
	Status     types.MessageStatus `bson:"status"`
	ReadAt     *time.Time          `bson:"read_at,omitempty"`
	CreatedAt  time.Time           `bson:"created_at"`
}
