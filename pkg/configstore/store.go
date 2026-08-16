package configstore

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNoActiveRevision = errors.New("no active configuration revision")

type Revision struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"`
	Note      string    `json:"note"`
	Checksum  string    `json:"checksum"`
	Active    bool      `json:"active"`
}

type Store struct {
	db   *sql.DB
	path string
}

func Open(path string) (*Store, error) {
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
		file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			return nil, err
		}
		if err := file.Close(); err != nil {
			return nil, err
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	store := &Store{db: db, path: path}
	if err := store.migrate(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Path() string { return s.path }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`PRAGMA foreign_keys = ON`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS config_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at TEXT NOT NULL,
			source TEXT NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			checksum TEXT NOT NULL,
			config_json BLOB NOT NULL,
			active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS config_revisions_one_active
			ON config_revisions(active) WHERE active = 1`,
		`CREATE TABLE IF NOT EXISTS metadata (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate configuration database: %w", err)
		}
	}
	return nil
}

func (s *Store) CreateRevision(ctx context.Context, payload []byte, source, note string, activate bool) (Revision, error) {
	if !json.Valid(payload) {
		return Revision{}, errors.New("configuration payload must be valid JSON")
	}
	checksum := checksum(payload)
	createdAt := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Revision{}, err
	}
	defer tx.Rollback()
	if activate {
		if _, err := tx.ExecContext(ctx, `UPDATE config_revisions SET active = 0 WHERE active = 1`); err != nil {
			return Revision{}, err
		}
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO config_revisions(created_at, source, note, checksum, config_json, active)
		VALUES (?, ?, ?, ?, ?, ?)`, createdAt.Format(time.RFC3339Nano), source, note, checksum, payload, activate)
	if err != nil {
		return Revision{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Revision{}, err
	}
	if err := tx.Commit(); err != nil {
		return Revision{}, err
	}
	return Revision{ID: id, CreatedAt: createdAt, Source: source, Note: note, Checksum: checksum, Active: activate}, nil
}

func (s *Store) Active(ctx context.Context) (Revision, []byte, error) {
	return s.revision(ctx, `WHERE active = 1`)
}

func (s *Store) Revision(ctx context.Context, id int64) (Revision, []byte, error) {
	return s.revision(ctx, `WHERE id = ?`, id)
}

func (s *Store) revision(ctx context.Context, where string, args ...any) (Revision, []byte, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, created_at, source, note, checksum, config_json, active
		FROM config_revisions `+where, args...)
	var revision Revision
	var createdAt string
	var payload []byte
	if err := row.Scan(&revision.ID, &createdAt, &revision.Source, &revision.Note, &revision.Checksum, &payload, &revision.Active); err != nil {
		if errors.Is(err, sql.ErrNoRows) && where == `WHERE active = 1` {
			return Revision{}, nil, ErrNoActiveRevision
		}
		return Revision{}, nil, err
	}
	parsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Revision{}, nil, err
	}
	revision.CreatedAt = parsed
	return revision, payload, nil
}

func (s *Store) List(ctx context.Context, limit int) ([]Revision, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, created_at, source, note, checksum, active
		FROM config_revisions ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Revision, 0)
	for rows.Next() {
		var revision Revision
		var createdAt string
		if err := rows.Scan(&revision.ID, &createdAt, &revision.Source, &revision.Note, &revision.Checksum, &revision.Active); err != nil {
			return nil, err
		}
		parsed, err := time.Parse(time.RFC3339Nano, createdAt)
		if err != nil {
			return nil, err
		}
		revision.CreatedAt = parsed
		result = append(result, revision)
	}
	return result, rows.Err()
}

func (s *Store) Activate(ctx context.Context, id int64) (Revision, []byte, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Revision{}, nil, err
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM config_revisions WHERE id = ?`, id).Scan(&exists); err != nil {
		return Revision{}, nil, err
	}
	if exists == 0 {
		return Revision{}, nil, sql.ErrNoRows
	}
	if _, err := tx.ExecContext(ctx, `UPDATE config_revisions SET active = 0 WHERE active = 1`); err != nil {
		return Revision{}, nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE config_revisions SET active = 1 WHERE id = ?`, id); err != nil {
		return Revision{}, nil, err
	}
	if err := tx.Commit(); err != nil {
		return Revision{}, nil, err
	}
	return s.Revision(ctx, id)
}

func (s *Store) Metadata(ctx context.Context, key string) (string, bool, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	return value, err == nil, err
}

func (s *Store) SetMetadata(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metadata(key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

func Checksum(payload []byte) string { return checksum(payload) }

func checksum(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
