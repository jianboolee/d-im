package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"d-im/internal/gateway/handler/middleware"
	groupSvc "d-im/internal/group/service"
	messageSvc "d-im/internal/message/service"
	"d-im/pkg/model"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/mongo"
)

// GroupHandler 群聊 HTTP 处理器。
type GroupHandler struct {
	groups  groupOperator
	convSvc conversationByChatReader
	msgSvc  groupMessageSender
	users   userReader
}

type groupOperator interface {
	CreateGroup(ctx context.Context, name, ownerUID string, memberUIDs []string) (*model.Chat, error)
	ListGroupsForMember(ctx context.Context, uid string, limit, offset int64) ([]*model.Chat, error)
	GetGroupForMember(ctx context.Context, chatID, uid string) (*model.Chat, error)
	JoinGroup(ctx context.Context, chatID, uid string) (*model.Chat, error)
	AddMembers(ctx context.Context, chatID, operatorUID string, uidList []string) (*model.Chat, error)
	RemoveMember(ctx context.Context, chatID, uid string) error
	LeaveGroup(ctx context.Context, chatID, uid string) (*model.Chat, error)
	KickMember(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Chat, error)
	UpdateInfo(ctx context.Context, chatID, operatorUID string, info groupSvc.UpdateGroupInfo) (*model.Chat, error)
	UpdateName(ctx context.Context, chatID, operatorUID, name string) (*model.Chat, error)
	UpdateSettings(ctx context.Context, chatID, operatorUID string, settings model.GroupSettings) (*model.Chat, error)
	SetAnnouncement(ctx context.Context, chatID, operatorUID, announcement string) (*model.Chat, error)
	SetMemberRole(ctx context.Context, chatID, operatorUID, targetUID string, role model.MemberRole) (*model.Chat, error)
	TransferOwner(ctx context.Context, chatID, operatorUID, targetUID string) (*model.Chat, error)
	DismissGroup(ctx context.Context, chatID, operatorUID string) (*model.Chat, error)
}

type conversationByChatReader interface {
	GetConversationByChatID(ctx context.Context, uid, chatID string) (*model.Conversation, error)
}

type groupMessageSender interface {
	Send(ctx context.Context, req *messageSvc.SendMessageReq) (*messageSvc.SendMessageResp, error)
}

// NewGroupHandler 创建群聊处理器。
func NewGroupHandler(groups groupOperator, convSvc conversationByChatReader, msgSvc groupMessageSender, users userReader) *GroupHandler {
	return &GroupHandler{groups: groups, convSvc: convSvc, msgSvc: msgSvc, users: users}
}

// ListGroups 获取当前用户加入的群列表。
// GET /api/v1/groups?limit=20&offset=0
func (h *GroupHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}
	limit, offset, ok := parseLimitOffset(w, r)
	if !ok {
		return
	}
	groups, err := h.groups.ListGroupsForMember(r.Context(), uid, int64(limit), int64(offset))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500406, "list groups failed")
		return
	}
	items := make([]groupDTO, 0, len(groups))
	for _, chat := range groups {
		items = append(items, h.groupDTO(chat, h.currentConversationID(r, chat.ChatID)))
	}
	writeAPISuccess(w, map[string]interface{}{
		"items": items,
	})
}

