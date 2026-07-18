package projector

import (
	"context"

	conversationRepo "d-im/internal/conversation/repository"
	"d-im/pkg/model"
	"d-im/pkg/types"
)

// ConversationProjector synchronously projects chat, membership and message facts
// into per-user conversation views. It deliberately owns no domain decisions.
type ConversationProjector struct {
	repo *conversationRepo.ConversationRepo
}

func NewConversationProjector(repo *conversationRepo.ConversationRepo) *ConversationProjector {
	return &ConversationProjector{repo: repo}
}

func (p *ConversationProjector) UserJoined(ctx context.Context, uid, chatID string, chatType types.ChatType, lastReadSeq int64) error {
	return p.repo.Upsert(ctx, &model.Conversation{
		UID: uid, ChatID: chatID, ChatType: chatType, LastReadSeq: lastReadSeq,
	})
}

func (p *ConversationProjector) EnsureUsers(ctx context.Context, uidList []string, chat *model.Chat) error {
	return p.repo.BatchUpsert(ctx, uidList, chat)
}

func (p *ConversationProjector) UserLeft(ctx context.Context, uid, chatID string) error {
	return p.repo.MarkLeft(ctx, uid, chatID)
}

func (p *ConversationProjector) MessageRead(ctx context.Context, uid, chatID string, seq int64) error {
	return p.repo.MarkRead(ctx, uid, chatID, seq)
}

func (p *ConversationProjector) MessageSent(ctx context.Context, participantIDs []string, senderID string, msg *model.Message, lastMsg *types.LastMessage) error {
	for _, uid := range participantIDs {
		if err := p.repo.UpdateLastMessage(ctx, uid, msg.ChatID, lastMsg); err != nil {
			return err
		}
	}
	if senderID != "" && msg.Seq > 0 {
		return p.repo.MarkRead(ctx, senderID, msg.ChatID, msg.Seq)
	}
	return nil
}
