好的，我来设计一个完整的 Push 服务架构，先用 Mock 实现，后续可以无缝切换到极光推送。

## 一、Push 服务的完整架构

### 1. 目录结构

```
im-system/
├── internal/
│   └── push/                               # 推送服务
│       ├── handler/
│       │   └── push_handler.go             # HTTP接口（手动测试用）
│       ├── service/
│       │   ├── push_service.go             # 推送核心逻辑
│       │   ├── dedup_service.go            # 去重逻辑
│       │   └── throttle_service.go         # 限流逻辑
│       ├── provider/                        # 推送提供商
│       │   ├── provider.go                 # 推送接口定义
│       │   ├── mock_provider.go            # Mock实现
│       │   ├── jpush_provider.go           # 极光推送实现（后续）
│       │   └── provider_factory.go         # 工厂方法
│       ├── consumer/
│       │   └── offline_push_consumer.go    # 离线推送消费者
│       └── model/
│           ├── push_message.go             # 推送消息模型
│           └── push_record.go              # 推送记录
│
├── pkg/
│   └── push/                               # 推送公共库
│       ├── types.go                        # 推送相关类型
│       └── constants.go                    # 推送常量
```

### 2. 核心接口定义

```go
// internal/push/provider/provider.go
package provider

import (
    "context"
    "im-system/pkg/types"
)

// PushProvider 推送提供商接口
// 这是关键：定义统一接口，Mock和极光推送都实现这个接口
type PushProvider interface {
    // Name 提供商名称
    Name() string
    
    // Push 单条推送
    Push(ctx context.Context, req *PushRequest) (*PushResponse, error)
    
    // BatchPush 批量推送
    BatchPush(ctx context.Context, reqs []*PushRequest) (*BatchPushResponse, error)
    
    // PushByAlias 按别名推送（如果提供商支持）
    PushByAlias(ctx context.Context, alias string, req *PushRequest) (*PushResponse, error)
    
    // IsHealthy 健康检查
    IsHealthy(ctx context.Context) bool
}

// PushRequest 推送请求
type PushRequest struct {
    // 推送目标
    Platform     types.Platform `json:"platform"`      // ios/android
    PushToken    string         `json:"push_token"`    // 设备推送Token
    UserID       string         `json:"user_id"`       // 用户ID
    
    // 推送内容
    Title        string         `json:"title"`         // 推送标题
    Body         string         `json:"body"`          // 推送内容
    Sound        string         `json:"sound"`         // 提示音
    
    // 角标
    Badge        int            `json:"badge"`         // 角标数字
    
    // 扩展数据（点击推送后跳转到指定会话）
    Extra        map[string]interface{} `json:"extra"`
    
    // 消息信息
    MsgID        string         `json:"msg_id"`        // 消息ID
    ChatID       string         `json:"chat_id"`       // 会话ID
    MsgType      types.MessageType `json:"msg_type"`   // 消息类型
    
    // 推送选项
    Priority     PushPriority   `json:"priority"`       // 推送优先级
    TTL          int64          `json:"ttl"`            // 存活时间（秒）
}

// PushResponse 推送响应
type PushResponse struct {
    Success   bool   `json:"success"`
    MsgID     string `json:"msg_id"`       // 第三方返回的消息ID
    ErrorCode string `json:"error_code"`
    ErrorMsg  string `json:"error_msg"`
}

// BatchPushResponse 批量推送响应
type BatchPushResponse struct {
    Total      int              `json:"total"`
    SuccessNum int              `json:"success_num"`
    FailedNum  int              `json:"failed_num"`
    Results    []*PushResponse  `json:"results"`
}

// PushPriority 推送优先级
type PushPriority string

const (
    PushPriorityHigh   PushPriority = "high"
    PushPriorityNormal PushPriority = "normal"
)

// PushContentBuilder 推送内容构建器
type PushContentBuilder struct{}

// BuildPushContent 根据消息类型构建推送内容
func (b *PushContentBuilder) BuildPushContent(msgType types.MessageType, content interface{}, senderName string) (title string, body string) {
    switch msgType {
    case types.MessageTypeText:
        textContent, ok := content.(types.TextContent)
        if ok {
            title = senderName
            body = textContent.Text
        }
    case types.MessageTypeImage:
        title = senderName
        body = "[图片]"
    case types.MessageTypeVideo:
        title = senderName
        body = "[视频]"
    case types.MessageTypeVoice:
        title = senderName
        body = "[语音]"
    case types.MessageTypeFile:
        title = senderName
        body = "[文件]"
    case types.MessageTypeLocation:
        title = senderName
        body = "[位置]"
    case types.MessageTypeCard:
        title = senderName
        body = "[卡片]"
    case types.MessageTypeLink:
        title = senderName
        body = "[链接]"
    case types.MessageTypeTemplate:
        title = senderName
        body = "[通知]"
    default:
        title = senderName
        body = "新消息"
    }
    
    // 限制推送内容长度
    if len(body) > 100 {
        body = body[:100] + "..."
    }
    
    return title, body
}
```

