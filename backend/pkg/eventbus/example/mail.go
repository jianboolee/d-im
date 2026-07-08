package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"d-im/pkg/eventbus"
	// "github.com/nats-io/nats.go"
	// amqp "github.com/rabbitmq/amqp091-go"
)

// 这里把事件的 subject 和 payload 定义直接写在 main 包里,方便单独运行这个示例,
// 不依赖项目里其他模块。实际项目中建议按之前说的惯例,把这部分放到各自模块的 events.go 里。
const SubjectGroupMemberJoined = "im.group.member_joined"

type GroupMemberJoined struct {
	GroupID  string `json:"group_id"`
	UserID   string `json:"user_id"`
	Operator string `json:"operator"`
	JoinedAt int64  `json:"joined_at"`
}

func main() {
	ctx := context.Background()

	// ---- 场景A:同进程内解耦,用 MemoryBus ----
	memBus := eventbus.NewMemoryBus()
	runDemo(ctx, memBus)

	// ---- 场景B:需要跨进程/持久化,换成 NatsBus,业务代码完全不用改 ----
	//
	// nc, _ := nats.Connect(nats.DefaultURL) // 实际项目里这个连接应该来自你的 pkg/nats
	// bus, err := eventbus.NewNatsBus(nc, "IM_EVENTS", eventbus.WithNatsRetryPolicy(eventbus.RetryPolicy{
	// 	MaxDeliver: 3,
	// 	Backoff:    2 * time.Second,
	// }))
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// // "dlq.>" 一定要加,否则死信消息发布不到任何stream里,见 NatsBus.EnsureStream 的注释
	// if err := bus.EnsureStream(ctx, []string{"im.>", "dlq.>"}); err != nil {
	// 	log.Fatal(err)
	// }
	// runDemo(ctx, bus)

	// ---- 场景C:换成 RabbitBus 同理 ----
	//
	// conn, _ := amqp.Dial("amqp://guest:guest@localhost:5672/") // 应该来自你的 pkg/rabbitmq
	// ch, _ := conn.Channel()
	// bus, err := eventbus.NewRabbitBus(ch, "im_events")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// runDemo(ctx, bus)
}

// runDemo 只依赖 eventbus.Bus 这个接口,不关心具体是内存/NATS/RabbitMQ,
// 这就是抽象一层接口带来的好处:实现可以换,调用方代码不用动。
func runDemo(ctx context.Context, bus eventbus.Bus) {
	// 订阅方A:推送模块,关心"成员加入"事件,负责给群里其他人推通知
	_, err := eventbus.SubscribeJSON(ctx, bus, SubjectGroupMemberJoined,
		func(ctx context.Context, data GroupMemberJoined, evt eventbus.Event) error {
			fmt.Printf("[推送模块] 用户 %s 加入了群 %s,推送通知给其他成员\n", data.UserID, data.GroupID)
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}

	// 订阅方B:审计日志模块,同样关心"成员加入"事件,但做完全不同的事,
	// 且完全不知道推送模块的存在——这就是事件总线带来的解耦
	_, err = eventbus.SubscribeJSON(ctx, bus, SubjectGroupMemberJoined,
		func(ctx context.Context, data GroupMemberJoined, evt eventbus.Event) error {
			fmt.Printf("[审计模块] 记录: %s 于 %d 加入群 %s (操作人: %s)\n",
				data.UserID, data.JoinedAt, data.GroupID, data.Operator)
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}

	// 发布方:群服务处理完"加人"的业务逻辑后,发一个事件,自己不用知道谁在监听、监听方要干嘛
	err = eventbus.PublishJSON(ctx, bus, SubjectGroupMemberJoined, GroupMemberJoined{
		GroupID:  "group_001",
		UserID:   "user_888",
		Operator: "user_001",
		JoinedAt: time.Now().Unix(),
	}, map[string]string{"trace_id": "demo-trace-1"})
	if err != nil {
		log.Fatal(err)
	}

	// MemoryBus 是异步 fire-and-forget 的,这里睡一下让 goroutine 有机会跑完再退出,
	// 真实服务里不需要这个,进程本身是长期运行的
	time.Sleep(100 * time.Millisecond)
}
