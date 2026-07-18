package service

import (
	"context"
	"errors"
	"strings"

	"d-im/internal/message/repository"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"
)

var ErrForbidden = errors.New("forbidden")

// MessageService 消息服务（依赖注入容器）
type MessageService struct {
	repo     *repository.MessageRepo
	chatRepo messageChatRepository
	groups   messageGroupReader
	users    messageUserReader
	convMgr  *model.ConversationManager
	natsPub  *natsq.Publisher
}

type messageGroupReader interface {
	GetMemberUIDs(ctx context.Context, chatID string) ([]string, error)
	CheckPermission(ctx context.Context, chatID, uid, action string) (bool, string, error)
}

type messageChatRepository interface {
	FindByChatID(ctx context.Context, chatID string) (*model.Chat, error)
	NextMessageSeq(ctx context.Context, chatID string) (int64, error)
}

type messageUserReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}

// NewMessageService 创建消息服务
func NewMessageService(repo *repository.MessageRepo, chatRepo messageChatRepository, convMgr *model.ConversationManager, natsPub *natsq.Publisher) *MessageService {
	return &MessageService{
		repo:     repo,
		chatRepo: chatRepo,
		convMgr:  convMgr,
		natsPub:  natsPub,
	}
}

func (s *MessageService) SetGroupReader(groups messageGroupReader) {
	s.groups = groups
}

func (s *MessageService) SetUserReader(users messageUserReader) {
	s.users = users
}

func (s *MessageService) senderDisplayName(ctx context.Context, senderID string) string {
	if s == nil || s.users == nil || strings.TrimSpace(senderID) == "" {
		return ""
	}
	user, err := s.users.FindByID(ctx, senderID)
	if err != nil || user == nil {
		return ""
	}
	return strings.TrimSpace(user.Nickname)
}

// GetHistory 获取会话历史消息页，返回值为新到旧顺序。
func (s *MessageService) GetHistory(ctx context.Context, uid, chatID string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.FindPageByChatSeq(ctx, chatID, limit, cursor)
}

// SearchHistory 在指定会话内搜索历史消息页，返回值为新到旧顺序。
func (s *MessageService) SearchHistory(ctx context.Context, uid, chatID, keyword string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.SearchPageByChatSeq(ctx, chatID, keyword, limit, cursor)
}
