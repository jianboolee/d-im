package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTHeader JWT头部
type JWTHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

// JWTClaims JWT载荷
type JWTClaims struct {
	Sub string `json:"sub"`           // uid
	Iat int64  `json:"iat"`           // 签发时间
	Exp int64  `json:"exp"`           // 过期时间
	Jti string `json:"jti,omitempty"` // 唯一ID（防重放）
}

// JWTManager JWT管理器
type JWTManager struct {
	secret []byte
	expire time.Duration
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secret string, expire time.Duration) *JWTManager {
	if expire <= 0 {
		expire = 24 * time.Hour
	}
	return &JWTManager{
		secret: []byte(secret),
		expire: expire,
	}
}

// Issue 签发票据
func (m *JWTManager) Issue(uid string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		Sub: uid,
		Iat: now.Unix(),
		Exp: now.Add(m.expire).Unix(),
	}

	return m.sign(claims)
}

// Verify 验证票据，成功返回 uid
func (m *JWTManager) Verify(token string) (string, error) {
	claims, err := m.parse(token)
	if err != nil {
		return "", err
	}

	now := time.Now().Unix()
	if claims.Exp > 0 && now > claims.Exp {
		return "", fmt.Errorf("token expired")
	}

	if claims.Sub == "" {
		return "", fmt.Errorf("missing sub claim")
	}

	return claims.Sub, nil
}

// sign 签名生成JWT
func (m *JWTManager) sign(claims JWTClaims) (string, error) {
	header := JWTHeader{Alg: "HS256", Typ: "JWT"}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64URLEncode(headerJSON)
	claimsB64 := base64URLEncode(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	signature := m.hmacSign(signingInput)

	return signingInput + "." + base64URLEncode(signature), nil
}

// parse 解析JWT
func (m *JWTManager) parse(token string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// 验证签名
	signingInput := parts[0] + "." + parts[1]
	expectedSig := base64URLEncode(m.hmacSign(signingInput))

	actualSig := parts[2]
	if !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
		return nil, fmt.Errorf("invalid signature")
	}

	// 解码claims
	claimsJSON, err := base64URLDecode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("unmarshal claims: %w", err)
	}

	return &claims, nil
}

// hmacSign HMAC-SHA256签名
func (m *JWTManager) hmacSign(input string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

func base64URLDecode(s string) ([]byte, error) {
	// 补齐padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}
