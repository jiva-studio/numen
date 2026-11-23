package editor_test

import (
	"context"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// The user's own switch is rejected while a page prompts; the user then closes the prompt, so nothing is
// unsaved. A plugin's open after that shows the vault.
func TestPluginOpenVaultAfterSwitchPromptClosesShowsVault(t *testing.T) {
	f := openTwoVaultsWith(t, container.Config.OpenEditor)
	stream, isDone := f.watchWindowClose(t)
	defer isDone()
	userResponded := make(chan struct{})
	f.respondPromptUntil(t, stream, userResponded)

	if err := f.installation.Show(t.Context(), f.second, nil); err == nil {
		t.Fatal("the user's switch during a page prompt returned nil")
	}
	close(userResponded)
	waitCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if !f.installation.WaitForFlush(waitCtx) {
		t.Fatal("closing the prompt never completed the flush request")
	}

	list := f.cfg.Endpoints.Endpoint.GetPluginVaultList()
	select {
	case err := <-callOpenVault(t.Context(), list, f.second.ID):
		if err != nil {
			t.Errorf("the plugin's open after the prompt closed returned %v, want nil", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("the plugin's open after the prompt closed never returned")
	}
	if got := f.installation.GetCurrentVault(); got.ID != f.second.ID {
		t.Errorf("the window shows %s after the plugin's open, want %s", got.Name, f.second.Name)
	}
}

func TestPluginCanEditTheShownVaultWhileTheWindowDrainsWrites(t *testing.T) {
	f := openTwoVaultsWith(t, container.Config.OpenEditor)
	writes := editor.PagesOf(f.installation).Writes
	<-writes.Seal()
	defer writes.Open()

	access := f.cfg.Endpoints.Endpoint.GetPluginVault()
	trashed, _, err := access.Arranger.RemoveNote(t.Context(), entropy)
	if err != nil || trashed == "" {
		t.Fatalf("plugin removal returned %q, %v", trashed, err)
	}
}
