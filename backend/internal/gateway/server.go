package gateway

import (
	"context"
	"log"
	"net/http"
)

// Config Gateway服务器配置
type Config struct {
	HTTPPort string `yaml:"http_port"`
	GRPCPort string `yaml:"grpc_port"`
}

// Server API Gateway服务器
type Server struct {
	httpServer *http.Server
}

// NewServer 创建Gateway服务器
func NewServer(cfg Config, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:    ":" + cfg.HTTPPort,
			Handler: handler,
		},
	}
}

// Start 启动HTTP服务
func (s *Server) Start(ctx context.Context) error {
	log.Printf("[gateway] http listening on %s", s.httpServer.Addr)

	go func() {
		<-ctx.Done()
		s.httpServer.Shutdown(context.Background())
	}()

	return s.httpServer.ListenAndServe()
}
