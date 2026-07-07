package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

func TestAliyunOSSStoragePutDeleteAndURL(t *testing.T) {
	bucket := &fakeAliyunOSSBucket{}
	store := &AliyunOSSStorage{
		endpoint:      "https://oss-cn-hangzhou.aliyuncs.com",
		bucketName:    "im-media",
		bucket:        bucket,
		directory:     "media",
		publicBaseURL: "https://cdn.example.com",
	}

	stored, err := store.Put(context.Background(), Object{
		Key:         "im/images/test.txt",
		Reader:      strings.NewReader("hello"),
		ContentType: "text/plain",
		FileName:    "test.txt",
	})
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	if bucket.putKey != "media/im/images/test.txt" {
		t.Fatalf("unexpected put key %q", bucket.putKey)
	}
	if bucket.putBody != "hello" {
		t.Fatalf("unexpected put body %q", bucket.putBody)
	}
	contentType, err := oss.FindOption(bucket.putOptions, oss.HTTPHeaderContentType, "")
	if err != nil {
		t.Fatalf("find content type option: %v", err)
	}
	if contentType != "text/plain" {
		t.Fatalf("expected content type text/plain, got %q", contentType)
	}
	if stored.Provider != ProviderAliyunOSS || stored.Bucket != "im-media" {
		t.Fatalf("unexpected stored object: %#v", stored)
	}
	if stored.URL != "https://cdn.example.com/media/im/images/test.txt" {
		t.Fatalf("unexpected public url %q", stored.URL)
	}

	if err := store.Delete(context.Background(), "im/images/test.txt"); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if bucket.deleteKey != "media/im/images/test.txt" {
		t.Fatalf("unexpected delete key %q", bucket.deleteKey)
	}

	publicURL, err := store.URL(context.Background(), "im/images/test.txt")
	if err != nil {
		t.Fatalf("object url: %v", err)
	}
	if publicURL != "https://cdn.example.com/media/im/images/test.txt" {
		t.Fatalf("unexpected url %q", publicURL)
	}
}

func TestAliyunOSSStorageRequiresConfig(t *testing.T) {
	_, err := NewAliyunOSSStorage(AliyunOSSConfig{})
	if err == nil {
		t.Fatal("expected config error")
	}
}

func TestNormalizeOSSDirectoryDefaultsToMedia(t *testing.T) {
	if got := normalizeOSSDirectory(""); got != "media" {
		t.Fatalf("unexpected default directory %q", got)
	}
}

type fakeAliyunOSSBucket struct {
	putKey     string
	putBody    string
	putOptions []oss.Option
	deleteKey  string
	deleteOpts []oss.Option
}

func (b *fakeAliyunOSSBucket) PutObject(objectKey string, reader io.Reader, options ...oss.Option) error {
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	b.putKey = objectKey
	b.putBody = string(data)
	b.putOptions = options
	return nil
}

func (b *fakeAliyunOSSBucket) DeleteObject(objectKey string, options ...oss.Option) error {
	b.deleteKey = objectKey
	b.deleteOpts = options
	return nil
}
