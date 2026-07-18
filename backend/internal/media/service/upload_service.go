package service

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"d-im/internal/media/storage"
)

const (
	DefaultMaxImageSize = 20 << 20
)

type UploadService struct {
	storage      storage.Storage
	maxImageSize int64
}

type UploadedFile struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Format   string `json:"format"`
	MediaID  string `json:"media_id,omitempty"`
	Key      string `json:"key,omitempty"`
	Provider string `json:"provider,omitempty"`
}

func NewUploadService(store storage.Storage, maxImageSize int64) *UploadService {
	if maxImageSize <= 0 {
		maxImageSize = DefaultMaxImageSize
	}
	return &UploadService{storage: store, maxImageSize: maxImageSize}
}

func (s *UploadService) MaxImageSize() int64 {
	if s == nil || s.maxImageSize <= 0 {
		return DefaultMaxImageSize
	}
	return s.maxImageSize
}

func (s *UploadService) UploadImage(ctx context.Context, header *multipart.FileHeader) (*UploadedFile, error) {
	if header == nil {
		return nil, fmt.Errorf("file is required")
	}
	if header.Size <= 0 {
		return nil, fmt.Errorf("file is empty")
	}
	if header.Size > s.maxImageSize {
		return nil, fmt.Errorf("image too large")
	}

	file, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	contentType, err := detectContentType(file)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("file is not an image")
	}

	width, height, format, err := readImageConfig(file)
	if err != nil {
		return nil, fmt.Errorf("invalid image: %w", err)
	}
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid image dimensions")
	}

	mediaID, err := storage.NewObjectID()
	if err != nil {
		return nil, err
	}
	ext := imageExtension(header.Filename, format, contentType)
	key := buildImageKey(time.Now(), mediaID, ext)

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}
	stored, err := s.storage.Put(ctx, storage.Object{
		Key:         key,
		Reader:      file,
		Size:        header.Size,
		ContentType: contentType,
		FileName:    header.Filename,
	})
	if err != nil {
		return nil, err
	}

	return &UploadedFile{
		URL:      stored.URL,
		Filename: header.Filename,
		Size:     stored.Size,
		Width:    width,
		Height:   height,
		Format:   format,
		MediaID:  mediaID,
		Key:      key,
		Provider: stored.Provider,
	}, nil
}

func detectContentType(file multipart.File) (string, error) {
	var buf [512]byte
	n, err := file.Read(buf[:])
	if err != nil && n == 0 {
		return "", err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

func readImageConfig(file multipart.File) (int, int, string, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, "", err
	}
	cfg, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, "", err
	}
	if _, err := file.Seek(0, 0); err != nil {
		return 0, 0, "", err
	}
	return cfg.Width, cfg.Height, format, nil
}

func buildImageKey(now time.Time, mediaID, ext string) string {
	return fmt.Sprintf("im/images/%04d/%02d/%02d/%s%s", now.Year(), now.Month(), now.Day(), mediaID, ext)
}

func imageExtension(filename, format, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return ext
	}
	switch format {
	case "jpeg":
		return ".jpg"
	case "png":
		return ".png"
	case "gif":
		return ".gif"
	}
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
