package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"d-im/internal/connector/ws"
	"d-im/pkg/crypto"
)

func main() {
	// 1. JWT验证器（与api-gateway共享同一secret）
	jwtMgr := crypto.NewJWTManager(
		getEnv("JWT_SECRET", "im-secret-key-change-me"),
		24*time.Hour,
	)

	// 2. 创建WebSocket服务器
	server := ws.NewServer(":"+getEnv("WS_PORT", "8081"), func(token string) (string, error) {
		return jwtMgr.Verify(token)
	})

	// 3. 设置默认消息处理器
	server.GetManager().SetMessageHandler(ws.DefaultHandler)

	// 4. 启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[connector] %v", err)
		}
	}()

	log.Printf("[connector] started on port %s", getEnv("WS_PORT", "8081"))

	// 5. 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[connector] shutting down...")
	cancel()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