### 3. Mock Provider 实现

```go
// internal/push/provider/mock_provider.go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "sync"
    "time"
)

// MockPushProvider Mock推送提供商
// 用于开发测试，后续替换为极光推送
type MockPushProvider struct {
    mu sync.RWMutex
    
    // 推送记录（模拟第三方推送服务的记录）
    pushRecords []*PushRecord
    
    // 统计信息
    stats MockStats
    
    // 存储路径（将推送记录写入文件，方便查看）
    logDir string
    logFile *os.File
    
    // 模拟配置
    config MockConfig
}

// PushRecord 推送记录
type PushRecord struct {
    ID         string      `json:"id"`
    PushRequest *PushRequest `json:"push_request"`
    Status     string      `json:"status"`      // success/failed
    SendTime   time.Time   `json:"send_time"`
    ErrorMsg   string      `json:"error_msg,omitempty"`
}

// MockStats Mock统计
type MockStats struct {
    TotalPushed   int64 `json:"total_pushed"`
    TotalSuccess  int64 `json:"total_success"`
    TotalFailed   int64 `json:"total_failed"`
    LastPushTime  time.Time `json:"last_push_time"`
}

// MockConfig Mock配置
type MockConfig struct {
    LogDir          string  `yaml:"log_dir"`           // 日志目录
    FailureRate     float64 `yaml:"failure_rate"`      // 模拟失败率 0.0-1.0
    LatencyMin      int64   `yaml:"latency_min"`       // 最小延迟（毫秒）
    LatencyMax      int64   `yaml:"latency_max"`       // 最大延迟（毫秒）
    StoreRecords    bool    `yaml:"store_records"`     // 是否存储推送记录
}

// NewMockPushProvider 创建Mock推送提供商
func NewMockPushProvider(config MockConfig) (*MockPushProvider, error) {
    provider := &MockPushProvider{
        pushRecords: make([]*PushRecord, 0),
        config:      config,
    }
    
    // 创建日志目录
    if config.LogDir != "" {
        if err := os.MkdirAll(config.LogDir, 0755); err != nil {
            return nil, fmt.Errorf("create log dir failed: %w", err)
        }
        
        // 打开日志文件
        logPath := filepath.Join(config.LogDir, "mock_push.log")
        file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
        if err != nil {
            return nil, fmt.Errorf("open log file failed: %w", err)
        }
        provider.logFile = file
    }
    
    log.Printf("[MockPush] Mock推送服务初始化完成，日志目录: %s", config.LogDir)
    return provider, nil
}

// Name 提供商名称
func (p *MockPushProvider) Name() string {
    return "mock"
}

// Push 单条推送
func (p *MockPushProvider) Push(ctx context.Context, req *PushRequest) (*PushResponse, error) {
    // 模拟推送延迟
    p.simulateLatency()
    
    // 模拟推送失败
    if p.shouldFail() {
        return p.mockFailure(req)
    }
    
    return p.mockSuccess(req)
}

// BatchPush 批量推送
func (p *MockPushProvider) BatchPush(ctx context.Context, reqs []*PushRequest) (*BatchPushResponse, error) {
    response := &BatchPushResponse{
        Total:   len(reqs),
        Results: make([]*PushResponse, 0, len(reqs)),
    }
    
    for _, req := range reqs {
        resp, err := p.Push(ctx, req)
        if err != nil {
            response.Results = append(response.Results, &PushResponse{
                Success:   false,
                ErrorCode: "PUSH_ERROR",
                ErrorMsg:  err.Error(),
            })
            response.FailedNum++
        } else {
            response.Results = append(response.Results, resp)
            if resp.Success {
                response.SuccessNum++
            } else {
                response.FailedNum++
            }
        }
    }
    
    return response, nil
}

// PushByAlias 按别名推送（Mock简单实现）
func (p *MockPushProvider) PushByAlias(ctx context.Context, alias string, req *PushRequest) (*PushResponse, error) {
    log.Printf("[MockPush] PushByAlias: alias=%s, title=%s, body=%s", alias, req.Title, req.Body)
    return p.Push(ctx, req)
}

// IsHealthy 健康检查
func (p *MockPushProvider) IsHealthy(ctx context.Context) bool {
    return true
}

// mockSuccess 模拟推送成功
func (p *MockPushProvider) mockSuccess(req *PushRequest) (*PushResponse, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.stats.TotalPushed++
    p.stats.TotalSuccess++
    p.stats.LastPushTime = time.Now()
    
    // 生成模拟的第三方消息ID
    mockMsgID := fmt.Sprintf("mock_push_%d_%d", time.Now().UnixNano(), p.stats.TotalPushed)
    
    // 记录推送
    record := &PushRecord{
        ID:          mockMsgID,
        PushRequest: req,
        Status:      "success",
        SendTime:    time.Now(),
    }
    
    if p.config.StoreRecords {
        p.pushRecords = append(p.pushRecords, record)
    }
    
    // 写入日志
    p.logPush(record)
    
    // 模拟角标更新
    badgeInfo := ""
    if req.Badge > 0 {
        badgeInfo = fmt.Sprintf(", badge=%d", req.Badge)
    }
    
    log.Printf("[MockPush] ✅ 推送成功 | platform=%s user=%s token=%s title=%s body=%s msg_id=%s%s",
        req.Platform, req.UserID, p.maskToken(req.PushToken), 
        req.Title, req.Body, req.MsgID, badgeInfo)
    
    return &PushResponse{
        Success: true,
        MsgID:   mockMsgID,
    }, nil
}

// mockFailure 模拟推送失败
func (p *MockPushProvider) mockFailure(req *PushRequest) (*PushResponse, error) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.stats.TotalPushed++
    p.stats.TotalFailed++
    p.stats.LastPushTime = time.Now()
    
    errorCode := "MOCK_SIMULATED_ERROR"
    errorMsg := "模拟推送失败（配置的失败率触发）"
    
    // 根据平台选择不同的模拟错误
    switch req.Platform {
    case types.PlatformIOS:
        errorCode = "APNS_MOCK_ERROR"
    case types.PlatformAndroid:
        errorCode = "FCM_MOCK_ERROR"
    }
    
    // 记录推送失败
    record := &PushRecord{
        ID:          fmt.Sprintf("mock_fail_%d", time.Now().UnixNano()),
        PushRequest: req,
        Status:      "failed",
        SendTime:    time.Now(),
        ErrorMsg:    errorMsg,
    }
    
    if p.config.StoreRecords {
        p.pushRecords = append(p.pushRecords, record)
    }
    
    p.logPush(record)
    
    log.Printf("[MockPush] ❌ 推送失败 | platform=%s user=%s token=%s error=%s",
        req.Platform, req.UserID, p.maskToken(req.PushToken), errorMsg)
    
    return &PushResponse{
        Success:    false,
        ErrorCode:  errorCode,
        ErrorMsg:   errorMsg,
    }, nil
}

// simulateLatency 模拟推送延迟
func (p *MockPushProvider) simulateLatency() {
    if p.config.LatencyMin > 0 || p.config.LatencyMax > 0 {
        latency := p.config.LatencyMin
        if p.config.LatencyMax > p.config.LatencyMin {
            latency += time.Now().UnixNano() % (p.config.LatencyMax - p.config.LatencyMin)
        }
        time.Sleep(time.Duration(latency) * time.Millisecond)
    }
}

// shouldFail 判断是否应该模拟失败
func (p *MockPushProvider) shouldFail() bool {
    if p.config.FailureRate <= 0 {
        return false
    }
    // 简单的随机失败模拟
    return time.Now().UnixNano()%100 < int64(p.config.FailureRate*100)
}

// logPush 记录推送日志
func (p *MockPushProvider) logPush(record *PushRecord) {
    if p.logFile == nil {
        return
    }
    
    data, _ := json.Marshal(record)
    p.logFile.WriteString(string(data) + "\n")
}

// GetStats 获取统计信息
func (p *MockPushProvider) GetStats() MockStats {
    p.mu.RLock()
    defer p.mu.RUnlock()
    return p.stats
}

// GetRecords 获取推送记录
func (p *MockPushProvider) GetRecords(limit int) []*PushRecord {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    records := p.pushRecords
    if limit > 0 && limit < len(records) {
        records = records[len(records)-limit:]
    }
    return records
}

// GetRecordByMsgID 根据消息ID获取推送记录
func (p *MockPushProvider) GetRecordByMsgID(msgID string) *PushRecord {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    for _, record := range p.pushRecords {
        if record.PushRequest.MsgID == msgID {
            return record
        }
    }
    return nil
}

// maskToken 脱敏Token
func (p *MockPushProvider) maskToken(token string) string {
    if len(token) <= 8 {
        return "****"
    }
    return token[:4] + "****" + token[len(token)-4:]
}

// Close 关闭Mock Provider
func (p *MockPushProvider) Close() error {
    if p.logFile != nil {
        return p.logFile.Close()
    }
    return nil
}
```

