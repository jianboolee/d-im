你的目录结构中确实遗漏了 auth 部分。让我帮你补充完整的 IM 认证授权架构，特别针对多设备登录的场景。

## 一、补充 auth 到目录结构

```
d-im/backend/
├── cmd/
│   └── auth/                               # 认证服务入口
│       └── main.go
│
├── internal/
│   └── auth/                               # 认证服务业务逻辑
│       ├── handler/
│       │   ├── auth_handler.go             # 登录/注册/刷新token
│       │   ├── device_handler.go           # 设备管理
│       │   └── middleware/
│       │       ├── jwt_middleware.go        # JWT验证中间件
│       │       ├── device_middleware.go     # 设备校验中间件
│       │       └── permission_middleware.go # 权限校验中间件
│       │
│       ├── service/
│       │   ├── auth_service.go             # 认证核心逻辑
│       │   ├── token_service.go            # Token生成/刷新/吊销
│       │   ├── device_service.go           # 设备管理服务
│       │   ├── session_service.go          # 会话管理
│       │   └── kickout_service.go          # 设备踢出逻辑
│       │
│       ├── repository/
│       │   ├── user_repo.go                # 用户数据访问
│       │   ├── device_repo.go              # 设备数据访问
│       │   ├── session_repo.go             # 会话数据访问
│       │   └── refresh_token_repo.go       # 刷新令牌数据访问
│       │
│       └── strategy/
│           ├── auth_strategy.go            # 认证策略接口
│           ├── password_strategy.go        # 密码登录
│           ├── sms_strategy.go             # 短信验证码登录
│           ├── oauth_strategy.go           # 第三方登录
│           └── qrcode_strategy.go          # 扫码登录
│
├── pkg/
│   └── auth/                               # 认证公共库
│       ├── token.go                        # Token工具
│       ├── claims.go                       # JWT Claims定义
│       ├── device.go                       # 设备信息结构
│       └── errors.go                       # 认证错误定义
```

## 二、IM 系统的 Auth 架构设计

### 1. 核心认证模型

```go
// pkg/model/user_auth.go
package model

import (
    "time"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// UserAuth 用户认证信息
type UserAuth struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    UID          string             `bson:"uid"`                    // 用户ID
    PasswordHash string             `bson:"password_hash"`          // 密码哈希
    Salt         string             `bson:"salt"`                   // 密码盐值
    Phone        string             `bson:"phone,omitempty"`        // 手机号
    Email        string             `bson:"email,omitempty"`        // 邮箱
    WechatOpenID string             `bson:"wechat_openid,omitempty"`// 微信OpenID
    
    // 安全相关
    FailedAttempts   int        `bson:"failed_attempts"`     // 登录失败次数
    LockedUntil      *time.Time `bson:"locked_until,omitempty"` // 锁定到何时
    LastLoginAt      time.Time  `bson:"last_login_at"`       // 最后登录时间
    LastLoginIP      string     `bson:"last_login_ip"`       // 最后登录IP
    
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}

// Device 设备信息
type Device struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    DeviceID    string             `bson:"device_id"`     // 设备唯一ID
    UID         string             `bson:"uid"`           // 用户ID
    Platform    Platform           `bson:"platform"`      // 平台类型
    DeviceName  string             `bson:"device_name"`   // 设备名称
    DeviceModel string             `bson:"device_model"`  // 设备型号
    OSVersion   string             `bson:"os_version"`    // 操作系统版本
    AppVersion  string             `bson:"app_version"`   // App版本
    PushToken   string             `bson:"push_token,omitempty"` // 推送Token
    
    // 设备状态
    IsOnline    bool       `bson:"is_online"`      // 是否在线
    LastOnlineAt time.Time `bson:"last_online_at"` // 最后在线时间
    IPAddress   string     `bson:"ip_address"`     // IP地址
    Location    string     `bson:"location"`       // 位置信息
    
    CreatedAt time.Time `bson:"created_at"`
    UpdatedAt time.Time `bson:"updated_at"`
}

// Platform 平台类型
type Platform string

const (
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWeb     Platform = "web"
    PlatformDesktop Platform = "desktop"
    PlatformIPad    Platform = "ipad"
)

// DeviceLimit 设备限制配置
type DeviceLimit struct {
    Platform Platform `bson:"platform"`
    MaxCount int      `bson:"max_count"` // 最大设备数
}

// 默认设备限制
var DefaultDeviceLimits = []DeviceLimit{
    {Platform: PlatformIOS, MaxCount: 3},
    {Platform: PlatformAndroid, MaxCount: 3},
    {Platform: PlatformWeb, MaxCount: 2},
    {Platform: PlatformDesktop, MaxCount: 2},
    {Platform: PlatformIPad, MaxCount: 2},
}
```

