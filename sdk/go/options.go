package dimsdk

// ClientOptions SDK 客户端配置
type ClientOptions struct {
	BaseURL string // IM 服务地址，如 http://localhost:8080
	APIKey  string // API Key，与 backend/.env 中 JWT_API_KEY 一致
}
