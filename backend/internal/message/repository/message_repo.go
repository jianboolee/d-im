package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"regexp"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MessageRepo 消息数据访问层
type MessageRepo struct {
	messagesColl *mongo.Collection
	mailboxColl  *mongo.Collection
}

type MessageCursor struct {
	Seq int64 `json:"seq"`
}

// NewMessageRepo 创建消息仓储
func NewMessageRepo(db *mongo.Database) *MessageRepo {
	return &MessageRepo{
		messagesColl: db.Collection(mongodb.CollectionMessages),
		mailboxColl:  db.Collection(mongodb.CollectionUserMailbox),
	}
}

// Insert 插入消息
func (r *MessageRepo) Insert(ctx context.Context, msg *model.Message) error {
	msg.CreatedAt = time.Now()
	msg.UpdatedAt = time.Now()
	msg.ServerTime = time.Now()

	_, err := r.messagesColl.InsertOne(ctx, msg)
	return err
}

// FindByMsgID 根据消息ID查询
func (r *MessageRepo) FindByMsgID(ctx context.Context, msgID string) (*model.Message, error) {
	var msg model.Message
	err := r.messagesColl.FindOne(ctx, bson.M{"msg_id": msgID}).Decode(&msg)
	if err != nil {
		return nil, err
	}
	msg.NormalizeContent()
	return &msg, nil
}

// FindByClientMsgID 根据发送者客户端消息ID查询，用于发送幂等。
func (r *MessageRepo) FindByClientMsgID(ctx context.Context, chatID, senderID, clientMsgID string) (*model.Message, error) {
	var msg model.Message
	err := r.messagesColl.FindOne(ctx, bson.M{
		"chat_id":           chatID,
		"sender_id":         senderID,
		"client_message_id": clientMsgID,
	}).Decode(&msg)
	if err != nil {
		return nil, err
	}
	msg.NormalizeContent()
	return &msg, nil
}

// FindByChatID 根据会话ID分页查询消息
func (r *MessageRepo) FindByChatID(ctx context.Context, chatID string, limit, offset int64) ([]*model.Message, error) {
	filter := bson.M{"chat_id": chatID}
	opts := options.Find().
		SetSort(bson.D{{Key: "client_time", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.messagesColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []*model.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	normalizeMessages(messages)
	return messages, nil
}

// FindPageByChatSeq 查询指定会话的历史消息页。内部按新到旧取数，调用方可按展示需要反转。
func (r *MessageRepo) FindPageByChatSeq(ctx context.Context, chatID string, limit int64, cursorValue string) ([]*model.Message, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{
		"chat_id":    chatID,
		"deleted_at": bson.M{"$exists": false},
		"seq":        bson.M{"$exists": true},
	}

	if cursorValue != "" {
		cursor, err := DecodeMessageCursor(cursorValue)
		if err != nil {
			return nil, "", false, err
		}
		filter["seq"] = bson.M{"$exists": true, "$lt": cursor.Seq}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "seq", Value: -1}}).
		SetLimit(limit + 1)

	dbCursor, err := r.messagesColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", false, err
	}
	defer dbCursor.Close(ctx)

	var messages []*model.Message
	if err := dbCursor.All(ctx, &messages); err != nil {
		return nil, "", false, err
	}
	normalizeMessages(messages)

	hasMore := int64(len(messages)) > limit
	if hasMore {
		messages = messages[:limit]
	}

	nextCursor := ""
	if hasMore && len(messages) > 0 {
		last := messages[len(messages)-1]
		nextCursor = EncodeMessageCursor(MessageCursor{Seq: last.Seq})
	}

	return messages, nextCursor, hasMore, nil
}

// SearchPageByChatSeq 在指定会话内按内容预览搜索消息。内部按新到旧取数，调用方可按展示需要反转。
func (r *MessageRepo) SearchPageByChatSeq(ctx context.Context, chatID, keyword string, limit int64, cursorValue string) ([]*model.Message, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filter := bson.M{
		"chat_id":         chatID,
		"deleted_at":      bson.M{"$exists": false},
		"seq":             bson.M{"$exists": true},
		"content_preview": bson.M{"$regex": regexp.QuoteMeta(keyword), "$options": "i"},
	}

	if cursorValue != "" {
		cursor, err := DecodeMessageCursor(cursorValue)
		if err != nil {
			return nil, "", false, err
		}
		filter["seq"] = bson.M{"$exists": true, "$lt": cursor.Seq}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "seq", Value: -1}}).
		SetLimit(limit + 1)

	dbCursor, err := r.messagesColl.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", false, err
	}
	defer dbCursor.Close(ctx)

	var messages []*model.Message
	if err := dbCursor.All(ctx, &messages); err != nil {
		return nil, "", false, err
	}
	normalizeMessages(messages)

	hasMore := int64(len(messages)) > limit
	if hasMore {
		messages = messages[:limit]
	}

	nextCursor := ""
	if hasMore && len(messages) > 0 {
		last := messages[len(messages)-1]
		nextCursor = EncodeMessageCursor(MessageCursor{Seq: last.Seq})
	}

	return messages, nextCursor, hasMore, nil
}

func normalizeMessages(messages []*model.Message) {
	for _, message := range messages {
		message.NormalizeContent()
	}
}

func EncodeMessageCursor(cursor MessageCursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(data)
}

func DecodeMessageCursor(value string) (MessageCursor, error) {
	data, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return MessageCursor{}, err
	}
	var cursor MessageCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return MessageCursor{}, err
	}
	return cursor, nil
}

// Recall 撤回消息
func (r *MessageRepo) Recall(ctx context.Context, msgID string) error {
	now := time.Now()
	filter := bson.M{"msg_id": msgID}
	update := bson.M{
		"$set": bson.M{
			"is_recalled": true,
			"recall_time": now,
			"status":      types.MessageStatusRecalled,
			"updated_at":  now,
		},
	}
	_, err := r.messagesColl.UpdateOne(ctx, filter, update)
	return err
}

// UpdateStatus 更新消息状态
func (r *MessageRepo) UpdateStatus(ctx context.Context, msgID string, status types.MessageStatus) error {
	filter := bson.M{"msg_id": msgID}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	_, err := r.messagesColl.UpdateOne(ctx, filter, update)
	return err
}

// FindMailbox 查询用户信箱中的指定消息投递记录。
func (r *MessageRepo) FindMailbox(ctx context.Context, uid, chatID, msgID string) (*model.UserMailbox, error) {
	var mailbox model.UserMailbox
	err := r.mailboxColl.FindOne(ctx, bson.M{
		"uid":     uid,
		"chat_id": chatID,
		"msg_id":  msgID,
	}).Decode(&mailbox)
	if err != nil {
		return nil, err
	}
	return &mailbox, nil
}

// InsertToMailbox 写入用户信箱（消息分发）
func (r *MessageRepo) InsertToMailbox(ctx context.Context, mailbox *model.UserMailbox) error {
	mailbox.CreatedAt = time.Now()
	_, err := r.mailboxColl.InsertOne(ctx, mailbox)
	return err
}

// BatchInsertToMailbox 批量写入用户信箱
func (r *MessageRepo) BatchInsertToMailbox(ctx context.Context, mailboxes []*model.UserMailbox) error {
	if len(mailboxes) == 0 {
		return nil
	}

	docs := make([]interface{}, len(mailboxes))
	now := time.Now()
	for i, mb := range mailboxes {
		mb.CreatedAt = now
		docs[i] = mb
	}

	_, err := r.mailboxColl.InsertMany(ctx, docs)
	return err
}
