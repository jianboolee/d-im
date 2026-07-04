package service

import (
	"context"
	"encoding/json"
	"log"

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

// SubscribeSync 订阅NATS用户同步主题
func (s *UserService) SubscribeSync(conn *nats.Conn, subject string) (*nats.Subscription, error) {
	sub, err := conn.Subscribe(subject, func(msg *nats.Msg) {
		s.handleSyncMsg(context.Background(), msg.Data)
	})
	if err != nil {
		return nil, err
	}
	log.Printf("[user] subscribed to sync subject: %s", subject)
	return sub, nil
}

// handleSyncMsg 处理同步消息
func (s *UserService) handleSyncMsg(ctx context.Context, data []byte) {
	var syncMsg model.UserSyncMsg
	if err := json.Unmarshal(data, &syncMsg); err != nil {
		log.Printf("[user] invalid sync message: %v", err)
		return
	}

	switch syncMsg.Action {
	case "create", "update":
		if err := s.repo.Upsert(ctx, &syncMsg.User); err != nil {
			log.Printf("[user] upsert failed: uid=%s, err=%v", syncMsg.User.ID, err)
		} else {
			log.Printf("[user] synced: uid=%s, nickname=%s", syncMsg.User.ID, syncMsg.User.Nickname)
		}
	case "delete":
		if err := s.repo.Delete(ctx, syncMsg.User.ID); err != nil {
			log.Printf("[user] delete failed: uid=%s, err=%v", syncMsg.User.ID, err)
		}
	}
}

// BatchSync 批量同步用户
func (s *UserService) BatchSync(ctx context.Context, users []*model.User) error {
	return s.repo.BatchUpsert(ctx, users)
}

// FindByID 查询用户
func (s *UserService) FindByID(ctx context.Context, id string) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}
