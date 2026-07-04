package types

import (
	"fmt"
	"net/url"
)

// ============================================================
// 文本消息
// ============================================================

// TextContent 文本消息
type TextContent struct {
	Text     string   `bson:"text" json:"text"`
	Mentions []string `bson:"mentions,omitempty" json:"mentions,omitempty"` // @的用户ID列表
	IsAtAll  bool     `bson:"is_at_all" json:"is_at_all"`
}

func (t TextContent) Type() MessageType { return MessageTypeText }

func (t TextContent) Validate() error {
	if t.Text == "" {
		return fmt.Errorf("text content cannot be empty")
	}
	if len(t.Text) > 5000 {
		return fmt.Errorf("text content too long")
	}
	return nil
}

// ============================================================
// 图片消息
// ============================================================

// ImageContent 图片消息
type ImageContent struct {
	URL      string `bson:"url" json:"url"`
	ThumbURL string `bson:"thumb_url,omitempty" json:"thumb_url,omitempty"`
	Width    int    `bson:"width" json:"width"`
	Height   int    `bson:"height" json:"height"`
	Size     int64  `bson:"size" json:"size"`     // 文件大小(bytes)
	Format   string `bson:"format" json:"format"` // jpg, png, gif, webp
	MD5      string `bson:"md5,omitempty" json:"md5,omitempty"`
	FileName string `bson:"file_name,omitempty" json:"file_name,omitempty"`
}

func (i ImageContent) Type() MessageType { return MessageTypeImage }

func (i ImageContent) Validate() error {
	if i.URL == "" {
		return fmt.Errorf("image url cannot be empty")
	}
	if _, err := url.Parse(i.URL); err != nil {
		return fmt.Errorf("invalid image url")
	}
	return nil
}

// ============================================================
// 视频消息
// ============================================================

// VideoContent 视频消息
type VideoContent struct {
	URL      string `bson:"url" json:"url"`
	ThumbURL string `bson:"thumb_url,omitempty" json:"thumb_url,omitempty"`
	Duration int    `bson:"duration" json:"duration"` // 视频时长(秒)
	Width    int    `bson:"width" json:"width"`
	Height   int    `bson:"height" json:"height"`
	Size     int64  `bson:"size" json:"size"`
	Format   string `bson:"format" json:"format"` // mp4, avi, mov
	MD5      string `bson:"md5,omitempty" json:"md5,omitempty"`
}

func (v VideoContent) Type() MessageType { return MessageTypeVideo }

func (v VideoContent) Validate() error {
	if v.URL == "" {
		return fmt.Errorf("video url cannot be empty")
	}
	if v.Duration <= 0 {
		return fmt.Errorf("video duration must be positive")
	}
	return nil
}

// ============================================================
// 语音消息
// ============================================================

// VoiceContent 语音消息
type VoiceContent struct {
	URL      string `bson:"url" json:"url"`
	Duration int    `bson:"duration" json:"duration"` // 语音时长(秒)
	Size     int64  `bson:"size" json:"size"`
	Format   string `bson:"format" json:"format"` // aac, mp3, wav
	MD5      string `bson:"md5,omitempty" json:"md5,omitempty"`
}

func (v VoiceContent) Type() MessageType { return MessageTypeVoice }

func (v VoiceContent) Validate() error {
	if v.URL == "" {
		return fmt.Errorf("voice url cannot be empty")
	}
	if v.Duration <= 0 {
		return fmt.Errorf("voice duration must be positive")
	}
	return nil
}

// ============================================================
// 卡片消息
// ============================================================

// CardContent 卡片消息
type CardContent struct {
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	ImageURL    string `bson:"image_url,omitempty" json:"image_url,omitempty"`
	ActionURL   string `bson:"action_url,omitempty" json:"action_url,omitempty"` // 点击跳转链接
}

func (c CardContent) Type() MessageType { return MessageTypeCard }

func (c CardContent) Validate() error {
	if c.Title == "" {
		return fmt.Errorf("card title cannot be empty")
	}
	return nil
}

