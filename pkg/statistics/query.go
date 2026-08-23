package statistics

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type Query struct {
	From      time.Time
	To        time.Time
	Bucket    string
	Provider  string
	Model     string
	Protocol  string
	AccessKey string
	Status    string
}

type Summary struct {
	Requests              int64   `json:"requests"`
	Successful            int64   `json:"successful"`
	SuccessRate           float64 `json:"success_rate"`
	InputTokens           int64   `json:"input_tokens"`
	OutputTokens          int64   `json:"output_tokens"`
	CachedTokens          int64   `json:"cached_tokens"`
	CacheWriteTokens      int64   `json:"cache_write_tokens"`
	ReasoningTokens       int64   `json:"reasoning_tokens"`
	TotalTokens           int64   `json:"total_tokens"`
	KnownUsage            int64   `json:"known_usage"`
	UsageRate             float64 `json:"usage_rate"`
	AverageLatency        float64 `json:"average_latency_ms"`
	P50Latency            float64 `json:"p50_latency_ms"`
	P95Latency            float64 `json:"p95_latency_ms"`
	AverageTTFT           float64 `json:"average_ttft_ms"`
	TTFTSamples           int64   `json:"ttft_samples"`
	P50TTFT               float64 `json:"p50_ttft_ms"`
	P95TTFT               float64 `json:"p95_ttft_ms"`
	OutputTokensPerSecond float64 `json:"output_tokens_per_second"`
}

type Point struct {
	Time         time.Time `json:"time"`
	Requests     int64     `json:"requests"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
}

type Breakdown struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Requests     int64   `json:"requests"`
	SuccessRate  float64 `json:"success_rate"`
	KnownUsage   int64   `json:"known_usage"`
	UsageRate    float64 `json:"usage_rate"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
}

type Failure struct {
	RequestID    string    `json:"request_id"`
	CreatedAt    time.Time `json:"created_at"`
	StatusCode   int       `json:"status_code"`
	Protocol     string    `json:"protocol"`
	ClientModel  string    `json:"client_model"`
	ProviderName string    `json:"provider_name"`
	LatencyMS    int64     `json:"latency_ms"`
}

type Filters struct {
	Provider  string `json:"provider,omitempty"`
	Model     string `json:"model,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	AccessKey string `json:"access_key,omitempty"`
	Status    string `json:"status,omitempty"`
}

type Overview struct {
	From           time.Time   `json:"from"`
	To             time.Time   `json:"to"`
	Bucket         string      `json:"bucket"`
	Filters        Filters     `json:"filters"`
	Summary        Summary     `json:"summary"`
	Series         []Point     `json:"series"`
	Providers      []Breakdown `json:"providers"`
	Models         []Breakdown `json:"models"`
	AccessKeys     []Breakdown `json:"access_keys"`
	Protocols      []Breakdown `json:"protocols"`
	RecentFailures []Failure   `json:"recent_failures"`
	DroppedEvents  uint64      `json:"dropped_events"`
	RetentionDays  int         `json:"retention_days"`
}

func (s *Service) Overview(ctx context.Context, query Query) (Overview, error) {
	query = normalizeQuery(query)
	result := Overview{
		From: query.From, To: query.To, Bucket: query.Bucket,
		Filters: Filters{Provider: query.Provider, Model: query.Model, Protocol: query.Protocol, AccessKey: query.AccessKey, Status: query.Status},
		Series:  []Point{}, Providers: []Breakdown{}, Models: []Breakdown{}, AccessKeys: []Breakdown{}, Protocols: []Breakdown{}, RecentFailures: []Failure{},
		DroppedEvents: s.Dropped(), RetentionDays: s.Settings().RetentionDays,
	}
	where, args := queryWhere(query)
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),
		COALESCE(SUM(CASE WHEN status_code BETWEEN 200 AND 399 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0),
		COALESCE(SUM(cached_tokens), 0), COALESCE(SUM(cache_write_tokens), 0),
		COALESCE(SUM(reasoning_tokens), 0), COALESCE(SUM(total_tokens), 0),
		COALESCE(SUM(CASE WHEN usage_source = 'upstream' THEN 1 ELSE 0 END), 0), COALESCE(AVG(latency_ms), 0), COALESCE(AVG(first_token_ms), 0), COUNT(first_token_ms),
		COALESCE(SUM(CASE WHEN output_tokens IS NOT NULL AND first_token_ms IS NOT NULL AND latency_ms > first_token_ms THEN output_tokens ELSE 0 END) * 1000.0 /
		NULLIF(SUM(CASE WHEN output_tokens IS NOT NULL AND first_token_ms IS NOT NULL AND latency_ms > first_token_ms THEN latency_ms - first_token_ms ELSE 0 END), 0), 0)
		FROM request_stats WHERE `+where, args...).Scan(
		&result.Summary.Requests, &result.Summary.Successful,
		&result.Summary.InputTokens, &result.Summary.OutputTokens,
		&result.Summary.CachedTokens, &result.Summary.CacheWriteTokens,
		&result.Summary.ReasoningTokens, &result.Summary.TotalTokens,
		&result.Summary.KnownUsage, &result.Summary.AverageLatency, &result.Summary.AverageTTFT, &result.Summary.TTFTSamples,
		&result.Summary.OutputTokensPerSecond)
	if err != nil {
		return Overview{}, err
	}
	if result.Summary.Requests > 0 {
		result.Summary.SuccessRate = float64(result.Summary.Successful) * 100 / float64(result.Summary.Requests)
		result.Summary.UsageRate = float64(result.Summary.KnownUsage) * 100 / float64(result.Summary.Requests)
	}
	if result.Summary.P50Latency, err = s.percentile(ctx, query, "latency_ms", 0.50); err != nil {
		return Overview{}, err
	}
	if result.Summary.P95Latency, err = s.percentile(ctx, query, "latency_ms", 0.95); err != nil {
		return Overview{}, err
	}
	if result.Summary.P50TTFT, err = s.percentile(ctx, query, "first_token_ms", 0.50); err != nil {
		return Overview{}, err
	}
	if result.Summary.P95TTFT, err = s.percentile(ctx, query, "first_token_ms", 0.95); err != nil {
		return Overview{}, err
	}
	if result.Series, err = s.series(ctx, query); err != nil {
		return Overview{}, err
	}
	if result.Providers, err = s.breakdown(ctx, query, "provider_id", "provider_name"); err != nil {
		return Overview{}, err
	}
	if result.Models, err = s.breakdown(ctx, query, "client_model", "client_model"); err != nil {
		return Overview{}, err
	}
	if result.AccessKeys, err = s.breakdown(ctx, query, "access_key_id", "access_key_id"); err != nil {
		return Overview{}, err
	}
	if result.Protocols, err = s.breakdown(ctx, query, "protocol", "protocol"); err != nil {
		return Overview{}, err
	}
	if result.RecentFailures, err = s.failures(ctx, query); err != nil {
		return Overview{}, err
	}
	return result, nil
}

