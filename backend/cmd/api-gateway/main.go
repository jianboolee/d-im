package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	chatRepo "d-im/internal/chat/repository"
	convSvc "d-im/internal/conversation/service"
	"d-im/internal/gateway"
	"d-im/internal/gateway/handler"
	"d-im/internal/gateway/router"
	groupAdapter "d-im/internal/group/adapter"
	groupAvatar "d-im/internal/group/avatar"
	groupRepo "d-im/internal/group/repository"
	groupSvc "d-im/internal/group/service"
	mediaSvc "d-im/internal/media/service"
	mediaStorage "d-im/internal/media/storage"
	"d-im/internal/message/dispatcher"
	"d-im/internal/message/repository"
	messageSvc "d-im/internal/message/service"
	userRepo "d-im/internal/user/repository"
	"d-im/internal/user/service"
	"d-im/pkg/config"
	"d-im/pkg/crypto"
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

	db, err := mongodb.NewClient(ctx, mongodb.Config{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		PoolSize: cfg.MongoDB.PoolSize,
		Timeout:  cfg.MongoDB.Timeout,
	})
	if err != nil {
		log.Fatalf("mongodb: %v", err)
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

	msgRepo := repository.NewMessageRepo(db)
	convMgr := model.NewConversationManager(db)
	chatR := chatRepo.NewChatRepo(db)
	gRepo := groupRepo.NewGroupRepo(db)
	mRepo := groupRepo.NewMemberRepo(db)
	groupService := groupSvc.NewGroupService(db, chatR, gRepo, mRepo, convMgr)
	memberService := groupSvc.NewMemberService(db, chatR, gRepo, mRepo, convMgr)
	msgSvc := messageSvc.NewMessageService(msgRepo, chatR, convMgr, natsPub)
	msgSvc.SetGroupReader(groupService)
	store, mediaStaticHandler, err := newMediaStorage(cfg)
	if err != nil {
		log.Fatalf("media storage: %v", err)
	}
	uploadSvc := mediaSvc.NewUploadService(store, cfg.Storage.MaxImageSize)

	conversationSvc := convSvc.NewConversationService(convMgr, chatR)
	uRepo := userRepo.NewUserRepo(db)

	// === message dispatcher（原 message 服务）===
	d := dispatcher.NewDispatcher(msgRepo, 4)
	d.Start(ctx)
	defer d.Stop()

	// === user 事件同步（JetStream durable consumer，原 user 服务）===
	if cfg.NATS.UserStream != "" {
		js, jsErr := natsPub.JetStream()
		if jsErr != nil {
			log.Printf("[gateway] WARNING: jetstream not available, user sync disabled: %v", jsErr)
		} else {
			syncSvc := service.NewUserSyncService(uRepo)
			if syncErr := syncSvc.Start(ctx, js, cfg.NATS.UserStream); syncErr != nil {
				log.Printf("[gateway] WARNING: user sync start failed, continuing without sync: %v", syncErr)
			}
		}
	}

	conn := natsPub.GetConn()

	// === NATS 消息消费（原 message 服务）===
	_, err = conn.Subscribe("im.message.send", func(msg *nats.Msg) {
		var req messageSvc.SendMessageReq
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("[gateway] unmarshal send request failed: %v", err)
			return
		}
		_, err := msgSvc.Send(context.Background(), &req)
		if err != nil {
			log.Printf("[gateway] send failed: chat=%s sender=%s type=%s err=%v", req.ChatID, req.SenderID, req.MsgType, err)
		}
	})
	if err != nil {
		log.Fatalf("[gateway] subscribe im.message.send: %v", err)
	}
	log.Printf("[gateway] message consumer started, subscribed to im.message.send")

	authHandler := handler.NewAuthHandler(jwtMgr, cfg.App.FrontendURL, cfg.Auth.SuperPassword)
	messageHandler := handler.NewMessageHandler(msgSvc, conversationSvc, uRepo, natsPub)
	convHandler := handler.NewConversationHandler(conversationSvc, chatR, groupService, uRepo)
	eventPub := groupSvc.NewEventPublisher(groupAdapter.NewCompositeEventAdapter(natsPub))
	eventPub.SetUserProfileReader(uRepo)
	avatarGenerator := groupAvatar.NewGenerator(store, uRepo)
	groupService.SetAvatarGenerator(avatarGenerator)
	memberService.SetAvatarGenerator(avatarGenerator)
	groupService.SetEventPublisher(eventPub)
	memberService.SetEventPublisher(eventPub)
	groupPushConsumer := groupAdapter.NewGroupPushConsumer(natsPub, gRepo, mRepo)
	if err := groupPushConsumer.Start(conn); err != nil {
		log.Fatalf("[gateway] start group push consumer: %v", err)
	}
	defer groupPushConsumer.Stop()
	groupHandler := handler.NewGroupHandler(groupService, memberService, conversationSvc, uRepo)
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
