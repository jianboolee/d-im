package service

import (
	"context"
	"time"

	"d-im/pkg/model"
)

// Repository 定义 Chat 领域服务需要的持久化端口。
type Repository interface {
	WithTransaction(ctx context.Context, fn func(context.Context) error) error
	InsertOrGetSingle(ctx context.Context, candidate *model.Chat) (*model.Chat, error)
	Insert(ctx context.Context, chat *model.Chat) error
	FindByChatID(ctx context.Context, chatID string) (*model.Chat, error)
}

// ConversationProjector 接收 Chat 生命周期事实并生成用户级会话视图。
type ConversationProjector interface {
	EnsureUsers(ctx context.Context, userIDs []string, chat *model.Chat) error
}

// ChatService 管理可承载消息的 Chat 实体生命周期。
type ChatService struct {
	repository    Repository
	conversations ConversationProjector
	now           func() time.Time
}

func NewChatService(repository Repository, conversations ConversationProjector) *ChatService {
	return &ChatService{repository: repository, conversations: conversations, now: time.Now}
}

// EnsureSingleChat 原子地创建或返回两个用户的单聊 Chat。
func (s *ChatService) EnsureSingleChat(ctx context.Context, userID, peerUserID string) (*model.Chat, error) {
	candidate, err := model.NewSingleChat(userID, peerUserID, s.now())
	if err != nil {
		return nil, err
	}
	var chat *model.Chat
	err = s.repository.WithTransaction(ctx, func(txCtx context.Context) error {
		chat, err = s.repository.InsertOrGetSingle(txCtx, candidate)
		if err != nil {
			return err
		}
		if s.conversations != nil {
			return s.conversations.EnsureUsers(txCtx, chat.Members, chat)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return chat, nil
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
