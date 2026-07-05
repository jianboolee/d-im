package handler

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"d-im/pkg/crypto"
)

// AuthHandler Token签发处理器
type AuthHandler struct {
	jwtMgr      *crypto.JWTManager
	frontendURL string
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtMgr *crypto.JWTManager, frontendURL string) *AuthHandler {
	return &AuthHandler{jwtMgr: jwtMgr, frontendURL: frontendURL}
}

// IssueTicket 业务系统调用：签发一次性 ticket
// POST /api/v1/auth/ticket
// Header: X-API-Key: <api_key>
// Body: {"uid": "xxx"}
func (h *AuthHandler) IssueTicket(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if !h.jwtMgr.VerifyAPIKey(apiKey) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "invalid api key"})
		return
	}

	var req struct {
		UID string `json:"uid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.UID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uid is required"})
		return
	}

	ticket, err := h.jwtMgr.IssueTicket(req.UID)
	if err != nil {
		log.Printf("[auth] issue ticket failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue ticket failed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ticket":       ticket,
		"redirect_url": h.frontendURL + "/im/enter?ticket=" + ticket,
	})
}

// ExchangeToken 前端用 ticket 换取 access_token + refresh_token
// POST /api/v1/auth/token
// Body: {"ticket": "xxx", "device_id": "web_chrome_abc"}
func (h *AuthHandler) ExchangeToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Ticket   string `json:"ticket"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.DeviceID == "" {
		req.DeviceID = "unknown"
	}

	uid, err := h.jwtMgr.VerifyAs(req.Ticket, crypto.TokenTypeTicket)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired ticket"})
		return
	}

	accessToken, err := h.jwtMgr.IssueAccessToken(uid, req.DeviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	refreshToken, err := h.jwtMgr.IssueRefreshToken(uid, req.DeviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	log.Printf("[auth] token exchanged: uid=%s device=%s", uid, req.DeviceID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900, // 15 minutes
	})
}

// RefreshToken 刷新 access_token
// POST /api/v1/auth/refresh
// Header: Authorization: Bearer <refresh_token>
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "missing token"})
		return
	}

	uid, err := h.jwtMgr.VerifyAs(token, crypto.TokenTypeRefresh)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid or expired refresh token"})
		return
	}

	// 从 refresh_token 中提取 device_id
	deviceID := extractClaim(token, "device_id")
	if deviceID == "" {
		deviceID = "unknown"
	}

	accessToken, err := h.jwtMgr.IssueAccessToken(uid, deviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	refreshToken, err := h.jwtMgr.IssueRefreshToken(uid, deviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	log.Printf("[auth] token refreshed: uid=%s device=%s", uid, deviceID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
	})
}

func extractClaim(tokenStr, key string) string {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

// Login 开发模式登录：uid + device_id 直接签发 token pair
// POST /api/v1/auth/login
// Body: {"uid": "xxx", "device_id": "web_chrome_v1"}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UID      string `json:"uid"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.UID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "uid is required"})
		return
	}
	if req.DeviceID == "" {
		req.DeviceID = "unknown"
	}

	accessToken, err := h.jwtMgr.IssueAccessToken(req.UID, req.DeviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}
	refreshToken, err := h.jwtMgr.IssueRefreshToken(req.UID, req.DeviceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "issue token failed"})
		return
	}

	log.Printf("[auth] login: uid=%s device=%s", req.UID, req.DeviceID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// Logout 登出（前端可调用，清除本地 token）
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 无状态 JWT 无法服务端吊销，前端自行清除即可
	// TODO: 可在此将 refresh_token 加入 Redis 黑名单
	log.Printf("[auth] logout: uid=%s", r.Header.Get("X-User-ID"))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
