package editor_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// A decision whose call ends while the page is still writing leaves no request of its own running,
// so the next decision is not refused because of it.
func TestADecisionAfterACallThatEndedIsGiven(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	flushRequest := make(chan struct{}, 1)
	release := make(chan struct{})
	f.answerFlushes(t, stream, flushRequest, release)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	ctx, hangUp := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := list.OpenVault(ctx, f.second.ID)
		result <- err
	}()
	select {
	case <-flushRequest:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the decision never asked the page to flush")
	}
	hangUp()
	if err := <-result; !errors.Is(err, pluginport.ErrUnavailable) {
		close(release)
		t.Fatalf("the decision whose call ended returned %v, want ErrUnavailable", err)
	}
	close(release)

	if _, err := list.OpenVault(t.Context(), f.second.ID); err != nil {
		t.Errorf("the decision after a call that ended returned %v", err)
	}
}
