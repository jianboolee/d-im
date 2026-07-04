package mongodb

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Config MongoDB 配置
type Config struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
	PoolSize uint64 `yaml:"pool_size"`
	Timeout  int    `yaml:"timeout"` // 秒
}

// NewClient 创建 MongoDB 客户端
func NewClient(ctx context.Context, cfg Config) (*mongo.Database, error) {
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.PoolSize).
		SetConnectTimeout(time.Duration(cfg.Timeout) * time.Second)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	log.Printf("[mongodb] connected to %s, database: %s", cfg.URI, cfg.Database)
	return client.Database(cfg.Database), nil
}