### 4. 极光推送 Provider（预留接口）

```go
// internal/push/provider/jpush_provider.go
package provider

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/jpush-go/jpush"  // 极光推送Go SDK
)

// JPushConfig 极光推送配置
type JPushConfig struct {
    AppKey       string `yaml:"app_key"`
    MasterSecret string `yaml:"master_secret"`
    ApnsProduction bool `yaml:"apns_production"` // APNs生产环境
    Timeout      int    `yaml:"timeout"`          // 超时时间（秒）
    RetryTimes   int    `yaml:"retry_times"`      // 重试次数
}

// JPushProvider 极光推送提供商
type JPushProvider struct {
    client *jpush.Client
    config JPushConfig
}

// NewJPushProvider 创建极光推送提供商
func NewJPushProvider(config JPushConfig) (*JPushProvider, error) {
    client, err := jpush.NewClient(config.AppKey, config.MasterSecret)
    if err != nil {
        return nil, fmt.Errorf("create jpush client failed: %w", err)
    }
    
    return &JPushProvider{
        client: client,
        config: config,
    }, nil
}

// Name 提供商名称
func (p *JPushProvider) Name() string {
    return "jpush"
}

// Push 单条推送
func (p *JPushProvider) Push(ctx context.Context, req *PushRequest) (*PushResponse, error) {
    // 构建极光推送请求
    pushReq := p.buildJPushRequest(req)
    
    // 调用极光推送API
    result, err := p.client.Push(pushReq)
    if err != nil {
        log.Printf("[JPush] 推送失败: %v", err)
        return &PushResponse{
            Success:    false,
            ErrorCode:  "JPUSH_ERROR",
            ErrorMsg:   err.Error(),
        }, nil
    }
    
    return &PushResponse{
        Success:   result.IsSuccess(),
        MsgID:     result.MsgID,
        ErrorCode: result.Error.Code,
        ErrorMsg:  result.Error.Message,
    }, nil
}

// BatchPush 批量推送
func (p *JPushProvider) BatchPush(ctx context.Context, reqs []*PushRequest) (*BatchPushResponse, error) {
    response := &BatchPushResponse{
        Total:   len(reqs),
        Results: make([]*PushResponse, 0, len(reqs)),
    }
    
    for _, req := range reqs {
        resp, _ := p.Push(ctx, req)
        response.Results = append(response.Results, resp)
        if resp.Success {
            response.SuccessNum++
        } else {
            response.FailedNum++
        }
    }
    
    return response, nil
}

// PushByAlias 按别名推送
func (p *JPushProvider) PushByAlias(ctx context.Context, alias string, req *PushRequest) (*PushResponse, error) {
    pushReq := p.buildJPushRequest(req)
    pushReq.Audience = jpush.Audience{
        Alias: []string{alias},
    }
    
    result, err := p.client.Push(pushReq)
    if err != nil {
        return &PushResponse{
            Success: false,
            ErrorCode: "JPUSH_ERROR",
            ErrorMsg: err.Error(),
        }, nil
    }
    
    return &PushResponse{
        Success: result.IsSuccess(),
        MsgID:   result.MsgID,
    }, nil
}

// IsHealthy 健康检查
func (p *JPushProvider) IsHealthy(ctx context.Context) bool {
    // 简单检查：尝试获取应用信息
    _, err := p.client.GetAppInfo()
    return err == nil
}

// buildJPushRequest 构建极光推送请求
func (p *JPushProvider) buildJPushRequest(req *PushRequest) *jpush.PushRequest {
    pushReq := &jpush.PushRequest{
        Platform: p.convertPlatform(req.Platform),
        Audience: jpush.Audience{
            RegistrationID: []string{req.PushToken},
        },
        Notification: &jpush.Notification{
            Alert: req.Body,
            Android: &jpush.AndroidNotification{
                Title: req.Title,
                Alert: req.Body,
                Badge: req.Badge,
                Extras: req.Extra,
            },
            IOS: &jpush.IOSNotification{
                Alert: jpush.IOSAlert{
                    Title: req.Title,
                    Body:  req.Body,
                },
                Badge:  req.Badge,
                Sound:  req.Sound,
                Extras: req.Extra,
            },
        },
        Options: jpush.Options{
            ApnsProduction: p.config.ApnsProduction,
            TimeToLive:     req.TTL,
        },
    }
    
    // 设置优先级
    if req.Priority == PushPriorityHigh {
        pushReq.Options.Priority = 1
    }
    
    return pushReq
}

// convertPlatform 转换平台类型
func (p *JPushProvider) convertPlatform(platform types.Platform) string {
    switch platform {
    case types.PlatformIOS:
        return "ios"
    case types.PlatformAndroid:
        return "android"
    default:
        return "all"
    }
}

// 后续接入极光推送时，只需：
// 1. go get github.com/jpush-go/jpush
// 2. 实现 JPushProvider
// 3. 修改配置文件
// 4. 修改 ProviderFactory 切换到极光
```

