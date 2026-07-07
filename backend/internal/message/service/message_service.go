package service

import (
	"context"

	"d-im/internal/message/repository"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"
	"d-im/pkg/snowflake"

	"go.mongodb.org/mongo-driver/mongo"
)

// MessageService 消息服务（依赖注入容器）
type MessageService struct {
	repo     *repository.MessageRepo
	idGen    *snowflake.Generator
	chatColl *mongo.Collection
	convMgr  *model.ConversationManager
	natsPub  *natsq.Publisher
}

// NewMessageService 创建消息服务
func NewMessageService(repo *repository.MessageRepo, idGen *snowflake.Generator, chatColl *mongo.Collection, convMgr *model.ConversationManager, natsPub *natsq.Publisher) *MessageService {
	return &MessageService{
		repo:     repo,
		idGen:    idGen,
		chatColl: chatColl,
		convMgr:  convMgr,
		natsPub:  natsPub,
	}
}

// GenerateMsgID 生成消息ID
func (s *MessageService) GenerateMsgID() string {
	return "msg_" + s.idGen.GenerateString()
}

// GetHistory 获取会话历史消息页，返回值为新到旧顺序。
func (s *MessageService) GetHistory(ctx context.Context, uid, chatID string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.FindPageByChatSeq(ctx, chatID, limit, cursor)
}

// SearchHistory 在指定会话内搜索历史消息页，返回值为新到旧顺序。
func (s *MessageService) SearchHistory(ctx context.Context, uid, chatID, keyword string, limit int64, cursor string) ([]*model.Message, string, bool, error) {
	return s.repo.SearchPageByChatSeq(ctx, chatID, keyword, limit, cursor)
}
