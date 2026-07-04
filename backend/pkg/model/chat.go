package model

import (
	"context"
	"fmt"
	"sort"
	"time"

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
	Name        string             `bson:"name,omitempty" json:"name,omitempty"`
	Avatar      string             `bson:"avatar,omitempty" json:"avatar,omitempty"`
	OwnerUID    string             `bson:"owner_uid,omitempty" json:"owner_uid,omitempty"`
	Members     []string           `bson:"members,omitempty" json:"members,omitempty"`
	MemberCount int                `bson:"member_count" json:"member_count"`
	CreatedBy   string             `bson:"created_by" json:"created_by"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// ChatIDManager ChatID管理器
type ChatIDManager struct {
	chatColl *mongo.Collection
}

// NewChatIDManager 创建ChatID管理器
func NewChatIDManager(db *mongo.Database) *ChatIDManager {
	return &ChatIDManager{
		chatColl: db.Collection("chats"),
	}
}

// GenerateSingleChatID 生成单聊ID（幂等 - 两个用户之间始终相同）
func GenerateSingleChatID(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("single_%s_%s", uids[0], uids[1])
}

// GenerateGroupChatID 生成群聊ID
func (m *ChatIDManager) GenerateGroupChatID() string {
	return fmt.Sprintf("group_%d", time.Now().UnixNano())
}

// CreateOrGetSingleChat 获取或创建单聊会话
func (m *ChatIDManager) CreateOrGetSingleChat(ctx context.Context, uid1, uid2 string) (*Chat, error) {
	chatID := GenerateSingleChatID(uid1, uid2)

	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"chat_id":      chatID,
			"chat_type":    types.ChatTypeSingle,
			"members":      []string{uid1, uid2},
			"member_count": 2,
			"created_at":   time.Now(),
		},
		"$set": bson.M{"updated_at": time.Now()},
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
	chatID := m.GenerateGroupChatID()

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
		CreatedBy:   ownerUID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	_, err := m.chatColl.InsertOne(ctx, chat)
	return chat, err
}

// AddMember 添加群成员
func (m *ChatIDManager) AddMember(ctx context.Context, chatID string, uid string) error {
	filter := bson.M{"chat_id": chatID}
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
	filter := bson.M{"chat_id": chatID}
	update := bson.M{
		"$pull": bson.M{"members": uid},
		"$inc":  bson.M{"member_count": -1},
		"$set":  bson.M{"updated_at": time.Now()},
	}
	_, err := m.chatColl.UpdateOne(ctx, filter, update)
	return err
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