### 5. Provider 工厂

```go
// internal/push/provider/provider_factory.go
package provider

import (
    "fmt"
    "sync"
)

// ProviderFactory 推送提供商工厂
type ProviderFactory struct {
    mu        sync.RWMutex
    providers map[string]PushProvider
    active    string  // 当前使用的提供商名称
}

// NewProviderFactory 创建工厂
func NewProviderFactory() *ProviderFactory {
    return &ProviderFactory{
        providers: make(map[string]PushProvider),
    }
}

// Register 注册提供商
func (f *ProviderFactory) Register(provider PushProvider) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.providers[provider.Name()] = provider
}

// SetActive 设置当前使用的提供商
func (f *ProviderFactory) SetActive(name string) error {
    f.mu.Lock()
    defer f.mu.Unlock()
    
    if _, ok := f.providers[name]; !ok {
        return fmt.Errorf("provider %s not registered", name)
    }
    f.active = name
    return nil
}

// GetProvider 获取当前提供商
func (f *ProviderFactory) GetProvider() (PushProvider, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    if f.active == "" {
        return nil, fmt.Errorf("no active provider")
    }
    
    provider, ok := f.providers[f.active]
    if !ok {
        return nil, fmt.Errorf("provider %s not found", f.active)
    }
    
    return provider, nil
}

// GetProviderByName 根据名称获取提供商
func (f *ProviderFactory) GetProviderByName(name string) (PushProvider, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()
    
    provider, ok := f.providers[name]
    if !ok {
        return nil, fmt.Errorf("provider %s not found", name)
    }
    
    return provider, nil
}
```

