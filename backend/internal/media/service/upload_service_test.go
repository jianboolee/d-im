package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"testing"

	"d-im/internal/media/storage"
)

func TestUploadImageReturnsDimensions(t *testing.T) {
	store, err := storage.NewLocalStorage(storage.LocalConfig{
		RootDir:       t.TempDir(),
		URLPrefix:     "/media",
		PublicBaseURL: "http://localhost:8080",
	})
	if err != nil {
		t.Fatalf("new local storage: %v", err)
	}
	svc := NewUploadService(store, DefaultMaxImageSize)

	header, err := imageFileHeader("avatar.png", 3, 2)
	if err != nil {
		t.Fatalf("build image file: %v", err)
	}

	uploaded, err := svc.UploadImage(context.Background(), header)
	if err != nil {
		t.Fatalf("upload image: %v", err)
	}

	if uploaded.Width != 3 || uploaded.Height != 2 {
		t.Fatalf("expected dimensions 3x2, got %dx%d", uploaded.Width, uploaded.Height)
	}
	if uploaded.Format != "png" {
		t.Fatalf("expected png format, got %q", uploaded.Format)
	}
	if uploaded.URL == "" || uploaded.Filename != "avatar.png" || uploaded.Size <= 0 {
		t.Fatalf("unexpected uploaded file: %#v", uploaded)
	}
	if uploaded.MediaID == "" || uploaded.Key == "" || uploaded.Provider != storage.ProviderLocal {
		t.Fatalf("expected media id, key and local provider, got %#v", uploaded)
	}
}

func imageFileHeader(filename string, width, height int) (*multipart.FileHeader, error) {
	var imageBuf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	if err := png.Encode(&imageBuf, img); err != nil {
		return nil, err
	}

	var multipartBuf bytes.Buffer
	writer := multipart.NewWriter(&multipartBuf)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(imageBuf.Bytes()); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, "/upload", &multipartBuf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(DefaultMaxImageSize); err != nil {
		return nil, err
	}
	return req.MultipartForm.File["file"][0], nil
}
