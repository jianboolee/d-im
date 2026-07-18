package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	chatSvc "d-im/internal/chat/service"
	"d-im/internal/group/repository"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const defaultMaxMembers = 100

type GroupService struct {
	db              *mongo.Database
	chats           *chatSvc.ChatService
	groups          *repository.GroupRepo
	members         *repository.MemberRepo
	conversations   conversationProjectionWriter
	avatarGenerator groupAvatarGenerator
	users           UserProfileReader
	eventPublisher  *EventPublisher
	maxMembers      int
}

type groupAvatarGenerator interface {
	GenerateAndStore(ctx context.Context, chatID string, memberUIDs []string) (string, error)
}

func NewGroupService(db *mongo.Database, chats *chatSvc.ChatService, groupRepo *repository.GroupRepo, memberRepo *repository.MemberRepo, conversations conversationProjectionWriter) *GroupService {
	return &GroupService{
		db:            db,
		chats:         chats,
		groups:        groupRepo,
		members:       memberRepo,
		conversations: conversations,
	}
}

func (s *GroupService) SetAvatarGenerator(generator groupAvatarGenerator) {
	s.avatarGenerator = generator
}

func (s *GroupService) SetUserProfileReader(users UserProfileReader) {
	s.users = users
}

func (s *GroupService) SetMaxMembers(maxMembers int) {
	if maxMembers > 0 {
		s.maxMembers = maxMembers
	}
}

func (s *GroupService) SetEventPublisher(publisher *EventPublisher) {
	s.eventPublisher = publisher
}

func (s *GroupService) publishEvent(ctx context.Context, event GroupSystemEvent) {
	if s.eventPublisher != nil {
		s.eventPublisher.Publish(ctx, event)
	}
}

func (s *GroupService) effectiveMaxMembers() int {
	if s != nil && s.maxMembers > 0 {
		return s.maxMembers
	}
	return defaultMaxMembers
}

func (s *GroupService) ensureCapacity(group *model.Group, adding int) error {
	return ensureCapacity(group, adding, s.effectiveMaxMembers())
}

func (s *GroupService) currentChatLastSeq(ctx context.Context, chatID string) (int64, error) {
	if s.chats == nil {
		return 0, nil
	}
	chat, err := s.chats.GetChat(ctx, chatID)
	if err != nil {
		return 0, err
	}
	return chat.LastSeq, nil
}

// CreateGroup 创建群（事务包裹 chats + groups + group_members + conversations）。
func (s *GroupService) CreateGroup(ctx context.Context, name, ownerUID string, memberUIDs []string) (*model.Group, error) {
	name = strings.TrimSpace(name)
	if ownerUID == "" {
		return nil, ErrInvalid
	}
	allMembers := uniqueNonEmpty(append([]string{ownerUID}, memberUIDs...))
	if len(allMembers) == 0 {
		allMembers = []string{ownerUID}
	}
	if name == "" {
		name = s.defaultGroupName(ctx, allMembers)
	}
	maxMembers := s.effectiveMaxMembers()

	var result *model.Group
	err := mongodb.WithRequiredTransaction(ctx, s.db, func(ctx context.Context) error {
		group, err := s.createGroupInternal(ctx, name, ownerUID, allMembers, maxMembers)
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
		EventType:   EventTypeGroupCreated,
		OperatorUID: ownerUID,
		GroupID:     result.ChatID,
		GroupName:   result.Name,
	})
	s.GenerateGroupAvatarAsync(result.ChatID)
	return result, nil
}

