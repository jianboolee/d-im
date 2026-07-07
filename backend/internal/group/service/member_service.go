package service

import (
	"context"
	"errors"
	"fmt"

	chatRepo "d-im/internal/chat/repository"
	"d-im/internal/group/repository"
	"d-im/pkg/model"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MemberService 群成员用例服务。
type MemberService struct {
	chatRepo *chatRepo.ChatRepo
	groups   *repository.GroupRepo
	members  *repository.MemberRepo
	convMgr  *model.ConversationManager
}

// NewMemberService 创建成员服务。
func NewMemberService(chatRepo *chatRepo.ChatRepo, groups *repository.GroupRepo, members *repository.MemberRepo, convMgr *model.ConversationManager) *MemberService {
	return &MemberService{
		chatRepo: chatRepo,
		groups:   groups,
		members:  members,
		convMgr:  convMgr,
	}
}

// JoinGroup 自由加入公开群。
func (s *MemberService) JoinGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
	group, err := s.groups.FindActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if _, err := s.members.Find(ctx, chatID, uid); err == nil {
		return group, nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}
	if group.Settings.JoinMethod != model.JoinMethodFree || !group.Settings.IsPublic {
		return nil, ErrForbidden
	}
	if err := ensureCapacity(group, 1); err != nil {
		return nil, err
	}
	inserted, err := s.members.Add(ctx, &model.GroupMember{
		ChatID: chatID,
		UID:    uid,
		Role:   model.MemberRoleMember,
	})
	if err != nil {
		return nil, err
	}
	if inserted {
		group, err = s.groups.IncMemberCount(ctx, chatID, 1)
		if err != nil {
			return nil, err
		}
		if s.convMgr != nil {
			conv := &model.Conversation{UID: uid, ChatID: chatID, ChatType: types.ChatTypeGroup}
			if err := s.convMgr.CreateOrUpdate(ctx, conv); err != nil {
				return nil, err
			}
		}
	}
	return group, nil
}