### 2. Token 与 Session 设计

```go
// pkg/model/session.go
package model

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

// TokenPair 令牌对
type TokenPair struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"`    // AccessToken过期时间(秒)
    TokenType    string `json:"token_type"`    // Bearer
}

// Claims JWT声明
type Claims struct {
    jwt.RegisteredClaims
    
    // 用户信息
    UID      string `json:"uid"`
    Platform Platform `json:"platform"`
    
    // 设备信息
    DeviceID string `json:"device_id"`
    
    // Token版本号(用于强制下线)
    TokenVersion int64 `json:"token_version"`
    
    // 自定义字段
    Role     string `json:"role,omitempty"`
}

// RefreshToken 刷新令牌存储
type RefreshToken struct {
    ID          primitive.ObjectID `bson:"_id,omitempty"`
    Token       string             `bson:"token"`         // 刷新令牌
    UID         string             `bson:"uid"`           // 用户ID
    DeviceID    string             `bson:"device_id"`     // 设备ID
    Platform    Platform           `bson:"platform"`      // 平台
    
    // 令牌状态
    IsRevoked   bool               `bson:"is_revoked"`    // 是否已吊销
    RevokedAt   *time.Time         `bson:"revoked_at,omitempty"`  // 吊销时间
    RevokeReason string            `bson:"revoke_reason,omitempty"` // 吊销原因
    
    // 安全信息
    IPAddress   string             `bson:"ip_address"`    // 签发IP
    UserAgent   string             `bson:"user_agent"`    // UserAgent
    
    // 过期时间
    ExpiresAt   time.Time          `bson:"expires_at"`    // 过期时间
    CreatedAt   time.Time          `bson:"created_at"`
    
    // 替换链(防止刷新令牌重放攻击)
    ReplacedBy  string             `bson:"replaced_by,omitempty"` // 被哪个新token替换
}

// Session 用户会话
type Session struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    UID           string             `bson:"uid"`
    DeviceID      string             `bson:"device_id"`
    Platform      Platform           `bson:"platform"`
    
    // 会话状态
    IsOnline      bool               `bson:"is_online"`
    LastActiveAt  time.Time          `bson:"last_active_at"`  // 最后活跃时间
    LastSeqID     int64              `bson:"last_seq_id"`     // 最后消息序列号
    
    // WebSocket连接信息(用于连接服务)
    ConnNodeID    string             `bson:"conn_node_id"`    // 连接节点ID
    ConnID        string             `bson:"conn_id"`         // 连接ID
    
    // 安全信息
    IPAddress     string             `bson:"ip_address"`
    UserAgent     string             `bson:"user_agent"`
    
    CreatedAt     time.Time          `bson:"created_at"`
    UpdatedAt     time.Time          `bson:"updated_at"`
}
```

### 3. 认证服务实现

```go
// internal/auth/service/auth_service.go
package service

import (
    "context"
    "fmt"
    "time"
    
    "d-im/pkg/auth"
    "d-im/pkg/model"
    "d-im/pkg/errcode"
)

// AuthService 认证服务
type AuthService struct {
    userRepo       *repository.UserRepo
    deviceRepo     *repository.DeviceRepo
    sessionRepo    *repository.SessionRepo
    refreshTokenRepo *repository.RefreshTokenRepo
    tokenService   *TokenService
    cache          *redis.Client
}

// LoginRequest 登录请求
type LoginRequest struct {
    Account     string          `json:"account"`     // 手机号/邮箱/用户名
    Password    string          `json:"password"`
    Platform    model.Platform  `json:"platform"`
    DeviceInfo  *DeviceInfo     `json:"device_info"`
}

// DeviceInfo 设备信息
type DeviceInfo struct {
    DeviceID    string `json:"device_id"`
    DeviceName  string `json:"device_name"`
    DeviceModel string `json:"device_model"`
    OSVersion   string `json:"os_version"`
    AppVersion  string `json:"app_version"`
    PushToken   string `json:"push_token"`
    IPAddress   string `json:"ip_address"`
}

