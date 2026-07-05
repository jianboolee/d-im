package sdk

import "encoding/json"

// GetLoginURL 获取 SSO 登录 URL（业务端可以直接跳转到此地址）
func (c *Client) GetLoginURL(uid string) (string, error) {
	respBody, err := c.do("POST", "/api/v1/auth/ticket", map[string]string{
		"uid": uid,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Ticket      string `json:"ticket"`
		RedirectURL string `json:"redirect_url"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.RedirectURL, nil
}
