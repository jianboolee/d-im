package middleware

import (
	"log"
	"net/http"
	"time"
)

// LoggerMiddleware HTTP请求日志中间件
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter 以捕获状态码
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		uid := GetUserID(r.Context())
		log.Printf("[http] %s %s %d %s uid=%s ip=%s",
			r.Method, r.URL.Path, lrw.statusCode, duration, uid, r.RemoteAddr)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
