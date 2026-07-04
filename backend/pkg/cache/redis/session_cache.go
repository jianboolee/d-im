package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// SessionCache 用户会话缓存
type SessionCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSessionCache 创建会话缓存
func NewSessionCache(client *redis.Client) *SessionCache {
	return &SessionCache{
		client: client,
		ttl:    7 * 24 * time.Hour, // 7天过期
	}
}

// SetToken 存储用户token
func (c *SessionCache) SetToken(ctx context.Context, uid, token string) error {
	return c.client.Set(ctx, "session:"+uid, token, c.ttl).Err()
}

// GetToken 获取用户token
func (c *SessionCache) GetToken(ctx context.Context, uid string) (string, error) {
	return c.client.Get(ctx, "session:"+uid).Result()
}

// DelToken 删除用户token
func (c *SessionCache) DelToken(ctx context.Context, uid string) error {
	return c.client.Del(ctx, "session:"+uid).Err()
}