func (s *GroupService) defaultGroupName(ctx context.Context, memberUIDs []string) string {
	names := make([]string, 0, 3)
	for _, uid := range firstDefaultGroupNameMembers(memberUIDs) {
		if s.users == nil {
			continue
		}
		user, err := s.users.FindByID(ctx, uid)
		if err != nil || user == nil {
			continue
		}
		if name := strings.TrimSpace(user.Nickname); name != "" {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "群聊"
	}
	return strings.Join(names, "、")
}

// createGroupInternal 创建群的核心逻辑，用于事务内部。
func (s *GroupService) createGroupInternal(ctx context.Context, name, ownerUID string, allMembers []string, maxMembers int) (*model.Group, error) {
	chat, err := s.chats.CreateGroupChat(ctx, ownerUID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	group := &model.Group{
		GroupID:     chat.ChatID,
		ChatID:      chat.ChatID,
		Name:        name,
		OwnerUID:    ownerUID,
		MemberCount: len(allMembers),
		MaxMembers:  maxMembers,
		Settings:    model.DefaultGroupSettings(),
		Status:      model.GroupStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.groups.Create(ctx, group); err != nil {
		return nil, err
	}

	memberDocs := make([]*model.GroupMember, 0, len(allMembers))
	for _, uid := range allMembers {
		role := model.MemberRoleMember
		if uid == ownerUID {
			role = model.MemberRoleOwner
		}
		memberDocs = append(memberDocs, &model.GroupMember{
			ChatID:    chat.ChatID,
			UID:       uid,
			InvitedBy: ownerUID,
			Role:      role,
			JoinedAt:  now,
			UpdatedAt: now,
		})
	}
	if err := s.members.CreateMany(ctx, memberDocs); err != nil {
		return nil, err
	}

	if s.conversations != nil {
		if err := s.conversations.EnsureUsers(ctx, allMembers, chat); err != nil {
			return nil, err
		}
	}

	return group, nil
}

func (s *GroupService) GenerateGroupAvatarAsync(chatID string) {
	if s == nil || s.avatarGenerator == nil || strings.TrimSpace(chatID) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		group, err := s.groups.FindActiveByChatID(ctx, chatID)
		if err != nil {
			log.Printf("[group] load group for avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if !shouldUpdateGeneratedAvatar(group.Avatar, group.ChatID) {
			return
		}
		memberUIDs, err := s.members.ListUIDs(ctx, group.ChatID)
		if err != nil {
			log.Printf("[group] load members for avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		avatarURL, err := s.avatarGenerator.GenerateAndStore(ctx, group.ChatID, memberUIDs)
		if err != nil {
			log.Printf("[group] generate group avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if avatarURL == "" {
			return
		}
		updated, err := s.groups.UpdateAvatar(ctx, group.ChatID, avatarURL)
		if err != nil {
			log.Printf("[group] update group avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if updated != nil {
			s.publishEvent(ctx, GroupSystemEvent{
				EventType: EventTypeAvatarUpdated,
				GroupID:   updated.ChatID,
				GroupName: updated.Name,
			})
		}
	}()
}

func (s *GroupService) ListGroupsForMember(ctx context.Context, uid string, limit, offset int64) ([]*model.Group, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, mongo.ErrNoDocuments
	}
	chatIDs, err := s.members.ListChatIDsByUID(ctx, uid, limit, offset)
	if err != nil {
		return nil, err
	}
	groupByChatID, err := s.groups.ListByChatIDs(ctx, chatIDs)
	if err != nil {
		return nil, err
	}
	groups := make([]*model.Group, 0, len(chatIDs))
	for _, chatID := range chatIDs {
		if group := groupByChatID[chatID]; group != nil {
			groups = append(groups, group)
		}
	}
	return groups, nil
}

func (s *GroupService) GetGroupForMember(ctx context.Context, chatID, uid string) (*model.Group, error) {
	group, err := s.groups.FindActiveByChatID(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if _, err := s.members.Find(ctx, chatID, uid); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *GroupService) GetGroup(ctx context.Context, chatID string) (*model.Group, error) {
	return s.groups.FindActiveByChatID(ctx, chatID)
}

func (s *GroupService) GetMember(ctx context.Context, chatID, uid string) (*model.GroupMember, error) {
	return s.members.Find(ctx, chatID, uid)
}

func (s *GroupService) ListMembers(ctx context.Context, chatID string, limit, offset int64) ([]*model.GroupMember, error) {
	if _, err := s.groups.FindActiveByChatID(ctx, chatID); err != nil {
		return nil, err
	}
	return s.members.List(ctx, chatID, limit, offset)
}

func (s *GroupService) GetMemberUIDs(ctx context.Context, chatID string) ([]string, error) {
	if _, err := s.groups.FindActiveByChatID(ctx, chatID); err != nil {
		return nil, err
	}
	return s.members.ListUIDs(ctx, chatID)
}

func (s *GroupService) AddMembers(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Group, []string, error) {
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

func (s *GroupService) JoinGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
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

func (s *GroupService) LeaveGroup(ctx context.Context, chatID, uid string) (*model.Group, error) {
	group, member, err := s.requireMember(ctx, chatID, uid)
	if err != nil {
		return nil, err
	}
	if group.MemberCount <= 1 {
		return s.DismissGroup(ctx, chatID, uid)
	}
	if member.Role == model.MemberRoleOwner {
		nextOwner, err := s.members.FirstJoinedExcept(ctx, chatID, uid)
		if err != nil {
			return nil, err
		}
		if _, err := s.members.SetRole(ctx, chatID, nextOwner.UID, model.MemberRoleOwner); err != nil {
			return nil, err
		}
		group, err = s.groups.UpdateFields(ctx, chatID, bson.M{"owner_uid": nextOwner.UID})
		if err != nil {
			return nil, err
		}
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
	if s.conversations != nil {
		if err := s.conversations.UserLeft(ctx, uid, chatID); err != nil {
			return nil, err
		}
	}
	return group, nil
}

func (s *GroupService) KickMember(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
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

func (s *GroupService) UpdateInfo(ctx context.Context, chatID, operatorUID string, info UpdateGroupInfo) (*model.Group, error) {
	group, operator, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !canEditGroupInfo(operator) {
		return nil, ErrForbidden
	}
	fields := bson.M{}
	var beforeName string
	var afterName string
	if info.Name != nil {
		name := strings.TrimSpace(*info.Name)
		if name == "" {
			return nil, fmt.Errorf("group name is required")
		}
		fields["name"] = name
		beforeName = group.Name
		afterName = name
	}
	if info.Avatar != nil {
		fields["avatar"] = strings.TrimSpace(*info.Avatar)
	}
	if info.Description != nil {
		fields["description"] = strings.TrimSpace(*info.Description)
	}
	if len(fields) == 0 {
		return s.groups.FindActiveByChatID(ctx, chatID)
	}
	result, err := s.groups.UpdateFields(ctx, chatID, fields)
	if err != nil {
		return nil, err
	}
	eventType := EventTypeGroupInfoUpdated
	if info.Name == nil && info.Avatar != nil {
		eventType = EventTypeAvatarUpdated
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   eventType,
		OperatorUID: operatorUID,
		GroupID:     chatID,
		GroupName:   result.Name,
		BeforeValue: beforeName,
		AfterValue:  afterName,
	})
	return result, nil
}

func (s *GroupService) UpdateName(ctx context.Context, chatID, operatorUID, name string) (*model.Group, error) {
	return s.UpdateInfo(ctx, chatID, operatorUID, UpdateGroupInfo{Name: &name})
}

func (s *GroupService) UpdateSettings(ctx context.Context, chatID, operatorUID string, settings model.GroupSettings) (*model.Group, error) {
	group, member, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !canUpdateGroupInfo(member) {
		return nil, ErrForbidden
	}
	if settings.JoinMethod == "" {
		settings.JoinMethod = group.Settings.JoinMethod
	}
	if settings.JoinMethod == "" {
		settings.JoinMethod = model.JoinMethodInvite
	}
	result, err := s.groups.UpdateFields(ctx, chatID, bson.M{"settings": settings})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *GroupService) SetAnnouncement(ctx context.Context, chatID, operatorUID, announcement string) (*model.Group, error) {
	_, member, err := s.requireMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !canEditGroupInfo(member) {
		return nil, ErrForbidden
	}
	result, err := s.groups.UpdateFields(ctx, chatID, bson.M{"announcement": strings.TrimSpace(announcement)})
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, GroupSystemEvent{
		EventType:   EventTypeAnnouncementUpdated,
		OperatorUID: operatorUID,
		GroupID:     chatID,
		GroupName:   result.Name,
	})
	return result, nil
}

func (s *GroupService) SetMemberRole(ctx context.Context, chatID, operatorUID, targetUID string, role model.MemberRole) (*model.Group, error) {
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

func (s *GroupService) TransferOwner(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Group, error) {
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

func (s *GroupService) DismissGroup(ctx context.Context, chatID, operatorUID string) (*model.Group, error) {
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
		memberUIDs, err := s.members.ListUIDs(ctx, group.ChatID)
		if err != nil {
			return nil, err
		}
		for _, uid := range memberUIDs {
			if err := s.conversations.UserLeft(ctx, uid, chatID); err != nil {
				return nil, err
			}
		}
	}
	return dismissed, nil
}

func (s *GroupService) CheckPermission(ctx context.Context, chatID, uid, action string) (bool, string, error) {
	group, member, err := s.requireMember(ctx, chatID, uid)
	if err != nil {
		return false, "not_group_member", err
	}
	if group.Settings.IsMutedAll && !isPrivileged(member) {
		return false, "group_muted_all", nil
	}
	if isMemberMuted(group, member, time.Now()) {
		return false, "member_muted", nil
	}
	switch action {
	case "invite_member":
		if !canInviteMembers(group, member) {
			return false, "permission_denied", nil
		}
	case "kick_member":
		if !canUpdateGroupInfo(member) {
			return false, "permission_denied", nil
		}
	case "update_group_info", "set_announcement":
		if !canEditGroupInfo(member) {
			return false, "permission_denied", nil
		}
	case "update_settings":
		if !canUpdateGroupInfo(member) {
			return false, "permission_denied", nil
		}
	case "dismiss_group", "transfer_owner", "set_member_role":
		if member.Role != model.MemberRoleOwner {
			return false, "owner_required", nil
		}
	}
	return true, "", nil
}

func (s *GroupService) requireMember(ctx context.Context, chatID, uid string) (*model.Group, *model.GroupMember, error) {
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

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}

func canUpdateGroupInfo(member *model.GroupMember) bool {
	return member != nil && (member.Role == model.MemberRoleOwner || member.Role == model.MemberRoleAdmin)
}

func canEditGroupInfo(member *model.GroupMember) bool {
	return member != nil
}

func canManageMembers(_ *model.Group, member *model.GroupMember) bool {
	return canUpdateGroupInfo(member)
}

func canInviteMembers(group *model.Group, member *model.GroupMember) bool {
	if canUpdateGroupInfo(member) {
		return true
	}
	return member != nil && allowsMemberInvite(group)
}

func allowsMemberInvite(group *model.Group) bool {
	return group == nil || group.Settings.AllowMemberInvite == nil || *group.Settings.AllowMemberInvite
}

func isPrivileged(member *model.GroupMember) bool {
	return member != nil && (member.Role == model.MemberRoleOwner || member.Role == model.MemberRoleAdmin)
}

func isMemberMuted(group *model.Group, member *model.GroupMember, now time.Time) bool {
	if member == nil {
		return false
	}
	if member.IsMuted {
		return member.MutedUntil == nil || member.MutedUntil.After(now)
	}
	return group != nil && containsString(group.Settings.MutedMembers, member.UID)
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func ensureCapacity(group *model.Group, adding int, fallbackMaxMembers int) error {
	if group == nil {
		return mongo.ErrNoDocuments
	}
	maxMembers := group.MaxMembers
	if maxMembers <= 0 {
		maxMembers = fallbackMaxMembers
	}
	if maxMembers <= 0 {
		maxMembers = defaultMaxMembers
	}
	if group.MemberCount+adding > maxMembers {
		return ErrGroupFull
	}
	return nil
}
