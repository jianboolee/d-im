package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"d-im/internal/push/service"
	"d-im/pkg/model"
	"d-im/pkg/types"

	"github.com/nats-io/nats.go"
)

type messageEnvelope struct {
	Type string `json:"type"`
	Data struct {
		Message messageDTO `json:"message"`
	} `json:"data"`
}

type messageDTO struct {
	MessageID       string                 `json:"message_id"`
	ConversationID  string                 `json:"conversation_id"`
	ChatID          string                 `json:"chat_id"`
	SenderID        string                 `json:"sender_id"`
	MessageType     string                 `json:"message_type"`
	Content         map[string]interface{} `json:"content"`
	ContentPreview  string                 `json:"content_preview"`
	ClientMessageID string                 `json:"client_message_id"`
}

// OfflinePushConsumer 离线推送消费者
type OfflinePushConsumer struct {
	pushSvc *service.PushService
	conn    *nats.Conn
}

// NewOfflinePushConsumer 创建离线推送消费者
func NewOfflinePushConsumer(pushSvc *service.PushService, conn *nats.Conn) *OfflinePushConsumer {
	return &OfflinePushConsumer{pushSvc: pushSvc, conn: conn}
}

// Start 启动订阅
func (c *OfflinePushConsumer) Start() error {
	_, err := c.conn.QueueSubscribe("im.push.offline.>", "offline-push", func(msg *nats.Msg) {
		parts := strings.Split(msg.Subject, ".")
		if len(parts) < 4 {
			return
		}
		targetUID := parts[3]

		var envelope messageEnvelope
		if err := json.Unmarshal(msg.Data, &envelope); err != nil {
			log.Printf("[offline_push] unmarshal failed: %v", err)
			return
		}
		if envelope.Type != "message" {
			return
		}
		m := model.Message{
			MsgID:          envelope.Data.Message.MessageID,
			ChatID:         envelope.Data.Message.ChatID,
			ClientMsgID:    envelope.Data.Message.ClientMessageID,
			SenderID:       envelope.Data.Message.SenderID,
			MsgType:        types.MessageType(envelope.Data.Message.MessageType),
			Content:        envelope.Data.Message.Content,
			ContentPreview: envelope.Data.Message.ContentPreview,
		}
		m.NormalizeContent()

		if err := c.pushSvc.PushByMessage(context.Background(), targetUID, &m); err != nil {
			log.Printf("[offline_push] push failed: uid=%s err=%v", targetUID, err)
			return
		}
		log.Printf("[offline_push] pushed: uid=%s msg=%s", targetUID, m.MsgID)
	})

	return err
}