// CreateGroup 创建群聊。
// POST /api/v1/groups
func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	var req struct {
		Name          string   `json:"name"`
		MemberUserIDs []string `json:"member_user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeAPIError(w, http.StatusBadRequest, 400020, "name is required")
		return
	}

	chat, err := h.groups.CreateGroup(r.Context(), req.Name, uid, req.MemberUserIDs)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500401, "create group failed")
		return
	}
	h.sendGroupSystemEvent(r, chat, groupSystemEvent{
		EventType:     "group_created",
		OperatorID:    uid,
		TargetUserIDs: withoutUID(chat.Members, uid),
		Text:          h.operatorName(r, uid) + "创建了群聊",
	})

	conversationID := ""
	resp := map[string]interface{}{}
	if h.convSvc != nil {
		if conv, err := h.convSvc.GetConversationByChatID(r.Context(), uid, chat.ChatID); err == nil {
			conversationID = conv.ConversationID
			resp["conversation"] = map[string]interface{}{
				"id":              conv.ConversationID,
				"conversation_id": conv.ConversationID,
				"chat_id":         conv.ChatID,
				"chat_type":       conv.ChatType,
			}
		}
	}
	resp["group"] = h.groupDTO(chat, conversationID)
	writeAPISuccess(w, resp)
}

// GetGroup 获取群详情。
// GET /api/v1/groups/{id}
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	writeAPISuccess(w, map[string]interface{}{
		"group":   h.groupDTO(chat, h.currentConversationID(r, chat.ChatID)),
		"members": h.memberDTOs(r, chat, 0, minInt(len(chat.Members), 20)),
	})
}

// UpdateGroup 更新群资料。
// PATCH /api/v1/groups/{id}
func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}

	var req struct {
		Name        *string `json:"name"`
		AvatarURL   *string `json:"avatar_url"`
		Avatar      *string `json:"avatar"`
		Description *string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.Name == nil && req.AvatarURL == nil && req.Avatar == nil && req.Description == nil {
		writeAPIError(w, http.StatusBadRequest, 400023, "no fields to update")
		return
	}
	avatar := req.Avatar
	if avatar == nil {
		avatar = req.AvatarURL
	}

	beforeName := chat.Name
	updated, err := h.groups.UpdateInfo(r.Context(), chat.ChatID, middleware.GetUserID(r.Context()), groupSvc.UpdateGroupInfo{
		Name:        req.Name,
		Avatar:      avatar,
		Description: req.Description,
	})
	if err != nil {
		h.writeGroupServiceError(w, err, 500402, "update group failed")
		return
	}
	if req.Name != nil && beforeName != updated.Name {
		h.sendGroupSystemEvent(r, updated, groupSystemEvent{
			EventType:   "group_name_updated",
			OperatorID:  middleware.GetUserID(r.Context()),
			Text:        h.operatorName(r, middleware.GetUserID(r.Context())) + "修改群名为“" + updated.Name + "”",
			BeforeValue: beforeName,
			AfterValue:  updated.Name,
		})
	}
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// ListMembers 获取群成员列表。
// GET /api/v1/groups/{id}/members?limit=20&cursor=0
func (h *GroupHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}

	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeAPIError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = minInt(parsed, 100)
	}

	start := 0
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		parsed, err := strconv.Atoi(cursor)
		if err != nil || parsed < 0 {
			writeAPIError(w, http.StatusBadRequest, 400009, "invalid cursor")
			return
		}
		start = parsed
	}

	end := minInt(start+limit, len(chat.Members))
	nextCursor := ""
	if end < len(chat.Members) {
		nextCursor = strconv.Itoa(end)
	}

	writeAPISuccess(w, map[string]interface{}{
		"items":       h.memberDTOs(r, chat, start, end),
		"next_cursor": nextCursor,
		"has_more":    end < len(chat.Members),
	})
}

// InviteMembers 邀请成员入群。
// POST /api/v1/groups/{id}/members
func (h *GroupHandler) InviteMembers(w http.ResponseWriter, r *http.Request) {
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}

	var req struct {
		MemberUserIDs []string `json:"member_user_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if len(req.MemberUserIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, 400021, "member_user_ids is required")
		return
	}

	beforeMembers := append([]string{}, chat.Members...)
	updated, err := h.groups.AddMembers(r.Context(), chat.ChatID, middleware.GetUserID(r.Context()), req.MemberUserIDs)
	if err != nil {
		h.writeGroupServiceError(w, err, 500403, "invite members failed")
		return
	}
	addedUserIDs := diffNewMembers(beforeMembers, updated.Members)
	if len(addedUserIDs) > 0 {
		h.sendGroupSystemEvent(r, updated, groupSystemEvent{
			EventType:     "members_invited",
			OperatorID:    middleware.GetUserID(r.Context()),
			TargetUserIDs: addedUserIDs,
			Text:          h.operatorName(r, middleware.GetUserID(r.Context())) + "邀请" + h.userListText(r, addedUserIDs) + "加入群聊",
		})
	}
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// LeaveGroup 退出群聊。
// POST /api/v1/groups/{id}/leave
func (h *GroupHandler) LeaveGroup(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	if _, err := h.groups.LeaveGroup(r.Context(), chat.ChatID, uid); err != nil {
		h.writeGroupServiceError(w, err, 500404, "leave group failed")
		return
	}
	h.sendGroupSystemEvent(r, chat, groupSystemEvent{
		EventType:     "member_left",
		OperatorID:    uid,
		TargetUserIDs: []string{uid},
		Text:          h.operatorName(r, uid) + "退出了群聊",
	})
	writeAPISuccess(w, map[string]interface{}{})
}

