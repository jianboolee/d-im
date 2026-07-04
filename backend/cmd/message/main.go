package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"d-im/internal/message/dispatcher"
	"d-im/internal/message/repository"
	"d-im/internal/message/service"
	"d-im/pkg/model"
	"d-im/pkg/mongodb"
	"d-im/pkg/snowflake"
)

func main() {
	ctx := context.Background()

	// 1. MongoDB
	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		Database: getEnv("MONGO_DB", "im_db"),
		PoolSize: 50,
		Timeout:  10,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	// 2. 雪花ID
	idGen, err := snowflake.NewGenerator(snowflake.Config{
		WorkerID:     2,
		DatacenterID: 1,
	})
	if err != nil {
		log.Fatalf("snowflake: %v", err)
	}

	// 3. 初始化
	chatMgr := model.NewChatIDManager(db)
	msgRepo := repository.NewMessageRepo(db)
	msgSvc := service.NewMessageService(msgRepo, idGen, chatMgr)

	// 4. 启动分发器
	d := dispatcher.NewDispatcher(msgRepo, 4)
	d.Start(ctx)

	// 5. NATS订阅（通过扩展消息处理逻辑注入msgSvc）
	// TODO: 连接NATS，订阅 im.message.send 主题，调用 msgSvc.Send()

	log.Printf("[message] service started, msg_svc=%v", msgSvc != nil)

	// 6. 等待退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[message] shutting down...")
	d.Stop()
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
