package embedderbridge

import (
	"testing"
	"time"
)

const listenerTimeout = 3 * time.Second

// A listener that registers another listener while it runs does not block the announcement, and the
// listener it registered runs on the next change.
func TestAListenerThatRegistersAnotherListenerDoesNotBlockTheAnnouncement(t *testing.T) {
	host, bridge, mock, _ := openConfigurable(t)
	registered := make(chan struct{})
	second := make(chan struct{}, 4)
	bridge.OnModelChange(func() {
		bridge.OnModelChange(func() { second <- struct{}{} })
		close(registered)
	})

	mock.modelName.Store("another-model")
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-registered:
	case <-time.After(listenerTimeout):
		t.Fatal("the listener did not return from registering another listener")
	}

	mock.modelName.Store("third-model")
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "model", "third-model"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-second:
	case <-time.After(listenerTimeout):
		t.Fatal("the listener registered by a listener did not run on the next change")
	}
}

// The listeners run after the embedder has released its lock, so a listener can read the model and
// can take the lock itself.
func TestListenersRunAfterTheEmbedderReleasesItsLock(t *testing.T) {
	host, bridge, mock, _ := openConfigurable(t)
	type outcome struct {
		unlocked bool
		model    string
	}
	seen := make(chan outcome, 1)
	bridge.OnModelChange(func() {
		unlocked := bridge.changeMu.TryLock()
		if unlocked {
			bridge.changeMu.Unlock()
		}
		seen <- outcome{unlocked: unlocked, model: bridge.GetModel().Name}
	})

	mock.modelName.Store("another-model")
	if err := host.UpdatePluginSetting(t.Context(), embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-seen:
		if !got.unlocked {
			t.Error("the embedder held its lock while the listener ran")
		}
		if got.model != "another-model" {
			t.Errorf("the listener read the model %q, want another-model", got.model)
		}
	case <-time.After(listenerTimeout):
		t.Fatal("the listener did not run")
	}
}