// JoinGroup 加入公开自由入群的群。
// POST /api/v1/groups/{id}/join
func (h *GroupHandler) JoinGroup(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}
	groupID := r.PathValue("id")
	if groupID == "" {
		writeAPIError(w, http.StatusBadRequest, 400022, "group_id is required")
		return
	}
	updated, err := h.groups.JoinGroup(r.Context(), groupID, uid)
	if err != nil {
		h.writeGroupServiceError(w, err, 500407, "join group failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:     "member_joined",
		OperatorID:    uid,
		TargetUserIDs: []string{uid},
		Text:          h.operatorName(r, uid) + "加入了群聊",
	})
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// KickMember 踢出群成员。
// DELETE /api/v1/groups/{id}/members/{uid}
func (h *GroupHandler) KickMember(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	targetUID := r.PathValue("uid")
	if targetUID == "" {
		writeAPIError(w, http.StatusBadRequest, 400024, "member uid is required")
		return
	}
	updated, err := h.groups.KickMember(r.Context(), chat.ChatID, operatorUID, targetUID)
	if err != nil {
		h.writeGroupServiceError(w, err, 500408, "kick member failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:     "member_kicked",
		OperatorID:    operatorUID,
		TargetUserIDs: []string{targetUID},
		Text:          h.operatorName(r, operatorUID) + "将" + h.operatorName(r, targetUID) + "移出了群聊",
	})
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// DismissGroup 解散群。
// DELETE /api/v1/groups/{id}
func (h *GroupHandler) DismissGroup(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	updated, err := h.groups.DismissGroup(r.Context(), chat.ChatID, operatorUID)
	if err != nil {
		h.writeGroupServiceError(w, err, 500409, "dismiss group failed")
		return
	}
	h.sendGroupSystemEvent(r, chat, groupSystemEvent{
		EventType:  "group_dismissed",
		OperatorID: operatorUID,
		Text:       h.operatorName(r, operatorUID) + "解散了群聊",
	})
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, ""),
	})
}

