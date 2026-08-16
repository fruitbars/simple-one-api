package config

import (
	"testing"
	"time"
)

func TestCircuitBreakerOpensAndRecovers(t *testing.T) {
	previous := *CurrentConfiguration()
	previousPath := CurrentConfigPath()
	t.Cleanup(func() { _ = ApplyConfiguration(previous, previousPath); resetCircuitBreakersForTest() })
	enabled := true
	conf := Configuration{
		CircuitBreaker: CircuitBreakerConf{Enabled: &enabled, FailureThreshold: 2, RecoveryTimeoutSeconds: 60, HalfOpenMaxRequests: 1},
		Services:       map[string][]ServiceModel{"openai": {{ID: "provider-a", Enabled: true, Models: []string{"model-a"}}}},
	}
	if err := ApplyConfiguration(conf, "test.json"); err != nil {
		t.Fatal(err)
	}
	resetCircuitBreakersForTest()

	RecordProviderResult("provider-a", "model-a", false)
	if !CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("breaker opened before threshold")
	}
	RecordProviderResult("provider-a", "model-a", false)
	if CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("breaker did not open")
	}

	providerBreakers.Lock()
	providerBreakers.states[breakerKey("provider-a", "model-a")].openedAt = time.Now().Add(-time.Minute)
	providerBreakers.Unlock()
	if !CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("half-open probe was not allowed")
	}
	if CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("second half-open probe was allowed")
	}
	providerBreakers.Lock()
	providerBreakers.states[breakerKey("provider-a", "model-a")].lastProbeAt = time.Now().Add(-time.Minute)
	providerBreakers.Unlock()
	if !CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("expired half-open probe lease was not released")
	}
	RecordProviderResult("provider-a", "model-a", true)
	if !CircuitBreakerAllow("provider-a", "model-a") {
		t.Fatal("successful probe did not close breaker")
	}
}
