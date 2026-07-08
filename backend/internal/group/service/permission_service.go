package service

import (
	"context"
	"time"

	"d-im/internal/group/repository"
	"d-im/pkg/model"
)

// PermissionResult 权限判断结果。
type PermissionResult struct {
	Allowed bool
	Reason  string
}

// PermissionAction 权限动作类型。
type PermissionAction string

const (
	ActionSendMessage     PermissionAction = "send_message"
	ActionInviteMember    PermissionAction = "invite_member"
	ActionJoinGroup       PermissionAction = "join_group"
	ActionKickMember      PermissionAction = "kick_member"
	ActionUpdateGroupInfo PermissionAction = "update_group_info"
	ActionUpdateSettings  PermissionAction = "update_settings"
	ActionDismissGroup    PermissionAction = "dismiss_group"
	ActionTransferOwner   PermissionAction = "transfer_owner"
	ActionSetMemberRole   PermissionAction = "set_member_role"
)

// PermissionService 群权限判断服务，只读、不修改数据库、不发布事件。
type PermissionService struct {
	groups  *repository.GroupRepo
	members *repository.MemberRepo
}

// NewPermissionService 创建权限服务。
func NewPermissionService(groups *repository.GroupRepo, members *repository.MemberRepo) *PermissionService {
	return &PermissionService{groups: groups, members: members}
}

// CheckPermission 判断操作是否允许。
func (p *PermissionService) CheckPermission(ctx context.Context, chatID, uid string, action PermissionAction) (PermissionResult, error) {
	group, member, err := p.requireMember(ctx, chatID, uid)
	if err != nil {
		return PermissionResult{Allowed: false, Reason: "not_group_member"}, err
	}
	if group.Settings.IsMutedAll && !isPrivileged(member) {
		return PermissionResult{Allowed: false, Reason: "group_muted_all"}, nil
	}
	if isMemberMuted(group, member, time.Now()) {
		return PermissionResult{Allowed: false, Reason: "member_muted"}, nil
	}
	switch action {
	case ActionSendMessage:
		return PermissionResult{Allowed: true}, nil
	case ActionInviteMember:
		if !canInviteMembers(group, member) {
			return PermissionResult{Allowed: false, Reason: "permission_denied"}, nil
		}
	case ActionKickMember:
		if !canManageMembers(group, member) {
			return PermissionResult{Allowed: false, Reason: "permission_denied"}, nil
		}
	case ActionUpdateGroupInfo:
		if !canEditGroupInfo(member) {
			return PermissionResult{Allowed: false, Reason: "permission_denied"}, nil
		}
	case ActionJoinGroup:
		if group.Settings.JoinMethod != model.JoinMethodFree || !group.Settings.IsPublic {
			return PermissionResult{Allowed: false, Reason: "join_not_allowed"}, nil
		}
	case ActionDismissGroup, ActionTransferOwner, ActionSetMemberRole, ActionUpdateSettings:
		if member.Role != model.MemberRoleOwner {
			return PermissionResult{Allowed: false, Reason: "owner_required"}, nil
		}
	}
	return PermissionResult{Allowed: true}, nil
}

func (p *PermissionService) requireMember(ctx context.Context, chatID, uid string) (*model.Group, *model.GroupMember, error) {
	group, err := p.groups.FindActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}
	member, err := p.members.Find(ctx, chatID, uid)
	if err != nil {
		return nil, nil, err
	}
	return group, member, nil
}
