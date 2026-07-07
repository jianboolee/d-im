package avatar

import (
	"bytes"
	"context"
	"fmt"
	"hash/fnv"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d-im/internal/media/storage"
	"d-im/pkg/model"
)

const (
	defaultSize = 256
)

type UserReader interface {
	FindByID(ctx context.Context, id string) (*model.User, error)
}

type Generator struct {
	store      storage.Storage
	users      UserReader
	httpClient *http.Client
}

type localStorageReader interface {
	RootDir() string
	URLPrefix() string
}

func NewGenerator(store storage.Storage, users UserReader) *Generator {
	return &Generator{
		store: store,
		users: users,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (g *Generator) GenerateAndStore(ctx context.Context, chatID string, memberUIDs []string) (string, error) {
	if g == nil || g.store == nil {
		return "", fmt.Errorf("avatar storage is required")
	}
	if chatID == "" {
		return "", fmt.Errorf("chat_id is required")
	}

	img := g.Generate(ctx, chatID, memberUIDs)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	key := fmt.Sprintf("im/group-avatars/%s.png", chatID)
	stored, err := g.store.Put(ctx, storage.Object{
		Key:         key,
		Reader:      bytes.NewReader(buf.Bytes()),
		Size:        int64(buf.Len()),
		ContentType: "image/png",
		FileName:    chatID + ".png",
	})
	if err != nil {
		return "", err
	}
	return stored.URL, nil
}

func (g *Generator) Generate(ctx context.Context, chatID string, memberUIDs []string) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, defaultSize, defaultSize))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.RGBA{R: 245, G: 246, B: 248, A: 255}), image.Point{}, draw.Src)

	items := firstNonEmpty(memberUIDs, 9)
	if len(items) == 0 {
		items = []string{chatID}
	}

	rects := layoutRects(len(items), defaultSize)
	for i, uid := range items {
		rect := rects[i]
		src := g.loadUserAvatar(ctx, uid)
		if src == nil {
			fillRect(dst, rect, fallbackColor(uid))
			continue
		}
		drawCropped(dst, rect, src)
	}

	return dst
}

func (g *Generator) loadUserAvatar(ctx context.Context, uid string) image.Image {
	if g == nil || g.users == nil || uid == "" {
		return nil
	}
	user, err := g.users.FindByID(ctx, uid)
	if err != nil || user == nil || strings.TrimSpace(user.Avatar) == "" {
		return nil
	}
	return g.loadImage(ctx, user.Avatar)
}

func (g *Generator) loadImage(ctx context.Context, rawURL string) image.Image {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil
	}
	if img := g.loadLocalImage(rawURL); img != nil {
		return img
	}
	if !(strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://")) {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil
	}
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil
	}

	limited := io.LimitReader(resp.Body, 5<<20)
	img, _, err := image.Decode(limited)
	if err != nil {
		return nil
	}
	return img
}

func (g *Generator) loadLocalImage(rawURL string) image.Image {
	local, ok := g.store.(localStorageReader)
	if !ok || local.RootDir() == "" {
		return nil
	}

	pathValue := rawURL
	if parsed, err := url.Parse(rawURL); err == nil && parsed.Path != "" {
		pathValue = parsed.Path
	}

	prefix := strings.TrimRight(local.URLPrefix(), "/")
	if prefix == "" {
		prefix = "/media"
	}
	if !strings.HasPrefix(pathValue, prefix+"/") {
		return nil
	}

	key := strings.TrimPrefix(pathValue, prefix+"/")
	cleanKey := filepath.Clean(filepath.FromSlash(key))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") || filepath.IsAbs(cleanKey) {
		return nil
	}

	rootDir := local.RootDir()
	targetPath := filepath.Join(rootDir, cleanKey)
	rel, err := filepath.Rel(rootDir, targetPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return nil
	}

	file, err := os.Open(targetPath)
	if err != nil {
		return nil
	}
	defer file.Close()

	img, _, err := image.Decode(io.LimitReader(file, 5<<20))
	if err != nil {
		return nil
	}
	return img
}

func firstNonEmpty(items []string, limit int) []string {
	result := make([]string, 0, limit)
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
		if len(result) == limit {
			break
		}
	}
	return result
}

func layoutRects(count, canvas int) []image.Rectangle {
	if count <= 0 {
		return nil
	}
	if count > 9 {
		count = 9
	}

	switch count {
	case 1:
		return buildRows(canvas, 180, 0, []int{1})
	case 2:
		return buildRows(canvas, 112, 8, []int{2})
	case 3:
		return buildRows(canvas, 76, 8, []int{1, 2})
	case 4:
		return buildRows(canvas, 112, 8, []int{2, 2})
	case 5:
		return buildRows(canvas, 76, 6, []int{2, 3})
	case 6:
		return buildRows(canvas, 76, 6, []int{3, 3})
	case 7:
		return buildRows(canvas, 72, 6, []int{1, 3, 3})
	case 8:
		return buildRows(canvas, 72, 6, []int{2, 3, 3})
	default:
		return buildRows(canvas, 72, 6, []int{3, 3, 3})
	}
}

func buildRows(canvas, cellSize, gap int, rows []int) []image.Rectangle {
	rects := make([]image.Rectangle, 0, 9)
	totalHeight := len(rows)*cellSize + (len(rows)-1)*gap
	y := (canvas - totalHeight) / 2
	for _, cols := range rows {
		totalWidth := cols*cellSize + (cols-1)*gap
		x := (canvas - totalWidth) / 2
		for i := 0; i < cols; i++ {
			rects = append(rects, image.Rect(x, y, x+cellSize, y+cellSize))
			x += cellSize + gap
		}
		y += cellSize + gap
	}
	return rects
}

func fillRect(dst draw.Image, rect image.Rectangle, c color.Color) {
	draw.Draw(dst, rect, image.NewUniform(c), image.Point{}, draw.Src)
}

func fallbackColor(seed string) color.RGBA {
	palette := []color.RGBA{
		{R: 78, G: 121, B: 167, A: 255},
		{R: 242, G: 142, B: 44, A: 255},
		{R: 89, G: 161, B: 79, A: 255},
		{R: 225, G: 87, B: 89, A: 255},
		{R: 118, G: 183, B: 178, A: 255},
		{R: 176, G: 122, B: 161, A: 255},
		{R: 255, G: 157, B: 167, A: 255},
		{R: 156, G: 117, B: 95, A: 255},
		{R: 186, G: 176, B: 172, A: 255},
	}
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(seed))
	return palette[int(hash.Sum32())%len(palette)]
}

func drawCropped(dst draw.Image, rect image.Rectangle, src image.Image) {
	srcBounds := src.Bounds()
	side := srcBounds.Dx()
	if srcBounds.Dy() < side {
		side = srcBounds.Dy()
	}
	if side <= 0 {
		fillRect(dst, rect, color.RGBA{R: 220, G: 224, B: 230, A: 255})
		return
	}

	srcX := srcBounds.Min.X + (srcBounds.Dx()-side)/2
	srcY := srcBounds.Min.Y + (srcBounds.Dy()-side)/2
	srcRect := image.Rect(srcX, srcY, srcX+side, srcY+side)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			sx := srcRect.Min.X + (x-rect.Min.X)*srcRect.Dx()/rect.Dx()
			sy := srcRect.Min.Y + (y-rect.Min.Y)*srcRect.Dy()/rect.Dy()
			dst.Set(x, y, src.At(sx, sy))
		}
	}
}
