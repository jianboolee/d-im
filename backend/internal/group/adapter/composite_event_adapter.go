package adapter

import (
	"context"
	"encoding/json"
	"log"
	"time"

	groupSvc "d-im/internal/group/service"
	"d-im/internal/message/service"
	natsq "d-im/pkg/queue/nats"
	"d-im/pkg/types"
)

// CompositeEventAdapter 同时通过两条 NATS 管道发布群系统事件。
//
// im.message.send 管道：Message Service 订阅消费 → 创建 Message → 分发 mailbox → 更新 conversation → im.push.message.{uid}。
// dim.group.* 管道：结构化领域事件，给未来消费者（审计/分析/补推）。
type CompositeEventAdapter struct {
	pub *natsq.Publisher
}

// NewCompositeEventAdapter 创建复合事件适配器。
func NewCompositeEventAdapter(pub *natsq.Publisher) groupSvc.SystemEventPort {
	return &CompositeEventAdapter{pub: pub}
}

func (a *CompositeEventAdapter) PublishGroupSystemEvent(ctx context.Context, event groupSvc.GroupSystemEvent) error {
	if a.pub == nil || event.EventType == "" {
		return nil
	}
	// 1. 发布到 im.message.send，创建系统消息 + 触发推送管道
	a.publishMessageSend(event)

	// 2. 发布到 dim.group.*，结构化领域事件
	a.publishDomainEvent(event)

	return nil
}

// publishMessageSend 将群系统事件转为 SendMessageReq，发布到 im.message.send。
func (a *CompositeEventAdapter) publishMessageSend(event groupSvc.GroupSystemEvent) {
	text := event.Text
	if text == "" {
		text = event.EventType
	}
	req := service.SendMessageReq{
		ChatID:   event.GroupID,
		ChatType: types.ChatTypeGroup,
		SenderID: event.OperatorUID,
		MsgType:  types.MessageTypeSystemEvent,
		Content: types.SystemEventContent{
			EventType:     event.EventType,
			Text:          text,
			Title:         text,
			OperatorID:    event.OperatorUID,
			TargetUserIDs: event.TargetUIDs,
			GroupID:       event.GroupID,
			GroupName:     event.GroupName,
			BeforeValue:   event.BeforeValue,
			AfterValue:    event.AfterValue,
		},
		ClientTime: time.Now(),
	}
	data, err := json.Marshal(req)
	if err != nil {
		log.Printf("[composite_adapter] marshal message_send failed: %v", err)
		return
	}
	if err := a.pub.Publish("im.message.send", data); err != nil {
		log.Printf("[composite_adapter] publish im.message.send failed: event=%s group=%s err=%v", event.EventType, event.GroupID, err)
	}
}

// publishDomainEvent 发布结构化领域事件到 dim.group.*。
func (a *CompositeEventAdapter) publishDomainEvent(event groupSvc.GroupSystemEvent) {
	subject := "dim.group." + toSnakeCase(event.EventType)
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if err := a.pub.Publish(subject, data); err != nil {
		log.Printf("[composite_adapter] nats publish failed: subject=%s err=%v", subject, err)
	}
}
