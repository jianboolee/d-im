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
	chatHandler *handler.ChatHandler,
	convHandler *handler.ConversationHandler,
	groupHandler *handler.GroupHandler,
	uploadHandler *handler.UploadHandler,
	mediaStaticHandler http.Handler,
	userHandler *handler.UserHandler,
	userSyncHandler *handler.UserSyncHandler,
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
	if mediaStaticHandler != nil {
		mux.Handle("/media/", mediaStaticHandler)
	}

	// 管理面路由（API Key 鉴权，在 handler 内完成）
	mux.HandleFunc("PUT /api/v1/management/users/{userID}", userSyncHandler.PutUserSnapshot)

	// ---- 受保护的路由（需要 JWT access_token）----
	protected := http.NewServeMux()
	protected.HandleFunc("POST /api/v1/messages", messageHandler.SendMessage)
	protected.HandleFunc("POST /api/v1/message/recall", messageHandler.RecallMessage)
	protected.HandleFunc("POST /api/v1/message/forward", messageHandler.ForwardMessage)
	protected.HandleFunc("GET /api/v1/message/list", messageHandler.ListMessages)
	protected.HandleFunc("GET /api/v1/conversations", convHandler.ListConversations)
	protected.HandleFunc("POST /api/v1/chats/single", chatHandler.EnsureSingleChat)
	protected.HandleFunc("GET /api/v1/conversations/{id}", convHandler.GetConversation)
	protected.HandleFunc("POST /api/v1/conversations/{id}/read", convHandler.ReadConversation)
	protected.HandleFunc("PATCH /api/v1/conversations/{id}/settings", convHandler.UpdateConversationSettings)
	// 消息历史路由 — 按 conversation_id 或 chat_id 查询
	protected.HandleFunc("GET /api/v1/conversations/{id}/messages", messageHandler.ListConversationMessages)
	protected.HandleFunc("GET /api/v1/conversations/{id}/messages/search", messageHandler.SearchConversationMessages)
	protected.HandleFunc("GET /api/v1/chats/{id}/conversation", convHandler.GetConversationByChat)
	protected.HandleFunc("GET /api/v1/chats/{id}/messages", messageHandler.ListChatMessages)
	protected.HandleFunc("GET /api/v1/chats/{id}/messages/search", messageHandler.SearchChatMessages)
	protected.HandleFunc("GET /api/v1/groups", groupHandler.ListGroups)
	protected.HandleFunc("POST /api/v1/groups", groupHandler.CreateGroup)
	protected.HandleFunc("GET /api/v1/groups/{id}", groupHandler.GetGroup)
	protected.HandleFunc("PATCH /api/v1/groups/{id}", groupHandler.UpdateGroup)
	protected.HandleFunc("DELETE /api/v1/groups/{id}", groupHandler.DismissGroup)
	protected.HandleFunc("POST /api/v1/groups/{id}/join", groupHandler.JoinGroup)
	protected.HandleFunc("PATCH /api/v1/groups/{id}/settings", groupHandler.UpdateSettings)
	protected.HandleFunc("PUT /api/v1/groups/{id}/announcement", groupHandler.SetAnnouncement)
	protected.HandleFunc("POST /api/v1/groups/{id}/owner", groupHandler.TransferOwner)
	protected.HandleFunc("GET /api/v1/groups/{id}/members", groupHandler.ListMembers)
	protected.HandleFunc("POST /api/v1/groups/{id}/members", groupHandler.InviteMembers)
	protected.HandleFunc("DELETE /api/v1/groups/{id}/members/{uid}", groupHandler.KickMember)
	protected.HandleFunc("PATCH /api/v1/groups/{id}/members/{uid}/role", groupHandler.SetMemberRole)
	protected.HandleFunc("POST /api/v1/groups/{id}/leave", groupHandler.LeaveGroup)
	protected.HandleFunc("POST /api/v1/uploads/image", uploadHandler.UploadImage)
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
	mux.Handle("/api/v1/chats", protectedHandler)
	mux.Handle("/api/v1/chats/", protectedHandler)
	mux.Handle("/api/v1/groups", protectedHandler)
	mux.Handle("/api/v1/groups/", protectedHandler)
	mux.Handle("/api/v1/uploads/", protectedHandler)
	mux.Handle("/api/v1/users/", protectedHandler)

	return mux
}
