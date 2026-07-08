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

type MemberRepo struct {
	coll *mongo.Collection
}

func NewMemberRepo(db *mongo.Database) *MemberRepo {
	return &MemberRepo{coll: db.Collection(mongodb.CollectionGroupMembers)}
}

func (r *MemberRepo) CreateMany(ctx context.Context, members []*model.GroupMember) error {
	if len(members) == 0 {
		return nil
	}
	docs := make([]interface{}, 0, len(members))
	for _, member := range members {
		if member != nil {
			docs = append(docs, member)
		}
	}
	if len(docs) == 0 {
		return nil
	}
	_, err := r.coll.InsertMany(ctx, docs)
	return err
}

func (r *MemberRepo) Add(ctx context.Context, member *model.GroupMember) (bool, error) {
	if member == nil {
		return false, mongo.ErrNilDocument
	}
	now := time.Now()
	if member.JoinedAt.IsZero() {
		member.JoinedAt = now
	}
	member.UpdatedAt = now
	update := bson.M{
		"$setOnInsert": member,
	}
	result, err := r.coll.UpdateOne(ctx, bson.M{
		"chat_id": member.ChatID,
		"uid":     member.UID,
	}, update, options.Update().SetUpsert(true))
	if err != nil {
		return false, err
	}
	return result.UpsertedCount > 0, nil
}

func (r *MemberRepo) Remove(ctx context.Context, chatID, uid string) (bool, error) {
	result, err := r.coll.DeleteOne(ctx, bson.M{
		"chat_id": chatID,
		"uid":     uid,
	})
	if err != nil {
		return false, err
	}
	return result.DeletedCount > 0, nil
}

func (r *MemberRepo) Find(ctx context.Context, chatID, uid string) (*model.GroupMember, error) {
	var member model.GroupMember
	err := r.coll.FindOne(ctx, bson.M{
		"chat_id": chatID,
		"uid":     uid,
	}).Decode(&member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *MemberRepo) FirstJoinedExcept(ctx context.Context, chatID, excludeUID string) (*model.GroupMember, error) {
	var member model.GroupMember
	err := r.coll.FindOne(ctx, bson.M{
		"chat_id": chatID,
		"uid":     bson.M{"$ne": excludeUID},
	}, options.FindOne().
		SetSort(bson.D{{Key: "joined_at", Value: 1}, {Key: "uid", Value: 1}})).
		Decode(&member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (r *MemberRepo) List(ctx context.Context, chatID string, limit, offset int64) ([]*model.GroupMember, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	cursor, err := r.coll.Find(ctx, bson.M{"chat_id": chatID}, options.Find().
		SetSort(bson.D{{Key: "joined_at", Value: 1}, {Key: "uid", Value: 1}}).
		SetLimit(limit).
		SetSkip(offset))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*model.GroupMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *MemberRepo) ListUIDs(ctx context.Context, chatID string) ([]string, error) {
	cursor, err := r.coll.Find(ctx, bson.M{"chat_id": chatID}, options.Find().
		SetProjection(bson.M{"uid": 1}).
		SetSort(bson.D{{Key: "joined_at", Value: 1}, {Key: "uid", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*model.GroupMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	uids := make([]string, 0, len(members))
	for _, member := range members {
		uids = append(uids, member.UID)
	}
	return uids, nil
}

func (r *MemberRepo) ListChatIDsByUID(ctx context.Context, uid string, limit, offset int64) ([]string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	cursor, err := r.coll.Find(ctx, bson.M{"uid": uid}, options.Find().
		SetProjection(bson.M{"chat_id": 1}).
		SetSort(bson.D{{Key: "joined_at", Value: -1}, {Key: "chat_id", Value: 1}}).
		SetLimit(limit).
		SetSkip(offset))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*model.GroupMember
	if err := cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	chatIDs := make([]string, 0, len(members))
	for _, member := range members {
		chatIDs = append(chatIDs, member.ChatID)
	}
	return chatIDs, nil
}

func (r *MemberRepo) SetRole(ctx context.Context, chatID, uid string, role model.MemberRole) (*model.GroupMember, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var member model.GroupMember
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"uid":     uid,
	}, bson.M{"$set": bson.M{
		"role":       role,
		"updated_at": time.Now(),
	}}, opts).Decode(&member)
	if err != nil {
		return nil, err
	}
	return &member, nil
}
