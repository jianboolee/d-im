package model

import (
	"fmt"
	"sort"
	"time"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Chat 会话实体 - 消息会话的物理存在。
type Chat struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChatID      string             `bson:"chat_id" json:"chat_id"`
	ChatType    types.ChatType     `bson:"chat_type" json:"chat_type"`
	SingleKey   string             `bson:"single_key,omitempty" json:"single_key,omitempty"`
	Members     []string           `bson:"members,omitempty" json:"members,omitempty"`
	MemberCount int                `bson:"member_count" json:"member_count"`
	LastSeq     int64              `bson:"last_seq" json:"last_seq"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// GenerateSingleChatKey 生成单聊幂等键，仅用于唯一约束，不作为公开会话 ID。
func GenerateSingleChatKey(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("%s:%s", uids[0], uids[1])
}
