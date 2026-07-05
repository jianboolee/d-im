package sdk

// UserData 用户同步数据
type UserData struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname,omitempty"`
	Avatar   string `json:"avatar_url,omitempty"`
	Status   string `json:"status,omitempty"` // active / disabled
}

// SyncUserResp 同步用户响应
type SyncUserResp struct {
	Ok string `json:"status"`
}

// SendMessageReq 发送消息请求
type SendMessageReq struct {
	FromUID    string      `json:"from_uid"`
	FromName   string      `json:"from_name,omitempty"`
	ChatID     string      `json:"chat_id"`
	ChatType   string      `json:"chat_type"` // single / group
	MsgType    string      `json:"msg_type"`  // text / image / ...
	Content    interface{} `json:"content"`
	TargetUIDs []string    `json:"target_uids,omitempty"`
}

// SendMessageResp 发送消息响应
type SendMessageResp struct {
	MsgID      string `json:"msg_id"`
	ServerTime string `json:"server_time"`
	Status     string `json:"status"`
}
