package adapter

import (
	"context"
	"encoding/json"
	"log"

	groupSvc "d-im/internal/group/service"
	"d-im/pkg/model"
	natsq "d-im/pkg/queue/nats"

	"github.com/nats-io/nats.go"
)

// GroupPushConsumer 订阅 dim.group.* 事件，转为 im.push.message.{uid} 推送给在线成员。
type GroupPushConsumer struct {
	pub     *natsq.Publisher
	groups  groupRepo
	members memberRepo
	sub     *nats.Subscription
}

type groupRepo interface {
	FindActiveByChatID(ctx context.Context, chatID string) (*model.Group, error)
}

type memberRepo interface {
	ListUIDs(ctx context.Context, chatID string) ([]string, error)
}

// NewGroupPushConsumer 创建群事件推送消费者。
func NewGroupPushConsumer(pub *natsq.Publisher, groups groupRepo, members memberRepo) *GroupPushConsumer {
	return &GroupPushConsumer{pub: pub, groups: groups, members: members}
}

// Start 订阅 dim.group.* 事件。
func (c *GroupPushConsumer) Start(conn *nats.Conn) error {
	sub, err := conn.Subscribe("dim.group.>", func(msg *nats.Msg) {
		c.handle(msg)
	})
	if err != nil {
		return err
	}
	c.sub = sub
	log.Printf("[group_push_consumer] subscribed to dim.group.>")
	return nil
}

// Stop 取消订阅。
func (c *GroupPushConsumer) Stop() {
	if c.sub != nil {
		_ = c.sub.Unsubscribe()
	}
}

func (c *GroupPushConsumer) handle(msg *nats.Msg) {
	var event groupSvc.GroupSystemEvent
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		log.Printf("[group_push_consumer] unmarshal event failed: %v", err)
		return
	}
	if event.GroupID == "" {
		return
	}

	ctx := context.Background()
	// 查询群内成员列表
	uids, err := c.members.ListUIDs(ctx, event.GroupID)
	if err != nil {
		log.Printf("[group_push_consumer] list members failed: group=%s err=%v", event.GroupID, err)
		return
	}

	// 查询群最新信息
	group, err := c.groups.FindActiveByChatID(ctx, event.GroupID)
	if err != nil {
		log.Printf("[group_push_consumer] get group failed: group=%s err=%v", event.GroupID, err)
		return
	}

	payload := groupSvc.GroupUpdatePayload{
		Event:   event.EventType,
		GroupID: event.GroupID,
		Group:   group,
	}
	payloadBytes, err := json.Marshal(map[string]interface{}{
		"type": "group_updated",
		"data": payload,
	})
	if err != nil {
		return
	}

	// 推送到每个在线成员的 push channel
	for _, uid := range uids {
		subject := "im.push.message." + uid
		if err := c.pub.Publish(subject, payloadBytes); err != nil {
			log.Printf("[group_push_consumer] push failed: uid=%s subject=%s err=%v", uid, subject, err)
		}
	}
}
