package upload

import (
	"net/url"
	"strings"
)

// 阿里云 OSS 图片处理：等比缩放，宽不超过 1920px
const imageResizeProcess = "x-oss-process=image/resize,w_960,m_lfit"

// displayImageURL 为展示用 URL 追加 OSS 图片裁剪参数
func displayImageURL(raw string) string {
	return imageURLWithProcess(raw, imageResizeProcess)
}

func imageURLWithProcess(raw string, process string) string {
	if raw == "" {
		return ""
	}
	clean := stripOSSProcess(raw)
	sep := "?"
	if strings.Contains(clean, "?") {
		sep = "&"
	}
	return clean + sep + process
}

func stripOSSProcess(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("x-oss-process")
	u.RawQuery = q.Encode()
	return u.String()
}
