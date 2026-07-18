package mongodb

import (
	"context"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// WithTransaction 在事务回调中执行 fn。
// 如果 MongoDB 不支持事务（standalone 模式），自动降级为顺序执行。
// fn 接收 context.Context（在事务模式下是 mongo.SessionContext，SessionContext 实现了 context.Context）。
func WithTransaction(ctx context.Context, db *mongo.Database, fn func(ctx context.Context) error) error {
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
	if err != nil && isTransactionNotSupported(err) {
		log.Printf("[mongodb] transaction not supported, falling back to sequential execution (standalone mode)")
		return fn(ctx)
	}
	return err
}

// WithRequiredTransaction executes fn atomically and never falls back to
// sequential writes. Use it for business-write + outbox guarantees.
func WithRequiredTransaction(ctx context.Context, db *mongo.Database, fn func(ctx context.Context) error) error {
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

// isTransactionNotSupported 判断错误是否因 MongoDB 不支持事务导致。
func isTransactionNotSupported(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Transaction numbers are only allowed on a replica set member or mongos") ||
		strings.Contains(msg, "transaction numbers are only allowed") ||
		strings.Contains(msg, "This node is not a replica set member")
}
