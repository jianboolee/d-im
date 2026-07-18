package service

import (
	"context"
	"errors"
	"log"
	"time"

	chatRepo "d-im/internal/chat/repository"
	"d-im/internal/group/repository"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// MemberService 群成员用例服务。
type MemberService struct {
	db              *mongo.Database
	chatRepo        *chatRepo.ChatRepo
	groups          *repository.GroupRepo
	members         *repository.MemberRepo
	conversations   conversationProjectionWriter
	avatarGenerator groupAvatarGenerator
	eventPublisher  *EventPublisher
	maxMembers      int
}

// NewMemberService 创建成员服务。
func NewMemberService(db *mongo.Database, chatRepo *chatRepo.ChatRepo, groups *repository.GroupRepo, members *repository.MemberRepo, conversations conversationProjectionWriter) *MemberService {
	return &MemberService{
		db:            db,
		chatRepo:      chatRepo,
		groups:        groups,
		members:       members,
		conversations: conversations,
	}
}

// SetEventPublisher 注入事件发布器。
func (s *MemberService) SetEventPublisher(publisher *EventPublisher) {
	s.eventPublisher = publisher
}

func (s *MemberService) SetAvatarGenerator(generator groupAvatarGenerator) {
	s.avatarGenerator = generator
}

func (s *MemberService) SetMaxMembers(maxMembers int) {
	if maxMembers > 0 {
		s.maxMembers = maxMembers
	}
}

func (s *MemberService) publishEvent(ctx context.Context, event GroupSystemEvent) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(ctx, event)
	}
}

func (s *MemberService) effectiveMaxMembers() int {
	if s != nil && s.maxMembers > 0 {
		return s.maxMembers
	}
	return defaultMaxMembers
}

func (s *MemberService) ensureCapacity(group *model.Group, adding int) error {
	return ensureCapacity(group, adding, s.effectiveMaxMembers())
}

func (s *MemberService) currentChatLastSeq(ctx context.Context, chatID string) (int64, error) {
	if s.chatRepo == nil {
		return 0, nil
	}
	chat, err := s.chatRepo.FindByChatID(ctx, chatID)
	if err != nil {
		return 0, err
	}
	return chat.LastSeq, nil
}

func (s *MemberService) maybeGenerateGroupAvatarAsync(chatID string, beforeMemberUIDs []string) {
	if s == nil || s.avatarGenerator == nil || chatID == "" {
		return
	}
	afterMemberUIDs, err := s.members.ListUIDs(context.Background(), chatID)
	if err != nil {
		log.Printf("[group] load members for avatar refresh failed: chat_id=%s err=%v", chatID, err)
		return
	}
	if !avatarAffectingMembersChanged(beforeMemberUIDs, afterMemberUIDs) {
		return
	}
	go func(memberUIDs []string) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		group, err := s.groups.FindActiveByChatID(ctx, chatID)
		if err != nil {
			log.Printf("[group] load group for avatar refresh failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if !shouldUpdateGeneratedAvatar(group.Avatar, group.ChatID) {
			return
		}
		avatarURL, err := s.avatarGenerator.GenerateAndStore(ctx, group.ChatID, memberUIDs)
		if err != nil {
			log.Printf("[group] refresh group avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if avatarURL == "" {
			return
		}
		updated, err := s.groups.UpdateAvatar(ctx, group.ChatID, avatarURL)
		if err != nil {
			log.Printf("[group] update refreshed group avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		s.publishEvent(ctx, GroupSystemEvent{
			EventType: EventTypeAvatarUpdated,
			GroupID:   updated.ChatID,
			GroupName: updated.Name,
		})
	}(afterMemberUIDs)
}

// JoinGroup 自由加入公开群（事务包裹）。
func (s *MemberService) JoinGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
	var result *model.Group
	var beforeMemberUIDs []string
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		uids, err := s.members.ListUIDs(ctx, chatID)
		if err != nil {
			return err
		}
		beforeMemberUIDs = uids
		group, err := s.joinGroupInternal(ctx, chatID, uid)
		if err != nil {
			return err
		}
		result = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeMemberJoined,
		OperatorUID: uid,
		TargetUIDs:  []string{uid},
		GroupID:     result.ChatID,
		GroupName:   result.Name,
	})
	s.maybeGenerateGroupAvatarAsync(result.ChatID, beforeMemberUIDs)
	return result, nil
}

