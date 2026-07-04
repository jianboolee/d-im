package ws

import (
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

// Client WebSocket客户端连接
type Client struct {
	UID    string
	conn   *websocket.Conn
	mgr    *ClientManager
	send   chan []byte
	closed bool
	mu     sync.Mutex
}

// NewClient 创建客户端
func NewClient(uid string, conn *websocket.Conn, mgr *ClientManager) *Client {
	return &Client{
		UID:  uid,
		conn: conn,
		mgr:  mgr,
		send: make(chan []byte, 256),
	}
}

// ReadPump 从WebSocket连接读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.mgr.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}
		c.mgr.HandleMessage(c, message)
	}
}

// WritePump 向WebSocket连接写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send 发送消息到客户端
func (c *Client) Send(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		select {
		case c.send <- data:
		default:
			log.Printf("[ws] send buffer full for uid=%s", c.UID)
		}
	}
}

// Close 关闭客户端
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
}

// ============================================================
// ClientManager - 客户端连接管理器
// ============================================================

// MessageHandler 消息处理函数类型
type MessageHandler func(client *Client, message []byte)

// ClientManager 客户端连接管理器
type ClientManager struct {
	clients    map[string]*Client // uid -> client
	mu         sync.RWMutex
	handler    MessageHandler
	register   chan *Client
	unregister chan *Client
}

// NewClientManager 创建客户端管理器
func NewClientManager() *ClientManager {
	mgr := &ClientManager{
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 128),
		unregister: make(chan *Client, 128),
	}
	go mgr.run()
	return mgr
}

// SetMessageHandler 设置消息处理函数
func (m *ClientManager) SetMessageHandler(handler MessageHandler) {
	m.handler = handler
}

// run 事件循环
func (m *ClientManager) run() {
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			// 如果已有同UID的连接，先关闭旧连接
			if old, exists := m.clients[client.UID]; exists {
				old.Close()
			}
			m.clients[client.UID] = client
			m.mu.Unlock()
			log.Printf("[ws] client connected: uid=%s", client.UID)

		case client := <-m.unregister:
			m.mu.Lock()
			if c, exists := m.clients[client.UID]; exists && c == client {
				delete(m.clients, client.UID)
			}
			m.mu.Unlock()
			client.Close()
			log.Printf("[ws] client disconnected: uid=%s", client.UID)
		}
	}
}

// Register 注册客户端
func (m *ClientManager) Register(client *Client) {
	m.register <- client
}

// Unregister 注销客户端
func (m *ClientManager) Unregister(client *Client) {
	m.unregister <- client
}

// HandleMessage 处理收到的消息
func (m *ClientManager) HandleMessage(client *Client, message []byte) {
	if m.handler != nil {
		m.handler(client, message)
	}
}

// GetClient 获取指定用户的客户端
func (m *ClientManager) GetClient(uid string) *Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[uid]
}

// SendToUser 向指定用户发送消息
func (m *ClientManager) SendToUser(uid string, data []byte) bool {
	m.mu.RLock()
	client := m.clients[uid]
	m.mu.RUnlock()
	if client == nil {
		return false
	}
	client.Send(data)
	return true
}

// BroadcastToAll 向所有在线用户广播消息
func (m *ClientManager) BroadcastToAll(data []byte) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, client := range m.clients {
		client.Send(data)
	}
}

// OnlineCount 在线用户数
func (m *ClientManager) OnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// IsOnline 检查用户是否在线
func (m *ClientManager) IsOnline(uid string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[uid]
	return ok
}
