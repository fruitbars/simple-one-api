package statistics

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestServiceRecordsNullableUsageAndAggregates(t *testing.T) {
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 30})
	now := time.Now().UTC()
	service.Record(Event{RequestID: "known", CreatedAt: now.Add(-time.Hour), Protocol: "chat", AccessKeyID: "key-a", ClientModel: "model-a", ProviderID: "provider-a", ProviderName: "OpenAI", UpstreamModel: "upstream-a", StatusCode: 200, LatencyMS: 120, Usage: Usage{InputTokens: Int64(10), OutputTokens: Int64(5), CachedTokens: Int64(4), CacheWriteTokens: Int64(2), ReasoningTokens: Int64(1), TotalTokens: Int64(15), Source: "upstream"}})
	service.Record(Event{RequestID: "missing", CreatedAt: now.Add(-30 * time.Minute), Protocol: "responses", AccessKeyID: "key-b", ClientModel: "model-b", ProviderID: "provider-b", ProviderName: "Backup", StatusCode: 502, LatencyMS: 300, Usage: Usage{Source: "missing"}})
	waitForRows(t, service, 2)

	var input sql.NullInt64
	var source string
	if err := service.db.QueryRow(`SELECT input_tokens, usage_source FROM request_stats WHERE request_id = 'missing'`).Scan(&input, &source); err != nil {
		t.Fatal(err)
	}
	if input.Valid || source != "missing" {
		t.Fatalf("missing usage stored as %#v, %q", input, source)
	}
	overview, err := service.Overview(context.Background(), Query{From: now.Add(-2 * time.Hour), To: now.Add(time.Minute), Bucket: "hour"})
	if err != nil {
		t.Fatal(err)
	}
	if overview.Summary.Requests != 2 || overview.Summary.Successful != 1 || overview.Summary.InputTokens != 10 || overview.Summary.OutputTokens != 5 || overview.Summary.CachedTokens != 4 || overview.Summary.CacheWriteTokens != 2 || overview.Summary.ReasoningTokens != 1 || overview.Summary.TotalTokens != 15 || overview.Summary.KnownUsage != 1 {
		t.Fatalf("unexpected summary: %#v", overview.Summary)
	}
	if len(overview.Providers) != 2 || len(overview.RecentFailures) != 1 || overview.RecentFailures[0].RequestID != "missing" {
		t.Fatalf("unexpected overview: %#v", overview)
	}
	if len(overview.Series) < 2 {
		t.Fatalf("expected empty time buckets to be preserved: %#v", overview.Series)
	}
}

func TestOverviewFiltersPerformanceAndCSV(t *testing.T) {
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 30})
	now := time.Now().UTC()
	service.Record(Event{RequestID: "a-fast", CreatedAt: now.Add(-2 * time.Hour), Protocol: "chat", ProviderID: "provider-a", ProviderName: "A", ClientModel: "model-a", StatusCode: 200, LatencyMS: 200, FirstTokenMS: Int64(50), Usage: Usage{InputTokens: Int64(10), OutputTokens: Int64(15), TotalTokens: Int64(25), Source: "upstream"}})
	service.Record(Event{RequestID: "a-slow", CreatedAt: now.Add(-time.Hour), Protocol: "chat", ProviderID: "provider-a", ProviderName: "A", ClientModel: "model-b", StatusCode: 500, LatencyMS: 800, FirstTokenMS: Int64(300), Usage: Usage{Source: "missing"}})
	service.Record(Event{RequestID: "b", CreatedAt: now.Add(-30 * time.Minute), Protocol: "responses", ProviderID: "provider-b", ProviderName: "B", ClientModel: "model-a", StatusCode: 200, LatencyMS: 100, FirstTokenMS: Int64(20), Usage: Usage{InputTokens: Int64(2), OutputTokens: Int64(8), TotalTokens: Int64(10), Source: "upstream"}})
	waitForRows(t, service, 3)

	query := Query{From: now.Add(-3 * time.Hour), To: now.Add(time.Minute), Bucket: "hour", Provider: "provider-a"}
	overview, err := service.Overview(context.Background(), query)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Summary.Requests != 2 || overview.Summary.KnownUsage != 1 || overview.Summary.UsageRate != 50 {
		t.Fatalf("unexpected filtered usage summary: %#v", overview.Summary)
	}
	if overview.Summary.P50Latency != 200 || overview.Summary.P95Latency != 800 || overview.Summary.P50TTFT != 50 || overview.Summary.P95TTFT != 300 {
		t.Fatalf("unexpected percentiles: %#v", overview.Summary)
	}
	if overview.Summary.OutputTokensPerSecond != 100 {
		t.Fatalf("output rate = %v, want 100", overview.Summary.OutputTokensPerSecond)
	}
	if len(overview.Providers) != 1 || overview.Providers[0].UsageRate != 50 || len(overview.RecentFailures) != 1 {
		t.Fatalf("filters not consistently applied: %#v", overview)
	}

	var exported bytes.Buffer
	if err := service.ExportCSV(context.Background(), query, &exported); err != nil {
		t.Fatal(err)
	}
	text := exported.String()
	if !strings.Contains(text, "a-fast") || !strings.Contains(text, "a-slow") || strings.Contains(text, "\nb,") {
		t.Fatalf("unexpected CSV export: %q", text)
	}
}

