package router

import (
	"fmt"
	"sort"
	"strings"

	"d-im/pkg/types"
)

// SingleRouter 单聊路由策略
type SingleRouter struct{}

// NewSingleRouter 创建单聊路由器
func NewSingleRouter() *SingleRouter {
	return &SingleRouter{}
}

// Route 单聊路由：chatID格式为 single_uid1_uid2，接收者是非发送者的另一方
func (r *SingleRouter) Route(chatID string, fromUID string) (*RouteResult, error) {
	if !strings.HasPrefix(chatID, "single_") {
		return nil, fmt.Errorf("invalid single chat id: %s", chatID)
	}

	// 单聊 chatID = "single_uidA_uidB"
	parts := strings.SplitN(chatID, "_", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid single chat id format: %s", chatID)
	}

	uid1 := parts[1]
	uid2 := parts[2]

	var targetUIDs []string
	if uid1 == fromUID {
		targetUIDs = append(targetUIDs, uid2)
	} else if uid2 == fromUID {
		targetUIDs = append(targetUIDs, uid1)
	} else {
		// 发送者不在会话中
		targetUIDs = append(targetUIDs, uid1, uid2)
	}

	return &RouteResult{
		ChatID:     chatID,
		ChatType:   types.ChatTypeSingle,
		TargetUIDs: targetUIDs,
	}, nil
}

// GroupRouter 群聊路由策略
type GroupRouter struct {
	// TODO: 注入群组成员查询接口
}

// NewGroupRouter 创建群聊路由器
func NewGroupRouter() *GroupRouter {
	return &GroupRouter{}
}

// Route 群聊路由：需要查询群成员列表，排除发送者
func (r *GroupRouter) Route(chatID string, fromUID string) (*RouteResult, error) {
	// TODO: 从 DB/缓存中查询群成员列表
	return &RouteResult{
		ChatID:   chatID,
		ChatType: types.ChatTypeGroup,
	}, nil
}

// GenerateSingleChatID 生成单聊会话ID（排序保证唯一性）
func GenerateSingleChatID(uid1, uid2 string) string {
	uids := []string{uid1, uid2}
	sort.Strings(uids)
	return fmt.Sprintf("single_%s_%s", uids[0], uids[1])
}
