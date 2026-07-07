package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"d-im/internal/group/adapter"
	groupRepo "d-im/internal/group/repository"
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

	// NATS
	natsTimeout, _ := time.ParseDuration(cfg.NATS.PublishTimeout)
	natsPub, err := natsq.NewPublisher(natsq.Config{
		URL:            cfg.NATS.URL,
		User:           cfg.NATS.User,
		Password:       cfg.NATS.Password,
		PublishTimeout: natsTimeout,
		Subjects: natsq.Subjects{
			MessagePush: cfg.NATS.Subjects.MessagePush,
		},
	})
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer natsPub.Close()

	// Repositories
	gRepo := groupRepo.NewGroupRepo(db)
	mRepo := groupRepo.NewMemberRepo(db)

	// Push consumer: 订阅 dim.group.* 事件，转为 im.push.message.{uid}
	consumer := adapter.NewGroupPushConsumer(natsPub, gRepo, mRepo)
	if err := consumer.Start(natsPub.GetConn()); err != nil {
		log.Fatalf("start group push consumer: %v", err)
	}
	defer consumer.Stop()

	log.Printf("[group] started, push_consumer=running")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[group] shutting down...")
}
