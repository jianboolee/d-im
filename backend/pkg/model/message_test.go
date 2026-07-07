package model

import (
	"testing"

	"d-im/pkg/types"

	"go.mongodb.org/mongo-driver/bson"
)

func TestNormalizeMessageContentFromBSON(t *testing.T) {
	content := NormalizeMessageContent(types.MessageTypeText, bson.D{
		{Key: "text", Value: "hello"},
		{Key: "mentions", Value: bson.A{"user_b"}},
		{Key: "is_at_all", Value: false},
	})

	text, ok := content.(types.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", content)
	}
	if text.Text != "hello" {
		t.Fatalf("expected text hello, got %q", text.Text)
	}
	if len(text.Mentions) != 1 || text.Mentions[0] != "user_b" {
		t.Fatalf("expected mentions [user_b], got %#v", text.Mentions)
	}
}

func TestNormalizeSystemEventContentFromBSON(t *testing.T) {
	content := NormalizeMessageContent(types.MessageTypeSystemEvent, bson.D{
		{Key: "event_type", Value: "members_invited"},
		{Key: "text", Value: "Alice邀请Bob加入群聊"},
		{Key: "operator_id", Value: "user_a"},
		{Key: "target_user_ids", Value: bson.A{"user_b"}},
		{Key: "group_id", Value: "chat_001"},
	})

	event, ok := content.(types.SystemEventContent)
	if !ok {
		t.Fatalf("expected SystemEventContent, got %T", content)
	}
	if event.EventType != "members_invited" || event.Text != "Alice邀请Bob加入群聊" {
		t.Fatalf("unexpected system event content: %#v", event)
	}
	if len(event.TargetUserIDs) != 1 || event.TargetUserIDs[0] != "user_b" {
		t.Fatalf("expected target user user_b, got %#v", event.TargetUserIDs)
	}
}

func TestContentMapFromTypedContent(t *testing.T) {
	content := ContentMap(types.TextContent{
		Text:    "hello",
		IsAtAll: true,
	})

	if content["text"] != "hello" {
		t.Fatalf("expected text hello, got %#v", content["text"])
	}
	if content["is_at_all"] != true {
		t.Fatalf("expected is_at_all true, got %#v", content["is_at_all"])
	}
}

func TestContentMapFromBSONMap(t *testing.T) {
	content := ContentMap(bson.M{"text": "hello"})

	if content["text"] != "hello" {
		t.Fatalf("expected text hello, got %#v", content["text"])
	}
}
