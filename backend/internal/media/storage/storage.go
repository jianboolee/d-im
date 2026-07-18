package storage

import (
	"context"
	"io"
	"strings"

	"github.com/google/uuid"
)

// Object 是待写入存储的对象。
type Object struct {
	Key         string
	Reader      io.Reader
	Size        int64
	ContentType string
	FileName    string
}

// StoredObject 是写入存储后的对象信息。
type StoredObject struct {
	Provider    string
	Bucket      string
	Key         string
	URL         string
	Size        int64
	ContentType string
	FileName    string
}

// Storage 抽象不同存储提供方，本地、阿里云 OSS、七牛都实现同一接口。
type Storage interface {
	Put(ctx context.Context, obj Object) (*StoredObject, error)
	Delete(ctx context.Context, key string) error
	URL(ctx context.Context, key string) (string, error)
	Provider() string
}

// NewObjectID 创建无前缀的 UUID v7 媒体对象 ID。
func NewObjectID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func JoinURL(baseURL, pathValue string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	pathValue = "/" + strings.TrimLeft(pathValue, "/")
	return baseURL + pathValue
}
