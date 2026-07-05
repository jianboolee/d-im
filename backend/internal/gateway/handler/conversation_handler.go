package handler

import (
	"encoding/json"
	"net/http"

	"d-im/internal/conversation/service"
	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"
)

// ConversationHandler 会话HTTP处理器
type ConversationHandler struct {
	convSvc *service.ConversationService
}

// NewConversationHandler 创建会话处理器
func NewConversationHandler(convSvc *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{convSvc: convSvc}
}

// ListConversations 获取用户会话列表
// GET /api/v1/conversation/list
func (h *ConversationHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	list, err := h.convSvc.GetList(r.Context(), uid, 100, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if list == nil {
		list = []*model.Conversation{}
	}
	writeJSON(w, http.StatusOK, list)
}

// ReadConversation 标记已读
// POST /api/v1/conversation/read
// Body: {"chat_id": "xxx"}
func (h *ConversationHandler) ReadConversation(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		ChatID string `json:"chat_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if err := h.convSvc.ReadMessage(r.Context(), uid, req.ChatID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
