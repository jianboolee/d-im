package service

import (
	"context"

	"d-im/pkg/model"
)

// ConversationService 会话服务
type ConversationService struct {
	convMgr *model.ConversationManager
}

// NewConversationService 创建会话服务
func NewConversationService(convMgr *model.ConversationManager) *ConversationService {
	return &ConversationService{convMgr: convMgr}
}

// GetList 获取用户会话列表
func (s *ConversationService) GetList(ctx context.Context, uid string, limit, offset int64) ([]*model.Conversation, error) {
	return s.convMgr.GetList(ctx, uid, limit, offset)
}

// SetTop 设置置顶
func (s *ConversationService) SetTop(ctx context.Context, uid, chatID string, isTop bool) error {
	return s.convMgr.SetTop(ctx, uid, chatID, isTop)
}

// SetMuted 设置免打扰
func (s *ConversationService) SetMuted(ctx context.Context, uid, chatID string, isMuted bool) error {
	return s.convMgr.SetMuted(ctx, uid, chatID, isMuted)
}

// ReadMessage 标记某会话已读
func (s *ConversationService) ReadMessage(ctx context.Context, uid, chatID string) error {
	return s.convMgr.ResetUnread(ctx, uid, chatID)
}
