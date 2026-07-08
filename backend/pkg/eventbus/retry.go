package eventbus

import "time"

// RetryPolicy 定义"消息处理失败后怎么办"。NatsBus 和 RabbitBus 都遵循同一套语义
// (对外暴露的配置一致),但底层实现机制不同:
//   - NatsBus:靠 JetStream 的 MaxDeliver + BackOff + 显式 Term 到死信subject
//   - RabbitBus:靠"重试队列(TTL)+ 死信交换机"这套没有额外插件依赖的原生模式
//
// MemoryBus 不支持重试策略——进程内总线本来就不做持久化,失败了重试也没意义
// (进程还在,直接调用方就能感知失败;进程没了,消息本来就没了),所以 MemoryBus 保持原样。
type RetryPolicy struct {
	// MaxDeliver 是一条消息最多被投递(即处理尝试)的次数,超过后进入死信、不再重试。
	MaxDeliver int

	// Backoff 是相邻两次重试之间的等待时间。
	// 用固定间隔而不是指数退避,是为了实现简单直观;如果你的失败场景主要是"下游依赖临时抖动",
	// 指数退避通常更合适,可以在这个结构体上加 BackoffMultiplier 字段自行扩展。
	Backoff time.Duration

	// DeadLetterSubject 根据原始 subject 计算死信存放位置(NATS的subject / RabbitMQ的routing key)。
	// 默认在原 subject 前加 "dlq." 前缀。
	DeadLetterSubject func(subject string) string

	// OnError 记录内部错误(比如死信发布失败、ack失败),不影响主流程,
	// 默认打印到标准日志,生产环境建议替换成接入你项目日志系统的实现。
	OnError func(err error)
}

// DefaultRetryPolicy 提供合理的默认值:最多投递5次,每次间隔5秒。
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxDeliver: 5,
		Backoff:    5 * time.Second,
		DeadLetterSubject: func(subject string) string {
			return "dlq." + subject
		},
		OnError: func(err error) {
			// 默认实现,调用方应该在生产环境覆盖掉
			println("eventbus error:", err.Error())
		},
	}
}

// withDefaults 补全零值字段,避免调用方少设置某个字段导致 panic(比如 DeadLetterSubject 为 nil)
func (p RetryPolicy) withDefaults() RetryPolicy {
	d := DefaultRetryPolicy()
	if p.MaxDeliver <= 0 {
		p.MaxDeliver = d.MaxDeliver
	}
	if p.Backoff <= 0 {
		p.Backoff = d.Backoff
	}
	if p.DeadLetterSubject == nil {
		p.DeadLetterSubject = d.DeadLetterSubject
	}
	if p.OnError == nil {
		p.OnError = d.OnError
	}
	return p
}
