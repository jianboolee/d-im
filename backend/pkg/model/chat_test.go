package model

import "testing"

func TestDefaultGroupSettings(t *testing.T) {
	settings := DefaultGroupSettings()

	if settings.JoinMethod != JoinMethodFree {
		t.Fatalf("expected default join method free, got %q", settings.JoinMethod)
	}
	if !settings.IsPublic {
		t.Fatalf("expected default group to be public")
	}
	if settings.AllowMemberInvite == nil || !*settings.AllowMemberInvite {
		t.Fatalf("expected default group to allow member invites")
	}
}
