package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	convSvc "d-im/internal/conversation/service"
	"d-im/internal/gateway"
	"d-im/internal/gateway/handler"
	"d-im/internal/gateway/router"
	groupSvc "d-im/internal/group/service"
	mediaSvc "d-im/internal/media/service"
	mediaStorage "d-im/internal/media/storage"
	"d-im/internal/message/repository"
	messageSvc "d-im/internal/message/service"
	userRepo "d-im/internal/user/repository"
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

	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	idGen, err := snowflake.NewGenerator(snowflake.Config{
		WorkerID:     cfg.Snowflake.WorkerID,
		DatacenterID: cfg.Snowflake.DatacenterID,
	})
	if err != nil {
		log.Fatalf("snowflake: %v", err)
	}

	accessExpire, _ := time.ParseDuration(cfg.JWT.AccessExpire)
	refreshExpire, _ := time.ParseDuration(cfg.JWT.RefreshExpire)
	ticketExpire, _ := time.ParseDuration(cfg.JWT.TicketExpire)
	jwtMgr := crypto.NewJWTManager(cfg.JWT.Secret, accessExpire, refreshExpire, ticketExpire, cfg.JWT.APIKey)

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

	chatMgr := model.NewChatIDManager(db, idGen)
	msgRepo := repository.NewMessageRepo(db)
	convMgr := model.NewConversationManager(db, idGen)
	msgSvc := messageSvc.NewMessageService(msgRepo, idGen, chatMgr, convMgr, natsPub)
	store, mediaStaticHandler, err := newMediaStorage(cfg)
	if err != nil {
		log.Fatalf("media storage: %v", err)
	}
	uploadSvc := mediaSvc.NewUploadService(store, cfg.Storage.MaxImageSize)

	conversationSvc := convSvc.NewConversationService(convMgr, chatMgr)
	uRepo := userRepo.NewUserRepo(db)
	authHandler := handler.NewAuthHandler(jwtMgr, cfg.App.FrontendURL, cfg.Auth.SuperPassword)
	messageHandler := handler.NewMessageHandler(msgSvc, conversationSvc, uRepo)
	convHandler := handler.NewConversationHandler(conversationSvc, chatMgr, uRepo)
	groupService := groupSvc.NewGroupService(chatMgr, convMgr)
	groupHandler := handler.NewGroupHandler(groupService, conversationSvc, msgSvc, uRepo)
	uploadHandler := handler.NewUploadHandler(uploadSvc)
	userHandler := handler.NewUserHandler(uRepo)
	sdkHandler := handler.NewSDKHandler(jwtMgr, uRepo)
	httpHandler := router.NewRouter(jwtMgr, authHandler, messageHandler, convHandler, groupHandler, uploadHandler, mediaStaticHandler, userHandler, sdkHandler)

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

func newMediaStorage(cfg *config.Config) (mediaStorage.Storage, http.Handler, error) {
	localStaticHandler, err := newLocalMediaStaticHandler(cfg)
	if err != nil {
		return nil, nil, err
	}

	provider := cfg.Storage.Provider
	if provider == "" {
		provider = mediaStorage.ProviderLocal
	}

	switch provider {
	case mediaStorage.ProviderLocal:
		local, err := mediaStorage.NewLocalStorage(mediaStorage.LocalConfig{
			RootDir:       cfg.Storage.Local.RootDir,
			URLPrefix:     cfg.Storage.Local.URLPrefix,
			PublicBaseURL: cfg.Storage.PublicBaseURL,
		})
		if err != nil {
			return nil, nil, err
		}
		return local, localStaticHandler, nil
	case mediaStorage.ProviderAliyunOSS:
		oss, err := mediaStorage.NewAliyunOSSStorage(mediaStorage.AliyunOSSConfig{
			Endpoint:        cfg.Storage.AliyunOSS.Endpoint,
			AccessKeyID:     cfg.Storage.AliyunOSS.AccessKeyID,
			AccessKeySecret: cfg.Storage.AliyunOSS.AccessKeySecret,
			Bucket:          cfg.Storage.AliyunOSS.Bucket,
			Directory:       cfg.Storage.AliyunOSS.Directory,
			PublicBaseURL:   cfg.Storage.AliyunOSS.PublicBaseURL,
		})
		if err != nil {
			return nil, nil, err
		}
		return oss, localStaticHandler, nil
	case "qiniu":
		return nil, nil, fmt.Errorf("qiniu storage is not implemented yet")
	default:
		return nil, nil, fmt.Errorf("unsupported storage provider %q", provider)
	}
}

func newLocalMediaStaticHandler(cfg *config.Config) (http.Handler, error) {
	local, err := mediaStorage.NewLocalStorage(mediaStorage.LocalConfig{
		RootDir:       cfg.Storage.Local.RootDir,
		URLPrefix:     cfg.Storage.Local.URLPrefix,
		PublicBaseURL: cfg.Storage.PublicBaseURL,
	})
	if err != nil {
		return nil, err
	}
	return http.StripPrefix(local.URLPrefix()+"/", http.FileServer(http.Dir(local.RootDir()))), nil
}
