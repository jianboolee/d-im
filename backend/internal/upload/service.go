package upload

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const (
	maxImageSize   = 20 << 20 // 20MB
	maxImagesCount = 9
)

var allowedImageMIMEs = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

type Service struct {
	oss *OSSClient
}

func NewService(ossClient *OSSClient) *Service {
	return &Service{oss: ossClient}
}

func (s *Service) UploadImage(ctx context.Context, file *multipart.FileHeader) (*ImageResult, error) {
	if s.oss == nil {
		return nil, ErrStorageUnavailable
	}
	return s.uploadOne(ctx, file)
}

func (s *Service) UploadImages(ctx context.Context, files []*multipart.FileHeader) ([]*ImageResult, error) {
	if len(files) == 0 {
		return nil, ErrNoFile
	}
	if len(files) > maxImagesCount {
		return nil, ErrTooManyFiles
	}

	results := make([]*ImageResult, 0, len(files))
	for _, file := range files {
		result, err := s.UploadImage(ctx, file)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (s *Service) uploadOne(
	ctx context.Context,
	file *multipart.FileHeader,
) (*ImageResult, error) {
	if file.Size > maxImageSize {
		return nil, ErrFileTooLarge
	}

	f, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	peek := make([]byte, 512)
	n, err := f.Read(peek)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("read file: %w", err)
	}
	mime := mimetype.Detect(peek[:n]).String()
	if !allowedImageMIMEs[mime] {
		return nil, ErrInvalidImage
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek file: %w", err)
	}

	filename := "image" + extFromMIME(mime)

	url, fileID, objectKey, err := s.oss.UploadImage(ctx, f, filename)
	if err != nil {
		return nil, err
	}

	ext := strings.TrimPrefix(extFromMIME(mime), ".")

	res := &ImageResult{
		URL:         displayImageURL(url),
		OriginalURL: url,
		FileID:      fileID,
		ObjectKey:   objectKey,
		Filename:    file.Filename,
		MimeType:    mime,
		Format:      ext,
		Extension:   "." + ext,
		Size:        file.Size,
	}
	if w, h, err := s.oss.GetImageInfo(ctx, objectKey); err == nil {
		res.Width = w
		res.Height = h
	}

	return res, nil
}

func filenameFromURL(u *url.URL, fallbackExt string) string {
	name := filepath.Base(u.Path)
	if name == "." || name == "/" || name == "" {
		name = "image" + fallbackExt
	}
	if filepath.Ext(name) == "" && fallbackExt != "" {
		name += fallbackExt
	}
	return name
}

func extFromMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	default:
		return ".img"
	}
}
