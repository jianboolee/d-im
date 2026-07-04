package router

import (
	"net/http"

	"d-im/internal/gateway/handler"
	"d-im/internal/gateway/handler/middleware"
)

// NewRouter 创建HTTP路由
func NewRouter(messageHandler *handler.MessageHandler) http.Handler {
	mux := http.NewServeMux()

	// 消息相关路由
	// POST /api/v1/message/send
	mux.HandleFunc("POST /api/v1/message/send", messageHandler.SendMessage)
	// POST /api/v1/message/recall
	mux.HandleFunc("POST /api/v1/message/recall", messageHandler.RecallMessage)
	// POST /api/v1/message/forward
	mux.HandleFunc("POST /api/v1/message/forward", messageHandler.ForwardMessage)
	// GET /api/v1/message/list
	mux.HandleFunc("GET /api/v1/message/list", messageHandler.ListMessages)

	// 健康检查
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 中间件链: recovery -> logger -> auth -> ratelimit -> router
	rateLimiter := middleware.NewRateLimiter(100, 200)

	handler := middleware.RecoveryMiddleware(mux)
	handler = middleware.LoggerMiddleware(handler)
	handler = middleware.AuthMiddleware(handler)
	handler = middleware.RateLimitMiddleware(rateLimiter)(handler)

	return handler
}
