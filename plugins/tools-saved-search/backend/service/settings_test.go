package service_test

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func receiveRun(t *testing.T, events <-chan event, computedAt string) {
	t.Helper()
	if start := receiveEvent(t, events); !strings.Contains(start.data, `"isRunning":true`) {
		t.Errorf("event = %+v, want the start of a run", start)
	}
	if write := receiveEvent(t, events); write.data != `{"computedAt":"`+computedAt+`","isRunning":false}` {
		t.Errorf("event = %+v, want the write of the run at %s", write, computedAt)
	}
}

func TestConfigureWithoutCooldownRunsTheStaleSearch(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	events := f.watchFirstRun(t)
	addWinterNote(f)
	f.clock.Advance(time.Minute)
	f.handleIndexChange(t)
	if res := f.getResults(t, stormsPath); !res.IsStale {
		t.Fatalf("response = %+v, want the search stale within the cooldown", res)
	}

	f.configure(t, map[string]string{"refreshMinutes": "0"})

	receiveRun(t, events, "2026-11-14T09:31:00Z")
	if got := getResultPaths(f.getResults(t, stormsPath)); !slices.Contains(got, "Harbor/Winter.md") {
		t.Errorf("results = %v, want the new note included", got)
	}
}

func TestConfigureWithANewRerankerRunsTheSearchAgain(t *testing.T) {
	f := newFixture(t, 10)
	f.addStormsNote()
	events := f.watchFirstRun(t)
	f.addReranker()
	f.clock.Advance(time.Minute)

	f.configure(t, map[string]string{"refreshMinutes": "10"})

	receiveRun(t, events, "2026-11-14T09:31:00Z")
	if record := f.readRecord(t, stormsID); record.Reranker == nil || record.Reranker.ConnectionID != "conn-harbor" {
		t.Errorf("record = %+v, want the results reranked by the new row", record)
	}
}
