package service

import (
	"context"
	"fmt"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

// SendMessageReq 发送消息请求
type SendMessageReq struct {
	ChatID     string            `json:"chat_id"`
	ChatType   types.ChatType    `json:"chat_type"`
	FromUID    string            `json:"from_uid"`
	FromName   string            `json:"from_name"`
	MsgType    types.MessageType `json:"msg_type"`
	Content    types.ContentType `json:"content"`
	ClientTime time.Time         `json:"client_time"`
	TargetUIDs []string          `json:"target_uids,omitempty"` // 接收者UID列表
	QuoteMsgID string            `json:"quote_msg_id,omitempty"`
}

// SendMessageResp 发送消息响应
type SendMessageResp struct {
	MsgID      string              `json:"msg_id"`
	ServerTime time.Time           `json:"server_time"`
	Status     types.MessageStatus `json:"status"`
}

// Send 发送消息（完整链路：构建消息 → 存储 → 分发到信箱）
func (s *MessageService) Send(ctx context.Context, req *SendMessageReq) (*SendMessageResp, error) {
	// 1. 校验内容
	if err := req.Content.Validate(); err != nil {
		return nil, fmt.Errorf("content validation failed: %w", err)
	}

	// 2. 生成消息ID
	msgID := s.GenerateMsgID()
	now := time.Now()

	// 3. 构建消息模型
	msg := &model.Message{
		MsgID:      msgID,
		ChatID:     req.ChatID,
		ChatType:   req.ChatType,
		FromUID:    req.FromUID,
		FromName:   req.FromName,
		MsgType:    req.MsgType,
		Content:    req.Content,
		Status:     types.MessageStatusSent,
		ClientTime: req.ClientTime,
		ServerTime: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	// 4. 处理引用消息
	if req.QuoteMsgID != "" {
		msg.QuoteMsgID = req.QuoteMsgID
		if quoted, err := s.repo.FindByMsgID(ctx, req.QuoteMsgID); err == nil {
			msg.QuoteMsg = &types.QuoteMessage{
				MsgID:          quoted.MsgID,
				FromUID:        quoted.FromUID,
				FromName:       quoted.FromName,
				MsgType:        quoted.MsgType,
				ContentPreview: getContentPreview(quoted.Content),
			}
		}
	}

	// 5. 存储消息
	if err := s.repo.Insert(ctx, msg); err != nil {
		return nil, fmt.Errorf("insert message: %w", err)
	}

	// 6. 分发到用户信箱
	if len(req.TargetUIDs) > 0 {
		s.distributeToMailbox(ctx, msg, req.TargetUIDs)
	}

	return &SendMessageResp{
		MsgID:      msgID,
		ServerTime: now,
		Status:     types.MessageStatusSent,
	}, nil
}

// distributeToMailbox 将消息分发到目标用户的信箱
func (s *MessageService) distributeToMailbox(ctx context.Context, msg *model.Message, targetUIDs []string) {
	mailboxes := make([]*model.UserMailbox, len(targetUIDs))
	for i, uid := range targetUIDs {
		mailboxes[i] = &model.UserMailbox{
			UID:    uid,
			ChatID: msg.ChatID,
			MsgID:  msg.MsgID,
			SeqID:  s.idGen.Generate(),
			Status: types.MessageStatusDelivered,
		}
	}
	// 信箱写入失败不影响主流程，异步记录日志
	_ = s.repo.BatchInsertToMailbox(ctx, mailboxes)
}

// getContentPreview 获取内容预览文本
func getContentPreview(content interface{}) string {
	switch c := content.(type) {
	case types.TextContent:
		preview := c.Text
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		return preview
	case types.ImageContent:
		return "[图片]"
	case types.VideoContent:
		return "[视频]"
	case types.VoiceContent:
		return "[语音]"
	case types.FileContent:
		return "[文件] " + c.FileName
	case types.LocationContent:
		return "[位置]"
	case types.CardContent:
		return "[卡片] " + c.Title
	case types.LinkContent:
		return "[链接] " + c.Title
	case types.TemplateContent:
		return "[模板] " + c.Title
	default:
		return "[消息]"
	}
}
