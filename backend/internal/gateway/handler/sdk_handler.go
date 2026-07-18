package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	userRepository "d-im/internal/user/repository"
	"d-im/pkg/crypto"
	"d-im/pkg/model"
)

type userSnapshotRequest struct {
	Nickname string                 `json:"nickname"`
	Avatar   string                 `json:"avatar_url"`
	Status   string                 `json:"status"`
	Version  int64                  `json:"version"`
	Ext      map[string]interface{} `json:"ext,omitempty"`
}

type userRepo interface {
	UpsertSnapshot(ctx context.Context, user *model.User) error
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

// PutUserSnapshot PUT /api/v1/sdk/users/{id}
func (h *SDKHandler) PutUserSnapshot(w http.ResponseWriter, r *http.Request) {
	if !h.auth(w, r) {
		return
	}

	userID := strings.TrimSpace(r.PathValue("id"))
	if userID == "" {
		writeAPIError(w, http.StatusBadRequest, 400101, "user id is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var snapshot userSnapshotRequest
	if err := decoder.Decode(&snapshot); err != nil {
		writeAPIError(w, http.StatusBadRequest, 400102, "invalid user snapshot")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeAPIError(w, http.StatusBadRequest, 400102, "invalid user snapshot")
		return
	}
	if snapshot.Version <= 0 {
		writeAPIError(w, http.StatusBadRequest, 400103, "version must be positive")
		return
	}
	if snapshot.Status != "active" && snapshot.Status != "disabled" {
		writeAPIError(w, http.StatusBadRequest, 400104, "status must be active or disabled")
		return
	}
	if len(snapshot.Nickname) > 200 || len(snapshot.Avatar) > 2048 {
		writeAPIError(w, http.StatusBadRequest, 400105, "user snapshot field too long")
		return
	}

	u := &model.User{
		ID:        userID,
		Nickname:  snapshot.Nickname,
		Avatar:    snapshot.Avatar,
		Status:    snapshot.Status,
		Version:   snapshot.Version,
		Ext:       snapshot.Ext,
		UpdatedAt: time.Now(),
	}
	if err := h.userRepo.UpsertSnapshot(r.Context(), u); err != nil {
		if errors.Is(err, userRepository.ErrStaleUserVersion) {
			writeAPIError(w, http.StatusConflict, 409101, "stale user version")
			return
		}
		log.Printf("[sdk] upsert user snapshot failed: uid=%s version=%d err=%v", userID, snapshot.Version, err)
		writeAPIError(w, http.StatusInternalServerError, 500101, "upsert user snapshot failed")
		return
	}
	log.Printf("[sdk] user snapshot synced: uid=%s version=%d", userID, snapshot.Version)
	writeAPISuccess(w, map[string]interface{}{"user_id": userID, "version": snapshot.Version})
}
