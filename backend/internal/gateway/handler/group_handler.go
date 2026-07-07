package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"d-im/internal/gateway/handler/middleware"
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
	GetGroupForMember(ctx context.Context, chatID, uid string) (*model.Chat, error)
	AddMembers(ctx context.Context, chatID string, uidList []string) (*model.Chat, error)
	RemoveMember(ctx context.Context, chatID, uid string) error
	UpdateName(ctx context.Context, chatID, name string) (*model.Chat, error)
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
		Name *string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.Name == nil || strings.TrimSpace(*req.Name) == "" {
		writeAPIError(w, http.StatusBadRequest, 400020, "name is required")
		return
	}

	beforeName := chat.Name
	updated, err := h.groups.UpdateName(r.Context(), chat.ChatID, *req.Name)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500402, "update group failed")
		return
	}
	h.sendGroupSystemEvent(r, updated, groupSystemEvent{
		EventType:   "group_name_updated",
		OperatorID:  middleware.GetUserID(r.Context()),
		Text:        h.operatorName(r, middleware.GetUserID(r.Context())) + "修改群名为“" + updated.Name + "”",
		BeforeValue: beforeName,
		AfterValue:  updated.Name,
	})
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
	updated, err := h.groups.AddMembers(r.Context(), chat.ChatID, req.MemberUserIDs)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500403, "invite members failed")
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
	if err := h.groups.RemoveMember(r.Context(), chat.ChatID, uid); err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500404, "leave group failed")
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
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id,omitempty"`
	Name           string `json:"name"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	OwnerID        string `json:"owner_id"`
	MemberCount    int    `json:"member_count"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
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
	return groupDTO{
		ID:             chat.ChatID,
		ConversationID: conversationID,
		Name:           chat.Name,
		AvatarURL:      chat.Avatar,
		OwnerID:        chat.OwnerUID,
		MemberCount:    chat.MemberCount,
		Status:         "active",
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
