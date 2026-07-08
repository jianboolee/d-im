package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"d-im/internal/user/repository"
	"d-im/pkg/model"

	"github.com/nats-io/nats.go/jetstream"
)

// UserSyncService 从账户系统 JetStream stream 消费用户事件，同步到本地 MongoDB。
// 使用 durable consumer "d-im-user-sync"，服务重启后自动从上次 Ack 位置继续消费。
type UserSyncService struct {
	repo *repository.UserRepo
}

// NewUserSyncService 创建用户同步服务
func NewUserSyncService(repo *repository.UserRepo) *UserSyncService {
	return &UserSyncService{repo: repo}
}

// Start 创建 durable consumer 并开始消费 dsaas.user.> 事件。
// stream 为 JetStream stream 名称（如 USER_SYNC）。
func (s *UserSyncService) Start(ctx context.Context, js jetstream.JetStream, stream string) error {
	consumerName := "d-im-user-sync"

	cons, err := js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: "dsaas.user.>",
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
	})
	if err != nil {
		return fmt.Errorf("user sync: create consumer %s on stream %s: %w", consumerName, stream, err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := s.handleEvent(ctx, msg); err != nil {
			log.Printf("[user-sync] handle event failed: subject=%s err=%v", msg.Subject(), err)
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("user sync: consume: %w", err)
	}

	log.Printf("[user-sync] started, consuming dsaas.user.> from stream=%s", stream)

	go func() {
		<-ctx.Done()
		cc.Stop()
		log.Println("[user-sync] stopped")
	}()

	return nil
}

// handleEvent 处理单条用户事件
func (s *UserSyncService) handleEvent(ctx context.Context, msg jetstream.Msg) error {
	var envelope model.EventEnvelope
	if err := json.Unmarshal(msg.Data(), &envelope); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	uid := envelope.AggregateID
	eventType := envelope.Type

	switch eventType {
	case "user.created", "user.profile_updated":
		user := envelopeToUser(&envelope)
		if err := s.repo.Upsert(ctx, user); err != nil {
			return fmt.Errorf("upsert: uid=%s event=%s: %w", uid, eventType, err)
		}
		log.Printf("[user-sync] synced: uid=%s event=%s nickname=%s", uid, eventType, user.Nickname)

	case "user.status_changed":
		status, _ := envelope.Data["status"].(string)
		user := envelopeToUser(&envelope)
		user.Status = status
		if err := s.repo.Upsert(ctx, user); err != nil {
			return fmt.Errorf("status update: uid=%s: %w", uid, err)
		}
		log.Printf("[user-sync] status_changed: uid=%s status=%s", uid, status)

	case "user.deleted":
		if err := s.repo.SoftDelete(ctx, uid); err != nil {
			return fmt.Errorf("soft_delete: uid=%s: %w", uid, err)
		}
		log.Printf("[user-sync] soft_deleted: uid=%s", uid)

	default:
		log.Printf("[user-sync] unknown event type: %s", eventType)
	}

	return nil
}

// envelopeToUser 将事件信封 data 映射为 User 模型
func envelopeToUser(envelope *model.EventEnvelope) *model.User {
	uid := envelope.AggregateID
	nickname, _ := envelope.Data["nickname"].(string)
	avatar, _ := envelope.Data["avatar_url"].(string)
	status, _ := envelope.Data["status"].(string)

	user := &model.User{
		ID:       uid,
		Nickname: nickname,
		Avatar:   avatar,
		Status:   status,
	}

	// 解析时间戳
	if createdAt, ok := envelope.Data["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			user.CreatedAt = t
		}
	}
	if updatedAt, ok := envelope.Data["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			user.UpdatedAt = t
		}
	}

	return user
}