// LoginResponse 登录响应
type LoginResponse struct {
    TokenPair    *model.TokenPair `json:"token_pair"`
    UserInfo     *UserInfo        `json:"user_info"`
    DeviceID     string           `json:"device_id"`
    
    // 多设备管理信息
    OtherDevices []*DeviceInfo    `json:"other_devices"`     // 其他已登录设备
    DeviceLimit  *DeviceLimitInfo `json:"device_limit,omitempty"` // 设备限制信息
}

type DeviceLimitInfo struct {
    Current  int `json:"current"`   // 当前设备数
    Max      int `json:"max"`       // 最大设备数
    Exceeded bool `json:"exceeded"` // 是否超限
}

// Login 登录（支持多设备）
func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    // 1. 验证账号密码
    userAuth, err := s.userRepo.FindByAccount(ctx, req.Account)
    if err != nil {
        return nil, errcode.ErrUserNotFound
    }
    
    if !s.verifyPassword(req.Password, userAuth.PasswordHash, userAuth.Salt) {
        // 记录失败次数
        s.handleLoginFailed(ctx, userAuth)
        return nil, errcode.ErrPasswordIncorrect
    }
    
    // 检查账户是否被锁定
    if userAuth.LockedUntil != nil && time.Now().Before(*userAuth.LockedUntil) {
        return nil, errcode.ErrAccountLocked
    }
    
    // 2. 检查设备限制
    deviceCount, err := s.deviceRepo.CountByUIDAndPlatform(ctx, userAuth.UID, req.Platform)
    if err != nil {
        return nil, err
    }
    
    deviceLimit := s.getDeviceLimit(req.Platform)
    
    // 3. 处理设备注册/更新
    device, isNew, err := s.registerOrUpdateDevice(ctx, userAuth.UID, req)
    if err != nil {
        return nil, err
    }
    
    // 4. 检查是否超出设备限制
    limitExceeded := deviceCount >= deviceLimit.MaxCount && isNew
    
    if limitExceeded {
        // 返回设备超限信息，让客户端决定是否踢掉旧设备
        otherDevices, _ := s.deviceRepo.FindByUIDAndPlatform(ctx, userAuth.UID, req.Platform)
        return &LoginResponse{
            OtherDevices: convertToDeviceInfo(otherDevices),
            DeviceLimit: &DeviceLimitInfo{
                Current:  deviceCount,
                Max:      deviceLimit.MaxCount,
                Exceeded: true,
            },
        }, errcode.ErrDeviceLimitExceeded
    }
    
    // 5. 生成TokenPair
    tokenPair, err := s.tokenService.GenerateTokenPair(ctx, &auth.TokenPayload{
        UID:          userAuth.UID,
        DeviceID:     device.DeviceID,
        Platform:     req.Platform,
        TokenVersion: s.getTokenVersion(ctx, userAuth.UID),
        IPAddress:    req.DeviceInfo.IPAddress,
        UserAgent:    req.DeviceInfo.DeviceModel,
    })
    if err != nil {
        return nil, err
    }
    
    // 6. 创建会话
    session := &model.Session{
        UID:          userAuth.UID,
        DeviceID:     device.DeviceID,
        Platform:     req.Platform,
        IsOnline:     true,
        LastActiveAt: time.Now(),
        IPAddress:    req.DeviceInfo.IPAddress,
        UserAgent:    req.DeviceInfo.DeviceModel,
    }
    
    if err := s.sessionRepo.CreateOrUpdate(ctx, session); err != nil {
        return nil, err
    }
    
    // 7. 更新设备在线状态
    s.deviceRepo.UpdateOnlineStatus(ctx, device.DeviceID, true)
    
    // 8. 更新登录记录
    s.userRepo.UpdateLoginInfo(ctx, userAuth.UID, req.DeviceInfo.IPAddress)
    
    // 9. 通知其他设备有新登录
    s.notifyNewLogin(ctx, userAuth.UID, device)
    
    // 10. 获取其他已登录设备列表
    otherDevices, _ := s.deviceRepo.FindOtherOnlineDevices(ctx, userAuth.UID, device.DeviceID)
    
    return &LoginResponse{
        TokenPair:    tokenPair,
        UserInfo:     s.getUserInfo(ctx, userAuth.UID),
        DeviceID:     device.DeviceID,
        OtherDevices: convertToDeviceInfo(otherDevices),
    }, nil
}

