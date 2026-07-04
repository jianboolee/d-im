package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	TokenTypeTicket  = "ticket"
)

// JWTManager JWT管理器（基于 golang-jwt/jwt v5）
type JWTManager struct {
	secret        []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
	ticketExpire  time.Duration
	apiKey        string
}

// NewJWTManager 创建JWT管理器
func NewJWTManager(secret string, accessExpire, refreshExpire, ticketExpire time.Duration, apiKey string) *JWTManager {
	return &JWTManager{
		secret:        []byte(secret),
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
		ticketExpire:  ticketExpire,
		apiKey:        apiKey,
	}
}

// VerifyAPIKey 验证业务系统 API Key
func (m *JWTManager) VerifyAPIKey(key string) bool {
	return key != "" && key == m.apiKey
}

// IssueAccessToken 签发访问令牌
func (m *JWTManager) IssueAccessToken(uid, deviceID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       uid,
		"device_id": deviceID,
		"type":      TokenTypeAccess,
		"iat":       now.Unix(),
		"exp":       now.Add(m.accessExpire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// IssueRefreshToken 签发刷新令牌
func (m *JWTManager) IssueRefreshToken(uid, deviceID string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":       uid,
		"device_id": deviceID,
		"type":      TokenTypeRefresh,
		"iat":       now.Unix(),
		"exp":       now.Add(m.refreshExpire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// IssueTicket 签发一次性 ticket（业务系统换取）
func (m *JWTManager) IssueTicket(uid string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":  uid,
		"type": TokenTypeTicket,
		"iat":  now.Unix(),
		"exp":  now.Add(m.ticketExpire).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Verify 验证 token，返回 claims map
func (m *JWTManager) Verify(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return "", fmt.Errorf("missing sub claim")
	}

	return sub, nil
}

// VerifyAs 验证 token 并检查类型
func (m *JWTManager) VerifyAs(tokenStr string, expectedType string) (string, error) {
	uid, err := m.Verify(tokenStr)
	if err != nil {
		return "", err
	}

	// 额外解析获取 type claim
	token, _ := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if tokenType, exists := claims["type"]; exists {
			if tokenType != expectedType {
				return "", fmt.Errorf("unexpected token type: %v", tokenType)
			}
		}
	}

	return uid, nil
}
