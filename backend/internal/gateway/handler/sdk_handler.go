package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"d-im/pkg/model"
	"d-im/pkg/sdk"
	"d-im/pkg/types"

	"d-im/pkg/crypto"
)

type userRepo interface {
	Upsert(ctx context.Context, user *model.User) error
	BatchUpsert(ctx context.Context, users []*model.User) error
}

type msgRepo interface {
	Insert(ctx context.Context, msg *model.Message) error
}

type chatMgr interface {
	CreateOrGetSingleChat(ctx context.Context, uid1, uid2 string) (*model.Chat, error)
}

// SDKHandler 业务 SDK HTTP 处理器
type SDKHandler struct {
	jwtMgr   *crypto.JWTManager
	userRepo userRepo
	msgRepo  msgRepo
	chatMgr  chatMgr
}

func NewSDKHandler(jwtMgr *crypto.JWTManager, uRepo userRepo, mRepo msgRepo, cMgr chatMgr) *SDKHandler {
	return &SDKHandler{jwtMgr: jwtMgr, userRepo: uRepo, msgRepo: mRepo, chatMgr: cMgr}
}

func (h *SDKHandler) auth(w http.ResponseWriter, r *http.Request) bool {
	apiKey := r.Header.Get("X-API-Key")
	if !h.jwtMgr.VerifyAPIKey(apiKey) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid api key"})
		return false
	}
	return true
}

// SyncUser POST /api/v1/sdk/user/sync
func (h *SDKHandler) SyncUser(w http.ResponseWriter, r *http.Request) {
	if !h.auth(w, r) {
		return
	}

	var user sdk.UserData
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	u := &model.User{
		ID:        user.UserID,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		UpdatedAt: time.Now(),
	}
	if err := h.userRepo.Upsert(r.Context(), u); err != nil {
		log.Printf("[sdk] upsert user failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert failed"})
		return
	}
	log.Printf("[sdk] user synced: uid=%s", user.UserID)
	writeJSON(w, http.StatusOK, sdk.SyncUserResp{Ok: "ok"})
}

// BatchSyncUsers POST /api/v1/sdk/user/batch-sync
func (h *SDKHandler) BatchSyncUsers(w http.ResponseWriter, r *http.Request) {
	if !h.auth(w, r) {
		return
	}

	var req struct {
		Users []sdk.UserData `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	now := time.Now()
	users := make([]*model.User, len(req.Users))
	for i, u := range req.Users {
		users[i] = &model.User{
			ID:        u.UserID,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
			Status:    u.Status,
			UpdatedAt: now,
		}
	}
	if err := h.userRepo.BatchUpsert(r.Context(), users); err != nil {
		log.Printf("[sdk] batch upsert failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "batch upsert failed"})
		return
	}
	log.Printf("[sdk] batch synced %d users", len(users))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// SendMessage POST /api/v1/sdk/message/send
func (h *SDKHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if !h.auth(w, r) {
		return
	}

	var req sdk.SendMessageReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	now := time.Now()
	msg := &model.Message{
		MsgID:      now.Format("20060102150405") + "000000",
		ChatID:     req.ChatID,
		ChatType:   types.ChatType(req.ChatType),
		FromUID:    req.FromUID,
		FromName:   req.FromName,
		MsgType:    types.MessageType(req.MsgType),
		Content:    req.Content,
		Status:     "sent",
		ClientTime: now,
		ServerTime: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.msgRepo.Insert(r.Context(), msg); err != nil {
		log.Printf("[sdk] insert message failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "insert failed"})
		return
	}
	log.Printf("[sdk] message sent: msg_id=%s from=%s to=%s", msg.MsgID, req.FromUID, req.ChatID)
	writeJSON(w, http.StatusOK, sdk.SendMessageResp{
		MsgID:      msg.MsgID,
		ServerTime: now.Format(time.RFC3339),
		Status:     "sent",
	})
}