### 6. Push Service 核心逻辑

```go
// internal/push/service/push_service.go
package service

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    
    "im-system/internal/push/provider"
    "im-system/pkg/types"
)

// PushService 推送服务
type PushService struct {
    factory       *provider.ProviderFactory
    dedupService  *DedupService
    throttleService *ThrottleService
    deviceRepo    *repository.DeviceRepo
    contentBuilder *provider.PushContentBuilder
}

// NewPushService 创建推送服务
func NewPushService(
    factory *provider.ProviderFactory,
    dedupService *DedupService,
    throttleService *ThrottleService,
    deviceRepo *repository.DeviceRepo,
) *PushService {
    return &PushService{
        factory:         factory,
        dedupService:    dedupService,
        throttleService: throttleService,
        deviceRepo:      deviceRepo,
        contentBuilder:  &provider.PushContentBuilder{},
    }
}

// PushByMessage 根据消息推送
func (s *PushService) PushByMessage(ctx context.Context, userID string, msg *PushMessageEvent) error {
    // 1. 去重检查
    if s.dedupService.IsDuplicate(ctx, userID, msg.MsgID) {
        log.Printf("[Push] 消息已推送过，跳过: user=%s msg=%s", userID, msg.MsgID)
        return nil
    }
    
    // 2. 限流检查
    if !s.throttleService.Allow(ctx, userID) {
        log.Printf("[Push] 推送频率限制: user=%s", userID)
        return nil
    }
    
    // 3. 获取用户的所有设备
    devices, err := s.deviceRepo.FindByUID(ctx, userID)
    if err != nil {
        return fmt.Errorf("get devices failed: %w", err)
    }
    
    if len(devices) == 0 {
        log.Printf("[Push] 用户无设备: user=%s", userID)
        return nil
    }
    
    // 4. 构建推送内容
    title, body := s.contentBuilder.BuildPushContent(msg.MsgType, msg.Content, msg.SenderName)
    
    // 5. 构建扩展数据（点击推送跳转到对应会话）
    extra := map[string]interface{}{
        "chat_id":  msg.ChatID,
        "msg_id":   msg.MsgID,
        "msg_type": msg.MsgType,
        "action":   "open_chat",  // App收到推送后的动作
    }
    
    // 6. 获取当前推送提供商
    pushProvider, err := s.factory.GetProvider()
    if err != nil {
        return fmt.Errorf("get provider failed: %w", err)
    }
    
    // 7. 对每个设备发送推送
    pushRequests := make([]*provider.PushRequest, 0, len(devices))
    for _, device := range devices {
        req := &provider.PushRequest{
            Platform:  device.Platform,
            PushToken: device.PushToken,
            UserID:    userID,
            Title:     title,
            Body:      body,
            Sound:     "default",
            Badge:     s.getUnreadBadge(ctx, userID),
            Extra:     extra,
            MsgID:     msg.MsgID,
            ChatID:    msg.ChatID,
            MsgType:   msg.MsgType,
            Priority:  provider.PushPriorityNormal,
            TTL:       3600, // 1小时过期
        }
        pushRequests = append(pushRequests, req)
    }
    
    // 8. 批量推送
    response, err := pushProvider.BatchPush(ctx, pushRequests)
    if err != nil {
        return fmt.Errorf("batch push failed: %w", err)
    }
    
    // 9. 记录推送结果
    log.Printf("[Push] 推送完成: user=%s total=%d success=%d failed=%d",
        userID, response.Total, response.SuccessNum, response.FailedNum)
    
    // 10. 设置去重标记
    s.dedupService.MarkPushed(ctx, userID, msg.MsgID)
    
    return nil
}

// getUnreadBadge 获取未读消息数（用于角标）
func (s *PushService) getUnreadBadge(ctx context.Context, userID string) int {
    // 从会话服务获取总未读数
    // 这里简化处理
    return 1
}
```

