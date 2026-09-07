package handler

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"simple-one-api/pkg/config"
	"simple-one-api/pkg/mycommon"
	"simple-one-api/pkg/utils"
)

func TestDispatchErrorStatusPreservesUpstreamStatus(t *testing.T) {
	err := fmt.Errorf("request failed: %w", utils.NewHTTPStatusError(http.StatusNotFound, "404 Not Found", "no route"))
	if got := dispatchErrorStatus(err); got != http.StatusNotFound {
		t.Fatalf("dispatchErrorStatus() = %d, want %d", got, http.StatusNotFound)
	}
}

func TestAcquireAttemptLimitsKeepsCredentialAndProviderModelScopesIndependent(t *testing.T) {
	modelLimit := config.Limit{Concurrency: 1, Timeout: 1}
	service := &config.ModelDetails{
		ServiceID: "provider-a",
		ServiceModel: config.ServiceModel{
			ModelLimits: map[string]config.Limit{"model-a": modelLimit},
		},
	}
	credential := mycommon.CredentialSelection{
		ID: "provider-a",
		Credentials: map[string]interface{}{
			"model_limits": map[string]interface{}{"model-a": modelLimit},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	release, err := acquireAttemptLimits(ctx, service, credential, &openai.ChatCompletionRequest{Model: "model-a"}, "model-a")
	if err != nil {
		t.Fatalf("independent model scopes should both be acquired: %v", err)
	}
	release()
}

func TestDispatchErrorStatusMapsLimitWaitToTooManyRequests(t *testing.T) {
	err := &mycommon.LimitWaitError{Key: "provider", Scope: mycommon.LimitScopeProvider, Err: context.DeadlineExceeded}
	if got := dispatchErrorStatus(err); got != http.StatusTooManyRequests {
		t.Fatalf("dispatchErrorStatus() = %d, want %d", got, http.StatusTooManyRequests)
	}
}
