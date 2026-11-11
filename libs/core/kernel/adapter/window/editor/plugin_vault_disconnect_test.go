package editor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// A plugin open whose call ends while the page is still writing leaves no request of its own running,
// so the next plugin open succeeds.
func TestAPluginOpenAfterACallThatEndedSucceeds(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.respondToFlushes(t, stream, flushRequest, release)
	list := newPluginVaultList(f)

	ctx, hangUp := context.WithCancel(t.Context())
	openResult := callOpenVault(ctx, list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the plugin's open never requested a flush from the page")
	}
	hangUp()
	if err := <-openResult; !errors.Is(err, domain.ErrUnavailable) {
		close(release)
		t.Fatalf("the plugin's open whose call ended returned %v, want ErrUnavailable", err)
	}
	close(release)

	if err := list.OpenVault(t.Context(), f.second.ID); err != nil {
		t.Errorf("the plugin's open after a call that ended returned %v", err)
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}
