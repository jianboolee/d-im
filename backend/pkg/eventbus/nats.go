package eventbus

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NatsBus 基于 JetStream 实现(而不是 NATS Core),因为事件总线通常需要"至少一次投递"
// 和"消费者断线重连后能续上没消费完的消息",这两点 NATS Core 的 pub/sub 都做不到,
// 必须用 JetStream。
//
// 注意:这里传入的是 *nats.Conn,复用你 pkg/nats 里已经建好、管理生命周期(重连、鉴权等)
// 的连接,这个文件不负责连接的建立和关闭。
type NatsBus struct {
	js          jetstream.JetStream
	stream      string
	retryPolicy RetryPolicy
}

// NatsOption 用于在创建时定制 NatsBus 的行为
type NatsOption func(*NatsBus)

// WithNatsRetryPolicy 覆盖默认的重试策略
func WithNatsRetryPolicy(p RetryPolicy) NatsOption {
	return func(b *NatsBus) { b.retryPolicy = p.withDefaults() }
}

// NewNatsBus 创建一个绑定到指定 stream 的总线实例。
// stream 是 JetStream 里的"流"名称,一个 stream 可以覆盖多个 subject(见 EnsureStream)。
func NewNatsBus(nc *nats.Conn, stream string, opts ...NatsOption) (*NatsBus, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("eventbus: create jetstream context: %w", err)
	}
	b := &NatsBus{js: js, stream: stream, retryPolicy: DefaultRetryPolicy()}
	for _, opt := range opts {
		opt(b)
	}
	return b, nil
}

// EnsureStream 创建或更新流的定义,subjects 是该流关心的主题(支持通配符),
// 例如 []string{"im.>"}。
//
// 重要:如果你用了默认的死信subject前缀("dlq." + subject),死信消息的subject
// 会变成类似 "dlq.im.group.member_joined",这**不匹配** "im.>" 这个通配符。
// 死信消息发布时会因为"没有stream接收这个subject"而失败(这里会走 OnError 回调,
// 但不影响主消息已经被Term掉这个事实——原消息不会被无限重试,只是死信记录丢了)。
// 所以务必把 "dlq.>" 也加进 subjects,或者单独建一个 DLQ_STREAM 来接收死信:
//
//	bus.EnsureStream(ctx, []string{"im.>", "dlq.>"})
func (b *NatsBus) EnsureStream(ctx context.Context, subjects []string) error {
	_, err := b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      b.stream,
		Subjects:  subjects,
		Retention: jetstream.LimitsPolicy,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("eventbus: ensure stream %s: %w", b.stream, err)
	}
	return nil
}

func (b *NatsBus) Publish(ctx context.Context, subject string, payload []byte, headers map[string]string) error {
	msg := &nats.Msg{Subject: subject, Data: payload, Header: toNatsHeader(headers)}
	if _, err := b.js.PublishMsg(ctx, msg); err != nil {
		return fmt.Errorf("eventbus: nats publish %s: %w", subject, err)
	}
	return nil
}