// ForceLogin 强制登录（踢掉最旧的设备）
func (s *AuthService) ForceLogin(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    // 获取当前平台设备列表
    devices, err := s.deviceRepo.FindByUIDAndPlatform(ctx, req.DeviceInfo.DeviceID, req.Platform)
    if err != nil {
        return nil, err
    }
    
    deviceLimit := s.getDeviceLimit(req.Platform)
    
    // 如果超出限制，踢掉最旧的设备
    if len(devices) >= deviceLimit.MaxCount {
        // 按最后活跃时间排序，踢掉最旧的
        oldestDevice := devices[0] // 假设已排序
        if err := s.kickoutDevice(ctx, oldestDevice.UID, oldestDevice.DeviceID, "设备数量超限"); err != nil {
            return nil, err
        }
    }
    
    // 重新执行登录
    return s.Login(ctx, req)
}
```

### 4. Token 服务实现

```go
// internal/auth/service/token_service.go
package service

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

// TokenService Token服务
type TokenService struct {
    config         *TokenConfig
    refreshTokenRepo *repository.RefreshTokenRepo
    cache          *redis.Client
}

// TokenConfig Token配置
type TokenConfig struct {
    // AccessToken配置
    AccessTokenSecret   string        `yaml:"access_token_secret"`
    AccessTokenExpire   time.Duration `yaml:"access_token_expire"`    // 2小时
    
    // RefreshToken配置
    RefreshTokenSecret  string        `yaml:"refresh_token_secret"`
    RefreshTokenExpire  time.Duration `yaml:"refresh_token_expire"`   // 30天
    
    // 安全配置
    Issuer             string        `yaml:"issuer"`
    RefreshTokenLength int           `yaml:"refresh_token_length"`   // 32
}

// TokenPayload Token负载
type TokenPayload struct {
    UID          string
    DeviceID     string
    Platform     model.Platform
    TokenVersion int64
    IPAddress    string
    UserAgent    string
}

// GenerateTokenPair 生成Token对
func (s *TokenService) GenerateTokenPair(ctx context.Context, payload *TokenPayload) (*model.TokenPair, error) {
    now := time.Now()
    
    // 1. 生成AccessToken
    accessClaims := &model.Claims{
        RegisteredClaims: jwt.RegisteredClaims{
            Issuer:    s.config.Issuer,
            Subject:   payload.UID,
            IssuedAt:  jwt.NewNumericDate(now),
            ExpiresAt: jwt.NewNumericDate(now.Add(s.config.AccessTokenExpire)),
            ID:        s.generateTokenID(),
        },
        UID:          payload.UID,
        Platform:     payload.Platform,
        DeviceID:     payload.DeviceID,
        TokenVersion: payload.TokenVersion,
    }
    
    accessToken, err := s.generateJWT(accessClaims, s.config.AccessTokenSecret)
    if err != nil {
        return nil, err
    }
    
    // 2. 生成RefreshToken
    refreshToken := s.generateRefreshToken()
    
    // 3. 存储RefreshToken
    refreshTokenDoc := &model.RefreshToken{
        Token:      refreshToken,
        UID:        payload.UID,
        DeviceID:   payload.DeviceID,
        Platform:   payload.Platform,
        IPAddress:  payload.IPAddress,
        UserAgent:  payload.UserAgent,
        ExpiresAt:  now.Add(s.config.RefreshTokenExpire),
        CreatedAt:  now,
    }
    
    if err := s.refreshTokenRepo.Create(ctx, refreshTokenDoc); err != nil {
        return nil, err
    }
    
    // 4. 缓存AccessToken(用于快速验证和吊销)
    s.cache.Set(ctx, 
        fmt.Sprintf("access_token:%s", payload.UID),
        accessToken,
        s.config.AccessTokenExpire,
    )
    
    return &model.TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
        ExpiresIn:    int64(s.config.AccessTokenExpire.Seconds()),
        TokenType:    "Bearer",
    }, nil
}

