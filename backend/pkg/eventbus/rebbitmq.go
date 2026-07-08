package eventbus

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// RabbitBus 基于 RabbitMQ 的 topic exchange 实现,重试/死信用"重试队列(TTL)+ 死信交换机"
// 这套原生模式实现,不依赖 delayed-message-exchange 之类的额外插件。
//
// 整体拓扑(以 subject = "im.group.member_joined" 为例):
//
//	主交换机(topic, 你传入的 exchange 参数)
//	  └─ 主队列 eventbus.im.group.member_joined  ← 正常消费在这里发生
//
//	死信交换机(topic, exchange + ".dlx")
//	  └─ 死信队列 eventbus.dlq.im.group.member_joined  ← 达到最大重试次数的消息落这里,供人工排查
//
//	重试队列 eventbus.retry.im.group.member_joined(不绑定任何交换机,直接用默认交换机+队列名投递)
//	  设置了 x-message-ttl = Backoff、x-dead-letter-exchange = 主交换机、x-dead-letter-routing-key = subject
//	  → 消息在这里"睡"够 Backoff 时长后,被自动重新投递回主交换机,回到主队列重新消费
//
// 注意:这里传入的是 *amqp.Channel,复用你 pkg/rabbitmq 里已经建好的连接/channel,
// 这个文件不负责连接的建立、重连、关闭。
type RabbitBus struct {
	ch          *amqp.Channel
	exchange    string
	dlx         string
	retryPolicy RetryPolicy
}

// RabbitOption 用于在创建时定制 RabbitBus 的行为
type RabbitOption func(*RabbitBus)

// WithRabbitRetryPolicy 覆盖默认的重试策略
func WithRabbitRetryPolicy(p RetryPolicy) RabbitOption {
	return func(b *RabbitBus) { b.retryPolicy = p.withDefaults() }
}

// NewRabbitBus 声明主交换机和死信交换机(都是 durable topic exchange),然后返回总线实例。
func NewRabbitBus(ch *amqp.Channel, exchange string, opts ...RabbitOption) (*RabbitBus, error) {
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: declare exchange %s: %w", exchange, err)
	}
	dlx := exchange + ".dlx"
	if err := ch.ExchangeDeclare(dlx, "topic", true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: declare dead letter exchange %s: %w", dlx, err)
	}

	b := &RabbitBus{ch: ch, exchange: exchange, dlx: dlx, retryPolicy: DefaultRetryPolicy()}
	for _, opt := range opts {
		opt(b)
	}
	return b, nil
}

