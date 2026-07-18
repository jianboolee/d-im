package middleware

import (
	"context"
	"net/http"
	"strings"

	"d-im/internal/gateway/httpapi"
	"d-im/pkg/crypto"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthMiddleware JWT认证中间件
func AuthMiddleware(jwtMgr *crypto.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httpapi.WriteError(w, http.StatusUnauthorized, httpapi.Error{Code: httpapi.CodeUnauthorized, Message: "missing authorization header"})
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				httpapi.WriteError(w, http.StatusUnauthorized, httpapi.Error{Code: httpapi.CodeUnauthorized, Message: "invalid authorization format"})
				return
			}

			uid, err := jwtMgr.Verify(parts[1])
			if err != nil {
				httpapi.WriteError(w, http.StatusUnauthorized, httpapi.Error{Code: httpapi.CodeUnauthorized, Message: "invalid or expired token"})
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID 从 context 中获取用户ID
func GetUserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}
