package model

import (
	"d-im/pkg/types"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserMailbox 用户消息信箱 - 用于消息分发和同步
type UserMailbox struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty"`
	UID        string              `bson:"uid"`
	ChatID     string              `bson:"chat_id"`
	MsgID      string              `bson:"msg_id"`
	MessageSeq int64               `bson:"message_seq"`
	SeqID      string              `bson:"seq_id"` // 用户维度同步流水序号(UUID v7)
	Status     types.MessageStatus `bson:"status"`
	ReadAt     *time.Time          `bson:"read_at,omitempty"`
	CreatedAt  time.Time           `bson:"created_at"`
}
