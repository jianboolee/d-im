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
	sdkHandler *handler.SDKHandler,
) http.Handler {
	mux := http.NewServeMux()

	// ---- 公开路由（不需要认证）----
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("POST /api/v1/auth/ticket", authHandler.IssueTicket)
	mux.HandleFunc("POST /api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("POST /api/v1/auth/token", authHandler.ExchangeToken)
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.RefreshToken)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)

	// 业务 SDK 路由（API Key 鉴权，在 handler 内完成）
	mux.HandleFunc("POST /api/v1/sdk/user/sync", sdkHandler.SyncUser)
	mux.HandleFunc("POST /api/v1/sdk/user/batch-sync", sdkHandler.BatchSyncUsers)
	mux.HandleFunc("POST /api/v1/sdk/message/send", sdkHandler.SendMessage)

	// ---- 受保护的路由（需要 JWT access_token）----
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/message/send", messageHandler.SendMessage)
	protected.HandleFunc("POST /api/v1/message/recall", messageHandler.RecallMessage)
	protected.HandleFunc("POST /api/v1/message/forward", messageHandler.ForwardMessage)
	protected.HandleFunc("GET /api/v1/message/list", messageHandler.ListMessages)
	protected.HandleFunc("GET /api/v1/conversation/list", convHandler.ListConversations)
	protected.HandleFunc("POST /api/v1/conversation/read", convHandler.ReadConversation)

	rateLimiter := middleware.NewRateLimiter(100, 200)
	protectedHandler := middleware.RecoveryMiddleware(protected)
	protectedHandler = middleware.LoggerMiddleware(protectedHandler)
	protectedHandler = middleware.AuthMiddleware(jwtMgr)(protectedHandler)
	protectedHandler = middleware.RateLimitMiddleware(rateLimiter)(protectedHandler)

	mux.Handle("/api/v1/message/", protectedHandler)
	mux.Handle("/api/v1/conversation/", protectedHandler)

	return mux
}
