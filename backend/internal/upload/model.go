package upload

// ImageResult 单张图片上传结果
type ImageResult struct {
	URL         string `json:"url"`          // 带 OSS 裁剪参数，适合前端展示
	OriginalURL string `json:"original_url"` // 原图地址
	FileID      string `json:"file_id"`
	MaterialID  string `json:"material_id,omitempty"`
	ObjectKey   string `json:"object_key"`
	Filename    string `json:"filename,omitempty"`
	MimeType    string `json:"mime_type,omitempty"`
	Format      string `json:"format,omitempty"`
	Extension   string `json:"extension,omitempty"`
	Size        int64  `json:"size,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
}
