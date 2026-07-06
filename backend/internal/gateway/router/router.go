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
	userHandler *handler.UserHandler,
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
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.RefreshToken)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/v1/auth/session", authHandler.CreateSession)

	// 业务 SDK 路由（API Key 鉴权，在 handler 内完成）
	mux.HandleFunc("POST /api/v1/sdk/user/sync", sdkHandler.SyncUser)
	mux.HandleFunc("POST /api/v1/sdk/user/batch-sync", sdkHandler.BatchSyncUsers)

	// ---- 受保护的路由（需要 JWT access_token）----
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/messages", messageHandler.SendMessage)
	protected.HandleFunc("POST /api/v1/message/recall", messageHandler.RecallMessage)
	protected.HandleFunc("POST /api/v1/message/forward", messageHandler.ForwardMessage)
	protected.HandleFunc("GET /api/v1/message/list", messageHandler.ListMessages)
	protected.HandleFunc("GET /api/v1/conversations", convHandler.ListConversations)
	protected.HandleFunc("POST /api/v1/conversations/single", convHandler.CreateSingleConversation)
	protected.HandleFunc("GET /api/v1/conversations/{id}", convHandler.GetConversation)
	protected.HandleFunc("POST /api/v1/conversations/{id}/read", convHandler.ReadConversation)
	protected.HandleFunc("PATCH /api/v1/conversations/{id}/settings", convHandler.UpdateConversationSettings)
	protected.HandleFunc("GET /api/v1/conversations/{id}/messages", messageHandler.ListConversationMessages)
	protected.HandleFunc("GET /api/v1/conversations/{id}/messages/search", messageHandler.SearchConversationMessages)
	protected.HandleFunc("GET /api/v1/users/me", userHandler.GetMe)
	protected.HandleFunc("GET /api/v1/users/{id}", userHandler.GetUser)

	rateLimiter := middleware.NewRateLimiter(100, 200)
	protectedHandler := middleware.RecoveryMiddleware(protected)
	protectedHandler = middleware.LoggerMiddleware(protectedHandler)
	protectedHandler = middleware.AuthMiddleware(jwtMgr)(protectedHandler)
	protectedHandler = middleware.RateLimitMiddleware(rateLimiter)(protectedHandler)

	mux.Handle("/api/v1/message/", protectedHandler)
	mux.Handle("/api/v1/messages", protectedHandler)
	mux.Handle("/api/v1/conversations", protectedHandler)
	mux.Handle("/api/v1/conversations/", protectedHandler)
	mux.Handle("/api/v1/users/", protectedHandler)

	return mux
}
