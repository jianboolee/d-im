package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/types"
)

func TestParseSystemEventContent(t *testing.T) {
	content, err := parseContent(types.MessageTypeSystemEvent, []byte(`{
		"event_type": "members_invited",
		"text": "Alice邀请Bob加入群聊",
		"operator_id": "user_a",
		"target_user_ids": ["user_b"],
		"group_id": "chat_001"
	}`))
	if err != nil {
		t.Fatalf("parse content: %v", err)
	}

	event, ok := content.(types.SystemEventContent)
	if !ok {
		t.Fatalf("expected SystemEventContent, got %T", content)
	}
	if event.EventType != "members_invited" || event.OperatorID != "user_a" {
		t.Fatalf("unexpected event content: %#v", event)
	}
	if err := event.Validate(); err != nil {
		t.Fatalf("validate event: %v", err)
	}
}

func TestSendMessageRejectsClientSystemEvent(t *testing.T) {
	handler := NewMessageHandler(nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewBufferString(`{
		"conversation_id": "conv_001",
		"message_type": "system_event",
		"content": {
			"event_type": "members_invited",
			"text": "fake event"
		}
	}`))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user_a"))
	rec := httptest.NewRecorder()

	handler.SendMessage(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "system_event cannot be sent by clients") {
		t.Fatalf("expected forbidden message, got %s", rec.Body.String())
	}
}
