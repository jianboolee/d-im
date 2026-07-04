package service

import (
	"context"
	"fmt"

	"d-im/pkg/types"
)

// RecallMessageReq 撤回消息请求
type RecallMessageReq struct {
	MsgID  string `json:"msg_id"`
	UserID string `json:"user_id"` // 请求撤回的用户ID
}

// Recall 撤回消息
func (s *MessageService) Recall(ctx context.Context, req *RecallMessageReq) error {
	// 1. 查询消息
	msg, err := s.repo.FindByMsgID(ctx, req.MsgID)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	// 2. 权限校验：只有发送者才能撤回
	// TODO: 群聊中管理员也可以撤回
	if msg.FromUID != req.UserID {
		return fmt.Errorf("only message sender can recall")
	}

	// 3. 撤回
	if err := s.repo.Recall(ctx, req.MsgID); err != nil {
		return fmt.Errorf("recall message: %w", err)
	}

	return nil
}

// CanRecall 检查消息是否可以撤回
func CanRecall(msgStatus types.MessageStatus, clientTime int64) bool {
	// 已撤回的消息不能再次撤回
	if msgStatus == types.MessageStatusRecalled {
		return false
	}
	// TODO: 超过2分钟的消息不能撤回（按业务需求调整）
	return true
}
