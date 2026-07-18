package dimsdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// UpsertUser 将业务系统的完整用户快照同步到 IM。
// Version 必须随用户变更单调递增，重试同一版本是幂等的。
func (c *Client) UpsertUser(ctx context.Context, user UserData) error {
	if user.UserID == "" {
		return fmt.Errorf("upsert user: user ID is required")
	}
	if user.Version <= 0 {
		return fmt.Errorf("upsert user: version must be positive")
	}
	if user.Status != "active" && user.Status != "disabled" {
		return fmt.Errorf("upsert user: status must be active or disabled")
	}
	respBody, err := c.do(ctx, http.MethodPut, "/api/v1/sdk/users/"+url.PathEscape(user.UserID), userSnapshot{
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
		Status:   user.Status,
		Version:  user.Version,
		Ext:      user.Ext,
	})
	if err != nil {
		return fmt.Errorf("upsert user: %w", err)
	}

	var resp struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("upsert user: unmarshal response: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("upsert user: %s", resp.Error)
	}
	return nil
}
