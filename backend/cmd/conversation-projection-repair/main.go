package main

import (
	"context"
	"flag"
	"log"

	"d-im/internal/conversation/projector"
	conversationRepair "d-im/internal/conversation/repair"
	conversationRepo "d-im/internal/conversation/repository"
	"d-im/pkg/config"
	"d-im/pkg/mongodb"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "config file path")
	chatID := flag.String("chat-id", "", "only repair this chat; empty repairs all chats")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	db, err := mongodb.NewClient(ctx, mongodb.Config{URI: cfg.MongoDB.URI, Database: cfg.MongoDB.Database, PoolSize: cfg.MongoDB.PoolSize, Timeout: cfg.MongoDB.Timeout})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}
	repairer := conversationRepair.NewRepairer(db, projector.NewConversationProjector(conversationRepo.NewConversationRepo(db)))
	if *chatID == "" {
		err = repairer.RepairAll(ctx)
	} else {
		err = repairer.RepairChat(ctx, *chatID)
	}
	if err != nil {
		log.Fatalf("repair conversation projections: %v", err)
	}
	log.Println("conversation projections repaired")
}
