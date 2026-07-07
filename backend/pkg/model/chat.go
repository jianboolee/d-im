package model

import (
	"context"
	"fmt"
	"sort"
	"time"

	"d-im/pkg/snowflake"
	"d-im/pkg/types"

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
	Name        string             `bson:"name,omitempty" json:"name,omitempty"`
	Avatar      string             `bson:"avatar,omitempty" json:"avatar,omitempty"`
	OwnerUID    string             `bson:"owner_uid,omitempty" json:"owner_uid,omitempty"`
	Members     []string           `bson:"members,omitempty" json:"members,omitempty"`
	MemberCount int                `bson:"member_count" json:"member_count"`
	LastSeq     int64              `bson:"last_seq" json:"last_seq"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ChatIDManager ChatID管理器
type ChatIDManager struct {
	chatColl *mongo.Collection
	idGen    *snowflake.Generator
}

// NewChatIDManager 创建ChatID管理器
func NewChatIDManager(db *mongo.Database, idGen *snowflake.Generator) *ChatIDManager {
	return &ChatIDManager{
		chatColl: db.Collection("chats"),
		idGen:    idGen,
	}
}

// GenerateSingleChatKey 生成单聊幂等键。它只用于唯一约束，不作为公开会话ID。
func GenerateSingleChatKey(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("%s:%s", uids[0], uids[1])
}

// GenerateSingleChatID 保留给旧 demo/脚本兼容；真实创建逻辑不再为新会话使用语义化ID。
func GenerateSingleChatID(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("single_%s_%s", uids[0], uids[1])
}

// GenerateChatID 生成不可语义化的会话实体ID。
func (m *ChatIDManager) GenerateChatID() string {
	return "chat_" + m.idGen.GenerateString()
}

// CreateOrGetSingleChat 获取或创建单聊会话
func (m *ChatIDManager) CreateOrGetSingleChat(ctx context.Context, uid1, uid2 string) (*Chat, error) {
	singleKey := GenerateSingleChatKey(uid1, uid2)
	legacyChatID := GenerateSingleChatID(uid1, uid2)
	chatID := m.GenerateChatID()
	now := time.Now()

	filter := bson.M{"$or": bson.A{
		bson.M{"chat_type": types.ChatTypeSingle, "single_key": singleKey},
		bson.M{"chat_id": legacyChatID},
	}}
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
	err := m.chatColl.FindOneAndUpdate(ctx, filter, update, opts).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// CreateGroupChat 创建群聊
func (m *ChatIDManager) CreateGroupChat(ctx context.Context, name string, ownerUID string, memberUIDs []string) (*Chat, error) {
	chatID := m.GenerateChatID()

	allMembers := append([]string{ownerUID}, memberUIDs...)
	allMembers = uniqueStrings(allMembers)

	now := time.Now()
	chat := &Chat{
		ChatID:      chatID,
		ChatType:    types.ChatTypeGroup,
		Name:        name,
		OwnerUID:    ownerUID,
		Members:     allMembers,
		MemberCount: len(allMembers),
		LastSeq:     0,
		CreatedBy:   ownerUID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := m.chatColl.InsertOne(ctx, chat)
	return chat, err
}

// AddMember 添加群成员
func (m *ChatIDManager) AddMember(ctx context.Context, chatID string, uid string) error {
	filter := bson.M{
		"chat_id":   chatID,
		"chat_type": types.ChatTypeGroup,
		"members":   bson.M{"$ne": uid},
	}
	update := bson.M{
		"$addToSet": bson.M{"members": uid},
		"$inc":      bson.M{"member_count": 1},
		"$set":      bson.M{"updated_at": time.Now()},
	}
	_, err := m.chatColl.UpdateOne(ctx, filter, update)
	return err
}

// RemoveMember 移除群成员
func (m *ChatIDManager) RemoveMember(ctx context.Context, chatID string, uid string) error {
	filter := bson.M{
		"chat_id":   chatID,
		"chat_type": types.ChatTypeGroup,
		"members":   uid,
	}
	update := bson.M{
		"$pull": bson.M{"members": uid},
		"$inc":  bson.M{"member_count": -1},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := m.chatColl.UpdateOne(ctx, filter, update)
	return err
}

// UpdateGroupName 修改群名称。
func (m *ChatIDManager) UpdateGroupName(ctx context.Context, chatID string, name string) (*Chat, error) {
	update := bson.M{
		"$set": bson.M{
			"name":       name,
			"updated_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var chat Chat
	err := m.chatColl.FindOneAndUpdate(ctx, bson.M{
		"chat_id":   chatID,
		"chat_type": types.ChatTypeGroup,
	}, update, opts).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetMembers 查询群成员列表
func (m *ChatIDManager) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	var chat Chat
	err := m.chatColl.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return chat.Members, nil
}

// FindByChatID 根据chatID查询Chat
func (m *ChatIDManager) FindByChatID(ctx context.Context, chatID string) (*Chat, error) {
	var chat Chat
	err := m.chatColl.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// NextMessageSeq 为指定 chat 原子分配下一条消息序号。
func (m *ChatIDManager) NextMessageSeq(ctx context.Context, chatID string) (int64, error) {
	now := time.Now()
	update := bson.M{
		"$inc": bson.M{"last_seq": 1},
		"$set": bson.M{"updated_at": now},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var chat Chat
	err := m.chatColl.FindOneAndUpdate(ctx, bson.M{"chat_id": chatID}, update, opts).Decode(&chat)
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
