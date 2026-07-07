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
	groups  *groupSvc.GroupService
	members *groupSvc.MemberService
	convSvc conversationByChatReader
	msgSvc  groupMessageSender
	users   userReader
}

type conversationByChatReader interface {
	GetConversationByChatID(ctx context.Context, uid, chatID string) (*model.Conversation, error)
}

type groupMessageSender interface {
	Send(ctx context.Context, req *messageSvc.SendMessageReq) (*messageSvc.SendMessageResp, error)
}

// NewGroupHandler 创建群聊处理器。
func NewGroupHandler(groups *groupSvc.GroupService, members *groupSvc.MemberService, convSvc conversationByChatReader, msgSvc groupMessageSender, users userReader) *GroupHandler {
	return &GroupHandler{groups: groups, members: members, convSvc: convSvc, msgSvc: msgSvc, users: users}
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
	for _, group := range groups {
		items = append(items, h.groupDTO(group, h.currentConversationID(r, group.ChatID)))
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

	group, err := h.groups.CreateGroup(r.Context(), req.Name, uid, req.MemberUserIDs)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500401, "create group failed")
		return
	}
	h.sendGroupSystemEvent(r, group, groupSystemEvent{
		EventType:     "group_created",
		OperatorID:    uid,
		TargetUserIDs: withoutUID(uniqueNonEmpty(append([]string{uid}, req.MemberUserIDs...)), uid),
		Text:          h.operatorName(r, uid) + "创建了群聊",
	})

	conversationID := ""
	resp := map[string]interface{}{}
	if h.convSvc != nil {
		if conv, err := h.convSvc.GetConversationByChatID(r.Context(), uid, group.ChatID); err == nil {
			conversationID = conv.ConversationID
			resp["conversation"] = map[string]interface{}{
				"id":              conv.ConversationID,
				"conversation_id": conv.ConversationID,
				"chat_id":         conv.ChatID,
				"chat_type":       conv.ChatType,
			}
		}
	}
	resp["group"] = h.groupDTO(group, conversationID)
	writeAPISuccess(w, resp)
}

// GetGroup 获取群详情。
// GET /api/v1/groups/{id}
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	members, err := h.groups.ListMembers(r.Context(), group.ChatID, 20, 0)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500405, "get group members failed")
		return
	}
	writeAPISuccess(w, map[string]interface{}{
		"group":   h.groupDTO(group, h.currentConversationID(r, group.ChatID)),
		"members": h.memberDTOs(r, group, members),
	})
}

// UpdateGroup 更新群资料。
// PATCH /api/v1/groups/{id}
func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	group := h.requireGroupMember(w, r)
	if group == nil {
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

	beforeName := group.Name
	updated, err := h.groups.UpdateInfo(r.Context(), group.ChatID, middleware.GetUserID(r.Context()), groupSvc.UpdateGroupInfo{
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
	group := h.requireGroupMember(w, r)
	if group == nil {
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

	members, err := h.groups.ListMembers(r.Context(), group.ChatID, int64(limit), int64(start))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500405, "get group members failed")
		return
	}
	nextCursor := ""
	if len(members) == limit {
		nextCursor = strconv.Itoa(start + len(members))
	}

	writeAPISuccess(w, map[string]interface{}{
		"items":       h.memberDTOs(r, group, members),
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// InviteMembers 邀请成员入群。
// POST /api/v1/groups/{id}/members
func (h *GroupHandler) InviteMembers(w http.ResponseWriter, r *http.Request) {
	group := h.requireGroupMember(w, r)
	if group == nil {
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

	updated, addedUserIDs, err := h.groups.AddMembers(r.Context(), group.ChatID, middleware.GetUserID(r.Context()), req.MemberUserIDs)
	if err != nil {
		h.writeGroupServiceError(w, err, 500403, "invite members failed")
		return
	}
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
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	if _, err := h.groups.LeaveGroup(r.Context(), group.ChatID, uid); err != nil {
		h.writeGroupServiceError(w, err, 500404, "leave group failed")
		return
	}
	h.sendGroupSystemEvent(r, group, groupSystemEvent{
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
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	targetUID := r.PathValue("uid")
	if targetUID == "" {
		writeAPIError(w, http.StatusBadRequest, 400024, "member uid is required")
		return
	}
	updated, err := h.groups.KickMember(r.Context(), group.ChatID, operatorUID, targetUID)
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
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	updated, err := h.groups.DismissGroup(r.Context(), group.ChatID, operatorUID)
	if err != nil {
		h.writeGroupServiceError(w, err, 500409, "dismiss group failed")
		return
	}
	h.sendGroupSystemEvent(r, group, groupSystemEvent{
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
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	settings := group.Settings
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
	updated, err := h.groups.UpdateSettings(r.Context(), group.ChatID, operatorUID, settings)
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
	group := h.requireGroupMember(w, r)
	if group == nil {
		return
	}
	var req struct {
		Announcement string `json:"announcement"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	updated, err := h.groups.SetAnnouncement(r.Context(), group.ChatID, operatorUID, req.Announcement)
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
	group := h.requireGroupMember(w, r)
	if group == nil {
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
	updated, err := h.groups.SetMemberRole(r.Context(), group.ChatID, operatorUID, targetUID, req.Role)
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
	group := h.requireGroupMember(w, r)
	if group == nil {
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
	updated, err := h.groups.TransferOwner(r.Context(), group.ChatID, operatorUID, targetUID)
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

func (h *GroupHandler) requireGroupMember(w http.ResponseWriter, r *http.Request) *model.Group {
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
	group, err := h.groups.GetGroupForMember(r.Context(), groupID, uid)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404002, "group not found")
			return nil
		}
		writeAPIError(w, http.StatusInternalServerError, 500405, "get group failed")
		return nil
	}
	return group
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

func (h *GroupHandler) sendGroupSystemEvent(r *http.Request, group *model.Group, event groupSystemEvent) {
	if h.msgSvc == nil || group == nil || event.EventType == "" || event.Text == "" {
		return
	}
	_, _ = h.msgSvc.Send(r.Context(), &messageSvc.SendMessageReq{
		ChatID:     group.ChatID,
		ChatType:   types.ChatTypeGroup,
		SenderID:   event.OperatorID,
		SenderName: h.operatorName(r, event.OperatorID),
		MsgType:    types.MessageTypeSystemEvent,
		Content: types.SystemEventContent{
			EventType:     event.EventType,
			Text:          event.Text,
			Title:         event.Text,
			OperatorID:    event.OperatorID,
			TargetUserIDs: event.TargetUserIDs,
			GroupID:       group.ChatID,
			GroupName:     group.Name,
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
