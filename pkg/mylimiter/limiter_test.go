package mylimiter

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"simple-one-api/pkg/mycomdef"
)

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
