package mycommon

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"simple-one-api/pkg/config"
)

func TestGetCredentialLimitAcceptsJSONFloatTimeout(t *testing.T) {
	var credentials map[string]interface{}
	if err := json.Unmarshal([]byte(`{"limit":{"qps":2,"timeout":12}}`), &credentials); err != nil {
		t.Fatalf("decode credentials: %v", err)
	}

	limitType, limit, timeout := GetCredentialLimit(credentials)
	if limitType != "qps" {
		t.Fatalf("unexpected limit type: %q", limitType)
	}
	if limit != 2 {
		t.Fatalf("unexpected limit value: %v", limit)
	}
	if timeout != 12 {
		t.Fatalf("JSON-decoded timeout must be preserved, got %d", timeout)
	}
}

func TestGetCredentialLimitsReadsCombinedValues(t *testing.T) {
	credentials := map[string]interface{}{"limit": map[string]interface{}{
		"qps": 1.0, "qpm": 2.0, "rpm": 3.0, "tpm": 400.0, "concurrency": 5.0, "timeout": 6.0,
	}}
	got := GetCredentialLimits(credentials)
	if got.QPS != 1 || got.QPM != 2 || got.RPM != 3 || got.TPM != 400 || got.Concurrency != 5 || got.Timeout != 6 {
		t.Fatalf("combined credential limits = %#v", got)
	}
}

func TestGetCredentialModelLimitMatchesConfiguredModel(t *testing.T) {
	credentials := map[string]interface{}{"model_limits": map[string]interface{}{
		"model-a": map[string]interface{}{"tpm": 1000.0, "concurrency": 2.0},
	}}
	limit, name, ok := GetCredentialModelLimit(credentials, "alias", "model-a")
	if !ok || name != "model-a" || limit.TPM != 1000 || limit.Concurrency != 2 {
		t.Fatalf("model limit = %#v, %q, %v", limit, name, ok)
	}
}

func TestGetCredentialCandidatesFallsBackToPositionalID(t *testing.T) {
	service := &config.ModelDetails{
		ServiceID: "provider-a",
		ServiceModel: config.ServiceModel{
			CredentialList: []map[string]interface{}{
				{"api_key": "first", "enabled": false},
				{"api_key": "second"},
			},
		},
	}
	candidates := GetCredentialCandidates(service, "model-a")
	if len(candidates) != 1 || candidates[0].ID != "provider-a:key-2" {
		t.Fatalf("candidates = %#v", candidates)
	}
}

func TestOrderCredentialCandidatesPrefersAvailableCapacity(t *testing.T) {
	service := &config.ModelDetails{
		ServiceID: "capacity-provider",
		ServiceModel: config.ServiceModel{CredentialList: []map[string]interface{}{
			{"id": "key-a", "model_limits": map[string]interface{}{"model-a": map[string]interface{}{"tpm": 100.0}}},
			{"id": "key-b", "model_limits": map[string]interface{}{"model-a": map[string]interface{}{"tpm": 100.0}}},
		}},
	}
	candidates := GetCredentialCandidates(service, "model-a")
	ReserveCredentialCapacity(candidates[0].Credentials, candidates[0].ID, "model-a", 80)
	ordered := OrderCredentialCandidates(service, "model-a", 80)
	if len(ordered) != 2 || ordered[0].ID != candidates[1].ID {
		t.Fatalf("ordered candidates = %#v, want available key first", ordered)
	}
}

func TestOrderCredentialCandidatesHonorsCooldown(t *testing.T) {
	service := &config.ModelDetails{
		ServiceID: "cooldown-provider",
		ServiceModel: config.ServiceModel{CredentialList: []map[string]interface{}{
			{"id": "key-a", "model_limits": map[string]interface{}{"model-a": map[string]interface{}{"tpm": 100.0}}},
			{"id": "key-b", "model_limits": map[string]interface{}{"model-a": map[string]interface{}{"tpm": 100.0}}},
		}},
	}
	candidates := GetCredentialCandidates(service, "model-a")
	MarkCredentialCooldown(candidates[0].ID, "model-a", time.Minute)
	ordered := OrderCredentialCandidates(service, "model-a", 1)
	if len(ordered) != 2 || ordered[0].ID != candidates[1].ID {
		t.Fatalf("ordered candidates = %#v, want non-cooled key first", ordered)
	}
}

func TestAcquireCombinedLimitsReportsProviderScope(t *testing.T) {
	key := t.Name() + ":provider"
	targets := []LimitTarget{{
		Key: key, Scope: LimitScopeProvider,
		Limit: config.Limit{Concurrency: 1, Timeout: 1},
	}}
	release, err := AcquireCombinedLimits(context.Background(), targets, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = AcquireCombinedLimits(ctx, targets, 1, 1)
	var waitErr *LimitWaitError
	if !errors.As(err, &waitErr) || waitErr.Scope != LimitScopeProvider {
		t.Fatalf("error = %#v, want provider LimitWaitError", err)
	}
}

func TestAcquireCombinedLimitsReleasesProviderWhenCredentialWaitFails(t *testing.T) {
	providerKey := t.Name() + ":provider"
	credentialKey := t.Name() + ":credential"
	credentialTarget := []LimitTarget{{
		Key: credentialKey, Scope: LimitScopeCredential,
		Limit: config.Limit{Concurrency: 1, Timeout: 1},
	}}
	releaseCredential, err := AcquireCombinedLimits(context.Background(), credentialTarget, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer releaseCredential()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err = AcquireCombinedLimits(ctx, []LimitTarget{
		{Key: providerKey, Scope: LimitScopeProvider, Limit: config.Limit{Concurrency: 1}},
		credentialTarget[0],
	}, 1, 1)
	var waitErr *LimitWaitError
	if !errors.As(err, &waitErr) || waitErr.Scope != LimitScopeCredential {
		t.Fatalf("error = %#v, want credential LimitWaitError", err)
	}

	releaseProvider, err := AcquireCombinedLimits(context.Background(), []LimitTarget{{
		Key: providerKey, Scope: LimitScopeProvider, Limit: config.Limit{Concurrency: 1},
	}}, 1, 1)
	if err != nil {
		t.Fatalf("provider permit leaked after credential wait failed: %v", err)
	}
	releaseProvider()
}
