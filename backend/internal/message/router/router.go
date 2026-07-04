package router

import (
	"d-im/pkg/types"
)

// RouteResult 路由结果
type RouteResult struct {
	ChatID     string
	ChatType   types.ChatType
	TargetUIDs []string // 消息接收者UID列表
}

// Router 消息路由接口
type Router interface {
	Route(chatID string, fromUID string) (*RouteResult, error)
}
