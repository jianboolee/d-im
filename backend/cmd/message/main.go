package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"d-im/internal/message/dispatcher"
	"d-im/internal/message/repository"
	"d-im/internal/message/service"
	"d-im/pkg/config"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	// 2. MongoDB
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	// 3. 初始化
	chatColl := model.ChatCollection(db)
	msgRepo := repository.NewMessageRepo(db)
	msgSvc := service.NewMessageService(msgRepo, chatColl, nil, nil)

	// 4. 启动分发器
	d := dispatcher.NewDispatcher(msgRepo, 4)
	d.Start(ctx)

	log.Printf("[message] started, msg_svc=%v", msgSvc != nil)

	// 5. 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[message] shutting down...")
	d.Stop()
}
