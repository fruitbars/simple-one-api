package mylog

import (
	"regexp"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
)

const liveLogCapacity = 500

type LiveEntry struct {
	ID      uint64    `json:"id"`
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
	Caller  string    `json:"caller,omitempty"`
}

type liveLogStore struct {
	mu      sync.RWMutex
	nextID  uint64
	entries []LiveEntry
	start   int
	size    int
}

var liveLogs = newLiveLogStore()

func newLiveLogStore() *liveLogStore {
	return &liveLogStore{entries: make([]LiveEntry, liveLogCapacity)}
}

var sensitiveLogPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(bearer\s+)[^\s,;]+`),
	regexp.MustCompile(`(?i)((?:api[_-]?key|secret[_-]?key|token)["'=:\s]+)[^\s,"'}]+`),
	regexp.MustCompile(`(?i)([?&](?:key|api_key|token)=)[^&\s]+`),
}

func sanitizeLiveMessage(message string) string {
	trimmed := strings.TrimSpace(message)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "[structured payload omitted]"
	}
	for _, pattern := range sensitiveLogPatterns {
		trimmed = pattern.ReplaceAllString(trimmed, `${1}[REDACTED]`)
	}
	if len(trimmed) > 2000 {
		trimmed = trimmed[:2000] + "..."
	}
	return trimmed
}

func (s *liveLogStore) append(entry zapcore.Entry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	liveEntry := LiveEntry{
		ID:      s.nextID,
		Time:    entry.Time,
		Level:   entry.Level.String(),
		Message: sanitizeLiveMessage(entry.Message),
		Caller:  entry.Caller.TrimmedPath(),
	}
	if len(s.entries) != liveLogCapacity {
		s.entries = make([]LiveEntry, liveLogCapacity)
		s.start = 0
		s.size = 0
	}
	index := (s.start + s.size) % liveLogCapacity
	if s.size == liveLogCapacity {
		index = s.start
		s.start = (s.start + 1) % liveLogCapacity
	} else {
		s.size++
	}
	s.entries[index] = liveEntry
}

func LiveEntries(after uint64, limit int) []LiveEntry {
	if limit <= 0 || limit > liveLogCapacity {
		limit = 200
	}
	liveLogs.mu.RLock()
	defer liveLogs.mu.RUnlock()
	result := make([]LiveEntry, 0, limit)
	for index := 0; index < liveLogs.size; index++ {
		entry := liveLogs.entries[(liveLogs.start+index)%liveLogCapacity]
		if entry.ID <= after {
			continue
		}
		result = append(result, entry)
		if len(result) == limit {
			break
		}
	}
	return result
}

type liveCore struct {
	zapcore.LevelEnabler
}

func (core liveCore) With([]zapcore.Field) zapcore.Core { return core }
func (core liveCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}
func (core liveCore) Write(entry zapcore.Entry, _ []zapcore.Field) error {
	liveLogs.append(entry)
	return nil
}
func (core liveCore) Sync() error { return nil }
