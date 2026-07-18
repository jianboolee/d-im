package model

import "github.com/google/uuid"

// NewMessageID 创建无前缀的 UUID v7 消息 ID。
func NewMessageID() string {
	return newUUIDV7()
}

// NewChatID 创建无前缀的 UUID v7 聊天 ID。
func NewChatID() string {
	return newUUIDV7()
}

// NewConversationID 创建无前缀的 UUID v7 会话视图 ID。
func NewConversationID() string {
	return newUUIDV7()
}

func newUUIDV7() string {
	return uuid.Must(uuid.NewV7()).String()
}
