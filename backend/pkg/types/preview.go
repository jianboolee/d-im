package types

func BuildContentPreview(msgType MessageType, content interface{}) string {
	switch c := content.(type) {
	case TextContent:
		return truncatePreview(c.Text, 50)
	case ImageContent:
		return "[图片]"
	case VideoContent:
		return "[视频]"
	case VoiceContent:
		return "[语音]"
	case FileContent:
		return "[文件] " + c.FileName
	case LocationContent:
		return "[位置]"
	case CardContent:
		if c.Title != "" {
			return c.Title
		}
		return previewByType(msgType)
	case LinkContent:
		return "[链接] " + c.Title
	case TemplateContent:
		if c.Title != "" {
			return c.Title
		}
		return previewByType(msgType)
	case map[string]interface{}:
		return previewFromMap(msgType, c)
	default:
		return previewByType(msgType)
	}
}

func previewFromMap(msgType MessageType, content map[string]interface{}) string {
	switch msgType {
	case MessageTypeText:
		if text, ok := content["text"].(string); ok {
			return truncatePreview(text, 50)
		}
	case MessageTypeFile:
		if fileName, ok := content["file_name"].(string); ok {
			return "[文件] " + fileName
		}
	case MessageTypeCard:
		if title, ok := content["title"].(string); ok {
			return title
		}
	case MessageTypeLink:
		if title, ok := content["title"].(string); ok {
			return "[链接] " + title
		}
	case MessageTypeTemplate:
		if title, ok := content["title"].(string); ok {
			return title
		}
	}
	return previewByType(msgType)
}

func previewByType(msgType MessageType) string {
	switch msgType {
	case MessageTypeImage:
		return "[图片]"
	case MessageTypeVideo:
		return "[视频]"
	case MessageTypeVoice:
		return "[语音]"
	case MessageTypeFile:
		return "[文件]"
	case MessageTypeLocation:
		return "[位置]"
	case MessageTypeCard:
		return "[消息]"
	case MessageTypeLink:
		return "[链接]"
	case MessageTypeTemplate:
		return "[消息]"
	default:
		return "[消息]"
	}
}

func truncatePreview(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "..."
}
