package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChatRepo 封装 chats 集合读写。
type ChatRepo struct {
	coll *mongo.Collection
}

// NewChatRepo 创建 ChatRepo。
func NewChatRepo(db *mongo.Database) *ChatRepo {
	return &ChatRepo{coll: db.Collection(mongodb.CollectionChats)}
}

// Collection 返回底层 MongoDB 集合（用于事务等场景）。
func (r *ChatRepo) Collection() *mongo.Collection {
	return r.coll
}

// CreateOrGetSingleChat 获取或创建单聊会话。
func (r *ChatRepo) CreateOrGetSingleChat(ctx context.Context, uid1, uid2 string) (*model.Chat, error) {
	singleKey := generateSingleChatKey(uid1, uid2)
	chatID := model.NewChatID()
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

	var chat model.Chat
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// CreateGroupChat 创建群聊对应的消息会话实体。
func (r *ChatRepo) CreateGroupChat(ctx context.Context, creatorUID string) (*model.Chat, error) {
	chatID := model.NewChatID()
	now := time.Now()
	chat := &model.Chat{
		ChatID:    chatID,
		ChatType:  types.ChatTypeGroup,
		LastSeq:   0,
		CreatedBy: creatorUID,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := r.coll.InsertOne(ctx, chat)
	return chat, err
}

// FindByChatID 根据 chatID 查询 Chat。
func (r *ChatRepo) FindByChatID(ctx context.Context, chatID string) (*model.Chat, error) {
	var chat model.Chat
	err := r.coll.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// GetMembers 查询 Chat 中的成员列表（主要用于单聊场景）。
func (r *ChatRepo) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	var chat model.Chat
	err := r.coll.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat)
	if err != nil {
		return nil, err
	}
	return chat.Members, nil
}

// NextMessageSeq 为指定 chat 原子分配下一条消息序号。
func (r *ChatRepo) NextMessageSeq(ctx context.Context, chatID string) (int64, error) {
	now := time.Now()
	update := bson.M{
		"$inc": bson.M{"last_seq": 1},
		"$set": bson.M{"updated_at": now},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var chat model.Chat
	err := r.coll.FindOneAndUpdate(ctx, bson.M{"chat_id": chatID}, update, opts).Decode(&chat)
	if err != nil {
		return 0, err
	}
	return chat.LastSeq, nil
}

// generateSingleChatKey 生成单聊幂等键。
func generateSingleChatKey(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("%s:%s", uids[0], uids[1])
}
