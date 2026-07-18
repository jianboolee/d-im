package service

import (
	"context"
	"encoding/json"
	"fmt"

	"d-im/pkg/types"
)

// ForwardMessageReq 转发消息请求
type ForwardMessageReq struct {
	SrcMsgID      string      `json:"src_msg_id"`      // 源消息ID
	TargetChatIDs []string    `json:"target_chat_ids"` // 目标会话ID列表
	SenderID      string      `json:"sender_id"`       // 转发者UID
	ClientTime    interface{} `json:"client_time"`     // 客户端时间
}

// ForwardMessageResp 转发消息响应
type ForwardMessageResp struct {
	SrcMsgID   string   `json:"src_msg_id"`
	NewMsgIDs  []string `json:"new_msg_ids"` // 生成的新消息ID列表
	SuccessCnt int      `json:"success_cnt"`
	FailedCnt  int      `json:"failed_cnt"`
}

// Forward 转发消息到指定的会话
func (s *MessageService) Forward(ctx context.Context, req *ForwardMessageReq) (*ForwardMessageResp, error) {
	// 1. 查询源消息
	srcMsg, err := s.repo.FindByMsgID(ctx, req.SrcMsgID)
	if err != nil {
		return nil, fmt.Errorf("source message not found: %w", err)
	}

	// 2. 验证源消息是否可以转发（已撤回的不能转发）
	if srcMsg.Status == types.MessageStatusRecalled {
		return nil, fmt.Errorf("cannot forward a recalled message")
	}

	// 3. 类型断言：确保源消息内容实现了 ContentType 接口
	content, ok := srcMsg.Content.(types.ContentType)
	if !ok {
		return nil, fmt.Errorf("source message content does not implement ContentType")
	}

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	// 4. 逐会话转发
	resp := &ForwardMessageResp{
		SrcMsgID: req.SrcMsgID,
	}

	for _, targetChatID := range req.TargetChatIDs {
		sendReq := &SendMessageReq{
			ChatID:     targetChatID,
			SenderID:   req.SenderID,
			MsgType:    srcMsg.MsgType,
			Content:    contentBytes,
			ClientTime: srcMsg.ClientTime,
		}

		sendResp, err := s.Send(ctx, sendReq)
		if err != nil {
			resp.FailedCnt++
			continue
		}

		resp.NewMsgIDs = append(resp.NewMsgIDs, sendResp.MsgID)
		resp.SuccessCnt++
	}

	return resp, nil
}
