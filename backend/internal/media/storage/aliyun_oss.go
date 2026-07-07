package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const ProviderAliyunOSS = "aliyun_oss"

type AliyunOSSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Bucket          string
	Directory       string
	PublicBaseURL   string
}

type AliyunOSSStorage struct {
	endpoint      string
	bucketName    string
	bucket        aliyunOSSBucket
	directory     string
	publicBaseURL string
}

type aliyunOSSBucket interface {
	PutObject(objectKey string, reader io.Reader, options ...oss.Option) error
	DeleteObject(objectKey string, options ...oss.Option) error
}

func NewAliyunOSSStorage(cfg AliyunOSSConfig) (*AliyunOSSStorage, error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		return nil, fmt.Errorf("aliyun oss endpoint is required")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("aliyun oss access_key_id is required")
	}
	if cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("aliyun oss access_key_secret is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("aliyun oss bucket is required")
	}

	endpoint = strings.TrimRight(endpoint, "/")
	client, err := oss.New(endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(cfg.Bucket)
	if err != nil {
		return nil, err
	}

	return &AliyunOSSStorage{
		endpoint:      endpoint,
		bucketName:    cfg.Bucket,
		bucket:        bucket,
		directory:     normalizeOSSDirectory(cfg.Directory),
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
	}, nil
}

func (s *AliyunOSSStorage) Put(ctx context.Context, obj Object) (*StoredObject, error) {
	if obj.Key == "" {
		return nil, fmt.Errorf("object key is required")
	}

	body, err := io.ReadAll(readerWithContext(ctx, obj.Reader))
	if err != nil {
		return nil, err
	}
	ossKey := s.ossKey(obj.Key)
	options := make([]oss.Option, 0, 1)
	if obj.ContentType != "" {
		options = append(options, oss.ContentType(obj.ContentType))
	}
	options = append(options, oss.WithContext(ctx))
	if err := s.bucket.PutObject(ossKey, bytes.NewReader(body), options...); err != nil {
		return nil, err
	}

	return &StoredObject{
		Provider:    ProviderAliyunOSS,
		Bucket:      s.bucketName,
		Key:         obj.Key,
		URL:         s.publicURL(obj.Key),
		Size:        int64(len(body)),
		ContentType: obj.ContentType,
		FileName:    obj.FileName,
	}, nil
}

func (s *AliyunOSSStorage) Delete(ctx context.Context, key string) error {
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	return s.bucket.DeleteObject(s.ossKey(key), oss.WithContext(ctx))
}

func (s *AliyunOSSStorage) URL(_ context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("object key is required")
	}
	return s.publicURL(key), nil
}

func (s *AliyunOSSStorage) Provider() string {
	return ProviderAliyunOSS
}

func (s *AliyunOSSStorage) publicURL(key string) string {
	escapedKey := (&url.URL{Path: s.ossKey(key)}).EscapedPath()
	if s.publicBaseURL != "" {
		return JoinURL(s.publicBaseURL, escapedKey)
	}
	endpointURL, err := parseEndpoint(s.endpoint)
	if err != nil {
		return ""
	}
	endpointURL.Host = s.bucketName + "." + endpointURL.Host
	endpointURL.Path = strings.TrimRight(endpointURL.Path, "/") + "/" + escapedKey
	return endpointURL.String()
}

func (s *AliyunOSSStorage) ossKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.directory == "" {
		return key
	}
	return s.directory + "/" + key
}

func normalizeOSSDirectory(directory string) string {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		directory = "/media"
	}
	directory = strings.Trim(directory, "/")
	if directory == "." {
		return ""
	}
	return directory
}

func parseEndpoint(endpoint string) (*url.URL, error) {
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("invalid aliyun oss endpoint")
	}
	return parsed, nil
}
