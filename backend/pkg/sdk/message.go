package sdk

import (
	"encoding/json"
	"fmt"
)

// SendMessage 发送消息（通用方法）
func (c *Client) SendMessage(req SendMessageReq) (*SendMessageResp, error) {
	respBody, err := c.do("POST", "/api/v1/sdk/message/send", req)
	if err != nil {
		return nil, err
	}

	var resp SendMessageResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &resp, nil
}

// SendTextMessage 发送文本消息（快捷方法）
func (c *Client) SendTextMessage(fromUID, fromName, chatID string, targetUIDs []string, text string) (*SendMessageResp, error) {
	return c.SendMessage(SendMessageReq{
		FromUID:    fromUID,
		FromName:   fromName,
		ChatID:     chatID,
		ChatType:   "single",
		MsgType:    "text",
		Content:    map[string]string{"text": text},
		TargetUIDs: targetUIDs,
	})
}
