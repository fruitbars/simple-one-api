package config

import (
	"sync"
	"time"
)

type breakerState struct {
	failures    int
	openedAt    time.Time
	halfOpen    int
	lastProbeAt time.Time
}

var providerBreakers = struct {
	sync.Mutex
	states map[string]*breakerState
}{states: make(map[string]*breakerState)}

func breakerKey(serviceID, model string) string { return serviceID + "\x00" + model }

func currentBreakerConfig() CircuitBreakerConf {
	conf := currentSnapshot().configuration.CircuitBreaker
	if conf.FailureThreshold <= 0 {
		conf.FailureThreshold = 5
	}
	if conf.RecoveryTimeoutSeconds <= 0 {
		conf.RecoveryTimeoutSeconds = 30
	}
	if conf.HalfOpenMaxRequests <= 0 {
		conf.HalfOpenMaxRequests = 1
	}
	return conf
}

// CircuitBreakerAllow returns whether a provider/model pair may receive a
// request. After the recovery timeout, a bounded number of requests probe the
// provider in half-open state.
func CircuitBreakerAllow(serviceID, model string) bool {
	conf := currentBreakerConfig()
	if conf.Enabled != nil && !*conf.Enabled || serviceID == "" {
		return true
	}
	providerBreakers.Lock()
	defer providerBreakers.Unlock()
	state := providerBreakers.states[breakerKey(serviceID, model)]
	if state == nil || state.openedAt.IsZero() {
		return true
	}
	if time.Since(state.openedAt) < time.Duration(conf.RecoveryTimeoutSeconds)*time.Second {
		return false
	}
	if state.halfOpen >= conf.HalfOpenMaxRequests && time.Since(state.lastProbeAt) >= time.Duration(conf.RecoveryTimeoutSeconds)*time.Second {
		state.halfOpen = 0
	}
	if state.halfOpen >= conf.HalfOpenMaxRequests {
		return false
	}
	state.halfOpen++
	state.lastProbeAt = time.Now()
	return true
}

func CircuitBreakerAvailable(serviceID, model string) bool {
	conf := currentBreakerConfig()
	if conf.Enabled != nil && !*conf.Enabled || serviceID == "" {
		return true
	}
	providerBreakers.Lock()
	defer providerBreakers.Unlock()
	state := providerBreakers.states[breakerKey(serviceID, model)]
	if state == nil || state.openedAt.IsZero() {
		return true
	}
	if time.Since(state.openedAt) < time.Duration(conf.RecoveryTimeoutSeconds)*time.Second {
		return false
	}
	return state.halfOpen < conf.HalfOpenMaxRequests || time.Since(state.lastProbeAt) >= time.Duration(conf.RecoveryTimeoutSeconds)*time.Second
}

func RecordProviderResult(serviceID, model string, success bool) {
	conf := currentBreakerConfig()
	if conf.Enabled != nil && !*conf.Enabled || serviceID == "" {
		return
	}
	key := breakerKey(serviceID, model)
	providerBreakers.Lock()
	defer providerBreakers.Unlock()
	state := providerBreakers.states[key]
	if success {
		delete(providerBreakers.states, key)
		return
	}
	if state == nil {
		state = &breakerState{}
		providerBreakers.states[key] = state
	}
	if !state.openedAt.IsZero() {
		state.openedAt = time.Now()
		state.halfOpen = 0
		state.lastProbeAt = time.Time{}
		return
	}
	state.failures++
	if state.failures >= conf.FailureThreshold {
		state.openedAt = time.Now()
	}
}

func resetCircuitBreakersForTest() {
	providerBreakers.Lock()
	defer providerBreakers.Unlock()
	providerBreakers.states = make(map[string]*breakerState)
}
