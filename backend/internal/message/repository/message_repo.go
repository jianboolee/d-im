package repository

import (
	"context"
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
	return messages, nil
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
