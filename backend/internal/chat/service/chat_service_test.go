package service

import (
	"context"
	"testing"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

type repositoryStub struct {
	inserted *model.Chat
}

func (r *repositoryStub) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type projectorStub struct {
	userIDs []string
	chat    *model.Chat
}

func (p *projectorStub) EnsureUsers(_ context.Context, userIDs []string, chat *model.Chat) error {
	p.userIDs = userIDs
	p.chat = chat
	return nil
}

func (r *repositoryStub) InsertOrGetSingle(_ context.Context, candidate *model.Chat) (*model.Chat, error) {
	r.inserted = candidate
	return candidate, nil
}

func (r *repositoryStub) Insert(_ context.Context, chat *model.Chat) error {
	r.inserted = chat
	return nil
}

func (r *repositoryStub) FindByChatID(_ context.Context, _ string) (*model.Chat, error) {
	return r.inserted, nil
}

func TestEnsureSingleChatDelegatesPersistenceOnlyAfterDomainConstruction(t *testing.T) {
	repository := &repositoryStub{}
	projector := &projectorStub{}
	service := NewChatService(repository, projector)

	chat, err := service.EnsureSingleChat(context.Background(), "user-b", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	if chat != repository.inserted || chat.ChatType != types.ChatTypeSingle || chat.SingleKey == "" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
	if projector.chat != chat || len(projector.userIDs) != 2 {
		t.Fatalf("single chat was not projected: %+v", projector)
	}
}

func TestCreateGroupChatDelegatesConstructedChat(t *testing.T) {
	repository := &repositoryStub{}
	service := NewChatService(repository, nil)

	chat, err := service.CreateGroupChat(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	if chat != repository.inserted || chat.ChatType != types.ChatTypeGroup || chat.CreatedBy != "owner" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
}

var _ Repository = (*repositoryStub)(nil)
var _ ConversationProjector = (*projectorStub)(nil)
