package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	chatRepo "d-im/internal/chat/repository"
	"d-im/internal/message/dispatcher"
	"d-im/internal/message/repository"
	"d-im/internal/message/service"
	"d-im/pkg/config"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	natsq "d-im/pkg/queue/nats"

	"github.com/nats-io/nats.go"
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
			MessageSend:  cfg.NATS.Subjects.MessageSend,
			MessagePush:  cfg.NATS.Subjects.MessagePush,
			MessageEvent: cfg.NATS.Subjects.MessageEvent,
		},
	})
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer natsPub.Close()

	// 初始化
	chatR := chatRepo.NewChatRepo(db)
	msgRepo := repository.NewMessageRepo(db)
	convMgr := model.NewConversationManager(db)
	msgSvc := service.NewMessageService(msgRepo, chatR, convMgr, natsPub)

	// 分发器
	d := dispatcher.NewDispatcher(msgRepo, 4)
	d.Start(ctx)

	// 订阅 im.message.send，消费用户发送的消息 + 群系统事件
	conn := natsPub.GetConn()
	_, err = conn.Subscribe("im.message.send", func(msg *nats.Msg) {
		var req service.SendMessageReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[message] unmarshal send request failed: %v", err)
			return
		}
		_, err := msgSvc.Send(context.Background(), &req)
		if err != nil {
			log.Printf("[message] send failed: chat=%s sender=%s type=%s err=%v", req.ChatID, req.SenderID, req.MsgType, err)
		}
	})
	if err != nil {
		log.Fatalf("[message] subscribe im.message.send: %v", err)
	}
	log.Printf("[message] started, subscribed to im.message.send")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[message] shutting down...")
	d.Stop()
}
