package service

import (
	"context"
	"errors"

	"d-im/internal/message/repository"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrForbidden = errors.New("forbidden")

// MessageService 消息服务（依赖注入容器）
type MessageService struct {
	repo     *repository.MessageRepo
	chatColl *mongo.Collection
	groups   messageGroupReader
	convMgr  *model.ConversationManager
	natsPub  *natsq.Publisher
}

type messageGroupReader interface {
	GetMemberUIDs(ctx context.Context, chatID string) ([]string, error)
	CheckPermission(ctx context.Context, chatID, uid, action string) (bool, string, error)
}

// NewMessageService 创建消息服务
func NewMessageService(repo *repository.MessageRepo, chatColl *mongo.Collection, convMgr *model.ConversationManager, natsPub *natsq.Publisher) *MessageService {
	return &MessageService{
		repo:     repo,
		chatColl: chatColl,
		convMgr:  convMgr,
		natsPub:  natsPub,
	}
}

func (s *MessageService) SetGroupReader(groups messageGroupReader) {
	s.groups = groups
}

// GenerateMsgID 生成消息ID（UUID v7）。
func (s *MessageService) GenerateMsgID() string {
	return "msg_" + uuid.Must(uuid.NewV7()).String()
}

// GetHistory 获取会话历史消息页，返回值为新到旧顺序。
func (s *MessageService) GetHistory(ctx context.Context, uid, chatID string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.FindPageByChatSeq(ctx, chatID, limit, cursor)
}

// SearchHistory 在指定会话内搜索历史消息页，返回值为新到旧顺序。
func (s *MessageService) SearchHistory(ctx context.Context, uid, chatID, keyword string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.SearchPageByChatSeq(ctx, chatID, keyword, limit, cursor)
}
