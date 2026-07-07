package model

import "testing"

func TestDefaultGroupSettings(t *testing.T) {
	settings := DefaultGroupSettings()

	if settings.JoinMethod != JoinMethodInvite {
		t.Fatalf("expected default join method invite, got %q", settings.JoinMethod)
	}
	if !settings.IsPublic {
		t.Fatalf("expected default group to be public")
	}
}