### 7. 离线推送消费者

```go
// internal/push/consumer/offline_push_consumer.go
package consumer

import (
    "context"
    "encoding/json"
    "log"
    
    "im-system/pkg/queue/nats"
)

// OfflinePushConsumer 离线推送消费者
type OfflinePushConsumer struct {
    nats        *nats.Manager
    pushService *service.PushService
}

// Start 启动消费者
func (c *OfflinePushConsumer) Start(ctx context.Context) error {
    subject := "im.push.offline.>"
    
    _, err := c.nats.QueueSubscribe(subject, "offline-push-group", func(msg *nats.Msg) {
        // 解析目标用户ID
        parts := strings.Split(msg.Subject, ".")
        if len(parts) < 4 {
            return
        }
        targetUID := parts[3]
        
        var offlineEvent OfflinePushEvent
        if err := json.Unmarshal(msg.Data, &offlineEvent); err != nil {
            log.Printf("[OfflinePush] 解析失败: %v", err)
            return
        }
        
        log.Printf("[OfflinePush] 收到离线推送事件: user=%s msg=%s", targetUID, offlineEvent.Message.MsgID)
        
        // 执行推送
        if err := c.pushService.PushByMessage(ctx, targetUID, offlineEvent.Message); err != nil {
            log.Printf("[OfflinePush] 推送失败: %v", err)
            return
        }
        
        log.Printf("[OfflinePush] 推送成功: user=%s", targetUID)
    })
    
    return err
}
```

### 8. 去重服务

```go
// internal/push/service/dedup_service.go
package service

import (
    "context"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
)

// DedupService 推送去重服务
type DedupService struct {
    redis *redis.Client
    ttl   time.Duration
}

// NewDedupService 创建去重服务
func NewDedupService(redis *redis.Client) *DedupService {
    return &DedupService{
        redis: redis,
        ttl:   10 * time.Minute, // 10分钟内不重复推送
    }
}

// IsDuplicate 检查是否重复
func (s *DedupService) IsDuplicate(ctx context.Context, userID, msgID string) bool {
    key := fmt.Sprintf("push_dedup:%s:%s", userID, msgID)
    exists, _ := s.redis.Exists(ctx, key).Result()
    return exists > 0
}

// MarkPushed 标记已推送
func (s *DedupService) MarkPushed(ctx context.Context, userID, msgID string) {
    key := fmt.Sprintf("push_dedup:%s:%s", userID, msgID)
    s.redis.Set(ctx, key, time.Now().Unix(), s.ttl)
}
```

