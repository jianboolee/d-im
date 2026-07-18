package repository

import (
	"context"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"

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

// InsertOrGetSingle 原子插入单聊候选实体，或返回同一 single_key 的已有 Chat。
func (r *ChatRepo) InsertOrGetSingle(ctx context.Context, candidate *model.Chat) (*model.Chat, error) {
	filter := bson.M{
		"chat_type":  candidate.ChatType,
		"single_key": candidate.SingleKey,
	}
	update := bson.M{
		"$setOnInsert": bson.M{
			"chat_id":      candidate.ChatID,
			"chat_type":    candidate.ChatType,
			"single_key":   candidate.SingleKey,
			"members":      candidate.Members,
			"member_count": candidate.MemberCount,
			"last_seq":     candidate.LastSeq,
			"created_by":   candidate.CreatedBy,
			"created_at":   candidate.CreatedAt,
			"updated_at":   candidate.UpdatedAt,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var chat model.Chat
	err := r.coll.FindOneAndUpdate(ctx, filter, update, opts).Decode(&chat)
	if mongo.IsDuplicateKeyError(err) {
		err = r.coll.FindOne(ctx, filter).Decode(&chat)
	}
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// Insert 插入已由领域服务构造完成的 Chat。
func (r *ChatRepo) Insert(ctx context.Context, chat *model.Chat) error {
	_, err := r.coll.InsertOne(ctx, chat)
	return err
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
