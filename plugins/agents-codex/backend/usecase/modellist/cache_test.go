package modellist

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/agentmodel"
	"github.com/jiva-studio/numen/plugins/agents-codex/backend/internal/adapter/codexcli"
)

func newTestCache(now Clock) *Cache {
	return NewCache(codexcli.Models{}, now, false)
}

func TestTheLimitsOfAListing(t *testing.T) {
	if listLimit != 15*time.Second {
		t.Errorf("listLimit = %v, want 15s", listLimit)
	}
	if failureBackoff != 30*time.Second {
		t.Errorf("failureBackoff = %v, want 30s", failureBackoff)
	}
}

// What Codex listed is cached for as long as the command line stays the same.
func TestCodexModelCache_Retention(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	counter := filepath.Join(t.TempDir(), "starts")
	command := writeScript(t, "codex",
		"echo x >> "+counter+"\n"+
			`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	cache := newTestCache(newFakeClock(time.Unix(1000, 0)))

	for range 2 {
		if got := cache.Get(t.Context(), []string{command}); len(got) != 1 || got[0].Name != "gpt-x" {
			t.Fatalf("Codex lists %+v", got)
		}
	}
	raw, err := os.ReadFile(counter)
	if err != nil {
		t.Fatal(err)
	}
	if starts := strings.Count(string(raw), "x"); starts != 1 {
		t.Errorf("Codex was started %d times, want 1", starts)
	}
}

// A request that ended before Codex answered keeps nothing.
func TestAnEndedRequestKeepsNoCodexModels(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	command := writeScript(t, "codex",
		`echo '{"id":1,"result":{}}'`+"\n"+
			`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
			"while read -r _; do :; done\n")
	cache := newTestCache(newFakeClock(time.Unix(1000, 0)))

	ended, cancel := context.WithCancel(t.Context())
	cancel()
	if got := cache.Get(ended, []string{command}); got != nil {
		t.Errorf("an ended request got %+v", got)
	}
	if got := cache.Get(t.Context(), []string{command}); len(got) != 1 {
		t.Errorf("the next request got %+v", got)
	}
}

// A failed listing is not retried for a while and is then requested again, and the models Codex lists then are
// cached.
func TestModels_FailedListingIsRequeried(t *testing.T) {
	needsShell(t)
	t.Setenv("HOME", t.TempDir())
	failing := writeScript(t, "codex", "exit 1\n")
	now := newFakeClock(time.Unix(1000, 0))
	cache := newTestCache(now)

	if got := cache.Get(t.Context(), []string{failing}); got != nil {
		t.Fatalf("a failed listing gave %+v", got)
	}
	if cache.failedKey == "" {
		t.Fatal("expected failedKey to be set")
	}
	// The same command line, working now.
	if err := os.WriteFile(failing, []byte("#!/bin/sh\n"+
		`echo '{"id":1,"result":{}}'`+"\n"+
		`echo '{"id":2,"result":{"data":[{"model":"gpt-x"}]}}'`+"\n"+
		"while read -r _; do :; done\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := cache.Get(t.Context(), []string{failing}); got != nil {
		t.Fatalf("a command line that just failed was asked again: %+v", got)
	}
	now.Advance(failureBackoff)
	if got := cache.Get(t.Context(), []string{failing}); len(got) != 1 || got[0].Name != "gpt-x" {
		t.Errorf("the next listing gave %+v", got)
	}
	if cache.failedKey != "" {
		t.Errorf("expected failedKey to be cleared, got %q", cache.failedKey)
	}
}

func TestGetModelsFallsBackToTheStoredList(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	cacheContent := `{"models": [{"slug": "fallback-model", "display_name": "Fallback Model", "visibility": "show"}]}`
	if err := os.WriteFile(filepath.Join(codexDir, "models_cache.json"), []byte(cacheContent), 0o600); err != nil {
		t.Fatal(err)
	}

	models, err := GetModels(t.Context(), codexcli.Models{}, []string{"/nonexistent/bin"})
	if err != nil {
		t.Fatalf("expected no error falling back to the stored list, got %v", err)
	}
	if len(models) != 1 || models[0].Name != "fallback-model" {
		t.Fatalf("expected fallback-model from the stored list, got %v", models)
	}
}

func TestGetModelsFailsWhenTheProgramAndTheStoredListGiveNone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if models, err := GetModels(t.Context(), codexcli.Models{}, []string{"/nonexistent-codex"}); err == nil || models != nil {
		t.Errorf("GetModels with no stored list and a bad command = (%v, %v), want error", models, err)
	}
}

func TestACachedListIsKeyedByTheWholeCommandLine(t *testing.T) {
	now := newFakeClock(time.Unix(1000, 0))
	c := newTestCache(now)
	c.key, c.models, c.isCached = "a b", []agentmodel.Model{{Name: "cached-ab"}}, true
	c.failedKey, c.failedAt = "fail-cmd", now.Now()

	if got := c.Get(t.Context(), []string{"a", "b"}); len(got) != 1 || got[0].Name != "cached-ab" {
		t.Fatalf("expected cached-ab for key 'a b', got %v", got)
	}
	now.Advance(failureBackoff - time.Nanosecond)
	if got := c.Get(t.Context(), []string{"fail-cmd"}); got != nil {
		t.Fatalf("expected nil before failureBackoff elapsed, got %v", got)
	}
}

// An asynchronous cache answers with the stored list while the background listing runs, and starts
// one listing per command line.
func TestAnAsynchronousCacheAnswersWithTheStoredListFirst(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".codex", "models_cache.json"), []byte(`{"models":[{"slug":"disk-m","display_name":"Disk"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	asyncCache := NewCache(codexcli.Models{}, newFakeClock(time.Unix(1000, 0)), true)
	asyncCache.inFlight = map[string]bool{"already": true}
	asyncCache.startBackgroundFetch(t.Context(), "already", []string{"/nonexistent"})
	cmd := []string{"/bin/sh", "-c", `read -r _; echo '{"id":1,"result":{}}'; read -r _; read -r _; echo '{"id":2,"result":{"data":[{"model":"bg-m"}]}}'`, "sh"}

	initial := asyncCache.Get(t.Context(), cmd)
	if len(initial) != 1 || initial[0].Name != "disk-m" {
		t.Fatalf("async initial get = %v, want disk-m", initial)
	}
	wall := clock.System{}
	deadline := wall.Now().Add(25 * time.Second)
	for wall.Now().Before(deadline) {
		asyncCache.mu.Lock()
		cached := asyncCache.isCached
		asyncCache.mu.Unlock()
		if cached {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	bgGot := asyncCache.Get(t.Context(), cmd)
	if len(bgGot) != 1 || bgGot[0].Name != "bg-m" {
		t.Fatalf("async cached get = %v, want bg-m", bgGot)
	}
}

func newFakeClock(at time.Time) *clock.Fake {
	fake := &clock.Fake{}
	fake.Set(at)
	return fake
}
