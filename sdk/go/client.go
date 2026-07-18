package dimsdk

import (
	"bytes"
	"context"
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
func (c *Client) GetSession(ctx context.Context, id string) (*TokenPair, error) {
	respBody, err := c.do(ctx, "POST", "/api/v1/auth/session", map[string]string{
		"id": id,
	})
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code  int       `json:"code"`
		Data  TokenPair `json:"data"`
		Error string    `json:"error"`
	}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &resp.Data, nil
}

// do 执行 API 请求
func (c *Client) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	return c.doWithToken(ctx, method, path, body, "")
}

func (c *Client) doWithToken(ctx context.Context, method, path string, body interface{}, accessToken string) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if accessToken == "" {
		req.Header.Set("X-API-Key", c.apiKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

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
