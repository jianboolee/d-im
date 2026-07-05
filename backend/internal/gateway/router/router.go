package router

import (
	"net/http"

	"d-im/internal/gateway/handler"
	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/crypto"
)

// NewRouter 创建HTTP路由
func NewRouter(
	jwtMgr *crypto.JWTManager,
	authHandler *handler.AuthHandler,
	messageHandler *handler.MessageHandler,
	convHandler *handler.ConversationHandler,
) http.Handler {
	mux := http.NewServeMux()

	// ---- 公开路由（不需要认证）----
	// 健康检查
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 业务系统内部接口：签发一次性 ticket（API Key 鉴权，在 handler 内完成）
	mux.HandleFunc("POST /api/v1/auth/ticket", authHandler.IssueTicket)

	// 前端公开接口：开发登录 / ticket 换取 token / 刷新 / 登出
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/token", authHandler.ExchangeToken)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.RefreshToken)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)

	// ---- 受保护的消息路由（需要 JWT access_token）----
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/message/send", messageHandler.SendMessage)
	protected.HandleFunc("POST /api/v1/message/recall", messageHandler.RecallMessage)
	protected.HandleFunc("POST /api/v1/message/forward", messageHandler.ForwardMessage)
	protected.HandleFunc("GET /api/v1/message/list", messageHandler.ListMessages)

	// 会话路由
	protected.HandleFunc("GET /api/v1/conversation/list", convHandler.ListConversations)
	protected.HandleFunc("POST /api/v1/conversation/read", convHandler.ReadConversation)

	// 中间件链: recovery -> logger -> auth -> ratelimit -> protected routes
	rateLimiter := middleware.NewRateLimiter(100, 200)
	protectedHandler := middleware.RecoveryMiddleware(protected)
	protectedHandler = middleware.LoggerMiddleware(protectedHandler)
	protectedHandler = middleware.AuthMiddleware(jwtMgr)(protectedHandler)
	protectedHandler = middleware.RateLimitMiddleware(rateLimiter)(protectedHandler)

	mux.Handle("/api/v1/message/", protectedHandler)
	mux.Handle("/api/v1/conversation/", protectedHandler)

	return mux
}
