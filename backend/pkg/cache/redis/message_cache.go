package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// MessageCache 消息缓存
type MessageCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewMessageCache 创建消息缓存
func NewMessageCache(client *redis.Client, ttl time.Duration) *MessageCache {
	if ttl == 0 {
		ttl = 10 * time.Minute
	}
	return &MessageCache{client: client, ttl: ttl}
}

// Set 缓存消息
func (c *MessageCache) Set(ctx context.Context, msgID string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, "msg:"+msgID, payload, c.ttl).Err()
}

// Get 获取缓存消息
func (c *MessageCache) Get(ctx context.Context, msgID string, dest interface{}) error {
	val, err := c.client.Get(ctx, "msg:"+msgID).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(val, dest)
}

// Del 删除缓存
func (c *MessageCache) Del(ctx context.Context, msgID string) error {
	return c.client.Del(ctx, "msg:"+msgID).Err()
}
