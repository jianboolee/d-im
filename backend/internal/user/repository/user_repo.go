package repository

import (
	"context"
	"time"

	"d-im/pkg/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

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

// Upsert 插入或更新用户（NATS同步时调用）
func (r *UserRepo) Upsert(ctx context.Context, user *model.User) error {
	now := time.Now()
	filter := bson.M{"_id": user.ID}
	update := bson.M{
		"$set": bson.M{
			"nickname":   user.Nickname,
			"avatar":     user.Avatar,
			"status":     user.Status,
			"ext":        user.Ext,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"_id":        user.ID,
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)
	_, err := r.coll.UpdateOne(ctx, filter, update, opts)
	return err
}

// BatchUpsert 批量同步用户
func (r *UserRepo) BatchUpsert(ctx context.Context, users []*model.User) error {
	if len(users) == 0 {
		return nil
	}

	now := time.Now()
	models := make([]mongo.WriteModel, len(users))

	for i, user := range users {
		filter := bson.M{"_id": user.ID}
		update := bson.M{
			"$set": bson.M{
				"nickname":   user.Nickname,
				"avatar":     user.Avatar,
				"status":     user.Status,
				"ext":        user.Ext,
				"updated_at": now,
			},
			"$setOnInsert": bson.M{
				"_id":        user.ID,
				"created_at": now,
			},
		}
		models[i] = mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)
	}

	_, err := r.coll.BulkWrite(ctx, models)
	return err
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

// Delete 删除用户（NATS同步delete时调用）
func (r *UserRepo) Delete(ctx context.Context, id string) error {
	_, err := r.coll.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
