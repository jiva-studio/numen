package editor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// A plugin open whose call ends while the page is still writing leaves no request of its own running,
// so the next plugin open succeeds.
func TestAPluginOpenAfterACallThatEndedSucceeds(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	list := editor.NewPluginVaultList(f.installation)

	ctx, hangUp := context.WithCancel(t.Context())
	openResult := callOpenVault(ctx, list, f.second.ID)
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the plugin's open never asked the page to flush")
	}
	hangUp()
	if err := <-openResult; !errors.Is(err, pluginport.ErrUnavailable) {
		close(release)
		t.Fatalf("the plugin's open whose call ended returned %v, want ErrUnavailable", err)
	}
	close(release)

	if err := list.OpenVault(t.Context(), f.second.ID); err != nil {
		t.Errorf("the plugin's open after a call that ended returned %v", err)
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}
