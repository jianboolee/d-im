package types

import (
	"strings"
	"testing"
)

func TestBuildContentPreviewForTextMessage(t *testing.T) {
	preview := BuildContentPreview(MessageTypeText, TextContent{Text: "hello"})
	if preview != "hello" {
		t.Fatalf("expected hello, got %q", preview)
	}
}

func TestBuildContentPreviewTruncatesRunes(t *testing.T) {
	preview := BuildContentPreview(MessageTypeText, TextContent{Text: strings.Repeat("你", 51)})
	if len([]rune(preview)) != 53 {
		t.Fatalf("expected 50 runes plus ellipsis, got %q", preview)
	}
}

func TestBuildContentPreviewFile(t *testing.T) {
	preview := BuildContentPreview(MessageTypeFile, FileContent{FileName: "a.pdf"})
	if preview != "[文件] a.pdf" {
		t.Fatalf("expected file preview, got %q", preview)
	}
}

func TestBuildContentPreviewCardUsesTitle(t *testing.T) {
	preview := BuildContentPreview(MessageTypeCard, CardContent{Title: "商品标题"})
	if preview != "商品标题" {
		t.Fatalf("expected card title, got %q", preview)
	}
}

func TestBuildContentPreviewTemplateUsesTitle(t *testing.T) {
	preview := BuildContentPreview(MessageTypeTemplate, TemplateContent{Title: "订单已支付"})
	if preview != "订单已支付" {
		t.Fatalf("expected template title, got %q", preview)
	}
}

func TestBuildContentPreviewUnknownType(t *testing.T) {
	preview := BuildContentPreview(MessageType("pay"), map[string]interface{}{"title": "支付成功"})
	if preview != "[消息]" {
		t.Fatalf("expected default preview, got %q", preview)
	}
}
