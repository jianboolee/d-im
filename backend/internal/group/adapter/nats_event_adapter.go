package adapter

import (
	"context"
	"encoding/json"

	groupSvc "d-im/internal/group/service"
	natsq "d-im/pkg/queue/nats"
)

// natsEventAdapter 实现 groupSvc.SystemEventPort，通过 NATS 发布群系统事件。
type natsEventAdapter struct {
	pub *natsq.Publisher
}

// NewNATSEventAdapter 创建 NATS 事件适配器。
func NewNATSEventAdapter(pub *natsq.Publisher) groupSvc.SystemEventPort {
	return &natsEventAdapter{pub: pub}
}

func (a *natsEventAdapter) PublishGroupSystemEvent(ctx context.Context, event groupSvc.GroupSystemEvent) error {
	if a.pub == nil || event.EventType == "" {
		return nil
	}
	subject := "dimgroup." + toSnakeCase(event.EventType)
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return a.pub.Publish(subject, data)
}

// toSnakeCase 将 CamelCase 事件类型转为 NATS subject 格式。
// 例如：MemberJoined → member_joined, GroupCreated → group_created
func toSnakeCase(s string) string {
	if s == "" {
		return s
	}
	var result []byte
	for i := 0; i < len(s); i++ {
		if i > 0 && s[i] >= 'A' && s[i] <= 'Z' {
			result = append(result, '_')
			result = append(result, s[i]+32)
		} else if s[i] >= 'A' && s[i] <= 'Z' {
			result = append(result, s[i]+32)
		} else {
			result = append(result, s[i])
		}
	}
	return string(result)
}
