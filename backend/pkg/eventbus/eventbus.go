package eventbus

import "context"

// Event 是总线上流转的统一消息封装。不管底层是 NATS 还是 RabbitMQ,
// 业务代码看到的都是这同一个结构,这样切换实现不需要改业务逻辑。
type Event struct {
	// ID 是事件的唯一标识,用于日志追踪、幂等去重。
	// 具体生成方式(UUID/雪花ID)由发布方决定,总线本身不强制。
	ID string

	// Subject 是事件主题,建议约定为 "{module}.{entity}.{action}" 格式,
	// 例如 "im.group.member_joined"、"account.user.banned"。
	// NATS 和 RabbitMQ(topic exchange)都原生支持这种点分层级 + 通配符订阅。
	Subject string

	// Payload 是序列化后的具体事件数据(通常是 JSON),总线本身不关心内容格式。
	Payload []byte

	// Headers 用于传递 trace_id、事件版本号等元数据,不作为业务payload的一部分。
	Headers map[string]string
}

// Handler 处理一条事件。
// 返回 error 的语义:告诉总线"这条消息没处理成功",不同实现会据此决定是否重试/重新入队:
//   - MemoryBus:仅记录日志,不重试(进程内総线本来就不保证可靠投递)
//   - NatsBus(JetStream):Nak,由 JetStream 按消费者的重试策略重新投递
//   - RabbitBus:Nack + requeue,消息重新回到队列
type Handler func(ctx context.Context, event Event) error

// Publisher 只关心"发布"这一半能力。
// 大部分业务代码只需要依赖这个接口,而不是整个 Bus,方便单测时mock。
type Publisher interface {
	Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error
}

// Subscription 代表一次订阅,调用 Unsubscribe 后停止接收新事件(不影响已经在处理中的事件)。
type Subscription interface {
	Unsubscribe() error
}

// Subscriber 只关心"订阅"这一半能力。
type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error)
}

// Bus 同时具备发布和订阅能力,是三种实现(Memory/Nats/Rabbit)共同满足的完整接口。
type Bus interface {
	Publisher
	Subscriber
	Close() error
}