func (b *RabbitBus) Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	amqpHeaders := amqp.Table{}
	for k, v := range headers {
		amqpHeaders[k] = v
	}
	err := b.ch.PublishWithContext(ctx, b.exchange, subject, false, false, amqp.Publishing{
		ContentType:  "application/json",
		Body:         payload,
		Headers:      amqpHeaders,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return fmt.Errorf("eventbus: rabbitmq publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe 声明主队列 + 死信队列 + 重试队列,然后开始消费,内置重试计数(通过自定义header
// "x-eventbus-retry-count"跟踪,而不是依赖RabbitMQ的x-death,因为x-death的语义是"死信次数"
// 而不是"业务重试次数",两者容易混淆,自己维护一个header更直观可控):
//   - 处理失败且重试次数 < MaxDeliver → 把消息投进重试队列(带TTL),Ack掉主队列里的这条,
//     等TTL到期后消息会被RabbitMQ自动重新路由回主队列
//   - 处理失败且重试次数已达上限 → 把消息投进死信队列供人工排查,Ack掉主队列里的这条
//
// 消费组语义:队列名固定按subject派生,同一个subject下的多个订阅方是"竞争消费"关系。
// 如果你需要多个订阅方各自收到全量消息,需要给每个订阅方分配不同的队列名前缀。
func (b *RabbitBus) Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error) {
	policy := b.retryPolicy

	queueName := "eventbus." + subject
	dlqName := "eventbus.dlq." + subject
	retryName := "eventbus.retry." + subject

	if _, err := b.ch.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: declare queue %s: %w", queueName, err)
	}
	if err := b.ch.QueueBind(queueName, subject, b.exchange, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: bind queue %s to %s: %w", queueName, subject, err)
	}

	if _, err := b.ch.QueueDeclare(dlqName, true, false, false, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: declare dlq %s: %w", dlqName, err)
	}
	if err := b.ch.QueueBind(dlqName, subject, b.dlx, false, nil); err != nil {
		return nil, fmt.Errorf("eventbus: bind dlq %s to %s: %w", dlqName, subject, err)
	}

	// 重试队列不绑定到任何交换机——消息是直接publish到这个队列名上的(见下方 requeueForRetry),
	// TTL到期后由 x-dead-letter-exchange/x-dead-letter-routing-key 转投回主交换机
	retryArgs := amqp.Table{
		"x-dead-letter-exchange":    b.exchange,
		"x-dead-letter-routing-key": subject,
		"x-message-ttl":             policy.Backoff.Milliseconds(),
	}
	if _, err := b.ch.QueueDeclare(retryName, true, false, false, false, retryArgs); err != nil {
		return nil, fmt.Errorf("eventbus: declare retry queue %s: %w", retryName, err)
	}

	if err := b.ch.Qos(10, 0, false); err != nil {
		return nil, fmt.Errorf("eventbus: set qos: %w", err)
	}

	msgs, err := b.ch.Consume(queueName, queueName, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("eventbus: consume %s: %w", queueName, err)
	}

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case d, ok := <-msgs:
				if !ok {
					return
				}
				evt := Event{
					Subject: d.RoutingKey,
					Payload: d.Body,
					Headers: fromAmqpHeaders(d.Headers),
				}

				if err := handler(context.Background(), evt); err == nil {
					_ = d.Ack(false)
					continue
				}

				retryCount := retryCountFromHeaders(d.Headers)
				if retryCount+1 < policy.MaxDeliver {
					if pubErr := b.requeueForRetry(retryName, d, retryCount+1); pubErr != nil {
						policy.OnError(fmt.Errorf("eventbus: requeue to retry queue failed for %s: %w", subject, pubErr))
					}
				} else {
					if pubErr := b.sendToDeadLetter(subject, d); pubErr != nil {
						policy.OnError(fmt.Errorf("eventbus: publish to dead letter failed for %s: %w", subject, pubErr))
					}
				}
				// 无论进重试队列还是死信队列,都要把原消息从主队列Ack掉,
				// 因为"下一份"已经在重试/死信队列里了,不能让原消息也留在主队列被重复消费
				_ = d.Ack(false)
			}
		}
	}()

	return &rabbitSubscription{ch: b.ch, queueName: queueName, done: done}, nil
}

// requeueForRetry 把消息发布到重试队列(用默认交换机+队列名作路由的方式,
// 这是AMQP里"直接发给某个队列"的标准写法),并把当前重试次数写进header
func (b *RabbitBus) requeueForRetry(retryQueue string, d amqp.Delivery, nextRetryCount int) error {
	headers := amqp.Table{}
	for k, v := range d.Headers {
		headers[k] = v
	}
	headers["x-eventbus-retry-count"] = int32(nextRetryCount)

	return b.ch.PublishWithContext(context.Background(), "", retryQueue, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		Body:         d.Body,
		Headers:      headers,
		DeliveryMode: amqp.Persistent,
	})
}

func (b *RabbitBus) sendToDeadLetter(subject string, d amqp.Delivery) error {
	return b.ch.PublishWithContext(context.Background(), b.dlx, subject, false, false, amqp.Publishing{
		ContentType:  d.ContentType,
		Body:         d.Body,
		Headers:      d.Headers,
		DeliveryMode: amqp.Persistent,
	})
}

func retryCountFromHeaders(h amqp.Table) int {
	v, ok := h["x-eventbus-retry-count"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case int32:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func (b *RabbitBus) Close() error { return b.ch.Close() }

type rabbitSubscription struct {
	ch        *amqp.Channel
	queueName string
	done      chan struct{}
}

func (s *rabbitSubscription) Unsubscribe() error {
	close(s.done)
	return s.ch.Cancel(s.queueName, false)
}

func fromAmqpHeaders(h amqp.Table) map[string]string {
	if len(h) == 0 {
		return nil
	}
	m := make(map[string]string, len(h))
	for k, v := range h {
		if s, ok := v.(string); ok {
			m[k] = s
		}
	}
	return m
}
