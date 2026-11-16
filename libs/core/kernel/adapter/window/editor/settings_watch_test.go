package editor_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

// watcher is one window's stream of the settings that change.
type watcher struct {
	stream *connect.ServerStreamForClient[v1.WatchSettingsResponse]
	next   chan []*v1.Setting
}

func (f *going) watchSettings(t *testing.T) *watcher {
	t.Helper()
	stream, err := f.configuring.WatchSettings(t.Context(), connect.NewRequest(&v1.WatchSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := stream.Close(); err != nil {
			t.Errorf("closing the stream: %v", err)
		}
	})
	one := &watcher{stream: stream, next: make(chan []*v1.Setting, 16)}
	go func() {
		for stream.Receive() {
			one.next <- stream.Msg().GetChanged()
		}
		close(one.next)
	}()
	if first := one.await(t); len(first) != 0 {
		t.Fatalf("the stream opened with the changes %v", first)
	}
	return one
}

func (w *watcher) await(t *testing.T) []*v1.Setting {
	t.Helper()
	select {
	case changed, open := <-w.next:
		if !open {
			t.Fatalf("the stream ended: %v", w.stream.Err())
		}
		return changed
	case <-time.After(5 * time.Second):
		t.Fatal("nothing arrived on the stream")
		return nil
	}
}

func (w *watcher) expectNothing(t *testing.T) {
	t.Helper()
	select {
	case changed := <-w.next:
		t.Fatalf("a change arrived that should not have: %v", changed)
	case <-time.After(600 * time.Millisecond):
	}
}

func expectWire(t *testing.T, changed []*v1.Setting, path []string, value string) {
	t.Helper()
	if len(changed) != 1 || !slices.Equal(changed[0].GetAt(), path) || changed[0].GetValue() != value {
		t.Errorf("the change is %v, want %v set to %s", changed, path, value)
	}
}

func TestAWriteFromOneWindowReachesTheStreamOfEveryWindowOnce(t *testing.T) {
	f := openWindow(t, nil, nil)
	writer, other := f.watchSettings(t), f.watchSettings(t)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{{At: []string{"app", "appearance", "mode"}, Value: `"dark"`}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	for _, one := range []*watcher{writer, other} {
		expectWire(t, one.await(t), []string{"app", "appearance", "mode"}, `"dark"`)
		one.expectNothing(t)
	}
}

func TestAHandEditOfTheSettingsFileReachesTheStreamOfEveryWindowOnce(t *testing.T) {
	f := openWindow(t, nil, nil)
	first, second := f.watchSettings(t), f.watchSettings(t)

	file := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(file, []byte(`{"app":{"appearance":{"textScale":1.2}}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, one := range []*watcher{first, second} {
		expectWire(t, one.await(t), []string{"app", "appearance", "textScale"}, `1.2`)
		one.expectNothing(t)
	}
}

func TestAWriteTheSettingsRejectReachesNoStream(t *testing.T) {
	f := openWindow(t, nil, nil)
	one := f.watchSettings(t)

	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{{At: []string{"md.numen.nobody", "mistyped"}, Value: `"x"`}},
	}))
	if err == nil {
		t.Fatal("a key nobody declares was written")
	}

	one.expectNothing(t)
}
