package router

import (
	"context"
	"fmt"

	chatRepo "d-im/internal/chat/repository"
	"d-im/pkg/types"
)

// SingleRouter 单聊路由策略
type SingleRouter struct {
	chatRepo *chatRepo.ChatRepo
}

// NewSingleRouter 创建单聊路由器
func NewSingleRouter(chatRepo *chatRepo.ChatRepo) *SingleRouter {
	return &SingleRouter{chatRepo: chatRepo}
}

// Route 单聊路由：查询会话成员，接收者是非发送者的另一方。
func (r *SingleRouter) Route(chatID string, fromUID string) (*RouteResult, error) {
	if r.chatRepo == nil {
		return &RouteResult{
			ChatID:   chatID,
			ChatType: types.ChatTypeSingle,
		}, nil
	}

	members, err := r.chatRepo.GetMembers(context.Background(), chatID)
	if err != nil {
		return nil, fmt.Errorf("get single chat members: %w", err)
	}

	targetUIDs := make([]string, 0, len(members))
	for _, uid := range members {
		if uid != fromUID {
			targetUIDs = append(targetUIDs, uid)
		}
	}

	return &RouteResult{
		ChatID:     chatID,
		ChatType:   types.ChatTypeSingle,
		TargetUIDs: targetUIDs,
	}, nil
}

// GroupRouter 群聊路由策略
type GroupRouter struct {
	groups groupMemberReader
}

type groupMemberReader interface {
	GetMemberUIDs(ctx context.Context, chatID string) ([]string, error)
}

// NewGroupRouter 创建群聊路由器
func NewGroupRouter(groups groupMemberReader) *GroupRouter {
	return &GroupRouter{groups: groups}
}

// Route 群聊路由：需要查询群成员列表，排除发送者
func (r *GroupRouter) Route(chatID string, fromUID string) (*RouteResult, error) {
	if r.groups == nil {
		return &RouteResult{
			ChatID:   chatID,
			ChatType: types.ChatTypeGroup,
		}, nil
	}

	members, err := r.groups.GetMemberUIDs(context.Background(), chatID)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}

	targetUIDs := make([]string, 0, len(members))
	for _, uid := range members {
		if uid != fromUID {
			targetUIDs = append(targetUIDs, uid)
		}
	}

	return &RouteResult{
		ChatID:     chatID,
		ChatType:   types.ChatTypeGroup,
		TargetUIDs: targetUIDs,
	}, nil
}
