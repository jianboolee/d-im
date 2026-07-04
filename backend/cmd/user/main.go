package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"d-im/internal/user/repository"
	"d-im/internal/user/service"
	"d-im/pkg/config"
	"d-im/pkg/mongodb"
	natsq "d-im/pkg/queue/nats"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// MongoDB
	db, err := mongodb.NewClient(context.Background(), mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	// NATS
	natsTimeout, _ := time.ParseDuration(cfg.NATS.PublishTimeout)
	natsPub, err := natsq.NewPublisher(natsq.Config{
		URL:            cfg.NATS.URL,
		User:           cfg.NATS.User,
		Password:       cfg.NATS.Password,
		PublishTimeout: natsTimeout,
	})
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer natsPub.Close()

	// User service
	userRepo := repository.NewUserRepo(db)
	userSvc := service.NewUserService(userRepo)

	// 订阅事件总线
	conn := natsPub.GetConn()
	if err := userSvc.SubscribeEvents(conn); err != nil {
		log.Fatalf("subscribe events: %v", err)
	}

	log.Println("[user] service started, listening for user events")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[user] shutting down...")
}
