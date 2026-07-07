package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	chatRepo "d-im/internal/chat/repository"
	"d-im/internal/conversation/service"
	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"
	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/mongo"
)

// ConversationHandler 会话HTTP处理器
type ConversationHandler struct {
	convSvc  *service.ConversationService
	chatRepo *chatRepo.ChatRepo
	groups   conversationGroupReader
	users    userReader
}

type conversationGroupReader interface {
	GetGroup(ctx context.Context, chatID string) (*model.Group, error)
	GetMemberUIDs(ctx context.Context, chatID string) ([]string, error)
}

// NewConversationHandler 创建会话处理器
func NewConversationHandler(convSvc *service.ConversationService, chatRepo *chatRepo.ChatRepo, groups conversationGroupReader, users userReader) *ConversationHandler {
	return &ConversationHandler{convSvc: convSvc, chatRepo: chatRepo, groups: groups, users: users}
}

// ListConversations 获取用户会话列表
// GET /api/v1/conversations?limit=20&cursor=
func (h *ConversationHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	limit := int64(20)
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, err := strconv.ParseInt(rawLimit, 10, 64)
		if err != nil || parsed <= 0 {
			writeAPIError(w, http.StatusBadRequest, 400004, "invalid limit")
			return
		}
		limit = parsed
	}

	list, nextCursor, hasMore, err := h.convSvc.GetListByCursor(r.Context(), uid, limit, r.URL.Query().Get("cursor"))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, 400005, "invalid cursor")
		return
	}

	if list == nil {
		list = []*model.Conversation{}
	}

	items := make([]conversationDTO, 0, len(list))
	for _, conv := range list {
		items = append(items, h.conversationDTO(r.Context(), conv, uid))
	}

	writeAPISuccess(w, map[string]interface{}{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

// GetConversation 获取会话详情
// GET /api/v1/conversations/{id}
func (h *ConversationHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeAPIError(w, http.StatusBadRequest, 400008, "conversation_id is required")
		return
	}

	conv, err := h.convSvc.GetConversation(r.Context(), uid, conversationID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, 500202, "get conversation failed")
		return
	}

	writeAPISuccess(w, h.conversationDTO(r.Context(), conv, uid))
}

// GetConversationByChat 获取当前用户在指定 chat 下的会话视图。
// GET /api/v1/chats/{id}/conversation
func (h *ConversationHandler) GetConversationByChat(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	chatID := r.PathValue("id")
	if chatID == "" {
		writeAPIError(w, http.StatusBadRequest, 400008, "chat_id is required")
		return
	}

	conv, err := h.convSvc.GetConversationByChatID(r.Context(), uid, chatID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, 500202, "get conversation failed")
		return
	}

	writeAPISuccess(w, h.conversationDTO(r.Context(), conv, uid))
}

// CreateSingleConversation 创建或获取单聊会话
// POST /api/v1/conversations/single
func (h *ConversationHandler) CreateSingleConversation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	var req struct {
		PeerUserID string `json:"peer_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.PeerUserID == "" {
		writeAPIError(w, http.StatusBadRequest, 400006, "peer_user_id is required")
		return
	}
	if req.PeerUserID == uid {
		writeAPIError(w, http.StatusBadRequest, 400007, "cannot create single conversation with self")
		return
	}

	conv, err := h.convSvc.CreateOrGetSingle(r.Context(), uid, req.PeerUserID)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, 500201, "create conversation failed")
		return
	}

	writeAPISuccess(w, h.conversationDTO(r.Context(), conv, uid))
}

// ReadConversation 标记已读
// POST /api/v1/conversations/{id}/read
func (h *ConversationHandler) ReadConversation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeAPIError(w, http.StatusBadRequest, 400008, "conversation_id is required")
		return
	}

	var req struct {
		LastReadSequence int64 `json:"last_read_sequence"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
			return
		}
	}
	if req.LastReadSequence < 0 {
		writeAPIError(w, http.StatusBadRequest, 400009, "last_read_sequence is invalid")
		return
	}

	conv, err := h.convSvc.ReadConversation(r.Context(), uid, conversationID, req.LastReadSequence)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, 500203, "mark conversation read failed")
		return
	}

	writeAPISuccess(w, map[string]interface{}{
		"conversation_id":    conv.ConversationID,
		"chat_id":            conv.ChatID,
		"last_read_sequence": conv.LastReadSeq,
		"read_at":            formatOptionalTime(conv.LastReadAt, conv.UpdatedAt),
	})
}

