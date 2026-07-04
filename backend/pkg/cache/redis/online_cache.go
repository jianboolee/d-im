package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// OnlineCache 在线状态缓存
type OnlineCache struct {
	client *redis.Client
}

// NewOnlineCache 创建在线缓存
func NewOnlineCache(client *redis.Client) *OnlineCache {
	return &OnlineCache{client: client}
}

// SetOnline 设置用户在线
func (c *OnlineCache) SetOnline(ctx context.Context, uid, serverID string, ttl time.Duration) error {
	return c.client.Set(ctx, "online:"+uid, serverID, ttl).Err()
}

// SetOffline 设置用户离线
func (c *OnlineCache) SetOffline(ctx context.Context, uid string) error {
	return c.client.Del(ctx, "online:"+uid).Err()
}

// IsOnline 检查用户是否在线
func (c *OnlineCache) IsOnline(ctx context.Context, uid string) bool {
	_, err := c.client.Get(ctx, "online:"+uid).Result()
	return err == nil
}

// GetOnlineCount 获取在线用户数（使用SCAN估计）
func (c *OnlineCache) GetOnlineCount(ctx context.Context) (int64, error) {
	var count int64
	iter := c.client.Scan(ctx, 0, "online:*", 100).Iterator()
	for iter.Next(ctx) {
		count++
	}
	return count, iter.Err()
}
