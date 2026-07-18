package dimsdk

import "encoding/json"

// GetLoginURL 获取 SSO 登录 URL（业务端可以直接跳转到此地址）
func (c *Client) GetLoginURL(id string) (string, error) {
	respBody, err := c.do("POST", "/api/v1/auth/ticket", map[string]string{
		"id": id,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
		Data  struct {
			Ticket      string `json:"ticket"`
			RedirectURL string `json:"redirect_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", err
	}

	return resp.Data.RedirectURL, nil
}
