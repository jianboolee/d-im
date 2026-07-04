package provider

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// MockPushProvider Mock推送提供商（开发测试用）
type MockPushProvider struct {
	mu      sync.RWMutex
	records []*PushRecord
	stats   MockStats
	config  MockConfig
}

// PushRecord 推送记录
type PushRecord struct {
	ID          string       `json:"id"`
	PushRequest *PushRequest `json:"push_request"`
	Status      string       `json:"status"`
	SendTime    time.Time    `json:"send_time"`
	ErrorMsg    string       `json:"error_msg,omitempty"`
}

// MockStats Mock统计
type MockStats struct {
	TotalPushed  int64     `json:"total_pushed"`
	TotalSuccess int64     `json:"total_success"`
	TotalFailed  int64     `json:"total_failed"`
	LastPushTime time.Time `json:"last_push_time"`
}

// MockConfig Mock配置
type MockConfig struct {
	FailureRate float64 `yaml:"failure_rate"`
}

// NewMockPushProvider 创建Mock推送提供商
func NewMockPushProvider(config MockConfig) *MockPushProvider {
	return &MockPushProvider{
		records: make([]*PushRecord, 0),
		config:  config,
	}
}

func (p *MockPushProvider) Name() string { return "mock" }

func (p *MockPushProvider) Push(ctx context.Context, req *PushRequest) (*PushResponse, error) {
	if p.shouldFail() {
		return p.mockFailure(req), nil
	}
	return p.mockSuccess(req), nil
}

func (p *MockPushProvider) BatchPush(ctx context.Context, reqs []*PushRequest) (*BatchPushResponse, error) {
	resp := &BatchPushResponse{
		Total:   len(reqs),
		Results: make([]*PushResponse, 0, len(reqs)),
	}
	for _, req := range reqs {
		r, _ := p.Push(ctx, req)
		resp.Results = append(resp.Results, r)
		if r.Success {
			resp.SuccessNum++
		} else {
			resp.FailedNum++
		}
	}
	return resp, nil
}

func (p *MockPushProvider) IsHealthy(ctx context.Context) bool { return true }

func (p *MockPushProvider) mockSuccess(req *PushRequest) *PushResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalPushed++
	p.stats.TotalSuccess++
	p.stats.LastPushTime = time.Now()

	id := fmt.Sprintf("mock_%d_%d", time.Now().UnixNano(), p.stats.TotalPushed)
	p.records = append(p.records, &PushRecord{
		ID: id, PushRequest: req, Status: "success", SendTime: time.Now(),
	})

	log.Printf("[MockPush] ✅ platform=%s user=%s msg=%s title=%s",
		req.Platform, req.UserID, req.MsgID, req.Title)
	return &PushResponse{Success: true, MsgID: id}
}

func (p *MockPushProvider) mockFailure(req *PushRequest) *PushResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalPushed++
	p.stats.TotalFailed++
	p.stats.LastPushTime = time.Now()

	p.records = append(p.records, &PushRecord{
		ID:          fmt.Sprintf("mock_fail_%d", time.Now().UnixNano()),
		PushRequest: req, Status: "failed", SendTime: time.Now(),
		ErrorMsg: "simulated failure",
	})

	log.Printf("[MockPush] ❌ failed: platform=%s user=%s", req.Platform, req.UserID)
	return &PushResponse{Success: false, ErrorCode: "MOCK_ERR", ErrorMsg: "simulated failure"}
}

func (p *MockPushProvider) shouldFail() bool {
	return p.config.FailureRate > 0 && time.Now().UnixNano()%100 < int64(p.config.FailureRate*100)
}

func (p *MockPushProvider) GetStats() MockStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

func (p *MockPushProvider) GetRecords(limit int) []*PushRecord {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if limit > 0 && limit < len(p.records) {
		return p.records[len(p.records)-limit:]
	}
	return p.records
}
