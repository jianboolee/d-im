package model

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"sort"
	"time"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrUserIDRequired     = errors.New("user ID is required")
	ErrSingleChatWithSelf = errors.New("cannot create single chat with self")
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

// NewSingleChatKey 为两个不透明的第三方用户 ID 创建对称、无歧义的单聊唯一键。
// 用户 ID 按原始字节处理，不做空白、大小写或 Unicode 归一化。
func NewSingleChatKey(userID1, userID2 string) (string, error) {
	if userID1 == "" || userID2 == "" {
		return "", ErrUserIDRequired
	}
	if userID1 == userID2 {
		return "", ErrSingleChatWithSelf
	}

	userIDs := []string{userID1, userID2}
	sort.Strings(userIDs)

	hash := sha256.New()
	_, _ = hash.Write([]byte("single-chat-key:v1"))
	var length [8]byte
	for _, userID := range userIDs {
		binary.BigEndian.PutUint64(length[:], uint64(len(userID)))
		_, _ = hash.Write(length[:])
		_, _ = hash.Write([]byte(userID))
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
