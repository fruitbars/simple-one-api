package main

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"simple-one-api/pkg/config"
)

func testDesktopBridge(handler http.Handler) (*DesktopBridge, *eventRecorder) {
	recorder := &eventRecorder{}
	bridge := NewDesktopBridge(handler)
	bridge.emit = recorder.emit
	bridge.startup(context.Background())
	return bridge, recorder
}

type eventRecorder struct {
	mu     sync.Mutex
	events []string
}

func (recorder *eventRecorder) emit(_ context.Context, _ string, data ...interface{}) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(data) > 0 {
		recorder.events = append(recorder.events, data[0].(string))
	}
}

func (recorder *eventRecorder) decoded(t *testing.T) []string {
	t.Helper()
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	result := make([]string, 0, len(recorder.events))
	for _, event := range recorder.events {
		data, err := base64.StdEncoding.DecodeString(event)
		if err != nil {
			t.Fatalf("decode event: %v", err)
		}
		result = append(result, string(data))
	}
	return result
}

func TestDesktopBridgeStreamsChunksAndForwardsAPIKey(t *testing.T) {
	bridge, events := testDesktopBridge(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer desktop-key" {
			t.Errorf("authorization = %q", got)
		}
		if got := request.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("data: first\n\n"))
		writer.(http.Flusher).Flush()
		_, _ = writer.Write([]byte("data: second\n\n"))
	}))

	if err := bridge.StreamChat("request-1", " desktop-key ", `{"stream":true}`); err != nil {
		t.Fatalf("stream chat: %v", err)
	}
	want := []string{"data: first\n\n", "data: second\n\n"}
	got := events.decoded(t)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("events = %#v, want %#v", got, want)
	}
}

func TestDesktopBridgeUsesConfiguredKeyWhenUIKeyIsEmpty(t *testing.T) {
	if err := config.ApplyConfiguration(config.Configuration{APIKey: "configured-key"}, "desktop-bridge-test"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = config.ApplyConfiguration(config.Configuration{}, "desktop-bridge-test-cleanup") })

	bridge, _ := testDesktopBridge(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Authorization"); got != "Bearer configured-key" {
			t.Errorf("authorization = %q", got)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	if err := bridge.StreamChat("configured-key-request", "", `{}`); err != nil {
		t.Fatalf("stream chat: %v", err)
	}
}

func TestDesktopBridgeReturnsBoundedHTTPError(t *testing.T) {
	bridge, events := testDesktopBridge(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(strings.Repeat("x", desktopErrorBodyLimit+1024)))
	}))

	err := bridge.StreamChat("request-2", "", `{}`)
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("error = %v", err)
	}
	if len(err.Error()) > desktopErrorBodyLimit+128 {
		t.Fatalf("error was not bounded: %d bytes", len(err.Error()))
	}
	if got := events.decoded(t); len(got) != 0 {
		t.Fatalf("unexpected streamed error body: %#v", got)
	}
}

func TestDesktopBridgeCancellationStopsRequest(t *testing.T) {
	started := make(chan struct{})
	bridge, _ := testDesktopBridge(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	done := make(chan error, 1)
	go func() { done <- bridge.StreamChat("request-3", "", `{}`) }()
	<-started
	bridge.CancelChat("request-3")

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop after cancellation")
	}
}

func TestDesktopBridgeRejectsInvalidAndDuplicateRequestIDs(t *testing.T) {
	started := make(chan struct{})
	bridge, _ := testDesktopBridge(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(started)
		<-request.Context().Done()
	}))
	if err := bridge.StreamChat("bad id", "", `{}`); err == nil {
		t.Fatal("invalid request ID was accepted")
	}

	done := make(chan error, 1)
	go func() { done <- bridge.StreamChat("duplicate", "", `{}`) }()
	<-started
	if err := bridge.StreamChat("duplicate", "", `{}`); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("duplicate error = %v", err)
	}
	bridge.CancelChat("duplicate")
	<-done
}
