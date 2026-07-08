package service

import (
	"context"
	"log"
	"strings"

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
	users     UserProfileReader
}

// NewEventPublisher 创建事件发布器。
func NewEventPublisher(eventPort SystemEventPort) *EventPublisher {
	return &EventPublisher{eventPort: eventPort}
}

// SetUserProfileReader 注入用户资料读取器，用于生成面向用户的系统事件文案。
func (p *EventPublisher) SetUserProfileReader(users UserProfileReader) {
	p.users = users
}

// Publish 发布群系统事件。通过 SystemEventPort 发送，不阻塞主流程。
func (p *EventPublisher) Publish(ctx context.Context, event GroupSystemEvent) {
	if p == nil || p.eventPort == nil {
		return
	}
	if strings.TrimSpace(event.Text) == "" {
		event.Text = p.formatText(ctx, event)
	}
	if err := p.eventPort.PublishGroupSystemEvent(ctx, event); err != nil {
		log.Printf("[event_publisher] publish event failed: type=%s group=%s err=%v", event.EventType, event.GroupID, err)
	}
}

func (p *EventPublisher) formatText(ctx context.Context, event GroupSystemEvent) string {
	operator := p.displayName(ctx, event.OperatorUID)
	targets := p.displayNames(ctx, event.TargetUIDs)
	groupName := strings.TrimSpace(event.GroupName)
	after := strings.TrimSpace(event.AfterValue)
	before := strings.TrimSpace(event.BeforeValue)

	switch event.EventType {
	case EventTypeGroupCreated:
		if groupName != "" {
			return operator + "创建了群「" + groupName + "」"
		}
		return operator + "创建了群"
	case EventTypeMembersInvited:
		if targets != "" {
			return operator + "邀请" + targets + "加入了群聊"
		}
		return operator + "邀请成员加入了群聊"
	case EventTypeMemberJoined:
		return operator + "加入了群聊"
	case EventTypeMemberLeft:
		return operator + "退出了群聊"
	case EventTypeMemberKicked:
		if targets != "" {
			return operator + "将" + targets + "移出了群聊"
		}
		return operator + "移出了群成员"
	case EventTypeGroupInfoUpdated:
		if before != "" && after != "" && before != after {
			return operator + "将群名修改为「" + after + "」"
		}
		return operator + "修改了群信息"
	case EventTypeAvatarUpdated:
		if operator != "" && operator != "系统" {
			return operator + "修改了群头像"
		}
		return "群头像已更新"
	case EventTypeAnnouncementUpdated:
		return operator + "更新了群公告"
	case EventTypeMemberRoleChanged:
		if targets != "" {
			return operator + "修改了" + targets + "的群角色"
		}
		return operator + "修改了群成员角色"
	case EventTypeOwnerTransferred:
		if targets != "" {
			return operator + "将群主转让给" + targets
		}
		return operator + "转让了群主"
	case EventTypeGroupDismissed:
		return operator + "解散了群聊"
	default:
		return "群聊状态已更新"
	}
}

func (p *EventPublisher) displayNames(ctx context.Context, uids []string) string {
	names := make([]string, 0, len(uids))
	for _, uid := range uids {
		if name := p.displayName(ctx, uid); name != "" {
			names = append(names, name)
		}
	}
	return strings.Join(names, "、")
}

func (p *EventPublisher) displayName(ctx context.Context, uid string) string {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "系统"
	}
	if p.users == nil {
		return uid
	}
	user, err := p.users.FindByID(ctx, uid)
	if err != nil || user == nil {
		return uid
	}
	if name := strings.TrimSpace(user.Nickname); name != "" {
		return name
	}
	return uid
}
