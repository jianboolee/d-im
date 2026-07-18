package service

import (
	"context"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

type conversationProjectionWriter interface {
	EnsureUsers(ctx context.Context, userIDs []string, chat *model.Chat) error
	UserJoined(ctx context.Context, uid, chatID string, chatType types.ChatType, lastReadSeq int64) error
	UserLeft(ctx context.Context, uid, chatID string) error
}
