package statistics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const defaultQueueSize = 1024
const defaultBatchSize = 64
const defaultBatchWait = 100 * time.Millisecond
const timestampLayout = "2006-01-02T15:04:05.000000000Z"

type Settings struct {
	Enabled       bool
	RetentionDays int
}

type Usage struct {
	InputTokens      *int64
	OutputTokens     *int64
	CachedTokens     *int64
	CacheWriteTokens *int64
	ReasoningTokens  *int64
	TotalTokens      *int64
	Source           string
}

type Event struct {
	RequestID     string
	CreatedAt     time.Time
	Protocol      string
	AccessKeyID   string
	ClientModel   string
	ProviderID    string
	ProviderName  string
	UpstreamModel string
	StatusCode    int
	LatencyMS     int64
	FirstByteMS   *int64
	FirstTokenMS  *int64
	Usage         Usage
}

type Service struct {
	db          *sql.DB
	queue       chan Event
	done        chan struct{}
	settings    atomic.Value
	dropped     atomic.Uint64
	lifecycleMu sync.RWMutex
	closed      bool
	closeOnce   sync.Once
	closeErr    error
	wg          sync.WaitGroup
}

func Open(path string, settings Settings) (*Service, error) {
	if path == "" {
		return nil, errors.New("empty SQLite path")
	}
	if path != ":memory:" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		path = absolute
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Service{db: db, queue: make(chan Event, defaultQueueSize), done: make(chan struct{})}
	s.Configure(settings)
	if err := s.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.cleanup(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	s.wg.Add(1)
	go s.run()
	return s, nil
}

func (s *Service) Configure(settings Settings) {
	if settings.RetentionDays <= 0 {
		settings.RetentionDays = 30
	}
	s.settings.Store(settings)
}

func (s *Service) Settings() Settings { return s.settings.Load().(Settings) }

func (s *Service) Record(event Event) bool {
	if s == nil {
		return false
	}
	s.lifecycleMu.RLock()
	defer s.lifecycleMu.RUnlock()
	if s.closed || !s.Settings().Enabled {
		return false
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	select {
	case s.queue <- event:
		return true
	default:
		s.dropped.Add(1)
		return false
	}
}

func (s *Service) Dropped() uint64 { return s.dropped.Load() }

func (s *Service) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.lifecycleMu.Lock()
		s.closed = true
		close(s.done)
		s.lifecycleMu.Unlock()
		s.wg.Wait()
		s.closeErr = s.db.Close()
	})
	return s.closeErr
}

func (s *Service) run() {
	defer s.wg.Done()
	cleanupTicker := time.NewTicker(6 * time.Hour)
	defer cleanupTicker.Stop()
	for {
		select {
		case event := <-s.queue:
			batch := s.collectBatch(event, defaultBatchWait)
			if err := s.insertBatch(context.Background(), batch); err != nil {
				s.dropped.Add(uint64(len(batch)))
			}
		case <-cleanupTicker.C:
			_ = s.cleanup(context.Background())
		case <-s.done:
			for len(s.queue) > 0 {
				batch := make([]Event, 0, defaultBatchSize)
				for len(batch) < defaultBatchSize && len(s.queue) > 0 {
					batch = append(batch, <-s.queue)
				}
				if err := s.insertBatch(context.Background(), batch); err != nil {
					s.dropped.Add(uint64(len(batch)))
				}
			}
			return
		}
	}
}

func (s *Service) collectBatch(first Event, wait time.Duration) []Event {
	batch := make([]Event, 1, defaultBatchSize)
	batch[0] = first
	timer := time.NewTimer(wait)
	defer timer.Stop()
	for len(batch) < defaultBatchSize {
		select {
		case event := <-s.queue:
			batch = append(batch, event)
		case <-timer.C:
			return batch
		case <-s.done:
			return batch
		}
	}
	return batch
}

func (s *Service) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA busy_timeout = 5000`,
		`PRAGMA journal_mode = WAL`,
		`PRAGMA synchronous = NORMAL`,
		`CREATE TABLE IF NOT EXISTS request_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			request_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			protocol TEXT NOT NULL,
			access_key_id TEXT NOT NULL DEFAULT '',
			client_model TEXT NOT NULL DEFAULT '',
			provider_id TEXT NOT NULL DEFAULT '',
			provider_name TEXT NOT NULL DEFAULT '',
			upstream_model TEXT NOT NULL DEFAULT '',
			status_code INTEGER NOT NULL,
			latency_ms INTEGER NOT NULL,
			first_byte_ms INTEGER,
			first_token_ms INTEGER,
			input_tokens INTEGER,
			output_tokens INTEGER,
			cached_tokens INTEGER,
			cache_write_tokens INTEGER,
			reasoning_tokens INTEGER,
			total_tokens INTEGER,
			usage_source TEXT NOT NULL CHECK (usage_source IN ('upstream', 'missing'))
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS request_stats_request_id ON request_stats(request_id)`,
		`CREATE INDEX IF NOT EXISTS request_stats_created_at ON request_stats(created_at)`,
		`CREATE INDEX IF NOT EXISTS request_stats_provider_created ON request_stats(provider_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS request_stats_model_created ON request_stats(client_model, created_at)`,
		`CREATE INDEX IF NOT EXISTS request_stats_key_created ON request_stats(access_key_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS request_stats_status_created ON request_stats(status_code, created_at)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate statistics database: %w", err)
		}
	}
	if err := s.ensureColumn(ctx, "first_token_ms", "INTEGER"); err != nil {
		return err
	}
	return nil
}

func (s *Service) ensureColumn(ctx context.Context, name, definition string) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(request_stats)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var column, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &column, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		if column == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE request_stats ADD COLUMN %s %s`, name, definition))
	return err
}

func (s *Service) insert(ctx context.Context, event Event) error {
	return s.insertBatch(ctx, []Event{event})
}

func (s *Service) insertBatch(ctx context.Context, events []Event) error {
	if len(events) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	statement, err := tx.PrepareContext(ctx, `INSERT INTO request_stats(
		request_id, created_at, protocol, access_key_id, client_model, provider_id, provider_name,
		upstream_model, status_code, latency_ms, first_byte_ms, first_token_ms, input_tokens, output_tokens,
		cached_tokens, cache_write_tokens, reasoning_tokens, total_tokens, usage_source
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, event := range events {
		usageSource := event.Usage.Source
		if usageSource != "upstream" {
			usageSource = "missing"
		}
		if _, err := statement.ExecContext(ctx,
			event.RequestID, event.CreatedAt.UTC().Format(timestampLayout), event.Protocol,
			event.AccessKeyID, event.ClientModel, event.ProviderID, event.ProviderName,
			event.UpstreamModel, event.StatusCode, event.LatencyMS, event.FirstByteMS, event.FirstTokenMS,
			event.Usage.InputTokens, event.Usage.OutputTokens, event.Usage.CachedTokens,
			event.Usage.CacheWriteTokens, event.Usage.ReasoningTokens, event.Usage.TotalTokens, usageSource); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Service) cleanup(ctx context.Context) error {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.Settings().RetentionDays).Format(timestampLayout)
	_, err := s.db.ExecContext(ctx, `DELETE FROM request_stats WHERE created_at < ?`, cutoff)
	return err
}

func Int64(value int64) *int64 { return &value }
