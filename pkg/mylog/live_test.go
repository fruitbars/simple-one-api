package mylog

import (
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestSanitizeLiveMessageRedactsSecretsAndPayloads(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: `{"messages":[{"content":"private prompt"}]}`, want: "[structured payload omitted]"},
		{input: "Authorization: Bearer sk-secret", want: "Authorization: Bearer [REDACTED]"},
		{input: "request https://example.test?key=secret", want: "request https://example.test?key=[REDACTED]"},
	}
	for _, test := range tests {
		if got := sanitizeLiveMessage(test.input); got != test.want {
			t.Errorf("sanitizeLiveMessage(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestLiveLogStoreConcurrentWritesRemainOrdered(t *testing.T) {
	store := newLiveLogStore()
	var group sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := 0; index < 200; index++ {
				store.append(zapcore.Entry{Time: time.Now(), Level: zapcore.InfoLevel, Message: "entry"})
			}
		}()
	}
	group.Wait()
	entries := storeEntries(store)
	if len(entries) != liveLogCapacity {
		t.Fatalf("entry count = %d", len(entries))
	}
	for index := 1; index < len(entries); index++ {
		if entries[index].ID != entries[index-1].ID+1 {
			t.Fatalf("IDs not ordered at %d: %d then %d", index, entries[index-1].ID, entries[index].ID)
		}
	}
}

func TestLiveLogStoreKeepsFixedCapacity(t *testing.T) {
	store := newLiveLogStore()
	for i := 0; i < liveLogCapacity+25; i++ {
		store.append(zapcore.Entry{Time: time.Now(), Level: zapcore.InfoLevel, Message: strings.Repeat("x", i%3+1)})
	}
	if got := store.size; got != liveLogCapacity {
		t.Fatalf("entry count = %d, want %d", got, liveLogCapacity)
	}
	entries := storeEntries(store)
	if entries[0].ID != 26 {
		t.Fatalf("oldest entry ID = %d, want 26", entries[0].ID)
	}
}

func storeEntries(store *liveLogStore) []LiveEntry {
	store.mu.RLock()
	defer store.mu.RUnlock()
	entries := make([]LiveEntry, 0, store.size)
	for index := 0; index < store.size; index++ {
		entries = append(entries, store.entries[(store.start+index)%liveLogCapacity])
	}
	return entries
}