// RefreshAccessToken 刷新AccessToken
func (s *TokenService) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (*model.TokenPair, error) {
    // 1. 查找RefreshToken
    oldToken, err := s.refreshTokenRepo.FindByToken(ctx, refreshTokenStr)
    if err != nil {
        return nil, errcode.ErrInvalidRefreshToken
    }
    
    // 2. 检查是否已吊销或过期
    if oldToken.IsRevoked || time.Now().After(oldToken.ExpiresAt) {
        // 如果token被吊销，可能是重放攻击，吊销该用户所有token
        if oldToken.IsRevoked {
            s.revokeAllUserTokens(ctx, oldToken.UID, "token_reuse_detected")
        }
        return nil, errcode.ErrTokenRevoked
    }
    
    // 3. 吊销旧RefreshToken(防止重放攻击)
    if err := s.refreshTokenRepo.Revoke(ctx, oldToken.Token, ""); err != nil {
        return nil, err
    }
    
    // 4. 生成新的TokenPair(使用相同的设备信息)
    payload := &TokenPayload{
        UID:          oldToken.UID,
        DeviceID:     oldToken.DeviceID,
        Platform:     oldToken.Platform,
        TokenVersion: s.getUserTokenVersion(ctx, oldToken.UID),
        IPAddress:    oldToken.IPAddress,
        UserAgent:    oldToken.UserAgent,
    }
    
    newPair, err := s.GenerateTokenPair(ctx, payload)
    if err != nil {
        return nil, err
    }
    
    // 5. 关联新旧Token(形成替换链)
    s.refreshTokenRepo.UpdateReplacedBy(ctx, oldToken.Token, newPair.RefreshToken)
    
    return newPair, nil
}

// RevokeToken 吊销Token(用户主动登出)
func (s *TokenService) RevokeToken(ctx context.Context, uid, deviceID string) error {
    // 1. 吊销RefreshToken
    if err := s.refreshTokenRepo.RevokeByDevice(ctx, uid, deviceID, "user_logout"); err != nil {
        return err
    }
    
    // 2. 从缓存中删除AccessToken
    s.cache.Del(ctx, fmt.Sprintf("access_token:%s", uid))
    
    // 3. 更新Token版本号(使所有旧的AccessToken失效)
    s.incrementTokenVersion(ctx, uid)
    
    return nil
}

// RevokeAllTokens 吊销用户所有Token(强制所有设备下线)
func (s *TokenService) RevokeAllTokens(ctx context.Context, uid, reason string) error {
    // 1. 吊销所有RefreshToken
    if err := s.refreshTokenRepo.RevokeAllByUser(ctx, uid, reason); err != nil {
        return err
    }
    
    // 2. 清除所有AccessToken缓存
    s.cache.Del(ctx, fmt.Sprintf("access_token:%s", uid))
    
    // 3. 更新Token版本号
    s.incrementTokenVersion(ctx, uid)
    
    return nil
}

// 生成RefreshToken
func (s *TokenService) generateRefreshToken() string {
    b := make([]byte, s.config.RefreshTokenLength)
    rand.Read(b)
    return hex.EncodeToString(b)
}

