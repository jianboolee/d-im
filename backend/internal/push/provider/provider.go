package provider

import (
	"context"

	"d-im/pkg/types"
)

// PushProvider 推送提供商接口
type PushProvider interface {
	Name() string
	Push(ctx context.Context, req *PushRequest) (*PushResponse, error)
	BatchPush(ctx context.Context, reqs []*PushRequest) (*BatchPushResponse, error)
	IsHealthy(ctx context.Context) bool
}

// PushRequest 推送请求
type PushRequest struct {
	Platform  Platform               `json:"platform"`
	PushToken string                 `json:"push_token"`
	UserID    string                 `json:"user_id"`
	Title     string                 `json:"title"`
	Body      string                 `json:"body"`
	Sound     string                 `json:"sound"`
	Badge     int                    `json:"badge"`
	Extra     map[string]interface{} `json:"extra"`
	MsgID     string                 `json:"msg_id"`
	ChatID    string                 `json:"chat_id"`
	MsgType   types.MessageType      `json:"msg_type"`
	Priority  PushPriority           `json:"priority"`
	TTL       int64                  `json:"ttl"`
}

// PushResponse 推送响应
type PushResponse struct {
	Success   bool   `json:"success"`
	MsgID     string `json:"msg_id"`
	ErrorCode string `json:"error_code,omitempty"`
	ErrorMsg  string `json:"error_msg,omitempty"`
}

// BatchPushResponse 批量推送响应
type BatchPushResponse struct {
	Total      int             `json:"total"`
	SuccessNum int             `json:"success_num"`
	FailedNum  int             `json:"failed_num"`
	Results    []*PushResponse `json:"results"`
}

// PushPriority 推送优先级
type PushPriority string

const (
	PushPriorityHigh   PushPriority = "high"
	PushPriorityNormal PushPriority = "normal"
)

// Platform 平台类型
type Platform = string

const (
	PlatformIOS     Platform = "ios"
	PlatformAndroid Platform = "android"
	PlatformWeb     Platform = "web"
)

// PushContentBuilder 推送内容构建器
type PushContentBuilder struct{}

// BuildPushContent 根据消息类型构建推送标题和内容
func (b *PushContentBuilder) BuildPushContent(msgType types.MessageType, content interface{}, senderName string) (title string, body string) {
	switch msgType {
	case types.MessageTypeText:
		if tc, ok := content.(types.TextContent); ok {
			title = senderName
			body = tc.Text
		}
	case types.MessageTypeImage:
		title = senderName
		body = "[图片]"
	case types.MessageTypeVideo:
		title = senderName
		body = "[视频]"
	case types.MessageTypeVoice:
		title = senderName
		body = "[语音]"
	case types.MessageTypeFile:
		title = senderName
		body = "[文件]"
	case types.MessageTypeLocation:
		title = senderName
		body = "[位置]"
	case types.MessageTypeCard:
		title = senderName
		body = "[卡片]"
	case types.MessageTypeLink:
		title = senderName
		body = "[链接]"
	case types.MessageTypeTemplate:
		title = senderName
		body = "[通知]"
	default:
		title = senderName
		body = "新消息"
	}

	runes := []rune(body)
	if len(runes) > 100 {
		body = string(runes[:100]) + "..."
	}

	return title, body
}