// AddMembers 邀请成员入群。
func (s *MemberService) AddMembers(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Group, []string, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, nil, err
	}
	if !canManageMembers(group, operator) {
		return nil, nil, ErrForbidden
	}
	newUIDs := uniqueNonEmpty(uidList)
	adding := make([]string, 0, len(newUIDs))
	for _, uid := range newUIDs {
		if _, err := s.members.Find(ctx, chatID, uid); err == nil {
			continue
		} else if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, err
		}
		adding = append(adding, uid)
	}
	if err := ensureCapacity(group, len(adding)); err != nil {
		return nil, nil, err
	}
	for _, uid := range adding {
		if inserted, err := s.members.Add(ctx, &model.GroupMember{
			ChatID:    chatID,
			UID:       uid,
			InvitedBy: operatorUID,
			Role:      model.MemberRoleMember,
		}); err != nil {
			return nil, nil, err
		} else if inserted {
			group, err = s.groups.IncMemberCount(ctx, chatID, 1)
			if err != nil {
				return nil, nil, err
			}
			if s.convMgr != nil {
				conv := &model.Conversation{UID: uid, ChatID: chatID, ChatType: types.ChatTypeGroup}
				if err := s.convMgr.CreateOrUpdate(ctx, conv); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return group, adding, nil
}

// LeaveGroup 退出群。
func (s *MemberService) LeaveGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
	group, member, err := s.requireMember(ctx, chatID, uid)
	if err != nil {
		return nil, err
	}
	if member.Role == model.MemberRoleOwner && group.MemberCount > 1 {
		return nil, fmt.Errorf("%w: owner must transfer owner before leaving", ErrInvalid)
	}
	if group.MemberCount <= 1 {
		return s.DismissGroup(ctx, chatID, uid)
	}
	removed, err := s.members.Remove(ctx, chatID, uid)
	if err != nil {
		return nil, err
	}
	if removed {
		group, err = s.groups.IncMemberCount(ctx, chatID, -1)
		if err != nil {
			return nil, err
		}
	}
	if s.convMgr != nil {
		if err := s.convMgr.MarkLeft(ctx, uid, chatID); err != nil {
			return nil, err
		}
	}
	return group, nil
}

// KickMember 踢出群成员。
func (s *MemberService) KickMember(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	target, err := s.members.Find(ctx, chatID, targetUID)
	if err != nil {
		return nil, err
	}
	if targetUID == operatorUID || target.Role == model.MemberRoleOwner {
		return nil, ErrInvalid
	}
	if !canManageMembers(group, operator) {
		return nil, ErrForbidden
	}
	if target.Role == model.MemberRoleAdmin && operator.Role != model.MemberRoleOwner {
		return nil, ErrForbidden
	}
	removed, err := s.members.Remove(ctx, chatID, targetUID)
	if err != nil {
		return nil, err
	}
	if removed {
		group, err = s.groups.IncMemberCount(ctx, chatID, -1)
		if err != nil {
			return nil, err
		}
	}
	if s.convMgr != nil {
		if err := s.convMgr.MarkLeft(ctx, targetUID, chatID); err != nil {
			return nil, err
		}
	}
	return group, nil
}

// SetMemberRole 设置成员角色。
func (s *MemberService) SetMemberRole(ctx context.Context, chatID, operatorUID, targetUID string, role model.MemberRole) (*model.Group, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if operator.Role != model.MemberRoleOwner {
		return nil, ErrForbidden
	}
	if targetUID == group.OwnerUID {
		return nil, ErrInvalid
	}
	switch role {
	case model.MemberRoleAdmin, model.MemberRoleMember:
	default:
		return nil, ErrInvalid
	}
	if _, err := s.members.SetRole(ctx, chatID, targetUID, role); err != nil {
		return nil, err
	}
	return s.groups.FindActiveByChatID(ctx, chatID)
}

// TransferOwner 转让群主。
func (s *MemberService) TransferOwner(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if operator.Role != model.MemberRoleOwner {
		return nil, ErrForbidden
	}
	if targetUID == "" || targetUID == operatorUID {
		return nil, ErrInvalid
	}
	if _, err := s.members.Find(ctx, chatID, targetUID); err != nil {
		return nil, err
	}
	if _, err := s.members.SetRole(ctx, chatID, operatorUID, model.MemberRoleAdmin); err != nil {
		return nil, err
	}
	if _, err := s.members.SetRole(ctx, chatID, targetUID, model.MemberRoleOwner); err != nil {
		return nil, err
	}
	group, err = s.groups.UpdateFields(ctx, chatID, bson.M{"owner_uid": targetUID})
	if err != nil {
		return nil, err
	}
	return group, nil
}

// DismissGroup 解散群（由成员用例内部调用，仅群主）。
func (s *MemberService) DismissGroup(ctx context.Context, chatID, operatorUID string) (*model.Group, error) {
	group, member, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if member.Role != model.MemberRoleOwner {
		return nil, ErrForbidden
	}
	dismissed, err := s.groups.Dismiss(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if s.convMgr != nil {
		memberUIDs, err := s.members.ListUIDs(ctx, group.ChatID)
		if err != nil {
			return nil, err
		}
		for _, uid := range memberUIDs {
			if err := s.convMgr.MarkLeft(ctx, uid, chatID); err != nil {
				return nil, err
			}
		}
	}
	return dismissed, nil
}

// ListMembers 获取群成员列表。
func (s *MemberService) ListMembers(ctx context.Context, chatID string, limit, offset int64) ([]*model.GroupMember, error) {
	if _, err := s.groups.FindActiveByChatID(ctx, chatID); err != nil {
		return nil, err
	}
	return s.members.List(ctx, chatID, limit, offset)
}

// GetMember 获取单个成员。
func (s *MemberService) GetMember(ctx context.Context, chatID, uid string) (*model.GroupMember, error) {
	return s.members.Find(ctx, chatID, uid)
}

// GetMemberUIDs 获取群内所有成员 UID。
func (s *MemberService) GetMemberUIDs(ctx context.Context, chatID string) ([]string, error) {
	if _, err := s.groups.FindActiveByChatID(ctx, chatID); err != nil {
		return nil, err
	}
	return s.members.ListUIDs(ctx, chatID)
}

func (s *MemberService) requireMember(ctx context.Context, chatID, uid string) (*model.Group, *model.GroupMember, error) {
	group, err := s.groups.FindActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, nil, err
	}
	member, err := s.members.Find(ctx, chatID, uid)
	if err != nil {
		return nil, nil, err
	}
	return group, member, nil
}
