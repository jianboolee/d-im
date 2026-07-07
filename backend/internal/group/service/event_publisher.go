package service

import (
	"context"
	"log"

	"d-im/pkg/model"
)

// GroupSystemEvent 群系统事件（结构化数据，通过 NATS 发布）。
type GroupSystemEvent struct {
	EventType   string   `json:"event_type"` // GroupCreated / GroupInfoUpdated / AvatarUpdated / ...
	OperatorUID string   `json:"operator_uid"`
	TargetUIDs  []string `json:"target_uids,omitempty"`
	GroupID     string   `json:"group_id"`
	GroupName   string   `json:"group_name"`
	BeforeValue string   `json:"before_value,omitempty"`
	AfterValue  string   `json:"after_value,omitempty"`
	Text        string   `json:"text,omitempty"` // 系统消息文案
}

// 事件类型常量
const (
	EventTypeGroupCreated        = "GroupCreated"
	EventTypeGroupDismissed      = "GroupDismissed"
	EventTypeGroupInfoUpdated    = "GroupInfoUpdated"
	EventTypeAvatarUpdated       = "AvatarUpdated"
	EventTypeMembersInvited      = "MembersInvited"
	EventTypeMemberJoined        = "MemberJoined"
	EventTypeMemberLeft          = "MemberLeft"
	EventTypeMemberKicked        = "MemberKicked"
	EventTypeMemberRoleChanged   = "MemberRoleChanged"
	EventTypeOwnerTransferred    = "OwnerTransferred"
	EventTypeAnnouncementUpdated = "AnnouncementUpdated"
)

// GroupUpdatePayload 推送给客户端的群信息变更载荷。
type GroupUpdatePayload struct {
	Event   string       `json:"event"`
	GroupID string       `json:"group_id"`
	Group   *model.Group `json:"group"`
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
