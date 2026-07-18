package dimsdk

import (
	"encoding/json"
	"fmt"
)

// CreateSingleConversation 创建或获取当前用户与指定用户的单聊会话。
func (c *Client) CreateSingleConversation(accessToken, peerUserID string) (*Conversation, error) {
	respBody, err := c.doWithToken("POST", "/api/v1/conversations/single", map[string]string{
		"peer_user_id": peerUserID,
	}, accessToken)
	if err != nil {
		return nil, fmt.Errorf("create single conversation: %w", err)
	}

	var resp struct {
		Code  int          `json:"code"`
		Data  Conversation `json:"data"`
		Error string       `json:"error"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal conversation: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("create single conversation: %s", resp.Error)
	}
	return &resp.Data, nil
}

// SendMessage 向已存在的会话发送消息。
func (c *Client) SendMessage(accessToken string, req SendMessageReq) (*SendMessageResp, error) {
	respBody, err := c.doWithToken("POST", "/api/v1/messages", req, accessToken)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}

	var resp struct {
		Code  int             `json:"code"`
		Data  SendMessageResp `json:"data"`
		Error string          `json:"error"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal send message response: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("send message: %s", resp.Error)
	}
	return &resp.Data, nil
}
