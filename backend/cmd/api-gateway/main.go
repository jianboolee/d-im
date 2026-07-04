package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"d-im/internal/gateway"
	"d-im/internal/gateway/handler"
	"d-im/internal/gateway/router"
	"d-im/internal/message/repository"
	messageSvc "d-im/internal/message/service"
	"d-im/pkg/crypto"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/snowflake"
)

func main() {
	// 1. 初始化 MongoDB
	ctx := context.Background()
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		Database: getEnv("MONGO_DB", "im_db"),
		PoolSize: 100,
		Timeout:  10,
	})
	if err != nil {
		log.Fatalf("mongodb connect: %v", err)
	}

	// 2. 初始化雪花ID
	idGen, err := snowflake.NewGenerator(snowflake.Config{
		WorkerID:     1,
		DatacenterID: 1,
	})
	if err != nil {
		log.Fatalf("snowflake init: %v", err)
	}

	// 3. 初始化 JWT
	jwtMgr := crypto.NewJWTManager(
		getEnv("JWT_SECRET", "im-secret-key-change-me"),
		24*time.Hour,
	)

	// 4. 初始化各层依赖
	chatMgr := model.NewChatIDManager(db)
	msgRepo := repository.NewMessageRepo(db)
	msgSvc := messageSvc.NewMessageService(msgRepo, idGen, chatMgr)

	// 5. 初始化 HTTP
	authHandler := handler.NewAuthHandler(jwtMgr)
	messageHandler := handler.NewMessageHandler(msgSvc)
	httpHandler := router.NewRouter(jwtMgr, authHandler, messageHandler)

	server := gateway.NewServer(gateway.Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		GRPCPort: getEnv("GRPC_PORT", "9080"),
	}, httpHandler)

	// 6. 启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[gateway] %v", err)
		}
	}()

	log.Println("[api-gateway] started")

	// 7. 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[api-gateway] shutting down...")
	cancel()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
