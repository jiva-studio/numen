package editor

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/windowview"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestPluginWindowWithNoVaultReturnsErrNoWindow(t *testing.T) {
	state := &State{Places: newPlaceBroadcaster()}
	window := newPluginWindow(state, nil, file.NewFocusPassages(vaultreader.Readers{}, windowview.View{Notices: state}))
	ctx := t.Context()

	if _, err := window.FocusNote(ctx, "notes/Heron.md"); !errors.Is(err, plugin.ErrNoWindow) {
		t.Errorf("FocusNote returned %v, want ErrNoWindow", err)
	}
	if err := window.FocusPassages(ctx, "library/Birds.pdf", []domain.ByteRange{{Start: 0, End: 4}}); !errors.Is(err, plugin.ErrNoWindow) {
		t.Errorf("FocusPassages returned %v, want ErrNoWindow", err)
	}
	if _, err := window.ReadOpenTabs(ctx); !errors.Is(err, plugin.ErrNoWindow) {
		t.Errorf("ReadOpenTabs returned %v, want ErrNoWindow", err)
	}
	if _, err := window.ActivateTab(ctx, "tab-1"); !errors.Is(err, plugin.ErrNoWindow) {
		t.Errorf("ActivateTab returned %v, want ErrNoWindow", err)
	}
}
