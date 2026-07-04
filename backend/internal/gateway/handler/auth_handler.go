package handler

import (
	"encoding/json"
	"net/http"

	"d-im/pkg/crypto"
)

// AuthHandler Token签发处理器
type AuthHandler struct {
	jwtMgr *crypto.JWTManager
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtMgr *crypto.JWTManager) *AuthHandler {
	return &AuthHandler{jwtMgr: jwtMgr}
}

// IssueToken 签发Token
// POST /api/v1/auth/token
// Body: {"uid": "xxx"}
func (h *AuthHandler) IssueToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.UID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uid is required"})
		return
	}

	token, err := h.jwtMgr.Issue(req.UID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":      token,
		"token_type": "Bearer",
	})
}
