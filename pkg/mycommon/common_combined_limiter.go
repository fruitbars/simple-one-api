package mycommon

import (
	"context"
	"fmt"
	"time"

	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mylimiter"
)

const (
	LimitScopeProvider        = "provider"
	LimitScopeModel           = "model"
	LimitScopeCredential      = "credential"
	LimitScopeCredentialModel = "credential_model"
)

type LimitTarget struct {
	Key   string
	Scope string
	Limit config.Limit
}

type LimitWaitError struct {
	Key   string
	Scope string
	Err   error
}

func (err *LimitWaitError) Error() string {
	return fmt.Sprintf("limit wait for %s: %v", err.Key, err.Err)
}

func (err *LimitWaitError) Unwrap() error {
	return err.Err
}

func HasCombinedLimit(limit config.Limit) bool {
	return limit.QPS > 0 || limit.QPM > 0 || limit.RPM > 0 || limit.TPM > 0 || limit.Concurrency > 0
}

// AcquireCombinedLimits applies every configured rate and concurrency limit in
// target order. The returned function must be called after the upstream attempt.
func AcquireCombinedLimits(ctx context.Context, targets []LimitTarget, tokenCost, defaultTimeout int) (func(), error) {
	acquired := make([]*mylimiter.Limiter, 0, len(targets))
	release := func() {
		for index := len(acquired) - 1; index >= 0; index-- {
			acquired[index].Release()
		}
	}
	for _, target := range targets {
		if !HasCombinedLimit(target.Limit) {
			continue
		}
		timeout := target.Limit.Timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}
		limitCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
		limiter := mylimiter.GetCombinedLimiter(target.Key, mylimiter.Limits{
			QPS: target.Limit.QPS, QPM: target.Limit.QPM, RPM: target.Limit.RPM,
			TPM: target.Limit.TPM, Concurrency: target.Limit.Concurrency,
		})
		err := limiter.WaitN(limitCtx, tokenCost)
		if err == nil {
			err = limiter.Acquire(limitCtx)
		}
		cancel()
		if err != nil {
			release()
			return func() {}, &LimitWaitError{Key: target.Key, Scope: target.Scope, Err: err}
		}
		acquired = append(acquired, limiter)
	}
	return release, nil
}
