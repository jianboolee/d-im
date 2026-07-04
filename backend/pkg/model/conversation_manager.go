package model

import (
	"context"
	"time"

	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ConversationManager 会话视图管理器
type ConversationManager struct {
	convColl *mongo.Collection
}

// NewConversationManager 创建会话管理器
func NewConversationManager(db *mongo.Database) *ConversationManager {
	return &ConversationManager{
		convColl: db.Collection(mongodb.CollectionConversations),
	}
}

// CreateOrUpdate 为用户创建或更新会话视图（upsert）
func (m *ConversationManager) CreateOrUpdate(ctx context.Context, conv *Conversation) error {
	now := time.Now()

	filter := bson.M{
		"uid":     conv.UID,
		"chat_id": conv.ChatID,
	}

	update := bson.M{
		"$setOnInsert": bson.M{
			"uid":        conv.UID,
			"chat_id":    conv.ChatID,
			"chat_type":  conv.ChatType,
			"joined_at":  now,
			"created_at": now,
		},
		"$set": bson.M{
			"updated_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := m.convColl.UpdateOne(ctx, filter, update, opts)
	return err
}

// BatchCreate 批量创建会话视图（新群聊时使用）
func (m *ConversationManager) BatchCreate(ctx context.Context, uidList []string, chat *Chat) error {
	if len(uidList) == 0 {
		return nil
	}

	now := time.Now()
	models := make([]mongo.WriteModel, len(uidList))

	for i, uid := range uidList {
		filter := bson.M{
			"uid":     uid,
			"chat_id": chat.ChatID,
		}

		update := bson.M{
			"$setOnInsert": bson.M{
				"uid":        uid,
				"chat_id":    chat.ChatID,
				"chat_type":  chat.ChatType,
				"joined_at":  now,
				"created_at": now,
			},
			"$set": bson.M{
				"updated_at": now,
			},
		}

		models[i] = mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)
	}

	_, err := m.convColl.BulkWrite(ctx, models)
	return err
}

// GetList 获取用户的会话列表（置顶优先，按更新时间倒序）
func (m *ConversationManager) GetList(ctx context.Context, uid string, limit, offset int64) ([]*Conversation, error) {
	filter := bson.M{
		"uid":     uid,
		"left_at": bson.M{"$exists": false},
	}

	opts := options.Find().
		SetSort(bson.D{
			{Key: "is_top", Value: -1},
			{Key: "updated_at", Value: -1},
		}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := m.convColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []*Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}
	return conversations, nil
}

// FindByUIDAndChatID 查询用户与指定会话的视图
func (m *ConversationManager) FindByUIDAndChatID(ctx context.Context, uid, chatID string) (*Conversation, error) {
	var conv Conversation
	err := m.convColl.FindOne(ctx, bson.M{
		"uid":     uid,
		"chat_id": chatID,
	}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// UpdateLastMsg 更新会话的最后一条消息摘要
func (m *ConversationManager) UpdateLastMsg(ctx context.Context, uid, chatID string, lastMsg *types.LastMessage) error {
	filter := bson.M{"uid": uid, "chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"last_msg":   lastMsg,
			"updated_at": time.Now(),
		},
		"$inc": bson.M{
			"total_msg_count": 1,
		},
	}

	_, err := m.convColl.UpdateOne(ctx, filter, update)
	return err
}

// UpdateUnreadCount 更新未读计数
func (m *ConversationManager) UpdateUnreadCount(ctx context.Context, uid, chatID string, delta int) error {
	filter := bson.M{"uid": uid, "chat_id": chatID}
	update := bson.M{
		"$inc": bson.M{"unread_count": delta},
		"$set": bson.M{"updated_at": time.Now()},
	}
	_, err := m.convColl.UpdateOne(ctx, filter, update)
	return err
}

// ResetUnread 重置未读计数为0
func (m *ConversationManager) ResetUnread(ctx context.Context, uid, chatID string) error {
	filter := bson.M{"uid": uid, "chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"unread_count": 0,
			"updated_at":   time.Now(),
		},
	}
	_, err := m.convColl.UpdateOne(ctx, filter, update)
	return err
}

// SetTop 设置/取消置顶
func (m *ConversationManager) SetTop(ctx context.Context, uid, chatID string, isTop bool) error {
	filter := bson.M{"uid": uid, "chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"is_top":     isTop,
			"updated_at": time.Now(),
		},
	}
	_, err := m.convColl.UpdateOne(ctx, filter, update)
	return err
}

// SetMuted 设置/取消免打扰
func (m *ConversationManager) SetMuted(ctx context.Context, uid, chatID string, isMuted bool) error {
	filter := bson.M{"uid": uid, "chat_id": chatID}
	update := bson.M{
		"$set": bson.M{
			"is_muted":   isMuted,
			"updated_at": time.Now(),
		},
	}
	_, err := m.convColl.UpdateOne(ctx, filter, update)
	return err
}
