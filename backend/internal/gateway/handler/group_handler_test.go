package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"d-im/internal/gateway/handler/middleware"
	"d-im/pkg/model"
	"d-im/pkg/types"
)

type fakeConversationByChatReader struct {
	conv *model.Conversation
}

func (f fakeConversationByChatReader) GetConversationByChatID(_ context.Context, _ string, _ string) (*model.Conversation, error) {
	return f.conv, nil
}

func TestNewGroupHandlerSignature(t *testing.T) {
	conv := fakeConversationByChatReader{conv: &model.Conversation{
		ConversationID: "conv_group",
		ChatID:         "chat_group",
		ChatType:       types.ChatTypeGroup,
	}}
	handler := NewGroupHandler(nil, nil, conv, nil)
	if handler == nil {
		t.Fatal("expected handler")
	}
}

func newAuthedJSONRequest(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, "user_a"))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestTimeFormatConstant(t *testing.T) {
	if timeFormatRFC3339Nano == "" {
		t.Fatal("expected time format constant")
	}
}
