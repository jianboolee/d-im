package service

import (
	"context"
	"log"

	"d-im/internal/push/provider"
	"d-im/pkg/model"
)

// PushService 推送服务
type PushService struct {
	factory *provider.ProviderFactory
	builder *provider.PushContentBuilder
}

// NewPushService 创建推送服务
func NewPushService(factory *provider.ProviderFactory) *PushService {
	return &PushService{
		factory: factory,
		builder: &provider.PushContentBuilder{},
	}
}

// PushByMessage 根据消息推送给目标用户
func (s *PushService) PushByMessage(ctx context.Context, userID string, msg *model.Message) error {
	p, err := s.factory.GetProvider()
	if err != nil {
		return err
	}

	title, body := s.builder.BuildPushContent(msg.MsgType, msg.Content, msg.FromName)

	req := &provider.PushRequest{
		Platform: provider.PlatformWeb,
		UserID:   userID,
		Title:    title,
		Body:     body,
		Sound:    "default",
		Badge:    1,
		Extra: map[string]interface{}{
			"chat_id": msg.ChatID,
			"msg_id":  msg.MsgID,
			"action":  "open_chat",
		},
		MsgID:    msg.MsgID,
		ChatID:   msg.ChatID,
		MsgType:  msg.MsgType,
		Priority: provider.PushPriorityNormal,
		TTL:      3600,
	}

	resp, err := p.Push(ctx, req)
	if err != nil {
		return err
	}

	log.Printf("[push] sent: user=%s msg=%s success=%v", userID, msg.MsgID, resp.Success)
	return nil
}
