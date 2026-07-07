package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoragePutAndURL(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStorage(LocalConfig{
		RootDir:       root,
		URLPrefix:     "/media",
		PublicBaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("new local storage: %v", err)
	}

	stored, err := store.Put(context.Background(), Object{
		Key:         "im/images/test.txt",
		Reader:      strings.NewReader("hello"),
		Size:        5,
		ContentType: "text/plain",
		FileName:    "test.txt",
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	if stored.Provider != ProviderLocal {
		t.Fatalf("expected local provider, got %q", stored.Provider)
	}
	if stored.URL != "http://localhost:8080/media/im/images/test.txt" {
		t.Fatalf("unexpected url %q", stored.URL)
	}
	data, err := os.ReadFile(filepath.Join(root, "im/images/test.txt"))
	if err != nil {
		t.Fatalf("read stored file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected stored content hello, got %q", string(data))
	}
}

func TestLocalStorageRejectsPathTraversal(t *testing.T) {
	store, err := NewLocalStorage(LocalConfig{RootDir: t.TempDir()})
	if err != nil {
		t.Fatalf("new local storage: %v", err)
	}
	if _, err := store.URL(context.Background(), "../secret.txt"); err == nil {
		t.Fatal("expected path traversal error")
	}
}