func TestStatisticsDatabaseUsesWAL(t *testing.T) {
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 30})
	var mode string
	if err := service.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Fatalf("journal mode = %q, want wal", mode)
	}
}

func TestCloseDrainsAcceptedEventsAndRejectsNewRecords(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statistics.db")
	service, err := Open(path, Settings{Enabled: true, RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	const accepted = 200
	for index := 0; index < accepted; index++ {
		if !service.Record(Event{RequestID: fmt.Sprintf("request-%d", index), Protocol: "chat", StatusCode: 200, Usage: Usage{Source: "missing"}}) {
			t.Fatalf("record %d was unexpectedly rejected", index)
		}
	}
	if err := service.Close(); err != nil {
		t.Fatal(err)
	}
	if service.Record(Event{RequestID: "after-close", Protocol: "chat", StatusCode: 200}) {
		t.Fatal("record was accepted after close")
	}
	if err := service.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}

	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM request_stats`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != accepted {
		t.Fatalf("persisted rows = %d, want %d", count, accepted)
	}
}

func TestOpenMigratesExistingStatisticsTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	database, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(`CREATE TABLE request_stats (
		id INTEGER PRIMARY KEY AUTOINCREMENT, request_id TEXT NOT NULL, created_at TEXT NOT NULL,
		protocol TEXT NOT NULL, access_key_id TEXT NOT NULL DEFAULT '', client_model TEXT NOT NULL DEFAULT '',
		provider_id TEXT NOT NULL DEFAULT '', provider_name TEXT NOT NULL DEFAULT '', upstream_model TEXT NOT NULL DEFAULT '',
		status_code INTEGER NOT NULL, latency_ms INTEGER NOT NULL, first_byte_ms INTEGER,
		input_tokens INTEGER, output_tokens INTEGER, cached_tokens INTEGER, cache_write_tokens INTEGER,
		reasoning_tokens INTEGER, total_tokens INTEGER, usage_source TEXT NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	service, err := Open(path, Settings{Enabled: true, RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	var found bool
	rows, err := service.db.Query(`PRAGMA table_info(request_stats)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		found = found || name == "first_token_ms"
	}
	if !found {
		t.Fatal("first_token_ms column was not migrated")
	}
}

func TestCleanupUsesConfiguredRetention(t *testing.T) {
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 7})
	if err := service.insert(context.Background(), Event{RequestID: "expired", CreatedAt: time.Now().UTC().Add(-8 * 24 * time.Hour), Protocol: "chat", StatusCode: 200, Usage: Usage{Source: "missing"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.insert(context.Background(), Event{RequestID: "current", CreatedAt: time.Now().UTC().Add(-6 * 24 * time.Hour), Protocol: "chat", StatusCode: 200, Usage: Usage{Source: "missing"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := service.db.QueryRow(`SELECT COUNT(*) FROM request_stats`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rows after cleanup = %d, want 1", count)
	}
}

func TestMiddlewareCapturesRouteUsageAndFingerprint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 30})
	SetDefault(service)
	t.Cleanup(func() { SetDefault(nil) })
	router := gin.New()
	router.Use(Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		SetRoute(c, "client-model", "provider-1", "OpenAI", "upstream-model")
		c.JSON(http.StatusOK, gin.H{"usage": gin.H{"prompt_tokens": 8, "completion_tokens": 3, "total_tokens": 11, "prompt_tokens_details": gin.H{"cached_tokens": 4}}})
	})
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"client-model"}`))
	request.Header.Set("Authorization", "Bearer raw-secret")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}
	waitForRows(t, service, 1)
	var event Event
	var input, cached, total sql.NullInt64
	err := service.db.QueryRow(`SELECT request_id, protocol, access_key_id, client_model, provider_id, provider_name, upstream_model, status_code, latency_ms, input_tokens, cached_tokens, total_tokens, usage_source FROM request_stats`).Scan(&event.RequestID, &event.Protocol, &event.AccessKeyID, &event.ClientModel, &event.ProviderID, &event.ProviderName, &event.UpstreamModel, &event.StatusCode, &event.LatencyMS, &input, &cached, &total, &event.Usage.Source)
	if err != nil {
		t.Fatal(err)
	}
	if event.AccessKeyID != Fingerprint("raw-secret") || strings.Contains(event.AccessKeyID, "raw-secret") {
		t.Fatalf("unsafe key identity: %q", event.AccessKeyID)
	}
	if event.ClientModel != "client-model" || event.ProviderID != "provider-1" || event.UpstreamModel != "upstream-model" {
		t.Fatalf("missing route dimensions: %#v", event)
	}
	if input.Int64 != 8 || cached.Int64 != 4 || total.Int64 != 11 || event.Usage.Source != "upstream" {
		t.Fatalf("unexpected usage: %v %v %v %q", input, cached, total, event.Usage.Source)
	}
}

func TestMiddlewareReturnsRequestIDWhenCollectionIsDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := openTestService(t, Settings{Enabled: false, RetentionDays: 30})
	SetDefault(service)
	t.Cleanup(func() { SetDefault(nil) })
	router := gin.New()
	router.Use(Middleware())
	router.POST("/v1/responses", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
	if response.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID while statistics are disabled")
	}
	var count int
	if err := service.db.QueryRow(`SELECT COUNT(*) FROM request_stats`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("disabled statistics recorded %d rows", count)
	}
}

func TestMiddlewareCapturesUsageOutsideTailAndStreamingTTFT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := openTestService(t, Settings{Enabled: true, RetentionDays: 30})
	SetDefault(service)
	t.Cleanup(func() { SetDefault(nil) })
	router := gin.New()
	router.Use(Middleware())
	router.POST("/v1/chat/completions", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		_, _ = c.Writer.Write([]byte(`{"usage":{"prompt_tokens":9,"completion_tokens":4,"total_tokens":13},"payload":"` + strings.Repeat("x", captureLimit+1024) + `"}`))
	})
	router.POST("/v1/responses", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n")
		_, _ = c.Writer.WriteString("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":2,\"output_tokens\":1,\"total_tokens\":3}}}\n\n")
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil))
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/responses", nil))
	waitForRows(t, service, 2)
	var input int64
	if err := service.db.QueryRow(`SELECT input_tokens FROM request_stats WHERE protocol = 'chat'`).Scan(&input); err != nil {
		t.Fatal(err)
	}
	if input != 9 {
		t.Fatalf("large response input tokens = %d, want 9", input)
	}
	var firstToken sql.NullInt64
	if err := service.db.QueryRow(`SELECT first_token_ms FROM request_stats WHERE protocol = 'responses'`).Scan(&firstToken); err != nil {
		t.Fatal(err)
	}
	if !firstToken.Valid {
		t.Fatal("streaming response did not record first token")
	}
}

