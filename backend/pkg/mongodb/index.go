package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// CreateIndexes 创建所有集合的索引
func CreateIndexes(ctx context.Context, db *mongo.Database) error {
	// Messages集合索引
	messageIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "client_time", Value: -1},
			},
			Options: options.Index().SetName("idx_chat_time"),
		},
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "seq", Value: -1},
			},
			Options: options.Index().SetName("idx_chat_seq"),
		},
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "seq", Value: 1},
			},
			Options: options.Index().
				SetName("idx_chat_seq_unique").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"seq": bson.M{"$exists": true}}),
		},
		{
			Keys: bson.D{
				{Key: "msg_id", Value: 1},
			},
			Options: options.Index().SetName("idx_msg_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "sender_id", Value: 1},
				{Key: "client_message_id", Value: 1},
			},
			Options: options.Index().
				SetName("idx_client_message_id").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"client_message_id": bson.M{"$exists": true}}),
		},
		{
			Keys: bson.D{
				{Key: "sender_id", Value: 1},
				{Key: "created_at", Value: -1},
			},
			Options: options.Index().SetName("idx_sender_id_time"),
		},
		{
			Keys: bson.D{
				{Key: "msg_type", Value: 1},
				{Key: "chat_id", Value: 1},
			},
			Options: options.Index().SetName("idx_type_chat"),
		},
		{
			Keys: bson.D{
				{Key: "created_at", Value: 1},
			},
			Options: options.Index().SetName("idx_created_at").SetExpireAfterSeconds(90 * 24 * 3600), // 90天TTL
		},
	}

	if _, err := db.Collection(CollectionMessages).Indexes().CreateMany(ctx, messageIndexes); err != nil {
		return err
	}

	// Conversations集合索引
	conversationIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "conversation_id", Value: 1},
			},
			Options: options.Index().SetName("idx_conversation_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "chat_id", Value: 1},
			},
			Options: options.Index().SetName("idx_uid_chat_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "updated_at", Value: -1},
			},
			Options: options.Index().SetName("idx_uid_updated"),
		},
	}

	if _, err := db.Collection(CollectionConversations).Indexes().CreateMany(ctx, conversationIndexes); err != nil {
		return err
	}

	// UserMailbox集合索引
	mailboxIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "seq_id", Value: -1},
			},
			Options: options.Index().SetName("idx_uid_seq"),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "chat_id", Value: 1},
				{Key: "seq_id", Value: -1},
			},
			Options: options.Index().SetName("idx_uid_chat_seq"),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "chat_id", Value: 1},
				{Key: "message_seq", Value: -1},
			},
			Options: options.Index().SetName("idx_uid_chat_message_seq"),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("idx_uid_status"),
		},
	}

	if _, err := db.Collection(CollectionUserMailbox).Indexes().CreateMany(ctx, mailboxIndexes); err != nil {
		return err
	}

	// Chats集合索引
	chatsIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
			},
			Options: options.Index().SetName("idx_chat_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "chat_type", Value: 1},
				{Key: "single_key", Value: 1},
			},
			Options: options.Index().
				SetName("idx_single_key").
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"single_key": bson.M{"$exists": true}}),
		},
		{
			Keys:    bson.D{{Key: "members", Value: 1}},
			Options: options.Index().SetName("idx_members"),
		},
	}

	if _, err := db.Collection(CollectionChats).Indexes().CreateMany(ctx, chatsIndexes); err != nil {
		return err
	}

	groupIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "chat_id", Value: 1}},
			Options: options.Index().SetName("idx_group_chat_id").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "settings.is_public", Value: 1},
				{Key: "updated_at", Value: -1},
			},
			Options: options.Index().SetName("idx_group_public_status_updated"),
		},
	}
	if _, err := db.Collection(CollectionGroups).Indexes().CreateMany(ctx, groupIndexes); err != nil {
		return err
	}

	groupMemberIndexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "uid", Value: 1},
			},
			Options: options.Index().SetName("idx_group_member_unique").SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "uid", Value: 1},
				{Key: "joined_at", Value: -1},
			},
			Options: options.Index().SetName("idx_group_member_uid_joined"),
		},
		{
			Keys: bson.D{
				{Key: "chat_id", Value: 1},
				{Key: "joined_at", Value: 1},
			},
			Options: options.Index().SetName("idx_group_member_chat_joined"),
		},
	}
	if _, err := db.Collection(CollectionGroupMembers).Indexes().CreateMany(ctx, groupMemberIndexes); err != nil {
		return err
	}

	return nil
}

// EnsureIndexes 确保索引存在（如果不存在则创建，不重复创建）
func EnsureIndexes(ctx context.Context, db *mongo.Database) error {
	// 获取已有的索引名，判断是否需要创建
	// 简化处理：直接调用 CreateIndexes，MongoDB 会自动跳过已存在的同名索引
	return CreateIndexes(ctx, db)
}
