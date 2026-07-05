package sdk

import (
	"encoding/json"
	"fmt"
)

// UpsertUser 同步用户（创建或更新）
// 不依赖 NATS，业务系统直接通过 HTTP 调用
func (c *Client) UpsertUser(user UserData) error {
	respBody, err := c.do("POST", "/api/v1/sdk/user/sync", user)
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	var resp SyncUserResp
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// BatchSyncUsers 批量同步用户
func (c *Client) BatchSyncUsers(users []UserData) error {
	_, err := c.do("POST", "/api/v1/sdk/user/batch-sync", struct {
		Users []UserData `json:"users"`
	}{Users: users})
	return err
}
