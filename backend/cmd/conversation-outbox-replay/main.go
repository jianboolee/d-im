package main

import (
	"context"
	"flag"
	"log"

	conversationOutbox "d-im/internal/conversation/outbox"
	"d-im/pkg/config"
	"d-im/pkg/mongodb"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	chatID := flag.String("chat-id", "", "only replay events for this chat; empty replays all retained events")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI: cfg.MongoDB.URI, Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize, Timeout: cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	repo := conversationOutbox.NewRepository(db)
	if err := repo.Replay(ctx, *chatID); err != nil {
		log.Fatalf("replay: %v", err)
	}
	if *chatID == "" {
		log.Println("all retained conversation projection events queued for replay")
	} else {
		log.Printf("conversation projection events queued for replay: chat_id=%s", *chatID)
	}
}
