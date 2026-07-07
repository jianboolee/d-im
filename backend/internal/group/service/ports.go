package service

import "context"

// SystemEventPort 群系统事件发布端口。
// 当前阶段通过 Message Service 发送 system_event 消息；后续可替换为 NATS 领域事件。
type SystemEventPort interface {
	PublishGroupSystemEvent(ctx context.Context, event GroupSystemEvent) error
}
