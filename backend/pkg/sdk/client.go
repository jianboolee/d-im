package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client IM HTTP 客户端（业务系统使用）
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient 创建客户端
func NewClient(opts ClientOptions) *Client {
	return &Client{
		baseURL: opts.BaseURL,
		apiKey:  opts.APIKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TokenPair SDK 会话令牌
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// GetSession 为指定用户创建 API Session（使用 API Key 换取 JWT）
func (c *Client) GetSession(uid string) (*TokenPair, error) {
	respBody, err := c.do("POST", "/api/v1/auth/session", map[string]string{
		"uid": uid,
	})
	if err != nil {
		return nil, err
	}

	var pair TokenPair
	if err := json.Unmarshal(respBody, &pair); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &pair, nil
}

// do 执行 API 请求
func (c *Client) do(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