// Subscribe 为该 subject 创建一个 durable consumer 并开始消费,内置重试+死信逻辑:
//   - 处理失败且还没到 MaxDeliver 次数上限 → Nak,交给 JetStream 按配置的 BackOff 重新投递
//   - 处理失败且已达上限 → 把消息原样发到死信subject,然后 Term(彻底终止,不再重试),
//     避免消息卡在stream里既不成功也不停止的"僵尸"状态
//
// 消费组语义:consumer 名字目前按 subject 派生,同一个 subject 下的多个订阅方是"竞争消费"关系
// (一条消息只会被其中一个处理)。如果你需要多个订阅方各自收到全量消息,
// 需要给每个订阅方分配不同的 durable 名字——当前实现先覆盖单一消费者的场景。
func (b *NatsBus) Subscribe(ctx context.Context, subject string, handler Handler) (Subscription, error) {
	policy := b.retryPolicy
	consumerName := "eventbus-" + sanitizeConsumerName(subject)

	// BackOff 数组长度必须 <= MaxDeliver,这里简单地用同一个间隔重复 MaxDeliver-1 次
	// (第1次投递不需要backoff,后续每次重试前都等这么久)
	backoffs := make([]time.Duration, 0, policy.MaxDeliver-1)
	for i := 0; i < policy.MaxDeliver-1; i++ {
		backoffs = append(backoffs, policy.Backoff)
	}

	cons, err := b.js.CreateOrUpdateConsumer(ctx, b.stream, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    policy.MaxDeliver,
		BackOff:       backoffs,
	})
	if err != nil {
		return nil, fmt.Errorf("eventbus: create consumer for %s: %w", subject, err)
	}

	consumeCtx, err := cons.Consume(func(msg jetstream.Msg) {
		evt := Event{
			Subject: msg.Subject(),
			Payload: msg.Data(),
			Headers: fromNatsHeader(msg.Headers()),
		}

		// 用 context.Background() 而不是外部传入的 ctx——订阅是长期运行的,
		// 外部 ctx 的生命周期通常绑定"发起订阅"这个动作,不该用来控制每条消息的处理
		err := handler(context.Background(), evt)
		if err == nil {
			if ackErr := msg.Ack(); ackErr != nil {
				// ack失败通常是网络抖动,消息可能被重新投递导致重复处理,
				// 所以 handler 的业务逻辑应当保证幂等(比如按event.ID去重)
				policy.OnError(fmt.Errorf("eventbus: ack failed for %s: %w", subject, ackErr))
			}
			return
		}

		meta, metaErr := msg.Metadata()
		numDelivered := uint64(1)
		if metaErr == nil {
			numDelivered = meta.NumDelivered
		}

		if numDelivered < uint64(policy.MaxDeliver) {
			// 还没到上限,交给JetStream按配置的backoff重新投递
			_ = msg.Nak()
			return
		}

		// 已达最大投递次数:发到死信subject,然后彻底终止这条消息的生命周期
		dlqSubject := policy.DeadLetterSubject(subject)
		dlqMsg := &nats.Msg{Subject: dlqSubject, Data: msg.Data(), Header: msg.Headers()}
		if _, pubErr := b.js.PublishMsg(context.Background(), dlqMsg); pubErr != nil {
			policy.OnError(fmt.Errorf("eventbus: publish to dead letter %s failed: %w", dlqSubject, pubErr))
		}
		if termErr := msg.Term(); termErr != nil {
			policy.OnError(fmt.Errorf("eventbus: term message on %s failed: %w", subject, termErr))
		}
	})
	if err != nil {
		return nil, fmt.Errorf("eventbus: consume %s: %w", subject, err)
	}

	return &natsSubscription{consumeCtx: consumeCtx}, nil
}

// Close 不关闭底层 *nats.Conn——连接的生命周期由 pkg/nats 统一管理。
func (b *NatsBus) Close() error { return nil }

type natsSubscription struct {
	consumeCtx jetstream.ConsumeContext
}

func (s *natsSubscription) Unsubscribe() error {
	s.consumeCtx.Stop()
	return nil
}

func toNatsHeader(h map[string]string) nats.Header {
	if len(h) == 0 {
		return nil
	}
	nh := nats.Header{}
	for k, v := range h {
		nh.Set(k, v)
	}
	return nh
}

func fromNatsHeader(h nats.Header) map[string]string {
	if len(h) == 0 {
		return nil
	}
	m := make(map[string]string, len(h))
	for k := range h {
		m[k] = h.Get(k)
	}
	return m
}

// sanitizeConsumerName 把 subject 里的 "." "*" ">" 替换掉,
// 因为 JetStream 的 durable consumer 名字不允许包含这些字符。
func sanitizeConsumerName(subject string) string {
	out := []rune(subject)
	for i, r := range out {
		if r == '.' || r == '*' || r == '>' {
			out[i] = '_'
		}
	}
	return string(out)
}
