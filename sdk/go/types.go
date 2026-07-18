package dimsdk

// UserData 用户同步数据
type UserData struct {
	UserID   string                 `json:"-"`
	Nickname string                 `json:"nickname"`
	Avatar   string                 `json:"avatar_url"`
	Status   string                 `json:"status"` // active / disabled
	Version  int64                  `json:"version"`
	Ext      map[string]interface{} `json:"ext,omitempty"`
}

type userSnapshot struct {
	Nickname string                 `json:"nickname"`
	Avatar   string                 `json:"avatar_url"`
	Status   string                 `json:"status"`
	Version  int64                  `json:"version"`
	Ext      map[string]interface{} `json:"ext,omitempty"`
}

// Conversation 会话的基本信息。
type Conversation struct {
	ConversationID string `json:"conversation_id"`
	ChatID         string `json:"chat_id"`
	ChatType       string `json:"chat_type"`
}

// SendMessageReq 发送消息请求。
type SendMessageReq struct {
	ChatID          string      `json:"chat_id"`
	MessageType     string      `json:"message_type"`
	Content         interface{} `json:"content"`
	ClientMessageID string      `json:"client_message_id,omitempty"`
	ClientTime      string      `json:"client_time,omitempty"`
	QuoteMessageID  string      `json:"quote_message_id,omitempty"`
}

// SendMessageResp 发送消息响应
type SendMessageResp struct {
	Status          string `json:"status"`
	ChatID          string `json:"chat_id"`
	ClientMessageID string `json:"client_message_id"`
}