// 生成JWT
func (s *TokenService) generateJWT(claims *model.Claims, secret string) (string, error) {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// Token版本号管理(存储于Redis)
func (s *TokenService) incrementTokenVersion(ctx context.Context, uid string) error {
    key := fmt.Sprintf("token_version:%s", uid)
    return s.cache.Incr(ctx, key).Err()
}

func (s *TokenService) getUserTokenVersion(ctx context.Context, uid string) int64 {
    key := fmt.Sprintf("token_version:%s", uid)
    val, _ := s.cache.Get(ctx, key).Int64()
    return val
}
```

### 5. 设备管理和踢出逻辑

```go
// internal/auth/service/device_service.go
package service

import (
    "context"
    "fmt"
    "sort"
    "time"
)

// DeviceService 设备管理服务
type DeviceService struct {
    deviceRepo     *repository.DeviceRepo
    sessionRepo    *repository.SessionRepo
    tokenService   *TokenService
    nats           *nats.Manager // 用于通知设备下线
}

// DeviceListItem 设备列表项
type DeviceListItem struct {
    DeviceID    string          `json:"device_id"`
    DeviceName  string          `json:"device_name"`
    Platform    model.Platform  `json:"platform"`
    IsOnline    bool            `json:"is_online"`
    IsCurrent   bool            `json:"is_current"`   // 是否当前设备
    LastActiveAt time.Time      `json:"last_active_at"`
    IPAddress   string          `json:"ip_address"`
    Location    string          `json:"location"`
}

// GetDevices 获取用户所有设备
func (s *DeviceService) GetDevices(ctx context.Context, uid string) ([]*DeviceListItem, error) {
    devices, err := s.deviceRepo.FindByUID(ctx, uid)
    if err != nil {
        return nil, err
    }
    
    items := make([]*DeviceListItem, 0, len(devices))
    for _, device := range devices {
        items = append(items, &DeviceListItem{
            DeviceID:     device.DeviceID,
            DeviceName:   device.DeviceName,
            Platform:     device.Platform,
            IsOnline:     device.IsOnline,
            LastActiveAt: device.LastOnlineAt,
            IPAddress:    device.IPAddress,
            Location:     device.Location,
        })
    }
    
    // 排序：在线优先，最近活跃优先
    sort.Slice(items, func(i, j int) bool {
        if items[i].IsOnline != items[j].IsOnline {
            return items[i].IsOnline
        }
        return items[i].LastActiveAt.After(items[j].LastActiveAt)
    })
    
    return items, nil
}

// KickoutDevice 踢出设备
func (s *DeviceService) KickoutDevice(ctx context.Context, uid, deviceID, reason string) error {
    // 1. 吊销该设备的Token
    if err := s.tokenService.RevokeToken(ctx, uid, deviceID); err != nil {
        return err
    }
    
    // 2. 更新设备状态为离线
    if err := s.deviceRepo.UpdateOnlineStatus(ctx, deviceID, false); err != nil {
        return err
    }
    
    // 3. 更新会话状态
    if err := s.sessionRepo.UpdateOnlineStatus(ctx, uid, deviceID, false); err != nil {
        return err
    }
    
    // 4. 通过NATS通知连接服务断开连接
    kickoutMsg := &KickoutMessage{
        UID:      uid,
        DeviceID: deviceID,
        Reason:   reason,
        Time:     time.Now(),
    }
    
    subject := fmt.Sprintf("im.auth.kickout.%s", uid)
    if err := s.nats.PublishMessage(ctx, subject, kickoutMsg); err != nil {
        return err
    }
    
    // 5. 发送推送通知给被踢出的设备
    s.sendKickoutNotification(ctx, uid, deviceID, reason)
    
    return nil
}

// KickoutMessage 踢出消息
type KickoutMessage struct {
    UID      string    `json:"uid"`
    DeviceID string    `json:"device_id"`
    Reason   string    `json:"reason"`
    Time     time.Time `json:"time"`
}
```

### 6. 认证中间件

```go
// internal/auth/handler/middleware/jwt_middleware.go
package middleware

import (
    "context"
    "net/http"
    "strings"
    
    "github.com/golang-jwt/jwt/v5"
)

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware(tokenService *TokenService) gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 提取Token
        tokenStr := extractToken(c)
        if tokenStr == "" {
            c.JSON(http.StatusUnauthorized, errcode.ErrTokenMissing)
            c.Abort()
            return
        }
        
        // 2. 解析并验证Token
        claims, err := tokenService.ValidateAccessToken(c.Request.Context(), tokenStr)
        if err != nil {
            c.JSON(http.StatusUnauthorized, err)
            c.Abort()
            return
        }
        
        // 3. 检查Token版本号
        currentVersion := tokenService.GetUserTokenVersion(c.Request.Context(), claims.UID)
        if claims.TokenVersion < currentVersion {
            c.JSON(http.StatusUnauthorized, errcode.ErrTokenExpired)
            c.Abort()
            return
        }
        
        // 4. 将用户信息注入上下文
        ctx := context.WithValue(c.Request.Context(), "uid", claims.UID)
        ctx = context.WithValue(ctx, "device_id", claims.DeviceID)
        ctx = context.WithValue(ctx, "platform", claims.Platform)
        ctx = context.WithValue(ctx, "token_version", claims.TokenVersion)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
    }
}

// MultiDeviceAuth 多设备认证中间件
func MultiDeviceAuth(deviceService *DeviceService) gin.HandlerFunc {
    return func(c *gin.Context) {
        uid := c.GetString("uid")
        deviceID := c.GetString("device_id")
        
        // 检查设备是否有效（未被踢出）
        device, err := deviceService.GetDevice(c.Request.Context(), uid, deviceID)
        if err != nil || !device.IsOnline {
            c.JSON(http.StatusUnauthorized, errcode.ErrDeviceKickedOut)
            c.Abort()
            return
        }
        
        // 更新设备最后活跃时间
        deviceService.UpdateLastActive(c.Request.Context(), deviceID)
        
        c.Next()
    }
}
```

### 7. 连接服务处理踢出逻辑

```go
// internal/connector/ws/handler.go