### 9. 限流服务

```go
// internal/push/service/throttle_service.go
package service

import (
    "context"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
)

// ThrottleService 推送限流服务
type ThrottleService struct {
    redis      *redis.Client
    maxPerMin  int64         // 每分钟最大推送次数
    windowSize time.Duration // 限流窗口
}

// NewThrottleService 创建限流服务
func NewThrottleService(redis *redis.Client) *ThrottleService {
    return &ThrottleService{
        redis:      redis,
        maxPerMin:  10,             // 每分钟最多10条推送
        windowSize: time.Minute,
    }
}

// Allow 检查是否允许推送
func (s *ThrottleService) Allow(ctx context.Context, userID string) bool {
    key := fmt.Sprintf("push_throttle:%s", userID)
    
    // 使用Redis的INCR + EXPIRE实现滑动窗口限流
    count, err := s.redis.Incr(ctx, key).Result()
    if err != nil {
        return true // Redis出错时放行
    }
    
    // 第一次设置过期时间
    if count == 1 {
        s.redis.Expire(ctx, key, s.windowSize)
    }
    
    return count <= s.maxPerMin
}
```

### 10. HTTP Handler（用于手动测试）

```go
// internal/push/handler/push_handler.go
package handler

import (
    "net/http"
    
    "github.com/gin-gonic/gin"
)

// PushHandler 推送处理器（用于手动测试）
type PushHandler struct {
    pushService *service.PushService
    factory     *provider.ProviderFactory
}

// TestPush 测试推送接口
// POST /api/v1/push/test
func (h *PushHandler) TestPush(c *gin.Context) {
    var req struct {
        UserID    string `json:"user_id" binding:"required"`
        Title     string `json:"title" binding:"required"`
        Body      string `json:"body" binding:"required"`
        Platform  string `json:"platform" binding:"required"` // ios/android
        PushToken string `json:"push_token" binding:"required"`
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"code": -1, "msg": err.Error()})
        return
    }
    
    // 手动构建推送请求
    pushReq := &provider.PushRequest{
        Platform:  req.Platform,
        PushToken: req.PushToken,
        UserID:    req.UserID,
        Title:     req.Title,
        Body:      req.Body,
        Sound:     "default",
        Badge:     1,
        Extra: map[string]interface{}{
            "action": "test",
        },
        Priority: provider.PushPriorityNormal,
        TTL:      3600,
    }
    
    // 获取当前推送提供商
    pushProvider, _ := h.factory.GetProvider()
    
    // 执行推送
    resp, err := pushProvider.Push(c.Request.Context(), pushReq)
    if err != nil {
        c.JSON(500, gin.H{"code": -1, "msg": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "code": 0,
        "data": resp,
    })
}

// GetMockRecords 获取Mock推送记录
// GET /api/v1/push/mock/records
func (h *PushHandler) GetMockRecords(c *gin.Context) {
    mockProvider, _ := h.factory.GetProviderByName("mock")
    if mockProvider == nil {
        c.JSON(400, gin.H{"code": -1, "msg": "mock provider not found"})
        return
    }
    
    mock, ok := mockProvider.(*provider.MockPushProvider)
    if !ok {
        c.JSON(400, gin.H{"code": -1, "msg": "not mock provider"})
        return
    }
    
    records := mock.GetRecords(50)
    stats := mock.GetStats()
    
    c.JSON(200, gin.H{
        "code": 0,
        "data": gin.H{
            "records": records,
            "stats":   stats,
        },
    })
}

// GetMockStats 获取Mock统计
// GET /api/v1/push/mock/stats
func (h *PushHandler) GetMockStats(c *gin.Context) {
    mockProvider, _ := h.factory.GetProviderByName("mock")
    if mockProvider == nil {
        c.JSON(400, gin.H{"code": -1, "msg": "mock provider not found"})
        return
    }
    
    mock, ok := mockProvider.(*provider.MockPushProvider)
    if !ok {
        c.JSON(400, gin.H{"code": -1, "msg": "not mock provider"})
        return
    }
    
    c.JSON(200, gin.H{
        "code": 0,
        "data": mock.GetStats(),
    })
}

// SwitchProvider 切换推送提供商
// POST /api/v1/push/provider/switch
func (h *PushHandler) SwitchProvider(c *gin.Context) {
    var req struct {
        Provider string `json:"provider" binding:"required"` // mock/jpush
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"code": -1, "msg": err.Error()})
        return
    }
    
    if err := h.factory.SetActive(req.Provider); err != nil {
        c.JSON(400, gin.H{"code": -1, "msg": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "code": 0,
        "msg":  fmt.Sprintf("switched to %s", req.Provider),
    })
}
```

