package nats

import (
	"log"

	"github.com/nats-io/nats.go"
)

// Subscriber NATS订阅者
type Subscriber struct {
	conn *nats.Conn
	subj Subjects
}

// NewSubscriber 创建NATS订阅者（复用连接）或直接通过Publisher获取连接
func NewSubscriber(conn *nats.Conn, cfg Config) *Subscriber {
	return &Subscriber{
		conn: conn,
		subj: cfg.Subjects,
	}
}

// Subscribe 订阅指定主题
func (s *Subscriber) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := s.conn.Subscribe(subject, handler)
	if err != nil {
		return nil, err
	}
	log.Printf("[nats] subscribed to subject: %s", subject)
	return sub, nil
}

// QueueSubscribe 队列组订阅（多个实例负载均衡）
func (s *Subscriber) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	sub, err := s.conn.QueueSubscribe(subject, queue, handler)
	if err != nil {
		return nil, err
	}
	log.Printf("[nats] queue subscribed: subject=%s, queue=%s", subject, queue)
	return sub, nil
}
