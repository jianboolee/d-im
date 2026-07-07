package service

import (
	"context"
	"fmt"
	"strings"

	"d-im/pkg/model"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/mongo"
)

// GroupService 群组服务
type GroupService struct {
	chatColl *mongo.Collection
	convMgr  *model.ConversationManager
}

// NewGroupService 创建群组服务
func NewGroupService(chatColl *mongo.Collection, convMgr *model.ConversationManager) *GroupService {
	return &GroupService{
		chatColl: chatColl,
		convMgr:  convMgr,
	}
}

// CreateGroup 创建群聊
func (s *GroupService) CreateGroup(ctx context.Context, name, ownerUID string, memberUIDs []string) (*model.Chat, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	chat, err := model.CreateGroupChat(ctx, s.chatColl, name, ownerUID, memberUIDs)
	if err != nil {
		return nil, err
	}

	// 为所有成员创建会话视图
	if err := s.convMgr.BatchCreate(ctx, chat.Members, chat); err != nil {
		return nil, err
	}

	return chat, nil
}

// GetGroupForMember 查询群，并校验当前用户仍在群内。
func (s *GroupService) GetGroupForMember(ctx context.Context, chatID, uid string) (*model.Chat, error) {
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if chat.ChatType != types.ChatTypeGroup {
		return nil, mongo.ErrNoDocuments
	}
	if !containsString(chat.Members, uid) {
		return nil, mongo.ErrNoDocuments
	}
	return chat, nil
}

// AddMember 添加群成员
func (s *GroupService) AddMember(ctx context.Context, chatID, uid string) error {
	if err := model.AddChatMember(ctx, s.chatColl, chatID, uid); err != nil {
		return err
	}

	// 为新成员创建会话视图
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
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

// AddMembers 批量邀请群成员。
func (s *GroupService) AddMembers(ctx context.Context, chatID string, uidList []string) (*model.Chat, error) {
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if chat.ChatType != types.ChatTypeGroup {
		return nil, mongo.ErrNoDocuments
	}
	for _, uid := range uniqueNonEmpty(uidList) {
		if err := s.AddMember(ctx, chatID, uid); err != nil {
			return nil, err
		}
	}
	return model.FindChatByID(ctx, s.chatColl, chatID)
}

// RemoveMember 移除群成员
func (s *GroupService) RemoveMember(ctx context.Context, chatID, uid string) error {
	if err := model.RemoveChatMember(ctx, s.chatColl, chatID, uid); err != nil {
		return err
	}
	return s.convMgr.MarkLeft(ctx, uid, chatID)
}

// GetMembers 获取群成员列表
func (s *GroupService) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	return model.GetChatMembers(ctx, s.chatColl, chatID)
}

// UpdateName 修改群名称。
func (s *GroupService) UpdateName(ctx context.Context, chatID, name string) (*model.Chat, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	return model.UpdateGroupChatName(ctx, s.chatColl, chatID, name)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}
