package repair

import (
	"context"
	"errors"

	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Projector interface {
	EnsureUsers(context.Context, []string, *model.Chat) error
	UserLeft(context.Context, string, string) error
	MessageSent(context.Context, []string, string, *model.Message, *types.LastMessage) error
}

type Repairer struct {
	chats, groups, members, messages, conversations *mongo.Collection
	projector                                       Projector
}

func NewRepairer(db *mongo.Database, projector Projector) *Repairer {
	return &Repairer{
		chats: db.Collection(mongodb.CollectionChats), groups: db.Collection(mongodb.CollectionGroups),
		members: db.Collection(mongodb.CollectionGroupMembers), messages: db.Collection(mongodb.CollectionMessages),
		conversations: db.Collection(mongodb.CollectionConversations),
		projector:     projector,
	}
}

func (r *Repairer) RepairAll(ctx context.Context) error {
	cursor, err := r.chats.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var chat model.Chat
		if err := cursor.Decode(&chat); err != nil {
			return err
		}
		if err := r.repair(ctx, &chat); err != nil {
			return err
		}
	}
	return cursor.Err()
}

func (r *Repairer) RepairChat(ctx context.Context, chatID string) error {
	var chat model.Chat
	if err := r.chats.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&chat); err != nil {
		return err
	}
	return r.repair(ctx, &chat)
}

func (r *Repairer) repair(ctx context.Context, chat *model.Chat) error {
	userIDs, active, err := r.activeUsers(ctx, chat)
	if err != nil || !active {
		return err
	}
	if err := r.projector.EnsureUsers(ctx, userIDs, chat); err != nil {
		return err
	}
	if err := r.markStaleUsersLeft(ctx, chat.ChatID, userIDs); err != nil {
		return err
	}

	var msg model.Message
	err = r.messages.FindOne(ctx, bson.M{"chat_id": chat.ChatID, "deleted_at": bson.M{"$exists": false}}, options.FindOne().SetSort(bson.D{{Key: "seq", Value: -1}})).Decode(&msg)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil
	}
	if err != nil {
		return err
	}
	last := &types.LastMessage{
		MsgID: msg.MsgID, Seq: msg.Seq, SenderID: msg.SenderID, SenderName: msg.SenderName,
		MsgType: msg.MsgType, ContentPreview: msg.ContentPreview, ClientTime: msg.ClientTime,
	}
	return r.projector.MessageSent(ctx, userIDs, msg.SenderID, &msg, last)
}

func (r *Repairer) markStaleUsersLeft(ctx context.Context, chatID string, activeUserIDs []string) error {
	active := make(map[string]struct{}, len(activeUserIDs))
	for _, uid := range activeUserIDs {
		active[uid] = struct{}{}
	}
	cursor, err := r.conversations.Find(ctx, bson.M{"chat_id": chatID, "left_at": bson.M{"$exists": false}})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)
	var views []model.Conversation
	if err := cursor.All(ctx, &views); err != nil {
		return err
	}
	for _, view := range views {
		if _, ok := active[view.UID]; !ok {
			if err := r.projector.UserLeft(ctx, view.UID, chatID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Repairer) activeUsers(ctx context.Context, chat *model.Chat) ([]string, bool, error) {
	if chat.ChatType == types.ChatTypeSingle {
		return chat.Members, true, nil
	}
	var group model.Group
	if err := r.groups.FindOne(ctx, bson.M{"chat_id": chat.ChatID, "status": model.GroupStatusActive}).Decode(&group); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, nil
		}
		return nil, false, err
	}
	cursor, err := r.members.Find(ctx, bson.M{"chat_id": chat.ChatID})
	if err != nil {
		return nil, false, err
	}
	defer cursor.Close(ctx)
	var docs []model.GroupMember
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, false, err
	}
	userIDs := make([]string, 0, len(docs))
	for _, member := range docs {
		if member.UID != "" {
			userIDs = append(userIDs, member.UID)
		}
	}
	return userIDs, true, nil
}
