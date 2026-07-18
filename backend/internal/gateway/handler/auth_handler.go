package handler

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"d-im/pkg/crypto"
)

// AuthHandler Token签发处理器
type AuthHandler struct {
	jwtMgr        *crypto.JWTManager
	frontendURL   string
	superPassword string
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtMgr *crypto.JWTManager, frontendURL string, superPassword string) *AuthHandler {
	return &AuthHandler{jwtMgr: jwtMgr, frontendURL: frontendURL, superPassword: superPassword}
}

// IssueTicket 签发一次性 ticket 或用 ticket 兑换 token
// POST /api/v1/auth/ticket
// 签发: Header: X-API-Key: <api_key>, Body: {"id": "xxx"}
// 兑换: Body: {"ticket": "xxx", "device_id": "web_chrome_abc"}
func (h *AuthHandler) IssueTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Ticket   string `json:"ticket"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}

	if req.Ticket != "" {
		h.exchangeTicket(w, req.Ticket, req.DeviceID)
		return
	}

	apiKey := r.Header.Get("X-API-Key")
	if !h.jwtMgr.VerifyAPIKey(apiKey) {
		writeError(w, http.StatusForbidden, 403001, "invalid api key")
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, 400002, "id is required")
		return
	}

	ticket, err := h.jwtMgr.IssueTicket(req.ID)
	if err != nil {
		log.Printf("[auth] issue ticket failed: %v", err)
		writeError(w, http.StatusInternalServerError, 500001, "issue ticket failed")
		return
	}

	writeSuccess(w, map[string]interface{}{
		"ticket":       ticket,
		"redirect_url": h.frontendURL + "/im/enter?ticket=" + ticket,
	})
}

func (h *AuthHandler) exchangeTicket(w http.ResponseWriter, ticket string, deviceID string) {
	if deviceID == "" {
		deviceID = "unknown"
	}

	uid, err := h.jwtMgr.VerifyAs(ticket, crypto.TokenTypeTicket)
	if err != nil {
		writeError(w, http.StatusUnauthorized, 401002, "invalid or expired ticket")
		return
	}

	tokenPair, err := h.issueTokenPair(uid, deviceID)
	if err != nil {
		log.Printf("[auth] exchange ticket failed: %v", err)
		writeError(w, http.StatusInternalServerError, 500002, "issue token failed")
		return
	}

	log.Printf("[auth] ticket exchanged: uid=%s device=%s", uid, deviceID)
	writeSuccess(w, tokenPair)
}

// CreateSession 业务系统用 API Key 为指定用户创建会话（签发 JWT）
// POST /api/v1/auth/session
// Header: X-API-Key: <api_key>
// Body: {"id": "xxx"}
func (h *AuthHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	apiKey := r.Header.Get("X-API-Key")
	if !h.jwtMgr.VerifyAPIKey(apiKey) {
		writeError(w, http.StatusForbidden, 403001, "invalid api key")
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	uid := req.ID
	if uid == "" {
		writeError(w, http.StatusBadRequest, 400002, "id is required")
		return
	}

	deviceID := "sdk_" + time.Now().Format("20060102150405")
	tokenPair, err := h.issueTokenPair(uid, deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500002, "issue token failed")
		return
	}

	log.Printf("[auth] session created: uid=%s", uid)
	writeSuccess(w, tokenPair)
}

// RefreshToken 刷新 access_token
// POST /api/v1/auth/refresh
// Header: Authorization: Bearer <refresh_token>
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, 401001, "missing token")
		return
	}

	uid, err := h.jwtMgr.VerifyAs(token, crypto.TokenTypeRefresh)
	if err != nil {
		writeError(w, http.StatusUnauthorized, 401003, "invalid or expired refresh token")
		return
	}

	// 从 refresh_token 中提取 device_id
	deviceID := extractClaim(token, "device_id")
	if deviceID == "" {
		deviceID = "unknown"
	}

	tokenPair, err := h.issueTokenPair(uid, deviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500002, "issue token failed")
		return
	}

	log.Printf("[auth] token refreshed: uid=%s device=%s", uid, deviceID)
	writeSuccess(w, tokenPair)
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

// Login 超级密码登录：id + password + device_id 签发 token pair
// POST /api/v1/auth/login
// Body: {"id": "xxx", "password": "super-password", "device_id": "web_chrome_v1"}
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       string `json:"id"`
		Password string `json:"password"`
		DeviceID string `json:"device_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, 400001, "invalid request")
		return
	}
	if req.ID == "" {
		writeError(w, http.StatusBadRequest, 400002, "id is required")
		return
	}
	if req.Password == "" {
		writeError(w, http.StatusBadRequest, 400003, "password is required")
		return
	}
	if h.superPassword == "" {
		writeError(w, http.StatusInternalServerError, 500003, "super password is not configured")
		return
	}
	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.superPassword)) != 1 {
		writeError(w, http.StatusUnauthorized, 401004, "invalid id or password")
		return
	}
	if req.DeviceID == "" {
		req.DeviceID = "unknown"
	}

	tokenPair, err := h.issueTokenPair(req.ID, req.DeviceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, 500002, "issue token failed")
		return
	}

	log.Printf("[auth] login: id=%s device=%s", req.ID, req.DeviceID)
	writeSuccess(w, tokenPair)
}

// Logout 登出（前端可调用，清除本地 token）
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// 无状态 JWT 无法服务端吊销，前端自行清除即可
	// TODO: 可在此将 refresh_token 加入 Redis 黑名单
	log.Printf("[auth] logout: uid=%s", r.Header.Get("X-User-ID"))
	writeSuccess(w, map[string]string{"status": "ok"})
}

func (h *AuthHandler) issueTokenPair(uid, deviceID string) (map[string]interface{}, error) {
	accessToken, err := h.jwtMgr.IssueAccessToken(uid, deviceID)
	if err != nil {
		return nil, err
	}
	refreshToken, err := h.jwtMgr.IssueRefreshToken(uid, deviceID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	}, nil
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}