func normalizeQuery(query Query) Query {
	now := time.Now().UTC()
	if query.To.IsZero() || query.To.After(now.Add(time.Minute)) {
		query.To = now
	}
	if query.From.IsZero() {
		query.From = query.To.Add(-24 * time.Hour)
	}
	if !query.From.Before(query.To) {
		query.From = query.To.Add(-24 * time.Hour)
	}
	if query.To.Sub(query.From) > 366*24*time.Hour {
		query.From = query.To.Add(-366 * 24 * time.Hour)
	}
	if query.To.Sub(query.From) > 90*24*time.Hour {
		query.Bucket = "day"
	}
	if query.Bucket != "day" {
		query.Bucket = "hour"
	}
	query.Provider = strings.TrimSpace(query.Provider)
	query.Model = strings.TrimSpace(query.Model)
	query.Protocol = strings.TrimSpace(query.Protocol)
	query.AccessKey = strings.TrimSpace(query.AccessKey)
	if query.Status != "success" && query.Status != "failure" {
		query.Status = ""
	}
	return query
}

func queryWhere(query Query) (string, []any) {
	clauses := []string{"created_at >= ?", "created_at < ?"}
	args := []any{query.From.UTC().Format(timestampLayout), query.To.UTC().Format(timestampLayout)}
	for _, filter := range []struct{ column, value string }{
		{"provider_id", query.Provider}, {"client_model", query.Model}, {"protocol", query.Protocol}, {"access_key_id", query.AccessKey},
	} {
		if filter.value != "" {
			clauses = append(clauses, filter.column+" = ?")
			args = append(args, filter.value)
		}
	}
	if query.Status == "success" {
		clauses = append(clauses, "status_code BETWEEN 200 AND 399")
	} else if query.Status == "failure" {
		clauses = append(clauses, "(status_code < 200 OR status_code >= 400)")
	}
	return strings.Join(clauses, " AND "), args
}

func (s *Service) percentile(ctx context.Context, query Query, column string, percentile float64) (float64, error) {
	if column != "latency_ms" && column != "first_token_ms" {
		return 0, fmt.Errorf("invalid percentile column")
	}
	where, args := queryWhere(query)
	statement := fmt.Sprintf(`SELECT COALESCE((SELECT %s FROM request_stats WHERE %s AND %s IS NOT NULL ORDER BY %s LIMIT 1 OFFSET (SELECT MAX(CAST(COUNT(*) * ? + 0.999999999 AS INTEGER) - 1, 0) FROM request_stats WHERE %s AND %s IS NOT NULL)), 0)`, column, where, column, column, where, column)
	parameters := append(append([]any{}, args...), percentile)
	parameters = append(parameters, args...)
	var result float64
	err := s.db.QueryRowContext(ctx, statement, parameters...).Scan(&result)
	return result, err
}

