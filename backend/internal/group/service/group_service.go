package service

import (
	"context"

	"d-im/pkg/model"
)

// GroupService 群组服务
type GroupService struct {
	chatMgr *model.ChatIDManager
	convMgr *model.ConversationManager
}

// NewGroupService 创建群组服务
func NewGroupService(chatMgr *model.ChatIDManager, convMgr *model.ConversationManager) *GroupService {
	return &GroupService{
		chatMgr: chatMgr,
		convMgr: convMgr,
	}
}

// CreateGroup 创建群聊
func (s *GroupService) CreateGroup(ctx context.Context, name, ownerUID string, memberUIDs []string) (*model.Chat, error) {
	chat, err := s.chatMgr.CreateGroupChat(ctx, name, ownerUID, memberUIDs)
	if err != nil {
		return nil, err
	}

	// 为所有成员创建会话视图
	if err := s.convMgr.BatchCreate(ctx, chat.Members, chat); err != nil {
		return nil, err
	}

	return chat, nil
}

// AddMember 添加群成员
func (s *GroupService) AddMember(ctx context.Context, chatID, uid string) error {
	if err := s.chatMgr.AddMember(ctx, chatID, uid); err != nil {
		return err
	}

	// 为新成员创建会话视图
	chat, err := s.chatMgr.FindByChatID(ctx, chatID)
	if err != nil {
		return err
	}

	conv := &model.Conversation{
		UID:      uid,
		ChatID:   chatID,
		ChatType: chat.ChatType,
	}
	return s.convMgr.CreateOrUpdate(ctx, conv)
}

// RemoveMember 移除群成员
func (s *GroupService) RemoveMember(ctx context.Context, chatID, uid string) error {
	return s.chatMgr.RemoveMember(ctx, chatID, uid)
}

// GetMembers 获取群成员列表
func (s *GroupService) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	return s.chatMgr.GetMembers(ctx, chatID)
}
