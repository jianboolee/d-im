package storage

import (
	"regexp"
	"testing"
)

func TestNewObjectIDUsesUUIDV7(t *testing.T) {
	id, err := NewObjectID()
	if err != nil {
		t.Fatalf("generate object id: %v", err)
	}
	matched, err := regexp.MatchString(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, id)
	if err != nil {
		t.Fatalf("compile regex: %v", err)
	}
	if !matched {
		t.Fatalf("expected UUID v7, got %q", id)
	}
}