// 在连接服务中订阅踢出消息
func (s *WSServer) SubscribeKickoutEvents(ctx context.Context) error {
    // 订阅踢出事件
    subject := fmt.Sprintf("im.auth.kickout.%s", "*") // 监听所有用户的踢出事件
    
    _, err := s.nats.Subscribe(subject, func(msg *nats.Msg) {
        var kickoutMsg KickoutMessage
        json.Unmarshal(msg.Data, &kickoutMsg)
        
        // 查找本地连接
        client := s.channelManager.GetClient(kickoutMsg.UID, kickoutMsg.DeviceID)
        if client != nil {
            // 发送踢出通知给客户端
            notification := map[string]interface{}{
                "type":   "kickout",
                "reason": kickoutMsg.Reason,
                "time":   kickoutMsg.Time,
            }
            
            client.SendJSON(notification)
            
            // 延迟关闭连接，让客户端有时间处理通知
            time.AfterFunc(3*time.Second, func() {
                client.Close()
            })
        }
    })
    
    return err
}
```

### 8. Redis 缓存设计

```go
// pkg/cache/redis/auth_cache.go

// 认证相关的Redis键设计
const (
    // Token相关
    KeyAccessToken     = "access_token:%s"      // access_token:uid
    KeyRefreshToken    = "refresh_token:%s"     // refresh_token:token
    
    // Token版本号(用于强制下线)
    KeyTokenVersion    = "token_version:%s"     // token_version:uid
    
    // 设备信息
    KeyDeviceInfo      = "device:%s:%s"         // device:uid:device_id
    
    // 在线状态
    KeyUserOnline      = "user_online:%s"       // user_online:uid (set of device_ids)
    
    // 登录失败限制
    KeyLoginFailed     = "login_failed:%s"      // login_failed:uid
    KeyLoginLocked     = "login_locked:%s"      // login_locked:uid
    
    // 验证码
    KeySmsCode         = "sms_code:%s"          // sms_code:phone
)
```

## 三、多设备登录的完整流程图

```
用户登录流程:
┌─────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│Client│     │Auth服务   │     │MongoDB   │     │Redis     │
└──┬──┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
   │  登录请求   │                │                │
   │───────────>│                │                │
   │            │ 查询用户        │                │
   │            │───────────────>│                │
   │            │<───────────────│                │
   │            │ 验证密码        │                │
   │            │ 检查设备限制     │                │
   │            │───────────────>│                │
   │            │<───────────────│                │
   │            │                │                │
   │            │ 生成TokenPair  │                │
   │            │───────────────────────────────>│
   │            │ 存储RefreshToken│               │
   │            │───────────────>│                │
   │            │ 创建Session    │                │
   │            │───────────────>│                │
   │ 返回Token  │                │                │
   │<───────────│                │                │
   │            │                │                │

设备被踢出流程:
┌─────┐     ┌──────────┐     ┌──────────┐     ┌──────────┐
│Client│     │Auth服务   │     │Connector │     │NATS      │
└──┬──┘     └────┬─────┘     └────┬─────┘     └────┬─────┘
   │  踢出设备   │                │                │
   │───────────>│                │                │
   │            │ 吊销Token      │                │
   │            │ 更新设备状态     │                │
   │            │                │                │
   │            │ 发布踢出事件     │                │
   │            │────────────────────────────────>│
   │            │                │                │
   │            │                │ 订阅踢出事件     │
   │            │                │<───────────────│
   │            │                │                │
   │            │      │ 查找连接并断开    │
   │            │      │<───────┘        │
   │            │      │ 发送踢出通知     │
   │            │      │────────>        │
   │<───────────────────── 连接断开通知    │
   │            │                │                │
```

这个 auth 架构设计的核心特点：

1. **多设备支持**：每个设备独立Token，互不影响
2. **设备限制**：按平台限制设备数量，支持踢出旧设备
3. **Token刷新**：使用RefreshToken机制，支持安全刷新
4. **强制下线**：通过Token版本号实现全设备强制下线
5. **实时踢出**：通过NATS实时通知连接服务断开设备
6. **防重放攻击**：RefreshToken使用替换链机制
7. **安全加固**：支持登录失败锁定、异常登录检测