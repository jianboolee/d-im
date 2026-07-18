package outbox

import (
	"context"
	"testing"

	"d-im/pkg/model"
	"d-im/pkg/types"
)

type projectorSpy struct{ called string }

func (p *projectorSpy) EnsureUsers(context.Context, []string, *model.Chat) error {
	p.called = EventUsersJoined
	return nil
}
func (p *projectorSpy) UserJoined(context.Context, string, string, types.ChatType, int64) error {
	p.called = EventUserJoined
	return nil
}
func (p *projectorSpy) UserLeft(context.Context, string, string) error {
	p.called = EventUserLeft
	return nil
}
func (p *projectorSpy) MessageSent(context.Context, []string, string, *model.Message, *types.LastMessage) error {
	p.called = EventMessageSent
	return nil
}

func TestWorkerDispatchesEveryConversationEvent(t *testing.T) {
	tests := []struct {
		eventType string
		payload   Payload
	}{
		{EventUsersJoined, Payload{ChatID: "chat-1", UserIDs: []string{"a", "b"}}},
		{EventUserJoined, Payload{ChatID: "chat-1", UserID: "a"}},
		{EventUserLeft, Payload{ChatID: "chat-1", UserID: "a"}},
		{EventMessageSent, Payload{ChatID: "chat-1", Message: &model.Message{ChatID: "chat-1"}, LastMessage: &types.LastMessage{}}},
	}
	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			spy := &projectorSpy{}
			worker := &Worker{projector: spy}
			if err := worker.apply(context.Background(), &Event{EventType: tt.eventType, Payload: tt.payload}); err != nil {
				t.Fatal(err)
			}
			if spy.called != tt.eventType {
				t.Fatalf("called %q, want %q", spy.called, tt.eventType)
			}
		})
	}
}

func TestWorkerRejectsUnknownEvent(t *testing.T) {
	worker := &Worker{projector: &projectorSpy{}}
	if err := worker.apply(context.Background(), &Event{EventType: "unknown"}); err == nil {
		t.Fatal("expected unknown event error")
	}
}