func (s *MemberService) joinGroupInternal(ctx context.Context, chatID, uid string) (*model.Group, error) {
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
	if err := s.ensureCapacity(group, 1); err != nil {
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
		if s.conversations != nil {
			lastReadSeq, err := s.currentChatLastSeq(ctx, chatID)
			if err != nil {
				return nil, err
			}
			// project the membership fact into the user conversation view
			if err := s.conversations.UserJoined(ctx, uid, chatID, types.ChatTypeGroup, lastReadSeq); err != nil {
				return nil, err
			}
		}
	}
	return group, nil
}

// AddMembers 邀请成员入群（事务包裹）。
func (s *MemberService) AddMembers(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Group, []string, error) {
	var result *model.Group
	var addedUIDs []string
	var beforeMemberUIDs []string
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		uids, err := s.members.ListUIDs(ctx, chatID)
		if err != nil {
			return err
		}
		beforeMemberUIDs = uids
		group, added, err := s.addMembersInternal(ctx, chatID, operatorUID, uidList)
		if err != nil {
			return err
		}
		result = group
		addedUIDs = added
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	if len(addedUIDs) > 0 {
		s.publishEvent(ctx, GroupSystemEvent{
			EventType:   EventTypeMembersInvited,
			OperatorUID: operatorUID,
			TargetUIDs:  addedUIDs,
			GroupID:     result.ChatID,
			GroupName:   result.Name,
		})
		s.maybeGenerateGroupAvatarAsync(result.ChatID, beforeMemberUIDs)
	}
	return result, addedUIDs, nil
}

func (s *MemberService) addMembersInternal(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Group, []string, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, nil, err
	}
	if !canInviteMembers(group, operator) {
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
	if err := s.ensureCapacity(group, len(adding)); err != nil {
		return nil, nil, err
	}
	lastReadSeq := int64(0)
	if s.conversations != nil && len(adding) > 0 {
		var err error
		lastReadSeq, err = s.currentChatLastSeq(ctx, chatID)
		if err != nil {
			return nil, nil, err
		}
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
			if s.conversations != nil {
				// project the membership fact into the user conversation view
				if err := s.conversations.UserJoined(ctx, uid, chatID, types.ChatTypeGroup, lastReadSeq); err != nil {
					return nil, nil, err
				}
			}
		}
	}
	return group, adding, nil
}

// LeaveGroup 退出群（事务包裹）。
func (s *MemberService) LeaveGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
	var result *model.Group
	var beforeMemberUIDs []string
	var ownerTransferredTo string
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		uids, err := s.members.ListUIDs(ctx, chatID)
		if err != nil {
			return err
		}
		beforeMemberUIDs = uids
		group, transferredTo, err := s.leaveGroupInternal(ctx, chatID, uid)
		if err != nil {
			return err
		}
		result = group
		ownerTransferredTo = transferredTo
		return nil
	})
	if err != nil {
		return nil, err
	}
	if ownerTransferredTo != "" {
		s.publishEvent(ctx, GroupSystemEvent{
			EventType:   EventTypeOwnerTransferred,
			OperatorUID: uid,
			TargetUIDs:  []string{ownerTransferredTo},
			GroupID:     result.ChatID,
			GroupName:   result.Name,
			BeforeValue: uid,
			AfterValue:  ownerTransferredTo,
		})
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeMemberLeft,
		OperatorUID: uid,
		TargetUIDs:  []string{uid},
		GroupID:     result.ChatID,
		GroupName:   result.Name,
	})
	s.maybeGenerateGroupAvatarAsync(result.ChatID, beforeMemberUIDs)
	return result, nil
}

