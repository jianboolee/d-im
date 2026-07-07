package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrForbidden = errors.New("forbidden")
	ErrInvalid   = errors.New("invalid group operation")
	ErrGroupFull = errors.New("group is full")
)

const defaultMaxMembers = 500

type UpdateGroupInfo struct {
	Name        *string
	Avatar      *string
	Description *string
}

// GroupService 群组服务
type GroupService struct {
	chatColl        *mongo.Collection
	convMgr         *model.ConversationManager
	avatarGenerator groupAvatarGenerator
}

// NewGroupService 创建群组服务
func NewGroupService(chatColl *mongo.Collection, convMgr *model.ConversationManager) *GroupService {
	return &GroupService{
		chatColl: chatColl,
		convMgr:  convMgr,
	}
}

type groupAvatarGenerator interface {
	GenerateAndStore(ctx context.Context, chatID string, memberUIDs []string) (string, error)
}

func (s *GroupService) SetAvatarGenerator(generator groupAvatarGenerator) {
	s.avatarGenerator = generator
}

// CreateGroup 创建群聊
func (s *GroupService) CreateGroup(ctx context.Context, name, ownerUID string, memberUIDs []string) (*model.Chat, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	chat, err := model.CreateGroupChat(ctx, s.chatColl, name, ownerUID, memberUIDs)
	if err != nil {
		return nil, err
	}

	// 为所有成员创建会话视图
	if err := s.convMgr.BatchCreate(ctx, chat.Members, chat); err != nil {
		return nil, err
	}

	s.GenerateGroupAvatarAsync(chat.ChatID)

	return chat, nil
}

// GenerateGroupAvatarAsync 异步生成并回写群宫格头像。
func (s *GroupService) GenerateGroupAvatarAsync(chatID string) {
	if s == nil || s.avatarGenerator == nil || strings.TrimSpace(chatID) == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
		if err != nil {
			log.Printf("[group] load chat for avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if !isActiveGroup(chat) || strings.TrimSpace(chat.Avatar) != "" {
			return
		}

		avatarURL, err := s.avatarGenerator.GenerateAndStore(ctx, chat.ChatID, chat.Members)
		if err != nil {
			log.Printf("[group] generate group avatar failed: chat_id=%s err=%v", chatID, err)
			return
		}
		if avatarURL == "" {
			return
		}
		if _, err := model.UpdateGroupChatAvatarIfEmpty(ctx, s.chatColl, chat.ChatID, avatarURL); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			log.Printf("[group] update group avatar failed: chat_id=%s err=%v", chatID, err)
		}
	}()
}

// ListGroupsForMember 查询当前用户加入的群列表。
func (s *GroupService) ListGroupsForMember(ctx context.Context, uid string, limit, offset int64) ([]*model.Chat, error) {
	if strings.TrimSpace(uid) == "" {
		return nil, mongo.ErrNoDocuments
	}
	return model.ListGroupChatsByMember(ctx, s.chatColl, uid, limit, offset)
}

// GetGroupForMember 查询群，并校验当前用户仍在群内。
func (s *GroupService) GetGroupForMember(ctx context.Context, chatID, uid string) (*model.Chat, error) {
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if chat.ChatType != types.ChatTypeGroup {
		return nil, mongo.ErrNoDocuments
	}
	if chat.Status == model.GroupStatusDismissed {
		return nil, mongo.ErrNoDocuments
	}
	if !containsString(chat.Members, uid) {
		return nil, mongo.ErrNoDocuments
	}
	return chat, nil
}

// AddMember 添加群成员。
func (s *GroupService) AddMember(ctx context.Context, chatID, uid string) error {
	if err := model.AddChatMember(ctx, s.chatColl, chatID, uid); err != nil {
		return err
	}

	// 为新成员创建会话视图
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return err
	}

	conv := &model.Conversation{
		UID:      uid,
		ChatID:   chatID,
		ChatType: chat.ChatType,
	}
	return s.convMgr.CreateOrUpdate(ctx, conv)
}

// AddMembers 批量邀请群成员。
func (s *GroupService) AddMembers(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Chat, error) {
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if !isActiveGroup(chat) {
		return nil, mongo.ErrNoDocuments
	}
	if !canManageMembers(chat, operatorUID) {
		return nil, ErrForbidden
	}
	newUIDs := uniqueNonEmpty(uidList)
	adding := 0
	for _, uid := range newUIDs {
		if !containsString(chat.Members, uid) {
			adding++
		}
	}
	if err := ensureCapacity(chat, adding); err != nil {
		return nil, err
	}
	for _, uid := range newUIDs {
		if err := s.AddMember(ctx, chatID, uid); err != nil {
			return nil, err
		}
	}
	return model.FindChatByID(ctx, s.chatColl, chatID)
}

