package service

import (
	"d-im/internal/message/repository"
	"d-im/pkg/model"
	"d-im/pkg/snowflake"
)

// MessageService 消息服务（依赖注入容器）
type MessageService struct {
	repo    *repository.MessageRepo
	idGen   *snowflake.Generator
	chatMgr *model.ChatIDManager
	// TODO: NATS publisher
}

// NewMessageService 创建消息服务
func NewMessageService(repo *repository.MessageRepo, idGen *snowflake.Generator, chatMgr *model.ChatIDManager) *MessageService {
	return &MessageService{
		repo:    repo,
		idGen:   idGen,
		chatMgr: chatMgr,
	}
}

// GenerateMsgID 生成消息ID
func (s *MessageService) GenerateMsgID() string {
	return s.idGen.GenerateString()
}
