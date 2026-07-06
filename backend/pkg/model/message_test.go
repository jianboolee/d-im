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
