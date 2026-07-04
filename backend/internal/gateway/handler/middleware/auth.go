package middleware

import (
	"context"
	"net/http"
	"strings"

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
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			uid, err := jwtMgr.Verify(parts[1])
			if err != nil {
				http.Error(w, `{"error":"invalid or expired token"}`, http.StatusUnauthorized)
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
