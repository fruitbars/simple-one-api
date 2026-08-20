package oai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestResolveEmbeddingURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "default", want: defaultEmbeddingURL},
		{name: "base", input: "https://example.com/v1", want: "https://example.com/v1/embeddings"},
		{name: "trailing slash", input: "https://example.com/v1/", want: "https://example.com/v1/embeddings"},
		{name: "chat endpoint", input: "https://example.com/v1/chat/completions", want: "https://example.com/v1/embeddings"},
		{name: "embedding endpoint", input: "https://example.com/v1/embeddings", want: "https://example.com/v1/embeddings"},
		{name: "query", input: "https://example.com/openai/v1/chat/completions?api-version=1", want: "https://example.com/openai/v1/embeddings?api-version=1"},
		{name: "invalid scheme", input: "ftp://example.com/v1", wantErr: true},
		{name: "relative", input: "/v1", wantErr: true},
		{name: "fragment", input: "https://example.com/v1#fragment", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveEmbeddingURL(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("resolveEmbeddingURL(%q) returned no error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("resolveEmbeddingURL(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestOpenAIEmbeddingUsesConfiguredServerURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/embeddings" || request.URL.Query().Get("tenant") != "test" {
			t.Errorf("request URL = %q", request.URL.String())
		}
		if request.Header.Get("Authorization") != "Bearer upstream-key" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		var payload EmbeddingRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload.Model != "embedding-model" {
			t.Errorf("model = %q", payload.Model)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[],"model":"embedding-model","usage":{"prompt_tokens":1,"total_tokens":1}}`))
	}))
	defer server.Close()

	request := &EmbeddingRequest{Model: "embedding-model"}
	_, err := OpenAIEmbedding(context.Background(), request, "upstream-key", server.URL+"/v1/chat/completions?tenant=test", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenAIEmbeddingReturnsUpstreamStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "model unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	_, err := OpenAIEmbedding(context.Background(), &EmbeddingRequest{}, "key", server.URL, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 503") || !strings.Contains(err.Error(), "model unavailable") {
		t.Fatalf("error = %v", err)
	}
}

func TestOpenAIEmbeddingPropagatesCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		select {
		case <-request.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := OpenAIEmbedding(ctx, &EmbeddingRequest{}, "key", server.URL, nil)
		done <- err
	}()
	<-started
	cancel()

	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}
