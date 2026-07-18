package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"d-im/pkg/model"

	"d-im/pkg/crypto"
)

type syncUserRequest struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`
	Status   string `json:"status,omitempty"`
}

type syncUserResponse struct {
	Status string `json:"status"`
}

type userRepo interface {
	Upsert(ctx context.Context, user *model.User) error
	BatchUpsert(ctx context.Context, users []*model.User) error
}

// SDKHandler 业务 SDK HTTP 处理器
type SDKHandler struct {
	jwtMgr   *crypto.JWTManager
	userRepo userRepo
}

func NewSDKHandler(jwtMgr *crypto.JWTManager, uRepo userRepo) *SDKHandler {
	return &SDKHandler{jwtMgr: jwtMgr, userRepo: uRepo}
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

	var user syncUserRequest
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	u := &model.User{ID: user.UserID, Nickname: user.Nickname, Avatar: user.Avatar, Status: user.Status, UpdatedAt: time.Now()}
	if err := h.userRepo.Upsert(r.Context(), u); err != nil {
		log.Printf("[sdk] upsert user failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "upsert failed"})
		return
	}
	log.Printf("[sdk] user synced: uid=%s", user.UserID)
	writeJSON(w, http.StatusOK, syncUserResponse{Status: "ok"})
}

// BatchSyncUsers POST /api/v1/sdk/user/batch-sync
func (h *SDKHandler) BatchSyncUsers(w http.ResponseWriter, r *http.Request) {
	if !h.auth(w, r) {
		return
	}

	var req struct {
		Users []syncUserRequest `json:"users"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}

	now := time.Now()
	users := make([]*model.User, len(req.Users))
	for i, u := range req.Users {
		users[i] = &model.User{ID: u.UserID, Nickname: u.Nickname, Avatar: u.Avatar, Status: u.Status, UpdatedAt: now}
	}
	if err := h.userRepo.BatchUpsert(r.Context(), users); err != nil {
		log.Printf("[sdk] batch upsert failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "batch upsert failed"})
		return
	}
	log.Printf("[sdk] batch synced %d users", len(users))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