// JoinGroup 按群设置申请或直接加入群。
func (s *GroupService) JoinGroup(ctx context.Context, chatID, uid string) (*model.Chat, error) {
	chat, err := model.FindChatByID(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if !isActiveGroup(chat) {
		return nil, mongo.ErrNoDocuments
	}
	if containsString(chat.Members, uid) {
		return chat, nil
	}
	if chat.Settings.JoinMethod != model.JoinMethodFree || !chat.Settings.IsPublic {
		return nil, ErrForbidden
	}
	if err := ensureCapacity(chat, 1); err != nil {
		return nil, err
	}
	if err := s.AddMember(ctx, chatID, uid); err != nil {
		return nil, err
	}
	return model.FindChatByID(ctx, s.chatColl, chatID)
}

// LeaveGroup 当前用户主动退出群。
func (s *GroupService) LeaveGroup(ctx context.Context, chatID, uid string) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, uid)
	if err != nil {
		return nil, err
	}
	if chat.OwnerUID == uid && len(chat.Members) > 1 {
		return nil, fmt.Errorf("%w: owner must transfer owner before leaving", ErrInvalid)
	}
	if len(chat.Members) <= 1 {
		return s.DismissGroup(ctx, chatID, uid)
	}
	if err := model.RemoveChatMember(ctx, s.chatColl, chatID, uid); err != nil {
		return nil, err
	}
	if s.convMgr != nil {
		if err := s.convMgr.MarkLeft(ctx, uid, chatID); err != nil {
			return nil, err
		}
	}
	return model.FindChatByID(ctx, s.chatColl, chatID)
}

// RemoveMember 保留旧调用语义，等价于当前用户退出。
func (s *GroupService) RemoveMember(ctx context.Context, chatID, uid string) error {
	_, err := s.LeaveGroup(ctx, chatID, uid)
	return err
}

// KickMember 踢出群成员。
func (s *GroupService) KickMember(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !containsString(chat.Members, targetUID) {
		return nil, mongo.ErrNoDocuments
	}
	if targetUID == chat.OwnerUID || targetUID == operatorUID {
		return nil, ErrInvalid
	}
	if !canManageMembers(chat, operatorUID) {
		return nil, ErrForbidden
	}
	if isAdmin(chat, targetUID) && chat.OwnerUID != operatorUID {
		return nil, ErrForbidden
	}
	if err := model.RemoveChatMember(ctx, s.chatColl, chatID, targetUID); err != nil {
		return nil, err
	}
	if s.convMgr != nil {
		if err := s.convMgr.MarkLeft(ctx, targetUID, chatID); err != nil {
			return nil, err
		}
	}
	return model.FindChatByID(ctx, s.chatColl, chatID)
}

// GetMembers 获取群成员列表
func (s *GroupService) GetMembers(ctx context.Context, chatID string) ([]string, error) {
	return model.GetChatMembers(ctx, s.chatColl, chatID)
}

// UpdateInfo 修改群资料。
func (s *GroupService) UpdateInfo(ctx context.Context, chatID, operatorUID string, info UpdateGroupInfo) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !canUpdateGroupInfo(chat, operatorUID) {
		return nil, ErrForbidden
	}
	fields := bson.M{}
	if info.Name != nil {
		name := strings.TrimSpace(*info.Name)
		if name == "" {
			return nil, fmt.Errorf("group name is required")
		}
		fields["name"] = name
	}
	if info.Avatar != nil {
		fields["avatar"] = strings.TrimSpace(*info.Avatar)
	}
	if info.Description != nil {
		fields["description"] = strings.TrimSpace(*info.Description)
	}
	if len(fields) == 0 {
		return chat, nil
	}
	return model.UpdateGroupChatFields(ctx, s.chatColl, chatID, fields)
}

// UpdateName 修改群名称。
func (s *GroupService) UpdateName(ctx context.Context, chatID, operatorUID, name string) (*model.Chat, error) {
	return s.UpdateInfo(ctx, chatID, operatorUID, UpdateGroupInfo{Name: &name})
}

// UpdateSettings 修改群设置。
func (s *GroupService) UpdateSettings(ctx context.Context, chatID, operatorUID string, settings model.GroupSettings) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !isOwner(chat, operatorUID) {
		return nil, ErrForbidden
	}
	if settings.JoinMethod == "" {
		settings.JoinMethod = chat.Settings.JoinMethod
	}
	if settings.JoinMethod == "" {
		settings.JoinMethod = model.JoinMethodInvite
	}
	return model.UpdateGroupChatFields(ctx, s.chatColl, chatID, bson.M{"settings": settings})
}

