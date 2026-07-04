package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"d-im/internal/connector/ws"
	"d-im/pkg/config"
	"d-im/pkg/crypto"
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

	// JWT
	accessExpire, _ := time.ParseDuration(cfg.JWT.AccessExpire)
	refreshExpire, _ := time.ParseDuration(cfg.JWT.RefreshExpire)
	ticketExpire, _ := time.ParseDuration(cfg.JWT.TicketExpire)
	jwtMgr := crypto.NewJWTManager(cfg.JWT.Secret, accessExpire, refreshExpire, ticketExpire, cfg.JWT.APIKey)

	// NATS
	natsPub, err := natsq.NewPublisher(natsq.Config{
		URL: cfg.NATS.URL,
		Subjects: natsq.Subjects{
			MessagePush: cfg.NATS.Subjects.MessagePush,
		},
	})
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer natsPub.Close()

	// WebSocket server
	addr := fmt.Sprintf(":%d", cfg.Server.Connector.WSPort)
	server := ws.NewServer(addr, func(token string) (string, error) {
		return jwtMgr.Verify(token)
	})
	server.GetManager().SetMessageHandler(ws.DefaultHandler)

	// 订阅 NATS 推送事件，转发到 WebSocket 客户端
	conn := natsPub.GetConn()
	conn.QueueSubscribe("im.push.message.>", "connector-push", func(msg *nats.Msg) {
		parts := strings.Split(msg.Subject, ".")
		if len(parts) != 4 {
			return
		}
		targetUID := parts[3]
		count := server.GetManager().SendToUser(targetUID, msg.Data)
		if count > 0 {
			log.Printf("[connector] pushed to uid=%s, devices=%d", targetUID, count)
		} else {
			// 用户不在线，发布离线推送事件
			offlineSubject := "im.push.offline." + targetUID
			if err := natsPub.Publish(offlineSubject, msg.Data); err != nil {
				log.Printf("[connector] offline push publish failed: uid=%s, err=%v", targetUID, err)
			} else {
				log.Printf("[connector] offline push published: uid=%s", targetUID)
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[connector] %v", err)
		}
	}()

	log.Printf("[connector] started on :%d", cfg.Server.Connector.WSPort)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[connector] shutting down...")
	cancel()
}
