package model

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"d-im/pkg/types"
)

func TestNewSingleChatKeyIsSymmetric(t *testing.T) {
	forward, err := NewSingleChatKey("third-party:user-a", "user-b")
	if err != nil {
		t.Fatalf("forward key: %v", err)
	}
	reverse, err := NewSingleChatKey("user-b", "third-party:user-a")
	if err != nil {
		t.Fatalf("reverse key: %v", err)
	}
	if forward != reverse {
		t.Fatalf("keys differ: %q != %q", forward, reverse)
	}
	if matched, _ := regexp.MatchString(`^[0-9a-f]{64}$`, forward); !matched {
		t.Fatalf("key is not an unprefixed SHA-256 value: %q", forward)
	}
}

func TestNewSingleChatBuildsCanonicalEntity(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	chat, err := NewSingleChat("user-b", "user-a", now)
	if err != nil {
		t.Fatal(err)
	}
	if chat.ChatType != types.ChatTypeSingle || chat.SingleKey == "" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
	if len(chat.Members) != 2 || chat.Members[0] != "user-a" || chat.Members[1] != "user-b" {
		t.Fatalf("members are not canonical: %v", chat.Members)
	}
	if chat.MemberCount != 2 || chat.CreatedAt != now || chat.UpdatedAt != now {
		t.Fatalf("unexpected defaults: %+v", chat)
	}
}

func TestNewGroupChatBuildsMessageContainer(t *testing.T) {
	now := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	chat, err := NewGroupChat("owner", now)
	if err != nil {
		t.Fatal(err)
	}
	if chat.ChatType != types.ChatTypeGroup || chat.CreatedBy != "owner" || chat.ChatID == "" {
		t.Fatalf("unexpected chat: %+v", chat)
	}
}

func TestNewSingleChatKeyAvoidsDelimiterCollisions(t *testing.T) {
	first, err := NewSingleChatKey("a", "b:c")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewSingleChatKey("a:b", "c")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatalf("different user pairs produced the same key: %q", first)
	}
}

func TestNewSingleChatKeyTreatsUserIDsAsOpaque(t *testing.T) {
	withSpace, err := NewSingleChatKey(" user-a", "user-b")
	if err != nil {
		t.Fatal(err)
	}
	withoutSpace, err := NewSingleChatKey("user-a", "user-b")
	if err != nil {
		t.Fatal(err)
	}
	if withSpace == withoutSpace {
		t.Fatal("user ID whitespace was normalized")
	}
}

func TestNewSingleChatKeyRejectsInvalidPairs(t *testing.T) {
	if _, err := NewSingleChatKey("", "user-b"); !errors.Is(err, ErrUserIDRequired) {
		t.Fatalf("empty ID error = %v", err)
	}
	if _, err := NewSingleChatKey("user-a", "user-a"); !errors.Is(err, ErrSingleChatWithSelf) {
		t.Fatalf("same-user error = %v", err)
	}
}
