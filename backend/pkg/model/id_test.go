package model

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestEntityIDsUseUnprefixedUUIDV7(t *testing.T) {
	tests := map[string]func() string{
		"message":      NewMessageID,
		"chat":         NewChatID,
		"conversation": NewConversationID,
	}

	for name, newID := range tests {
		t.Run(name, func(t *testing.T) {
			id := newID()
			if strings.Contains(id, "_") {
				t.Fatalf("ID must not contain a prefix: %q", id)
			}
			parsed, err := uuid.Parse(id)
			if err != nil {
				t.Fatalf("parse UUID: %v", err)
			}
			if parsed.Version() != 7 {
				t.Fatalf("version = %d, want 7", parsed.Version())
			}
		})
	}
}
