package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"
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

// GenerateObjectID 生成媒体对象 ID。媒体 ID 独立于业务 snowflake，使用 UUID v7。
func GenerateObjectID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	b[6] = (b[6] & 0x0f) | 0x70
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16],
	), nil
}

func JoinURL(baseURL, pathValue string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	pathValue = "/" + strings.TrimLeft(pathValue, "/")
	return baseURL + pathValue
}