// UpdateSettings 更新群设置。
// PATCH /api/v1/groups/{id}/settings
func (h *GroupHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	settings := chat.Settings
	var req struct {
		JoinMethod   *model.JoinMethod `json:"join_method"`
		IsMutedAll   *bool             `json:"is_muted_all"`
		IsPublic     *bool             `json:"is_public"`
		MutedMembers []string          `json:"muted_members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.JoinMethod != nil {
		settings.JoinMethod = *req.JoinMethod
	}
	if req.IsMutedAll != nil {
		settings.IsMutedAll = *req.IsMutedAll
	}
	if req.IsPublic != nil {
		settings.IsPublic = *req.IsPublic
	}
	if req.MutedMembers != nil {
		settings.MutedMembers = uniqueNonEmpty(req.MutedMembers)
	}
	if !validJoinMethod(settings.JoinMethod) {
		writeAPIError(w, http.StatusBadRequest, 400025, "invalid join_method")
		return
	}
	updated, err := h.groups.UpdateSettings(r.Context(), chat.ChatID, operatorUID, settings)
	if err != nil {
		h.writeGroupServiceError(w, err, 500410, "update group settings failed")
		return
	}
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// SetAnnouncement 设置群公告。
// PUT /api/v1/groups/{id}/announcement
func (h *GroupHandler) SetAnnouncement(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	var req struct {
		Announcement string `json:"announcement"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	updated, err := h.groups.SetAnnouncement(r.Context(), chat.ChatID, operatorUID, req.Announcement)
	if err != nil {
		h.writeGroupServiceError(w, err, 500411, "set announcement failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:  "group_announcement_updated",
		OperatorID: operatorUID,
		Text:       h.operatorName(r, operatorUID) + "更新了群公告",
		AfterValue: updated.Announcement,
	})
	writeAPISuccess(w, map[string]interface{}{
		"announcement": updated.Announcement,
		"group":        h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// SetMemberRole 设置群成员角色。
// PATCH /api/v1/groups/{id}/members/{uid}/role
func (h *GroupHandler) SetMemberRole(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	targetUID := r.PathValue("uid")
	var req struct {
		Role model.MemberRole `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	updated, err := h.groups.SetMemberRole(r.Context(), chat.ChatID, operatorUID, targetUID, req.Role)
	if err != nil {
		h.writeGroupServiceError(w, err, 500412, "set member role failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:     "member_role_changed",
		OperatorID:    operatorUID,
		TargetUserIDs: []string{targetUID},
		Text:          h.operatorName(r, operatorUID) + "更新了" + h.operatorName(r, targetUID) + "的群角色",
		AfterValue:    string(req.Role),
	})
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

// TransferOwner 转让群主。
// POST /api/v1/groups/{id}/owner
func (h *GroupHandler) TransferOwner(w http.ResponseWriter, r *http.Request) {
	operatorUID := middleware.GetUserID(r.Context())
	chat := h.requireGroupMember(w, r)
	if chat == nil {
		return
	}
	var req struct {
		OwnerUID string `json:"owner_uid"`
		UserID   string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	targetUID := strings.TrimSpace(req.OwnerUID)
	if targetUID == "" {
		targetUID = strings.TrimSpace(req.UserID)
	}
	if targetUID == "" {
		writeAPIError(w, http.StatusBadRequest, 400026, "owner_uid is required")
		return
	}
	updated, err := h.groups.TransferOwner(r.Context(), chat.ChatID, operatorUID, targetUID)
	if err != nil {
		h.writeGroupServiceError(w, err, 500413, "transfer owner failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:     "group_owner_transferred",
		OperatorID:    operatorUID,
		TargetUserIDs: []string{targetUID},
		Text:          h.operatorName(r, operatorUID) + "将群主转让给" + h.operatorName(r, targetUID),
		BeforeValue:   operatorUID,
		AfterValue:    targetUID,
	})
	writeAPISuccess(w, map[string]interface{}{
		"group": h.groupDTO(updated, h.currentConversationID(r, updated.ChatID)),
	})
}

func (h *GroupHandler) requireGroupMember(w http.ResponseWriter, r *http.Request) *model.Chat {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return nil
	}
	groupID := r.PathValue("id")
	if groupID == "" {
		writeAPIError(w, http.StatusBadRequest, 400022, "group_id is required")
		return nil
	}
	chat, err := h.groups.GetGroupForMember(r.Context(), groupID, uid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404002, "group not found")
			return nil
		}
		writeAPIError(w, http.StatusInternalServerError, 500405, "get group failed")
		return nil
	}
	return chat
}

func (h *GroupHandler) currentConversationID(r *http.Request, chatID string) string {
	if h.convSvc == nil {
		return ""
	}
	uid := middleware.GetUserID(r.Context())
	conv, err := h.convSvc.GetConversationByChatID(r.Context(), uid, chatID)
	if err != nil {
		return ""
	}
	return conv.ConversationID
}

type groupDTO struct {
	ID             string              `json:"id"`
	ConversationID string              `json:"conversation_id,omitempty"`
	Name           string              `json:"name"`
	AvatarURL      string              `json:"avatar_url,omitempty"`
	Description    string              `json:"description,omitempty"`
	OwnerID        string              `json:"owner_id"`
	Admins         []string            `json:"admins,omitempty"`
	MemberCount    int                 `json:"member_count"`
	MaxMembers     int                 `json:"max_members,omitempty"`
	Settings       model.GroupSettings `json:"settings"`
	Announcement   string              `json:"announcement,omitempty"`
	Status         string              `json:"status"`
	CreatedAt      string              `json:"created_at"`
	UpdatedAt      string              `json:"updated_at"`
}

type groupMemberDTO struct {
	ID            string   `json:"id"`
	GroupID       string   `json:"group_id"`
	UserID        string   `json:"user_id"`
	Role          string   `json:"role"`
	Status        string   `json:"status"`
	GroupNickname string   `json:"group_nickname,omitempty"`
	JoinedAt      string   `json:"joined_at"`
	InvitedBy     string   `json:"invited_by,omitempty"`
	UserInfo      *userDTO `json:"user_info,omitempty"`
}

func (h *GroupHandler) groupDTO(chat *model.Chat, conversationID string) groupDTO {
	status := string(chat.Status)
	if status == "" {
		status = string(model.GroupStatusActive)
	}
	return groupDTO{
		ID:             chat.ChatID,
		ConversationID: conversationID,
		Name:           chat.Name,
		AvatarURL:      chat.Avatar,
		Description:    chat.Description,
		OwnerID:        chat.OwnerUID,
		Admins:         chat.Admins,
		MemberCount:    chat.MemberCount,
		MaxMembers:     chat.MaxMembers,
		Settings:       chat.Settings,
		Announcement:   chat.Announcement,
		Status:         status,
		CreatedAt:      chat.CreatedAt.Format(timeFormatRFC3339Nano),
		UpdatedAt:      chat.UpdatedAt.Format(timeFormatRFC3339Nano),
	}
}

func (h *GroupHandler) memberDTOs(r *http.Request, chat *model.Chat, start, end int) []groupMemberDTO {
	if start < 0 {
		start = 0
	}
	if end > len(chat.Members) {
		end = len(chat.Members)
	}
	if start > end {
		start = end
	}
	items := make([]groupMemberDTO, 0, end-start)
	for _, userID := range chat.Members[start:end] {
		role := "member"
		if userID == chat.OwnerUID {
			role = "owner"
		} else if containsUID(chat.Admins, userID) {
			role = "admin"
		}
		item := groupMemberDTO{
			ID:        chat.ChatID + ":" + userID,
			GroupID:   chat.ChatID,
			UserID:    userID,
			Role:      role,
			Status:    "active",
			JoinedAt:  chat.CreatedAt.Format(timeFormatRFC3339Nano),
			InvitedBy: chat.OwnerUID,
		}
		if h.users != nil {
			if user, err := h.users.FindByID(r.Context(), userID); err == nil {
				dto := userDTOFromModel(user)
				item.UserInfo = &dto
			}
		}
		items = append(items, item)
	}
	return items
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type groupSystemEvent struct {
	EventType     string
	OperatorID    string
	TargetUserIDs []string
	Text          string
	BeforeValue   string
	AfterValue    string
}

func (h *GroupHandler) sendGroupSystemEvent(r *http.Request, chat *model.Chat, event groupSystemEvent) {
	if h.msgSvc == nil || chat == nil || event.EventType == "" || event.Text == "" {
		return
	}
	_, _ = h.msgSvc.Send(r.Context(), &messageSvc.SendMessageReq{
		ChatID:     chat.ChatID,
		ChatType:   chat.ChatType,
		SenderID:   event.OperatorID,
		SenderName: h.operatorName(r, event.OperatorID),
		MsgType:    types.MessageTypeSystemEvent,
		Content: types.SystemEventContent{
			EventType:     event.EventType,
			Text:          event.Text,
			Title:         event.Text,
			OperatorID:    event.OperatorID,
			TargetUserIDs: event.TargetUserIDs,
			GroupID:       chat.ChatID,
			GroupName:     chat.Name,
			BeforeValue:   event.BeforeValue,
			AfterValue:    event.AfterValue,
		},
	})
}

func (h *GroupHandler) operatorName(r *http.Request, uid string) string {
	if uid == "" {
		return "系统"
	}
	if h.users != nil {
		if user, err := h.users.FindByID(r.Context(), uid); err == nil && user.Nickname != "" {
			return user.Nickname
		}
	}
	return uid
}

func (h *GroupHandler) userListText(r *http.Request, userIDs []string) string {
	names := make([]string, 0, len(userIDs))
	for _, uid := range userIDs {
		names = append(names, h.operatorName(r, uid))
	}
	return strings.Join(names, "、")
}

func withoutUID(items []string, uid string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item != "" && item != uid {
			result = append(result, item)
		}
	}
	return result
}

func diffNewMembers(before, after []string) []string {
	seen := make(map[string]bool, len(before))
	for _, uid := range before {
		seen[uid] = true
	}
	result := make([]string, 0)
	for _, uid := range after {
		if uid == "" || seen[uid] {
			continue
		}
		seen[uid] = true
		result = append(result, uid)
	}
	return result
}

func (h *GroupHandler) writeGroupServiceError(w http.ResponseWriter, err error, fallbackCode int, fallbackMessage string) {
	switch {
	case errors.Is(err, mongo.ErrNoDocuments):
		writeAPIError(w, http.StatusNotFound, 404002, "group not found")
	case errors.Is(err, groupSvc.ErrForbidden):
		writeAPIError(w, http.StatusForbidden, 403001, "forbidden")
	case errors.Is(err, groupSvc.ErrInvalid):
		writeAPIError(w, http.StatusBadRequest, 400027, err.Error())
	case errors.Is(err, groupSvc.ErrGroupFull):
		writeAPIError(w, http.StatusBadRequest, 400028, "group is full")
	default:
		writeAPIError(w, http.StatusInternalServerError, fallbackCode, fallbackMessage)
	}
}

func parseLimitOffset(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil || parsed <= 0 {
			writeAPIError(w, http.StatusBadRequest, 400004, "invalid limit")
			return 0, 0, false
		}
		limit = minInt(parsed, 100)
	}
	offset := 0
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		parsed, err := strconv.Atoi(rawOffset)
		if err != nil || parsed < 0 {
			writeAPIError(w, http.StatusBadRequest, 400009, "invalid offset")
			return 0, 0, false
		}
		offset = parsed
	}
	return limit, offset, true
}

func validJoinMethod(method model.JoinMethod) bool {
	switch method {
	case "", model.JoinMethodFree, model.JoinMethodVerify, model.JoinMethodInvite, model.JoinMethodForbidden:
		return true
	default:
		return false
	}
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

func containsUID(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
