package model

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"d-im/pkg/mongodb"
	"d-im/pkg/snowflake"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConversationCursor struct {
	IsTop     bool      `json:"is_top"`
	UpdatedAt time.Time `json:"updated_at"`
	ChatID    string    `json:"chat_id"`
}

// ConversationManager 会话视图管理器
type ConversationManager struct {
	convColl *mongo.Collection
	idGen    *snowflake.Generator
}

// NewConversationManager 创建会话管理器
func NewConversationManager(db *mongo.Database, idGen *snowflake.Generator) *ConversationManager {
	return &ConversationManager{
		convColl: db.Collection(mongodb.CollectionConversations),
		idGen:    idGen,
	}
}

func (m *ConversationManager) GenerateConversationID() string {
	return "conv_" + m.idGen.GenerateString()
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
			"conversation_id": m.GenerateConversationID(),
			"uid":             conv.UID,
			"chat_id":         conv.ChatID,
			"chat_type":       conv.ChatType,
			"joined_at":       now,
			"created_at":      now,
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
				"conversation_id": m.GenerateConversationID(),
				"uid":             uid,
				"chat_id":         chat.ChatID,
				"chat_type":       chat.ChatType,
				"joined_at":       now,
				"created_at":      now,
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

// GetListByCursor 获取用户会话列表（置顶优先，按更新时间倒序，chat_id稳定补序）
func (m *ConversationManager) GetListByCursor(ctx context.Context, uid string, limit int64, cursorValue string) ([]*Conversation, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{
		"uid":     uid,
		"left_at": bson.M{"$exists": false},
	}

	if cursorValue != "" {
		cursor, err := DecodeConversationCursor(cursorValue)
		if err != nil {
			return nil, "", false, err
		}
		filter["$or"] = bson.A{
			bson.M{"is_top": bson.M{"$lt": cursor.IsTop}},
			bson.M{
				"is_top":     cursor.IsTop,
				"updated_at": bson.M{"$lt": cursor.UpdatedAt},
			},
			bson.M{
				"is_top":     cursor.IsTop,
				"updated_at": cursor.UpdatedAt,
				"chat_id":    bson.M{"$gt": cursor.ChatID},
			},
		}
	}

	opts := options.Find().
		SetSort(bson.D{
			{Key: "is_top", Value: -1},
			{Key: "updated_at", Value: -1},
			{Key: "chat_id", Value: 1},
		}).
		SetLimit(limit + 1)

	dbCursor, err := m.convColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", false, err
	}
	defer dbCursor.Close(ctx)

	var conversations []*Conversation
	if err := dbCursor.All(ctx, &conversations); err != nil {
		return nil, "", false, err
	}

	hasMore := int64(len(conversations)) > limit
	if hasMore {
		conversations = conversations[:limit]
	}

	nextCursor := ""
	if hasMore && len(conversations) > 0 {
		last := conversations[len(conversations)-1]
		nextCursor = EncodeConversationCursor(ConversationCursor{
			IsTop:     last.IsTop,
			UpdatedAt: last.UpdatedAt,
			ChatID:    last.ChatID,
		})
	}

	return conversations, nextCursor, hasMore, nil
}

func EncodeConversationCursor(cursor ConversationCursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func DecodeConversationCursor(value string) (ConversationCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ConversationCursor{}, err
	}
	var cursor ConversationCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return ConversationCursor{}, err
	}
	return cursor, nil
}

// FindByUIDAndConversationID 查询用户与指定会话视图。
func (m *ConversationManager) FindByUIDAndConversationID(ctx context.Context, uid, conversationID string) (*Conversation, error) {
	var conv Conversation
	err := m.convColl.FindOne(ctx, bson.M{
		"uid":             uid,
		"conversation_id": conversationID,
		"left_at":         bson.M{"$exists": false},
	}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// FindByUIDAndChatID 查询用户与指定会话的视图
func (m *ConversationManager) FindByUIDAndChatID(ctx context.Context, uid, chatID string) (*Conversation, error) {
	var conv Conversation
	err := m.convColl.FindOne(ctx, bson.M{
		"uid":     uid,
		"chat_id": chatID,
		"left_at": bson.M{"$exists": false},
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

// MarkRead 推进当前用户在会话中的最后已读消息序列。
func (m *ConversationManager) MarkRead(ctx context.Context, uid, chatID string, lastReadSeq int64) error {
	if lastReadSeq < 0 {
		lastReadSeq = 0
	}
	now := time.Now()
	filter := bson.M{
		"uid":     uid,
		"chat_id": chatID,
		"$or": bson.A{
			bson.M{"last_read_seq": bson.M{"$lt": lastReadSeq}},
			bson.M{"last_read_seq": bson.M{"$exists": false}},
		},
	}
	update := bson.M{
		"$set": bson.M{
			"last_read_seq": lastReadSeq,
			"last_read_at":  now,
			"updated_at":    now,
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
