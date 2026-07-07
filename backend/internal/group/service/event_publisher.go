package service

import (
	"context"
	"log"
)

// GroupSystemEvent 群系统事件（结构化数据，由 event_publisher 生成文案）。
type GroupSystemEvent struct {
	EventType   string // GroupCreated / MembersInvited / ...
	OperatorUID string
	TargetUIDs  []string
	GroupID     string
	GroupName   string
	BeforeValue string
	AfterValue  string
	Text        string // 系统消息文案，由调用方传入
}

// EventPublisher 群领域事件发布器。
type EventPublisher struct {
	eventPort SystemEventPort
}

// NewEventPublisher 创建事件发布器。
func NewEventPublisher(eventPort SystemEventPort) *EventPublisher {
	return &EventPublisher{eventPort: eventPort}
}

// Publish 发布群系统事件。通过 SystemEventPort 发送，不阻塞主流程。
func (p *EventPublisher) Publish(ctx context.Context, event GroupSystemEvent) {
	if p == nil || p.eventPort == nil {
		return
	}
	if err := p.eventPort.PublishGroupSystemEvent(ctx, event); err != nil {
		log.Printf("[event_publisher] publish event failed: type=%s group=%s err=%v", event.EventType, event.GroupID, err)
	}
}