func (s *MemberService) leaveGroupInternal(ctx context.Context, chatID, uid string) (*model.Group, string, error) {
	group, member, err := s.requireMember(ctx, chatID, uid)
	if err != nil {
		return nil, "", err
	}
	if group.MemberCount <= 1 {
		dismissed, err := s.dismissGroupInternal(ctx, chatID, uid)
		return dismissed, "", err
	}
	ownerTransferredTo := ""
	if member.Role == model.MemberRoleOwner {
		nextOwner, err := s.members.FirstJoinedExcept(ctx, chatID, uid)
		if err != nil {
			return nil, "", err
		}
		if _, err := s.members.SetRole(ctx, chatID, nextOwner.UID, model.MemberRoleOwner); err != nil {
			return nil, "", err
		}
		group, err = s.groups.UpdateFields(ctx, chatID, bson.M{"owner_uid": nextOwner.UID})
		if err != nil {
			return nil, "", err
		}
		ownerTransferredTo = nextOwner.UID
	}
	removed, err := s.members.Remove(ctx, chatID, uid)
	if err != nil {
		return nil, "", err
	}
	if removed {
		group, err = s.groups.IncMemberCount(ctx, chatID, -1)
		if err != nil {
			return nil, "", err
		}
	}
	if s.conversations != nil {
		if err := s.conversations.UserLeft(ctx, uid, chatID); err != nil {
			return nil, "", err
		}
	}
	return group, ownerTransferredTo, nil
}

// KickMember 踢出群成员（事务包裹）。
func (s *MemberService) KickMember(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
	var result *model.Group
	var beforeMemberUIDs []string
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		uids, err := s.members.ListUIDs(ctx, chatID)
		if err != nil {
			return err
		}
		beforeMemberUIDs = uids
		group, err := s.kickMemberInternal(ctx, chatID, operatorUID, targetUID)
		if err != nil {
			return err
		}
		result = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeMemberKicked,
		OperatorUID: operatorUID,
		TargetUIDs:  []string{targetUID},
		GroupID:     result.ChatID,
		GroupName:   result.Name,
	})
	s.maybeGenerateGroupAvatarAsync(result.ChatID, beforeMemberUIDs)
	return result, nil
}

func (s *MemberService) kickMemberInternal(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
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
	if s.conversations != nil {
		if err := s.conversations.UserLeft(ctx, targetUID, chatID); err != nil {
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
	result, err := s.groups.FindActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeMemberRoleChanged,
		OperatorUID: operatorUID,
		TargetUIDs:  []string{targetUID},
		GroupID:     result.ChatID,
		GroupName:   result.Name,
		AfterValue:  string(role),
	})
	return result, nil
}

// TransferOwner 转让群主（事务包裹）。
func (s *MemberService) TransferOwner(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
	var result *model.Group
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		group, err := s.transferOwnerInternal(ctx, chatID, operatorUID, targetUID)
		if err != nil {
			return err
		}
		result = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeOwnerTransferred,
		OperatorUID: operatorUID,
		TargetUIDs:  []string{targetUID},
		GroupID:     result.ChatID,
		GroupName:   result.Name,
		BeforeValue: operatorUID,
		AfterValue:  targetUID,
	})
	return result, nil
}

func (s *MemberService) transferOwnerInternal(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
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

// DismissGroup 解散群（事务包裹）。
func (s *MemberService) DismissGroup(ctx context.Context, chatID, operatorUID string) (*model.Group, error) {
	var result *model.Group
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		group, err := s.dismissGroupInternal(ctx, chatID, operatorUID)
		if err != nil {
			return err
		}
		result = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeGroupDismissed,
		OperatorUID: operatorUID,
		GroupID:     result.ChatID,
		GroupName:   result.Name,
	})
	return result, nil
}

func (s *MemberService) dismissGroupInternal(ctx context.Context, chatID, operatorUID string) (*model.Group, error) {
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
	if s.conversations != nil {
		memberUIDs, uidsErr := s.members.ListUIDs(ctx, group.ChatID)
		if uidsErr != nil {
			return nil, uidsErr
		}
		for _, uid := range memberUIDs {
			if err := s.conversations.UserLeft(ctx, uid, chatID); err != nil {
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

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
