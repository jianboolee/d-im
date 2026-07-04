package ws

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

// Server WebSocket服务器
type Server struct {
	addr     string
	upgrader websocket.Upgrader
	manager  *ClientManager
	authFunc AuthFunc
}

// AuthFunc 认证函数类型
type AuthFunc func(token string) (uid string, err error)

// NewServer 创建WebSocket服务器
func NewServer(addr string, authFunc AuthFunc) *Server {
	return &Server{
		addr: addr,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 生产环境应校验Origin
			},
		},
		manager:  NewClientManager(),
		authFunc: authFunc,
	}
}

// Start 启动WebSocket服务
func (s *Server) Start(ctx context.Context) error {
	http.HandleFunc("/ws", s.handleConnection)

	srv := &http.Server{Addr: s.addr}
	log.Printf("[websocket] listening on %s", s.addr)

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

// handleConnection 处理WebSocket连接
func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	// 从URL参数获取认证token
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	// 认证
	uid, err := s.authFunc(token)
	if err != nil {
		http.Error(w, "auth failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// 升级为WebSocket连接
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websocket] upgrade error: %v", err)
		return
	}

	// 创建客户端并注册
	client := NewClient(uid, conn, s.manager)
	s.manager.Register(client)

	go client.ReadPump()
	go client.WritePump()
}

// GetManager 获取客户端管理器
func (s *Server) GetManager() *ClientManager {
	return s.manager
}
