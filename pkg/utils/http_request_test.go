package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"simple-one-api/pkg/mylog"
)

func TestMain(m *testing.M) {
	mylog.Logger = zap.NewNop()
	m.Run()
}

func TestSendHTTPRequestContextSupportsConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			body, err := SendHTTPRequestContext(context.Background(), "key", server.URL, []byte(fmt.Sprintf(`{"id":%d}`, i)), nil)
			if err != nil {
				t.Errorf("SendHTTPRequestContext() error = %v", err)
			}
			if string(body) != `{"ok":true}` {
				t.Errorf("body = %q", body)
			}
		}()
	}
	wg.Wait()
}

func TestNewHTTPClientHandlesTypedNilTransport(t *testing.T) {
	var transport *http.Transport
	client := NewHTTPClient(transport, time.Second)
	if client.Transport == nil {
		t.Fatal("NewHTTPClient() did not install the shared transport")
	}
}

func TestSendHTTPRequestContextCancelsUpstream(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := SendHTTPRequestContext(ctx, "key", server.URL, nil, nil)
		done <- err
	}()
	<-started
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(time.Second):
		t.Fatal("request did not stop after cancellation")
	}
}

func TestSendSSERequestContextDeliversFinalLineWithoutNewline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: first\n\ndata: final"))
	}))
	defer server.Close()

	var events []string
	err := SendSSERequestContext(context.Background(), "key", server.URL, nil, func(data string) {
		events = append(events, data)
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0] != "first" || events[1] != "final" {
		t.Fatalf("events = %#v", events)
	}
}

func TestSendSSERequestContextReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	err := SendSSERequestContext(context.Background(), "key", server.URL, nil, func(string) {}, nil)
	if err == nil {
		t.Fatal("expected upstream status error")
	}
}

func BenchmarkSendHTTPRequestContextParallel(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := SendHTTPRequestContext(context.Background(), "key", server.URL, []byte(`{"ping":true}`), nil); err != nil {
				b.Fatal(err)
			}
		}
	})
}