// UpdateConversationSettings 更新当前用户的会话设置。
// PATCH /api/v1/conversations/{id}/settings
func (h *ConversationHandler) UpdateConversationSettings(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeAPIError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		writeAPIError(w, http.StatusBadRequest, 400008, "conversation_id is required")
		return
	}

	var req struct {
		Pinned *bool `json:"pinned"`
		Muted  *bool `json:"muted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.Pinned == nil && req.Muted == nil {
		writeAPIError(w, http.StatusBadRequest, 400013, "settings is empty")
		return
	}

	conv, err := h.convSvc.UpdateSettings(r.Context(), uid, conversationID, req.Pinned, req.Muted)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			writeAPIError(w, http.StatusNotFound, 404001, "conversation not found")
			return
		}
		writeAPIError(w, http.StatusInternalServerError, 500204, "update conversation settings failed")
		return
	}

	writeAPISuccess(w, h.conversationDTO(r.Context(), conv, uid))
}

type conversationDTO struct {
	ID               string             `json:"id"`
	ConversationID   string             `json:"conversation_id"`
	ChatID           string             `json:"chat_id"`
	ChatType         types.ChatType     `json:"chat_type"`
	Title            string             `json:"title"`
	Avatar           string             `json:"avatar"`
	Participants     []string           `json:"participants"`
	PeerUser         *userDTO           `json:"peer_user"`
	Group            interface{}        `json:"group"`
	GroupID          string             `json:"group_id,omitempty"`
	GroupInfo        *groupSummaryDTO   `json:"group_info,omitempty"`
	LastMessage      *types.LastMessage `json:"last_message"`
	LastReadSequence int64              `json:"last_read_sequence"`
	LastReadAt       string             `json:"last_read_at,omitempty"`
	Muted            bool               `json:"muted"`
	Pinned           bool               `json:"pinned"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
	LastActivityAt   string             `json:"last_activity_at"`
}

func (h *ConversationHandler) conversationDTO(ctx context.Context, conv *model.Conversation, currentUserID string) conversationDTO {
	participants := []string{}
	var peer *userDTO
	var groupInfo *groupSummaryDTO
	title := conv.CustomName
	avatar := ""
	groupID := ""

	if h.chatRepo != nil {
		if chat, err := h.chatRepo.FindByChatID(ctx, conv.ChatID); err == nil {
			if conv.ChatType == types.ChatTypeSingle {
				participants = append(participants, chat.Members...)
			}
		}
	}
	if conv.ChatType == types.ChatTypeGroup && h.groups != nil {
		if group, err := h.groups.GetGroup(ctx, conv.ChatID); err == nil {
			groupID = group.ChatID
			groupInfo = &groupSummaryDTO{
				ID:          group.ChatID,
				Name:        group.Name,
				AvatarURL:   group.Avatar,
				MemberCount: group.MemberCount,
			}
			if title == "" {
				title = group.Name
			}
			avatar = group.Avatar
		}
		if memberUIDs, err := h.groups.GetMemberUIDs(ctx, conv.ChatID); err == nil {
			participants = append(participants, memberUIDs...)
		}
	}

	if conv.ChatType == types.ChatTypeSingle {
		peerID := findPeerID(participants, currentUserID)
		if peerID != "" && h.users != nil {
			if user, err := h.users.FindByID(ctx, peerID); err == nil {
				dto := userDTOFromModel(user)
				peer = &dto
				if title == "" {
					title = dto.Nickname
				}
				if avatar == "" {
					avatar = dto.Avatar
				}
			}
		}
	}

	lastActivityAt := conv.UpdatedAt
	dto := conversationDTO{
		ID:               conv.ConversationID,
		ConversationID:   conv.ConversationID,
		ChatID:           conv.ChatID,
		ChatType:         conv.ChatType,
		Title:            title,
		Avatar:           avatar,
		Participants:     participants,
		PeerUser:         peer,
		Group:            groupInfo,
		GroupID:          groupID,
		GroupInfo:        groupInfo,
		LastMessage:      conv.LastMsg,
		LastReadSequence: conv.LastReadSeq,
		LastReadAt:       formatOptionalTime(conv.LastReadAt, time.Time{}),
		Muted:            conv.IsMuted,
		Pinned:           conv.IsTop,
		CreatedAt:        conv.CreatedAt.Format(timeFormatRFC3339Nano),
		UpdatedAt:        conv.UpdatedAt.Format(timeFormatRFC3339Nano),
		LastActivityAt:   lastActivityAt.Format(timeFormatRFC3339Nano),
	}

	return dto
}

type groupSummaryDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	MemberCount int    `json:"member_count"`
}

func findPeerID(participants []string, currentUserID string) string {
	for _, id := range participants {
		if id != "" && id != currentUserID {
			return id
		}
	}
	return ""
}

const timeFormatRFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"

func formatOptionalTime(value *time.Time, fallback time.Time) string {
	if value != nil && !value.IsZero() {
		return value.Format(timeFormatRFC3339Nano)
	}
	if fallback.IsZero() {
		return ""
	}
	return fallback.Format(timeFormatRFC3339Nano)
}
