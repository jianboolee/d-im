package repository

import (
	"context"
	"errors"
	"time"

	"d-im/pkg/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrStaleUserVersion = errors.New("stale user version")

// UserRepo 用户数据访问层（副本存储）
type UserRepo struct {
	coll *mongo.Collection
}

// NewUserRepo 创建用户仓储
func NewUserRepo(db *mongo.Database) *UserRepo {
	return &UserRepo{
		coll: db.Collection("users"),
	}
}

// UpsertSnapshot 写入完整用户快照，只接受更新的版本。
func (r *UserRepo) UpsertSnapshot(ctx context.Context, user *model.User) error {
	now := time.Now()
	filter := bson.M{"_id": user.ID, "version": bson.M{"$lt": user.Version}}
	update := bson.M{
		"$set": bson.M{
			"nickname":   user.Nickname,
			"avatar":     user.Avatar,
			"status":     user.Status,
			"ext":        user.Ext,
			"version":    user.Version,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":        user.ID,
			"created_at": now,
		},
	}

	result, err := r.coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount > 0 {
		return nil
	}

	var current struct {
		Version int64 `bson:"version"`
	}
	if err := r.coll.FindOne(ctx, bson.M{"_id": user.ID}).Decode(&current); err == nil {
		if current.Version == user.Version {
			return nil
		}
		return ErrStaleUserVersion
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return err
	}

	user.CreatedAt = now
	user.UpdatedAt = now
	if _, err := r.coll.InsertOne(ctx, user); mongo.IsDuplicateKeyError(err) {
		return ErrStaleUserVersion
	} else {
		return err
	}
}

// FindByID 根据ID查询用户
func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