// ============================================================
// 链接消息
// ============================================================

// LinkContent 链接消息
type LinkContent struct {
	URL         string `bson:"url" json:"url"`
	Title       string `bson:"title" json:"title"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	ThumbURL    string `bson:"thumb_url,omitempty" json:"thumb_url,omitempty"`
	Favicon     string `bson:"favicon,omitempty" json:"favicon,omitempty"`
}

func (l LinkContent) Type() MessageType { return MessageTypeLink }

func (l LinkContent) Validate() error {
	if l.URL == "" {
		return fmt.Errorf("link url cannot be empty")
	}
	if _, err := url.Parse(l.URL); err != nil {
		return fmt.Errorf("invalid link url")
	}
	return nil
}

// ============================================================
// 模板消息
// ============================================================

// TemplateItem 模板消息中的单个项
type TemplateItem struct {
	Label     string `bson:"label" json:"label"`                               // 标签
	Value     string `bson:"value" json:"value"`                               // 值
	Type      string `bson:"type,omitempty" json:"type,omitempty"`             // text, link, image, money, date
	Color     string `bson:"color,omitempty" json:"color,omitempty"`           // 值的颜色
	ActionURL string `bson:"action_url,omitempty" json:"action_url,omitempty"` // 点击跳转(当type为link时)
}

// TemplateContent 模板消息（支持多个label-value对）
type TemplateContent struct {
	TemplateID  string         `bson:"template_id" json:"template_id"`                     // 模板ID
	Title       string         `bson:"title,omitempty" json:"title,omitempty"`             // 模板标题
	Items       []TemplateItem `bson:"items" json:"items"`                                 // 键值对列表
	Description string         `bson:"description,omitempty" json:"description,omitempty"` // 备注信息
	ActionURL   string         `bson:"action_url,omitempty" json:"action_url,omitempty"`   // 整个模板的跳转链接
	ActionText  string         `bson:"action_text,omitempty" json:"action_text,omitempty"` // 跳转链接文案
}

func (t TemplateContent) Type() MessageType { return MessageTypeTemplate }

func (t TemplateContent) Validate() error {
	if t.TemplateID == "" {
		return fmt.Errorf("template id cannot be empty")
	}
	if len(t.Items) == 0 {
		return fmt.Errorf("template items cannot be empty")
	}
	if len(t.Items) > 20 {
		return fmt.Errorf("template items cannot exceed 20")
	}
	for i, item := range t.Items {
		if item.Label == "" || item.Value == "" {
			return fmt.Errorf("template item %d: label and value cannot be empty", i)
		}
	}
	return nil
}

// ============================================================
// 文件消息
// ============================================================

// FileContent 文件消息
type FileContent struct {
	URL      string `bson:"url" json:"url"`
	FileName string `bson:"file_name" json:"file_name"`
	Size     int64  `bson:"size" json:"size"`
	Format   string `bson:"format" json:"format"` // pdf, doc, xlsx, zip
	MD5      string `bson:"md5,omitempty" json:"md5,omitempty"`
}

func (f FileContent) Type() MessageType { return MessageTypeFile }

func (f FileContent) Validate() error {
	if f.URL == "" {
		return fmt.Errorf("file url cannot be empty")
	}
	if f.FileName == "" {
		return fmt.Errorf("file name cannot be empty")
	}
	return nil
}

// ============================================================
// 位置消息
// ============================================================

// LocationContent 位置消息
type LocationContent struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`
	Longitude float64 `bson:"longitude" json:"longitude"`
	Address   string  `bson:"address,omitempty" json:"address,omitempty"`
	Name      string  `bson:"name,omitempty" json:"name,omitempty"`
}

func (l LocationContent) Type() MessageType { return MessageTypeLocation }

func (l LocationContent) Validate() error {
	if l.Latitude < -90 || l.Latitude > 90 {
		return fmt.Errorf("invalid latitude")
	}
	if l.Longitude < -180 || l.Longitude > 180 {
		return fmt.Errorf("invalid longitude")
	}
	return nil
}
