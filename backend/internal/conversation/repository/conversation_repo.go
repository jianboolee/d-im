package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Cursor struct {
	IsTop     bool      `json:"is_top"`
	UpdatedAt time.Time `json:"updated_at"`
	ChatID    string    `json:"chat_id"`
}

type ConversationRepo struct{ coll *mongo.Collection }

func NewConversationRepo(db *mongo.Database) *ConversationRepo {
	return &ConversationRepo{coll: db.Collection(mongodb.CollectionConversations)}
}

func (r *ConversationRepo) Upsert(ctx context.Context, conv *model.Conversation) error {
	now := time.Now()
	seq := conv.LastReadSeq
	if seq < 0 {
		seq = 0
	}
	set := bson.M{"updated_at": now}
	if seq > 0 {
		set["last_read_at"] = now
	}
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": conv.UID, "chat_id": conv.ChatID}, bson.M{
		"$setOnInsert": bson.M{"conversation_id": model.NewConversationID(), "uid": conv.UID, "chat_id": conv.ChatID, "chat_type": conv.ChatType, "joined_at": now, "created_at": now},
		"$set":         set,
		"$max":         bson.M{"last_read_seq": seq},
		"$unset":       bson.M{"left_at": ""},
	}, options.Update().SetUpsert(true))
	return err
}

func (r *ConversationRepo) BatchUpsert(ctx context.Context, uidList []string, chat *model.Chat) error {
	if len(uidList) == 0 {
		return nil
	}
	now := time.Now()
	writes := make([]mongo.WriteModel, 0, len(uidList))
	for _, uid := range uidList {
		writes = append(writes, mongo.NewUpdateOneModel().
			SetFilter(bson.M{"uid": uid, "chat_id": chat.ChatID}).
			SetUpdate(bson.M{
				"$setOnInsert": bson.M{"conversation_id": model.NewConversationID(), "uid": uid, "chat_id": chat.ChatID, "chat_type": chat.ChatType, "joined_at": now, "created_at": now},
				"$set":         bson.M{"updated_at": now},
				"$unset":       bson.M{"left_at": ""},
			}).SetUpsert(true))
	}
	_, err := r.coll.BulkWrite(ctx, writes)
	return err
}

func (r *ConversationRepo) GetList(ctx context.Context, uid string, limit, offset int64) ([]*model.Conversation, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"uid": uid, "left_at": bson.M{"$exists": false}}, options.Find().
		SetSort(bson.D{{Key: "is_top", Value: -1}, {Key: "updated_at", Value: -1}}).SetLimit(limit).SetSkip(offset))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var result []*model.Conversation
	if err := cursor.All(ctx, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ConversationRepo) GetListByCursor(ctx context.Context, uid string, limit int64, value string) ([]*model.Conversation, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	filter := bson.M{"uid": uid, "left_at": bson.M{"$exists": false}}
	if value != "" {
		cursor, err := decodeCursor(value)
		if err != nil {
			return nil, "", false, err
		}
		filter["$or"] = bson.A{
			bson.M{"is_top": bson.M{"$lt": cursor.IsTop}},
			bson.M{"is_top": cursor.IsTop, "updated_at": bson.M{"$lt": cursor.UpdatedAt}},
			bson.M{"is_top": cursor.IsTop, "updated_at": cursor.UpdatedAt, "chat_id": bson.M{"$gt": cursor.ChatID}},
		}
	}
	dbCursor, err := r.coll.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "is_top", Value: -1}, {Key: "updated_at", Value: -1}, {Key: "chat_id", Value: 1}}).SetLimit(limit+1))
	if err != nil {
		return nil, "", false, err
	}
	defer dbCursor.Close(ctx)
	var result []*model.Conversation
	if err := dbCursor.All(ctx, &result); err != nil {
		return nil, "", false, err
	}
	hasMore := int64(len(result)) > limit
	if hasMore {
		result = result[:limit]
	}
	next := ""
	if hasMore && len(result) > 0 {
		last := result[len(result)-1]
		next = encodeCursor(Cursor{IsTop: last.IsTop, UpdatedAt: last.UpdatedAt, ChatID: last.ChatID})
	}
	return result, next, hasMore, nil
}

func encodeCursor(cursor Cursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func decodeCursor(value string) (Cursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, err
	}
	var cursor Cursor
	err = json.Unmarshal(data, &cursor)
	return cursor, err
}

func (r *ConversationRepo) FindByUIDAndConversationID(ctx context.Context, uid, conversationID string) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.coll.FindOne(ctx, bson.M{"uid": uid, "conversation_id": conversationID, "left_at": bson.M{"$exists": false}}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepo) FindByUIDAndChatID(ctx context.Context, uid, chatID string) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.coll.FindOne(ctx, bson.M{"uid": uid, "chat_id": chatID, "left_at": bson.M{"$exists": false}}).Decode(&conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (r *ConversationRepo) UpdateLastMessage(ctx context.Context, uid, chatID string, lastMsg *types.LastMessage) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": uid, "chat_id": chatID}, bson.M{
		"$set": bson.M{"last_msg": lastMsg, "updated_at": time.Now()},
		"$inc": bson.M{"total_msg_count": 1},
	})
	return err
}

func (r *ConversationRepo) MarkRead(ctx context.Context, uid, chatID string, seq int64) error {
	if seq < 0 {
		seq = 0
	}
	now := time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": uid, "chat_id": chatID, "$or": bson.A{
		bson.M{"last_read_seq": bson.M{"$lt": seq}}, bson.M{"last_read_seq": bson.M{"$exists": false}},
	}}, bson.M{"$set": bson.M{"last_read_seq": seq, "last_read_at": now, "updated_at": now}})
	return err
}

func (r *ConversationRepo) MarkLeft(ctx context.Context, uid, chatID string) error {
	now := time.Now()
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": uid, "chat_id": chatID, "left_at": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"left_at": now, "updated_at": now}})
	return err
}

func (r *ConversationRepo) SetTop(ctx context.Context, uid, chatID string, value bool) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": uid, "chat_id": chatID}, bson.M{"$set": bson.M{"is_top": value, "updated_at": time.Now()}})
	return err
}

func (r *ConversationRepo) SetMuted(ctx context.Context, uid, chatID string, value bool) error {
	_, err := r.coll.UpdateOne(ctx, bson.M{"uid": uid, "chat_id": chatID}, bson.M{"$set": bson.M{"is_muted": value, "updated_at": time.Now()}})
	return err
}