func (s *Service) series(ctx context.Context, query Query) ([]Point, error) {
	format := "%Y-%m-%dT%H:00:00Z"
	if query.Bucket == "day" {
		format = "%Y-%m-%dT00:00:00Z"
	}
	where, args := queryWhere(query)
	parameters := append([]any{format}, args...)
	rows, err := s.db.QueryContext(ctx, `SELECT strftime(?, created_at), COUNT(*), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(total_tokens), 0) FROM request_stats WHERE `+where+` GROUP BY 1 ORDER BY 1`, parameters...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	observed := make(map[time.Time]Point)
	for rows.Next() {
		var raw string
		var point Point
		if err := rows.Scan(&raw, &point.Requests, &point.InputTokens, &point.OutputTokens, &point.TotalTokens); err != nil {
			return nil, err
		}
		point.Time, err = time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, err
		}
		observed[point.Time] = point
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	step := time.Hour
	start := query.From.UTC().Truncate(time.Hour)
	if query.Bucket == "day" {
		step = 24 * time.Hour
		start = time.Date(query.From.UTC().Year(), query.From.UTC().Month(), query.From.UTC().Day(), 0, 0, 0, 0, time.UTC)
	}
	result := make([]Point, 0, int(query.To.Sub(start)/step)+1)
	for current := start; current.Before(query.To); current = current.Add(step) {
		point := observed[current]
		point.Time = current
		result = append(result, point)
	}
	return result, nil
}

func (s *Service) breakdown(ctx context.Context, query Query, idColumn, nameColumn string) ([]Breakdown, error) {
	allowed := map[string]bool{"provider_id": true, "provider_name": true, "client_model": true, "access_key_id": true, "protocol": true}
	if !allowed[idColumn] || !allowed[nameColumn] {
		return nil, fmt.Errorf("invalid breakdown column")
	}
	where, args := queryWhere(query)
	statement := fmt.Sprintf(`SELECT %s, %s, COUNT(*), COALESCE(SUM(CASE WHEN status_code BETWEEN 200 AND 399 THEN 1 ELSE 0 END), 0), COALESCE(SUM(CASE WHEN usage_source = 'upstream' THEN 1 ELSE 0 END), 0), COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(total_tokens), 0) FROM request_stats WHERE %s GROUP BY %s, %s ORDER BY COUNT(*) DESC LIMIT 50`, idColumn, nameColumn, where, idColumn, nameColumn)
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Breakdown{}
	for rows.Next() {
		var item Breakdown
		var successful int64
		if err := rows.Scan(&item.ID, &item.Name, &item.Requests, &successful, &item.KnownUsage, &item.InputTokens, &item.OutputTokens, &item.TotalTokens); err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.Name) == "" {
			item.Name = "未识别"
		}
		if item.Requests > 0 {
			item.SuccessRate = float64(successful) * 100 / float64(item.Requests)
			item.UsageRate = float64(item.KnownUsage) * 100 / float64(item.Requests)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) failures(ctx context.Context, query Query) ([]Failure, error) {
	where, args := queryWhere(query)
	rows, err := s.db.QueryContext(ctx, `SELECT request_id, created_at, status_code, protocol, client_model, provider_name, latency_ms FROM request_stats WHERE `+where+` AND (status_code < 200 OR status_code >= 400) ORDER BY created_at DESC LIMIT 20`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Failure{}
	for rows.Next() {
		var item Failure
		var created string
		if err := rows.Scan(&item.RequestID, &created, &item.StatusCode, &item.Protocol, &item.ClientModel, &item.ProviderName, &item.LatencyMS); err != nil {
			return nil, err
		}
		parsed, parseErr := time.Parse(time.RFC3339Nano, created)
		if parseErr != nil {
			return nil, parseErr
		}
		item.CreatedAt = parsed
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *Service) ExportCSV(ctx context.Context, query Query, output io.Writer) error {
	query = normalizeQuery(query)
	where, args := queryWhere(query)
	rows, err := s.db.QueryContext(ctx, `SELECT created_at, request_id, protocol, access_key_id, client_model, provider_id, provider_name, upstream_model, status_code, latency_ms, first_byte_ms, first_token_ms, input_tokens, output_tokens, cached_tokens, cache_write_tokens, reasoning_tokens, total_tokens, usage_source FROM request_stats WHERE `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	_, _ = io.WriteString(output, "\xEF\xBB\xBF")
	writer := csv.NewWriter(output)
	if err := writer.Write([]string{"created_at", "request_id", "protocol", "access_key", "client_model", "provider_id", "provider_name", "upstream_model", "status_code", "latency_ms", "first_byte_ms", "first_token_ms", "input_tokens", "output_tokens", "cached_tokens", "cache_write_tokens", "reasoning_tokens", "total_tokens", "usage_source"}); err != nil {
		return err
	}
	for rows.Next() {
		values := make([]any, 19)
		pointers := make([]any, len(values))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err := rows.Scan(pointers...); err != nil {
			return err
		}
		record := make([]string, len(values))
		for index, value := range values {
			switch typed := value.(type) {
			case nil:
				record[index] = ""
			case []byte:
				record[index] = string(typed)
			case int64:
				record[index] = strconv.FormatInt(typed, 10)
			case float64:
				record[index] = strconv.FormatFloat(typed, 'f', -1, 64)
			default:
				record[index] = fmt.Sprint(typed)
			}
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return err
	}
	return rows.Err()
}
