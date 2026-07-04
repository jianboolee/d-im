package consumer

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"d-im/internal/push/service"
	"d-im/pkg/model"

	"github.com/nats-io/nats.go"
)

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

		var m model.Message
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			log.Printf("[offline_push] unmarshal failed: %v", err)
			return
		}

		if err := c.pushSvc.PushByMessage(context.Background(), targetUID, &m); err != nil {
			log.Printf("[offline_push] push failed: uid=%s err=%v", targetUID, err)
			return
		}
		log.Printf("[offline_push] pushed: uid=%s msg=%s", targetUID, m.MsgID)
	})

	return err
}
