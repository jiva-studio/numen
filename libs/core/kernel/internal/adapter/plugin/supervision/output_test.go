package supervision

import (
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

type outputRecorder struct {
	mu      sync.Mutex
	entries []domainlogs.Entry
}

func (r *outputRecorder) Record(entry domainlogs.Entry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry)
}

func (r *outputRecorder) List() []domainlogs.Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]domainlogs.Entry(nil), r.entries...)
}

func TestProcessOutputIsRecordedWithPluginID(t *testing.T) {
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Skip("no sh")
	}
	cmd := exec.CommandContext(t.Context(), sh, "-c", "echo to-out; echo 'a pass failed' >&2")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	rec := &outputRecorder{}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); forwardOutput(stdout, rec.Record, "org.example.sync", domainlogs.LevelInfo) }()
	go func() {
		defer wg.Done()
		forwardOutput(stderr, rec.Record, "org.example.sync", domainlogs.LevelWarning)
	}()
	wg.Wait()
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	got := map[string]domainlogs.Level{}
	for _, entry := range rec.List() {
		if entry.Source != SourcePlugin || entry.Properties["plugin"] != strconv.Quote("org.example.sync") {
			t.Fatalf("entry = %+v", entry)
		}
		got[entry.Message] = entry.Level
	}
	if got["to-out"] != domainlogs.LevelInfo || got["a pass failed"] != domainlogs.LevelWarning || len(got) != 2 {
		t.Fatalf("recorded = %v", got)
	}
}

func TestLongOutputLineIsTruncated(t *testing.T) {
	rec := &outputRecorder{}
	forwardOutput(strings.NewReader(strings.Repeat("x", 3*maxOutputLineBytes)+"\nnext\n"), rec.Record, "p", domainlogs.LevelInfo)
	entries := rec.List()
	if len(entries) != 2 || len(entries[0].Message) > maxOutputLineBytes+32 || !strings.HasSuffix(entries[0].Message, "[truncated]") || entries[1].Message != "next" {
		t.Fatalf("entries = %d", len(entries))
	}
}

func TestSlowLogDoesNotBlockPluginOutput(t *testing.T) {
	release := make(chan struct{})
	reader, writer := io.Pipe()
	finished := make(chan struct{})
	go func() {
		forwardOutput(reader, func(domainlogs.Entry) { <-release }, "p", domainlogs.LevelInfo)
		close(finished)
	}()
	for range 10 * outputQueueLines {
		if _, err := io.WriteString(writer, "line\n"); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("pump did not finish")
	}
}
