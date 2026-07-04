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

	"d-im/internal/connector/ws"
	"d-im/pkg/config"
	"d-im/pkg/crypto"
)

func main() {
	configPath := flag.String("config", "configs/config.dev.yaml", "config file path")
	flag.Parse()

	// 1. 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 2. JWT验证器
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

	// 3. WebSocket服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Connector.WSPort)
	server := ws.NewServer(addr, func(token string) (string, error) {
		return jwtMgr.Verify(token)
	})

	// 4. 设置消息处理器
	server.GetManager().SetMessageHandler(ws.DefaultHandler)

	// 5. 启动
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := server.Start(ctx); err != nil {
			log.Fatalf("[connector] %v", err)
		}
	}()

	log.Printf("[connector] started on :%d", cfg.Server.Connector.WSPort)

	// 6. 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[connector] shutting down...")
	cancel()
}