### 11. 配置文件

```yaml
# configs/config.yaml
push:
  # 当前使用的推送提供商: mock / jpush
  provider: mock
  
  # Mock配置（开发测试用）
  mock:
    log_dir: /tmp/im_push_logs
    failure_rate: 0.05          # 5%模拟失败率
    latency_min: 10             # 最小延迟10ms
    latency_max: 100            # 最大延迟100ms
    store_records: true         # 存储推送记录
  
  # 极光推送配置（生产环境）
  jpush:
    app_key: "your_app_key"
    master_secret: "your_master_secret"
    apns_production: false      # 开发环境false，生产true
    timeout: 30
    retry_times: 3
  
  # 去重配置
  dedup:
    ttl: 600                    # 10分钟
  
  # 限流配置
  throttle:
    max_per_min: 10             # 每分钟最多10条
```

### 12. 主程序初始化

```go
// cmd/push/main.go
package main

func main() {
    // 加载配置
    cfg := loadConfig()
    
    // 创建Mock Provider
    mockProvider, err := provider.NewMockPushProvider(provider.MockConfig{
        LogDir:       cfg.Push.Mock.LogDir,
        FailureRate:  cfg.Push.Mock.FailureRate,
        LatencyMin:   cfg.Push.Mock.LatencyMin,
        LatencyMax:   cfg.Push.Mock.LatencyMax,
        StoreRecords: cfg.Push.Mock.StoreRecords,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // 创建Provider Factory
    factory := provider.NewProviderFactory()
    factory.Register(mockProvider)
    
    // 设置当前使用的Provider
    factory.SetActive(cfg.Push.Provider)
    
    // 创建Push Service
    pushService := service.NewPushService(
        factory,
        dedupService,
        throttleService,
        deviceRepo,
    )
    
    // 后续接入极光推送时：
    // jpushProvider, _ := provider.NewJPushProvider(provider.JPushConfig{...})
    // factory.Register(jpushProvider)
    // factory.SetActive("jpush")
    
    log.Printf("[Push] 推送服务启动，当前提供商: %s", cfg.Push.Provider)
}
```

## 二、Mock Provider 的特点

1. **完整的接口实现**：实现了 PushProvider 接口的所有方法
2. **可配置的失败率**：模拟真实的推送失败场景
3. **模拟延迟**：模拟网络延迟
4. **推送记录**：记录所有推送请求，方便调试
5. **日志输出**：详细的日志，方便追踪
6. **统计信息**：推送成功/失败统计

## 三、切换到极光推送的步骤

```go
// 只需三步：

// 1. 安装极光推送SDK
// go get github.com/jpush-go/jpush

// 2. 修改配置文件
push:
  provider: jpush           # 改为jpush
  jpush:
    app_key: "你的AppKey"
    master_secret: "你的MasterSecret"

// 3. 在主程序中注册极光推送
jpushProvider, _ := provider.NewJPushProvider(provider.JPushConfig{...})
factory.Register(jpushProvider)
factory.SetActive("jpush")
```

## 四、测试 Mock Push

```bash
# 1. 测试推送
curl -X POST http://localhost:8080/api/v1/push/test \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "title": "新消息",
    "body": "你有一条新消息",
    "platform": "ios",
    "push_token": "device_token_abc123"
  }'

# 2. 查看Mock推送记录
curl http://localhost:8080/api/v1/push/mock/records

# 3. 查看Mock统计
curl http://localhost:8080/api/v1/push/mock/stats

# 4. 查看Mock日志文件
cat /tmp/im_push_logs/mock_push.log
```

这个设计让你现在就可以完整开发和测试推送功能，后续接入极光推送时只需要实现 PushProvider 接口即可，业务代码完全不需要修改。