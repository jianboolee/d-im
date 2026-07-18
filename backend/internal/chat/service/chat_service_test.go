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
	service := NewChatService(repository)

	chat, err := service.EnsureSingleChat(context.Background(), "user-b", "user-a")
	if err != nil {
		t.Fatal(err)
	}
	if chat != repository.inserted || chat.ChatType != types.ChatTypeSingle || chat.SingleKey == "" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
}

func TestCreateGroupChatDelegatesConstructedChat(t *testing.T) {
	repository := &repositoryStub{}
	service := NewChatService(repository)

	chat, err := service.CreateGroupChat(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	if chat != repository.inserted || chat.ChatType != types.ChatTypeGroup || chat.CreatedBy != "owner" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
}

var _ Repository = (*repositoryStub)(nil)
