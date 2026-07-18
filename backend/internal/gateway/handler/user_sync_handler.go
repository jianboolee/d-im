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

	"d-im/internal/gateway/httpapi"
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

// UserSyncHandler 处理业务系统向 IM 同步用户快照的管理面 API。
type UserSyncHandler struct {
	jwtMgr   *crypto.JWTManager
	userRepo userRepo
}

func NewUserSyncHandler(jwtMgr *crypto.JWTManager, uRepo userRepo) *UserSyncHandler {
	return &UserSyncHandler{jwtMgr: jwtMgr, userRepo: uRepo}
}

func (h *UserSyncHandler) authenticate(w http.ResponseWriter, r *http.Request) bool {
	apiKey := r.Header.Get("X-API-Key")
	if !h.jwtMgr.VerifyAPIKey(apiKey) {
		httpapi.WriteError(w, http.StatusForbidden, httpapi.Error{Code: httpapi.CodeInvalidAPIKey, Message: "invalid api key"})
		return false
	}
	return true
}

// PutUserSnapshot PUT /api/v1/management/users/{userID}
func (h *UserSyncHandler) PutUserSnapshot(w http.ResponseWriter, r *http.Request) {
	if !h.authenticate(w, r) {
		return
	}

	userID := strings.TrimSpace(r.PathValue("userID"))
	if userID == "" {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserIDRequired, Message: "user ID is required"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var snapshot userSnapshotRequest
	if err := decoder.Decode(&snapshot); err != nil {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserSnapshotInvalid, Message: "invalid user snapshot"})
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserSnapshotInvalid, Message: "invalid user snapshot"})
		return
	}
	if snapshot.Version <= 0 {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserVersionInvalid, Message: "version must be positive"})
		return
	}
	if snapshot.Status != "active" && snapshot.Status != "disabled" {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserStatusInvalid, Message: "status must be active or disabled"})
		return
	}
	if len(snapshot.Nickname) > 200 || len(snapshot.Avatar) > 2048 {
		httpapi.WriteError(w, http.StatusBadRequest, httpapi.Error{Code: httpapi.CodeUserFieldTooLong, Message: "user snapshot field too long"})
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
			httpapi.WriteError(w, http.StatusConflict, httpapi.Error{Code: httpapi.CodeUserVersionStale, Message: "stale user version"})
			return
		}
		log.Printf("[sdk] upsert user snapshot failed: uid=%s version=%d err=%v", userID, snapshot.Version, err)
		httpapi.WriteError(w, http.StatusInternalServerError, httpapi.Error{Code: httpapi.CodeUserSnapshotWriteFailed, Message: "upsert user snapshot failed"})
		return
	}
	log.Printf("[sdk] user snapshot synced: uid=%s version=%d", userID, snapshot.Version)
	httpapi.WriteSuccess(w, http.StatusOK, map[string]interface{}{"user_id": userID, "version": snapshot.Version})
}
