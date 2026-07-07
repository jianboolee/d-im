package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"d-im/pkg/model"

	"d-im/internal/user/repository"

	"github.com/nats-io/nats.go"
)

// UserService 用户服务
type UserService struct {
	repo *repository.UserRepo
}

// NewUserService 创建用户服务
func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{repo: repo}
}

// SubscribeEvents 订阅事件总线用户事件
func (s *UserService) SubscribeEvents(conn *nats.Conn) error {
	subjects := []string{
		"dimuser.created",
		"dimuser.profile_updated",
		"dimuser.status_changed",
		"dimuser.deleted",
	}

	for _, subject := range subjects {
		if _, err := conn.QueueSubscribe(subject, "im-user-sync", func(msg *nats.Msg) {
			s.handleEvent(context.Background(), msg)
		}); err != nil {
			return err
		}
		log.Printf("[user] subscribed: %s", subject)
	}
	return nil
}

// handleEvent 处理事件总线消息
func (s *UserService) handleEvent(ctx context.Context, msg *nats.Msg) {
	var envelope model.EventEnvelope
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		log.Printf("[user] invalid event: %v", err)
		return
	}

	uid := envelope.AggregateID
	eventType := envelope.Type

	switch eventType {
	case "user.created", "user.profile_updated":
		user := s.envelopeToUser(&envelope)
		if err := s.repo.Upsert(ctx, user); err != nil {
			log.Printf("[user] upsert failed: uid=%s event=%s err=%v", uid, eventType, err)
		} else {
			log.Printf("[user] synced: uid=%s event=%s nickname=%s", uid, eventType, user.Nickname)
		}

	case "user.status_changed":
		status, _ := envelope.Data["status"].(string)
		user := s.envelopeToUser(&envelope)
		user.Status = status
		if err := s.repo.Upsert(ctx, user); err != nil {
			log.Printf("[user] status update failed: uid=%s err=%v", uid, err)
		} else {
			log.Printf("[user] status_changed: uid=%s status=%s", uid, status)
		}

	case "user.deleted":
		if err := s.repo.SoftDelete(ctx, uid); err != nil {
			log.Printf("[user] soft_delete failed: uid=%s err=%v", uid, err)
		} else {
			log.Printf("[user] soft_deleted: uid=%s", uid)
		}
	}
}

// envelopeToUser 将事件信封 data 映射为 User 模型
func (s *UserService) envelopeToUser(envelope *model.EventEnvelope) *model.User {
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

// FindByID 查询用户
func (s *UserService) FindByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}
