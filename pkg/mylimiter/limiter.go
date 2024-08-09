package mylimiter

import (
	"context"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"simple-one-api/pkg/mycomdef"
	"sync"
)

type Limiter struct {
	QPSLimiter         *rate.Limiter
	QPMLimiter         *SlidingWindowLimiter
	ConcurrencyLimiter *semaphore.Weighted
}

type SlidingWindowLimiter struct {
	mu          sync.Mutex
	maxRequests int
	interval    time.Duration
	requests    []time.Time
}

var (
	limiterMap = make(map[string]*Limiter)
	mapMutex   sync.RWMutex
)

func NewSlidingWindowLimiter(qpm int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		maxRequests: qpm,
		interval:    time.Minute,
		requests:    make([]time.Time, 0, qpm),
	}
}

func (l *SlidingWindowLimiter) Allow() bool {
	now := time.Now()
	windowStart := now.Add(-l.interval)

	l.mu.Lock()
	defer l.mu.Unlock()

	// 移除窗口外的请求
	i := 0
	for ; i < len(l.requests) && l.requests[i].Before(windowStart); i++ {
	}
	l.requests = l.requests[i:]

	// 检查是否允许新请求
	if len(l.requests) < l.maxRequests {
		l.requests = append(l.requests, now)
		return true
	}
	return false
}

func (l *SlidingWindowLimiter) Wait(ctx context.Context) error {
	waitTime := 10 * time.Millisecond // 初始等待时间

	for {
		if l.Allow() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(waitTime):
			l.mu.Lock()
			if len(l.requests) > 0 {
				// 计算到下一个请求可以被允许的时间间隔
				nextAllowedTime := l.requests[0].Add(l.interval)
				timeUntilNextAllowed := time.Until(nextAllowedTime)

				// 根据时间间隔调整等待时间
				if timeUntilNextAllowed < waitTime {
					waitTime = timeUntilNextAllowed
				} else {
					waitTime *= 2
					if waitTime > time.Second {
						waitTime = time.Second
					}
				}
			}
			l.mu.Unlock()
		}
	}
}

// NewLimiter 创建一个新的限流器，根据指定的类型和限制值进行配置
func NewLimiter(limitType string, limitn float64) *Limiter {
	lim := &Limiter{}
	switch limitType {
	case mycomdef.KEYNAME_QPS:
		lim.QPSLimiter = rate.NewLimiter(rate.Limit(limitn), int(limitn))
	case mycomdef.KEYNAME_QPM, mycomdef.KEYNAME_RPM:
		lim.QPMLimiter = NewSlidingWindowLimiter(int(limitn))
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
	if l.QPMLimiter != nil {
		return l.QPMLimiter.Wait(ctx)
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
