package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"d-im/internal/gateway/httpapi"
)

// RecoveryMiddleware panic恢复中间件
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[panic] %v\n%s", err, debug.Stack())
				httpapi.WriteError(w, http.StatusInternalServerError, httpapi.Error{Code: httpapi.CodeInternal, Message: "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
