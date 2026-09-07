package mylimiter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"simple-one-api/pkg/mycomdef"
)

func TestCombinedLimiterEnablesEveryConfiguredConstraint(t *testing.T) {
	limiter := NewCombinedLimiter(Limits{QPS: 2, QPM: 3, RPM: 4, TPM: 100, Concurrency: 5})
	if limiter.QPSLimiter == nil || limiter.QPMLimiter == nil || limiter.RPMLimiter == nil || limiter.TPMLimiter == nil || limiter.ConcurrencyLimiter == nil {
		t.Fatalf("combined limiter did not initialize every constraint: %#v", limiter)
	}
}

func TestCombinedLimiterRejectsRequestLargerThanTPMImmediately(t *testing.T) {
	limiter := NewCombinedLimiter(Limits{QPS: 1, TPM: 10})
	started := time.Now()
	err := limiter.WaitN(context.Background(), 11)
	if !errors.Is(err, ErrTokenCostExceedsLimit) {
		t.Fatalf("WaitN() error = %v, want ErrTokenCostExceedsLimit", err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("oversized TPM request was not rejected immediately: %s", elapsed)
	}
}

func TestCombinedLimiterKeepsPositiveFractionalCountLimitsUsable(t *testing.T) {
	limiter := NewCombinedLimiter(Limits{QPM: 0.5, RPM: 0.5, TPM: 0.5, Concurrency: 0.5})
	if limiter.QPMLimiter.maxRequests != 1 || limiter.RPMLimiter.maxRequests != 1 || limiter.TPMLimiter.maximum != 1 || limiter.ConcurrencyLimiter == nil {
		t.Fatalf("fractional count limits were not normalized to a usable minimum: %#v", limiter)
	}
}

func TestConcurrencyLimiterNeverExceedsLimit(t *testing.T) {
	limiter := NewLimiter(mycomdef.KEYNAME_CONCURRENCY, 3)
	var active atomic.Int32
	var maximum atomic.Int32
	var wg sync.WaitGroup

	for range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := limiter.Acquire(context.Background()); err != nil {
				t.Errorf("Acquire() error = %v", err)
				return
			}
			current := active.Add(1)
			for {
				old := maximum.Load()
				if current <= old || maximum.CompareAndSwap(old, current) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			active.Add(-1)
			limiter.Release()
		}()
	}
	wg.Wait()

	if got := maximum.Load(); got != 3 {
		t.Fatalf("maximum concurrency = %d, want 3", got)
	}
}

func TestConcurrencyAcquireHonorsContext(t *testing.T) {
	limiter := NewLimiter(mycomdef.KEYNAME_CONCURRENCY, 1)
	if err := limiter.Acquire(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer limiter.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := limiter.Acquire(ctx); err == nil {
		t.Fatal("Acquire() succeeded while the only permit was held")
	}
}

func TestGetLimiterIsConcurrentSafe(t *testing.T) {
	const key = "shared-test-limiter"
	var first atomic.Pointer[Limiter]
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got := GetLimiter(key, mycomdef.KEYNAME_QPS, 10)
			if existing := first.Load(); existing == nil {
				first.CompareAndSwap(nil, got)
			} else if got != existing {
				t.Errorf("GetLimiter() returned different instances")
			}
		}()
	}
	wg.Wait()
}
