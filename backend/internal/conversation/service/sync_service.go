package service

import (
	"context"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

// SyncService 会话同步服务
type SyncService struct {
	convMgr *model.ConversationManager
}

// NewSyncService 创建同步服务
func NewSyncService(convMgr *model.ConversationManager) *SyncService {
	return &SyncService{convMgr: convMgr}
}

// InitConversation 为用户初始化会话视图（首次聊天时调用）
func (s *SyncService) InitConversation(ctx context.Context, uid, chatID string, chatType string) error {
	conv := &model.Conversation{
		UID:      uid,
		ChatID:   chatID,
		ChatType: types.ChatType(chatType),
	}
	return s.convMgr.CreateOrUpdate(ctx, conv)
}

// OnNewMessage 新消息到达时更新会话摘要
func (s *SyncService) OnNewMessage(ctx context.Context, uid, chatID string, lastMsg *types.LastMessage) error {
	// 更新最后消息摘要 & 总消息数+1
	if err := s.convMgr.UpdateLastMsg(ctx, uid, chatID, lastMsg); err != nil {
		return err
	}
	// 未读+1（除了发送者自己）
	return s.convMgr.UpdateUnreadCount(ctx, uid, chatID, 1)
}
