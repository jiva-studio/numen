package container

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/logevents"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/logfiles"
)

func TestBackgroundErrorIsRecordedAndReported(t *testing.T) {
	index := filepath.Join(t.TempDir(), "index.db")
	problem := errors.New("watcher stopped")
	var reported []error
	c := NewConfig(Config{Paths: Paths{IndexPath: index}, ErrorHandler: func(err error) { reported = append(reported, err) }})
	handleError(c, problem)
	entries, err := logfiles.NewBesideIndex(index).ReadRecent(c.Clock().Now(), 10)
	if err != nil || len(entries) != 1 {
		t.Fatalf("read logs: %v, entries: %v", err, entries)
	}
	if entries[0].Level != logs.LevelError || entries[0].Message != "Background operation failed" || entries[0].Properties["error"] != `"watcher stopped"` {
		t.Errorf("entry = %+v", entries[0])
	}
	if len(reported) != 1 || !errors.Is(reported[0], problem) {
		t.Errorf("reported = %v", reported)
	}
}

func TestDependencyLogRecordsOnlyTerminalStates(t *testing.T) {
	var entries []logs.Entry
	record := func(entry logs.Entry) { entries = append(entries, entry) }
	logevents.RecordDependencyStatus(record, "model", plugin.DependencyStatus{State: plugin.DependencyStateProvisioning})
	logevents.RecordDependencyStatus(record, "model", plugin.DependencyStatus{State: plugin.DependencyStateFailed, Error: "checksum mismatch"})
	logevents.RecordDependencyStatus(record, "model", plugin.DependencyStatus{State: plugin.DependencyStateReady})
	if len(entries) != 2 {
		t.Fatalf("entries = %v", entries)
	}
	if entries[0].Level != logs.LevelError || entries[0].Properties["dependency"] != `"model"` || entries[0].Properties["error"] != `"checksum mismatch"` {
		t.Errorf("failure = %+v", entries[0])
	}
	if entries[1].Level != logs.LevelInfo || entries[1].Message != "Plugin dependency ready" {
		t.Errorf("ready = %+v", entries[1])
	}
}

func TestBackgroundLogWriteFailureIsReportedWithoutRecursion(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(parent, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	problem := errors.New("watcher stopped")
	var reported []error
	c := NewConfig(Config{Paths: Paths{IndexPath: filepath.Join(parent, "index.db")}, ErrorHandler: func(err error) { reported = append(reported, err) }})
	handleError(c, problem)
	if len(reported) != 2 || !errors.Is(reported[1], problem) {
		t.Errorf("reported = %v", reported)
	}
}
