package service

import (
	"context"
	"testing"
)

func TestDefaultGroupNameUsesFirstThreeNicknames(t *testing.T) {
	svc := &GroupService{}
	svc.SetUserProfileReader(fakeUserProfileReader{
		"user_a": {ID: "user_a", Nickname: "Alice"},
		"user_b": {ID: "user_b", Nickname: "Bob"},
		"user_c": {ID: "user_c", Nickname: "Carol"},
		"user_d": {ID: "user_d", Nickname: "Dave"},
	})

	got := svc.defaultGroupName(context.Background(), []string{"user_a", "user_b", "user_c", "user_d"})
	if got != "Alice、Bob、Carol" {
		t.Fatalf("expected first three names, got %q", got)
	}
}

func TestDefaultGroupNameSkipsMissingNicknames(t *testing.T) {
	svc := &GroupService{}
	svc.SetUserProfileReader(fakeUserProfileReader{
		"user_a": {ID: "user_a", Nickname: ""},
		"user_b": {ID: "user_b", Nickname: "Bob"},
		"user_c": {ID: "user_c", Nickname: "Carol"},
	})

	got := svc.defaultGroupName(context.Background(), []string{"user_a", "user_b", "user_c"})
	if got != "Bob、Carol" {
		t.Fatalf("expected available names, got %q", got)
	}
}

func TestDefaultGroupNameFallsBackToGroupChat(t *testing.T) {
	svc := &GroupService{}
	svc.SetUserProfileReader(fakeUserProfileReader{
		"user_a": {ID: "user_a"},
	})

	got := svc.defaultGroupName(context.Background(), []string{"user_a"})
	if got != "群聊" {
		t.Fatalf("expected fallback group name, got %q", got)
	}
}
