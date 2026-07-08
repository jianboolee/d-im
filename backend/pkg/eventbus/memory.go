package eventbus

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// MemoryBus 是纯进程内实现:不跨进程、不持久化、进程崩溃事件即丢失。
// 适用场景:
//   - 同一个服务进程内的模块间解耦(比如群消息发出后,推送/未读计数/审计日志互不感知对方存在)
//   - 单元测试里替代真实的 NATS/RabbitMQ,不用起真实broker
//
// 不适用场景:任何"不能丢""要保证送达"的事件,那些应该走 NatsBus/RabbitBus。
type MemoryBus struct {
	mu       sync.RWMutex
	handlers map[string][]*memorySub
}

type memorySub struct {
	id      string
	subject string
	handler Handler
	bus     *MemoryBus
}

func NewMemoryBus() *MemoryBus {
	return &MemoryBus{handlers: make(map[string][]*memorySub)}
}

func (b *MemoryBus) Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	b.mu.RLock()
	// 复制一份再释放锁,避免 handler 里再次调用 Subscribe/Unsubscribe 时死锁
	subs := make([]*memorySub, len(b.handlers[subject]))
	copy(subs, b.handlers[subject])
	b.mu.RUnlock()

	evt := Event{ID: uuid.NewString(), Subject: subject, Payload: payload, Headers: headers}
	for _, s := range subs {
		s := s
		// 每个订阅者独立 goroutine 执行,一是不让慢订阅者拖慢发布方,
		// 二是订阅者之间互不影响(一个panic不会波及另一个)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// 生产环境这里应该接入你项目的日志/告警,而不是仅仅recover吞掉
					_ = r
				}
			}()
			_ = s.handler(ctx, evt) // 内存总线不重试:失败了就是失败了,由handler自己决定要不要记日志
		}()
	}
	return nil
}

func (b *MemoryBus) Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sub := &memorySub{id: uuid.NewString(), subject: subject, handler: handler, bus: b}
	b.handlers[subject] = append(b.handlers[subject], sub)
	return sub, nil
}

func (b *MemoryBus) Close() error { return nil }

func (s *memorySub) Unsubscribe() error {
	s.bus.mu.Lock()
	defer s.bus.mu.Unlock()
	subs := s.bus.handlers[s.subject]
	for i, h := range subs {
		if h.id == s.id {
			s.bus.handlers[s.subject] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
	return nil
}
