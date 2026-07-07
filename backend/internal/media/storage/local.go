package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const ProviderLocal = "local"

type LocalConfig struct {
	RootDir       string
	URLPrefix     string
	PublicBaseURL string
}

type LocalStorage struct {
	rootDir       string
	urlPrefix     string
	publicBaseURL string
}

func NewLocalStorage(cfg LocalConfig) (*LocalStorage, error) {
	rootDir := strings.TrimSpace(cfg.RootDir)
	if rootDir == "" {
		rootDir = "./data/media"
	}
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absRoot, 0o755); err != nil {
		return nil, err
	}

	urlPrefix := cfg.URLPrefix
	if urlPrefix == "" {
		urlPrefix = "/media"
	}
	if !strings.HasPrefix(urlPrefix, "/") {
		urlPrefix = "/" + urlPrefix
	}

	return &LocalStorage{
		rootDir:       absRoot,
		urlPrefix:     strings.TrimRight(urlPrefix, "/"),
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (s *LocalStorage) Put(ctx context.Context, obj Object) (*StoredObject, error) {
	if obj.Key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	targetPath, err := s.objectPath(obj.Key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	written, err := io.Copy(file, readerWithContext(ctx, obj.Reader))
	if err != nil {
		return nil, err
	}

	return &StoredObject{
		Provider:    ProviderLocal,
		Key:         obj.Key,
		URL:         s.publicURL(obj.Key),
		Size:        written,
		ContentType: obj.ContentType,
		FileName:    obj.FileName,
	}, nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	targetPath, err := s.objectPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStorage) URL(_ context.Context, key string) (string, error) {
	if _, err := s.objectPath(key); err != nil {
		return "", err
	}
	return s.publicURL(key), nil
}

func (s *LocalStorage) Provider() string {
	return ProviderLocal
}

func (s *LocalStorage) RootDir() string {
	return s.rootDir
}

func (s *LocalStorage) URLPrefix() string {
	return s.urlPrefix
}

func (s *LocalStorage) objectPath(key string) (string, error) {
	cleanKey := filepath.Clean(strings.TrimLeft(key, "/"))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") || filepath.IsAbs(cleanKey) {
		return "", fmt.Errorf("invalid object key")
	}
	targetPath := filepath.Join(s.rootDir, cleanKey)
	rel, err := filepath.Rel(s.rootDir, targetPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", fmt.Errorf("invalid object key")
	}
	return targetPath, nil
}

func (s *LocalStorage) publicURL(key string) string {
	escapedKey := (&url.URL{Path: strings.TrimLeft(key, "/")}).EscapedPath()
	pathValue := s.urlPrefix + "/" + escapedKey
	if s.publicBaseURL == "" {
		return pathValue
	}
	return JoinURL(s.publicBaseURL, pathValue)
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func readerWithContext(ctx context.Context, r io.Reader) io.Reader {
	if ctx == nil {
		return r
	}
	return &contextReader{ctx: ctx, r: r}
}

func (r *contextReader) Read(p []byte) (int, error) {
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
		return r.r.Read(p)
	}
}
