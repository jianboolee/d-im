package service

import (
	"context"
	"fmt"

	chatSvc "d-im/internal/chat/service"
	conversationRepo "d-im/internal/conversation/repository"
	"d-im/pkg/model"

	"go.mongodb.org/mongo-driver/mongo"
)

// ConversationService 会话服务
type ConversationService struct {
	repo  *conversationRepo.ConversationRepo
	chats *chatSvc.ChatService
}

// NewConversationService 创建会话服务
func NewConversationService(repo *conversationRepo.ConversationRepo, chats *chatSvc.ChatService) *ConversationService {
	return &ConversationService{repo: repo, chats: chats}
}

// GetList 获取用户会话列表
func (s *ConversationService) GetList(ctx context.Context, uid string, limit, offset int64) ([]*model.Conversation, error) {
	return s.repo.GetList(ctx, uid, limit, offset)
}

// GetListByCursor 获取用户会话列表（cursor分页）
func (s *ConversationService) GetListByCursor(ctx context.Context, uid string, limit int64, cursor string) ([]*model.Conversation, string, bool, error) {
	return s.repo.GetListByCursor(ctx, uid, limit, cursor)
}

// GetConversation 获取当前用户的会话视图。
func (s *ConversationService) GetConversation(ctx context.Context, uid, conversationID string) (*model.Conversation, error) {
	return s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// GetConversationByChatID 获取当前用户在指定 chat 下的会话视图。
func (s *ConversationService) GetConversationByChatID(ctx context.Context, uid, chatID string) (*model.Conversation, error) {
	return s.repo.FindByUIDAndChatID(ctx, uid, chatID)
}

// SetTop 设置置顶。
func (s *ConversationService) SetTop(ctx context.Context, uid, conversationID string, isTop bool) (*model.Conversation, error) {
	conv, err := s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetTop(ctx, uid, conv.ChatID, isTop); err != nil {
		return nil, err
	}
	return s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// SetMuted 设置免打扰。
func (s *ConversationService) SetMuted(ctx context.Context, uid, conversationID string, isMuted bool) (*model.Conversation, error) {
	conv, err := s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.SetMuted(ctx, uid, conv.ChatID, isMuted); err != nil {
		return nil, err
	}
	return s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
}

func (s *ConversationService) UpdateSettings(ctx context.Context, uid, conversationID string, pinned, muted *bool) (*model.Conversation, error) {
	conv, err := s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if pinned != nil {
		if err := s.repo.SetTop(ctx, uid, conv.ChatID, *pinned); err != nil {
			return nil, err
		}
	}
	if muted != nil {
		if err := s.repo.SetMuted(ctx, uid, conv.ChatID, *muted); err != nil {
			return nil, err
		}
	}
	return s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
}

// ReadConversation 标记某会话已读。lastReadSeq 为 0 时使用会话当前最新消息序列。
func (s *ConversationService) ReadConversation(ctx context.Context, uid, conversationID string, lastReadSeq int64) (*model.Conversation, error) {
	conv, err := s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
	if err != nil {
		return nil, err
	}
	if lastReadSeq <= 0 {
		if s.chats == nil {
			return nil, fmt.Errorf("chat repository is required")
		}
		chat, err := s.chats.GetChat(ctx, conv.ChatID)
		if err != nil {
			if err == mongo.ErrNoDocuments {
				return nil, err
			}
			return nil, fmt.Errorf("find chat: %w", err)
		}
		lastReadSeq = chat.LastSeq
	} else if s.chats != nil {
		chat, err := s.chats.GetChat(ctx, conv.ChatID)
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
	if err := s.repo.MarkRead(ctx, uid, conv.ChatID, lastReadSeq); err != nil {
		return nil, err
	}
	return s.repo.FindByUIDAndConversationID(ctx, uid, conversationID)
}
