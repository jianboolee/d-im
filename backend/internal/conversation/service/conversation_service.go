package service

import (
	"context"
	"fmt"

	"d-im/pkg/model"

	"go.mongodb.org/mongo-driver/mongo"
)

// ConversationService 会话服务
type ConversationService struct {
	convMgr  *model.ConversationManager
	chatColl *mongo.Collection
}

// NewConversationService 创建会话服务
func NewConversationService(convMgr *model.ConversationManager, chatColl *mongo.Collection) *ConversationService {
	return &ConversationService{convMgr: convMgr, chatColl: chatColl}
}

// GetList 获取用户会话列表
func (s *ConversationService) GetList(ctx context.Context, uid string, limit, offset int64) ([]*model.Conversation, error) {
	return s.convMgr.GetList(ctx, uid, limit, offset)
}

// GetListByCursor 获取用户会话列表（cursor分页）
func (s *ConversationService) GetListByCursor(ctx context.Context, uid string, limit int64, cursor string) ([]*model.Conversation, string, bool, error) {
	return s.convMgr.GetListByCursor(ctx, uid, limit, cursor)
}

// GetConversation 获取当前用户的会话视图。
func (s *ConversationService) GetConversation(ctx context.Context, uid, conversationID string) (*model.Conversation, error) {
	return s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// GetConversationByChatID 获取当前用户在指定 chat 下的会话视图。
func (s *ConversationService) GetConversationByChatID(ctx context.Context, uid, chatID string) (*model.Conversation, error) {
	return s.convMgr.FindByUIDAndChatID(ctx, uid, chatID)
}

// CreateOrGetSingle 创建或获取单聊会话，并确保双方会话视图存在
func (s *ConversationService) CreateOrGetSingle(ctx context.Context, uid, peerUserID string) (*model.Conversation, error) {
	chat, err := model.CreateOrGetSingleChat(ctx, s.chatColl, uid, peerUserID)
	if err != nil {
		return nil, err
	}

	for _, memberID := range chat.Members {
		conv := &model.Conversation{
			UID:      memberID,
			ChatID:   chat.ChatID,
			ChatType: chat.ChatType,
		}
		if err := s.convMgr.CreateOrUpdate(ctx, conv); err != nil {
			return nil, err
		}
	}

	return s.convMgr.FindByUIDAndChatID(ctx, uid, chat.ChatID)
}

// SetTop 设置置顶。
func (s *ConversationService) SetTop(ctx context.Context, uid, conversationID string, isTop bool) (*model.Conversation, error) {
	conv, err := s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.convMgr.SetTop(ctx, uid, conv.ChatID, isTop); err != nil {
		return nil, err
	}
	return s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// SetMuted 设置免打扰。
func (s *ConversationService) SetMuted(ctx context.Context, uid, conversationID string, isMuted bool) (*model.Conversation, error) {
	conv, err := s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.convMgr.SetMuted(ctx, uid, conv.ChatID, isMuted); err != nil {
		return nil, err
	}
	return s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
}

func (s *ConversationService) UpdateSettings(ctx context.Context, uid, conversationID string, pinned, muted *bool) (*model.Conversation, error) {
	conv, err := s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if pinned != nil {
		if err := s.convMgr.SetTop(ctx, uid, conv.ChatID, *pinned); err != nil {
			return nil, err
		}
	}
	if muted != nil {
		if err := s.convMgr.SetMuted(ctx, uid, conv.ChatID, *muted); err != nil {
			return nil, err
		}
	}
	return s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// ReadConversation 标记某会话已读。lastReadSeq 为 0 时使用会话当前最新消息序列。
func (s *ConversationService) ReadConversation(ctx context.Context, uid, conversationID string, lastReadSeq int64) (*model.Conversation, error) {
	conv, err := s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if lastReadSeq <= 0 {
		if s.chatColl == nil {
			return nil, fmt.Errorf("chat collection is required")
		}
		chat, err := model.FindChatByID(ctx, s.chatColl, conv.ChatID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, err
			}
			return nil, fmt.Errorf("find chat: %w", err)
		}
		lastReadSeq = chat.LastSeq
	} else if s.chatColl != nil {
		chat, err := model.FindChatByID(ctx, s.chatColl, conv.ChatID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, err
			}
			return nil, fmt.Errorf("find chat: %w", err)
		}
		if lastReadSeq > chat.LastSeq {
			lastReadSeq = chat.LastSeq
		}
	}
	if err := s.convMgr.MarkRead(ctx, uid, conv.ChatID, lastReadSeq); err != nil {
		return nil, err
	}
	return s.convMgr.FindByUIDAndConversationID(ctx, uid, conversationID)
}
