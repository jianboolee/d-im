package redis

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config Redis配置
type Config struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
	PoolSize int    `yaml:"pool_size"`
}

// NewClient 创建Redis客户端
func NewClient(cfg Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	log.Printf("[redis] connected to %s, db=%d", cfg.Addr, cfg.DB)
	return client
}

// HealthCheck 健康检查
func HealthCheck(ctx context.Context, client *redis.Client) error {
	return client.Ping(ctx).Err()
}
