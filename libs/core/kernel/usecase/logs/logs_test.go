package logs_test

import (
	"context"
	"testing"
	"time"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	usecaselogs "github.com/jiva-studio/numen/libs/core/kernel/usecase/logs"
)

type stubClock struct{ now time.Time }

func (c stubClock) Now() time.Time { return c.now }

type memStore struct {
	written []domainlogs.Entry
}

func (m *memStore) WriteEntry(entry domainlogs.Entry) error {
	m.written = append(m.written, entry)
	return nil
}

func (m *memStore) ReadRecent(_ time.Time, _ int) ([]domainlogs.Entry, error) {
	return m.written, nil
}

func TestRecordFiltersByConfiguredLevelAndStreamsToWatchers(t *testing.T) {
	instant := time.Date(2026, 11, 18, 15, 0, 0, 0, time.UTC)
	entries := usecaselogs.NewEntries()
	store := &memStore{}
	currentLevel := domainlogs.LevelInfo
	record := usecaselogs.NewRecord(entries, store, func() domainlogs.Level { return currentLevel }, stubClock{now: instant})
	watch := usecaselogs.NewWatch(entries, store, stubClock{now: instant})

	if err := record.Execute(domainlogs.Entry{
		Level:   domainlogs.LevelDebug,
		Source:  domainlogs.SourceSQL,
		Message: "SQL query",
	}); err != nil {
		t.Fatal(err)
	}
	if len(store.written) != 0 {
		t.Fatalf("debug entry recorded at info level: %+v", store.written)
	}

	if err := record.Execute(domainlogs.Entry{
		Level:      domainlogs.LevelInfo,
		Source:     domainlogs.SourceCore,
		Message:    "vault opened",
		Properties: map[string]string{"vault": `"Test"`},
	}); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stream := watch.Execute(ctx)

	initial := <-stream
	if len(initial) != 1 || initial[0].Message != "vault opened" {
		t.Fatalf("initial stream = %+v, want vault opened", initial)
	}

	currentLevel = domainlogs.LevelDebug
	if err := record.Execute(domainlogs.Entry{
		Level:      domainlogs.LevelDebug,
		Source:     domainlogs.SourceSQL,
		Message:    "SQL query",
		Properties: map[string]string{"sql": `"SELECT 1"`},
	}); err != nil {
		t.Fatal(err)
	}

	next := <-stream
	if len(next) != 1 || next[0].Message != "SQL query" || next[0].Properties["sql"] != `"SELECT 1"` {
		t.Fatalf("live stream batch = %+v, want SQL query", next)
	}
}
