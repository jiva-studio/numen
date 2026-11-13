package sourcebridge_test

import (
	"sync"
	"testing"
	"time"
)

const listenerTimeout = 3 * time.Second

// A listener that registers another listener while it runs does not block the announcement, and the
// listener it registered runs on the next change.
func TestAListenerThatRegistersAnotherListenerDoesNotBlockTheAnnouncement(t *testing.T) {
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	registered := make(chan struct{})
	second := make(chan struct{}, 4)
	var once sync.Once
	bridge.OnDescriptionChange(func() {
		// The listener stays registered, so it runs again on the next change.
		once.Do(func() {
			bridge.OnDescriptionChange(func() { second <- struct{}{} })
			close(registered)
		})
	})

	mock.modelName.Store("another-model")
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-registered:
	case <-time.After(listenerTimeout):
		t.Fatal("the listener did not return from registering another listener")
	}

	mock.modelName.Store("third-model")
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "third-model"); err != nil {
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
	host, bridge, mock, _ := openConfigurableEmbedder(t)
	type outcome struct {
		unlocked bool
		model    string
	}
	outcomes := make(chan outcome, 1)
	bridge.OnDescriptionChange(func() {
		unlocked := bridge.IsListenerLockFree()
		outcomes <- outcome{unlocked: unlocked, model: bridge.Describe().Name}
	})

	mock.modelName.Store("another-model")
	if err := updateSetting(t.Context(), host, embeddingPluginID, "model", "another-model"); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-outcomes:
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
