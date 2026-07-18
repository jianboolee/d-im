package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	chatSvc "d-im/internal/chat/service"
	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"
	"d-im/pkg/types"
)

type ChatHandler struct {
	chats *chatSvc.ChatService
}

func NewChatHandler(chats *chatSvc.ChatService) *ChatHandler {
	return &ChatHandler{chats: chats}
}

type chatDTO struct {
	ChatID      string         `json:"chat_id"`
	ChatType    types.ChatType `json:"chat_type"`
	MemberIDs   []string       `json:"member_user_ids"`
	MemberCount int            `json:"member_count"`
	CreatedAt   time.Time      `json:"created_at"`
}

// EnsureSingleChat creates or returns the Chat entity shared by two users.
// POST /api/v1/chats/single
func (h *ChatHandler) EnsureSingleChat(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, 401001, "unauthorized")
		return
	}

	var req struct {
		PeerUserID string `json:"peer_user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.PeerUserID == "" {
		writeError(w, http.StatusBadRequest, 400006, "peer_user_id is required")
		return
	}

	chat, err := h.chats.EnsureSingleChat(r.Context(), uid, req.PeerUserID)
	if err != nil {
		if errors.Is(err, model.ErrSingleChatWithSelf) || errors.Is(err, model.ErrUserIDRequired) {
			writeError(w, http.StatusBadRequest, 400007, err.Error())
			return
		}
		log.Printf("[chat_handler] ensure single chat failed: user_id=%s peer_user_id=%s err=%v", uid, req.PeerUserID, err)
		writeError(w, http.StatusInternalServerError, 500201, "ensure single chat failed")
		return
	}

	writeSuccess(w, chatDTO{
		ChatID: chat.ChatID, ChatType: chat.ChatType, MemberIDs: chat.Members,
		MemberCount: chat.MemberCount, CreatedAt: chat.CreatedAt,
	})
}
