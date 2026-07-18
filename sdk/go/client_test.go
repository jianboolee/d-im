package dimsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientUsesExpectedAuthentication(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/management/users/user-a":
			if r.Method != http.MethodPut {
				t.Fatalf("method = %q, want PUT", r.Method)
			}
			if got := r.Header.Get("X-API-Key"); got != "test-key" {
				t.Fatalf("X-API-Key = %q, want test-key", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": map[string]interface{}{"user_id": "user-a", "version": 1}})
		case "/api/v1/conversations/single":
			if got := r.Header.Get("Authorization"); got != "Bearer access-token" {
				t.Fatalf("Authorization = %q, want Bearer access-token", got)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0,
				"data": Conversation{ConversationID: "conversation-1", ChatID: "chat-1", ChatType: "single"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL, APIKey: "test-key"})
	if err := client.UpsertUser(context.Background(), UserData{UserID: "user-a", Status: "active", Version: 1}); err != nil {
		t.Fatalf("UpsertUser() error = %v", err)
	}

	conversation, err := client.CreateSingleConversation(context.Background(), "access-token", "user-b")
	if err != nil {
		t.Fatalf("CreateSingleConversation() error = %v", err)
	}
	if conversation.ChatID != "chat-1" {
		t.Fatalf("ChatID = %q, want chat-1", conversation.ChatID)
	}
}

func TestSendMessageUsesCurrentAPIContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/messages" {
			t.Fatalf("path = %q, want /api/v1/messages", r.URL.Path)
		}
		var req SendMessageReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.ChatID != "chat-1" || req.MessageType != "text" {
			t.Fatalf("unexpected request: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"code": 0,
			"data": SendMessageResp{Status: "accepted", ChatID: "chat-1"},
		})
	}))
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})
	resp, err := client.SendMessage(context.Background(), "access-token", SendMessageReq{
		ChatID:      "chat-1",
		MessageType: "text",
		Content:     map[string]string{"text": "hello"},
	})
	if err != nil {
		t.Fatalf("SendMessage() error = %v", err)
	}
	if resp.Status != "accepted" {
		t.Fatalf("Status = %q, want accepted", resp.Status)
	}
}
