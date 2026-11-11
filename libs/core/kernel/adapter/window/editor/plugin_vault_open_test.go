package editor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// The user's own switch is rejected while a page prompts; the user then closes the prompt, so nothing is
// unsaved. A plugin's open after that shows the vault.
func TestPluginOpenVaultAfterSwitchPromptClosesShowsVault(t *testing.T) {
	f := openTwoVaults(t)
	f.bindVaultSwitch()
	stream, isDone := f.watchQuit(t)
	defer isDone()
	userResponded := make(chan struct{})
	f.respondPromptUntil(t, stream, userResponded)

	if err := f.installation.State.Switch.Execute(t.Context(), f.second, nil); err == nil {
		t.Fatal("the user's switch during a page prompt returned nil")
	}
	close(userResponded)
	waitCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if !f.installation.WaitForFlush(waitCtx) {
		t.Fatal("closing the prompt never completed the flush request")
	}

	list := newPluginVaultList(f)
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Errorf("the plugin's open after the prompt closed returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open after the prompt closed never returned")
	}
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}

// A vault switch given a plugin's open that has ended takes the exclusion like
// any other switch: during the user's close it is rejected and the close completes.
func TestVaultSwitchAfterPluginOpenFailsDuringUserClose(t *testing.T) {
	f := openTwoVaults(t)
	var mu sync.Mutex
	var pluginOpenCtx context.Context
	var finishedOpen *windowsession.PluginOpen
	f.installation.WrapPluginSwitch(func(windowsession.VaultSwitch) windowsession.VaultSwitch {
		return switchFunc(func(ctx context.Context, v domain.Vault, open *windowsession.PluginOpen) error {
		mu.Lock()
		if pluginOpenCtx == nil {
			pluginOpenCtx, finishedOpen = ctx, open
		}
		mu.Unlock()
			return f.installation.Show(ctx, v, open)
		})
	})
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
	list := newPluginVaultList(f)
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
		t.Fatal("the user's close never requested a flush from the page")
	}
	mu.Lock()
	switchCtx, switchOpen := pluginOpenCtx, finishedOpen
	mu.Unlock()
	err := f.installation.Show(switchCtx, f.first, switchOpen)
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
	if got := f.installation.State.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s, want %s", got.Name, f.second.Name)
	}
}
