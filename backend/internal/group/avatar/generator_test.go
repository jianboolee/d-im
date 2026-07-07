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
	firstRect := layoutRects(5, 256)[0]
	center := image.Pt((firstRect.Min.X+firstRect.Max.X)/2, (firstRect.Min.Y+firstRect.Max.Y)/2)
	if img.At(20, 20) == img.At(center.X, center.Y) {
		t.Fatalf("expected grid cells to differ from background")
	}
}

func TestLayoutRectsUsesBalancedRows(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		rowSizes []int
	}{
		{name: "one", count: 1, rowSizes: []int{1}},
		{name: "three", count: 3, rowSizes: []int{1, 2}},
		{name: "five", count: 5, rowSizes: []int{2, 3}},
		{name: "seven", count: 7, rowSizes: []int{1, 3, 3}},
		{name: "eight", count: 8, rowSizes: []int{2, 3, 3}},
		{name: "nine", count: 9, rowSizes: []int{3, 3, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rects := layoutRects(tt.count, 256)
			if len(rects) != tt.count {
				t.Fatalf("expected %d rects, got %d", tt.count, len(rects))
			}

			rows := groupRectsByY(rects)
			if len(rows) != len(tt.rowSizes) {
				t.Fatalf("expected %d rows, got %d", len(tt.rowSizes), len(rows))
			}
			for i, expected := range tt.rowSizes {
				if len(rows[i]) != expected {
					t.Fatalf("row %d expected %d rects, got %d", i, expected, len(rows[i]))
				}
			}

			firstRow := rows[0]
			left := firstRow[0].Min.X
			right := firstRow[len(firstRow)-1].Max.X
			if left != 256-right {
				t.Fatalf("first row is not centered: left=%d right_space=%d", left, 256-right)
			}
		})
	}
}

func groupRectsByY(rects []image.Rectangle) [][]image.Rectangle {
	rows := make([][]image.Rectangle, 0)
	for _, rect := range rects {
		if len(rows) == 0 || rows[len(rows)-1][0].Min.Y != rect.Min.Y {
			rows = append(rows, []image.Rectangle{rect})
			continue
		}
		rows[len(rows)-1] = append(rows[len(rows)-1], rect)
	}
	return rows
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
