package middleware

import (
	"context"
	"encoding/json"
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
				writeAPIError(w, http.StatusUnauthorized, 401001, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeAPIError(w, http.StatusUnauthorized, 401001, "invalid authorization format")
				return
			}

			uid, err := jwtMgr.Verify(parts[1])
			if err != nil {
				writeAPIError(w, http.StatusUnauthorized, 401001, "invalid or expired token")
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

func writeAPIError(w http.ResponseWriter, status int, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code  int         `json:"code"`
		Data  interface{} `json:"data"`
		Error string      `json:"error"`
	}{
		Code:  code,
		Data:  nil,
		Error: message,
	})
}
