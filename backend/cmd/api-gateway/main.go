package main

import (
	"context"
	"flag"
	"fmt"
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
	"d-im/pkg/config"
	"d-im/pkg/crypto"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/snowflake"
)

func main() {
	configPath := flag.String("config", "configs/config.dev.yaml", "config file path")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 2. 初始化 MongoDB
	ctx := context.Background()
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	// 3. 初始化雪花ID
	idGen, err := snowflake.NewGenerator(snowflake.Config{
		WorkerID:     cfg.Snowflake.WorkerID,
		DatacenterID: cfg.Snowflake.DatacenterID,
	})
	if err != nil {
		log.Fatalf("snowflake: %v", err)
	}

	// 4. 初始化 JWT
	accessExpire, err := time.ParseDuration(cfg.JWT.AccessExpire)
	if err != nil {
		log.Fatalf("jwt access_expire: %v", err)
	}
	refreshExpire, err := time.ParseDuration(cfg.JWT.RefreshExpire)
	if err != nil {
		log.Fatalf("jwt refresh_expire: %v", err)
	}
	ticketExpire, err := time.ParseDuration(cfg.JWT.TicketExpire)
	if err != nil {
		log.Fatalf("jwt ticket_expire: %v", err)
	}
	jwtMgr := crypto.NewJWTManager(cfg.JWT.Secret, accessExpire, refreshExpire, ticketExpire, cfg.JWT.APIKey)

	// 5. 初始化各层依赖
	chatMgr := model.NewChatIDManager(db)
	msgRepo := repository.NewMessageRepo(db)
	msgSvc := messageSvc.NewMessageService(msgRepo, idGen, chatMgr)

	// 6. 初始化 HTTP
	authHandler := handler.NewAuthHandler(jwtMgr)
	messageHandler := handler.NewMessageHandler(msgSvc)
	httpHandler := router.NewRouter(jwtMgr, authHandler, messageHandler)

	server := gateway.NewServer(gateway.Config{
		HTTPPort: itoa(cfg.Server.Gateway.HTTPPort),
		GRPCPort: itoa(cfg.Server.Gateway.GRPCPort),
	}, httpHandler)

	// 7. 启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[gateway] %v", err)
		}
	}()

	log.Printf("[api-gateway] started on :%d", cfg.Server.Gateway.HTTPPort)

	// 8. 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[api-gateway] shutting down...")
	cancel()
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
