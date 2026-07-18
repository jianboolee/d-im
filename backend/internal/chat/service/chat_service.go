package service

import (
	"context"
	"time"

	"d-im/pkg/model"
)

// Repository 定义 Chat 领域服务需要的持久化端口。
type Repository interface {
	InsertOrGetSingle(ctx context.Context, candidate *model.Chat) (*model.Chat, error)
	Insert(ctx context.Context, chat *model.Chat) error
	FindByChatID(ctx context.Context, chatID string) (*model.Chat, error)
}

// ChatService 管理可承载消息的 Chat 实体生命周期。
type ChatService struct {
	repository Repository
	now        func() time.Time
}

func NewChatService(repository Repository) *ChatService {
	return &ChatService{repository: repository, now: time.Now}
}

// EnsureSingleChat 原子地创建或返回两个用户的单聊 Chat。
func (s *ChatService) EnsureSingleChat(ctx context.Context, userID, peerUserID string) (*model.Chat, error) {
	candidate, err := model.NewSingleChat(userID, peerUserID, s.now())
	if err != nil {
		return nil, err
	}
	return s.repository.InsertOrGetSingle(ctx, candidate)
}

// CreateGroupChat 创建用于承载群消息的 Chat。群资料和成员规则由 GroupService 管理。
func (s *ChatService) CreateGroupChat(ctx context.Context, creatorUserID string) (*model.Chat, error) {
	chat, err := model.NewGroupChat(creatorUserID, s.now())
	if err != nil {
		return nil, err
	}
	if err := s.repository.Insert(ctx, chat); err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) GetChat(ctx context.Context, chatID string) (*model.Chat, error) {
	return s.repository.FindByChatID(ctx, chatID)
}
