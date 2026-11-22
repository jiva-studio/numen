package windowsession

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestAWindowWithoutAVaultSwitchCannotOpenAVaultForAPlugin(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())

	err := w.useCases.Switching.OpenPlugin.Execute(t.Context(), studio().ID)

	if err == nil {
		t.Error("a window with no switch opened a vault")
	}
}

func TestAPluginOpenShowsTheVaultThroughTheSwitchItIsGiven(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	show := func(ctx context.Context, v domain.Vault, open *PluginOpen) error {
		return w.useCases.Switching.Show.Execute(ctx, v, open)
	}

	if err := openVia(w, show).Execute(t.Context(), studio().ID); err != nil {
		t.Fatal(err)
	}

	if got := w.rt.GetCurrentVault(); got.ID != studio().ID {
		t.Errorf("the window shows %q, want %q", got.ID, studio().ID)
	}
}

func TestAPluginOpenWhileAPageAsksAQuestionAsksTheUserToSave(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	w.pages.isAsking = true
	show := func(context.Context, domain.Vault, *PluginOpen) error { return nil }

	err := openVia(w, show).Execute(t.Context(), studio().ID)

	if !errors.Is(err, plugin.ErrUnsavedChanges) {
		t.Errorf("the open returned %v, want %v", err, plugin.ErrUnsavedChanges)
	}
}

func TestAPluginOpenRefusedForAnUnansweredPageAsksTheUserToSave(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	show := func(context.Context, domain.Vault, *PluginOpen) error { return window.ErrAnswerPending }

	err := openVia(w, show).Execute(t.Context(), studio().ID)

	if !errors.Is(err, plugin.ErrUnsavedChanges) {
		t.Errorf("the open returned %v, want %v", err, plugin.ErrUnsavedChanges)
	}
}

func TestAUserSwitchCancelsAPluginOpenThatIsRunning(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	entered, canceled := make(chan struct{}), make(chan error, 1)
	show := func(ctx context.Context, _ domain.Vault, _ *PluginOpen) error {
		close(entered)
		<-ctx.Done()
		return context.Cause(ctx)
	}
	go func() { canceled <- openVia(w, show).Execute(t.Context(), studio().ID) }()
	<-entered

	if err := w.useCases.Switching.Show.Execute(t.Context(), studio(), nil); err != nil {
		t.Fatal(err)
	}

	if err := <-canceled; !errors.Is(err, domain.ErrUnavailable) {
		t.Errorf("the plugin open returned %v, want %v", err, domain.ErrUnavailable)
	}
}

type switchFunc func(ctx context.Context, v domain.Vault, open *PluginOpen) error

func (f switchFunc) Execute(ctx context.Context, v domain.Vault, open *PluginOpen) error {
	return f(ctx, v, open)
}

func openVia(w *testWindow, show func(context.Context, domain.Vault, *PluginOpen) error) OpenPluginVault {
	return NewOpenPluginVault(w.useCases.guard, knownVaults{}, w.useCases.identity, w.pages, switchFunc(show))
}

// A switch given a plugin open that has ended takes the exclusion like any other switch: during the
// close by the user it is rejected, and the close completes.
func TestASwitchOfAnEndedPluginOpenFailsDuringTheClose(t *testing.T) {
	w := newTestWindow(t, nil)
	w.open(t, garden())
	var recordedCtx context.Context
	var recordedOpen *PluginOpen
	record := func(ctx context.Context, v domain.Vault, open *PluginOpen) error {
		recordedCtx, recordedOpen = ctx, open
		return w.useCases.Switching.Show.Execute(ctx, v, open)
	}
	if err := openVia(w, record).Execute(t.Context(), studio().ID); err != nil {
		t.Fatal(err)
	}

	w.pages.holdNextFlush()
	closed := make(chan bool, 1)
	go func() { closed <- w.useCases.Closing.Flush.Execute(t.Context()) }()
	w.pages.waitUntilHeld(t)
	err := w.useCases.Switching.Show.Execute(recordedCtx, garden(), recordedOpen)
	w.pages.release()

	if err == nil {
		t.Error("a switch under the ended plugin open ran during the close")
	}
	if !<-closed {
		t.Error("the close did not complete")
	}
	if got := w.rt.GetCurrentVault(); got.ID != studio().ID {
		t.Errorf("the window shows %q, want %q", got.ID, studio().ID)
	}
}
