package index_test

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/index"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type traceClock struct{}

func (traceClock) Now() time.Time { return time.Date(2026, 11, 18, 12, 0, 0, 0, time.UTC) }

func TestWithQueryLogRecordsExecutedQueriesAndArguments(t *testing.T) {
	var mu sync.Mutex
	var recorded []logs.Entry

	db, err := index.Open(
		t.Context(),
		filepath.Join(t.TempDir(), "index.db"),
		index.DisableSync(),
		index.WithQueryLog(traceClock{}, func(entry logs.Entry) {
			mu.Lock()
			recorded = append(recorded, entry)
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	})

	if err := db.Vaults().Register(t.Context(), domain.VaultID("v1")); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	foundInsert := false
	for _, entry := range recorded {
		if entry.Source == logs.SourceSQL && strings.Contains(entry.Properties["sql"], "INSERT") && strings.Contains(entry.Properties["args"], "v1") {
			foundInsert = true
			break
		}
	}
	if !foundInsert {
		t.Errorf("expected recorded INSERT query with vault arguments, got %d entries", len(recorded))
	}
}
