package mylimiter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"simple-one-api/pkg/mycomdef"
	"sync"
)

var ErrTokenCostExceedsLimit = errors.New("estimated token cost exceeds TPM limit")

type Limiter struct {
	QPSLimiter         *rate.Limiter
	QPMLimiter         *SlidingWindowLimiter
	RPMLimiter         *SlidingWindowLimiter
	TPMLimiter         *WeightedSlidingWindowLimiter
	ConcurrencyLimiter *semaphore.Weighted
}

type Limits struct {
	QPS         float64
	QPM         float64
	RPM         float64
	TPM         float64
	Concurrency float64
}

type SlidingWindowLimiter struct {
	mu          sync.Mutex
	maxRequests int
	interval    time.Duration
	requests    []time.Time
}

type weightedWindowEntry struct {
	at   time.Time
	cost int
}

type WeightedSlidingWindowLimiter struct {
	mu       sync.Mutex
	maximum  int
	interval time.Duration
	used     int
	entries  []weightedWindowEntry
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

func NewWeightedSlidingWindowLimiter(maximum int) *WeightedSlidingWindowLimiter {
	return &WeightedSlidingWindowLimiter{maximum: maximum, interval: time.Minute}
}

func (l *WeightedSlidingWindowLimiter) reserve(cost int) (bool, time.Duration) {
	if cost < 1 {
		cost = 1
	}
	if cost > l.maximum {
		return false, l.interval
	}
	now := time.Now()
	windowStart := now.Add(-l.interval)
	l.mu.Lock()
	defer l.mu.Unlock()
	index := 0
	for index < len(l.entries) && l.entries[index].at.Before(windowStart) {
		l.used -= l.entries[index].cost
		index++
	}
	l.entries = l.entries[index:]
	if l.used+cost <= l.maximum {
		l.entries = append(l.entries, weightedWindowEntry{at: now, cost: cost})
		l.used += cost
		return true, 0
	}
	if len(l.entries) == 0 {
		return false, l.interval
	}
	return false, time.Until(l.entries[0].at.Add(l.interval))
}

func (l *WeightedSlidingWindowLimiter) Wait(ctx context.Context, cost int) error {
	if cost > l.maximum {
		return fmt.Errorf("%w: cost %d, limit %d", ErrTokenCostExceedsLimit, cost, l.maximum)
	}
	for {
		allowed, wait := l.reserve(cost)
		if allowed {
			return nil
		}
		if wait < 10*time.Millisecond {
			wait = 10 * time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
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

func NewCombinedLimiter(limits Limits) *Limiter {
	limiter := &Limiter{}
	if limits.QPS > 0 {
		burst := int(limits.QPS)
		if burst < 1 {
			burst = 1
		}
		limiter.QPSLimiter = rate.NewLimiter(rate.Limit(limits.QPS), burst)
	}
	if limits.QPM > 0 {
		limiter.QPMLimiter = NewSlidingWindowLimiter(positiveWholeLimit(limits.QPM))
	}
	if limits.RPM > 0 {
		limiter.RPMLimiter = NewSlidingWindowLimiter(positiveWholeLimit(limits.RPM))
	}
	if limits.TPM > 0 {
		limiter.TPMLimiter = NewWeightedSlidingWindowLimiter(positiveWholeLimit(limits.TPM))
	}
	if limits.Concurrency > 0 {
		limiter.ConcurrencyLimiter = semaphore.NewWeighted(int64(positiveWholeLimit(limits.Concurrency)))
	}
	return limiter
}

func positiveWholeLimit(value float64) int {
	whole := int(value)
	if whole < 1 {
		return 1
	}
	return whole
}

// Wait 使用QPS限流器等待直到获得令牌
func (l *Limiter) Wait(ctx context.Context) error {
	return l.WaitN(ctx, 1)
}

// WaitN applies every configured rate constraint. tokenCost is only consumed
// by the TPM window; request-based windows consume one request.
func (l *Limiter) WaitN(ctx context.Context, tokenCost int) error {
	if l.TPMLimiter != nil && tokenCost > l.TPMLimiter.maximum {
		return fmt.Errorf("%w: cost %d, limit %d", ErrTokenCostExceedsLimit, tokenCost, l.TPMLimiter.maximum)
	}
	if l.QPSLimiter != nil {
		if err := l.QPSLimiter.Wait(ctx); err != nil {
			return err
		}
	}
	if l.QPMLimiter != nil {
		if err := l.QPMLimiter.Wait(ctx); err != nil {
			return err
		}
	}
	if l.RPMLimiter != nil {
		if err := l.RPMLimiter.Wait(ctx); err != nil {
			return err
		}
	}
	if l.TPMLimiter != nil {
		if err := l.TPMLimiter.Wait(ctx, tokenCost); err != nil {
			return err
		}
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

func GetCombinedLimiter(key string, limits Limits) *Limiter {
	signature := key + ":combined:" + fmt.Sprintf("%g:%g:%g:%g:%g", limits.QPS, limits.QPM, limits.RPM, limits.TPM, limits.Concurrency)
	mapMutex.RLock()
	if limiter, exists := limiterMap[signature]; exists {
		mapMutex.RUnlock()
		return limiter
	}
	mapMutex.RUnlock()
	mapMutex.Lock()
	defer mapMutex.Unlock()
	if limiter, exists := limiterMap[signature]; exists {
		return limiter
	}
	limiter := NewCombinedLimiter(limits)
	limiterMap[signature] = limiter
	return limiter
}
