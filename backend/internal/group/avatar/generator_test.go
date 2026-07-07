package avatar

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"d-im/internal/media/storage"
)

type fakeStorage struct {
	key         string
	contentType string
	data        []byte
	rootDir     string
	urlPrefix   string
}

func (s *fakeStorage) Put(_ context.Context, obj storage.Object) (*storage.StoredObject, error) {
	s.key = obj.Key
	s.contentType = obj.ContentType
	data, err := io.ReadAll(obj.Reader)
	if err != nil {
		return nil, err
	}
	s.data = data
	return &storage.StoredObject{
		Provider:    "fake",
		Key:         obj.Key,
		URL:         "/media/" + obj.Key,
		Size:        int64(len(data)),
		ContentType: obj.ContentType,
		FileName:    obj.FileName,
	}, nil
}

func (s *fakeStorage) Delete(_ context.Context, _ string) error {
	return nil
}

func (s *fakeStorage) URL(_ context.Context, key string) (string, error) {
	return "/media/" + key, nil
}

func (s *fakeStorage) Provider() string {
	return "fake"
}

func (s *fakeStorage) RootDir() string {
	return s.rootDir
}

func (s *fakeStorage) URLPrefix() string {
	if s.urlPrefix == "" {
		return "/media"
	}
	return s.urlPrefix
}

func TestGenerateBuildsGridAvatar(t *testing.T) {
	gen := NewGenerator(nil, nil)
	img := gen.Generate(context.Background(), "chat_group", []string{
		"user_a", "user_b", "user_c", "user_d", "user_e",
	})

	if got := img.Bounds(); got != image.Rect(0, 0, 256, 256) {
		t.Fatalf("unexpected bounds: %v", got)
	}
	if img.At(20, 20) == img.At(20, 60) {
		t.Fatalf("expected grid cells to differ from background")
	}
}

func TestGenerateAndStoreWritesPNG(t *testing.T) {
	store := &fakeStorage{}
	gen := NewGenerator(store, nil)

	url, err := gen.GenerateAndStore(context.Background(), "chat_group", []string{"user_a", "user_b"})
	if err != nil {
		t.Fatalf("generate and store: %v", err)
	}
	if url != "/media/im/group-avatars/chat_group.png" {
		t.Fatalf("unexpected url: %q", url)
	}
	if store.key != "im/group-avatars/chat_group.png" {
		t.Fatalf("unexpected key: %q", store.key)
	}
	if store.contentType != "image/png" {
		t.Fatalf("unexpected content type: %q", store.contentType)
	}
	if _, err := png.Decode(bytes.NewReader(store.data)); err != nil {
		t.Fatalf("stored data is not png: %v", err)
	}
}

func TestLoadLocalImageFromMediaURL(t *testing.T) {
	rootDir := t.TempDir()
	imagePath := filepath.Join(rootDir, "im", "images", "avatar.png")
	if err := os.MkdirAll(filepath.Dir(imagePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
		}
	}
	file, err := os.Create(imagePath)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := png.Encode(file, src); err != nil {
		_ = file.Close()
		t.Fatalf("encode image: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close image: %v", err)
	}

	gen := NewGenerator(&fakeStorage{rootDir: rootDir, urlPrefix: "/media"}, nil)
	img := gen.loadImage(context.Background(), "/media/im/images/avatar.png")
	if img == nil {
		t.Fatalf("expected local image")
	}
}
