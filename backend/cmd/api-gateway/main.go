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
	natsq "d-im/pkg/queue/nats"
	"d-im/pkg/snowflake"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	// MongoDB
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	// 雪花ID
	idGen, err := snowflake.NewGenerator(snowflake.Config{
		WorkerID:     cfg.Snowflake.WorkerID,
		DatacenterID: cfg.Snowflake.DatacenterID,
	})
	if err != nil {
		log.Fatalf("snowflake: %v", err)
	}

	// JWT
	accessExpire, _ := time.ParseDuration(cfg.JWT.AccessExpire)
	refreshExpire, _ := time.ParseDuration(cfg.JWT.RefreshExpire)
	ticketExpire, _ := time.ParseDuration(cfg.JWT.TicketExpire)
	jwtMgr := crypto.NewJWTManager(cfg.JWT.Secret, accessExpire, refreshExpire, ticketExpire, cfg.JWT.APIKey)

	// NATS
	natsTimeout, _ := time.ParseDuration(cfg.NATS.PublishTimeout)
	natsPub, err := natsq.NewPublisher(natsq.Config{
		URL:            cfg.NATS.URL,
		User:           cfg.NATS.User,
		Password:       cfg.NATS.Password,
		PublishTimeout: natsTimeout,
		Subjects: natsq.Subjects{
			MessageSend:  cfg.NATS.Subjects.MessageSend,
			MessagePush:  cfg.NATS.Subjects.MessagePush,
			MessageEvent: cfg.NATS.Subjects.MessageEvent,
		},
	})
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer natsPub.Close()

	// 依赖注入
	chatMgr := model.NewChatIDManager(db)
	msgRepo := repository.NewMessageRepo(db)
	msgSvc := messageSvc.NewMessageService(msgRepo, idGen, chatMgr, natsPub)

	// HTTP
	authHandler := handler.NewAuthHandler(jwtMgr)
	messageHandler := handler.NewMessageHandler(msgSvc)
	httpHandler := router.NewRouter(jwtMgr, authHandler, messageHandler)

	server := gateway.NewServer(gateway.Config{
		HTTPPort: itoa(cfg.Server.Gateway.HTTPPort),
		GRPCPort: itoa(cfg.Server.Gateway.GRPCPort),
	}, httpHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[gateway] %v", err)
		}
	}()

	log.Printf("[api-gateway] started on :%d", cfg.Server.Gateway.HTTPPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[api-gateway] shutting down...")
	cancel()
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
