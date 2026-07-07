package model

import (
	"context"
	"fmt"
	"sort"
	"time"

	"d-im/pkg/types"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Chat 会话实体 - 物理存在的会话
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

// ChatCollection 返回 chats 集合。
func ChatCollection(db *mongo.Database) *mongo.Collection {
	return db.Collection("chats")
}

// GenerateChatID 生成会话实体ID（UUID v7）。
func GenerateChatID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// GenerateSingleChatKey 生成单聊幂等键。它只用于唯一约束，不作为公开会话ID。
func GenerateSingleChatKey(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("%s:%s", uids[0], uids[1])
}

// CreateOrGetSingleChat 获取或创建单聊会话
func CreateOrGetSingleChat(ctx context.Context, coll *mongo.Collection, uid1, uid2 string) (*Chat, error) {
	singleKey := GenerateSingleChatKey(uid1, uid2)
	chatID := GenerateChatID()
	now := time.Now()

	filter := bson.M{
		"chat_type":  types.ChatTypeSingle,
		"single_key": singleKey,
	}
	update := bson.M{
		"$setOnInsert": bson.M{
			"chat_id":      chatID,
			"chat_type":    types.ChatTypeSingle,
			"members":      []string{uid1, uid2},
			"member_count": 2,
			"last_seq":     0,
			"created_at":   now,
		},
		"$set": bson.M{
			"single_key": singleKey,
			"updated_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var chat Chat
	err := coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// CreateGroupChat 创建群聊对应的消息会话实体。
func CreateGroupChat(ctx context.Context, coll *mongo.Collection, creatorUID string) (*Chat, error) {
	chatID := GenerateChatID()
	now := time.Now()
	chat := &Chat{
		ChatID:    chatID,
		ChatType:  types.ChatTypeGroup,
		LastSeq:   0,
		CreatedBy: creatorUID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := coll.InsertOne(ctx, chat)
	return chat, err
}

// GetChatMembers 查询群成员列表
func GetChatMembers(ctx context.Context, coll *mongo.Collection, chatID string) ([]string, error) {
	var chat Chat
	err := coll.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return chat.Members, nil
}

// FindChatByID 根据chatID查询Chat
func FindChatByID(ctx context.Context, coll *mongo.Collection, chatID string) (*Chat, error) {
	var chat Chat
	err := coll.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// NextChatMessageSeq 为指定 chat 原子分配下一条消息序号。
func NextChatMessageSeq(ctx context.Context, coll *mongo.Collection, chatID string) (int64, error) {
	now := time.Now()
	update := bson.M{
		"$inc": bson.M{"last_seq": 1},
		"$set": bson.M{"updated_at": now},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var chat Chat
	err := coll.FindOneAndUpdate(ctx, bson.M{"chat_id": chatID}, update, opts).Decode(&chat)
	if err != nil {
		return 0, err
	}
	return chat.LastSeq, nil
}

// uniqueStrings 字符串切片去重
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
