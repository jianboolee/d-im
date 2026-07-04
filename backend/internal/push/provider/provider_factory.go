package provider

import (
	"fmt"
	"sync"
)

// ProviderFactory 推送提供商工厂
type ProviderFactory struct {
	mu        sync.RWMutex
	providers map[string]PushProvider
	active    string
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
	p, ok := f.providers[f.active]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", f.active)
	}
	return p, nil
}

// GetMock 获取Mock提供商（便捷方法）
func (f *ProviderFactory) GetMock() (*MockPushProvider, error) {
	p, err := f.GetProvider()
	if err != nil {
		return nil, err
	}
	m, ok := p.(*MockPushProvider)
	if !ok {
		return nil, fmt.Errorf("active provider is not mock")
	}
	return m, nil
}
