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

type GroupRepo struct {
	coll *mongo.Collection
}

func NewGroupRepo(db *mongo.Database) *GroupRepo {
	return &GroupRepo{coll: db.Collection(mongodb.CollectionGroups)}
}

func (r *GroupRepo) Create(ctx context.Context, group *model.Group) error {
	if group == nil {
		return mongo.ErrNilDocument
	}
	_, err := r.coll.InsertOne(ctx, group)
	return err
}

func (r *GroupRepo) FindByChatID(ctx context.Context, chatID string) (*model.Group, error) {
	var group model.Group
	err := r.coll.FindOne(ctx, bson.M{"chat_id": chatID}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) FindActiveByChatID(ctx context.Context, chatID string) (*model.Group, error) {
	var group model.Group
	err := r.coll.FindOne(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
	}).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) ListByChatIDs(ctx context.Context, chatIDs []string) (map[string]*model.Group, error) {
	if len(chatIDs) == 0 {
		return map[string]*model.Group{}, nil
	}
	cursor, err := r.coll.Find(ctx, bson.M{
		"chat_id": bson.M{"$in": chatIDs},
		"status":  model.GroupStatusActive,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var groups []*model.Group
	if err := cursor.All(ctx, &groups); err != nil {
		return nil, err
	}
	result := make(map[string]*model.Group, len(groups))
	for _, group := range groups {
		result[group.ChatID] = group
	}
	return result, nil
}

func (r *GroupRepo) UpdateFields(ctx context.Context, chatID string, fields bson.M) (*model.Group, error) {
	set := bson.M{"updated_at": time.Now()}
	for key, value := range fields {
		set[key] = value
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var group model.Group
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
	}, bson.M{"$set": set}, opts).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) IncMemberCount(ctx context.Context, chatID string, delta int) (*model.Group, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var group model.Group
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
	}, bson.M{
		"$inc": bson.M{"member_count": delta},
		"$set": bson.M{"updated_at": time.Now()},
	}, opts).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) Dismiss(ctx context.Context, chatID string) (*model.Group, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var group model.Group
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
	}, bson.M{"$set": bson.M{
		"status":     model.GroupStatusDismissed,
		"updated_at": time.Now(),
	}}, opts).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) UpdateAvatarIfEmpty(ctx context.Context, chatID, avatar string) (*model.Group, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var group model.Group
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
		"$or": bson.A{
			bson.M{"avatar": ""},
			bson.M{"avatar": bson.M{"$exists": false}},
		},
	}, bson.M{"$set": bson.M{
		"avatar":     avatar,
		"updated_at": time.Now(),
	}}, opts).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *GroupRepo) UpdateAvatar(ctx context.Context, chatID, avatar string) (*model.Group, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var group model.Group
	err := r.coll.FindOneAndUpdate(ctx, bson.M{
		"chat_id": chatID,
		"status":  model.GroupStatusActive,
	}, bson.M{"$set": bson.M{
		"avatar":     avatar,
		"updated_at": time.Now(),
	}}, opts).Decode(&group)
	if err != nil {
		return nil, err
	}
	return &group, nil
}