func TestParseUsageFromSSEAndPreservesMissing(t *testing.T) {
	usage := parseUsage([]byte("event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":7,\"total_tokens\":19,\"output_tokens_details\":{\"reasoning_tokens\":3}}}}\n\n"))
	if usage.Source != "upstream" || usage.InputTokens == nil || *usage.InputTokens != 12 || usage.ReasoningTokens == nil || *usage.ReasoningTokens != 3 {
		t.Fatalf("unexpected SSE usage: %#v", usage)
	}
	missing := parseUsage([]byte(`{"choices":[{"message":{"content":"hello"}}]}`))
	if missing.Source != "missing" || missing.TotalTokens != nil {
		t.Fatalf("unexpected missing usage: %#v", missing)
	}
	anthropic := parseUsage([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":6,\"output_tokens\":0}}}\n\nevent: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":4}}\n\n"))
	if anthropic.InputTokens == nil || *anthropic.InputTokens != 6 || anthropic.OutputTokens == nil || *anthropic.OutputTokens != 4 || anthropic.TotalTokens == nil || *anthropic.TotalTokens != 10 {
		t.Fatalf("split Anthropic usage was not merged: %#v", anthropic)
	}
}

func openTestService(t *testing.T, settings Settings) *Service {
	t.Helper()
	service, err := Open(filepath.Join(t.TempDir(), "statistics.db"), settings)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = service.Close() })
	return service
}

func waitForRows(t *testing.T, service *Service, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var count int
		if err := service.db.QueryRow(`SELECT COUNT(*) FROM request_stats`).Scan(&count); err == nil && count >= want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("statistics writer did not persist %d rows", want)
}
