package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// WithTransaction 在事务回调中执行 fn。
// fn 接受一个 mongo.SessionContext，回调内通过该 sc 调用的所有 MongoDB 操作自动参与事务。
func WithTransaction(ctx context.Context, db *mongo.Database, fn func(sc mongo.SessionContext) error) error {
	session, err := db.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(ctx)

	txnCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = session.WithTransaction(txnCtx, func(sc mongo.SessionContext) (interface{}, error) {
		return nil, fn(sc)
	})
	return err
}
