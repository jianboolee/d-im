package upload

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	gonanoid "github.com/matoous/go-nanoid"
)

// OSSClient 阿里云 OSS 客户端
type OSSClient struct {
	client       *oss.Client
	bucket       *oss.Bucket
	bucketName   string
	customDomain string
	directory    string
	log          *slog.Logger
}

// NewOSSClient 创建 OSS 客户端
func NewOSSClient(cfg *StorageConfig, log *slog.Logger) (*OSSClient, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("OSS Endpoint 未配置")
	}
	if cfg.AccessKeyID == "" {
		return nil, fmt.Errorf("OSS AccessKey ID 未配置")
	}
	if cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("OSS AccessKey Secret 未配置")
	}
	if cfg.BucketName == "" {
		return nil, fmt.Errorf("OSS Bucket 名称未配置")
	}
	if log == nil {
		log = slog.Default()
	}

	endpoint := strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://")

	client, err := oss.New(endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("初始化 OSS 客户端失败: %w", err)
	}

	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS Bucket 失败: %w", err)
	}

	directory := cfg.Directory
	if directory == "" {
		directory = "/uploads"
	}
	if !strings.HasPrefix(directory, "/") {
		directory = "/" + directory
	}
	directory = strings.TrimSuffix(directory, "/")

	c := &OSSClient{
		client:       client,
		bucket:       bucket,
		bucketName:   cfg.BucketName,
		customDomain: strings.TrimSuffix(cfg.CustomDomain, "/"),
		directory:    directory,
		log:          log,
	}
	c.log.Info("OSS 客户端初始化成功",
		"endpoint", endpoint,
		"bucket", cfg.BucketName,
		"directory", directory,
	)
	return c, nil
}

func (c *OSSClient) generateObjectKey(filename, subdir string) (objectKey, fileID string, err error) {
	ext := filepath.Ext(filename)
	fileID, err = gonanoid.Generate("0123456789abcdefghijklmnopqrstuvwxyz", 32)
	if err != nil {
		return "", "", fmt.Errorf("生成文件 ID 失败: %w", err)
	}

	newFilename := fileID + ext
	timestamp := time.Now().Format("20060102")
	baseDir := c.directory
	if subdir != "" {
		subdir = strings.Trim(subdir, "/")
		baseDir = fmt.Sprintf("%s/%s", baseDir, subdir)
	}
	objectKey = fmt.Sprintf("%s/%s/%s", baseDir, timestamp, newFilename)
	return strings.TrimPrefix(objectKey, "/"), fileID, nil
}

func (c *OSSClient) buildFileURL(objectKey string) string {
	if c.customDomain != "" {
		return fmt.Sprintf("%s/%s", c.customDomain, objectKey)
	}
	return fmt.Sprintf("https://%s.%s/%s", c.bucketName, c.client.Config.Endpoint, objectKey)
}

func (c *OSSClient) IsCustomDomainURL(raw string) bool {
	if c == nil || c.customDomain == "" || raw == "" {
		return false
	}
	custom, err := url.Parse(c.customDomain)
	if err != nil || custom.Host == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, custom.Host)
}

// UploadImage 上传图片到 images 子目录
func (c *OSSClient) UploadImage(ctx context.Context, reader io.Reader, filename string) (url, fileID, objectKey string, err error) {
	return c.upload(ctx, reader, filename, "images")
}

// UploadAttachmentImage 上传业务附件图片到 attachments 子目录，不进入素材库
func (c *OSSClient) UploadAttachmentImage(ctx context.Context, reader io.Reader, filename string) (url, fileID, objectKey string, err error) {
	return c.upload(ctx, reader, filename, "attachments")
}

func (c *OSSClient) upload(ctx context.Context, reader io.Reader, filename, subdir string) (url, fileID, objectKey string, err error) {
	_ = ctx
	objectKey, fileID, err = c.generateObjectKey(filename, subdir)
	if err != nil {
		return "", "", "", err
	}

	if err := c.bucket.PutObject(objectKey, reader); err != nil {
		c.log.Error("上传 OSS 失败", "object_key", objectKey, "error", err)
		return "", "", "", fmt.Errorf("上传文件失败: %w", err)
	}

	url = c.buildFileURL(objectKey)
	c.log.Info("文件上传成功", "object_key", objectKey, "file_id", fileID, "url", url)
	return url, fileID, objectKey, nil
}

type imageInfo struct {
	ImageWidth  any `json:"ImageWidth"`
	ImageHeight any `json:"ImageHeight"`
}

// GetImageInfo 通过 OSS 图片处理接口获取宽高
func (c *OSSClient) GetImageInfo(ctx context.Context, objectKey string) (width, height int, err error) {
	ossURL := fmt.Sprintf("https://%s.%s/%s?x-oss-process=image/info",
		c.bucketName, c.client.Config.Endpoint, objectKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ossURL, nil)
	if err != nil {
		return 0, 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("获取图片信息失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("获取图片信息失败: HTTP %d", resp.StatusCode)
	}

	var info imageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, 0, fmt.Errorf("解析图片信息失败: %w", err)
	}

	return convertToInt(info.ImageWidth), convertToInt(info.ImageHeight), nil
}

func convertToInt(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	case map[string]any:
		if s, ok := val["value"].(string); ok {
			var result int
			if _, err := fmt.Sscanf(s, "%d", &result); err == nil {
				return result
			}
		}
		if n, ok := val["value"].(float64); ok {
			return int(n)
		}
	}
	return 0
}
