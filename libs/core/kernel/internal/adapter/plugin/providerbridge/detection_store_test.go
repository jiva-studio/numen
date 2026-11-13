package providerbridge_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/providerbridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
)

// standingClock is a clock that stands still until advance moves it.
type standingClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *standingClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *standingClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func setPluginVersion(types detectingTypes, version string) detectingTypes {
	changed := make(detectingTypes, len(types))
	for i, one := range types {
		one.PluginVersion = version
		changed[i] = one
	}
	return changed
}

func cacheIn(t *testing.T, root string) *storage.Storage {
	t.Helper()
	files := storage.New(func(err error) { t.Errorf("the storage reported %v", err) }, func(string) string { return "1.0.0" })
	files.SetStorageRoots(root, "")
	return files
}

func (f *detectionFixture) restart(types detectingTypes, files providerbridge.DetectionFiles) *providerbridge.Detection {
	detection := providerbridge.NewDetection(types, f.source, f.clock)
	detection.SetFiles(files)
	return detection
}

func (f *detectionFixture) countCalls() int {
	f.source.mu.Lock()
	defer f.source.mu.Unlock()
	return f.source.calls
}

func TestARestartListsTheStoredAnswerWithoutAskingTheProvider(t *testing.T) {
	f := newDetectionFixture()
	root := t.TempDir()
	f.detection.SetFiles(cacheIn(t, root))
	f.detect(t)
	calls := f.countCalls()

	restarted := f.restart(f.types, cacheIn(t, root))

	listed := restarted.ListDetected()
	if len(listed) != 1 || listed[0].ID != kiteID || listed[0].Fields["parallelRequests"] != float64(2) || listed[0].Fields["program"] != "kite" {
		t.Fatalf("listed = %+v, want Kite agent with its fields", listed)
	}
	if listed[0].Path != kitePath {
		t.Errorf("path = %q, want the path the provider answered, %q", listed[0].Path, kitePath)
	}
	if restarted.IsPending(kiteTypeID) {
		t.Error("the type is pending after a restart with its answer kept")
	}
	if got := f.countCalls(); got != calls {
		t.Errorf("the providers were asked %d times after the restart, want none", got-calls)
	}
}

func TestAStoredAnswerOfAnotherPluginVersionIsPending(t *testing.T) {
	f := newDetectionFixture()
	root := t.TempDir()
	f.detection.SetFiles(cacheIn(t, root))
	f.detect(t)

	restarted := f.restart(setPluginVersion(f.types, "2.0.0"), cacheIn(t, root))

	if listed := restarted.ListDetected(); len(listed) != 0 || !restarted.IsPending(kiteTypeID) {
		t.Errorf("listed = %+v, pending = %v, want none and pending", listed, restarted.IsPending(kiteTypeID))
	}
}

func TestAFileThatCannotBeReadIsReportedAndHasNoAnswer(t *testing.T) {
	f := newDetectionFixture()
	root := t.TempDir()
	cacheIn(t, root).WriteDetectedConnections(agentsPluginID, []byte("{not json"))
	var reported []error
	restarted := f.restart(f.types, cacheIn(t, root))
	restarted.Follow(func() {}, func(err error) { reported = append(reported, err) })

	if !restarted.IsPending(kiteTypeID) || len(reported) != 1 {
		t.Errorf("pending = %v, reported = %v, want pending and one report", restarted.IsPending(kiteTypeID), reported)
	}
}

func TestRefreshSkipsAPluginAskedWithinTheRefreshInterval(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newDetectionFixture()
		f.detect(t)
		calls := f.countCalls()

		f.clock.advance(providerbridge.RefreshInterval - time.Second)
		f.detection.Refresh()
		synctest.Wait()

		if got := f.countCalls(); got != calls {
			t.Errorf("the providers were asked %d times, want none", got-calls)
		}
	})
}

func TestRefreshAsksAPluginAgainOnceTheRefreshIntervalPassed(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newDetectionFixture()
		f.detect(t)
		calls := f.countCalls()

		f.clock.advance(providerbridge.RefreshInterval)
		f.detection.Refresh()
		synctest.Wait()

		if got := f.countCalls(); got != calls+1 {
			t.Errorf("the providers were asked %d times, want once", got-calls)
		}
	})
}

func TestTheAnswersOfAPluginAreKeptInItsFileOfTheConnectionsFolder(t *testing.T) {
	f := newDetectionFixture()
	root := t.TempDir()
	f.detection.SetFiles(cacheIn(t, root))

	f.detect(t)

	content, err := os.ReadFile(filepath.Join(root, "connections", agentsPluginID+".json"))
	if err != nil || !strings.Contains(string(content), "Kite agent") {
		t.Errorf("content = %s, %v, want the answer of Kite", content, err)
	}
}
