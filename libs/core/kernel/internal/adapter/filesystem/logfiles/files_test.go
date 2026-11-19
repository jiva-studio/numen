package logfiles_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/logfiles"
)

func TestWriteEntryAppendsDailyJSONLAndReadRecentReturnsTail(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	store := logfiles.NewFiles(dir)
	day := time.Date(2026, 11, 18, 14, 30, 0, 0, time.UTC)

	first := logs.Entry{
		Time:       day,
		Level:      logs.LevelInfo,
		Source:     logs.SourceCore,
		Message:    "vault opened",
		Properties: map[string]string{"vault": `"Test"`},
	}
	second := logs.Entry{
		Time:       day.Add(time.Second),
		Level:      logs.LevelDebug,
		Source:     logs.SourceSQL,
		Message:    "SQL query",
		Properties: map[string]string{"sql": `"SELECT 1"`, "durationMs": "0.5"},
	}

	if err := store.WriteEntry(first); err != nil {
		t.Fatal(err)
	}
	if err := store.WriteEntry(second); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "numen-2026-11-18.jsonl")); err != nil {
		t.Fatalf("daily file was not written: %v", err)
	}

	recent, err := store.ReadRecent(day, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent) != 1 || recent[0].Message != "SQL query" || recent[0].Properties["sql"] != `"SELECT 1"` {
		t.Errorf("recent = %+v, want only the last entry", recent)
	}
}

func TestWriteEntryPrunesDailyFilesOlderThanRetentionWindow(t *testing.T) {
	dir := t.TempDir()
	expiredPath := filepath.Join(dir, "numen-2026-10-01.jsonl")
	recentPath := filepath.Join(dir, "numen-2026-11-10.jsonl")
	otherPath := filepath.Join(dir, "keep.txt")
	for _, p := range []string{expiredPath, recentPath, otherPath} {
		if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	store := logfiles.NewFiles(dir)
	today := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	if err := store.WriteEntry(logs.Entry{
		Time:    today,
		Level:   logs.LevelInfo,
		Source:  logs.SourceCore,
		Message: "started",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(expiredPath); !os.IsNotExist(err) {
		t.Errorf("expired file %s still exists", expiredPath)
	}
	if _, err := os.Stat(recentPath); err != nil {
		t.Errorf("recent file %s was removed: %v", recentPath, err)
	}
	if _, err := os.Stat(otherPath); err != nil {
		t.Errorf("unrelated file %s was removed: %v", otherPath, err)
	}
}
