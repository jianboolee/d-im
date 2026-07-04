package service

import (
	"d-im/internal/message/repository"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"
	"d-im/pkg/snowflake"
)

// MessageService 消息服务（依赖注入容器）
type MessageService struct {
	repo    *repository.MessageRepo
	idGen   *snowflake.Generator
	chatMgr *model.ChatIDManager
	natsPub *natsq.Publisher
}

// NewMessageService 创建消息服务
func NewMessageService(repo *repository.MessageRepo, idGen *snowflake.Generator, chatMgr *model.ChatIDManager, natsPub *natsq.Publisher) *MessageService {
	return &MessageService{
		repo:    repo,
		idGen:   idGen,
		chatMgr: chatMgr,
		natsPub: natsPub,
	}
}

// GenerateMsgID 生成消息ID
func (s *MessageService) GenerateMsgID() string {
	return s.idGen.GenerateString()
}
