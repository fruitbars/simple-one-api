package mylimiter

import (
	"context"
	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"simple-one-api/pkg/mycomdef"
	"sync"
)

type Limiter struct {
	QPSLimiter         *rate.Limiter
	ConcurrencyLimiter *semaphore.Weighted
}

var (
	limiterMap = make(map[string]*Limiter)
	mapMutex   sync.RWMutex
)

// NewLimiter 创建一个新的限流器，根据指定的类型和限制值进行配置
func NewLimiter(limitType string, limitn float64) *Limiter {
	lim := &Limiter{}
	switch limitType {
	case mycomdef.KEYNAME_QPS:
		lim.QPSLimiter = rate.NewLimiter(rate.Limit(limitn), int(limitn))
	case mycomdef.KEYNAME_QPM, mycomdef.KEYNAME_RPM:
		qps := float64(limitn) / 60.0
		lim.QPSLimiter = rate.NewLimiter(rate.Limit(qps), int(qps*2))
	case mycomdef.KEYNAME_CONCURRENCY:
		lim.ConcurrencyLimiter = semaphore.NewWeighted(int64(limitn))
	default:
		// 对无效类型无操作，或者可以抛出错误
	}
	return lim
}

// Wait 使用QPS限流器等待直到获得令牌
func (l *Limiter) Wait(ctx context.Context) error {
	if l.QPSLimiter != nil {
		return l.QPSLimiter.Wait(ctx)
	}
	return nil
}

// Acquire 尝试获取并发限制的许可，如果设置了超时则可以被中断
func (l *Limiter) Acquire(ctx context.Context) error {
	if l.ConcurrencyLimiter != nil {
		return l.ConcurrencyLimiter.Acquire(ctx, 1)
	}
	return nil
}

// Release 释放并发限制的一个许可
func (l *Limiter) Release() {
	if l.ConcurrencyLimiter != nil {
		l.ConcurrencyLimiter.Release(1)
	}
}

// GetLimiter 根据键获取或创建对应的限流器，支持线程安全操作
func GetLimiter(key string, limitType string, limitn float64) *Limiter {
	mapMutex.RLock()
	if lim, exists := limiterMap[key]; exists {
		mapMutex.RUnlock()
		return lim
	}
	mapMutex.RUnlock()

	mapMutex.Lock()
	defer mapMutex.Unlock()
	// 双重检查以防在锁定期间已被创建
	if lim, exists := limiterMap[key]; exists {
		return lim
	}

	lim := NewLimiter(limitType, limitn)
	limiterMap[key] = lim
	return lim
}
