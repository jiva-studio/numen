package editor_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// The user's own switch is rejected while a page asks; the user then answers the page, so nothing is
// unsaved. A plugin's open after that shows the vault.
func TestPluginOpenVaultAfterTheUserAnswersTheSwitchQuestionShowsTheVault(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userAnswered := make(chan struct{})
	f.answerAskingUntil(t, stream, userAnswered)

	if err := f.installation.State.OpenVault(t.Context(), f.second); err == nil {
		t.Fatal("the user's switch while a page asks returned nil")
	}
	close(userAnswered)
	waitCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if !f.installation.WaitForFlush(waitCtx) {
		t.Fatal("the page's answer never completed the flush request")
	}

	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Errorf("the plugin's open after the user answered the page returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open after the user answered the page never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}

// A vault switch made under the context of a plugin's open that has ended takes the exclusion like
// any other switch: during the user's close it is rejected and the close completes.
func TestAVaultSwitchWithTheContextOfAFinishedPluginOpenFailsDuringTheUsersClose(t *testing.T) {
	f := openTwoVaults(t)
	var mu sync.Mutex
	var pluginOpenCtx context.Context
	f.installation.State.OpenVault = func(ctx context.Context, v domain.Vault) error {
		mu.Lock()
		if pluginOpenCtx == nil {
			pluginOpenCtx = ctx
		}
		mu.Unlock()
		return f.installation.Show(ctx, v)
	}
	stream, isDone := f.watchQuit(t)
	defer isDone()
	secondFlush := make(chan struct{}, 1)
	release := make(chan struct{})
	go func() {
		count := 0
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			count++
			if count == 2 {
				secondFlush <- struct{}{}
				<-release
			}
			if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
				return
			}
		}
	}()
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Fatalf("the plugin's open returned %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}

	closeCtx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	closeResult := make(chan bool, 1)
	go func() { closeResult <- f.installation.FlushPending(closeCtx) }()
	select {
	case <-secondFlush:
	case <-time.After(5 * time.Second):
		close(release)
		t.Fatal("the user's close never asked the page to flush")
	}
	mu.Lock()
	switchCtx := pluginOpenCtx
	mu.Unlock()
	err := f.installation.Show(switchCtx, f.first)
	close(release)
	if err == nil {
		t.Error("a switch under the context of an ended plugin open ran during the user's close")
	}
	select {
	case isFlushed := <-closeResult:
		if !isFlushed {
			t.Error("the user's close did not complete")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the user's close never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s, want %s", got.Name, f.second.Name)
	}
}

// flushKinds is what a page records of the flush requests it receives: whether each is silent.
type flushKinds struct {
	mu       sync.Mutex
	isSilent []bool
}

// getSince returns the kinds recorded from the index on.
func (k *flushKinds) getSince(index int) []bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	return slices.Clone(k.isSilent[index:])
}

func (k *flushKinds) getCount() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.isSilent)
}

// recordFlushKinds answers every flush of the page with NOTHING, after it records whether the
// request is silent.
func (f *showing) recordFlushKinds(t *testing.T, stream *connect.ServerStreamForClient[v1.WatchQuitResponse]) *flushKinds {
	kinds := &flushKinds{}
	go func() {
		for stream.Receive() {
			if !stream.Msg().GetFlush() {
				continue
			}
			kinds.mu.Lock()
			kinds.isSilent = append(kinds.isSilent, stream.Msg().GetIsSilent())
			kinds.mu.Unlock()
			if f.reportFlush(t, stream.Msg().GetToken(), v1.FlushResult_FLUSH_RESULT_NOTHING) != nil {
				return
			}
		}
	}()
	return kinds
}

// requireKinds fails the test unless at least one flush was recorded and each has the kind.
func requireKinds(t *testing.T, action string, got []bool, isSilent bool) {
	t.Helper()
	if len(got) == 0 {
		t.Errorf("%s sent no flush request", action)
	}
	for i, one := range got {
		if one != isSilent {
			t.Errorf("flush request %d of %s has is_silent %v, want %v", i, action, one, isSilent)
		}
	}
}

func TestPluginOpenVaultSendsSilentFlushRequests(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	kinds := f.recordFlushKinds(t, stream)
	list := editor.NewPluginVaultList(f.installation, f.cfg.CheckRoot())

	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Fatalf("the plugin's open returned %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open never returned")
	}

	requireKinds(t, "the plugin's open", kinds.getSince(0), true)
}

func TestTheUsersSwitchAndCloseSendFlushRequestsThatAreNotSilent(t *testing.T) {
	f := openTwoVaults(t)
	f.bindShow()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	kinds := f.recordFlushKinds(t, stream)

	if err := f.installation.State.OpenVault(t.Context(), f.second); err != nil {
		t.Fatalf("the user's switch returned %v", err)
	}
	requireKinds(t, "the user's switch", kinds.getSince(0), false)

	afterSwitch := kinds.getCount()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	if !f.installation.FlushPending(ctx) {
		t.Fatal("the user's close did not complete")
	}
	requireKinds(t, "the user's close", kinds.getSince(afterSwitch), false)
}
