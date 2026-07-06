package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strings"

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
				return true
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
	token := r.URL.Query().Get("access_token")
	if token == "" {
		log.Printf("[websocket] auth failed: missing access_token remote=%s path=%s", r.RemoteAddr, r.URL.String())
		http.Error(w, "missing access_token", http.StatusUnauthorized)
		return
	}

	uid, err := s.authFunc(token)
	if err != nil {
		log.Printf("[websocket] auth failed: %v remote=%s", err, r.RemoteAddr)
		http.Error(w, "auth failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	deviceID := extractDeviceID(token)
	if deviceID == "" {
		deviceID = r.URL.Query().Get("device_id")
	}
	if deviceID == "" {
		deviceID = "unknown"
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[websocket] upgrade error: %v", err)
		return
	}

	client := NewClient(uid, deviceID, conn, s.manager)
	s.manager.Register(client)

	go client.ReadPump()
	go client.WritePump()
}

func extractDeviceID(tokenStr string) string {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	if v, ok := claims["device_id"].(string); ok {
		return v
	}
	return ""
}

// GetManager 获取客户端管理器
func (s *Server) GetManager() *ClientManager {
	return s.manager
}
