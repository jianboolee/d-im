package service

import (
	"context"

	"d-im/pkg/model"
)

// SystemEventPort 群系统事件发布端口。
// 当前阶段通过 Message Service 发送 system_event 消息；后续可替换为 NATS 领域事件。
type SystemEventPort interface {
	PublishGroupSystemEvent(ctx context.Context, event GroupSystemEvent) error
}

// UserProfileReader 读取用户资料，用于群系统事件生成展示文案。
type UserProfileReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}
