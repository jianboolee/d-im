package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGroupMaxMembersFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
app:
  env: test
group:
  max_members: 100
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("GROUP_MAX_MEMBERS", "42")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Group.MaxMembers != 42 {
		t.Fatalf("expected group max members from env, got %d", cfg.Group.MaxMembers)
	}
}

func TestLoadGroupMaxMembersDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`
app:
  env: test
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Group.MaxMembers != 100 {
		t.Fatalf("expected default group max members 100, got %d", cfg.Group.MaxMembers)
	}
}