// SetAnnouncement 设置或清空群公告。
func (s *GroupService) SetAnnouncement(ctx context.Context, chatID, operatorUID, announcement string) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !canUpdateGroupInfo(chat, operatorUID) {
		return nil, ErrForbidden
	}
	return model.UpdateGroupChatFields(ctx, s.chatColl, chatID, bson.M{"announcement": strings.TrimSpace(announcement)})
}

// SetMemberRole 设置管理员或普通成员角色。
func (s *GroupService) SetMemberRole(ctx context.Context, chatID, operatorUID, targetUID string, role model.MemberRole) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !isOwner(chat, operatorUID) {
		return nil, ErrForbidden
	}
	if targetUID == chat.OwnerUID || !containsString(chat.Members, targetUID) {
		return nil, ErrInvalid
	}
	admins := removeString(chat.Admins, targetUID)
	switch role {
	case model.MemberRoleAdmin:
		admins = append(admins, targetUID)
	case model.MemberRoleMember:
	default:
		return nil, ErrInvalid
	}
	return model.UpdateGroupChatFields(ctx, s.chatColl, chatID, bson.M{"admins": uniqueNonEmpty(admins)})
}

// TransferOwner 转让群主。
func (s *GroupService) TransferOwner(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !isOwner(chat, operatorUID) {
		return nil, ErrForbidden
	}
	if targetUID == "" || !containsString(chat.Members, targetUID) || targetUID == operatorUID {
		return nil, ErrInvalid
	}
	admins := removeString(chat.Admins, targetUID)
	admins = append(admins, operatorUID)
	return model.UpdateGroupChatFields(ctx, s.chatColl, chatID, bson.M{
		"owner_uid": targetUID,
		"admins":    uniqueNonEmpty(admins),
	})
}

// DismissGroup 解散群聊。
func (s *GroupService) DismissGroup(ctx context.Context, chatID, operatorUID string) (*model.Chat, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, operatorUID)
	if err != nil {
		return nil, err
	}
	if !isOwner(chat, operatorUID) {
		return nil, ErrForbidden
	}
	dismissed, err := model.DismissGroupChat(ctx, s.chatColl, chatID)
	if err != nil {
		return nil, err
	}
	if s.convMgr != nil {
		for _, uid := range chat.Members {
			if err := s.convMgr.MarkLeft(ctx, uid, chatID); err != nil {
				return nil, err
			}
		}
	}
	return dismissed, nil
}

// CheckPermission 检查群权限。
func (s *GroupService) CheckPermission(ctx context.Context, chatID, uid, action string) (bool, string, error) {
	chat, err := s.GetGroupForMember(ctx, chatID, uid)
	if err != nil {
		return false, "not_group_member", err
	}
	if chat.Settings.IsMutedAll && !isOwner(chat, uid) && !isAdmin(chat, uid) {
		return false, "group_muted_all", nil
	}
	if containsString(chat.Settings.MutedMembers, uid) {
		return false, "member_muted", nil
	}
	switch action {
	case "invite_member", "kick_member", "update_group_info", "set_announcement":
		if !canUpdateGroupInfo(chat, uid) {
			return false, "permission_denied", nil
		}
	case "dismiss_group", "transfer_owner", "set_member_role", "update_settings":
		if !isOwner(chat, uid) {
			return false, "owner_required", nil
		}
	}
	return true, "", nil
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
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

func removeString(items []string, target string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != target {
			result = append(result, item)
		}
	}
	return result
}

func isActiveGroup(chat *model.Chat) bool {
	return chat != nil && chat.ChatType == types.ChatTypeGroup && chat.Status != model.GroupStatusDismissed
}

func isOwner(chat *model.Chat, uid string) bool {
	return chat != nil && uid != "" && chat.OwnerUID == uid
}

func isAdmin(chat *model.Chat, uid string) bool {
	return chat != nil && uid != "" && containsString(chat.Admins, uid)
}

func canUpdateGroupInfo(chat *model.Chat, uid string) bool {
	return isOwner(chat, uid) || isAdmin(chat, uid)
}

func canManageMembers(chat *model.Chat, uid string) bool {
	return canUpdateGroupInfo(chat, uid)
}

func ensureCapacity(chat *model.Chat, adding int) error {
	if chat == nil {
		return mongo.ErrNoDocuments
	}
	maxMembers := chat.MaxMembers
	if maxMembers <= 0 {
		maxMembers = defaultMaxMembers
	}
	if chat.MemberCount+adding > maxMembers {
		return ErrGroupFull
	}
	return nil
}
