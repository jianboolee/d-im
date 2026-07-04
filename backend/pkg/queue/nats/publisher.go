package nats

import (
	"errors"
	"log"

	"github.com/nats-io/nats.go"
)

// Config NATS配置
type Config struct {
	URL      string   `yaml:"url"`
	Subjects Subjects `yaml:"subjects"`
}

// Subjects NATS主题定义
type Subjects struct {
	MessageSend  string `yaml:"message_send"`
	MessagePush  string `yaml:"message_push"`
	MessageEvent string `yaml:"message_event"`
}

// Publisher NATS发布者
type Publisher struct {
	conn *nats.Conn
	subj Subjects
}

// NewPublisher 创建NATS发布者
func NewPublisher(cfg Config) (*Publisher, error) {
	conn, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, err
	}

	log.Printf("[nats] connected to %s", cfg.URL)
	return &Publisher{
		conn: conn,
		subj: cfg.Subjects,
	}, nil
}

// Close 关闭连接
func (p *Publisher) Close() {
	p.conn.Close()
}

// GetConn 获取底层 NATS 连接（用于订阅）
func (p *Publisher) GetConn() *nats.Conn {
	return p.conn
}

// Publish 发布消息到指定主题
func (p *Publisher) Publish(subject string, data []byte) error {
	if p.conn == nil {
		return errors.New("nats connection not initialized")
	}
	return p.conn.Publish(subject, data)
}

// PublishMessageSend 发布消息发送事件
func (p *Publisher) PublishMessageSend(data []byte) error {
	return p.Publish(p.subj.MessageSend, data)
}

// PublishMessagePush 发布推送事件
func (p *Publisher) PublishMessagePush(data []byte) error {
	return p.Publish(p.subj.MessagePush, data)
}

// PublishMessageEvent 发布消息事件
func (p *Publisher) PublishMessageEvent(data []byte) error {
	return p.Publish(p.subj.MessageEvent, data)
}
