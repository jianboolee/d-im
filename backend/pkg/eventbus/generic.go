package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
)

// PublishJSON 对 Publisher 做泛型封装,调用方直接传结构体,内部自动 JSON 序列化。
//
// 用法:
//
//	eventbus.PublishJSON(ctx, bus, im.SubjectGroupMemberJoined, im.GroupMemberJoined{
//		GroupID: groupID, UserID: userID, JoinedAt: time.Now().Unix(),
//	}, nil)
func PublishJSON[T any](ctx context.Context, p Publisher, subject string, data T, headers map[string]string) error {
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("eventbus: marshal payload for %s: %w", subject, err)
	}
	return p.Publish(ctx, subject, b, headers)
}

// SubscribeJSON 对 Subscriber 做泛型封装,自动把 event.Payload 反序列化成具体类型 T 再交给回调。
//
// 用法:
//
//	eventbus.SubscribeJSON(ctx, bus, im.SubjectGroupMemberJoined,
//		func(ctx context.Context, data im.GroupMemberJoined, evt eventbus.Event) error {
//			return mailboxSvc.OnMemberJoined(ctx, data)
//		})
func SubscribeJSON[T any](ctx context.Context, s Subscriber, subject string, fn func(ctx context.Context, data T, evt Event) error) (Subscription, error) {
	return s.Subscribe(ctx, subject, func(ctx context.Context, evt Event) error {
		var data T
		if err := json.Unmarshal(evt.Payload, &data); err != nil {
			// 反序列化失败通常意味着"生产者和消费者的payload版本不一致",
			// 这里选择返回error(而不是丢弃),让上层决定是否重试/告警,
			// 因为重试大概率也不会成功,建议在 handler 里针对这类错误单独监控。
			return fmt.Errorf("eventbus: unmarshal payload for %s into %T: %w", subject, data, err)
		}
		return fn(ctx, data, evt)
	})
}
